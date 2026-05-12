package utils

import (
	"testing"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestToPodNameList(t *testing.T) {
	manager := PodManager{}

	t.Run("empty list", func(t *testing.T) {
		podList := &core.PodList{}
		result := manager.ToPodNameList(podList)
		if len(result) != 0 {
			t.Errorf("expected empty list, got %v", result)
		}
	})

	t.Run("three pods", func(t *testing.T) {
		podList := &core.PodList{
			Items: []core.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "pod-1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod-2"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "pod-3"}},
			},
		}
		result := manager.ToPodNameList(podList)
		if len(result) != 3 {
			t.Fatalf("expected 3 names, got %d", len(result))
		}
		expected := []string{"pod-1", "pod-2", "pod-3"}
		for i, name := range expected {
			if result[i] != name {
				t.Errorf("index %d: expected %q, got %q", i, name, result[i])
			}
		}
	})
}
