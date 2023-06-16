package main

import (
	"collector-backend/pkg/crontab"
	"collector-backend/pkg/rabbitmq"
	"collector-backend/util"
)

func main() {
	run()

}

func run() {
	config := rabbitmq.Config{Url: "amqp://root:password@192.168.88.112:5672/"}
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
