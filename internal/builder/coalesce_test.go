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
	a, b := new(int), new(int)
	if FirstNonNil(a, b) != a {
		t.Error("FirstNonNil(a, b) did not return a")
	}
	if FirstNonNil(nil, b) != b {
		t.Error("FirstNonNil(nil, b) did not return b")
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
