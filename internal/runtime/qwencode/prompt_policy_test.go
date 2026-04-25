package qwencode

import (
	"testing"
)

func TestBuildQwenArgsDoesNotRequireSemanticJSONOutput(t *testing.T) {
	t.Parallel()

	args := buildQwenArgsWithIncludeDirectories([]string{"/tmp/repo"}, "prompt")
	if containsArg(args, "--output-format") {
		t.Fatalf("qwen args must not force semantic JSON output: %v", args)
	}
	for idx, arg := range args {
		if arg == "json" && idx > 0 && args[idx-1] == "--output-format" {
			t.Fatalf("qwen args must not request output-format json: %v", args)
		}
	}
	if !containsArg(args, "--chat-recording") || !containsArg(args, "--yolo") || !containsArg(args, "-p") {
		t.Fatalf("expected qwen noninteractive args to be preserved, got %v", args)
	}
}

func TestQwenAdapterUsesSharedUnavailableMarkers(t *testing.T) {
	t.Parallel()

	markers := (qwenAdapter{}).UnavailableMarkers()
	if !containsArg(markers, "rate limit") || !containsArg(markers, "ssl") {
		t.Fatalf("expected shared unavailable markers, got %v", markers)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
