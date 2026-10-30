package utils

import "testing"

func TestFluentdQuote(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"plain value", "s3cret", `'s3cret'`},
		{"embedded ruby expression", `p#{exec("id")}a`, `'p#{exec("id")}a'`},
		{"single quote", "it's", `'it\'s'`},
		{"backslash", `a\b`, `'a\\b'`},
		{"empty value", "", `''`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := FluentdQuote(test.value); got != test.expected {
				t.Errorf("FluentdQuote(%q) = %q, want %q", test.value, got, test.expected)
			}
		})
	}
}
