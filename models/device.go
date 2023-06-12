package models

type SNMPConnection struct {
	Target    string `json:"target"`
	Port      int    `json:"port"`
	Community string `json:"community"`
	Version   string `json:"version"`
}

type Pdu struct {
	Oid   string      `json:"oid"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type SqlQuery struct {
	Query string
	Args  []any
}
