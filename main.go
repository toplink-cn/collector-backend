package main

import (
	"collector-backend/pkg/rabbitmq"
	"collector-backend/services"
	"collector-backend/util"
)

func main() {
	config := rabbitmq.Config{Url: "amqp://root:password@192.168.88.112:5672/"}
	conn, err := rabbitmq.NewConnection(config)
	util.LogIfErr(err)
	defer conn.Conn.Close()

	ctrl := services.RabbitMQCtrl()

	ctrl.SetupChannelAndQueue("collector-return", conn.Conn)
	ctrl.RunTimer()
	go ctrl.ListenInfluxChannel()
	go ctrl.ListenSqlQueryChannel()
	ctrl.RunCtrl()
}
