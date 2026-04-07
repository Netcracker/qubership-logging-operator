package utils

import (
	"testing"
	"time"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{"1.5 seconds", 1500 * time.Millisecond, "1.5s"},
		{"zero", 0, "0s"},
		{"2m30.5s", 2*time.Minute + 30*time.Second + 500*time.Millisecond, "2m30.5s"},
		{"sub-millisecond truncated", 1*time.Second + 500*time.Microsecond, "1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			if result != tt.expected {
				t.Errorf("ToString(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetTagFromImage(t *testing.T) {
	tests := []struct {
		name     string
		image    string
		expected string
	}{
		{"standard image", "registry.example.com/image:v1.2.3", "v1.2.3"},
		{"image with port in registry", "registry:5000/image:latest", "latest"},
		{"no tag (no colon)", "myimage", "myimage"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTagFromImage(tt.image)
			if result != tt.expected {
				t.Errorf("GetTagFromImage(%q) = %q, want %q", tt.image, result, tt.expected)
			}
		})
	}
}

func TestGetAggregatorIds(t *testing.T) {
	tests := []struct {
		name     string
		num      int
		expected []int
	}{
		{"zero", 0, []int{}},
		{"one", 1, []int{0}},
		{"three", 3, []int{0, 1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetAggregatorIds(tt.num)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected len %d, got %d", len(tt.expected), len(result))
			}
			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("index %d: expected %d, got %d", i, v, result[i])
				}
			}
		})
	}
}

func TestGetFromResourceMap(t *testing.T) {
	t.Run("valid key", func(t *testing.T) {
		rl := core.ResourceList{
			core.ResourceName("cpu"): resource.MustParse("500m"),
		}
		result := GetFromResourceMap(rl, "cpu")
		if result != "500m" {
			t.Errorf("expected 500m, got %q", result)
		}
	})

	t.Run("missing key returns zero", func(t *testing.T) {
		rl := core.ResourceList{}
		result := GetFromResourceMap(rl, "memory")
		if result != "0" {
			t.Errorf("expected 0, got %q", result)
		}
	})
}

func TestToJSON(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		input := struct {
			Name string `json:"name"`
		}{Name: "test"}
		result := ToJSON(input)
		if result != `{"name":"test"}` {
			t.Errorf("unexpected JSON: %q", result)
		}
	})

	t.Run("nil returns null", func(t *testing.T) {
		result := ToJSON(nil)
		if result != "null" {
			t.Errorf("expected null, got %q", result)
		}
	})
}
