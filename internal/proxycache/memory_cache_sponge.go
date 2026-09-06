package proxycache

import (
	"fmt"
	"time"

	"github.com/dgraph-io/ristretto"

	spongecache "github.com/go-dev-frame/sponge/pkg/cache"
	"github.com/go-dev-frame/sponge/pkg/logger"
)

// GetCurrentTime allows overriding time in tests.
type GetCurrentTime func() time.Time

// MemoryCache provides a cache implementation backed by sponge's ristretto cache.
type MemoryCache struct {
	client         *ristretto.Cache
	capacity       int
	maxItemSize    int
	getCurrentTime GetCurrentTime
}

// NewMemoryCache constructs a memory cache bounded by capacity and per-item size.
func NewMemoryCache(capacity, maxItemSize int) *MemoryCache {
	opts := []spongecache.Option{}
	if capacity > 0 {
		opts = append(opts, spongecache.WithMaxCost(int64(capacity)))
		if numCounters := deriveNumCounters(capacity, maxItemSize); numCounters > 0 {
			opts = append(opts, spongecache.WithNumCounters(numCounters))
		}
	}

	client := spongecache.InitMemory(opts...)

	return &MemoryCache{
		client:         client,
		capacity:       capacity,
		maxItemSize:    maxItemSize,
		getCurrentTime: time.Now,
	}
}

// Set stores a value if it fits per-item limits, leveraging sponge's cache for eviction.
func (c *MemoryCache) Set(key CacheKey, value []byte, expiresAt time.Time) {
	if c.client == nil {
		return
	}

	itemSize := len(value)
	if itemSize > c.maxItemSize || (c.capacity > 0 && itemSize > c.capacity) {
		logger.Debug(
			"proxy cache: item too large",
			logger.Int("item_size", itemSize),
			logger.Int("max_item_size", c.maxItemSize),
			logger.Int("capacity", c.capacity),
		)
		return
	}

	currentTime := c.getCurrentTime()
	ttl := expiresAt.Sub(currentTime)
	if ttl <= 0 {
		logger.Debug(
			"proxy cache: item already expired",
			logger.Any("key", key),
			logger.Time("expires_at", expiresAt),
		)
		return
	}

	valueCopy := append([]byte(nil), value...)
	ristrettoKey := uint64(key) // ristretto expects built-in numeric types, not custom aliases
	if ok := c.client.SetWithTTL(ristrettoKey, valueCopy, int64(itemSize), ttl); !ok {
		logger.Debug(
			"proxy cache: failed to store item",
			logger.Any("key", key),
			logger.Int("size", itemSize),
		)
		return
	}
	c.client.Wait()

	logger.Debug(
		"proxy cache: item stored",
		logger.Any("key", key),
		logger.Int("size", itemSize),
		logger.Time("expires_at", expiresAt),
	)
}

// Get retrieves a stored item when present and not expired.
func (c *MemoryCache) Get(key CacheKey) ([]byte, bool) {
	if c.client == nil {
		return nil, false
	}

	value, ok := c.client.Get(uint64(key))
	if !ok {
		return nil, false
	}

	data, ok := value.([]byte)
	if !ok {
		logger.Error(
			"proxy cache: unexpected item type",
			logger.Any("key", key),
			logger.String("type", fmt.Sprintf("%T", value)),
		)
		return nil, false
	}

	return data, true
}

// deriveNumCounters sizes ristretto's frequency sketch so metadata overhead scales with the cache capacity.
func deriveNumCounters(capacity, maxItemSize int) int64 {
	if capacity <= 0 {
		return 0
	}

	const (
		minCounters     = 1_000
		maxCounters     = 10_000_000
		minAvgItemBytes = 1 << 10  // assume responses are at least 1KiB on average
		maxAvgItemBytes = 16 << 10 // cap assumed average at 16KiB to avoid undersizing
	)

	avgItemBytes := maxItemSize / 4
	if maxItemSize <= 0 {
		avgItemBytes = minAvgItemBytes
	}
	if avgItemBytes < minAvgItemBytes {
		avgItemBytes = minAvgItemBytes
	}
	if avgItemBytes > maxAvgItemBytes {
		avgItemBytes = maxAvgItemBytes
	}

	estimatedItems := capacity / avgItemBytes
	if estimatedItems <= 0 {
		estimatedItems = 1
	}

	numCounters := int64(estimatedItems * 10)
	if numCounters < minCounters {
		numCounters = minCounters
	}
	if numCounters > maxCounters {
		numCounters = maxCounters
	}

	return numCounters
}
