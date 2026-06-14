package api

import "encoding/json"

type InMessage struct {
	Action Action          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type OutMessage struct {
	Action Action `json:"action"`
	Data   any    `json:"data"`
}
