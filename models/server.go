package models

import "time"

type Server struct {
	ID           uint64         `json:"id"`
	CabinetID    uint64         `json:"cabinet_id"`
	Connection   IPMIConnection `json:"-"`
	PowerStatus  string         `json:"power_status"`
	PowerReading int            `json:"power_reading"`
	Time         time.Time      `json:"time"`
}

type IPMIConnection struct {
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Interface string `json:"interface"`
}
