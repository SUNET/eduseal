package model

type Heartbeat struct {
	AddTS   int64  `json:"add_ts"`
	Message string `json:"message"`
}
