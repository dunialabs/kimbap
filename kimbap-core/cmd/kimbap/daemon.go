package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
	"github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/spf13/cobra"
)

func newDaemonCommand() *cobra.Command {
	var daemonToken string
	var daemonTokenFile string

	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Start a persistent runtime daemon (unix socket)",
		Long:  "Keeps the runtime alive to avoid cold-start overhead on every kimbap call. Listens on a unix socket.",
		RunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(daemonTokenFile) != "" {
				data, readErr := os.ReadFile(daemonTokenFile)
				if readErr != nil {
					return fmt.Errorf("read token file: %w", readErr)
				}
				daemonToken = strings.TrimSpace(string(data))
				if daemonToken == "" {
					return fmt.Errorf("token file %q is empty", daemonTokenFile)
				}
			}
			if strings.TrimSpace(daemonToken) == "" {
				daemonToken = strings.TrimSpace(os.Getenv("KIMBAP_DAEMON_TOKEN"))
			}

			cfg, err := loadAppConfig()
			if err != nil {
				return err
			}

			rt, err := buildRuntimeFromConfig(cfg)
			if err != nil {
				return fmt.Errorf("build runtime: %w", err)
			}

			socketPath := daemonSocketPath(cfg.DataDir)
			if err := os.MkdirAll(filepath.Dir(socketPath), 0o755); err != nil {
				return fmt.Errorf("create socket dir: %w", err)
			}

			lockPath := socketPath + ".lock"
			lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
			if err != nil {
				return fmt.Errorf("open daemon lock: %w", err)
			}
			defer lockFile.Close()
			if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
				return fmt.Errorf("daemon already starting on %s", socketPath)
			}
			defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

			listener, err := net.Listen("unix", socketPath)
			if err != nil {
				if conn, dialErr := net.Dial("unix", socketPath); dialErr == nil {
					conn.Close()
					return fmt.Errorf("daemon already running on %s", socketPath)
				}
				_ = os.Remove(socketPath)
				listener, err = net.Listen("unix", socketPath)
			}
			if err != nil {
				return fmt.Errorf("listen unix socket: %w", err)
			}
			defer listener.Close()
			defer os.Remove(socketPath)

			if err := os.Chmod(socketPath, 0o600); err != nil {
				return fmt.Errorf("chmod socket: %w", err)
			}

			mux := http.NewServeMux()
			mux.HandleFunc("/call", daemonCallHandler(rt))
			mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "pid": os.Getpid()})
			})
			mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "shutting_down"})
				go func() {
					time.Sleep(100 * time.Millisecond)
					_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
				}()
			})

			server := &http.Server{Handler: daemonAuthMiddleware(daemonToken, mux)}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			go func() {
				<-ctx.Done()
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = server.Shutdown(shutdownCtx)
			}()

			authStatus := "no auth"
			if strings.TrimSpace(daemonToken) != "" {
				authStatus = "token auth enabled"
			}

			_, _ = fmt.Fprintf(os.Stderr, "kimbap daemon listening on %s (pid %d, %s)\n", socketPath, os.Getpid(), authStatus)

			if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("serve: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&daemonToken, "token", "", "require this token in X-Kimbap-Token header (visible in process list; prefer --token-file)")
	cmd.Flags().StringVar(&daemonTokenFile, "token-file", "", "read daemon auth token from file (recommended over --token)")

	return cmd
}

type daemonCallRequest struct {
	Action string         `json:"action"`
	Input  map[string]any `json:"input"`
}

func daemonCallHandler(rt *runtime.Runtime) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req daemonCallRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
			return
		}

		if strings.TrimSpace(req.Action) == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "action name required"})
			return
		}

		requestID := fmt.Sprintf("req_%d", time.Now().UTC().UnixNano())
		execReq := actions.ExecutionRequest{
			RequestID:      requestID,
			IdempotencyKey: requestID,
			TenantID:       defaultTenantID(),
			Principal: actions.Principal{
				ID:        "daemon",
				TenantID:  defaultTenantID(),
				AgentName: "kimbap-daemon",
				Type:      "operator",
			},
			Action: actions.ActionDefinition{Name: req.Action},
			Input:  req.Input,
			Mode:   actions.ModeCall,
		}

		result := rt.Execute(r.Context(), execReq)

		w.Header().Set("Content-Type", "application/json")
		if result.Status != actions.StatusSuccess {
			status := result.HTTPStatus
			if status < 100 || status > 999 {
				status = http.StatusInternalServerError
			}
			w.WriteHeader(status)
		}
		_ = json.NewEncoder(w).Encode(result)
	}
}

func daemonSocketPath(dataDir string) string {
	return filepath.Join(dataDir, "daemon.sock")
}

func daemonAuthMiddleware(token string, next http.Handler) http.Handler {
	if strings.TrimSpace(token) == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		provided := strings.TrimSpace(r.Header.Get("X-Kimbap-Token"))
		if !constantTimeEqual(provided, token) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "invalid or missing X-Kimbap-Token"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func constantTimeEqual(a, b string) bool {
	aHash := sha256.Sum256([]byte(a))
	bHash := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(aHash[:], bHash[:]) == 1
}
