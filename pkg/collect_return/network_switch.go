package collect_return

import (
	"collector-backend/models"
	"collector-backend/util"
	"encoding/json"
	"strconv"
	"time"

	client "github.com/influxdata/influxdb1-client"
)

type SwitchCollectReturn struct {
	InfluxPointChannel chan client.Point
	SqlQueryChannel    chan models.SqlQuery
}

func NewSwitchCollectReturn(pointChannel chan client.Point, SqlQueryChannel chan models.SqlQuery) *SwitchCollectReturn {
	return &SwitchCollectReturn{
		InfluxPointChannel: pointChannel,
		SqlQueryChannel:    SqlQueryChannel,
	}
}

func (scr *SwitchCollectReturn) HandleCollectReturn(data string) error {
	// fmt.Println(data)
	var ns models.NetworkSwitch
	err := json.Unmarshal([]byte(data), &ns)
	util.FailOnError(err, "无法解析JSON数据")
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
		scr.InfluxPointChannel <- p
	}

	// port
	directions := map[string]string{"in": "in", "out": "out"}
	mapStatus := map[string]string{"disconnected": "disconnected", "disabled": "disabled"}
	for _, port := range ns.Ports {
		sql_query := models.SqlQuery{
			Query: "UPDATE switch_ports SET updated_at = ?",
			Args:  []any{time.Now()},
		}
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
				scr.InfluxPointChannel <- p
				continue
			}
			_, ok = mapStatus[pdu.Key]
			if ok {
				sql_query.Query += " , " + pdu.Key + " = ?"
				_status := 0
				if int(_val.(float64)) == 2 {
					_status = 1
				}
				sql_query.Args = append(sql_query.Args, _status)
				continue
			}
		}
		sql_query.Query += " where id = ?"
		sql_query.Args = append(sql_query.Args, port.ID)
		scr.SqlQueryChannel <- sql_query
	}

	return nil
}
