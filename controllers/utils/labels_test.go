package utils

import (
	"strings"
	"testing"
)

func TestCommonLabels(t *testing.T) {
	labels := CommonLabels()

	expected := map[string]string{
		"app.kubernetes.io/part-of":             PartOfLogging,
		"app.kubernetes.io/managed-by":          ManagedByOperator,
		"app.kubernetes.io/managed-by-operator": OperatorDeploymentName,
	}

	if len(labels) != len(expected) {
		t.Fatalf("expected %d labels, got %d", len(expected), len(labels))
	}
	for k, v := range expected {
		if labels[k] != v {
			t.Errorf("key %q: expected %q, got %q", k, v, labels[k])
		}
	}

	// Mutation safety: modifying returned map should not affect next call
	labels["extra"] = "value"
	labels2 := CommonLabels()
	if _, ok := labels2["extra"]; ok {
		t.Error("CommonLabels returned shared map; mutation leaked between calls")
	}
}

func TestResourceLabels(t *testing.T) {
	labels := ResourceLabels("my-service", "backend")

	if labels["name"] != "my-service" {
		t.Errorf("expected name=my-service, got %q", labels["name"])
	}
	if labels["app.kubernetes.io/name"] != "my-service" {
		t.Errorf("expected app.kubernetes.io/name=my-service, got %q", labels["app.kubernetes.io/name"])
	}
	if labels["app.kubernetes.io/component"] != "backend" {
		t.Errorf("expected component=backend, got %q", labels["app.kubernetes.io/component"])
	}
	// Should include CommonLabels
	if labels["app.kubernetes.io/part-of"] != PartOfLogging {
		t.Errorf("missing CommonLabels key part-of")
	}
}

func TestMergeLabels(t *testing.T) {
	tests := []struct {
		name     string
		maps     []map[string]string
		expected map[string]string
	}{
		{
			name:     "disjoint maps",
			maps:     []map[string]string{{"a": "1"}, {"b": "2"}},
			expected: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:     "overlapping keys, later wins",
			maps:     []map[string]string{{"a": "1"}, {"a": "2"}},
			expected: map[string]string{"a": "2"},
		},
		{
			name:     "nil maps skipped",
			maps:     []map[string]string{{"a": "1"}, nil, {"b": "2"}},
			expected: map[string]string{"a": "1", "b": "2"},
		},
		{
			name:     "all nil maps",
			maps:     []map[string]string{nil, nil},
			expected: map[string]string{},
		},
		{
			name:     "no maps",
			maps:     nil,
			expected: map[string]string{},
		},
		{
			name:     "single map returns copy",
			maps:     []map[string]string{{"a": "1"}},
			expected: map[string]string{"a": "1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeLabels(tt.maps...)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d keys, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("key %q: expected %q, got %q", k, v, result[k])
				}
			}
		})
	}

	// Verify single map returns a new map (not same pointer)
	original := map[string]string{"a": "1"}
	result := MergeLabels(original)
	result["b"] = "2"
	if _, ok := original["b"]; ok {
		t.Error("MergeLabels returned same map reference, not a copy")
	}
}

func TestMergeInto(t *testing.T) {
	t.Run("adds keys", func(t *testing.T) {
		dst := map[string]string{"a": "1"}
		MergeInto(dst, map[string]string{"b": "2"})
		if dst["b"] != "2" {
			t.Errorf("expected b=2, got %q", dst["b"])
		}
	})

	t.Run("overwrites on conflict", func(t *testing.T) {
		dst := map[string]string{"a": "1"}
		MergeInto(dst, map[string]string{"a": "2"})
		if dst["a"] != "2" {
			t.Errorf("expected a=2, got %q", dst["a"])
		}
	})

	t.Run("nil src is no-op", func(t *testing.T) {
		dst := map[string]string{"a": "1"}
		MergeInto(dst, nil)
		if len(dst) != 1 || dst["a"] != "1" {
			t.Errorf("expected dst unchanged, got %v", dst)
		}
	})
}

func TestTruncLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short string unchanged", "hello", "hello"},
		{"exactly 63 chars", strings.Repeat("a", 63), strings.Repeat("a", 63)},
		{"64 chars truncated to 63", strings.Repeat("a", 64), strings.Repeat("a", 63)},
		{"100 chars truncated to 63", strings.Repeat("a", 100), strings.Repeat("a", 63)},
		{"trailing dash trimmed after truncation", strings.Repeat("a", 62) + "-x", strings.Repeat("a", 62)},
		{"leading dash trimmed", "-hello", "hello"},
		{"empty string", "", ""},
		{"dashes only short", "---", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncLabel(tt.input)
			if result != tt.expected {
				t.Errorf("TruncLabel(%q) = %q, want %q", tt.input, result, tt.expected)
			}
			if len(result) > 63 {
				t.Errorf("TruncLabel result exceeds 63 chars: len=%d", len(result))
			}
		})
	}
}

func TestGetInstanceLabel(t *testing.T) {
	t.Run("short combo", func(t *testing.T) {
		result := GetInstanceLabel("svc", "ns")
		if result != "svc-ns" {
			t.Errorf("expected svc-ns, got %q", result)
		}
	})

	t.Run("long combo triggers truncation", func(t *testing.T) {
		name := strings.Repeat("a", 40)
		ns := strings.Repeat("b", 40)
		result := GetInstanceLabel(name, ns)
		if len(result) > 63 {
			t.Errorf("result exceeds 63 chars: len=%d", len(result))
		}
	})
}

func TestLabelInput_instanceVersionTechnologyMap(t *testing.T) {
	t.Run("all fields set", func(t *testing.T) {
		in := LabelInput{Instance: "inst", Version: "v1", Technology: "go"}
		m := in.instanceVersionTechnologyMap()
		if m["app.kubernetes.io/instance"] != "inst" {
			t.Error("missing instance")
		}
		if m["app.kubernetes.io/version"] != "v1" {
			t.Error("missing version")
		}
		if m["app.kubernetes.io/technology"] != "go" {
			t.Error("missing technology")
		}
	})

	t.Run("empty fields omitted", func(t *testing.T) {
		in := LabelInput{}
		m := in.instanceVersionTechnologyMap()
		if len(m) != 0 {
			t.Errorf("expected empty map, got %v", m)
		}
	})
}

func TestLabelInput_resourceLabels(t *testing.T) {
	t.Run("ComponentLabels override base", func(t *testing.T) {
		in := LabelInput{
			Name:            "svc",
			Component:       "comp",
			ComponentLabels: map[string]string{"name": "override"},
		}
		labels := in.resourceLabels(nil)
		if labels["name"] != "override" {
			t.Errorf("ComponentLabels should override base, got name=%q", labels["name"])
		}
	})

	t.Run("existing labels preserved when not overridden", func(t *testing.T) {
		existing := map[string]string{"custom": "value"}
		in := LabelInput{Name: "svc", Component: "comp"}
		labels := in.resourceLabels(existing)
		if labels["custom"] != "value" {
			t.Error("existing labels lost")
		}
	})
}

func TestLabelInput_templateLabels(t *testing.T) {
	in := LabelInput{
		Name:            "svc",
		Component:       "comp",
		Instance:        "inst",
		ComponentLabels: map[string]string{"extra": "value"},
	}
	labels := in.templateLabels(nil)

	// Should have base labels + instance
	if labels["app.kubernetes.io/instance"] != "inst" {
		t.Error("missing instance in template labels")
	}

	// Should NOT have ComponentLabels
	if _, ok := labels["extra"]; ok {
		t.Error("ComponentLabels should not appear in template labels")
	}
}

func TestPodTemplateLabels(t *testing.T) {
	labels := PodTemplateLabels("svc", "comp", "inst", "v1", "go")

	if labels["app.kubernetes.io/instance"] != "inst" {
		t.Error("missing instance")
	}
	if labels["app.kubernetes.io/version"] != "v1" {
		t.Error("missing version")
	}
	if labels["app.kubernetes.io/technology"] != "go" {
		t.Error("missing technology")
	}
	if labels["app.kubernetes.io/name"] != "svc" {
		t.Error("missing name")
	}

	// Empty optional params should be absent
	labels2 := PodTemplateLabels("svc", "comp", "", "", "")
	if _, ok := labels2["app.kubernetes.io/version"]; ok {
		t.Error("empty version should not produce a label")
	}
}
