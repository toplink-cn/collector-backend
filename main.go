package main

import (
	"collector-backend/pkg/crontab"
	"collector-backend/pkg/rabbitmq"
	"collector-backend/util"
	"os"
)

func main() {
	run()
}

func run() {
	// 访问环境变量
	amqpUsername := os.Getenv("RABBITMQ_USERNAME")
	amqpPassowd := os.Getenv("RABBITMQ_PASSWORD")
	amqpUrl := os.Getenv("RABBITMQ_URL")

	url := "amqp://" + amqpUsername + ":" + amqpPassowd + "@" + amqpUrl

	config := rabbitmq.Config{Url: url}
	conn, err := rabbitmq.NewConnection(config)
	util.LogIfErr(err)
	defer conn.Conn.Close()

	ctrl := rabbitmq.NewCtrl()

	ctrl.SetupChannelAndQueue("collector-return", conn.Conn)

	ct := crontab.NewCrontab(ctrl.SqlQueryChannel, ctrl.Channel, ctrl.Queue)
	ct.Run()

	ctrl.RunTimer()
	go ctrl.ListenInfluxChannel()
	go ctrl.ListenSqlQueryChannel()
	ctrl.RunCtrl()
}
