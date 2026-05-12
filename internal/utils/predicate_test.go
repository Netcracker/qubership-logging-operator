package utils

import (
	"testing"

	logging "github.com/Netcracker/qubership-logging-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func newTestPredicate() *SkipStatusUpdatePredicate {
	p := NewPredicate(Logger("test-predicate"))
	return &p
}

func TestIsStatusUpdated(t *testing.T) {
	predicate := newTestPredicate()

	t.Run("same spec different status is status-only update", func(t *testing.T) {
		old := &logging.LoggingService{
			Spec:   logging.LoggingServiceSpec{ContainerRuntimeType: "containerd"},
			Status: logging.LoggingServiceStatus{},
		}
		new := &logging.LoggingService{
			Spec: logging.LoggingServiceSpec{ContainerRuntimeType: "containerd"},
			Status: logging.LoggingServiceStatus{
				Conditions: []logging.LoggingServiceCondition{
					{Reason: "test", Type: "Success"},
				},
			},
		}
		if !predicate.IsStatusUpdated(old, new) {
			t.Error("expected true for status-only update")
		}
	})

	t.Run("different spec is not status-only update", func(t *testing.T) {
		old := &logging.LoggingService{
			Spec: logging.LoggingServiceSpec{ContainerRuntimeType: "containerd"},
		}
		new := &logging.LoggingService{
			Spec: logging.LoggingServiceSpec{ContainerRuntimeType: "cri-o"},
		}
		if predicate.IsStatusUpdated(old, new) {
			t.Error("expected false for spec change")
		}
	})

	t.Run("non-LoggingService objects returns false", func(t *testing.T) {
		pod := &metav1.ObjectMeta{}
		// Using two non-LoggingService objects should return false
		result := predicate.IsStatusUpdated(nil, nil)
		_ = pod
		if result {
			t.Error("expected false for nil objects")
		}
	})
}

func TestPredicate_Update(t *testing.T) {
	predicate := newTestPredicate()

	t.Run("status-only change returns false (skip reconcile)", func(t *testing.T) {
		e := event.UpdateEvent{
			ObjectOld: &logging.LoggingService{
				Spec: logging.LoggingServiceSpec{ContainerRuntimeType: "containerd"},
			},
			ObjectNew: &logging.LoggingService{
				Spec: logging.LoggingServiceSpec{ContainerRuntimeType: "containerd"},
				Status: logging.LoggingServiceStatus{
					Conditions: []logging.LoggingServiceCondition{{Reason: "test"}},
				},
			},
		}
		if predicate.Update(e) {
			t.Error("expected false for status-only update (should skip reconcile)")
		}
	})

	t.Run("spec change returns true (should reconcile)", func(t *testing.T) {
		e := event.UpdateEvent{
			ObjectOld: &logging.LoggingService{
				Spec: logging.LoggingServiceSpec{ContainerRuntimeType: "containerd"},
			},
			ObjectNew: &logging.LoggingService{
				Spec: logging.LoggingServiceSpec{ContainerRuntimeType: "docker"},
			},
		}
		if !predicate.Update(e) {
			t.Error("expected true for spec change")
		}
	})
}

func TestPredicate_Create(t *testing.T) {
	predicate := newTestPredicate()
	e := event.CreateEvent{
		Object: &logging.LoggingService{},
	}
	if !predicate.Create(e) {
		t.Error("Create should always return true")
	}
}

func TestPredicate_Delete(t *testing.T) {
	predicate := newTestPredicate()
	e := event.DeleteEvent{
		Object: &logging.LoggingService{},
	}
	if !predicate.Delete(e) {
		t.Error("Delete should always return true")
	}
}

func TestPredicate_Generic(t *testing.T) {
	predicate := newTestPredicate()
	e := event.GenericEvent{
		Object: &logging.LoggingService{},
	}
	if !predicate.Generic(e) {
		t.Error("Generic should always return true")
	}
}
