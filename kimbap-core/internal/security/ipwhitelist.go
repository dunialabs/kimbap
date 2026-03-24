package security

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	AllowAllCIDR = "0.0.0.0/0"
	AllowAllMode = "allow-all"
)

type IPWhitelistStore interface {
	LoadWhitelist(ctx context.Context) ([]string, error)
	AddIP(ctx context.Context, ip string) error
	RemoveIP(ctx context.Context, ip string) error
	ReplaceAll(ctx context.Context, ips []string) error
}

const ipWhitelistCacheTTL = 15 * time.Minute

type IPWhitelistService struct {
	mu          sync.RWMutex
	store       IPWhitelistStore
	list        []string
	mode        string
	lastLoaded  time.Time
	lastAttempt time.Time
}

func NewIPWhitelistService(store IPWhitelistStore) *IPWhitelistService {
	return &IPWhitelistService{
		store: store,
		list:  []string{AllowAllCIDR},
		mode:  AllowAllMode,
	}
}

func (s *IPWhitelistService) LoadFromDB() error {
	if s.store == nil {
		return nil
	}
	ips, err := s.store.LoadWhitelist(context.Background())
	if err != nil {
		return err
	}
	s.applyInMemory(ips)
	s.mu.Lock()
	s.lastLoaded = time.Now()
	s.mu.Unlock()
	return nil
}

func (s *IPWhitelistService) CheckIP(ip string) bool {
	s.refreshIfStale()
	normalized := normalizeIP(ip)

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.mode == AllowAllMode {
		return true
	}

	for _, entry := range s.list {
		if ipMatches(normalized, entry) {
			return true
		}
	}
	return false
}

func (s *IPWhitelistService) refreshIfStale() {
	s.mu.RLock()
	stale := time.Since(s.lastLoaded) > ipWhitelistCacheTTL
	recentAttempt := time.Since(s.lastAttempt) <= ipWhitelistCacheTTL
	s.mu.RUnlock()
	if !stale || recentAttempt {
		return
	}

	s.mu.Lock()
	if time.Since(s.lastAttempt) <= ipWhitelistCacheTTL {
		s.mu.Unlock()
		return
	}
	s.lastAttempt = time.Now()
	s.mu.Unlock()

	if err := s.LoadFromDB(); err != nil {
		log.Warn().Err(err).Msg("failed to refresh IP whitelist from DB")
	}
}

func (s *IPWhitelistService) AddIP(ip string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}

	if s.store != nil {
		if err := s.store.AddIP(context.Background(), ip); err != nil {
			return err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, existing := range s.list {
		if existing == ip {
			return nil
		}
	}
	s.list = append(s.list, ip)
	s.recomputeMode()
	return nil
}

func (s *IPWhitelistService) RemoveIP(ip string) error {
	ip = strings.TrimSpace(ip)

	if s.store != nil {
		if err := s.store.RemoveIP(context.Background(), ip); err != nil {
			return err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	filtered := make([]string, 0, len(s.list))
	for _, existing := range s.list {
		if existing != ip {
			filtered = append(filtered, existing)
		}
	}
	s.list = filtered
	s.recomputeMode()
	return nil
}

func (s *IPWhitelistService) ReplaceAll(ips []string) error {
	if s.store != nil {
		if err := s.store.ReplaceAll(context.Background(), ips); err != nil {
			return err
		}
	}
	s.applyInMemory(ips)
	return nil
}

func (s *IPWhitelistService) applyInMemory(ips []string) {

	copyList := make([]string, 0, len(ips))
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			copyList = append(copyList, ip)
		}
	}
	if len(copyList) == 0 {
		copyList = []string{AllowAllCIDR}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.list = copyList
	s.recomputeMode()
}

func (s *IPWhitelistService) GetAll() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.list))
	copy(out, s.list)
	return out
}

func (s *IPWhitelistService) recomputeMode() {
	s.mode = ""
	for _, entry := range s.list {
		if entry == AllowAllMode || entry == AllowAllCIDR {
			s.mode = AllowAllMode
			return
		}
	}
}

func normalizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ip = strings.Trim(ip, "[]")

	if strings.HasPrefix(ip, "::ffff:") {
		ip = strings.TrimPrefix(ip, "::ffff:")
	}
	if ip == "::1" {
		return "127.0.0.1"
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	return parsed.String()
}

func ipMatches(ip string, pattern string) bool {
	if pattern == AllowAllCIDR || pattern == AllowAllMode {
		return true
	}

	if strings.Contains(pattern, "/") {
		_, network, err := net.ParseCIDR(pattern)
		if err != nil {
			return false
		}
		parsed := net.ParseIP(ip)
		if parsed == nil {
			return false
		}
		return network.Contains(parsed)
	}

	return normalizeIP(ip) == normalizeIP(pattern)
}
