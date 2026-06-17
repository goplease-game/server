package ws

import "log"

// ActionLogger provides a lightweight conditional logger for websocket actions and events.
// It can be enabled or disabled via configuration to avoid noisy logging in production or tests.
type ActionLogger struct {
	enabled bool
}

// NewActionLogger creates a new ActionLogger instance.
// If enabled is false, all logging calls become no-ops.
func NewActionLogger(enabled bool) *ActionLogger {
	return &ActionLogger{enabled: enabled}
}

// Received logs an incoming action received from a player over websocket.
func (l *ActionLogger) Received(playerName, action string) {
	l.log("%s: -> %s", playerName, action)
}

// Sent logs an outgoing action sent to a player over websocket.
func (l *ActionLogger) Sent(playerName, action string) {
	l.log("%s: <- %s", playerName, action)
}

// Event logs a generic websocket-related event for a player.
func (l *ActionLogger) Event(playerName, event string) {
	l.log("%s: %s", playerName, event)
}

// log writes a formatted log message if logging is enabled.
// It acts as an internal helper used by higher-level logging methods.
func (l *ActionLogger) log(format string, args ...any) {
	if l.enabled {
		log.Printf(format, args...)
	}
}
