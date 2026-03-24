package flows

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/dunialabs/kimbap-core/internal/connectors"
)

type FlowSelector struct{}

type EnvironmentInfo struct {
	IsSSH          bool
	HasTTY         bool
	HasDisplay     bool
	CanOpenBrowser bool
}

func (s *FlowSelector) SelectFlow(requested connectors.FlowType, provider connectors.ProviderDefinition) (connectors.FlowType, error) {
	if requested != "" && requested != connectors.FlowType("auto") {
		if !provider.SupportsFlow(requested) {
			return "", fmt.Errorf("requested flow %q is not supported by provider %q", requested, provider.ID)
		}
		return requested, nil
	}

	env := DetectEnvironment()
	if env.CanOpenBrowser && provider.SupportsBrowserFlow() {
		return connectors.FlowBrowser, nil
	}
	if provider.SupportsDeviceFlow() {
		return connectors.FlowDevice, nil
	}
	if provider.SupportsClientCredentials() {
		return connectors.FlowClientCredentials, nil
	}

	return "", errors.New("no viable oauth flow: provider supports none of browser/device/client_credentials for current environment")
}

func DetectEnvironment() EnvironmentInfo {
	isSSH := os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_CLIENT") != ""
	hasTTY := false
	if info, err := os.Stdin.Stat(); err == nil {
		hasTTY = (info.Mode() & os.ModeCharDevice) != 0
	}

	hasDisplay := true
	if runtime.GOOS == "linux" {
		hasDisplay = os.Getenv("DISPLAY") != ""
	}

	canOpenBrowser := !isSSH && hasTTY && hasDisplay
	return EnvironmentInfo{
		IsSSH:          isSSH,
		HasTTY:         hasTTY,
		HasDisplay:     hasDisplay,
		CanOpenBrowser: canOpenBrowser,
	}
}
