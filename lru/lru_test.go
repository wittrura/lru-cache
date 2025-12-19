package lru_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

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

func TestPutAtCapacityEvictsAndStillInsertsNewKey(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("k1", "v1")
	cache.Put("k2", "v2")

	// Cache is now at capacity.
	cache.Put("k3", "v3") // should evict k1 AND insert k3

	// k1 should be gone
	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected k1 to be evicted when inserting k3 at capacity")
	}

	// k3 must be present
	if v, ok := cache.Get("k3"); !ok || v != "v3" {
		t.Fatalf("expected k3 to be inserted with value v3, got %q ok=%v", v, ok)
	}

	// One of the original keys (k2) should still exist
	if v, ok := cache.Get("k2"); !ok || v != "v2" {
		t.Fatalf("expected k2 to remain with value v2, got %q ok=%v", v, ok)
	}
}

func TestGetMovesKeyToMostRecentlyUsed(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("k1", "v1")
	cache.Put("k2", "v2")

	// Access k1, making it MRU. Now k2 should be LRU.
	if v, ok := cache.Get("k1"); !ok || v != "v1" {
		t.Fatalf("expected Get(k1) to return v1, got %q ok=%v", v, ok)
	}

	// Add k3. Should evict k2 (the LRU), not k1.
	cache.Put("k3", "v3")

	if _, ok := cache.Get("k2"); ok {
		t.Fatalf("expected k2 to be evicted (LRU) after accessing k1 then inserting k3")
	}
	if v, ok := cache.Get("k1"); !ok || v != "v1" {
		t.Fatalf("expected k1 to remain, got %q ok=%v", v, ok)
	}
	if v, ok := cache.Get("k3"); !ok || v != "v3" {
		t.Fatalf("expected k3 to remain, got %q ok=%v", v, ok)
	}
}

func TestPutOnExistingKeyUpdatesValueAndMovesToMostRecentlyUsed(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("k1", "v1")
	cache.Put("k2", "v2")

	// Updating k1 should make it MRU.
	cache.Put("k1", "v1b")

	// Next insert should evict k2.
	cache.Put("k3", "v3")

	if _, ok := cache.Get("k2"); ok {
		t.Fatalf("expected k2 to be evicted after updating k1 then inserting k3")
	}
	if v, ok := cache.Get("k1"); !ok || v != "v1b" {
		t.Fatalf("expected k1 to remain with updated value v1b, got %q ok=%v", v, ok)
	}
	if v, ok := cache.Get("k3"); !ok || v != "v3" {
		t.Fatalf("expected k3 to remain, got %q ok=%v", v, ok)
	}
}

func TestGetMissDoesNotAffectRecency(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("k1", "v1")
	cache.Put("k2", "v2")

	// Miss should not change which key is LRU.
	if _, ok := cache.Get("nope"); ok {
		t.Fatalf("expected miss to return ok=false")
	}

	// Insert k3; should evict k1 (still LRU).
	cache.Put("k3", "v3")

	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected k1 to be evicted (still LRU) after a miss and then inserting k3")
	}
	if v, ok := cache.Get("k2"); !ok || v != "v2" {
		t.Fatalf("expected k2 to remain, got %q ok=%v", v, ok)
	}
	if v, ok := cache.Get("k3"); !ok || v != "v3" {
		t.Fatalf("expected k3 to remain, got %q ok=%v", v, ok)
	}
}

func TestRepeatedGetsKeepKeyMostRecentlyUsed(t *testing.T) {
	cache := NewLRU(3)

	cache.Put("a", "1")
	cache.Put("b", "2")
	cache.Put("c", "3")

	// Touch a a couple times; it should remain MRU.
	if _, ok := cache.Get("a"); !ok {
		t.Fatalf("expected a to be present")
	}
	if _, ok := cache.Get("a"); !ok {
		t.Fatalf("expected a to be present")
	}

	// Touch b once. Now c should be LRU (it hasn't been touched since insertion).
	if _, ok := cache.Get("b"); !ok {
		t.Fatalf("expected b to be present")
	}

	// Insert d, capacity=3 => evict c.
	cache.Put("d", "4")

	if _, ok := cache.Get("c"); ok {
		t.Fatalf("expected c to be evicted as LRU")
	}
	if v, ok := cache.Get("a"); !ok || v != "1" {
		t.Fatalf("expected a to remain, got %q ok=%v", v, ok)
	}
	if v, ok := cache.Get("b"); !ok || v != "2" {
		t.Fatalf("expected b to remain, got %q ok=%v", v, ok)
	}
	if v, ok := cache.Get("d"); !ok || v != "4" {
		t.Fatalf("expected d to remain, got %q ok=%v", v, ok)
	}
}

func TestZeroCapacityActsAsDisabledCache(t *testing.T) {
	cache := NewLRU(0)

	cache.Put("k1", "v1")
	cache.Put("k2", "v2")

	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected disabled cache (capacity=0) to always miss on Get")
	}
	if _, ok := cache.Get("k2"); ok {
		t.Fatalf("expected disabled cache (capacity=0) to always miss on Get")
	}
	if _, ok := cache.Get("missing"); ok {
		t.Fatalf("expected disabled cache (capacity=0) to always miss on Get")
	}
}

func TestNegativeCapacityActsAsDisabledCache(t *testing.T) {
	cache := NewLRU(-1)

	cache.Put("k1", "v1")
	cache.Put("k2", "v2")

	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected disabled cache (capacity<0) to always miss on Get")
	}
	if _, ok := cache.Get("k2"); ok {
		t.Fatalf("expected disabled cache (capacity<0) to always miss on Get")
	}
	if _, ok := cache.Get("missing"); ok {
		t.Fatalf("expected disabled cache (capacity<0) to always miss on Get")
	}
}

func TestConcurrentPutGetDoesNotPanicOrRace(t *testing.T) {
	// Run this with:
	//   go test -race ./...
	//
	// This test is mostly about safety (no panic / no data race).
	cache := NewLRU(64)

	const (
		goroutines = 16
		opsPerG    = 2000
		keySpace   = 128
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(seed int64) {
			defer wg.Done()

			r := rand.New(rand.NewSource(seed))
			for i := 0; i < opsPerG; i++ {
				k := fmt.Sprintf("k-%d", r.Intn(keySpace))

				// Mix reads/writes.
				if r.Intn(100) < 45 {
					cache.Put(k, fmt.Sprintf("v-%d", r.Int()))
				} else {
					_, _ = cache.Get(k)
				}
			}
		}(time.Now().UnixNano() + int64(g))
	}

	wg.Wait()

	// Simple sanity check: after some writes, at least one key should be present.
	// (Not strictly guaranteed, but extremely likely given the workload.)
	found := false
	for i := 0; i < keySpace; i++ {
		if _, ok := cache.Get(fmt.Sprintf("k-%d", i)); ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one key to be present after concurrent ops")
	}
}

func TestConcurrentSameKeyWritersAndReaders(t *testing.T) {
	// Hammer a single key concurrently. This catches common issues around
	// list element mutation and map updates.
	cache := NewLRU(2)

	const (
		writers = 8
		readers = 8
		ops     = 2000
	)

	var wg sync.WaitGroup
	wg.Add(writers + readers)

	for w := 0; w < writers; w++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				cache.Put("hot", fmt.Sprintf("writer-%d-%d", id, i))
			}
		}(w)
	}

	for r := 0; r < readers; r++ {
		go func() {
			defer wg.Done()
			for i := 0; i < ops; i++ {
				_, _ = cache.Get("hot")
			}
		}()
	}

	wg.Wait()

	// Must not panic; value is nondeterministic but should be present
	// unless capacity is disabled.
	if _, ok := cache.Get("hot"); !ok {
		t.Fatalf("expected 'hot' key to exist after many concurrent writes")
	}
}

func TestConcurrentDisabledCacheStillSafe(t *testing.T) {
	// Even if capacity disables the cache, concurrent access should be safe.
	cache := NewLRU(0)

	const (
		goroutines = 16
		opsPerG    = 1000
	)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerG; i++ {
				cache.Put(fmt.Sprintf("k-%d", id), "v")
				_, _ = cache.Get("k-any")
			}
		}(g)
	}

	wg.Wait()

	// Disabled cache should always miss.
	if _, ok := cache.Get("k-any"); ok {
		t.Fatalf("expected disabled cache to always miss")
	}
}
