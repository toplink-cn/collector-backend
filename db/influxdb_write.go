package db

import (
	"fmt"
	"net/url"
	"sync"

	client "github.com/influxdata/influxdb1-client"
)

const influxWriteClientCap = 20

type InfluxDBWriteConnection struct {
	InfluxDBWriteClientChan chan *client.Client
}

var write_once sync.Once
var internalInfluxWriteDBClient *InfluxDBWriteConnection

func NewInfluxDBWriteConnection() *InfluxDBWriteConnection {
	write_once.Do(func() {
		internalInfluxWriteDBClient = &InfluxDBWriteConnection{}
		internalInfluxWriteDBClient.InfluxDBWriteClientChan = make(chan *client.Client, influxWriteClientCap)

		sockUrl := "/app/run/influxdb.sock"
		u, err := url.Parse("http://localhost")
		if err != nil {
			fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
		}
		config := client.Config{
			URL:        *u,
			UnixSocket: sockUrl,
		}

		for i := 0; i < influxWriteClientCap; i++ {
			c, err := client.NewClient(config)
			if err != nil {
				fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
			}
			internalInfluxWriteDBClient.InfluxDBWriteClientChan <- c
		}
		fmt.Println("InfluxDBWriteClientChan len: ", len(internalInfluxWriteDBClient.InfluxDBWriteClientChan))
	})

	return internalInfluxWriteDBClient
}

func (ic InfluxDBWriteConnection) GetClient() *client.Client {
	return <-ic.InfluxDBWriteClientChan
}

func (ic InfluxDBWriteConnection) CloseClient(c *client.Client) {
	ic.InfluxDBWriteClientChan <- c
}
