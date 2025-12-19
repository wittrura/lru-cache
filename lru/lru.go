package lru

import "container/list"

type LRUCache struct {
	capacity int

	values map[string]string
	list   *list.List
}

func NewLRU(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		values:   make(map[string]string),
		list:     list.New(),
	}
}

func (c *LRUCache) Get(key string) (value string, ok bool) {
	value, ok = c.values[key]
	return value, ok
}

func (c *LRUCache) Put(key, value string) {
	if c.list.Len() == c.capacity {
		v := c.list.Front()
		c.list.Remove(v)
		delete(c.values, v.Value.(string))
	} else {
		c.list.PushBack(key)
	}
	c.values[key] = value
}
