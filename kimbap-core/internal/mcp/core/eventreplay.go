package core

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
)

type EventReplayService struct {
	store *PersistentEventStore
}

func NewEventReplayService(store *PersistentEventStore) *EventReplayService {
	return &EventReplayService{store: store}
}

func (s *EventReplayService) ReplayAfter(ctx context.Context, w http.ResponseWriter, lastEventID string, sessionID string) error {
	if s.store == nil {
		return fmt.Errorf("EventStore not available for this session")
	}
	if strings.ContainsAny(sessionID, "\r\n") {
		return fmt.Errorf("invalid session id")
	}

	headers := w.Header()
	headers.Set("Content-Type", "text/event-stream")
	headers.Set("Cache-Control", "no-cache, no-transform")
	headers.Set("Connection", "keep-alive")
	headers.Set("Mcp-Session-Id", sessionID)
	headers.Set("mcp-session-id", sessionID)
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	_, err := s.store.ReplayEventsAfter(ctx, mcptypes.EventID(lastEventID), mcptypes.ReplayOptions{
		Send: func(_ context.Context, eventID mcptypes.EventID, message mcptypes.JSONRPCMessage) error {
			_, err := fmt.Fprintf(w, "event: message\nid: %s\ndata: %s\n\n", eventID, mustJSON(message))
			if err != nil {
				return err
			}
			if ok {
				flusher.Flush()
			}
			return nil
		},
	})
	return err
}
