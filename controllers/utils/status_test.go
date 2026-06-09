package utils

import (
	"testing"

	loggingService "github.com/Netcracker/qubership-logging-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestStatusUpdater(conditions []loggingService.LoggingServiceCondition) (*StatusUpdater, *loggingService.LoggingService) {
	testScheme := runtime.NewScheme()
	_ = loggingService.AddToScheme(testScheme)

	cr := &loggingService.LoggingService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logging",
			Namespace: "logging",
		},
		Status: loggingService.LoggingServiceStatus{
			Conditions: conditions,
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithObjects(cr).Build()
	updater := NewStatusUpdater(fakeClient, cr)
	return &updater, cr
}

func TestGetCondition(t *testing.T) {
	conditions := []loggingService.LoggingServiceCondition{
		{Reason: "GraylogStatus", Type: "Successful"},
		{Reason: "FluentdStatus", Type: "Failed"},
	}

	t.Run("found by reason", func(t *testing.T) {
		updater, _ := newTestStatusUpdater(conditions)
		idx, cond := updater.GetCondition("FluentdStatus")
		if idx != 1 || cond == nil {
			t.Errorf("expected index 1, got %d", idx)
		}
		if cond.Type != "Failed" {
			t.Errorf("expected type Failed, got %q", cond.Type)
		}
	})

	t.Run("not found", func(t *testing.T) {
		updater, _ := newTestStatusUpdater(conditions)
		idx, cond := updater.GetCondition("NonExistent")
		if idx != -1 || cond != nil {
			t.Error("expected -1 and nil for non-existent condition")
		}
	})

	t.Run("empty reason", func(t *testing.T) {
		updater, _ := newTestStatusUpdater(conditions)
		idx, cond := updater.GetCondition("")
		if idx != -1 || cond != nil {
			t.Error("expected -1 and nil for empty reason")
		}
	})

	t.Run("empty conditions list", func(t *testing.T) {
		updater, _ := newTestStatusUpdater(nil)
		idx, cond := updater.GetCondition("GraylogStatus")
		if idx != -1 || cond != nil {
			t.Error("expected -1 and nil for empty conditions")
		}
	})
}

func TestIsStatusEqual(t *testing.T) {
	base := loggingService.LoggingServiceCondition{
		Type:               "Failed",
		Status:             false,
		Reason:             "GraylogStatus",
		Message:            "error occurred",
		LastTransitionTime: "2024-01-01T00:00:00Z",
	}

	t.Run("same conditions different time", func(t *testing.T) {
		source := base
		source.LastTransitionTime = "2024-06-01T00:00:00Z"
		target := base
		if !IsStatusEqual(source, &target) {
			t.Error("expected equal when only LastTransitionTime differs")
		}
	})

	t.Run("different Type", func(t *testing.T) {
		source := base
		target := base
		source.Type = "Successful"
		if IsStatusEqual(source, &target) {
			t.Error("expected not equal when Type differs")
		}
	})

	t.Run("different Message", func(t *testing.T) {
		source := base
		target := base
		source.Message = "different error"
		if IsStatusEqual(source, &target) {
			t.Error("expected not equal when Message differs")
		}
	})

	t.Run("different Reason", func(t *testing.T) {
		source := base
		target := base
		source.Reason = "FluentdStatus"
		if IsStatusEqual(source, &target) {
			t.Error("expected not equal when Reason differs")
		}
	})

	t.Run("different Status bool", func(t *testing.T) {
		source := base
		target := base
		source.Status = true
		if IsStatusEqual(source, &target) {
			t.Error("expected not equal when Status differs")
		}
	})
}

func TestIsStatusFailed(t *testing.T) {
	t.Run("exists and Failed", func(t *testing.T) {
		updater, _ := newTestStatusUpdater([]loggingService.LoggingServiceCondition{
			{Reason: "GraylogStatus", Type: Failed},
		})
		if !updater.IsStatusFailed("GraylogStatus") {
			t.Error("expected true for Failed condition")
		}
	})

	t.Run("exists but Successful", func(t *testing.T) {
		updater, _ := newTestStatusUpdater([]loggingService.LoggingServiceCondition{
			{Reason: "GraylogStatus", Type: Success},
		})
		if updater.IsStatusFailed("GraylogStatus") {
			t.Error("expected false for Successful condition")
		}
	})

	t.Run("does not exist", func(t *testing.T) {
		updater, _ := newTestStatusUpdater(nil)
		if updater.IsStatusFailed("GraylogStatus") {
			t.Error("expected false for missing condition")
		}
	})
}
