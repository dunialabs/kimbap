package log

import (
	"encoding/json"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/types"
)

type ServerLogger struct {
	serverID string
	service  *LogService
}

func NewServerLogger(serverID string) *ServerLogger {
	return &ServerLogger{serverID: serverID, service: GetLogService()}
}

func (l *ServerLogger) LogServerLifecycle(action int, errMsg string) {
	serverID := l.serverID
	l.service.EnqueueLog(database.Log{
		Action:   action,
		ServerID: &serverID,
		Error:    errMsg,
	})
}

func (l *ServerLogger) LogServerCapabilityUpdate(params any) {
	serverID := l.serverID
	req := ""
	if params != nil {
		if b, err := json.Marshal(params); err == nil {
			req = string(b)
		}
	}
	l.service.EnqueueLog(database.Log{
		Action:        types.MCPEventLogTypeServerCapabilityUpdate,
		ServerID:      &serverID,
		RequestParams: req,
	})
}

func (l *ServerLogger) LogError(action int, errorMsg string) {
	serverID := l.serverID
	l.service.EnqueueLog(database.Log{
		Action:   action,
		ServerID: &serverID,
		Error:    errorMsg,
	})
}
