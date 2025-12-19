package lru

import "container/list"

type LRUCache struct {
	capacity int

	values map[string]*list.Element
	list   *list.List
}

func NewLRU(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		values:   make(map[string]*list.Element),
		list:     list.New(),
	}
}

func (c *LRUCache) Get(key string) (value string, ok bool) {
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
			// remove front element from the list
			e := c.list.Front()
			_ = c.list.Remove(e)
			// and delete its key from the map
			delete(c.values, e.Value.(entry).key)
		}
		// add the new one to the end
		e := c.list.PushBack(entry{key: key, value: value})
		// update the map
		c.values[key] = e
	}
}

type entry struct {
	key, value string
}
