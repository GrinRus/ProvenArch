package providers

import (
	"fmt"

	acpruntime "github.com/GrinRus/ProvenArch/internal/runtime"
	"github.com/GrinRus/ProvenArch/internal/runtime/claudecode"
	"github.com/GrinRus/ProvenArch/internal/runtime/codexcode"
	"github.com/GrinRus/ProvenArch/internal/runtime/fakeruntime"
	"github.com/GrinRus/ProvenArch/internal/runtime/qwencode"
)

func BuildRunner(runtimeMode string, provider acpruntime.Provider) (acpruntime.Runner, error) {
	mode, err := acpruntime.NormalizeMode(runtimeMode)
	if err != nil {
		return nil, err
	}
	if provider == "" {
		provider = acpruntime.ProviderClaudeCode
	}
	providerName := string(provider)

	switch mode {
	case acpruntime.RuntimeModeFake:
		return fakeruntime.Runner{}, nil
	case acpruntime.RuntimeModeHeadless:
		switch provider {
		case acpruntime.ProviderClaudeCode:
			return claudecode.HeadlessRunner{}, nil
		case acpruntime.ProviderQwenCode:
			return qwencode.HeadlessRunner{}, nil
		case acpruntime.ProviderCodexCode:
			return codexcode.HeadlessRunner{}, nil
		default:
			return nil, fmt.Errorf(
				"unsupported runtime provider %q (allowed: %s, %s, %s)",
				providerName,
				acpruntime.ProviderClaudeCode,
				acpruntime.ProviderQwenCode,
				acpruntime.ProviderCodexCode,
			)
		}
	default:
		return nil, fmt.Errorf("unsupported runtime %q (allowed: %s, %s)", runtimeMode, acpruntime.RuntimeModeFake, acpruntime.RuntimeModeHeadless)
	}
}
