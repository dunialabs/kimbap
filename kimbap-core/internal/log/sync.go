package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

var logSyncHTTPClient = &http.Client{Timeout: 30 * time.Second}

type LogSyncService struct {
	mu             sync.Mutex
	repo           *repository.LogRepository
	proxyRepo      *repository.ProxyRepository
	proxyID        int
	lastSyncedID   int
	webhookURL     string
	syncTicker     *time.Ticker
	stopCh         chan struct{}
	doneCh         chan struct{}
	isShuttingDown bool
	isSyncing      bool
	tickerRunning  bool
	shutdownOnce   sync.Once
	syncCtx        context.Context
	syncCancel     context.CancelFunc
	log            zerolog.Logger
}

var (
	logSyncOnce sync.Once
	logSyncInst *LogSyncService
)

func GetLogSyncService() *LogSyncService {
	logSyncOnce.Do(func() {
		logSyncInst = &LogSyncService{
			repo:      repository.NewLogRepository(nil),
			proxyRepo: repository.NewProxyRepository(nil),
			log:       logger.CreateLogger("LogSyncService"),
		}
	})
	return logSyncInst
}

func (s *LogSyncService) Initialize() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	proxy, err := s.proxyRepo.FindFirst()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if proxy == nil {
		return nil
	}

	s.proxyID = proxy.ID
	s.lastSyncedID = proxy.LastSyncedLogID
	if proxy.LogWebhookURL != nil {
		s.webhookURL = *proxy.LogWebhookURL
	}
	if s.webhookURL == "" {
		return nil
	}

	s.startTickerLocked()
	return nil
}

func (s *LogSyncService) startTickerLocked() {
	if s.tickerRunning {
		return
	}
	s.syncTicker = time.NewTicker(time.Duration(SyncIntervalMS) * time.Millisecond)
	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.syncCtx, s.syncCancel = context.WithCancel(context.Background())
	s.tickerRunning = true
	stopCh := s.stopCh
	doneCh := s.doneCh
	ticker := s.syncTicker
	syncCtx := s.syncCtx
	go func() {
		defer close(doneCh)
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				_ = s.syncLogs(syncCtx)
			}
		}
	}()
}

func (s *LogSyncService) stopTickerLocked() chan struct{} {
	if !s.tickerRunning {
		return nil
	}
	doneCh := s.doneCh
	if s.syncTicker != nil {
		s.syncTicker.Stop()
		s.syncTicker = nil
	}
	if s.syncCancel != nil {
		s.syncCancel()
		s.syncCancel = nil
		s.syncCtx = nil
	}
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
	s.doneCh = nil
	s.tickerRunning = false
	return doneCh
}

func (s *LogSyncService) syncLogs(ctx context.Context) error {
	return s.syncLogsInternal(ctx, false)
}

func (s *LogSyncService) syncLogsInternal(ctx context.Context, allowDuringShutdown bool) error {
	s.mu.Lock()
	if (s.isShuttingDown && !allowDuringShutdown) || s.isSyncing || s.webhookURL == "" || s.proxyID == 0 {
		s.mu.Unlock()
		return nil
	}
	s.isSyncing = true
	startID := s.lastSyncedID + 1
	webhook := s.webhookURL
	proxyID := s.proxyID
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isSyncing = false
		s.mu.Unlock()
	}()

	logs, err := s.repo.FindLogsFromID(startID, SyncBatchSize)
	if err != nil {
		s.recordSyncFailure(err)
		return err
	}
	if len(logs) == 0 {
		return nil
	}

	if err := s.sendLogsToWebhook(ctx, webhook, logs); err != nil {
		s.recordSyncFailure(err)
		return err
	}

	lastID := logs[len(logs)-1].ID
	if _, err := s.proxyRepo.Update(proxyID, map[string]any{"last_synced_log_id": lastID}); err != nil {
		s.recordSyncFailure(err)
		return err
	}

	s.mu.Lock()
	s.lastSyncedID = lastID
	s.mu.Unlock()
	return nil
}

func (s *LogSyncService) sendLogsToWebhook(ctx context.Context, webhook string, logs []database.Log) error {
	payload := map[string]any{
		"logs":      logs,
		"count":     len(logs),
		"timestamp": time.Now().UnixMilli(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt <= RetryCount; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		httpCtx, cancel := context.WithTimeout(ctx, time.Duration(HTTPTimeoutMS)*time.Millisecond)
		req, reqErr := http.NewRequestWithContext(httpCtx, http.MethodPost, webhook, bytes.NewReader(body))
		if reqErr != nil {
			cancel()
			return reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "KIMBAP-MCP-Console-LogSync/1.0")

		resp, err := logSyncHTTPClient.Do(req)
		cancel()
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				_ = resp.Body.Close()
				return nil
			}
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			_ = resp.Body.Close()
		} else {
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			lastErr = err
		}

		if attempt < RetryCount {
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return ctx.Err()
			case <-timer.C:
			}
		}
	}

	if lastErr == nil {
		lastErr = errors.New("webhook sync failed")
	}
	return lastErr
}

func (s *LogSyncService) ReloadWebhookURL() error {
	proxy, err := s.proxyRepo.FindFirst()
	if err != nil {
		return err
	}

	webhookURL := ""
	if proxy != nil && proxy.LogWebhookURL != nil {
		webhookURL = *proxy.LogWebhookURL
	}

	s.mu.Lock()
	s.webhookURL = webhookURL
	doneCh := s.stopTickerLocked()
	s.mu.Unlock()

	if err := waitForWorkerDone(doneCh); err != nil {
		return err
	}
	if webhookURL == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isShuttingDown || s.webhookURL != webhookURL {
		return nil
	}
	s.startTickerLocked()
	return nil
}

func (s *LogSyncService) Shutdown() error {
	var doneCh chan struct{}
	runDrain := false
	s.shutdownOnce.Do(func() {
		s.mu.Lock()
		s.isShuttingDown = true
		doneCh = s.stopTickerLocked()
		s.mu.Unlock()
		runDrain = true
	})
	if !runDrain {
		return nil
	}

	if err := waitForWorkerDone(doneCh); err != nil {
		return err
	}

	drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer drainCancel()
	if err := s.syncLogsInternal(drainCtx, true); err != nil {
		return err
	}

	return nil
}
func waitForWorkerDone(doneCh chan struct{}) error {
	if doneCh == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ShutdownTimeout)*time.Millisecond)
	defer cancel()
	select {
	case <-doneCh:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("log sync worker shutdown timed out: %w", ctx.Err())
	}
}

func (s *LogSyncService) recordSyncFailure(err error) {
	if err == nil {
		return
	}
	s.log.Error().Err(err).Int("proxyId", s.proxyID).Int("lastSyncedId", s.lastSyncedID).Msg("failed to sync logs")
}
