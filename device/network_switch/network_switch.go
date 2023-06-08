package network_switch

import (
	"collector-backend/db"
	"collector-backend/device"
	"collector-backend/device/network_switch/switch_port"
	"encoding/json"
	"fmt"
	"os"
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
	if err != nil {
		fmt.Printf("无法解析JSON数据: %v", err)
		return
	}
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
		fmt.Printf("unexpected error.  expected %v, actual %v", nil, err)
	}
	if r != nil {
		fmt.Printf("unexpected response. expected %v, actual %v", nil, r)
	}

	fmt.Println("done")
}

func writeLocalData(content string) {
	// 打开文件，如果文件不存在则创建，文件已存在则截断内容
	file, err := os.OpenFile("./data.txt", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println("无法打开文件:", err)
		return
	}
	defer file.Close()

	// 将文本内容写入文件
	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println("写入文件失败:", err)
		return
	}

	fmt.Println("文本写入成功！")
}
