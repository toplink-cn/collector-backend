package system_collect_return

import (
	"collector-backend/models"
	model_system "collector-backend/models/system"
	"collector-backend/util"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	client "github.com/influxdata/influxdb1-client"
)

type SystemCollectReturn struct {
	InfluxPointChannel chan client.Point
	SqlQueryChannel    chan models.SqlQuery
}

func NewSystemCollectReturn(pointChannel chan client.Point, SqlQueryChannel chan models.SqlQuery) *SystemCollectReturn {
	return &SystemCollectReturn{
		InfluxPointChannel: pointChannel,
		SqlQueryChannel:    SqlQueryChannel,
	}
}

func (scr *SystemCollectReturn) HandleCollectReturn(data string) error {
	var s model_system.SystemInfo
	err := json.Unmarshal([]byte(data), &s)
	util.FailOnError(err, "无法解析JSON数据")

	var t string
	var id string
	if s.ID > 0 {
		t = "slave"
		id = strconv.Itoa(int(s.ID))
	} else {
		t = "master"
		id = "0"
	}

	for _, parame := range s.Parames {
		for key, val := range parame.Value.(map[string]interface{}) {
			field := reflect.ValueOf(val)
			// fmt.Printf("字段名称：%s，字段值：%v，类型: %v \n", key, val, field.Kind())
			switch field.Kind() {
			case reflect.Float64:
				p := client.Point{
					Measurement: "server_power",
					Tags: map[string]string{
						"type":     t,
						"slave_id": id,
						"parame":   "server_" + parame.Key,
						"label":    key,
					},
					Time: s.Time,
					Fields: map[string]interface{}{
						"value": val,
					},
				}
				fmt.Println(p)
				scr.InfluxPointChannel <- p
			case reflect.Map:
				for k, v := range val.(map[string]interface{}) {
					p := client.Point{
						Measurement: "server_power",
						Tags: map[string]string{
							"type":     t,
							"slave_id": id,
							"parame":   "server_" + parame.Key,
							"label":    key,
							"status":   k,
						},
						Time: s.Time,
						Fields: map[string]interface{}{
							"value": v,
						},
					}
					fmt.Println(p)
					scr.InfluxPointChannel <- p
				}
			}
		}
	}

	return nil
}
