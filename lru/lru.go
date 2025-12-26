package lru

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type LRUStringCache struct {
	*LRUCache[string, string]
}

func NewLRUStringCache(capacity int) *LRUStringCache {
	return &LRUStringCache{
		LRUCache: NewLRUCache[string, string](capacity),
	}
}

func NewLRUStringCacheWithNow(capacity int, now func() time.Time) *LRUStringCache {
	return &LRUStringCache{
		LRUCache: NewLRUCacheWithNow[string, string](capacity, now),
	}
}

type LRUCache[K comparable, V any] struct {
	capacity int

	values map[K]*list.Element
	list   *list.List

	now func() time.Time

	evictorIsRunning bool

	mu sync.Mutex
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		values:   make(map[K]*list.Element),
		list:     list.New(),
		now:      time.Now,
	}
}

func NewLRUCacheWithNow[K comparable, V any](capacity int, now func() time.Time) *LRUCache[K, V] {
	if now == nil {
		now = time.Now
	}

	return &LRUCache[K, V]{
		capacity: capacity,
		values:   make(map[K]*list.Element),
		list:     list.New(),
		now:      now,
	}
}

func (c *LRUCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var zero V

	if c.capacity <= 0 {
		return zero, false
	}

	e, ok := c.values[key]
	if !ok {
		return zero, false
	}

	entry := e.Value.(entry[K, V])

	if entry.isExpired(c.now()) {
		c.evict(e)
		return zero, false
	}

	c.list.MoveToBack(e)
	return entry.value, true
}

func (c *LRUCache[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	entry := entry[K, V]{key: key, value: value, expiration: time.Time{}}
	c.put(entry)
}

func (c *LRUCache[K, V]) PutWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	entry := entry[K, V]{key: key, value: value, expiration: c.now().Add(ttl)}
	c.put(entry)
}

func (c *LRUCache[K, V]) put(entry entry[K, V]) {
	// check if this is an update
	if e, ok := c.values[entry.key]; ok {
		e.Value = entry
		c.list.MoveToBack(e)
	} else { // otherwise
		// evict oldeset if necessary
		if c.list.Len() == c.capacity {
			c.evictOldest()
		}

		// and add new key
		e := c.list.PushBack(entry)
		c.values[entry.key] = e
	}
}

func (c *LRUCache[K, V]) evictOldest() {
	e := c.list.Front()
	c.evict(e)
}

func (c *LRUCache[K, V]) evict(e *list.Element) {
	if e == nil {
		return
	}
	value := c.list.Remove(e)
	delete(c.values, value.(entry[K, V]).key)
}

func (c *LRUCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.values)
}

func (c *LRUCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.values = make(map[K]*list.Element)
	c.list = list.New()
}

func (c *LRUCache[K, V]) Resize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = size
	if size <= 0 {
		// clear
		c.values = make(map[K]*list.Element)
		c.list = list.New()
		return
	}

	for len(c.values) > size {
		c.evictOldest()
	}
}

func (c *LRUCache[K, V]) StartEvictor(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		return
	}

	// avoid kicking off multiple go routines for evicting
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.evictorIsRunning {
		return
	}
	c.evictorIsRunning = true

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				t := c.now()
				c.mu.Lock()
				for e := c.list.Front(); e != nil; {
					next := e.Next() // grab next in case we evict current
					entry := e.Value.(entry[K, V])
					if entry.isExpired(t) {
						c.evict(e)
					}
					e = next
				}
				c.mu.Unlock()
			}
		}
	}()
}

type entry[K comparable, V any] struct {
	key        K
	value      V
	expiration time.Time
}

func (e entry[K, V]) isExpired(t time.Time) bool {
	return !e.expiration.IsZero() &&
		!e.expiration.After(t) // expire is before OR equal to current time t
}
