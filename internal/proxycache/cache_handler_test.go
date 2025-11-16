package proxycache

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type recordingCacheEntry struct {
	value     []byte
	expiresAt time.Time
}

type recordingCache struct {
	mu      sync.Mutex
	entries map[CacheKey]recordingCacheEntry
}

func newRecordingCache() *recordingCache {
	return &recordingCache{entries: make(map[CacheKey]recordingCacheEntry)}
}

func (c *recordingCache) Get(key CacheKey) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if entry.expiresAt.Before(time.Now()) {
		delete(c.entries, key)
		return nil, false
	}

	valueCopy := append([]byte(nil), entry.value...)
	return valueCopy, true
}

func (c *recordingCache) Set(key CacheKey, value []byte, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = recordingCacheEntry{
		value:     append([]byte(nil), value...),
		expiresAt: expiresAt,
	}
}

func (c *recordingCache) Contains(key CacheKey) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		return false
	}
	if entry.expiresAt.Before(time.Now()) {
		delete(c.entries, key)
		return false
	}
	return true
}

func TestCacheHandlerStoresVariantUnderVariantKey(t *testing.T) {
	cache := newRecordingCache()
	var originHits int

	originHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originHits++
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Content-Encoding", r.Header.Get("Accept-Encoding"))
		_, _ = w.Write([]byte("payload-" + r.Header.Get("Accept-Encoding")))
	})

	cacheHandler := NewCacheHandler(cache, 1024, originHandler)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/resource", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rr := httptest.NewRecorder()
	cacheHandler.ServeHTTP(rr, req)

	assert.Equal(t, 1, originHits)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "miss", rr.Header().Get("X-Cache"))
	assert.Equal(t, "payload-gzip", rr.Body.String())

	baseVariant := NewVariant(req)
	baseKey := baseVariant.CacheKey()

	varyHeader := http.Header{}
	varyHeader.Set("Vary", "Accept-Encoding")
	variantWithVary := NewVariant(req)
	variantWithVary.SetResponseHeader(varyHeader)
	variantKey := variantWithVary.CacheKey()

	assert.NotEqual(t, baseKey, variantKey)
	assert.False(t, cache.Contains(baseKey))
	assert.True(t, cache.Contains(variantKey))

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "http://example.com/resource", nil)
	req2.Header.Set("Accept-Encoding", "gzip")
	cacheHandler.ServeHTTP(rr2, req2)

	assert.Equal(t, 1, originHits, "expected cache hit without re-entering origin handler")
	assert.Equal(t, http.StatusOK, rr2.Code)
	assert.Equal(t, "hit", rr2.Header().Get("X-Cache"))
	assert.Equal(t, "payload-gzip", rr2.Body.String())
}
