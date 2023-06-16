package crontab

import (
	"collector-backend/db"
	"collector-backend/models"
	models_system "collector-backend/models/system"
	"collector-backend/pkg/rabbitmq"
	"collector-backend/pkg/system"
	"encoding/json"
	"fmt"
	"time"

	"github.com/robfig/cron"
	"github.com/streadway/amqp"
)

const (
	PointChanCap    int = 1000
	SqlQueryChanCap int = 1000
)

var expression string
var disabled bool

func init() {
	expression = "* * * * * "
	disabled = false
}

type Crontab struct {
	SqlQueryChannel chan models.SqlQuery
	Channel         *amqp.Channel
	Queue           amqp.Queue
}

func NewCrontab(SqlQueryChannel chan models.SqlQuery, ch *amqp.Channel, q amqp.Queue) *Crontab {
	return &Crontab{
		SqlQueryChannel: SqlQueryChannel,
		Channel:         ch,
		Queue:           q,
	}
}

func (c *Crontab) Run() {
	go func() {
		cr := cron.New()
		cr.AddFunc(expression, func() {
			fmt.Println("执行任务:", time.Now())
			c.doCollectSystemInfo()
		})
		cr.Start()
	}()

	go func() {
		// ticker := time.NewTicker(1 * time.Minute)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.getCrontabFromDB()
			}
		}
	}()
}

func (c *Crontab) getCrontabFromDB() error {
	fmt.Println("getCrontabFromDB start")
	mysql_conn := db.GetMysqlConnection()
	defer mysql_conn.Close()
	sql_query := models.SqlQuery{
		Query: "SELECT id,disabled,expression FROM schedules where payload = ?",
		Args:  []any{`App\Schedules\Servers\SystemInfo`},
	}

	schedule := models.Schedule{}
	err := mysql_conn.QueryRow(sql_query.Query, sql_query.Args...).Scan(&schedule.ID, &schedule.Disabled, &schedule.Expression)
	if err != nil {
		fmt.Println("Cannot get data from dcim schedules, Err: ", err.Error())
		return err
	}

	if schedule.Disabled == 1 {
		fmt.Println("SystemInfo disabled")
		disabled = false
		return err
	}

	disabled = true
	expression = schedule.Expression
	return nil
}

func (c *Crontab) doCollectSystemInfo() {
	if disabled {
		return
	}
	sc := system.NewSystemCollector(&models_system.SystemInfo{ID: 0})
	sc.Collect()

	jsonData, err := json.Marshal(sc.SystemInfo)
	if err != nil {
		fmt.Printf("无法编码为JSON格式: %v", err)
	}
	returnMsg := models.Msg{Type: "system", Time: time.Now().Unix(), Data: string(jsonData)}
	if err := rabbitmq.PublishMsg(c.Channel, c.Queue, returnMsg); err != nil {
		fmt.Println("发送失败")
	}
	fmt.Println("doCollectSystemInfo done")
}