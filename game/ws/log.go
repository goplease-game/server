package ws

import "log"

type ActionLogger struct {
	enabled bool
}

func NewActionLogger(enabled bool) *ActionLogger {
	return &ActionLogger{enabled: enabled}
}

func (l *ActionLogger) log(format string, args ...any) {
	if l.enabled {
		log.Printf(format, args...)
	}
}

func (l *ActionLogger) Received(playerName, action string) {
	l.log("%s: -> %s", playerName, action)
}

func (l *ActionLogger) Sent(playerName, action string) {
	l.log("%s: <- %s", playerName, action)
}

func (l *ActionLogger) Event(playerName, event string) {
	l.log("%s: %s", playerName, event)
}
