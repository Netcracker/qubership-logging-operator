package utils

import (
	"net/http"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	t.Run("simple substitution", func(t *testing.T) {
		result, err := ParseTemplate("Hello {{ .Name }}", "test.tpl", struct{ Name string }{"World"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", result)
		}
	})

	t.Run("sprig function", func(t *testing.T) {
		result, err := ParseTemplate(`{{ .Value | upper }}`, "test.tpl", struct{ Value string }{"hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "HELLO" {
			t.Errorf("expected HELLO, got %q", result)
		}
	})

	t.Run("invalid template syntax", func(t *testing.T) {
		_, err := ParseTemplate("{{ .Invalid", "bad.tpl", nil)
		if err == nil {
			t.Error("expected error for invalid template")
		}
	})

	t.Run("missing field returns error", func(t *testing.T) {
		_, err := ParseTemplate("{{ .Missing }}", "test.tpl", struct{}{})
		if err == nil {
			t.Error("expected error for missing field")
		}
	})
}

func TestParseTemplate_isValidShards(t *testing.T) {
	tpl := `{{ if isValidShards .V }}valid{{ else }}invalid{{ end }}`

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"int 1 is valid", 1, "valid"},
		{"int 0 is invalid", 0, "invalid"},
		{"int -1 is invalid", -1, "invalid"},
		{"string 2 is valid", "2", "valid"},
		{"string 0 is invalid", "0", "invalid"},
		{"string abc is invalid", "abc", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTemplate(tpl, "test.tpl", struct{ V interface{} }{tt.value})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("isValidShards(%v) rendered %q, want %q", tt.value, result, tt.expected)
			}
		})
	}

	t.Run("nil pointer is invalid", func(t *testing.T) {
		result, err := ParseTemplate(tpl, "test.tpl", struct{ V interface{} }{(*int)(nil)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "invalid" {
			t.Errorf("expected invalid for nil pointer, got %q", result)
		}
	})
}

func TestParseTemplate_isValidReplicas(t *testing.T) {
	tpl := `{{ if isValidReplicas .V }}valid{{ else }}invalid{{ end }}`

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"int 0 is valid (replicas can be 0)", 0, "valid"},
		{"int 1 is valid", 1, "valid"},
		{"int -1 is invalid", -1, "invalid"},
		{"string 0 is valid", "0", "valid"},
		{"string abc is invalid", "abc", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseTemplate(tpl, "test.tpl", struct{ V interface{} }{tt.value})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("isValidReplicas(%v) rendered %q, want %q", tt.value, result, tt.expected)
			}
		})
	}

	t.Run("nil pointer is invalid", func(t *testing.T) {
		result, err := ParseTemplate(tpl, "test.tpl", struct{ V interface{} }{(*int)(nil)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "invalid" {
			t.Errorf("expected invalid for nil pointer, got %q", result)
		}
	})
}

func TestSetAuthHeader(t *testing.T) {
	t.Run("basic auth", func(t *testing.T) {
		rc := &RestClient{Auth: &Creds{Name: "user", Password: "pass"}}
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		rc.SetAuthHeader(req)
		auth := req.Header.Get("Authorization")
		// base64("user:pass") = "dXNlcjpwYXNz"
		expected := "Basic dXNlcjpwYXNz"
		if auth != expected {
			t.Errorf("expected %q, got %q", expected, auth)
		}
	})

	t.Run("bearer auth", func(t *testing.T) {
		rc := &RestClient{Auth: &Creds{Token: "mytoken"}}
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		rc.SetAuthHeader(req)
		auth := req.Header.Get("Authorization")
		if auth != "Bearer mytoken" {
			t.Errorf("expected 'Bearer mytoken', got %q", auth)
		}
	})

	t.Run("nil creds no header", func(t *testing.T) {
		rc := &RestClient{Auth: nil}
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		rc.SetAuthHeader(req)
		auth := req.Header.Get("Authorization")
		if auth != "" {
			t.Errorf("expected no Authorization header, got %q", auth)
		}
	})

	t.Run("name and password take precedence over token", func(t *testing.T) {
		rc := &RestClient{Auth: &Creds{Name: "user", Password: "pass", Token: "tok"}}
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		rc.SetAuthHeader(req)
		auth := req.Header.Get("Authorization")
		if auth != "Basic dXNlcjpwYXNz" {
			t.Errorf("basic auth should take precedence, got %q", auth)
		}
	})
}
