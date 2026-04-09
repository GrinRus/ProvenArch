package runtime

import "testing"

func TestParseProvider(t *testing.T) {
	provider, err := ParseProvider("claude-code")
	if err != nil {
		t.Fatalf("parse claude-code provider: %v", err)
	}
	if provider != ProviderClaudeCode {
		t.Fatalf("unexpected provider %q", provider)
	}

	provider, err = ParseProvider("QWEN-CODE")
	if err != nil {
		t.Fatalf("parse qwen-code provider: %v", err)
	}
	if provider != ProviderQwenCode {
		t.Fatalf("unexpected provider %q", provider)
	}
}

func TestParseProviderRejectsUnsupportedValue(t *testing.T) {
	if _, err := ParseProvider("bogus"); err == nil {
		t.Fatalf("expected unsupported provider error")
	}
}

func TestResolveProviderCLIOverridesEnv(t *testing.T) {
	t.Setenv(RuntimeProviderEnv, "qwen-code")

	provider, err := ResolveProvider("claude-code")
	if err != nil {
		t.Fatalf("resolve provider with cli override: %v", err)
	}
	if provider != ProviderClaudeCode {
		t.Fatalf("expected claude-code provider, got %q", provider)
	}
}

func TestResolveProviderUsesEnvFallback(t *testing.T) {
	t.Setenv(RuntimeProviderEnv, "qwen-code")

	provider, err := ResolveProvider("")
	if err != nil {
		t.Fatalf("resolve provider from env: %v", err)
	}
	if provider != ProviderQwenCode {
		t.Fatalf("expected qwen-code provider, got %q", provider)
	}
}

func TestResolveProviderDefaultsToClaudeCode(t *testing.T) {
	provider, err := ResolveProvider("")
	if err != nil {
		t.Fatalf("resolve provider default: %v", err)
	}
	if provider != ProviderClaudeCode {
		t.Fatalf("expected claude-code default provider, got %q", provider)
	}
}

func TestNormalizeMode(t *testing.T) {
	mode, err := NormalizeMode("HEADLESS")
	if err != nil {
		t.Fatalf("normalize headless mode: %v", err)
	}
	if mode != RuntimeModeHeadless {
		t.Fatalf("unexpected mode %q", mode)
	}

	mode, err = NormalizeMode("")
	if err != nil {
		t.Fatalf("normalize empty mode: %v", err)
	}
	if mode != RuntimeModeFake {
		t.Fatalf("unexpected default mode %q", mode)
	}
}
