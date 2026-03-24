package config

var SKILLS_CONFIG = struct {
	SkillsDir           string
	HostSkillsDir       string
	MaxZipSize          int
	MaxUncompressedSize int
	MaxEntryCount       int
	SkillMetadataFile   string
}{
	SkillsDir:           Env("SKILLS_DIR", "./skills"),
	HostSkillsDir:       Env("HOST_SKILLS_DIR"),
	MaxZipSize:          10 * 1024 * 1024,
	MaxUncompressedSize: 50 * 1024 * 1024,
	MaxEntryCount:       1000,
	SkillMetadataFile:   "SKILL.md",
}
