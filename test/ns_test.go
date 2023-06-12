package test

import (
	database "collector-backend/db"
	"collector-backend/models"
	"collector-backend/util"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	client "github.com/influxdata/influxdb1-client"
)

func TestWritePoint(t *testing.T) {
	c := database.GetInfluxDbConnection()

	points := []client.Point{}

	p := client.Point{
		Measurement: "switch_monitor",
		Tags: map[string]string{
			"switch_id": "1",
			"type":      "cpu",
		},
		Time: time.Now(),
		Fields: map[string]interface{}{
			"value": float64(11),
		},
	}
	fmt.Println(p)

	points = append(points, p)

	bp := client.BatchPoints{
		Points:   points,
		Database: "dcim",
	}

	r, err := c.Write(bp)
	if err != nil {
		t.Fatalf("unexpected error.  expected %v, actual %v", nil, err)
	}
	if r != nil {
		t.Fatalf("unexpected response. expected %v, actual %v", nil, r)
	}

	fmt.Println("done")
}

func TestClient(t *testing.T) {
	sockUrl := "/Users/pange/run/influxdb.sock"
	u, err := url.Parse("http://localhost")
	if err != nil {
		t.Fatalf("unexpected error.  expected %v, actual %v \n", nil, err)
	}
	config := client.Config{
		URL:        *u,
		UnixSocket: sockUrl,
	}
	c, err := client.NewClient(config)
	if err != nil {
		t.Fatalf("unexpected error.  expected %v, actual %v \n", nil, err)
	}
	d, version, err := c.Ping()
	if err != nil {
		t.Fatalf("unexpected error.  expected %v, actual %v \n", nil, err)
	}
	if d.Nanoseconds() == 0 {
		t.Fatalf("expected a duration greater than zero.  actual %v \n", d.Nanoseconds())
	}
	if version != "x.x" {
		t.Fatalf("unexpected version.  expected %s,  actual %v", "x.x \n", version)
	}
}

func TestData(t *testing.T) {
	// 打开文件
	file, err := os.Open("../data.txt")
	util.FailOnError(err, "无法打开文件")
	defer file.Close()
	// 读取文件内容
	content, err := io.ReadAll(file)
	util.FailOnError(err, "无法读取文件内容")
	// 解析JSON
	var ns models.NetworkSwitch
	err = json.Unmarshal(content, &ns)
	util.FailOnError(err, "无法解析JSON")

	c := database.GetInfluxDbConnection()

	fmt.Println("start add points")
	points := []client.Point{}
	// switch
	for _, pdu := range ns.Pdus {
		if pdu.Key == "" {
			continue
		}
		p := client.Point{
			Measurement: "switch_monitor",
			Tags: map[string]string{
				"switch_id": strconv.Itoa(int(ns.ID)),
				"type":      pdu.Key,
			},
			Time: ns.Time,
			Fields: map[string]interface{}{
				"value": pdu.Value.(float64),
			},
		}
		points = append(points, p)
	}

	// port
	directions := map[string]string{"in": "in", "out": "out"}
	for _, port := range ns.Ports {
		for _, pdu := range port.Pdus {
			if pdu.Key == "" {
				continue
			}
			_, ok := directions[pdu.Key]
			if ok {
				p := client.Point{
					Measurement: "flow",
					Tags: map[string]string{
						"type":      pdu.Key,
						"switch_id": strconv.Itoa(int(ns.ID)),
						"port_id":   strconv.Itoa(int(port.ID)),
					},
					Time: ns.Time,
					Fields: map[string]interface{}{
						"value": pdu.Value.(float64),
					},
				}
				points = append(points, p)
			} else {
				fmt.Println("need write into mysql", pdu.Key, pdu.Value)
			}

		}
	}

	// fmt.Println(points)
	bp := client.BatchPoints{
		Points:   points,
		Database: "dcim",
	}

	r, err := c.Write(bp)
	if err != nil {
		t.Fatalf("unexpected error.  expected %v, actual %v", nil, err)
	}
	if r != nil {
		t.Fatalf("unexpected response. expected %v, actual %v", nil, r)
	}

	fmt.Println("done")
}
