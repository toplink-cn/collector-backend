package main

import (
	"collector-backend/pkg/crontab"
	"collector-backend/pkg/rabbitmq"
	"collector-backend/util"
	"log"
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
	// url := "amqps://" + amqpUsername + ":" + amqpPassowd + "@" + amqpUrl
	log.Println("amqp url: ", url)

	config := rabbitmq.Config{Url: url}
	conn, err := rabbitmq.NewConnection(config)
	// conn, err := rabbitmq.NewConnectionWithTLS(config)
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
