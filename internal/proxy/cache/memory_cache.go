package proxycache

import (
	"math/rand"
	"sync"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"
)

// GetCurrentTime allows overriding time in tests.
type GetCurrentTime func() time.Time

// MemoryCacheEntry stores cached values along with bookkeeping metadata.
type MemoryCacheEntry struct {
	lastAccessedAt time.Time
	expiresAt      time.Time
	value          []byte
}

// MemoryCacheEntryMap simplifies map declarations.
type MemoryCacheEntryMap map[CacheKey]*MemoryCacheEntry

// MemoryCacheKeyList tracks cache keys in insertion order.
type MemoryCacheKeyList []CacheKey

// MemoryCache provides an in-memory bounded cache with random sampling eviction.
type MemoryCache struct {
	sync.Mutex
	capacity       int
	maxItemSize    int
	size           int
	keys           MemoryCacheKeyList
	items          MemoryCacheEntryMap
	getCurrentTime GetCurrentTime
}

// NewMemoryCache constructs a memory cache bounded by capacity and per-item size.
func NewMemoryCache(capacity, maxItemSize int) *MemoryCache {
	return &MemoryCache{
		capacity:       capacity,
		maxItemSize:    maxItemSize,
		size:           0,
		keys:           MemoryCacheKeyList{},
		items:          MemoryCacheEntryMap{},
		getCurrentTime: time.Now,
	}
}

// Set stores a value if it fits per-item limits, evicting older entries if required.
func (c *MemoryCache) Set(key CacheKey, value []byte, expiresAt time.Time) {
	c.Lock()
	defer c.Unlock()

	itemSize := len(value)
	if itemSize > c.maxItemSize || itemSize > c.capacity {
		logger.Debug("proxy cache: item too large", logger.Int("item_size", itemSize), logger.Int("max_item_size", c.maxItemSize), logger.Int("capacity", c.capacity))
		return
	}

	limit := c.capacity - itemSize
	for c.size > limit && len(c.keys) > 0 {
		logger.Debug("proxy cache: evicting to make space", logger.Int("current_size", c.size), logger.Int("limit", limit))
		c.evictOldestItem()
	}

	if existingItem, ok := c.items[key]; ok {
		c.size -= len(existingItem.value)
	} else {
		c.keys = append(c.keys, key)
	}

	c.items[key] = &MemoryCacheEntry{
		lastAccessedAt: c.getCurrentTime(),
		expiresAt:      expiresAt,
		value:          value,
	}

	c.size += itemSize

	logger.Debug("proxy cache: item stored", logger.Any("key", key), logger.Int("size", itemSize), logger.Time("expires_at", expiresAt), logger.Int("current_size", c.size))
}

// Get retrieves a stored item when present and not expired.
func (c *MemoryCache) Get(key CacheKey) ([]byte, bool) {
	c.Lock()
	defer c.Unlock()

	now := c.getCurrentTime()

	item, ok := c.items[key]
	if !ok || item.expiresAt.Before(now) {
		return nil, false
	}

	item.lastAccessedAt = now
	return item.value, true
}

func (c *MemoryCache) evictOldestItem() {
	if len(c.keys) == 0 {
		return
	}

	var oldestKey CacheKey
	var oldestIndex int
	var oldest time.Time

	now := c.getCurrentTime()

	// Pick random items and evict the oldest among the sample. Prioritise expired items.
	for i := 0; i < 5 && len(c.keys) > 0; i++ {
		index := rand.Intn(len(c.keys))
		key := c.keys[index]
		entry := c.items[key]

		if entry.expiresAt.Before(now) {
			oldestKey = key
			oldestIndex = index
			break
		}

		if entry.lastAccessedAt.Before(oldest) || oldest.IsZero() {
			oldest = entry.lastAccessedAt
			oldestKey = key
			oldestIndex = index
		}
	}

	c.keys[oldestIndex] = c.keys[len(c.keys)-1]
	c.keys = c.keys[:len(c.keys)-1]

	c.size -= len(c.items[oldestKey].value)
	delete(c.items, oldestKey)
}
