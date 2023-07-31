package models

import (
	"sync"

	client "github.com/influxdata/influxdb1-client"
)

type MyPoint struct {
	Wg    *sync.WaitGroup
	Point client.Point
}
