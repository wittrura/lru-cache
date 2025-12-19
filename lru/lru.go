package lru

import (
	"container/list"
	"sync"
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

	c.list.MoveToBack(e)
	return e.Value.(entry).value, ok
}

func (c *LRUCache) Put(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	// check if this is an update
	if e, ok := c.values[key]; ok {
		e.Value = entry{key: key, value: value}
		c.list.MoveToBack(e)
	} else {
		// check about evicting
		if c.list.Len() == c.capacity {
			c.evictOldest()
		}
		// add the new one to the end
		e := c.list.PushBack(entry{key: key, value: value})
		// update the map
		c.values[key] = e
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
	value := c.list.Remove(e)
	delete(c.values, value.(entry).key)
}

type entry struct {
	key, value string
}
