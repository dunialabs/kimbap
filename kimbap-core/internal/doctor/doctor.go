package doctor

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/policy"
	"gopkg.in/yaml.v3"

	_ "modernc.org/sqlite"
)

type CheckResult struct {
	Name    string
	Status  string
	Message string
}

type Doctor struct {
	dataDir    string
	configPath string
}

func NewDoctor(dataDir, configPath string) *Doctor {
	return &Doctor{dataDir: strings.TrimSpace(dataDir), configPath: strings.TrimSpace(configPath)}
}

func (d *Doctor) RunAll(ctx context.Context) []CheckResult {
	results := make([]CheckResult, 0, 7)

	configPath, configCheck := d.checkConfigFile()
	results = append(results, configCheck)

	cfg, _ := d.loadConfig()
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	if strings.TrimSpace(d.dataDir) != "" {
		prevDataDir := cfg.DataDir
		cfg.DataDir = d.dataDir
		if cfg.Vault.Path == filepath.Join(prevDataDir, "vault.db") {
			cfg.Vault.Path = filepath.Join(cfg.DataDir, "vault.db")
		}
		if cfg.Skills.Dir == filepath.Join(prevDataDir, "skills") {
			cfg.Skills.Dir = filepath.Join(cfg.DataDir, "skills")
		}
		if cfg.Policy.Path == filepath.Join(prevDataDir, "policy.yaml") {
			cfg.Policy.Path = filepath.Join(cfg.DataDir, "policy.yaml")
		}
	}
	if strings.TrimSpace(configPath) != "" {
		d.configPath = configPath
	}

	results = append(results, d.checkDataDirWritable(cfg.DataDir))
	results = append(results, d.checkVaultAccessible(ctx, cfg.Vault.Path))
	results = append(results, d.checkSkillsDir(cfg.Skills.Dir))
	results = append(results, d.checkPolicyFile(cfg.Policy.Path))
	results = append(results, d.checkCACertificate(cfg.Mode, cfg.DataDir))
	results = append(results, d.checkConnectivity(ctx, cfg.Mode, cfg.Auth.ServerURL))

	return results
}

func (d *Doctor) checkConfigFile() (string, CheckResult) {
	path, err := d.resolveConfigPath()
	if err != nil {
		return "", CheckResult{Name: "config file", Status: "fail", Message: err.Error()}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return path, CheckResult{Name: "config file", Status: "fail", Message: fmt.Sprintf("read config: %v", err)}
	}
	var raw any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return path, CheckResult{Name: "config file", Status: "fail", Message: fmt.Sprintf("invalid YAML: %v", err)}
	}
	if _, err := config.LoadKimbapConfigWithoutDefault(path); err != nil {
		return path, CheckResult{Name: "config file", Status: "fail", Message: err.Error()}
	}
	return path, CheckResult{Name: "config file", Status: "ok", Message: path}
}

func (d *Doctor) resolveConfigPath() (string, error) {
	if strings.TrimSpace(d.configPath) != "" {
		return d.configPath, nil
	}
	xdg := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	xdgPath := ""
	xdgIsDir := false
	if xdg != "" {
		xdgPath = filepath.Join(xdg, "kimbap", "config.yaml")
		if st, err := os.Stat(xdgPath); err == nil {
			if !st.IsDir() {
				return xdgPath, nil
			}
			xdgIsDir = true
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat xdg config path %q: %w", xdgPath, err)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		if xdgPath != "" {
			if xdgIsDir {
				return "", fmt.Errorf("config path is a directory: %s", xdgPath)
			}
			return xdgPath, nil
		}
		return "", fmt.Errorf("resolve user home directory")
	}
	legacyPath := filepath.Join(home, ".kimbap", "config.yaml")
	if xdgPath != "" {
		if st, err := os.Stat(legacyPath); err == nil {
			if st.IsDir() {
				return "", fmt.Errorf("legacy config path is a directory: %s", legacyPath)
			}
			return legacyPath, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat legacy config path %q: %w", legacyPath, err)
		}
		if xdgIsDir {
			return "", fmt.Errorf("config path is a directory: %s", xdgPath)
		}
		return xdgPath, nil
	}
	if st, err := os.Stat(legacyPath); err == nil && st.IsDir() {
		return "", fmt.Errorf("legacy config path is a directory: %s", legacyPath)
	}
	return legacyPath, nil
}

func (d *Doctor) loadConfig() (*config.KimbapConfig, error) {
	if strings.TrimSpace(d.configPath) != "" {
		return config.LoadKimbapConfigWithoutDefault(d.configPath)
	}
	return config.LoadKimbapConfig()
}

func (d *Doctor) checkDataDirWritable(dataDir string) CheckResult {
	st, err := os.Stat(dataDir)
	if err != nil {
		return CheckResult{Name: "data directory writable", Status: "fail", Message: err.Error()}
	}
	if !st.IsDir() {
		return CheckResult{Name: "data directory writable", Status: "fail", Message: "path is not a directory"}
	}
	file, err := os.CreateTemp(dataDir, "kimbap-doctor-*.tmp")
	if err != nil {
		return CheckResult{Name: "data directory writable", Status: "fail", Message: err.Error()}
	}
	_ = file.Close()
	_ = os.Remove(file.Name())
	return CheckResult{Name: "data directory writable", Status: "ok", Message: dataDir}
}

func (d *Doctor) checkVaultAccessible(ctx context.Context, vaultPath string) CheckResult {
	if strings.TrimSpace(vaultPath) == "" {
		return CheckResult{Name: "vault database accessible", Status: "fail", Message: "vault path is empty"}
	}
	st, err := os.Stat(vaultPath)
	if err != nil {
		return CheckResult{Name: "vault database accessible", Status: "fail", Message: err.Error()}
	}
	if st.IsDir() {
		return CheckResult{Name: "vault database accessible", Status: "fail", Message: "path is not a file"}
	}
	db, err := sql.Open("sqlite", vaultPath)
	if err != nil {
		return CheckResult{Name: "vault database accessible", Status: "fail", Message: err.Error()}
	}
	defer db.Close()
	var exists int
	if err := db.QueryRowContext(ctx, "SELECT 1 FROM secrets LIMIT 1").Scan(&exists); err != nil && err != sql.ErrNoRows {
		return CheckResult{Name: "vault database accessible", Status: "fail", Message: err.Error()}
	}
	return CheckResult{Name: "vault database accessible", Status: "ok", Message: vaultPath}
}

func (d *Doctor) checkSkillsDir(skillsDir string) CheckResult {
	st, err := os.Stat(skillsDir)
	if err != nil {
		return CheckResult{Name: "skills directory exists", Status: "fail", Message: err.Error()}
	}
	if !st.IsDir() {
		return CheckResult{Name: "skills directory exists", Status: "fail", Message: "path is not a directory"}
	}
	return CheckResult{Name: "skills directory exists", Status: "ok", Message: skillsDir}
}

func (d *Doctor) checkPolicyFile(path string) CheckResult {
	if strings.TrimSpace(path) == "" {
		return CheckResult{Name: "policy file valid", Status: "warn", Message: "policy path is empty"}
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return CheckResult{Name: "policy file valid", Status: "warn", Message: "policy file not present"}
	}
	if _, err := policy.ParseDocumentFile(path); err != nil {
		return CheckResult{Name: "policy file valid", Status: "fail", Message: err.Error()}
	}
	return CheckResult{Name: "policy file valid", Status: "ok", Message: path}
}

func (d *Doctor) checkCACertificate(mode, dataDir string) CheckResult {
	if !strings.EqualFold(strings.TrimSpace(mode), "proxy") {
		return CheckResult{Name: "CA certificate exists", Status: "warn", Message: "not required for current mode"}
	}
	path := filepath.Join(dataDir, "ca.crt")
	st, err := os.Stat(path)
	if err != nil {
		return CheckResult{Name: "CA certificate exists", Status: "fail", Message: err.Error()}
	}
	if st.IsDir() {
		return CheckResult{Name: "CA certificate exists", Status: "fail", Message: "ca.crt is a directory"}
	}
	return CheckResult{Name: "CA certificate exists", Status: "ok", Message: path}
}

func (d *Doctor) checkConnectivity(ctx context.Context, mode, serverURL string) CheckResult {
	if !strings.EqualFold(strings.TrimSpace(mode), "connected") {
		return CheckResult{Name: "server connectivity", Status: "warn", Message: "not required for current mode"}
	}
	if strings.TrimSpace(serverURL) == "" {
		return CheckResult{Name: "server connectivity", Status: "fail", Message: "auth.server_url is empty"}
	}

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, serverURL, nil)
	if err != nil {
		return CheckResult{Name: "server connectivity", Status: "fail", Message: err.Error()}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return CheckResult{Name: "server connectivity", Status: "fail", Message: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return CheckResult{Name: "server connectivity", Status: "fail", Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}
	return CheckResult{Name: "server connectivity", Status: "ok", Message: fmt.Sprintf("HTTP %d", resp.StatusCode)}
}
