package lru_test

import (
	"testing"

	. "example.com/lru-cache/lru"
)

func TestGetOnEmptyCacheReturnsNotFound(t *testing.T) {
	cache := NewLRU(2)

	if _, ok := cache.Get("missing"); ok {
		t.Fatalf("expected ok=false for missing key in empty cache")
	}
}

func TestPutAndGetSingleItem(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("foo", "bar")

	got, ok := cache.Get("foo")
	if !ok {
		t.Fatalf("expected key 'foo' to be found")
	}
	if got != "bar" {
		t.Fatalf("expected value 'bar', got %q", got)
	}
}

func TestPutOverwritesExistingKey(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("foo", "bar")
	cache.Put("foo", "baz") // overwrite

	got, ok := cache.Get("foo")
	if !ok {
		t.Fatalf("expected key 'foo' to be found after overwrite")
	}
	if got != "baz" {
		t.Fatalf("expected value 'baz' after overwrite, got %q", got)
	}
}

func TestEvictsLeastRecentlyInsertedWhenCapacityExceeded(t *testing.T) {
	// For this first iteration we only rely on insertion order:
	// with no intervening Get calls, the earliest inserted key
	// should be evicted when capacity is exceeded.
	cache := NewLRU(2)

	cache.Put("k1", "v1")
	cache.Put("k2", "v2")
	cache.Put("k3", "v3") // should evict k1

	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected 'k1' to be evicted after exceeding capacity")
	}

	if v, ok := cache.Get("k2"); !ok || v != "v2" {
		t.Fatalf("expected 'k2' to remain with value 'v2', got %q, ok=%v", v, ok)
	}

	if v, ok := cache.Get("k3"); !ok || v != "v3" {
		t.Fatalf("expected 'k3' to remain with value 'v3', got %q, ok=%v", v, ok)
	}
}

func TestEvictionWithExactCapacity(t *testing.T) {
	cache := NewLRU(1)

	cache.Put("a", "1")
	if v, ok := cache.Get("a"); !ok || v != "1" {
		t.Fatalf("expected 'a' to be present with value '1', got %q, ok=%v", v, ok)
	}

	cache.Put("b", "2") // should evict 'a'

	if _, ok := cache.Get("a"); ok {
		t.Fatalf("expected 'a' to be evicted when inserting 'b' into capacity-1 cache")
	}

	if v, ok := cache.Get("b"); !ok || v != "2" {
		t.Fatalf("expected 'b' to be present with value '2', got %q, ok=%v", v, ok)
	}
}
