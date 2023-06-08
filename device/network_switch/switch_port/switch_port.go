package switch_port

import (
	"collector-backend/device"
)

type Status uint8

const (
	StatusUp           Status = 1
	StatusDown         Status = 2
	StatusTesting      Status = 3
	StatusUnknown      Status = 4
	StatusDormant      Status = 5
	StatusNotPresent   Status = 6
	StatusLowLayerDown Status = 7
)

type SwitchPort struct {
	ID                   uint64                `json:"id"`
	PortIndex            uint64                `json:"port_index"`
	Oids                 map[string]string     `json:""`
	Pdus                 []device.Pdu          `json:"pdus"`
	Connection           device.SNMPConnection `json:"-"`
	RevertedOriginalOids map[string]string     `json:"-"`
}
