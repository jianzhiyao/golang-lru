package lru

import (
	"github.com/hashicorp/golang-lru/simplelru"
	"sync"
	"sync/atomic"
)

// Cache is a thread-safe fixed size LRU cache.
type Cache struct {
	lru  simplelru.LRUCache
	lock sync.RWMutex
}

// New creates an LRU of the given size.
func New(size int) (*Cache, error) {
	return NewWithEvict(size, nil)
}

// NewWithEvict constructs a fixed size cache with the given eviction
// callback.
func NewWithEvict(size int, onEvicted func(key interface{}, value interface{})) (*Cache, error) {
	lru, err := simplelru.NewLRU(size, simplelru.EvictCallback(onEvicted))
	if err != nil {
		return nil, err
	}
	c := &Cache{
		lru: lru,
	}
	return c, nil
}


// NewWithMetric creates an LRU with metric of the given size.
func NewWithMetric(size int) (*CacheWithMetric, error) {
	return NewWithMetricAndEvict(size, nil)
}


// NewWithMetricAndEvict constructs a fixed size cache with metric and the given eviction
// callback.
func NewWithMetricAndEvict(size int, onEvicted func(key interface{}, value interface{})) (*CacheWithMetric, error) {
	c, err := NewWithEvict(size, onEvicted)
	if err != nil {
		return nil, err
	}

	return &CacheWithMetric{
		Cache: c,
	}, nil
}


// Purge is used to completely clear the cache.
func (c *Cache) Purge() {
	c.lock.Lock()
	c.lru.Purge()
	c.lock.Unlock()
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *Cache) Add(key, value interface{}) (evicted bool) {
	c.lock.Lock()
	evicted = c.lru.Add(key, value)
	c.lock.Unlock()
	return evicted
}

// Get looks up a key's value from the cache.
func (c *Cache) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	value, ok = c.lru.Get(key)
	c.lock.Unlock()
	return value, ok
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or deleting it for being stale.
func (c *Cache) Contains(key interface{}) bool {
	c.lock.RLock()
	containKey := c.lru.Contains(key)
	c.lock.RUnlock()
	return containKey
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key.
func (c *Cache) Peek(key interface{}) (value interface{}, ok bool) {
	c.lock.RLock()
	value, ok = c.lru.Peek(key)
	c.lock.RUnlock()
	return value, ok
}

// ContainsOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *Cache) ContainsOrAdd(key, value interface{}) (ok, evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.lru.Contains(key) {
		return true, false
	}
	evicted = c.lru.Add(key, value)
	return false, evicted
}

// PeekOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *Cache) PeekOrAdd(key, value interface{}) (previous interface{}, ok, evicted bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	previous, ok = c.lru.Peek(key)
	if ok {
		return previous, true, false
	}

	evicted = c.lru.Add(key, value)
	return nil, false, evicted
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key interface{}) (present bool) {
	c.lock.Lock()
	present = c.lru.Remove(key)
	c.lock.Unlock()
	return
}

// Resize changes the cache size.
func (c *Cache) Resize(size int) (evicted int) {
	c.lock.Lock()
	evicted = c.lru.Resize(size)
	c.lock.Unlock()
	return evicted
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() (key, value interface{}, ok bool) {
	c.lock.Lock()
	key, value, ok = c.lru.RemoveOldest()
	c.lock.Unlock()
	return
}

// GetOldest returns the oldest entry
func (c *Cache) GetOldest() (key, value interface{}, ok bool) {
	c.lock.Lock()
	key, value, ok = c.lru.GetOldest()
	c.lock.Unlock()
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *Cache) Keys() []interface{} {
	c.lock.RLock()
	keys := c.lru.Keys()
	c.lock.RUnlock()
	return keys
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.lock.RLock()
	length := c.lru.Len()
	c.lock.RUnlock()
	return length
}

// CacheWithMetric is a thread-safe fixed size LRU cache with metric.
type CacheWithMetric struct {
	*Cache
	hitCount uint64
	missCount uint64
}

// Get looks up a key's value from the cache.
func (c *CacheWithMetric) Get(key interface{}) (value interface{}, ok bool) {
	c.lock.Lock()
	value, ok = c.lru.Get(key)

	if ok {
		atomic.AddUint64(&c.hitCount, 1)
	} else {
		atomic.AddUint64(&c.missCount, 1)
	}

	c.lock.Unlock()
	return value, ok
}

// HitCount returns hit count
func (c *CacheWithMetric) HitCount() uint64 {
	return atomic.LoadUint64(&c.hitCount)
}

// MissCount returns miss count
func (c *CacheWithMetric) MissCount() uint64 {
	return atomic.LoadUint64(&c.missCount)
}

// LookupCount returns lookup count
func (c *CacheWithMetric) LookupCount() uint64 {
	return c.HitCount() + c.MissCount()
}

// HitRate returns hit rate
func (c *CacheWithMetric) HitRate() float64 {
	hc, mc := c.HitCount(), c.MissCount()
	total := hc + mc
	if total == 0 {
		return 0.0
	}
	return float64(hc) / float64(total)
}
