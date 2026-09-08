package server

import (
	"net/http"

	"thrust_oauth2id/internal/config"
)

// HTTPOption setting up http
type HTTPOption func(*httpOptions)

type httpOptions struct {
	handler http.Handler
	isProd  bool
	tls     *config.TLS
}

func defaultHTTPOptions() *httpOptions {
	return &httpOptions{
		isProd: false,
	}
}

func (o *httpOptions) apply(opts ...HTTPOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// WithHTTPIsProd setting up production environment markers
func WithHTTPIsProd(isProd bool) HTTPOption {
	return func(o *httpOptions) {
		o.isProd = isProd
	}
}

// WithHTTPTLS setting up tls
func WithHTTPTLS(tls config.TLS) HTTPOption {
	return func(o *httpOptions) {
		o.tls = &tls
	}
}

// WithHTTPHandler supplies the application handler.
func WithHTTPHandler(handler http.Handler) HTTPOption {
	return func(o *httpOptions) { o.handler = handler }
}
