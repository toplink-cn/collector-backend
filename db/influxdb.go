package db

import (
	"fmt"
	"net/url"
	"sync"

	client "github.com/influxdata/influxdb1-client"
)

const influxClientCap = 10

type InfluxDBConnection struct {
	InfluxDBClientChan chan *client.Client
}

var once sync.Once
var internalInfluxDBClient *InfluxDBConnection

func NewInfluxDBConnection() *InfluxDBConnection {
	once.Do(func() {
		internalInfluxDBClient = &InfluxDBConnection{}
		internalInfluxDBClient.InfluxDBClientChan = make(chan *client.Client, influxClientCap)

		sockUrl := "/app/run/influxdb.sock"
		u, err := url.Parse("http://localhost")
		if err != nil {
			fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
		}
		config := client.Config{
			URL:        *u,
			UnixSocket: sockUrl,
		}

		for i := 0; i < influxClientCap; i++ {
			c, err := client.NewClient(config)
			if err != nil {
				fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
			}
			internalInfluxDBClient.InfluxDBClientChan <- c
		}
		fmt.Println("InfluxDBClientChan len: ", len(internalInfluxDBClient.InfluxDBClientChan))
	})

	return internalInfluxDBClient
}

func (ic InfluxDBConnection) GetClient() *client.Client {
	return <-ic.InfluxDBClientChan
}

func (ic InfluxDBConnection) CloseClient(c *client.Client) {
	ic.InfluxDBClientChan <- c
}

// func GetInfluxDbConnection() *client.Client {
// 	sockUrl := "/app/run/influxdb.sock"
// 	u, err := url.Parse("http://localhost")
// 	if err != nil {
// 		fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
// 	}
// 	config := client.Config{
// 		URL:        *u,
// 		UnixSocket: sockUrl,
// 	}
// 	c, err := client.NewClient(config)
// 	if err != nil {
// 		fmt.Printf("unexpected error.  expected %v, actual %v \n", nil, err)
// 	}
// 	return c
// }
