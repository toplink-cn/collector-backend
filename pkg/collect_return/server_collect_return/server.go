package server_collect_return

import (
	"collector-backend/models"
	"collector-backend/util"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	client "github.com/influxdata/influxdb1-client"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ServerCollectReturn struct {
	InfluxPointChannel chan *models.MyPoint
	SqlQueryChannel    chan *models.SqlQuery
}

func NewServerCollectReturn(pointChannel chan *models.MyPoint, SqlQueryChannel chan *models.SqlQuery) *ServerCollectReturn {
	return &ServerCollectReturn{
		InfluxPointChannel: pointChannel,
		SqlQueryChannel:    SqlQueryChannel,
	}
}

func (scr *ServerCollectReturn) HandleCollectReturn(data string) error {
	var s models.Server
	err := json.Unmarshal([]byte(data), &s)
	util.FailOnError(err, "无法解析JSON数据")

	wg := sync.WaitGroup{}
	// power reading
	p := client.Point{
		Measurement: "server_power",
		//['type' => 'instant', 'server_id' => $server->id, 'cabinet_id' =>  $server->cabinet_id],
		Tags: map[string]string{
			"type":       "instant",
			"server_id":  strconv.Itoa(int(s.ID)),
			"cabinet_id": strconv.Itoa(int(s.CabinetID)),
		},
		Time: s.Time,
		Fields: map[string]interface{}{
			"value": float64(s.PowerReading),
		},
	}
	wg.Add(1)
	scr.InfluxPointChannel <- &models.MyPoint{
		Wg:    &wg,
		Point: p,
	}

	// power status
	if s.PowerStatus != "" {
		mapPowerStatus := map[string]string{"On": "On", "Off": "Off", "Unknown": "Unknown"}
		power_stats := cases.Title(language.Und, cases.NoLower).String(s.PowerStatus)
		_, ok := mapPowerStatus[power_stats]
		if ok {
			sql_query := models.SqlQuery{
				Query: "UPDATE servers SET updated_at = ?, power_status = ? where id = ?",
				Args:  []any{time.Now(), power_stats, s.ID},
			}
			scr.SqlQueryChannel <- &sql_query
		}
	}
	wg.Wait()

	return nil
}
