package models

import (
	"time"
)

type NetworkSwitch struct {
	ID         uint64            `json:"id"`
	Connection SNMPConnection    `json:"connection"`
	Oids       map[string]string `json:"oids"`
	Ports      []SwitchPort      `json:"ports"`
	PortOids   map[string]string `json:"port_oids"`
	Pdus       []Pdu             `json:"pdus"`
	Time       time.Time         `json:"time"`
}
