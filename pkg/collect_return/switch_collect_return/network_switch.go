package switch_collect_return

import (
	"collector-backend/db"
	"collector-backend/models"
	"collector-backend/pkg/logger"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	client "github.com/influxdata/influxdb1-client"
)

type SwitchCollectReturn struct {
	InfluxPointChannel  chan *models.MyPoint
	SqlQueryChannel     chan *models.SqlQuery
	NotificationChannel chan *models.Notification
}

func NewSwitchCollectReturn(pointChannel chan *models.MyPoint, SqlQueryChannel chan *models.SqlQuery) *SwitchCollectReturn {
	return &SwitchCollectReturn{
		InfluxPointChannel: pointChannel,
		SqlQueryChannel:    SqlQueryChannel,
	}
}

func (scr *SwitchCollectReturn) HandleCollectReturn(data string) error {
	var ns models.NetworkSwitch
	err := json.Unmarshal([]byte(data), &ns)
	logger.LogIfErrWithMsg(err, "NetworkSwitch Unable To Parse JSON Data")
	wg := sync.WaitGroup{}
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
		wg.Add(1)
		scr.InfluxPointChannel <- &models.MyPoint{
			Wg:    &wg,
			Point: p,
		}
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
				lastVal, _ := scr.getLastPortFlow(ns.ID, port.ID, pdu.Key)
				curVal := _val.(float64)

				var diffVal float64
				if lastVal == curVal {
					diffVal = 0
				} else if curVal > lastVal {
					diffVal = curVal - lastVal
				} else {
					diffVal = curVal
				}

				p1 := client.Point{
					Measurement: "flow_total",
					Tags: map[string]string{
						"type":      pdu.Key,
						"switch_id": strconv.Itoa(int(ns.ID)),
						"port_id":   strconv.Itoa(int(port.ID)),
					},
					Time: ns.Time,
					Fields: map[string]interface{}{
						"value": diffVal,
					},
				}
				wg.Add(1)
				scr.InfluxPointChannel <- &models.MyPoint{
					Wg:    &wg,
					Point: p1,
				}

				p2 := client.Point{
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
				wg.Add(1)
				scr.InfluxPointChannel <- &models.MyPoint{
					Wg:    &wg,
					Point: p2,
				}
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
		scr.SqlQueryChannel <- &sql_query
	}

	wg.Wait()
	scr.NotificationChannel <- &models.Notification{
		Type:  "switch",
		RelID: ns.ID,
		Time:  time.Now(),
	}

	return nil
}

func (scr *SwitchCollectReturn) getLastPortFlow(switchId uint64, portId uint64, direction string) (float64, error) {
	conn := db.NewInfluxDBReadConnection()
	c := conn.GetClient()

	query := fmt.Sprintf("SELECT * FROM flow where switch_id='%s' and port_id='%s' and type='%s' order by time desc LIMIT 1", strconv.Itoa(int(switchId)), strconv.Itoa(int(portId)), direction)
	q := client.Query{
		Command:  query,
		Database: "dcim",
	}

	response, err := c.Query(q)
	if err != nil {
		conn.CloseClient(c)
		return 0, err
	}

	if response.Error() != nil {
		conn.CloseClient(c)
		return 0, response.Error()
	}

	vals := map[string]interface{}{}
	for _, serie := range response.Results[0].Series {
		for index, column := range serie.Columns {
			val := serie.Values[0][index]
			vals[column] = val
		}
	}
	var val float64
	v := reflect.ValueOf(vals["value"])
	if !v.IsValid() {
		errMsg := "value is invalid, query:" + query
		logger.Println(errMsg)
		conn.CloseClient(c)
		return val, errors.New(errMsg)
	}
	if !v.CanInterface() {
		logger.Printf("if is %#+v (%v)\n", v.Interface(), v.Interface() == nil)
		errMsg := "value cannot interface"
		logger.Println(errMsg)
		conn.CloseClient(c)
		return val, errors.New(errMsg)
	}
	switch typeStr := v.Type().String(); typeStr {
	case "json.Number":
		v, ok := vals["value"].(json.Number)
		if ok {
			if intValue, err := v.Int64(); err == nil {
				val = float64(intValue)
			} else if floatValue, err := v.Float64(); err == nil {
				val = float64(floatValue)
			} else {
				logger.Printf("Invalid number format: %v \n", v)
			}
		}
	case "string":
		i, err := strconv.Atoi(v.String())
		if err != nil {
			val = float64(i)
		}
	case "int", "int8", "int16", "int32", "int64":
		val = float64(v.Float())
	case "float", "float32", "float64":
		val = float64(v.Float())
	default:
		logger.Println("unexcepted value type")
	}
	conn.CloseClient(c)
	return val, nil
}
