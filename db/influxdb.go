package db

import (
	"fmt"
	"net/url"

	client "github.com/influxdata/influxdb1-client"
)

func GetInfluxDbConnection() *client.Client {
	sockUrl := "/Users/pange/run/influxdb.sock"
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
