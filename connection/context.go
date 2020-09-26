package connection

import "sync"

type Context struct {
	mu sync.RWMutex

	Keys map[string]interface{}
}

func (c *Context) Set(key string, value interface{}) {
	c.mu.Lock()
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}

	c.Keys[key] = value
	c.mu.Unlock()
}

func (c *Context) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Keys == nil {
		return
	}
	delete(c.Keys, key)
}

func (c *Context) Get(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	value, exists = c.Keys[key]
	c.mu.RUnlock()
	return
}

func (c *Context) reset() {
	c.mu.Lock()
	c.Keys = nil
	c.mu.Unlock()
}
