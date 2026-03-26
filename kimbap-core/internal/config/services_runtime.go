package config

var SERVICE_RUNTIME_CONFIG = struct {
	ServicesDir            string
	HostServicesDir        string
	MaxZipSize             int
	MaxUncompressedSize    int
	MaxEntryCount          int
	AgentSkillMetadataFile string
}{
	ServicesDir:            Env("SERVICES_DIR", "./services"),
	HostServicesDir:        Env("HOST_SERVICES_DIR"),
	MaxZipSize:             10 * 1024 * 1024,
	MaxUncompressedSize:    50 * 1024 * 1024,
	MaxEntryCount:          1000,
	AgentSkillMetadataFile: "SKILL.md",
}
