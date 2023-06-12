package services

import "github.com/streadway/amqp"

type IRabbitMQCtrl interface {
	SetupChannelAndQueue(name string, amqpConn *amqp.Connection) error
	RunCtrl()
	ListenQueue()
	RunTimer()
	ListenInfluxChannel()
	ListenSqlQueryChannel()
}

var localRabbitMQCtrlServices IRabbitMQCtrl

func RabbitMQCtrl() IRabbitMQCtrl {
	if localRabbitMQCtrlServices == nil {
		panic("impl not found for IRabbitMQCtrl")
	}
	return localRabbitMQCtrlServices
}

func RegisterRabbitMQCtrl(i IRabbitMQCtrl) {
	localRabbitMQCtrlServices = i
}
