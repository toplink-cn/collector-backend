package main

import (
	"collector-backend/lib"
	"collector-backend/util"
	"log"

	"github.com/streadway/amqp"
)

func main() {
	run()
}

func run() {
	// 连接到RabbitMQ服务器
	conn, err := amqp.Dial("amqp://root:password@192.168.88.112:5672/")
	util.FailOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	var connection lib.Connection

	ch := connection.Init(conn)
	defer ch.Close()

	// 处理接收到的消息
	forever := make(chan bool)

	go connection.ListenQ(connection.Ch, connection.Q)

	log.Printf(" [*] Waiting for messages. To exit, press CTRL+C")
	<-forever
}
