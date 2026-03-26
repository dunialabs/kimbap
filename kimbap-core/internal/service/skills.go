package service

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9\-_]+$`)

type ServiceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	UpdatedAt   string `json:"updatedAt"`
}

type ServicesService struct {
	servicesDir string
	log         zerolog.Logger
}

var ErrNoServicesDirectoryFound = errors.New("No services directory found")

func NewServicesService() *ServicesService {
	return &ServicesService{servicesDir: filepath.Clean(config.SKILLS_CONFIG.SkillsDir), log: logger.CreateLogger("ServicesService")}
}

func (s *ServicesService) ListServices(serverID string) ([]ServiceInfo, error) {
	if err := validateName(serverID); err != nil {
		return nil, err
	}
	serverDir := s.serverServicesDirPath(serverID)
	if _, err := os.Stat(serverDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoServicesDirectoryFound
		}
		return nil, err
	}
	entries, err := os.ReadDir(serverDir)
	if err != nil {
		return nil, err
	}
	out := make([]ServiceInfo, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		serviceDir := filepath.Join(serverDir, entry.Name())
		metaPath := filepath.Join(serviceDir, config.SKILLS_CONFIG.SkillMetadataFile)
		content, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		meta := parseServiceMetadata(content)
		stat, err := os.Stat(serviceDir)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(meta["name"])
		if name == "" {
			name = entry.Name()
		}
		version := strings.TrimSpace(meta["version"])
		if version == "" {
			version = "1.0.0"
		}
		out = append(out, ServiceInfo{
			Name:        name,
			Description: meta["description"],
			Version:     version,
			UpdatedAt:   stat.ModTime().UTC().Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z"),
		})
	}
	return out, nil
}

func (s *ServicesService) UploadService(serverID string, zipBuffer []byte) ([]string, error) {
	if len(zipBuffer) > config.SKILLS_CONFIG.MaxZipSize {
		return nil, fmt.Errorf("zip file exceeds maximum size of %d bytes", config.SKILLS_CONFIG.MaxZipSize)
	}
	serverDir, err := s.ensureServerServicesDir(serverID)
	if err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(bytes.NewReader(zipBuffer), int64(len(zipBuffer)))
	if err != nil {
		return nil, err
	}

	tempDir, err := os.MkdirTemp("", "services-upload-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tempDir)

	if err := s.safeExtractZip(reader, tempDir); err != nil {
		return nil, err
	}
	serviceDirs, err := s.findServiceDirectories(tempDir)
	if err != nil {
		return nil, err
	}
	if len(serviceDirs) == 0 {
		return nil, fmt.Errorf("no valid services found: %s not found", config.SKILLS_CONFIG.SkillMetadataFile)
	}

	uploaded := make([]string, 0, len(serviceDirs))
	for _, dir := range serviceDirs {
		serviceName := filepath.Base(dir)
		if err := validateName(serviceName); err != nil {
			continue
		}
		targetDir := filepath.Join(serverDir, serviceName)
		if !s.isPathSafe(targetDir) {
			continue
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return nil, fmt.Errorf("failed to remove existing service dir %q: %w", targetDir, err)
		}
		if err := copyDirectory(dir, targetDir); err != nil {
			return nil, err
		}
		uploaded = append(uploaded, serviceName)
	}
	if len(uploaded) == 0 {
		return nil, errors.New("no valid services could be uploaded")
	}
	return uploaded, nil
}

func (s *ServicesService) DeleteService(serverID string, serviceName string) error {
	if err := validateName(serverID); err != nil {
		return err
	}
	if err := validateName(serviceName); err != nil {
		return err
	}
	path := filepath.Join(s.servicesDir, serverID, serviceName)
	if !s.isPathSafe(path) {
		return errors.New("invalid service path")
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("service not found: %s", serviceName)
		}
		return err
	}
	return os.RemoveAll(path)
}

func (s *ServicesService) DeleteServerServices(serverID string) error {
	if err := validateName(serverID); err != nil {
		return err
	}
	path := filepath.Join(s.servicesDir, serverID)
	if !s.isPathSafe(path) {
		return errors.New("invalid server service path")
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return os.RemoveAll(path)
}

func (s *ServicesService) ensureServerServicesDir(serverID string) (string, error) {
	if err := validateName(serverID); err != nil {
		return "", err
	}
	dir := s.serverServicesDirPath(serverID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *ServicesService) serverServicesDirPath(serverID string) string {
	return filepath.Join(s.servicesDir, serverID)
}

func validateName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	if !validNameRe.MatchString(name) {
		return errors.New("name contains invalid characters")
	}
	return nil
}

func (s *ServicesService) isPathSafe(targetPath string) bool {
	resolved := filepath.Clean(targetPath)
	base := filepath.Clean(s.servicesDir)
	return strings.HasPrefix(resolved+string(filepath.Separator), base+string(filepath.Separator))
}

func (s *ServicesService) safeExtractZip(reader *zip.Reader, targetDir string) error {
	if len(reader.File) > config.SKILLS_CONFIG.MaxEntryCount {
		return fmt.Errorf("zip has too many entries: %d", len(reader.File))
	}

	targetBase := filepath.Clean(targetDir)
	totalSize := uint64(0)
	type extractionEntry struct {
		file       *zip.File
		targetPath string
	}
	validEntries := make([]extractionEntry, 0, len(reader.File))

	for _, file := range reader.File {
		totalSize += file.UncompressedSize64
		if totalSize > uint64(config.SKILLS_CONFIG.MaxUncompressedSize) {
			return errors.New("zip uncompressed size exceeds limit")
		}

		if file.FileInfo().IsDir() {
			continue
		}

		name := file.Name
		if filepath.IsAbs(name) {
			return fmt.Errorf("invalid zip entry path: %s", name)
		}

		normalized := filepath.Clean(name)
		segments := strings.Split(normalized, string(filepath.Separator))
		for _, seg := range segments {
			if seg == ".." {
				return fmt.Errorf("path traversal detected: %s", name)
			}
		}

		targetPath := filepath.Join(targetDir, normalized)
		resolvedTarget := filepath.Clean(targetPath)
		if !strings.HasPrefix(resolvedTarget+string(filepath.Separator), targetBase+string(filepath.Separator)) {
			return fmt.Errorf("entry escapes target directory: %s", name)
		}

		validEntries = append(validEntries, extractionEntry{file: file, targetPath: resolvedTarget})
	}

	extractedSize := uint64(0)
	for _, entry := range validEntries {
		if err := os.MkdirAll(filepath.Dir(entry.targetPath), 0o755); err != nil {
			return err
		}

		rc, err := entry.file.Open()
		if err != nil {
			return err
		}

		f, err := os.OpenFile(entry.targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = rc.Close()
			return err
		}
		remaining := uint64(config.SKILLS_CONFIG.MaxUncompressedSize) - extractedSize
		written, err := io.Copy(f, io.LimitReader(rc, int64(remaining)+1))
		if err == nil && uint64(written) > remaining {
			err = errors.New("zip actual uncompressed size exceeds limit")
		}
		if cerr := rc.Close(); err == nil && cerr != nil {
			err = cerr
		}
		if cerr := f.Close(); err == nil && cerr != nil {
			err = cerr
		}
		if err != nil {
			return err
		}
		extractedSize += uint64(written)
	}
	return nil
}

func (s *ServicesService) findServiceDirectories(root string) ([]string, error) {
	results := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		metaPath := filepath.Join(path, config.SKILLS_CONFIG.SkillMetadataFile)
		if _, err := os.Stat(metaPath); err == nil {
			results = append(results, path)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

func parseServiceMetadata(content []byte) map[string]string {
	result := map[string]string{}
	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return result
	}

	frontmatter := make([]string, 0)
	foundClosing := false
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			foundClosing = true
			break
		}
		frontmatter = append(frontmatter, line)
	}
	if !foundClosing {
		return result
	}

	parsed := map[string]any{}
	if err := yaml.Unmarshal([]byte(strings.Join(frontmatter, "\n")), &parsed); err != nil {
		return result
	}
	for key, value := range parsed {
		if strings.TrimSpace(key) == "" {
			continue
		}
		result[key] = strings.TrimSpace(fmt.Sprint(value))
	}
	return result
}

func copyDirectory(src string, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		in, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			_ = in.Close()
			return err
		}
		_, err = io.Copy(out, in)
		if cerr := in.Close(); err == nil && cerr != nil {
			err = cerr
		}
		if cerr := out.Close(); err == nil && cerr != nil {
			err = cerr
		}
		if err != nil {
			return err
		}
	}
	return nil
}
