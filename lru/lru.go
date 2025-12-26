package lru

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type LRUCache struct {
	capacity int

	values map[string]*list.Element
	list   *list.List

	now func() time.Time

	evictorIsRunning bool

	mu sync.Mutex
}

func NewLRU(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		values:   make(map[string]*list.Element),
		list:     list.New(),
		now:      time.Now,
	}
}

func NewLRUWithNow(capacity int, now func() time.Time) *LRUCache {
	if now == nil {
		now = time.Now
	}

	return &LRUCache{
		capacity: capacity,
		values:   make(map[string]*list.Element),
		list:     list.New(),
		now:      now,
	}
}

func (c *LRUCache) Get(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return "", false
	}

	e, ok := c.values[key]
	if !ok {
		return "", false
	}

	entry := e.Value.(entry)

	if entry.isExpired(c.now()) {
		c.evict(e)
		return "", false
	}

	c.list.MoveToBack(e)
	return entry.value, true
}

func (c *LRUCache) Put(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	entry := entry{key: key, value: value, expiration: time.Time{}}
	c.put(entry)
}

func (c *LRUCache) PutWithTTL(key, value string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	entry := entry{key: key, value: value, expiration: c.now().Add(ttl)}
	c.put(entry)
}

func (c *LRUCache) put(entry entry) {
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

func (c *LRUCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.values)
}

func (c *LRUCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.values = make(map[string]*list.Element)
	c.list = list.New()
}

func (c *LRUCache) Resize(size int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.capacity = size
	if size <= 0 {
		// clear
		c.values = make(map[string]*list.Element)
		c.list = list.New()
		return
	}

	for len(c.values) > size {
		c.evictOldest()
	}
}

func (c *LRUCache) StartEvictor(ctx context.Context, interval time.Duration) {
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
					entry := e.Value.(entry)
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

func (c *LRUCache) evictOldest() {
	e := c.list.Front()
	c.evict(e)
}

func (c *LRUCache) evict(e *list.Element) {
	if e == nil {
		return
	}
	value := c.list.Remove(e)
	delete(c.values, value.(entry).key)
}

type entry struct {
	key, value string
	expiration time.Time
}

func (e entry) isExpired(t time.Time) bool {
	return !e.expiration.IsZero() &&
		!e.expiration.After(t) // expire is before OR equal to current time t
}

type LRU[K comparable, V any] struct {
	capacity int

	values map[K]*list.Element
	list   *list.List

	now func() time.Time

	evictorIsRunning bool

	mu sync.Mutex
}

func New[K comparable, V any](capacity int) *LRU[K, V] {
	return &LRU[K, V]{
		capacity: capacity,
		values:   make(map[K]*list.Element),
		list:     list.New(),
		now:      time.Now,
	}
}

func (c *LRU[K, V]) Get(key K) (V, bool) {
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

	entry := e.Value.(entry2[K, V])

	if entry.isExpired(c.now()) {
		c.evict(e)
		return zero, false
	}

	c.list.MoveToBack(e)
	return entry.value, true
}

func (c *LRU[K, V]) Put(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	entry := entry2[K, V]{key: key, value: value, expiration: time.Time{}}
	c.put(entry)
}

func (c *LRU[K, V]) PutWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	entry := entry2[K, V]{key: key, value: value, expiration: c.now().Add(ttl)}
	c.put(entry)
}

func (c *LRU[K, V]) put(entry entry2[K, V]) {
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

func (c *LRU[K, V]) evictOldest() {
	e := c.list.Front()
	c.evict(e)
}

func (c *LRU[K, V]) evict(e *list.Element) {
	if e == nil {
		return
	}
	value := c.list.Remove(e)
	delete(c.values, value.(entry2[K, V]).key)
}

func (c *LRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.values)
}

func (c *LRU[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.values = make(map[K]*list.Element)
	c.list = list.New()
}

func (c *LRU[K, V]) Resize(size int) {
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

type entry2[K comparable, V any] struct {
	key        K
	value      V
	expiration time.Time
}

func (e entry2[K, V]) isExpired(t time.Time) bool {
	return !e.expiration.IsZero() &&
		!e.expiration.After(t) // expire is before OR equal to current time t
}
