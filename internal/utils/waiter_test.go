package utils

import (
	"testing"
)

func TestToComponentNameList(t *testing.T) {
	reconciler := &ComponentsPendingReconciler{}

	t.Run("empty list", func(t *testing.T) {
		list := &[]Component{}
		result := reconciler.ToComponentNameList(list)
		if len(result) != 0 {
			t.Errorf("expected empty list, got %v", result)
		}
	})

	t.Run("two components", func(t *testing.T) {
		list := &[]Component{
			{ComponentName: "logging-fluentd", StatusName: "FluentdStatus"},
			{ComponentName: "logging-fluentbit", StatusName: "FluentbitStatus"},
		}
		result := reconciler.ToComponentNameList(list)
		if len(result) != 2 {
			t.Fatalf("expected 2 names, got %d", len(result))
		}
		if result[0] != "logging-fluentd" {
			t.Errorf("expected logging-fluentd, got %q", result[0])
		}
		if result[1] != "logging-fluentbit" {
			t.Errorf("expected logging-fluentbit, got %q", result[1])
		}
	})
}
