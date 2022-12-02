package gev

import (
	"testing"
)

func TestKeyValueContext(t *testing.T) {
	ctx := KeyValueContext{}

	// Delete non-existent key
	ctx.Delete("1")

	// Set
	ctx.Set("1", 1)
	ctx.Set("2", 2)
	ctx.Set("3", 3)

	// Get
	if i, ok := ctx.Get("1"); !ok || i.(int) != 1 {
		t.Fatalf("i should be %v, but %v", i, i)
	}
	if i, ok := ctx.Get("2"); !ok || i.(int) != 2 {
		t.Fatalf("i should be %v, but %v", i, i)
	}
	if i, ok := ctx.Get("3"); !ok || i.(int) != 3 {
		t.Fatalf("i should be %v, but %v", i, i)
	}

	// Delete
	ctx.Delete("1")
	if _, ok := ctx.Get("1"); ok {
		t.Fatalf("ok should be false, but %t", ok)
	}

	// reset
	ctx.reset()
	if _, ok := ctx.Get("2"); ok {
		t.Fatalf("ok should be false, but %t", ok)
	}
}
