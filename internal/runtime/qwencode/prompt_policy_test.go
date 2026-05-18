package qwencode

import (
	"testing"
)

func TestBuildQwenArgsUsesStreamJSONActivityOutput(t *testing.T) {
	t.Parallel()

	args := buildQwenArgsWithIncludeDirectories([]string{"/tmp/repo"}, "prompt")
	for idx, arg := range args {
		if arg == "json" && idx > 0 && args[idx-1] == "--output-format" {
			t.Fatalf("qwen args must not request output-format json: %v", args)
		}
	}
	if !containsArg(args, "--chat-recording") || !containsArg(args, "--yolo") || !containsArg(args, "-p") {
		t.Fatalf("expected qwen noninteractive args to be preserved, got %v", args)
	}
	if !containsArg(args, "--output-format") || !containsArg(args, "stream-json") || !containsArg(args, "--include-partial-messages") {
		t.Fatalf("expected qwen stream-json activity args, got %v", args)
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
