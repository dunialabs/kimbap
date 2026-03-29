package providers

import "embed"

// EmbeddedProviders contains the embedded provider manifest YAML files
// bundled into the binary at build time.
//
//go:embed embedded/*.yaml
var EmbeddedProviders embed.FS
