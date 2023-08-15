package crontab

import (
	"collector-backend/db"
	"collector-backend/models"
	models_system "collector-backend/models/system"
	"collector-backend/pkg/logger"
	"collector-backend/pkg/rabbitmq"
	"collector-backend/pkg/system"
	"encoding/json"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/streadway/amqp"
)

const (
	PointChanCap    int = 10000
	SqlQueryChanCap int = 10000
)

var expression string
var disabled bool

func init() {
	expression = "* * * * * "
	disabled = true
}

type Crontab struct {
	SqlQueryChannel chan *models.SqlQuery
	Channel         *amqp.Channel
	Queue           amqp.Queue
}

func NewCrontab(SqlQueryChannel chan *models.SqlQuery, ch *amqp.Channel, q amqp.Queue) *Crontab {
	return &Crontab{
		SqlQueryChannel: SqlQueryChannel,
		Channel:         ch,
		Queue:           q,
	}
}

func (c *Crontab) Run() {
	go func() {
		cr := cron.New(cron.WithParser(cron.NewParser(
			cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
		)))
		cr.AddFunc(expression, func() {
			logger.Printf("disabled: %v, expression: %v \n", disabled, expression)
			if !disabled {
				c.doCollectSystemInfo()
			}
		})
		cr.Start()
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
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
	logger.Println("getCrontabFromDB start")
	mysql_conn := db.GetMysqlConnection()
	defer mysql_conn.Close()
	sql_query := models.SqlQuery{
		Query: "SELECT id,disabled,expression FROM schedules where payload = ?",
		Args:  []any{`App\Schedules\Servers\SystemInfo`},
	}

	schedule := models.Schedule{}
	err := mysql_conn.QueryRow(sql_query.Query, sql_query.Args...).Scan(&schedule.ID, &schedule.Disabled, &schedule.Expression)
	if err != nil {
		logger.Printf("Cannot get data from dcim schedules, Err: %v \n", err.Error())
		return err
	}

	if schedule.Disabled == 1 {
		logger.Println("SystemInfo disabled")
		disabled = true
		return err
	}

	disabled = false
	expression = schedule.Expression
	return nil
}

func (c *Crontab) doCollectSystemInfo() {
	logger.Println("start doCollectSystemInfo")
	sc := system.NewSystemCollector(&models_system.SystemInfo{ID: 0})
	sc.Collect()

	jsonData, err := json.Marshal(sc.SystemInfo)
	if err != nil {
		logger.Printf("无法编码为JSON格式: %v", err.Error())
	}
	returnMsg := models.Msg{Type: "system", Time: time.Now().Unix(), Data: string(jsonData)}
	if err := rabbitmq.PublishMsg(c.Channel, c.Queue, returnMsg); err != nil {
		logger.Println("发送失败")
	}
	logger.Println("doCollectSystemInfo done")
}
