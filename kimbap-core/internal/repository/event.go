package repository

import (
	"context"
	"errors"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

const maxReplayEvents = 10000

func NewEventRepository(db *gorm.DB) *EventRepository {
	if db == nil {
		db = database.DB
	}
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *database.Event) (*database.Event, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if event == nil {
		return nil, errors.New("event is required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if event.ExpiresAt.IsZero() {
		event.ExpiresAt = time.Now().Add(24 * time.Hour)
	}
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func (r *EventRepository) FindAfterEventID(ctx context.Context, streamID, afterEventID string) ([]database.Event, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	now := time.Now()
	var afterEvent database.Event
	err := r.db.WithContext(ctx).Where("stream_id = ? AND event_id = ? AND expires_at > ?", streamID, afterEventID, now).First(&afterEvent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			var events []database.Event
			err = r.db.WithContext(ctx).Where("stream_id = ? AND expires_at > ?", streamID, now).Order("created_at asc, id asc").Limit(maxReplayEvents).Find(&events).Error
			return events, err
		}
		return nil, err
	}

	var events []database.Event
	err = r.db.WithContext(ctx).Where("stream_id = ? AND expires_at > ? AND (created_at > ? OR (created_at = ? AND id > ?))", streamID, now, afterEvent.CreatedAt, afterEvent.CreatedAt, afterEvent.ID).Order("created_at asc, id asc").Limit(maxReplayEvents).Find(&events).Error
	return events, err
}

func (r *EventRepository) DeleteByStreamID(ctx context.Context, streamID string) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := r.db.WithContext(ctx).Where("stream_id = ?", streamID).Delete(&database.Event{})
	return result.RowsAffected, result.Error
}

func (r *EventRepository) DeleteExpired(ctx context.Context) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := r.db.WithContext(ctx).Where("expires_at < ?", time.Now()).Delete(&database.Event{})
	return result.RowsAffected, result.Error
}
