package builder

import (
	"reflect"
	"testing"
)

func TestFirst(t *testing.T) {
	if got := First("a", "b"); got != "a" {
		t.Errorf("First(a, b) = %q, want a", got)
	}
	if got := First("", "b"); got != "b" {
		t.Errorf("First(\"\", b) = %q, want b", got)
	}
	if got := First(0, 7); got != 7 {
		t.Errorf("First(0, 7) = %d, want 7", got)
	}
}

func TestFirstNonNil(t *testing.T) {
	a, b := 1, 2
	if got := FirstNonNil(&a, &b); got != &a {
		t.Errorf("FirstNonNil(&a, &b) did not return &a")
	}
	if got := FirstNonNil(nil, &b); got != &b {
		t.Errorf("FirstNonNil(nil, &b) did not return &b")
	}
}

func TestFirstSlice(t *testing.T) {
	if got := FirstSlice([]int{1}, []int{2}); !reflect.DeepEqual(got, []int{1}) {
		t.Errorf("FirstSlice([1], [2]) = %v, want [1]", got)
	}
	if got := FirstSlice(nil, []int{2}); !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("FirstSlice(nil, [2]) = %v, want [2]", got)
	}
}

func TestFirstMap(t *testing.T) {
	a, b := map[string]int{"a": 1}, map[string]int{"b": 2}
	if got := FirstMap(a, b); !reflect.DeepEqual(got, a) {
		t.Errorf("FirstMap(a, b) = %v, want a", got)
	}
	if got := FirstMap(nil, b); !reflect.DeepEqual(got, b) {
		t.Errorf("FirstMap(nil, b) = %v, want b", got)
	}
}
