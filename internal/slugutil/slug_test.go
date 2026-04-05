package slugutil

import "testing"

func TestSlugify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: "unknown"},
		{name: "spaces", input: "   ", want: "unknown"},
		{name: "alphanumeric", input: "Payments2", want: "payments2"},
		{name: "trims and lowers", input: "  Payments Service  ", want: "payments-service"},
		{name: "collapses separators", input: "svc__payments///api", want: "svc-payments-api"},
		{name: "leading trailing separators", input: "__payments__", want: "payments"},
		{name: "non latin only", input: "Привет", want: "unknown"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Slugify(tc.input)
			if got != tc.want {
				t.Fatalf("Slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
