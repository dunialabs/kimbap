package log

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/rs/zerolog"
)

const (
	logBatchSize     = 100
	logFlushInterval = 5 * time.Second
	logQueueMaxSize  = 10000
)

type LogService struct {
	repo *repository.LogRepository

	mu             sync.Mutex
	logQueue       []database.Log
	isShuttingDown bool
	flushRunning   bool

	stopCh   chan struct{}
	doneCh   chan struct{}
	onceStop sync.Once
	flushWg  sync.WaitGroup
	log      zerolog.Logger
	dropped  int
}

var (
	logServiceOnce sync.Once
	logServiceInst *LogService
)

func GetLogService() *LogService {
	logServiceOnce.Do(func() {
		logServiceInst = newLogService(repository.NewLogRepository(nil))
	})
	return logServiceInst
}

func newLogService(repo *repository.LogRepository) *LogService {
	if repo == nil {
		repo = repository.NewLogRepository(nil)
	}

	s := &LogService{
		repo:     repo,
		logQueue: make([]database.Log, 0),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		log:      logger.CreateLogger("LogService"),
	}

	go s.runFlushTimer()
	return s
}

func (s *LogService) runFlushTimer() {
	ticker := time.NewTicker(logFlushInterval)
	defer ticker.Stop()
	defer close(s.doneCh)

	for {
		select {
		case <-ticker.C:
			s.requestFlush()
		case <-s.stopCh:
			return
		}
	}
}

func (s *LogService) EnqueueLog(entry database.Log) {
	s.mu.Lock()
	if s.isShuttingDown {
		s.mu.Unlock()
		if _, err := s.repo.Save(entry); err != nil {
			s.log.Error().Err(err).Msg("failed to save log during shutdown")
		}
		return
	}

	if len(s.logQueue) >= logQueueMaxSize {
		s.dropped++
		if s.dropped%100 == 1 {
			s.log.Warn().Int("dropped", s.dropped).Int("maxQueue", logQueueMaxSize).Msg("log queue full, dropping new entries")
		}
		s.mu.Unlock()
		return
	}

	s.logQueue = append(s.logQueue, entry)
	shouldFlush := len(s.logQueue) >= logBatchSize
	s.mu.Unlock()

	if shouldFlush {
		s.requestFlush()
	}
}

func (s *LogService) requestFlush() {
	s.mu.Lock()
	if s.isShuttingDown {
		s.mu.Unlock()
		return
	}
	if s.flushRunning {
		s.mu.Unlock()
		return
	}
	s.flushRunning = true
	s.flushWg.Add(1)
	s.mu.Unlock()

	go s.flushWorker()
}

func (s *LogService) flushWorker() {
	defer s.flushWg.Done()

	for {
		batch := s.dequeueBatch()
		if len(batch) == 0 {
			s.mu.Lock()
			s.flushRunning = false
			s.mu.Unlock()
			return
		}

		saved, err := s.flushBatch(batch)
		if err != nil {
			s.log.Error().Err(err).Msg("failed to flush logs")
			s.requeueBatch(batch[saved:])
			s.mu.Lock()
			s.flushRunning = false
			s.mu.Unlock()
			return
		}

		s.log.Debug().Int("count", len(batch)).Msg("flushed logs")
	}
}

func (s *LogService) dequeueBatch() []database.Log {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.logQueue) == 0 {
		return nil
	}

	n := len(s.logQueue)
	if n > logBatchSize {
		n = logBatchSize
	}

	batch := append([]database.Log(nil), s.logQueue[:n]...)
	s.logQueue = s.logQueue[n:]
	return batch
}

func (s *LogService) requeueBatch(batch []database.Log) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(batch) == 0 {
		return
	}

	requeued := make([]database.Log, 0, len(batch)+len(s.logQueue))
	requeued = append(requeued, batch...)
	requeued = append(requeued, s.logQueue...)
	if len(requeued) > logQueueMaxSize {
		overflow := len(requeued) - logQueueMaxSize
		requeued = requeued[:logQueueMaxSize]
		s.dropped += overflow
		if s.dropped%100 == 1 {
			s.log.Warn().Int("dropped", s.dropped).Int("maxQueue", logQueueMaxSize).Msg("log queue full, dropping requeued entries")
		}
	}
	s.logQueue = requeued
}

func (s *LogService) flushBatch(batch []database.Log) (int, error) {
	for i, entry := range batch {
		if _, err := s.repo.Save(entry); err != nil {
			return i, err
		}
	}
	return len(batch), nil
}

func (s *LogService) Shutdown(ctx context.Context) error {
	s.onceStop.Do(func() {
		s.mu.Lock()
		s.isShuttingDown = true
		s.mu.Unlock()
		close(s.stopCh)
	})

	select {
	case <-ctx.Done():
		s.log.Warn().Err(ctx.Err()).Msg("shutdown context cancelled while waiting for flush timer")
		return ctx.Err()
	case <-s.doneCh:
	}

	flushDone := make(chan struct{})
	go func() {
		s.flushWg.Wait()
		close(flushDone)
	}()
	select {
	case <-ctx.Done():
		s.log.Warn().Err(ctx.Err()).Msg("shutdown context cancelled while waiting for flush worker")
		return ctx.Err()
	case <-flushDone:
	}

	for {
		batch := s.dequeueBatch()
		if len(batch) == 0 {
			break
		}
		saved, err := s.flushBatch(batch)
		if err != nil {
			s.log.Error().Err(err).Msg("failed to flush logs")
			s.requeueBatch(batch[saved:])
			return err
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.log.Info().Msg("shutdown complete")
	return nil
}

func (s *LogService) GenerateUniformRequestID(sessionID string) string {
	ts := time.Now().UnixMilli()
	return fmt.Sprintf("%s_%d_%s", sessionID, ts, randomBase36(4))
}

func randomBase36(length int) string {
	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	if length <= 0 {
		return ""
	}
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			out[i] = '0'
			continue
		}
		out[i] = alphabet[n.Int64()]
	}
	return string(out)
}
