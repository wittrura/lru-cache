package lru_test

import (
	"context"
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

func TestLenOnEmptyCache(t *testing.T) {
	cache := NewLRU(3)

	if got := cache.Len(); got != 0 {
		t.Fatalf("expected Len()=0 on empty cache, got %d", got)
	}
}

func TestLenAfterPutsAndEvictions(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("a", "1")
	if got := cache.Len(); got != 1 {
		t.Fatalf("expected Len()=1 after first Put, got %d", got)
	}

	cache.Put("b", "2")
	if got := cache.Len(); got != 2 {
		t.Fatalf("expected Len()=2 after second Put, got %d", got)
	}

	// Evict one, but length should remain at capacity.
	cache.Put("c", "3")
	if got := cache.Len(); got != 2 {
		t.Fatalf("expected Len()=2 after eviction Put, got %d", got)
	}
}

func TestClearEmptiesCache(t *testing.T) {
	cache := NewLRU(3)

	cache.Put("a", "1")
	cache.Put("b", "2")
	cache.Put("c", "3")

	cache.Clear()

	if got := cache.Len(); got != 0 {
		t.Fatalf("expected Len()=0 after Clear, got %d", got)
	}
	if _, ok := cache.Get("a"); ok {
		t.Fatalf("expected Get(a) to miss after Clear")
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatalf("expected Get(b) to miss after Clear")
	}
	if _, ok := cache.Get("c"); ok {
		t.Fatalf("expected Get(c) to miss after Clear")
	}
}

func TestClearIsIdempotent(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("a", "1")
	cache.Clear()
	cache.Clear() // should not panic

	if got := cache.Len(); got != 0 {
		t.Fatalf("expected Len()=0 after repeated Clear, got %d", got)
	}
}

func TestResizeDownEvictsOldestUntilWithinCapacity(t *testing.T) {
	cache := NewLRU(5)

	cache.Put("a", "1")
	cache.Put("b", "2")
	cache.Put("c", "3")
	cache.Put("d", "4")
	cache.Put("e", "5")

	// Make "a" and then "b" the most recently used so eviction order is clear.
	_, _ = cache.Get("a")
	_, _ = cache.Get("b")

	// Current LRU order (front->back) should be: c, d, e, a, b
	cache.Resize(2)

	if got := cache.Len(); got != 2 {
		t.Fatalf("expected Len()=2 after Resize(2), got %d", got)
	}

	// Only the two most-recently-used should remain: a and b.
	if _, ok := cache.Get("c"); ok {
		t.Fatalf("expected c to be evicted after Resize down")
	}
	if _, ok := cache.Get("d"); ok {
		t.Fatalf("expected d to be evicted after Resize down")
	}
	if _, ok := cache.Get("e"); ok {
		t.Fatalf("expected e to be evicted after Resize down")
	}

	if v, ok := cache.Get("a"); !ok || v != "1" {
		t.Fatalf("expected a to remain with value 1, got %q ok=%v", v, ok)
	}
	if v, ok := cache.Get("b"); !ok || v != "2" {
		t.Fatalf("expected b to remain with value 2, got %q ok=%v", v, ok)
	}
}

func TestResizeUpAllowsMoreEntries(t *testing.T) {
	cache := NewLRU(1)

	cache.Put("a", "1")
	cache.Resize(3)

	cache.Put("b", "2")
	cache.Put("c", "3")

	if got := cache.Len(); got != 3 {
		t.Fatalf("expected Len()=3 after Resize up and 3 puts, got %d", got)
	}

	if v, ok := cache.Get("a"); !ok || v != "1" {
		t.Fatalf("expected a to remain with value 1, got %q ok=%v", v, ok)
	}
	if v, ok := cache.Get("b"); !ok || v != "2" {
		t.Fatalf("expected b to remain with value 2, got %q ok=%v", v, ok)
	}
	if v, ok := cache.Get("c"); !ok || v != "3" {
		t.Fatalf("expected c to remain with value 3, got %q ok=%v", v, ok)
	}
}

func TestResizeToZeroDisablesCacheAndClearsExistingItems(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("a", "1")
	cache.Put("b", "2")
	cache.Resize(0)

	if got := cache.Len(); got != 0 {
		t.Fatalf("expected Len()=0 after Resize(0), got %d", got)
	}
	if _, ok := cache.Get("a"); ok {
		t.Fatalf("expected a to miss after Resize(0)")
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatalf("expected b to miss after Resize(0)")
	}

	// Disabled semantics: Put is a no-op.
	cache.Put("c", "3")
	if got := cache.Len(); got != 0 {
		t.Fatalf("expected Len()=0 after Put on disabled cache, got %d", got)
	}
	if _, ok := cache.Get("c"); ok {
		t.Fatalf("expected c to miss on disabled cache")
	}
}

func TestResizeNegativeDisablesCacheAndClearsExistingItems(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("a", "1")
	cache.Put("b", "2")
	cache.Resize(-5)

	if got := cache.Len(); got != 0 {
		t.Fatalf("expected Len()=0 after Resize(-5), got %d", got)
	}
	if _, ok := cache.Get("a"); ok {
		t.Fatalf("expected a to miss after Resize(-5)")
	}
	if _, ok := cache.Get("b"); ok {
		t.Fatalf("expected b to miss after Resize(-5)")
	}
}

func TestPutWithTTL_GetBeforeExpiryHits(t *testing.T) {
	cache := NewLRU(2)

	cache.PutWithTTL("k1", "v1", 150*time.Millisecond)

	if v, ok := cache.Get("k1"); !ok || v != "v1" {
		t.Fatalf("expected k1 to be present before expiry, got %q ok=%v", v, ok)
	}
}

func TestPutWithTTL_GetAfterExpiryMissesAndRemovesEntry(t *testing.T) {
	cache := NewLRU(2)

	cache.PutWithTTL("k1", "v1", 30*time.Millisecond)
	if got := cache.Len(); got != 1 {
		t.Fatalf("expected Len()=1 after PutWithTTL, got %d", got)
	}

	time.Sleep(80 * time.Millisecond)

	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected k1 to be expired and missed")
	}

	// Lazy expiration should delete the entry on access.
	if got := cache.Len(); got != 0 {
		t.Fatalf("expected Len()=0 after accessing expired key, got %d", got)
	}
}

func TestPutWithoutTTL_DoesNotExpire(t *testing.T) {
	cache := NewLRU(2)

	cache.Put("k1", "v1")

	time.Sleep(80 * time.Millisecond)

	if v, ok := cache.Get("k1"); !ok || v != "v1" {
		t.Fatalf("expected non-TTL entry to not expire, got %q ok=%v", v, ok)
	}
}

func TestExpiredEntryDoesNotBlockCapacityAfterItExpires(t *testing.T) {
	cache := NewLRU(1)

	cache.PutWithTTL("k1", "v1", 30*time.Millisecond)
	time.Sleep(80 * time.Millisecond)

	// After expiry, inserting a new key should work normally.
	cache.Put("k2", "v2")

	if v, ok := cache.Get("k2"); !ok || v != "v2" {
		t.Fatalf("expected k2 to be present, got %q ok=%v", v, ok)
	}

	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected k1 to be expired and missing")
	}

	if got := cache.Len(); got != 1 {
		t.Fatalf("expected Len()=1 after inserting k2 into capacity-1 cache, got %d", got)
	}
}

func waitUntil(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within %s", timeout)
}

func TestEvictorRemovesExpiredEntriesWithoutAccess(t *testing.T) {
	cache := NewLRU(10)

	ctx := t.Context()

	cache.StartEvictor(ctx, 10*time.Millisecond)

	cache.PutWithTTL("k1", "v1", 25*time.Millisecond)
	cache.PutWithTTL("k2", "v2", 25*time.Millisecond)

	// Wait until both are expired AND evictor has swept.
	waitUntil(t, 300*time.Millisecond, func() bool {
		return cache.Len() == 0
	})

	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected k1 to be evicted after TTL expiry")
	}
	if _, ok := cache.Get("k2"); ok {
		t.Fatalf("expected k2 to be evicted after TTL expiry")
	}
}

func TestEvictorDoesNotRemoveNonExpiredEntries(t *testing.T) {
	cache := NewLRU(10)

	ctx := t.Context()

	cache.StartEvictor(ctx, 10*time.Millisecond)

	cache.PutWithTTL("k1", "v1", 200*time.Millisecond)

	// Give the evictor time to run a few cycles.
	time.Sleep(60 * time.Millisecond)

	if v, ok := cache.Get("k1"); !ok || v != "v1" {
		t.Fatalf("expected k1 to still be present before expiry, got %q ok=%v", v, ok)
	}
	if cache.Len() != 1 {
		t.Fatalf("expected Len()=1 before expiry, got %d", cache.Len())
	}
}

func TestEvictorStopsAfterContextCancel(t *testing.T) {
	cache := NewLRU(10)

	ctx, cancel := context.WithCancel(context.Background())
	cache.StartEvictor(ctx, 10*time.Millisecond)

	cache.PutWithTTL("k1", "v1", 25*time.Millisecond)

	// Cancel quickly so the evictor should stop before it can reliably sweep.
	cancel()

	// Wait long enough that TTL has definitely expired.
	time.Sleep(80 * time.Millisecond)

	// After cancel, we should NOT expect the background goroutine to remove it.
	// Lazy expiration via Get is still allowed, so we only check Len *before* Get.
	if cache.Len() == 0 {
		t.Fatalf("expected k1 to still be present in storage after cancel (no background sweep)")
	}

	// Now access should lazily remove it (existing behavior).
	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected k1 to be expired and missed on Get")
	}
	if cache.Len() != 0 {
		t.Fatalf("expected Len()=0 after Get removes expired entry, got %d", cache.Len())
	}
}

func TestEvictorOnDisabledCacheIsSafeNoop(t *testing.T) {
	cache := NewLRU(0)

	ctx := t.Context()

	// Should not panic / race
	cache.StartEvictor(ctx, 5*time.Millisecond)

	cache.PutWithTTL("k1", "v1", 10*time.Millisecond)
	time.Sleep(30 * time.Millisecond)

	if cache.Len() != 0 {
		t.Fatalf("expected disabled cache to remain empty, got Len()=%d", cache.Len())
	}
	if _, ok := cache.Get("k1"); ok {
		t.Fatalf("expected disabled cache to always miss")
	}
}
