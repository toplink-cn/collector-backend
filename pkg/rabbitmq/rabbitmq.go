package rabbitmq

import (
	"collector-backend/db"
	"collector-backend/models"
	"collector-backend/pkg/collect_return/server_collect_return"
	"collector-backend/pkg/collect_return/switch_collect_return"
	"collector-backend/pkg/collect_return/system_collect_return"
	"collector-backend/services"
	"collector-backend/util"
	"collector-backend/util/crypt_util"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	client "github.com/influxdata/influxdb1-client"
	"github.com/streadway/amqp"
)

const (
	PointChanCap    int   = 1000
	SqlQueryChanCap int   = 1000
	PoolCap         int32 = 100
)

type Connection struct {
	Config Config
	Conn   *amqp.Connection
}
type Config struct {
	Url string
}

func NewConnection(config Config) (Connection, error) {
	conn := Connection{}
	amqpConn, err := amqp.Dial(config.Url)
	util.FailOnError(err, "Failed to connect to RabbitMQ")
	conn.Conn = amqpConn
	return conn, nil
}

type Controller struct {
	Channel                *amqp.Channel
	Queue                  amqp.Queue
	InfluxPointChannel     chan client.Point
	InfluxDbSwitch         bool
	InfluxDbResetTimer     *time.Timer
	SqlQueryChannel        chan models.SqlQuery
	SqlQuerySwitch         bool
	SqlQueryResetTimer     *time.Timer
	Pool                   gopool.Pool
	LastInfluxPointChanLen int
	LastSqlQueryChanLen    int
}

// func init() {
// 	services.RegisterRabbitMQCtrl(NewCtrl())
// }

func NewCtrl() *Controller {
	return &Controller{
		InfluxPointChannel: make(chan client.Point, PointChanCap),
		SqlQueryChannel:    make(chan models.SqlQuery, SqlQueryChanCap),
		Pool:               gopool.NewPool("collector-handler", PoolCap, gopool.NewConfig()),
		InfluxDbResetTimer: time.NewTimer(10 * time.Second),
		SqlQueryResetTimer: time.NewTimer(10 * time.Second),
	}
}

func (ctrl *Controller) SetupChannelAndQueue(name string, amqpConn *amqp.Connection) error {
	ch, err := amqpConn.Channel()
	util.FailOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		name,  // 队列名称
		false, // 是否持久化
		false, // 是否自动删除
		false, // 是否具有排他性
		false, // 是否阻塞等待
		nil,   // 额外的属性
	)
	util.FailOnError(err, "Failed to declare a queue")

	log.Printf("%s channel & queue declared", name)

	ctrl.Channel = ch
	ctrl.Queue = q

	return nil
}

func (ctrl *Controller) RunCtrl() {
	defer ctrl.Channel.Close()
	forever := make(chan bool)
	go ctrl.ListenQueue()
	log.Printf(" [*] Waiting for messages. To exit, press CTRL+C")
	<-forever
}

func (ctrl *Controller) ListenQueue() {
	msgs, err := ctrl.Channel.Consume(
		ctrl.Queue.Name, // 队列名称
		"",              // 消费者标签
		true,            // 是否自动回复
		false,           // 是否独占
		false,           // 是否阻塞等待
		false,           // 额外的属性
		nil,             // 消费者取消回调函数
	)
	util.FailOnError(err, "Failed to register a consumer")

	// 处理接收到的消息
	for d := range msgs {
		// log.Printf("Received a message: %s", d.Body)
		var msg models.Msg
		decryptedMsg, err := crypt_util.New().DecryptViaPrivate(d.Body)
		if err != nil {
			log.Println("fail to decrypt data, ", string(d.Body))
			return
		}
		err = json.Unmarshal(decryptedMsg, &msg)
		if err != nil {
			fmt.Printf("无法解析JSON数据: %v", err)
			return
		}
		if msg.Type == "" {
			return
		}
		ctrl.Pool.Go(func() {
			switch msg.Type {
			case "switch":
				services.RegisterCollectReturn(switch_collect_return.NewSwitchCollectReturn(ctrl.InfluxPointChannel, ctrl.SqlQueryChannel))
				services.CollectReturn().HandleCollectReturn(msg.Data)
			case "server":
				services.RegisterCollectReturn(server_collect_return.NewServerCollectReturn(ctrl.InfluxPointChannel, ctrl.SqlQueryChannel))
				services.CollectReturn().HandleCollectReturn(msg.Data)
			case "system":
				services.RegisterCollectReturn(system_collect_return.NewSystemCollectReturn(ctrl.InfluxPointChannel, ctrl.SqlQueryChannel))
				services.CollectReturn().HandleCollectReturn(msg.Data)
			}
		})
	}
}

func (ctrl *Controller) RunTimer() {
	go func(ctrl *Controller) {
		resetTimer := ctrl.InfluxDbResetTimer
		ticker := time.NewTicker(1 * time.Second)
		second := 0
		for {
			select {
			case <-ticker.C:
				second++
				// fmt.Println("InfluxDbSwitch current second:", second)
				ctrl.InfluxDbSwitch = false
				if second%10 == 0 {
					second = 0
					resetTimer.Reset(10 * time.Second)
					ctrl.InfluxDbSwitch = true
				}
			case <-resetTimer.C:
				second = 0
				resetTimer.Reset(10 * time.Second)
				ctrl.InfluxDbSwitch = true
			}
		}
	}(ctrl)
	go func(ctrl *Controller) {
		resetTimer := ctrl.SqlQueryResetTimer
		ticker := time.NewTicker(1 * time.Second)
		second := 0
		for {
			select {
			case <-ticker.C:
				second++
				// fmt.Println("SqlQuerySwitch current second:", second)
				ctrl.SqlQuerySwitch = false
				if second%10 == 0 {
					second = 0
					resetTimer.Reset(10 * time.Second)
					ctrl.SqlQuerySwitch = true
				}
			case <-resetTimer.C:
				second = 0
				resetTimer.Reset(10 * time.Second)
				ctrl.SqlQuerySwitch = true
			}
		}
	}(ctrl)
}

func (ctrl *Controller) ListenInfluxChannel() {
	for {
		len := len(ctrl.InfluxPointChannel)
		// fmt.Printf("%v points chan len: %d, InfluxDbSwitch: %v  \n", time.Now().Format("2016-01-02 15:04:05"), len, ctrl.InfluxDbSwitch)

		if len == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		if ctrl.LastInfluxPointChanLen != len {
			ctrl.InfluxDbSwitch = false
			ctrl.InfluxDbResetTimer.Reset(10 * time.Second)
		}

		if ctrl.InfluxDbSwitch || len >= PointChanCap {
			// write
			fmt.Println("start write points")
			c := db.GetInfluxDbConnection()
			points := []client.Point{}
			for i := 0; i < len; i++ {
				points = append(points, <-ctrl.InfluxPointChannel)
			}
			bp := client.BatchPoints{
				Points:   points,
				Database: "dcim",
			}

			r, err := c.Write(bp)
			if err != nil {
				fmt.Printf("unexpected error.  expected %v, actual %v", nil, err)
			}
			if r != nil {
				fmt.Printf("unexpected response. expected %v, actual %v", nil, r)
			}
			fmt.Println("write points done")
			ctrl.LastInfluxPointChanLen = 0
		} else {
			ctrl.LastInfluxPointChanLen = len
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

func (ctrl *Controller) ListenSqlQueryChannel() {
	for {
		len := len(ctrl.SqlQueryChannel)
		// fmt.Printf("%v sqlQuery chan len: %d, SqlQuerySwitch: %v \n", time.Now().Format("2016-01-02 15:04:05"), len, ctrl.SqlQuerySwitch)

		if len == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		if ctrl.LastSqlQueryChanLen != len {
			ctrl.SqlQuerySwitch = false
			ctrl.SqlQueryResetTimer.Reset(10 * time.Second)
		}

		if ctrl.SqlQuerySwitch || len >= PointChanCap {
			//write
			fmt.Println("start write sql")
			mysql_conn := db.GetMysqlConnection()

			tx, err := mysql_conn.Begin()
			util.LogIfErr(err)

			for i := 0; i < len; i++ {
				sql_query := <-ctrl.SqlQueryChannel
				_, err := tx.Exec(sql_query.Query, sql_query.Args...)
				if err != nil {
					tx.Rollback()
					util.FailOnError(err, "执行sql update 出错")
					continue
				}
			}
			err = tx.Commit()
			util.LogIfErr(err)
			mysql_conn.Close()
			fmt.Println("write sql done")
			ctrl.LastSqlQueryChanLen = 0
		} else {
			ctrl.LastSqlQueryChanLen = len
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

func PublishMsg(ch *amqp.Channel, q amqp.Queue, msg models.Msg) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		fmt.Printf("Cannot be encoded in json format: %v", err)
		return err
	}
	encryptedMsg, err := crypt_util.New().EncryptViaPub(jsonData)
	if err != nil {
		fmt.Printf("Cannot encrypted data: %v", err)
		return err
	}
	// 发布消息到队列
	err = ch.Publish(
		"",     // 交换机名称
		q.Name, // 队列名称
		false,  // 是否强制
		false,  // 是否立即发送
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        encryptedMsg,
		},
	)
	if err != nil {
		log.Printf("无法发布消息: %v", err)
		return err
	}

	fmt.Println("消息已发送到队列！")
	return nil
}
