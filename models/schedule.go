package models

type Schedule struct {
	ID         uint64 `json:"id"`
	Label      string `json:"label"`
	Expression string `json:"expression"`
	Type       string `json:"type"`
	Payload    string `json:"payload"`
	Priority   int64  `json:"priority"`
	Disabled   int64  `json:"disabled"`
}
