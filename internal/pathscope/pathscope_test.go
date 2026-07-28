package pathscope

import "testing"

func TestRecursiveGlobConformance(t *testing.T) {
	tests := []struct {
		pattern, candidate string
		want               bool
	}{
		{"**", "README.md", true},
		{"**/*.go", "main.go", true},
		{"**/*.go", "internal/api/server.go", true},
		{"src/**", "src", true},
		{"src/**", "src/a/b.go", true},
		{"src/*", "src/a.go", true},
		{"src/*", "src/a/b.go", false},
		{"src", "src/a/b.go", true},
		{"*.md", "README.md", true},
		{"*.md", "docs/README.md", false},
		{".", ".", true},
		{"src\\**", "src\\nested\\main.go", true},
	}
	for _, test := range tests {
		compiled, err := Compile(test.pattern)
		if err != nil {
			t.Fatal(err)
		}
		if got := compiled.Match(test.candidate); got != test.want {
			t.Fatalf("Match(%q, %q) = %v, want %v", test.pattern, test.candidate, got, test.want)
		}
	}
}

func TestCompileRejectsInvalidPatterns(t *testing.T) {
	for _, value := range []string{"", "/src/**", `C:\\src\\**`, "../src/**", "src//**", "src/**.go", "src/["} {
		if _, err := Compile(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
