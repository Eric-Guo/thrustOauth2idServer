package proxycache

import (
	"net/http"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"
)

// CacheKey uniquely identifies a cached response for a request variant.
type CacheKey uint64

// Cache describes the storage backend used by the handler.
type Cache interface {
	Get(key CacheKey) ([]byte, bool)
	Set(key CacheKey, value []byte, expiresAt time.Time)
}

// CacheHandler intercepts responses to add caching semantics around the next handler.
type CacheHandler struct {
	cache       Cache
	next        http.Handler
	maxBodySize int
}

// NewCacheHandler constructs a caching handler in front of the provided next handler.
func NewCacheHandler(cache Cache, maxBodySize int, next http.Handler) *CacheHandler {
	return &CacheHandler{
		cache:       cache,
		next:        next,
		maxBodySize: maxBodySize,
	}
}

// ServeHTTP attempts to serve a cached response, falling back to the next handler.
func (h *CacheHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	variant := NewVariant(r)
	response, key, found := h.fetchFromCache(r, variant)

	if found {
		variant.SetResponseHeader(response.HttpHeader)
		if !variant.Matches(response.VariantHeader) {
			response, key, found = h.fetchFromCache(r, variant)
		}
	}

	if found {
		response.WriteCachedResponse(w, r)
		return
	}

	if !h.shouldCacheRequest(r) {
		logger.Debug("proxy cache: bypassing request", logger.String("path", r.URL.Path), logger.String("method", r.Method))
		w.Header().Set("X-Cache", "bypass")
		h.next.ServeHTTP(w, r)
		return
	}

	cr := NewCacheableResponse(w, h.maxBodySize)
	h.next.ServeHTTP(cr, r)

	cacheable, expires := cr.CacheStatus()
	if cacheable {
		variant.SetResponseHeader(cr.HttpHeader)
		cr.VariantHeader = variant.VariantHeader()

		encoded, err := cr.ToBuffer()
		if err != nil {
			logger.Error("proxy cache: encode response failed", logger.String("path", r.URL.Path), logger.Err(err))
		} else {
			h.cache.Set(key, encoded, expires)
			logger.Debug("proxy cache: stored response", logger.String("path", r.URL.Path), logger.Any("key", key), logger.Time("expires", expires), logger.Int("size", len(encoded)))
		}
	}
}

// Private

func (h *CacheHandler) fetchFromCache(r *http.Request, variant *Variant) (CacheableResponse, CacheKey, bool) {
	key := variant.CacheKey()
	cached, found := h.cache.Get(key)

	if found {
		response, err := CacheableResponseFromBuffer(cached)
		if err != nil {
			logger.Error("proxy cache: decode cached response failed", logger.String("path", r.URL.Path), logger.Err(err))
			return CacheableResponse{}, key, false
		}

		return response, key, true
	}

	return CacheableResponse{}, key, false
}

func (h *CacheHandler) shouldCacheRequest(r *http.Request) bool {
	allowedMethod := r.Method == http.MethodGet || r.Method == http.MethodHead
	isUpgrade := r.Header.Get("Connection") == "Upgrade" || r.Header.Get("Upgrade") == "websocket"
	isRange := r.Header.Get("Range") != ""

	return allowedMethod && !isUpgrade && !isRange
}
