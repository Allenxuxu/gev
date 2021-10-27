package gev

import "sync"

type KeyValueContext struct {
	mu sync.RWMutex

	kv map[string]interface{}
}

func (c *KeyValueContext) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.kv == nil {
		c.kv = make(map[string]interface{})
	}

	c.kv[key] = value
	c.mu.Unlock()
}

func (c *KeyValueContext) Delete(key string) {
	c.mu.Lock()
	delete(c.kv, key)
	c.mu.Unlock()
}

func (c *KeyValueContext) Get(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	value, exists = c.kv[key]
	c.mu.RUnlock()
	return
}

func (c *KeyValueContext) reset() {
	c.mu.Lock()
	c.kv = nil
	c.mu.Unlock()
}
