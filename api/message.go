// Package api ...
// TODO find a better place for this code
package api //nolint:revive

import "encoding/json"

// InMessage is a message received from a client, with the payload left
// as raw JSON until the action is known.
type InMessage struct {
	Action Action          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

// OutMessage is a message sent to a client.
type OutMessage struct {
	Action Action `json:"action"`
	Data   any    `json:"data"`
}
