package secretredact

import (
	"strings"
	"testing"
)

func TestRedactTextMasksCommonSecretForms(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		"Authorization: Bearer live-token-123",
		`{"token":"json-token","api_key":"json-key"}`,
		"password=plain-password",
		`password="two words secret"`,
		"secret='single quoted secret'",
		"--api-key cli-key --model qwen",
		"--token cli-token",
		"ordinary diagnostic text",
	}, "\n")

	got := RedactText(input)
	for _, secret := range []string{"live-token-123", "json-token", "json-key", "plain-password", "two words secret", "single quoted secret", "cli-key", "cli-token"} {
		if strings.Contains(got, secret) {
			t.Fatalf("secret %q leaked in redacted output:\n%s", secret, got)
		}
	}
	if !strings.Contains(got, "--model qwen") {
		t.Fatalf("non-secret diagnostic text was over-redacted:\n%s", got)
	}
	if !strings.Contains(got, "ordinary diagnostic text") {
		t.Fatalf("ordinary diagnostic text missing:\n%s", got)
	}
}

func TestRedactTextLeavesNonSecretTextStable(t *testing.T) {
	t.Parallel()

	input := "model=qwen status=ok"
	if got := RedactText(input); got != input {
		t.Fatalf("expected non-secret text unchanged, got %q", got)
	}
}
