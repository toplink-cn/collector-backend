package db

import (
	"fmt"
	"net/url"
	"sync"

	client "github.com/influxdata/influxdb1-client"
)

const influxReadClientCap = 20

type InfluxDBReadConnection struct {
	InfluxDBReadClientChan chan *client.Client
}

var read_once sync.Once
var internalInfluxDBReadClient *InfluxDBReadConnection

func NewInfluxDBReadConnection() *InfluxDBReadConnection {
	read_once.Do(func() {
		internalInfluxDBReadClient = &InfluxDBReadConnection{}
		internalInfluxDBReadClient.InfluxDBReadClientChan = make(chan *client.Client, influxReadClientCap)

		sockUrl := "/app/run/influxdb.sock"
		u, err := url.Parse("http://localhost")
		if err != nil {
			fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
		}
		config := client.Config{
			URL:        *u,
			UnixSocket: sockUrl,
		}

		for i := 0; i < influxReadClientCap; i++ {
			c, err := client.NewClient(config)
			if err != nil {
				fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
			}
			internalInfluxDBReadClient.InfluxDBReadClientChan <- c
		}
		fmt.Println("InfluxDBReadClientChan len: ", len(internalInfluxDBReadClient.InfluxDBReadClientChan))
	})

	return internalInfluxDBReadClient
}

func (ic InfluxDBReadConnection) GetClient() *client.Client {
	return <-ic.InfluxDBReadClientChan
}

func (ic InfluxDBReadConnection) CloseClient(c *client.Client) {
	ic.InfluxDBReadClientChan <- c
}
