package db

import (
	"fmt"
	"log"
	"net/url"
	"os"

	client "github.com/influxdata/influxdb1-client"
	"github.com/joho/godotenv"
)

func GetInfluxDbConnection() *client.Client {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("无法加载 .env 文件")
	}

	// 访问环境变量
	sockUrl := os.Getenv("INFLUXDB_UNIXSOCKET")
	u, err := url.Parse("http://localhost")
	if err != nil {
		fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
	}
	config := client.Config{
		URL:        *u,
		UnixSocket: sockUrl,
	}
	c, err := client.NewClient(config)
	if err != nil {
		fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
	}
	return c
}
