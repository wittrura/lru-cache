package lru

import (
	"container/list"
	"sync"
	"time"
)

type LRUCache struct {
	capacity int

	values map[string]*list.Element
	list   *list.List

	mu sync.Mutex
}

func NewLRU(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		values:   make(map[string]*list.Element),
		list:     list.New(),
	}
}

func (c *LRUCache) Get(key string) (value string, ok bool) {
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

	if !entry.expiration.IsZero() && entry.expiration.Before(time.Now()) {
		c.evict(e)
		return "", false
	}

	c.list.MoveToBack(e)
	return entry.value, ok
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

	entry := entry{key: key, value: value, expiration: time.Now().Add(ttl)}
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
	return len(c.values)
}

func (c *LRUCache) Clear() {
	c.values = make(map[string]*list.Element)
	c.list = list.New()
}

func (c *LRUCache) Resize(size int) {
	c.capacity = size
	if size < 0 {
		c.Clear()
		return
	}

	for c.Len() > size {
		c.evictOldest()
	}
}

func (c *LRUCache) evictOldest() {
	e := c.list.Front()
	c.evict(e)
}

func (c *LRUCache) evict(e *list.Element) {
	value := c.list.Remove(e)
	delete(c.values, value.(entry).key)
}

type entry struct {
	key, value string
	expiration time.Time
}
