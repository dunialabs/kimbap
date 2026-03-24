package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/security"
	"github.com/dunialabs/kimbap-core/internal/types"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

type TunnelCredentials struct {
	AccountTag   string `json:"AccountTag"`
	TunnelSecret string `json:"TunnelSecret"`
	TunnelID     string `json:"TunnelID"`
	TunnelName   string `json:"TunnelName,omitempty"`
}

type CloudflaredService struct {
	db     *gorm.DB
	docker *CloudflaredDockerService
	log    zerolog.Logger
}

var cloudflaredSafeNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._:-]+$`)

const maxCloudAPIErrorBodyBytes = 64 * 1024

func NewCloudflaredService() *CloudflaredService {
	return &CloudflaredService{
		db:     database.DB,
		docker: NewCloudflaredDockerService(),
		log:    logger.CreateLogger("CloudflaredService"),
	}
}

func (s *CloudflaredService) GetContainerStatus() ContainerStatus {
	return s.docker.GetContainerStatus()
}

func (s *CloudflaredService) AutoStartIfConfigExists() {
	if !s.docker.CheckDockerAvailable() {
		return
	}
	var conf database.DnsConf
	if err := s.db.Where("type = ?", 1).Order("id asc").First(&conf).Error; err != nil {
		return
	}
	decryptedCreds := decryptCredentials(conf.Credentials)
	if decryptedCreds != "" {
		if err := validateCloudflaredInput(conf.TunnelID, conf.Subdomain); err != nil {
			s.log.Warn().Err(err).Msg("cloudflared autostart skipped due to invalid db config values")
			return
		}
		var creds TunnelCredentials
		if err := json.Unmarshal([]byte(decryptedCreds), &creds); err == nil {
			if creds.TunnelSecret == "" {
				s.log.Warn().Msg("cloudflared db credentials are missing tunnel secret")
			} else if err := s.generateLocalFiles(conf.TunnelID, creds, conf.Subdomain); err != nil {
				s.log.Error().Err(err).Msg("failed to start cloudflared from db credentials")
			} else if err := s.docker.EnsureImageExists(); err != nil {
				s.log.Error().Err(err).Msg("failed to start cloudflared from db credentials")
			} else if err := s.docker.StartContainer(); err != nil {
				s.log.Error().Err(err).Msg("failed to start cloudflared from db credentials")
			} else {
				return
			}
		} else {
			s.log.Error().Err(err).Msg("failed to parse cloudflared db credentials")
		}
	}
	if s.checkLocalCredentialsFile(conf.TunnelID) {
		if err := validateCloudflaredTunnelID(conf.TunnelID); err != nil {
			s.log.Warn().Err(err).Msg("cloudflared autostart skipped due to invalid tunnel id")
			return
		}
		credentialsPath := filepath.Join(config.CloudflaredCfg().ConfigDir, conf.TunnelID+".json")
		content, err := os.ReadFile(credentialsPath)
		if err == nil {
			var creds TunnelCredentials
			if err := json.Unmarshal(content, &creds); err == nil {
				if mkErr := os.MkdirAll(config.CloudflaredCfg().ConfigDir, 0o755); mkErr != nil {
					s.log.Error().Err(mkErr).Msg("failed to start cloudflared from local files")
					return
				}

				configPath := filepath.Join(config.CloudflaredCfg().ConfigDir, "config.yml")
				configYAML := s.docker.GenerateConfigYAML(conf.TunnelID, conf.Subdomain)
				if writeErr := os.WriteFile(configPath, []byte(configYAML), 0o644); writeErr != nil {
					s.log.Error().Err(writeErr).Msg("failed to start cloudflared from local files")
					return
				}

				if err := s.docker.EnsureImageExists(); err != nil {
					s.log.Error().Err(err).Msg("failed to start cloudflared from local files")
					return
				}
				if err := s.docker.StartContainer(); err != nil {
					s.log.Error().Err(err).Msg("failed to start cloudflared from local files")
					return
				}
				return
			}
			s.log.Error().Err(err).Msg("failed to start cloudflared from local files")
			return
		}
		s.log.Error().Err(err).Msg("failed to start cloudflared from local files")
		return
	}
	if err := s.db.Delete(&database.DnsConf{}, conf.ID).Error; err != nil {
		s.log.Warn().Err(err).Int("dnsConfId", conf.ID).Int("dnsConfType", conf.Type).Str("tunnelId", conf.TunnelID).Str("subdomain", conf.Subdomain).Bool("hasDbCredentials", conf.Credentials != "").Msg("cloudflared autostart cleanup: failed to delete invalid dns config (will retry)")
	}
}

func (s *CloudflaredService) StartCloudflared() error {
	if err := s.docker.EnsureImageExists(); err != nil {
		return err
	}
	return s.docker.StartContainer()
}

type RestartResult struct {
	Success         bool              `json:"success"`
	Message         string            `json:"message"`
	ContainerStatus ContainerStatus   `json:"containerStatus"`
	Config          *TunnelConfigInfo `json:"config,omitempty"`
}

type TunnelConfigInfo struct {
	TunnelID  string `json:"tunnelId"`
	Subdomain string `json:"subdomain"`
	PublicURL string `json:"publicUrl"`
}

type StopResult struct {
	Success         bool            `json:"success"`
	Message         string          `json:"message"`
	ContainerStatus ContainerStatus `json:"containerStatus"`
	AlreadyStopped  bool            `json:"alreadyStopped"`
}

func (s *CloudflaredService) StopCloudflared() (*StopResult, error) {
	initialStatus := s.docker.GetContainerStatus()
	if initialStatus != ContainerRunning {
		return &StopResult{
			Success:         true,
			Message:         "Cloudflared container is already stopped",
			ContainerStatus: initialStatus,
			AlreadyStopped:  true,
		}, nil
	}

	if err := s.docker.StopContainer(); err != nil {
		return nil, &types.AdminError{Message: fmt.Sprintf("Failed to stop cloudflared: %s", err), Code: types.AdminErrorCodeCloudflaredStopFailed}
	}

	finalStatus := s.docker.GetContainerStatus()
	if finalStatus == ContainerRunning {
		return &StopResult{
			Success:         true,
			Message:         "Cloudflared stop command completed (container status verification inconclusive)",
			ContainerStatus: finalStatus,
			AlreadyStopped:  false,
		}, nil
	}

	return &StopResult{
		Success:         true,
		Message:         "Cloudflared stopped successfully",
		ContainerStatus: finalStatus,
		AlreadyStopped:  false,
	}, nil
}

func (s *CloudflaredService) validateLocalSetup() (tunnelID string, err error) {
	var dnsConf database.DnsConf
	if err := s.db.Where("type = ?", 1).First(&dnsConf).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", &types.AdminError{Message: "No cloudflared configuration found in database", Code: types.AdminErrorCodeCloudflaredDBConfigNotFound}
		}
		return "", &types.AdminError{Message: fmt.Sprintf("Failed to query cloudflared configuration: %v", err), Code: types.AdminErrorCodeDatabaseOpFailed}
	}
	if dnsConf.TunnelID == "" {
		return "", &types.AdminError{Message: "No cloudflared configuration found in database", Code: types.AdminErrorCodeCloudflaredDBConfigNotFound}
	}

	if decCreds := decryptCredentials(dnsConf.Credentials); decCreds != "" {
		var creds TunnelCredentials
		if err := json.Unmarshal([]byte(decCreds), &creds); err == nil && creds.TunnelSecret != "" {
			return dnsConf.TunnelID, nil
		}
	}

	if s.checkLocalCredentialsFile(dnsConf.TunnelID) {
		return dnsConf.TunnelID, nil
	}

	credFile := filepath.Join(config.CloudflaredCfg().ConfigDir, dnsConf.TunnelID+".json")
	return dnsConf.TunnelID, &types.AdminError{Message: fmt.Sprintf("Credentials file not found: %s", credFile), Code: types.AdminErrorCodeCloudflaredLocalFileNotFound}
}

func (s *CloudflaredService) RestartCloudflared() (*RestartResult, error) {
	tunnelID, err := s.validateLocalSetup()
	if err != nil {
		return nil, err
	}

	var dnsConf database.DnsConf
	if err := s.db.Where("type = ?", 1).First(&dnsConf).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			s.log.Warn().Err(err).Msg("failed to load cloudflared config from database")
		}
	}

	if dnsConf.ID > 0 {
		if decCreds := decryptCredentials(dnsConf.Credentials); decCreds != "" {
			var creds TunnelCredentials
			if err := json.Unmarshal([]byte(decCreds), &creds); err == nil && creds.TunnelSecret != "" {
				if err := s.generateLocalFiles(tunnelID, creds, dnsConf.Subdomain); err != nil {
					s.log.Warn().Err(err).Msg("failed to regenerate files from database, using existing files")
				}
			}
		}
	}

	if err := s.docker.EnsureImageExists(); err != nil {
		return nil, &types.AdminError{Message: fmt.Sprintf("Failed to restart cloudflared: %s", err), Code: types.AdminErrorCodeCloudflaredRestartFailed}
	}
	if err := s.docker.RestartContainer(); err != nil {
		return nil, &types.AdminError{Message: fmt.Sprintf("Failed to restart cloudflared: %s", err), Code: types.AdminErrorCodeCloudflaredRestartFailed}
	}

	containerStatus := s.docker.GetContainerStatus()
	if containerStatus != ContainerRunning {
		return nil, &types.AdminError{Message: fmt.Sprintf("Cloudflared container is not running after restart (status: %s)", containerStatus), Code: types.AdminErrorCodeCloudflaredRestartFailed}
	}

	result := &RestartResult{
		Success:         true,
		Message:         "Cloudflared restarted successfully",
		ContainerStatus: containerStatus,
		Config: &TunnelConfigInfo{
			TunnelID: tunnelID,
		},
	}

	if dnsConf.Subdomain != "" {
		result.Config.Subdomain = dnsConf.Subdomain
		result.Config.PublicURL = s.PublicURL(dnsConf.Subdomain)
	}

	return result, nil
}

func (s *CloudflaredService) UpdateConfig(proxyID int, tunnelID string, subdomain string, credentials TunnelCredentials, publicIP string) (*database.DnsConf, bool, string, error) {
	if err := validateCloudflaredInput(tunnelID, subdomain); err != nil {
		return nil, false, "", err
	}
	now := int(time.Now().Unix())

	var existing database.DnsConf
	err := s.db.Where("proxy_id = ? AND type = ?", proxyID, 1).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, "", err
	}
	if existing.ID > 0 && publicIP == "" {
		publicIP = existing.PublicIP
	}

	credsJSON, _ := json.Marshal(credentials)
	encryptedCreds := encryptCredentials(credsJSON)
	if existing.ID > 0 {
		if existing.CreatedBy == 0 && existing.TunnelID != "" && existing.TunnelID != tunnelID {
			if err := s.deleteTunnel(existing.TunnelID); err != nil {
				s.log.Warn().Err(err).Str("oldTunnelID", existing.TunnelID).Msg("failed to delete old local tunnel")
			}
		}

		existing.TunnelID = tunnelID
		existing.Subdomain = subdomain
		existing.PublicIP = publicIP
		existing.UpdateTime = now
		existing.CreatedBy = 1
		existing.Credentials = encryptedCreds
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, false, "", err
		}
	} else {
		existing = database.DnsConf{
			TunnelID:    tunnelID,
			Subdomain:   subdomain,
			PublicIP:    publicIP,
			Type:        1,
			ProxyID:     proxyID,
			Addtime:     now,
			UpdateTime:  now,
			CreatedBy:   1,
			Credentials: encryptedCreds,
		}
		if err := s.db.Create(&existing).Error; err != nil {
			return nil, false, "", err
		}
	}

	if err := s.generateLocalFiles(tunnelID, credentials, subdomain); err != nil {
		return nil, false, "", err
	}

	var restartErr string
	restarted := true
	if err := s.docker.EnsureImageExists(); err != nil {
		restarted = false
		restartErr = err.Error()
	} else if err := s.docker.RestartContainer(); err != nil {
		restarted = false
		restartErr = err.Error()
	}
	return &existing, restarted, restartErr, nil
}

func (s *CloudflaredService) deleteTunnel(tunnelID string) error {
	baseURL := strings.TrimRight(config.CloudflaredCfg().CloudAPIBaseURL, "/")
	if baseURL == "" {
		err := errors.New("KIMBAP_CLOUD_API_URL is not set")
		s.log.Warn().Err(err).Str("tunnelID", tunnelID).Msg("failed to delete tunnel")
		return err
	}

	requestBody, err := json.Marshal(map[string]string{"tunnelId": tunnelID})
	if err != nil {
		s.log.Error().Err(err).Str("tunnelID", tunnelID).Msg("failed to marshal delete tunnel request")
		return fmt.Errorf("failed to delete tunnel: unable to build request")
	}

	url := baseURL + config.CloudflaredCfg().CloudAPIEndpoints.TunnelDelete
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		s.log.Error().Err(err).Str("tunnelID", tunnelID).Msg("failed to create delete-tunnel request")
		return fmt.Errorf("failed to delete tunnel: unable to build request")
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		s.log.Error().Err(err).Str("tunnelID", tunnelID).Msg("failed to delete tunnel")
		return fmt.Errorf("failed to delete tunnel: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(response.Body, maxCloudAPIErrorBodyBytes+1))
		if len(responseBody) > maxCloudAPIErrorBodyBytes {
			responseBody = responseBody[:maxCloudAPIErrorBodyBytes]
		}
		s.log.Error().Str("tunnelID", tunnelID).Int("statusCode", response.StatusCode).Msg("cloud API delete tunnel request failed")
		if len(responseBody) > 0 {
			return fmt.Errorf("failed to delete tunnel: cloud API returned %d: %s", response.StatusCode, string(responseBody))
		}
		return fmt.Errorf("failed to delete tunnel: cloud API returned %d", response.StatusCode)
	}

	s.log.Info().Str("tunnelID", tunnelID).Msg("deleted tunnel")
	return nil
}

func (s *CloudflaredService) GetConfigs(proxyID *int, tunnelID *string, subdomain *string, confType *int) ([]map[string]any, error) {
	query := s.db.Model(&database.DnsConf{})
	if proxyID != nil {
		query = query.Where("proxy_id = ?", *proxyID)
	}
	if tunnelID != nil {
		query = query.Where("tunnel_id = ?", *tunnelID)
	}
	if subdomain != nil {
		query = query.Where("subdomain = ?", *subdomain)
	}
	if confType != nil {
		query = query.Where("type = ?", *confType)
	}
	var confs []database.DnsConf
	if err := query.Find(&confs).Error; err != nil {
		return nil, err
	}
	status := s.docker.GetContainerStatus()
	out := make([]map[string]any, 0, len(confs))
	for _, conf := range confs {
		out = append(out, map[string]any{"dnsConf": conf, "status": status})
	}
	return out, nil
}

func (s *CloudflaredService) DeleteConfig(id *int, tunnelID *string) error {
	if id == nil && tunnelID == nil {
		return errors.New("either id or tunnelId must be provided")
	}
	query := s.db.Model(&database.DnsConf{})
	if id != nil {
		query = query.Where("id = ?", *id)
	}
	if tunnelID != nil {
		query = query.Where("tunnel_id = ?", *tunnelID)
	}
	var conf database.DnsConf
	if err := query.First(&conf).Error; err != nil {
		return err
	}

	if conf.TunnelID != "" {
		if err := validateCloudflaredTunnelID(conf.TunnelID); err != nil {
			return err
		}
		if err := s.deleteTunnel(conf.TunnelID); err != nil {
			return err
		}
	}
	if err := s.stopAndDeleteContainer(); err != nil {
		return err
	}
	if err := s.cleanupLocalFiles(conf.TunnelID); err != nil {
		return err
	}
	return s.db.Delete(&database.DnsConf{}, conf.ID).Error
}

func (s *CloudflaredService) generateLocalFiles(tunnelID string, credentials TunnelCredentials, subdomain string) error {
	if err := validateCloudflaredInput(tunnelID, subdomain); err != nil {
		return err
	}
	if err := os.MkdirAll(config.CloudflaredCfg().ConfigDir, 0o755); err != nil {
		return err
	}
	credentialsPath := filepath.Join(config.CloudflaredCfg().ConfigDir, tunnelID+".json")
	backupPath := filepath.Join(config.CloudflaredCfg().ConfigDir, "credentials.json")
	configPath := filepath.Join(config.CloudflaredCfg().ConfigDir, "config.yml")

	b, _ := json.MarshalIndent(credentials, "", "  ")
	if err := os.WriteFile(credentialsPath, b, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(backupPath, b, 0o600); err != nil {
		return err
	}
	yml := s.docker.GenerateConfigYAML(tunnelID, subdomain)
	if err := os.WriteFile(configPath, []byte(yml), 0o644); err != nil {
		return err
	}
	return nil
}

func (s *CloudflaredService) stopAndDeleteContainer() error {
	var cleanupErr error
	if err := s.docker.StopContainer(); err != nil {
		s.log.Warn().Err(err).Msg("failed to stop container during delete")
		cleanupErr = errors.Join(cleanupErr, err)
	}
	if err := s.docker.DeleteContainer(); err != nil {
		s.log.Warn().Err(err).Msg("failed to delete container during delete")
		cleanupErr = errors.Join(cleanupErr, err)
	}
	return cleanupErr
}

func (s *CloudflaredService) cleanupLocalFiles(tunnelID string) error {
	if err := validateCloudflaredTunnelID(tunnelID); err != nil {
		return err
	}
	var cleanupErr error
	files := []string{
		filepath.Join(config.CloudflaredCfg().ConfigDir, tunnelID+".json"),
		filepath.Join(config.CloudflaredCfg().ConfigDir, "credentials.json"),
		filepath.Join(config.CloudflaredCfg().ConfigDir, "config.yml"),
	}
	for _, file := range files {
		if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
			s.log.Warn().Err(err).Str("file", file).Msg("failed deleting cloudflared file")
			cleanupErr = errors.Join(cleanupErr, err)
		}
	}
	return cleanupErr
}

func (s *CloudflaredService) checkLocalCredentialsFile(tunnelID string) bool {
	if err := validateCloudflaredTunnelID(tunnelID); err != nil {
		return false
	}
	file := filepath.Join(config.CloudflaredCfg().ConfigDir, tunnelID+".json")
	b, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	var creds TunnelCredentials
	if err := json.Unmarshal(b, &creds); err != nil {
		return false
	}
	return creds.TunnelSecret != ""
}

func (s *CloudflaredService) PublicURL(subdomain string) string {
	if err := validateCloudflaredSubdomain(subdomain); err != nil {
		return ""
	}
	v := strings.TrimSpace(subdomain)
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	return fmt.Sprintf("https://%s", v)
}

func validateCloudflaredInput(tunnelID string, subdomain string) error {
	if err := validateCloudflaredTunnelID(tunnelID); err != nil {
		return err
	}
	return validateCloudflaredSubdomain(subdomain)
}

func validateCloudflaredTunnelID(tunnelID string) error {
	v := strings.TrimSpace(tunnelID)
	if v == "" {
		return errors.New("invalid tunnel id")
	}
	if strings.Contains(v, "..") || strings.ContainsAny(v, "/\\") {
		return errors.New("invalid tunnel id")
	}
	if strings.ContainsAny(v, "\r\n\t") || hasControlChars(v) {
		return errors.New("invalid tunnel id")
	}
	if !cloudflaredSafeNamePattern.MatchString(v) {
		return errors.New("invalid tunnel id")
	}
	return nil
}

func validateCloudflaredSubdomain(subdomain string) error {
	v := strings.TrimSpace(subdomain)
	if v == "" {
		return errors.New("invalid subdomain")
	}
	if strings.Contains(v, "..") || strings.ContainsAny(v, "/\\") {
		return errors.New("invalid subdomain")
	}
	if strings.ContainsAny(v, "\r\n\t") || hasControlChars(v) {
		return errors.New("invalid subdomain")
	}
	if !cloudflaredSafeNamePattern.MatchString(v) {
		return errors.New("invalid subdomain")
	}
	return nil
}

func hasControlChars(v string) bool {
	for _, r := range v {
		if r < 32 || r == 127 {
			return true
		}
	}
	return false
}

func tunnelCredEncryptionKey() string {
	key := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if key == "" {
		key = "kimbap-tunnel-default-key"
	}
	return key
}

func encryptCredentials(plainJSON []byte) string {
	encrypted, err := security.EncryptData(string(plainJSON), tunnelCredEncryptionKey())
	if err != nil {
		return string(plainJSON)
	}
	return encrypted
}

func decryptCredentials(stored string) string {
	if stored == "" {
		return ""
	}
	decrypted, err := security.DecryptDataFromString(stored, tunnelCredEncryptionKey())
	if err != nil {
		return stored
	}
	return decrypted
}
