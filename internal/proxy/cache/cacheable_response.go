package proxycache

import (
	"bytes"
	"encoding/gob"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	publicExp  = regexp.MustCompile(`\bpublic\b`)
	noCacheExp = regexp.MustCompile(`\bno-cache\b`)
	sMaxAgeExp = regexp.MustCompile(`\bs-max-age=(\d+)\b`)
	maxAgeExp  = regexp.MustCompile(`\bmax-age=(\d+)\b`)
)

// CacheableResponse captures enough of the downstream response to replay it from cache.
type CacheableResponse struct {
	StatusCode    int
	HttpHeader    http.Header
	Body          []byte
	VariantHeader http.Header

	responseWriter http.ResponseWriter
	stasher        *stashingWriter
	headersWritten bool
}

// NewCacheableResponse wraps the downstream writer, retaining the response in memory up to maxBodyLength.
func NewCacheableResponse(w http.ResponseWriter, maxBodyLength int) *CacheableResponse {
	return &CacheableResponse{
		StatusCode: http.StatusOK,
		HttpHeader: http.Header{},

		responseWriter: w,
		stasher:        NewStashingWriter(maxBodyLength, w),
	}
}

// CacheableResponseFromBuffer decodes a cached payload back into a response structure.
func CacheableResponseFromBuffer(b []byte) (CacheableResponse, error) {
	var cr CacheableResponse
	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&cr)

	return cr, err
}

// ToBuffer serialises all cached response fields for storage.
func (c *CacheableResponse) ToBuffer() ([]byte, error) {
	c.Body = c.stasher.Body()

	headerForStorage := cloneHeader(c.HttpHeader)
	if cacheable, _ := c.CacheStatus(); cacheable {
		headerForStorage.Del("Set-Cookie")
	}

	originalHeader := c.HttpHeader
	c.HttpHeader = headerForStorage
	defer func() {
		c.HttpHeader = originalHeader
	}()

	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	err := encoder.Encode(c)

	return b.Bytes(), err
}

// Header implements http.ResponseWriter.
func (c *CacheableResponse) Header() http.Header {
	return c.HttpHeader
}

// Write implements http.ResponseWriter.
func (c *CacheableResponse) Write(bytes []byte) (int, error) {
	if !c.headersWritten {
		c.WriteHeader(http.StatusOK)
	}
	return c.stasher.Write(bytes)
}

// WriteHeader implements http.ResponseWriter.
func (c *CacheableResponse) WriteHeader(statusCode int) {
	c.StatusCode = statusCode
	c.copyHeaders(c.responseWriter, false, c.StatusCode)
	c.headersWritten = true
}

// Flush implements http.Flusher when the downstream supports it.
func (c *CacheableResponse) Flush() {
	if flusher, ok := c.responseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// CacheStatus reports whether the response qualifies for caching along with cache expiry.
func (c *CacheableResponse) CacheStatus() (bool, time.Time) {
	if c.stasher.Overflowed() {
		return false, time.Time{}
	}

	if c.StatusCode < 200 || c.StatusCode > 399 || c.StatusCode == http.StatusNotModified {
		return false, time.Time{}
	}

	if strings.Contains(c.HttpHeader.Get("Vary"), "*") {
		return false, time.Time{}
	}

	cc := c.HttpHeader.Get("Cache-Control")

	if !publicExp.MatchString(cc) || noCacheExp.MatchString(cc) {
		return false, time.Time{}
	}

	matches := sMaxAgeExp.FindStringSubmatch(cc)
	if len(matches) != 2 {
		matches = maxAgeExp.FindStringSubmatch(cc)
	}
	if len(matches) != 2 {
		return false, time.Time{}
	}

	maxAge, err := strconv.Atoi(matches[1])
	if err != nil || maxAge <= 0 {
		return false, time.Time{}
	}

	return true, time.Now().Add(time.Duration(maxAge) * time.Second)
}

// WriteCachedResponse replays a cached response to the client, respecting conditional headers.
func (c *CacheableResponse) WriteCachedResponse(w http.ResponseWriter, r *http.Request) {
	if c.wasNotModified(r) {
		c.copyHeaders(w, true, http.StatusNotModified)
	} else {
		c.copyHeaders(w, true, c.StatusCode)
		_, _ = io.Copy(w, bytes.NewReader(c.Body))
	}
}

// Private

func (c *CacheableResponse) wasNotModified(r *http.Request) bool {
	responseEtag := c.HttpHeader.Get("Etag")
	if responseEtag == "" {
		return false
	}

	ifNoneMatch := strings.Split(r.Header.Get("If-None-Match"), ",")
	for _, etag := range ifNoneMatch {
		if strings.TrimSpace(etag) == responseEtag {
			return true
		}
	}

	return false
}

func (c *CacheableResponse) copyHeaders(w http.ResponseWriter, wasHit bool, statusCode int) {
	for k, v := range c.HttpHeader {
		w.Header()[k] = v
	}

	if wasHit {
		w.Header().Set("X-Cache", "hit")
	} else {
		w.Header().Set("X-Cache", "miss")
	}

	w.WriteHeader(statusCode)
}

func cloneHeader(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for key, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[key] = copied
	}
	return dst
}

// stashingWriter mirrors output to both the downstream writer and an in-memory buffer when size permits.
type stashingWriter struct {
	limit      int
	dest       io.Writer
	buffer     bytes.Buffer
	overflowed bool
}

// NewStashingWriter constructs a writer that buffers up to limit bytes.
func NewStashingWriter(limit int, dest io.Writer) *stashingWriter {
	return &stashingWriter{
		limit: limit,
		dest:  dest,
	}
}

func (w *stashingWriter) Write(p []byte) (int, error) {
	if w.buffer.Len()+len(p) > w.limit {
		w.overflowed = true
	} else {
		_, _ = w.buffer.Write(p)
	}

	return w.dest.Write(p)
}

func (w *stashingWriter) Body() []byte {
	if w.overflowed {
		return nil
	}
	return w.buffer.Bytes()
}

func (w *stashingWriter) Overflowed() bool {
	return w.overflowed
}
