package middleware

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/logger"
)

type ipWhitelistChecker interface {
	CheckIP(ip string) bool
	GetAll() []string
}

type IPWhitelistMiddleware struct {
	service ipWhitelistChecker
}

func NewIPWhitelistMiddleware(service ipWhitelistChecker) *IPWhitelistMiddleware {
	return &IPWhitelistMiddleware{service: service}
}

func (m *IPWhitelistMiddleware) Middleware(next http.Handler) http.Handler {
	log := logger.CreateLogger("IPWhitelistMiddleware")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := ClientIPFromRequest(r)

		allowed := func() (ok bool) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error().Interface("recover", rec).Msg("Error occurred, denying access by default (fail-closed)")
					ok = false
				}
			}()
			return m.service.CheckIP(clientIP)
		}()

		if !allowed {
			writeJSONRPCError(w, http.StatusForbidden, -32000, fmt.Sprintf("Access denied: IP %s not in whitelist", clientIP), nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ClientIPFromRequest(r *http.Request) string {
	remoteIP := remoteIPFromRequest(r)
	if isTrustedProxyIP(remoteIP) {
		if cfi := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cfi != "" && isTrustedCFSource(remoteIP) {
			candidate := normalizeIP(cfi)
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}

		if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
			if clientIP := clientIPFromXFF(xff); clientIP != "" {
				return clientIP
			}
		}

		if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
			candidate := normalizeIP(xri)
			if net.ParseIP(candidate) != nil {
				return candidate
			}
		}
	}

	return remoteIP
}

func clientIPFromXFF(xff string) string {
	parts := strings.Split(xff, ",")
	valid := make([]string, 0, len(parts))
	for _, part := range parts {
		candidate := normalizeIP(part)
		if net.ParseIP(candidate) != nil {
			valid = append(valid, candidate)
		}
	}
	if len(valid) == 0 {
		return ""
	}
	idx := len(valid) - 1
	for idx >= 0 && isTrustedProxyIP(valid[idx]) {
		idx--
	}
	if idx >= 0 {
		return valid[idx]
	}
	return valid[0]
}

func remoteIPFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return normalizeIP(host)
	}
	return normalizeIP(r.RemoteAddr)
}

func isTrustedProxyIP(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	if parsed.IsLoopback() {
		return true
	}
	for _, block := range trustedProxyCIDRs() {
		if block.Contains(parsed) {
			return true
		}
	}
	return false
}

func trustedProxyCIDRs() []*net.IPNet {
	return parseCIDRListEnv("KIMBAP_TRUSTED_PROXY_CIDRS")
}

func trustedCloudflareCIDRs() []*net.IPNet {
	return parseCIDRListEnv("KIMBAP_TRUSTED_CF_CIDRS")
}

func isTrustedCFSource(remoteIP string) bool {
	parsed := net.ParseIP(strings.TrimSpace(remoteIP))
	if parsed == nil {
		return false
	}
	for _, block := range trustedCloudflareCIDRs() {
		if block.Contains(parsed) {
			return true
		}
	}
	return false
}

func parseCIDRListEnv(envKey string) []*net.IPNet {
	raw := strings.TrimSpace(os.Getenv(envKey))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	trusted := make([]*net.IPNet, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if strings.Contains(item, "/") {
			_, cidr, err := net.ParseCIDR(item)
			if err == nil {
				trusted = append(trusted, cidr)
			}
			continue
		}
		ip := net.ParseIP(item)
		if ip == nil {
			continue
		}
		bits := 128
		if ip.To4() != nil {
			bits = 32
		}
		trusted = append(trusted, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	return trusted
}

func normalizeIP(ip string) string {
	ip = strings.TrimSpace(ip)
	ip = strings.Trim(ip, "[]")
	if strings.HasPrefix(ip, "::ffff:") {
		ip = strings.TrimPrefix(ip, "::ffff:")
	}
	if ip == "::1" {
		return "127.0.0.1"
	}
	if parsed := net.ParseIP(ip); parsed != nil {
		if v4 := parsed.To4(); v4 != nil {
			return v4.String()
		}
		return parsed.String()
	}
	return ip
}
