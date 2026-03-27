package jobs

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type ApprovalExpirer interface {
	ExpireStale(ctx context.Context) (int, error)
}

type Worker struct {
	interval time.Duration
	expirer  ApprovalExpirer
	logger   *slog.Logger
	stopCh   chan struct{}

	startOnce sync.Once
	stopOnce  sync.Once
	wg        sync.WaitGroup
}

func NewWorker(interval time.Duration, expirer ApprovalExpirer, logger *slog.Logger) *Worker {
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		interval: interval,
		expirer:  expirer,
		logger:   logger,
		stopCh:   make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) {
	if w == nil {
		return
	}
	w.startOnce.Do(func() {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			ticker := time.NewTicker(w.interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-w.stopCh:
					return
				case <-ticker.C:
					if w.expirer == nil {
						continue
					}
					expireCtx, expireCancel := context.WithTimeout(ctx, w.interval)
					expired, err := w.expirer.ExpireStale(expireCtx)
					expireCancel()
					if err != nil {
						w.logger.Warn("approval expiry job failed", "error", err)
						continue
					}
					w.logger.Debug("approval expiry job ran", "expired_count", expired)
				}
			}
		}()
	})
}

func (w *Worker) Stop() {
	if w == nil {
		return
	}
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
	w.wg.Wait()
}
