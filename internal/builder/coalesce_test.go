package build

import (
	"reflect"
	"testing"
)

func TestFirst(t *testing.T) {
	if got := First("a", "b"); got != "a" {
		t.Fatalf("First non-zero a: got %q", got)
	}
	if got := First("", "b"); got != "b" {
		t.Fatalf("First zero a falls back: got %q", got)
	}
	if got := First(0, 7); got != 7 {
		t.Fatalf("First int zero a falls back: got %d", got)
	}
}

func TestFirstPtr(t *testing.T) {
	v := 42
	if got := FirstPtr(&v, 7); got != 42 {
		t.Fatalf("FirstPtr non-nil: got %d", got)
	}
	if got := FirstPtr[int](nil, 7); got != 7 {
		t.Fatalf("FirstPtr nil falls back: got %d", got)
	}
}

func TestFirstSlice(t *testing.T) {
	a := []string{"x"}
	b := []string{"y", "z"}
	if got := FirstSlice(a, b); !reflect.DeepEqual(got, a) {
		t.Fatalf("FirstSlice non-empty a: got %v", got)
	}
	if got := FirstSlice[string](nil, b); !reflect.DeepEqual(got, b) {
		t.Fatalf("FirstSlice nil a falls back: got %v", got)
	}
	if got := FirstSlice([]string{}, b); !reflect.DeepEqual(got, b) {
		t.Fatalf("FirstSlice empty a falls back: got %v", got)
	}
}

func TestFirstMap(t *testing.T) {
	a := map[string]int{"k": 1}
	b := map[string]int{"k": 2, "j": 3}
	if got := FirstMap(a, b); !reflect.DeepEqual(got, a) {
		t.Fatalf("FirstMap non-empty a: got %v", got)
	}
	if got := FirstMap[string, int](nil, b); !reflect.DeepEqual(got, b) {
		t.Fatalf("FirstMap nil a falls back: got %v", got)
	}
}
