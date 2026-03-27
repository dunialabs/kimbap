package providers

import "embed"

// EmbeddedProviders contains the official provider manifest YAML files
// bundled into the binary at build time.
//
//go:embed official/*.yaml
var EmbeddedProviders embed.FS
