package proxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/go-dev-frame/sponge/pkg/logger"
)

// Options configures how the reverse proxy behaves.
type Options struct {
	TargetURL      *url.URL
	BadGatewayPage string
	ForwardHeaders bool
}

// NewReverseProxy builds an httputil.ReverseProxy configured similar to the
// upstream thruster implementation.
func NewReverseProxy(opts Options) *httputil.ReverseProxy {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(opts.TargetURL)
			setXForwarded(r, opts.ForwardHeaders)
		},
		ErrorHandler: proxyErrorHandler(opts.BadGatewayPage),
		Transport:    createProxyTransport(),
	}

	return proxy
}

func proxyErrorHandler(badGatewayPage string) func(http.ResponseWriter, *http.Request, error) {
	content, err := os.ReadFile(badGatewayPage)
	if err != nil {
		logger.Debug("no custom 502 page found", logger.String("path", badGatewayPage))
		content = nil
	}

	return func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Info("unable to proxy request", logger.String("path", r.URL.Path), logger.Err(err))

		if isRequestEntityTooLarge(err) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		if content != nil {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write(content)
			return
		}

		w.WriteHeader(http.StatusBadGateway)
	}
}

func setXForwarded(r *httputil.ProxyRequest, forwardHeaders bool) {
	if forwardHeaders {
		r.Out.Header["X-Forwarded-For"] = r.In.Header["X-Forwarded-For"]
	}

	r.SetXForwarded()

	if forwardHeaders {
		if value := r.In.Header.Get("X-Forwarded-Host"); value != "" {
			r.Out.Header.Set("X-Forwarded-Host", value)
		}
		if value := r.In.Header.Get("X-Forwarded-Proto"); value != "" {
			r.Out.Header.Set("X-Forwarded-Proto", value)
		}
	}
}

func isRequestEntityTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func createProxyTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableCompression = true
	return transport
}
