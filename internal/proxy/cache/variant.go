package proxycache

import (
	"hash/fnv"
	"net/http"
	"slices"
	"strings"
)

// Variant captures the request characteristics that affect cacheability.
type Variant struct {
	r           *http.Request
	headerNames []string
}

// NewVariant builds a variant tracker for the provided request.
func NewVariant(r *http.Request) *Variant {
	return &Variant{r: r}
}

// SetResponseHeader inspects the response headers to learn about the Vary configuration.
func (v *Variant) SetResponseHeader(header http.Header) {
	v.headerNames = v.parseVaryHeader(header)
}

// CacheKey computes the stable cache key for the request variant.
func (v *Variant) CacheKey() CacheKey {
	hash := fnv.New64()
	hash.Write([]byte(v.r.Method))
	hash.Write([]byte(v.r.URL.Path))
	hash.Write([]byte(v.r.URL.Query().Encode()))
	hash.Write([]byte(v.r.Host))

	for _, name := range v.headerNames {
		hash.Write([]byte(name + "=" + v.r.Header.Get(name)))
	}

	return CacheKey(hash.Sum64())
}

// Matches verifies whether the response headers align with the request variant.
func (v *Variant) Matches(responseHeader http.Header) bool {
	for _, name := range v.headerNames {
		if responseHeader.Get(name) != v.r.Header.Get(name) {
			return false
		}
	}
	return true
}

// VariantHeader returns the headers that make this variant unique.
func (v *Variant) VariantHeader() http.Header {
	requestHeader := http.Header{}
	for _, name := range v.headerNames {
		requestHeader.Set(name, v.r.Header.Get(name))
	}
	return requestHeader
}

// Private

func (v *Variant) parseVaryHeader(responseHeader http.Header) []string {
	list := responseHeader.Get("Vary")
	if list == "" {
		return []string{}
	}

	names := strings.Split(list, ",")
	for i, name := range names {
		names[i] = http.CanonicalHeaderKey(strings.TrimSpace(name))
	}
	slices.Sort(names)

	return names
}
