package models

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
	ID                   uint64            `json:"id"`
	PortIndex            uint64            `json:"port_index"`
	Oids                 map[string]string `json:""`
	Pdus                 []Pdu             `json:"pdus"`
	Connection           SNMPConnection    `json:"-"`
	RevertedOriginalOids map[string]string `json:"-"`
}

type SwitchPortModel struct {
	ID           int
	Disconnected int
	Disabled     int
}
