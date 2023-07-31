package models

import "time"

type Notification struct {
	Type  string `json:"type"`
	RelID uint64 `json:"rel_id"`
	Time  time.Time
}
