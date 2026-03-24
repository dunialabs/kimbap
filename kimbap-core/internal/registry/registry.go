package registry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/skills"
)

type RegistrySource struct {
	Name    string
	BaseURL string
}

type LockEntry struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Source      string `json:"source"`
	SHA256      string `json:"sha256"`
	InstalledAt string `json:"installed_at"`
}

type Lockfile struct {
	Version string      `json:"version"`
	Entries []LockEntry `json:"entries"`
}

type VerifyResult struct {
	Name           string `json:"name"`
	Status         string `json:"status"`
	ExpectedSHA256 string `json:"expected_sha256"`
	ActualSHA256   string `json:"actual_sha256"`
	Message        string `json:"message"`
}

type Registry struct {
	skillsDir string
	lockPath  string
}

const maxManifestResponseBytes int64 = 4 << 20

var registryHTTPClient = &http.Client{Timeout: 30 * time.Second}

func NewRegistry(skillsDir string) *Registry {
	clean := filepath.Clean(skillsDir)
	return &Registry{
		skillsDir: clean,
		lockPath:  filepath.Join(clean, "skills.lock.json"),
	}
}

func (r *Registry) Install(ctx context.Context, source, ref string) (*LockEntry, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}
	if strings.TrimSpace(ref) == "" {
		return nil, fmt.Errorf("skill ref is required")
	}

	manifestYAML, err := r.fetchManifestYAML(ctx, source, ref)
	if err != nil {
		return nil, err
	}

	manifest, err := skills.ParseManifest(manifestYAML)
	if err != nil {
		return nil, fmt.Errorf("parse skill manifest: %w", err)
	}

	if err := os.MkdirAll(r.skillsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create skills directory: %w", err)
	}

	dstPath := filepath.Join(r.skillsDir, manifest.Name+".yaml")
	if err := os.WriteFile(dstPath, manifestYAML, 0o644); err != nil {
		return nil, fmt.Errorf("write installed skill: %w", err)
	}

	sum := sha256.Sum256(manifestYAML)
	entry := LockEntry{
		Name:        manifest.Name,
		Version:     manifest.Version,
		Source:      normalizeSource(source),
		SHA256:      hex.EncodeToString(sum[:]),
		InstalledAt: time.Now().UTC().Format(time.RFC3339),
	}

	lf, err := r.LoadLockfile()
	if err != nil {
		return nil, err
	}
	lf.Entries = upsertLockEntry(lf.Entries, entry)
	if err := r.SaveLockfile(lf); err != nil {
		return nil, err
	}

	return &entry, nil
}

func (r *Registry) Verify(ctx context.Context) ([]VerifyResult, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	lf, err := r.LoadLockfile()
	if err != nil {
		return nil, err
	}

	results := make([]VerifyResult, 0, len(lf.Entries))
	for _, entry := range lf.Entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		skillPath := filepath.Join(r.skillsDir, entry.Name+".yaml")
		manifestYAML, readErr := os.ReadFile(skillPath)
		if readErr != nil {
			results = append(results, VerifyResult{
				Name:           entry.Name,
				Status:         "fail",
				ExpectedSHA256: entry.SHA256,
				Message:        fmt.Sprintf("read skill file: %v", readErr),
			})
			continue
		}

		sum := sha256.Sum256(manifestYAML)
		actual := hex.EncodeToString(sum[:])
		status := "ok"
		msg := "digest matches lockfile"
		if !strings.EqualFold(actual, entry.SHA256) {
			status = "fail"
			msg = "digest mismatch"
		}

		results = append(results, VerifyResult{
			Name:           entry.Name,
			Status:         status,
			ExpectedSHA256: entry.SHA256,
			ActualSHA256:   actual,
			Message:        msg,
		})
	}

	return results, nil
}

func (r *Registry) LoadLockfile() (*Lockfile, error) {
	if r == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	lf := &Lockfile{Version: "1", Entries: []LockEntry{}}
	b, err := os.ReadFile(r.lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return lf, nil
		}
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	if err := json.Unmarshal(b, lf); err != nil {
		return nil, fmt.Errorf("parse lockfile: %w", err)
	}
	if strings.TrimSpace(lf.Version) == "" {
		lf.Version = "1"
	}
	if lf.Entries == nil {
		lf.Entries = []LockEntry{}
	}

	return lf, nil
}

func (r *Registry) SaveLockfile(lf *Lockfile) error {
	if r == nil {
		return fmt.Errorf("registry is nil")
	}
	if lf == nil {
		return fmt.Errorf("lockfile is nil")
	}
	if strings.TrimSpace(lf.Version) == "" {
		lf.Version = "1"
	}
	if lf.Entries == nil {
		lf.Entries = []LockEntry{}
	}
	sort.Slice(lf.Entries, func(i, j int) bool {
		return lf.Entries[i].Name < lf.Entries[j].Name
	})

	if err := os.MkdirAll(filepath.Dir(r.lockPath), 0o755); err != nil {
		return fmt.Errorf("create lockfile directory: %w", err)
	}
	b, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(r.lockPath, b, 0o644); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}
	return nil
}

func (r *Registry) Diff(oldManifest, newManifest []byte) string {
	if bytes.Equal(oldManifest, newManifest) {
		return "no changes"
	}

	oldLines := splitLines(oldManifest)
	newLines := splitLines(newManifest)
	ops := lcsDiff(oldLines, newLines)

	var b strings.Builder
	for _, op := range ops {
		switch op.Kind {
		case diffDelete:
			b.WriteString("- ")
			b.WriteString(op.Line)
			b.WriteByte('\n')
		case diffAdd:
			b.WriteString("+ ")
			b.WriteString(op.Line)
			b.WriteByte('\n')
		}
	}

	out := strings.TrimSpace(b.String())
	if out == "" {
		return "manifest changed"
	}
	return out
}

func (r *Registry) fetchManifestYAML(ctx context.Context, source, ref string) ([]byte, error) {
	ref = strings.TrimSpace(ref)
	source = normalizeSource(source)

	if source == "github" || source == "private" || strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ref, nil)
		if err != nil {
			return nil, fmt.Errorf("build manifest request: %w", err)
		}
		resp, err := registryHTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download manifest: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("download manifest: unexpected HTTP status %d", resp.StatusCode)
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, maxManifestResponseBytes+1))
		if err != nil {
			return nil, fmt.Errorf("read manifest response: %w", err)
		}
		if int64(len(b)) > maxManifestResponseBytes {
			return nil, fmt.Errorf("manifest response exceeded %d bytes", maxManifestResponseBytes)
		}
		return b, nil
	}

	b, err := os.ReadFile(ref)
	if err != nil {
		return nil, fmt.Errorf("read local manifest: %w", err)
	}
	return b, nil
}

func normalizeSource(source string) string {
	clean := strings.ToLower(strings.TrimSpace(source))
	if clean == "" {
		return "local"
	}
	return clean
}

func upsertLockEntry(entries []LockEntry, entry LockEntry) []LockEntry {
	for idx := range entries {
		if entries[idx].Name == entry.Name {
			entries[idx] = entry
			return entries
		}
	}
	return append(entries, entry)
}

func splitLines(b []byte) []string {
	if len(b) == 0 {
		return []string{}
	}
	text := strings.ReplaceAll(string(b), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return []string{}
	}
	return strings.Split(text, "\n")
}

type diffKind int

const (
	diffEqual diffKind = iota
	diffAdd
	diffDelete
)

type diffOp struct {
	Kind diffKind
	Line string
}

func lcsDiff(a, b []string) []diffOp {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := m - 1; i >= 0; i-- {
		for j := n - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	ops := make([]diffOp, 0, m+n)
	i, j := 0, 0
	for i < m && j < n {
		switch {
		case a[i] == b[j]:
			ops = append(ops, diffOp{Kind: diffEqual, Line: a[i]})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			ops = append(ops, diffOp{Kind: diffDelete, Line: a[i]})
			i++
		default:
			ops = append(ops, diffOp{Kind: diffAdd, Line: b[j]})
			j++
		}
	}
	for ; i < m; i++ {
		ops = append(ops, diffOp{Kind: diffDelete, Line: a[i]})
	}
	for ; j < n; j++ {
		ops = append(ops, diffOp{Kind: diffAdd, Line: b[j]})
	}

	return ops
}
