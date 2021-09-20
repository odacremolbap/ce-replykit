package main

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type value struct {
	n       int
	expires time.Time
	le      *list.Element
}

// RequestCountPerID is an inmemory aggregate count for IDs.
type RequestCountPerID struct {
	count map[string]*value
	rl    *list.List
	ttl   time.Duration

	m sync.RWMutex
}

// NewRequestCountPerID creates a new inmemory ID storage
// and starts a garbage collector for stale elements.
func NewRequestCountPerID(ctx context.Context, ttl, gc time.Duration) *RequestCountPerID {
	rc := &RequestCountPerID{
		count: make(map[string]*value),
		rl:    list.New(),
		ttl:   ttl,
	}

	go func() {
		for {
			select {
			case <-time.After(gc):
				rc.RemoveStale()
			case <-ctx.Done():
				return
			}
		}
	}()

	return rc
}

// Increase the request count for the ID at the ephemeral
// in-memory storage. Returns the updated count value.
func (c *RequestCountPerID) Increase(id string) int {
	expires := time.Now().Add(c.ttl)
	c.m.Lock()
	defer c.m.Unlock()

	v, ok := c.count[id]
	if ok {
		if v.expires.Before(time.Now()) {
			// reset value for stale keys.
			ok = false
		}
	}

	if !ok {
		el := c.rl.PushFront(id)
		c.count[id] = &value{
			n:       0,
			expires: expires,
			le:      el,
		}
		return 0
	}

	v.n++
	v.expires = expires
	c.rl.MoveToFront(v.le)

	return v.n
}

// Get the request count for the ID
func (c *RequestCountPerID) Get(id string) int {
	c.m.RLock()
	defer c.m.RUnlock()

	v, ok := c.count[id]
	if !ok || v.expires.Before(time.Now()) {
		return 0
	}

	return v.n
}

// Delete the request count for the ID
func (c *RequestCountPerID) Delete(id string) {
	c.m.Lock()
	defer c.m.Unlock()

	if v, ok := c.count[id]; ok {
		c.rl.Remove(v.le)
		delete(c.count, id)
	}
}

// Reset removes all elements in the store.
func (c *RequestCountPerID) Reset() {
	c.m.Lock()
	defer c.m.Unlock()

	c.count = make(map[string]*value)
	c.rl = list.New()
}

// RemoveStale looks for request count elements and removes
// expired ones.
func (c *RequestCountPerID) RemoveStale() {
	c.m.Lock()
	defer c.m.Unlock()

	for {
		e := c.rl.Back()
		if e == nil {
			return
		}

		id := e.Value.(string)
		if c.count[id].expires.Before(time.Now()) {
			return
		}
		c.rl.Remove(e)
		delete(c.count, id)
	}
}
