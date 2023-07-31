package main

import (
	"collector-backend/pkg/crontab"
	"collector-backend/pkg/rabbitmq"
	"collector-backend/util"
	"log"
)

func main() {
	run()
}

func run() {
	// 访问环境变量
	// amqpUsername := os.Getenv("RABBITMQ_USERNAME")
	// amqpPassowd := os.Getenv("RABBITMQ_PASSWORD")
	// amqpUrl := os.Getenv("RABBITMQ_URL")

	// url := "amqp://" + amqpUsername + ":" + amqpPassowd + "@" + amqpUrl
	// url := "amqps://" + amqpUsername + ":" + amqpPassowwd + "@" + amqpUrl
	url := "amqp://guest:guest@rabbitmq:5672/"
	log.Println("amqp url: ", url)

	config := rabbitmq.Config{Url: url}
	conn, err := rabbitmq.NewConnection(config)
	// conn, err := rabbitmq.NewConnectionWithTLS(config)
	util.LogIfErr(err)
	defer conn.Conn.Close()

	notifyCtrl := rabbitmq.NewCtrl()
	notifyCtrl.SetupChannelAndQueue("collector-notify", conn.Conn)

	returnCtrl := rabbitmq.NewCtrl()
	returnCtrl.SetupChannelAndQueue("collector-return", conn.Conn)
	returnCtrl.NotifyChannel = notifyCtrl.Channel
	returnCtrl.NotifyQueue = notifyCtrl.Queue

	ct := crontab.NewCrontab(returnCtrl.SqlQueryChannel, returnCtrl.Channel, returnCtrl.Queue)
	ct.Run()

	returnCtrl.RunTimer()
	go returnCtrl.ListenInfluxChannel()
	go returnCtrl.ListenSqlQueryChannel()
	go returnCtrl.ListenNotificationChannel()
	returnCtrl.RunCtrl()
}
