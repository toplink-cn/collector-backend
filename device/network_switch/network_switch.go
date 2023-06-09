package network_switch

import (
	"collector-backend/db"
	"collector-backend/device"
	"collector-backend/device/network_switch/switch_port"
	"collector-backend/util"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	client "github.com/influxdata/influxdb1-client"
)

type NetworkSwitch struct {
	ID         uint64                   `json:"id"`
	Connection device.SNMPConnection    `json:"connection"`
	Oids       map[string]string        `json:"oids"`
	Ports      []switch_port.SwitchPort `json:"ports"`
	PortOids   map[string]string        `json:"port_oids"`
	Pdus       []device.Pdu             `json:"pdus"`
	Time       time.Time                `json:"time"`
}

func HandleCollectReturn(data string) {
	// fmt.Println(data)
	var ns NetworkSwitch
	err := json.Unmarshal([]byte(data), &ns)
	util.FailOnError(err, "无法解析JSON数据")
	c := db.GetInfluxDbConnection()

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
	mysql_conn := db.GetMysqlConnection()
	defer mysql_conn.Close()
	directions := map[string]string{"in": "in", "out": "out"}
	mapStatus := map[string]string{"disconnected": "disconnected", "disabled": "disabled"}
	for _, port := range ns.Ports {
		port_query := "UPDATE switch_ports SET updated_at = ?"
		port_query_args := []any{time.Now()}
		for _, pdu := range port.Pdus {
			if pdu.Key == "" {
				continue
			}
			_val := pdu.Value
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
						"value": _val.(float64),
					},
				}
				points = append(points, p)
				continue
			}
			_, ok = mapStatus[pdu.Key]
			if ok {
				port_query += " , " + pdu.Key + " = ?"
				_status := 0
				if int(_val.(float64)) == 2 {
					_status = 1
				}
				port_query_args = append(port_query_args, _status)
				continue
			}
		}
		port_query += " where id = ?"
		port_query_args = append(port_query_args, port.ID)
		_, err := mysql_conn.Exec(port_query, port_query_args...)
		util.FailOnError(err, "执行sql update 出错")
	}

	// fmt.Println(points)
	bp := client.BatchPoints{
		Points:   points,
		Database: "dcim",
	}

	r, err := c.Write(bp)
	if err != nil {
		fmt.Printf("unexpected error.  expected %v, actual %v", nil, err)
	}
	if r != nil {
		fmt.Printf("unexpected response. expected %v, actual %v", nil, r)
	}

	fmt.Println("done")
}
