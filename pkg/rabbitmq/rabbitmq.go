package rabbitmq

import (
	"collector-backend/db"
	"collector-backend/models"
	"collector-backend/pkg/collect_return/server_collect_return"
	"collector-backend/pkg/collect_return/switch_collect_return"
	"collector-backend/pkg/collect_return/system_collect_return"
	"collector-backend/pkg/logger"
	"collector-backend/services"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"runtime"
	"time"

	"github.com/bytedance/gopkg/util/gopool"
	client "github.com/influxdata/influxdb1-client"
	"github.com/streadway/amqp"
)

const (
	PointChanCap        int = 10000
	SqlQueryChanCap     int = 10000
	NotificationChanCap int = 10000
	PoolCapPreCoreNum   int = 2
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
	logger.ExitIfErr(err, "Failed to connect to RabbitMQ")
	conn.Conn = amqpConn
	return conn, nil
}

func NewConnectionWithTLS(config Config) (Connection, error) {
	cert, err := tls.LoadX509KeyPair("/app/ssl/client.crt", "/app/ssl/client.key")
	if err != nil {
		logger.ExitIfErr(err, "Failed to load X509 key pair")
	}

	caCert, err := ioutil.ReadFile("/app/ssl/ca.crt")
	if err != nil {
		logger.ExitIfErr(err, "Failed to read CA certificate")
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
	}

	conn := Connection{}
	amqpConn, err := amqp.DialTLS(config.Url, tlsConfig)
	logger.ExitIfErr(err, "Failed to connect to RabbitMQ")
	conn.Conn = amqpConn
	return conn, nil
}

type Controller struct {
	Channel                 *amqp.Channel
	Queue                   amqp.Queue
	NotifyChannel           *amqp.Channel
	NotifyQueue             amqp.Queue
	InfluxPointChannel      chan *models.MyPoint
	InfluxPointWriteChannel chan *models.MyPoint
	InfluxDbSwitch          bool
	InfluxDbResetTimer      *time.Timer
	SqlQueryChannel         chan *models.SqlQuery
	NotificationChannel     chan *models.Notification
	SqlQuerySwitch          bool
	SqlQueryResetTimer      *time.Timer
	Pool                    gopool.Pool
	LastInfluxPointChanLen  int
	LastSqlQueryChanLen     int
	LastNotificationChanLen int
}

// func init() {
// 	services.RegisterRabbitMQCtrl(NewCtrl())
// }

func NewCtrl() *Controller {
	numCPU := runtime.NumCPU()
	poolCap := int32(PoolCapPreCoreNum * numCPU)
	logger.Printf("Number of CPU cores: %d, poolCap: %d\n", numCPU, poolCap)
	return &Controller{
		InfluxPointChannel:      make(chan *models.MyPoint, PointChanCap),
		InfluxPointWriteChannel: make(chan *models.MyPoint, PointChanCap),
		SqlQueryChannel:         make(chan *models.SqlQuery, SqlQueryChanCap),
		NotificationChannel:     make(chan *models.Notification, NotificationChanCap),
		Pool:                    gopool.NewPool("collector-handler", poolCap, gopool.NewConfig()),
		InfluxDbResetTimer:      time.NewTimer(1 * time.Second),
		SqlQueryResetTimer:      time.NewTimer(1 * time.Second),
	}
}

func (ctrl *Controller) SetupChannelAndQueue(name string, amqpConn *amqp.Connection) error {
	ch, err := amqpConn.Channel()
	logger.ExitIfErr(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		name,  // 队列名称
		false, // 是否持久化
		true,  // 是否自动删除
		false, // 是否具有排他性
		true,  // 是否阻塞等待
		nil,   // 额外的属性
	)
	logger.ExitIfErr(err, "Failed to declare a queue")

	logger.Printf("%s channel & queue declared", name)

	ctrl.Channel = ch
	ctrl.Queue = q

	return nil
}

func (ctrl *Controller) RunCtrl() {
	defer ctrl.Channel.Close()
	forever := make(chan bool)
	go ctrl.ListenQueue()
	logger.Println(" [*] Waiting for messages. To exit, press CTRL+C")
	<-forever
}

func (ctrl *Controller) ListenQueue() {
	msgs, err := ctrl.Channel.Consume(
		ctrl.Queue.Name, // 队列名称
		"slave-0",       // 消费者标签
		true,            // 是否自动回复
		false,           // 是否独占
		false,           // 是否阻塞等待
		false,           // 额外的属性
		nil,             // 消费者取消回调函数
	)
	logger.ExitIfErr(err, "Failed to register a consumer")

	// 处理接收到的消息
	for d := range msgs {
		var msg models.Msg
		decodedMsg, err := base64.StdEncoding.DecodeString(string(d.Body))
		if err != nil {
			logger.Printf("fail to decode base64 data, %s \n", string(d.Body))
			return
		}
		err = json.Unmarshal(decodedMsg, &msg)
		logger.LogIfErrWithMsg(err, "Fail To Decode JSON Data")
		if msg.Type == "" {
			return
		}
		ctrl.Pool.Go(func() {
			switch msg.Type {
			case "switch":
				switchCollectReturn := switch_collect_return.NewSwitchCollectReturn(ctrl.InfluxPointChannel, ctrl.SqlQueryChannel)
				switchCollectReturn.NotificationChannel = ctrl.NotificationChannel
				services.RegisterCollectReturn(switchCollectReturn)
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
				ctrl.InfluxDbSwitch = false
				if second%1 == 0 {
					second = 0
					resetTimer.Reset(1 * time.Second)
					ctrl.InfluxDbSwitch = true
				}
			case <-resetTimer.C:
				second = 0
				resetTimer.Reset(1 * time.Second)
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
				ctrl.SqlQuerySwitch = false
				if second%1 == 0 {
					second = 0
					resetTimer.Reset(1 * time.Second)
					ctrl.SqlQuerySwitch = true
				}
			case <-resetTimer.C:
				second = 0
				resetTimer.Reset(1 * time.Second)
				ctrl.SqlQuerySwitch = true
			}
		}
	}(ctrl)
}

func (ctrl *Controller) ListenInfluxChannel() {
	for {
		len := len(ctrl.InfluxPointChannel)
		// logger.Printf("%v buffer points chan len: %d, switch: %v  \n", time.Now().Format("2006-01-02 15:04:05"), len, ctrl.InfluxDbSwitch)

		if len == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		// if ctrl.LastInfluxPointChanLen != len {
		// 	ctrl.InfluxDbSwitch = false
		// 	ctrl.InfluxDbResetTimer.Reset(10 * time.Second)
		// }

		if ctrl.InfluxDbSwitch || len >= PointChanCap {
			var tmp *models.MyPoint
			for i := 0; i < len; i++ {
				tmp = <-ctrl.InfluxPointChannel
				ctrl.InfluxPointWriteChannel <- tmp
			}
			ctrl.LastInfluxPointChanLen = 0
		} else {
			ctrl.LastInfluxPointChanLen = len
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

func (ctrl *Controller) ListenInfluxWriteChannel() {
	for {
		len := len(ctrl.InfluxPointWriteChannel)
		if len == 0 {
			time.Sleep(1 * time.Second)
			continue
		}
		// logger.Printf("%v write points chan len: %d, switch: %v  \n", time.Now().Format("2006-01-02 15:04:05"), len, ctrl.InfluxDbSwitch)
		points := []client.Point{}
		for i := 0; i < len; i++ {
			myPoint := <-ctrl.InfluxPointWriteChannel
			points = append(points, myPoint.Point)
			if myPoint.Wg != nil {
				myPoint.Wg.Done()
			}
		}

		bp := client.BatchPoints{
			Points:   points,
			Database: "dcim",
		}
		conn := db.NewInfluxDBWriteConnection()
		c := conn.GetClient()
		r, err := c.Write(bp)
		if err != nil {
			logger.Printf("unexpected error.  expected %v, actual %v", nil, err.Error())
		}
		if r != nil {
			logger.Printf("unexpected response. expected %v, actual %v", nil, r.Error())
		}
		// logger.Println("write points done")
		conn.CloseClient(c)
	}
}

func (ctrl *Controller) ListenSqlQueryChannel() {
	for {
		len := len(ctrl.SqlQueryChannel)
		// logger.Printf("%v sqlQuery chan len: %d, SqlQuerySwitch: %v \n", time.Now().Format("2016-01-02 15:04:05"), len, ctrl.SqlQuerySwitch)

		if len == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		if ctrl.LastSqlQueryChanLen != len {
			ctrl.SqlQuerySwitch = false
			ctrl.SqlQueryResetTimer.Reset(1 * time.Second)
		}

		if ctrl.SqlQuerySwitch || len >= PointChanCap {
			//write
			mysql_conn := db.GetMysqlConnection()

			tx, err := mysql_conn.Begin()
			logger.LogIfErr(err)

			for i := 0; i < len; i++ {
				sql_query := <-ctrl.SqlQueryChannel
				_, err := tx.Exec(sql_query.Query, sql_query.Args...)
				if err != nil {
					tx.Rollback()
					logger.LogIfErrWithMsg(err, "exec sql update error")
					continue
				}
			}
			err = tx.Commit()
			logger.LogIfErr(err)
			mysql_conn.Close()
			logger.Println("write sql done")
			ctrl.LastSqlQueryChanLen = 0
		} else {
			ctrl.LastSqlQueryChanLen = len
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

func (ctrl *Controller) ListenNotificationChannel() {
	for {
		len := len(ctrl.NotificationChannel)

		if len == 0 {
			time.Sleep(1 * time.Second)
			continue
		}

		notification := <-ctrl.NotificationChannel

		notifyData, err := json.Marshal(notification)
		if err != nil {
			logger.Printf("Unable To Parse Data In JSON Format: %v", err.Error())
		}
		notifyMsg := models.Msg{Type: "notification", Time: time.Now().Unix(), Data: string(notifyData)}
		PublishMsg(ctrl.NotifyChannel, ctrl.NotifyQueue, notifyMsg)
	}
}

func PublishMsg(ch *amqp.Channel, q amqp.Queue, msg models.Msg) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		logger.Printf("Cannot be encoded in json format: %v \n", err.Error())
		return err
	}
	encodedMsg := base64.StdEncoding.EncodeToString(jsonData)
	// 发布消息到队列
	err = ch.Publish(
		"",     // 交换机名称
		q.Name, // 队列名称
		false,  // 是否强制
		false,  // 是否立即发送
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(encodedMsg),
		},
	)
	if err != nil {
		logger.Printf("无法发布消息: %v", err.Error())
		return err
	}

	return nil
}
