package config

import "sync"

type cloudflaredConfig struct {
	InDocker             bool
	ContainerName        string
	ConfigDir            string
	ContainerConfigDir   string
	Image                string
	KimbapCoreServiceURL string
	CloudAPIBaseURL      string
	CloudAPIEndpoints    struct {
		TunnelCreate string
		TunnelDelete string
	}
}

var (
	cloudflaredOnce   sync.Once
	cloudflaredCached cloudflaredConfig
)

func GetCloudflaredConfig() cloudflaredConfig {
	cloudflaredOnce.Do(func() {
		inDocker := Env("KIMBAP_CORE_IN_DOCKER") == "true"
		configDir := "cloudflared"
		if inDocker {
			configDir = "/app/cloudflared"
		}
		serviceURL := "http://host.docker.internal:" + Env("BACKEND_PORT", "3002")
		if inDocker {
			serviceURL = "http://kimbap-core:" + Env("BACKEND_PORT", "3002")
		}
		cloudflaredCached = cloudflaredConfig{
			InDocker:             inDocker,
			ContainerName:        Env("CLOUDFLARED_CONTAINER_NAME", "kimbap-core-cloudflared"),
			ConfigDir:            configDir,
			ContainerConfigDir:   Env("CLOUDFLARED_CONTAINER_DIR", "/etc/cloudflared"),
			Image:                "cloudflare/cloudflared:latest",
			KimbapCoreServiceURL: serviceURL,
			CloudAPIBaseURL:      Env("KIMBAP_CLOUD_API_URL", ""),
			CloudAPIEndpoints: struct {
				TunnelCreate string
				TunnelDelete string
			}{
				TunnelCreate: "/tunnel/create",
				TunnelDelete: "/tunnel/delete",
			},
		}
	})
	return cloudflaredCached
}

func CloudflaredCfg() cloudflaredConfig {
	return GetCloudflaredConfig()
}
