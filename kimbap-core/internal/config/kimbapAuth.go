package config

import "sync"

type kimbapAuthConfig struct {
	InDocker bool
	BaseURL  string
}

var (
	kimbapAuthOnce   sync.Once
	kimbapAuthCached kimbapAuthConfig
)

func GetKimbapAuthConfig() kimbapAuthConfig {
	kimbapAuthOnce.Do(func() {
		inDocker := Env("KIMBAP_CORE_IN_DOCKER") == "true"
		baseURL := "http://localhost:7788"
		if inDocker {
			baseURL = "http://kimbap-auth:7788"
		}
		kimbapAuthCached = kimbapAuthConfig{
			InDocker: inDocker,
			BaseURL:  baseURL,
		}
	})
	return kimbapAuthCached
}

var KIMBAP_AUTH_CONFIG = GetKimbapAuthConfig
