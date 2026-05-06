// Package response provides structures and utilities for API responses.
package response

import "time"

// Success is a constant Status variable representing a successful operation.
var Success = Status{Success: true}

// Status represents a simple response structure indicating the success or failure of an operation.
type Status struct {
	Success bool `json:"success"`
}

// ServerStatus provides information about the running server instance.
type ServerStatus struct {
	Env     string    `json:"env"`
	Version string    `json:"version"`
	Time    time.Time `json:"time"`
}
