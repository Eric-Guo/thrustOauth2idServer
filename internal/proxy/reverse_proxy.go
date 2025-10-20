package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-dev-frame/sponge/pkg/logger"
	"golang.org/x/net/http2"
)

// Options configures how the reverse proxy behaves.
type Options struct {
	TargetURL      *url.URL
	BadGatewayPage string
	ForwardHeaders bool
	// UnixSocketPath, if non-empty, makes the proxy connect to the upstream
	// via a UNIX domain socket instead of TCP. The HTTP request URL is still
	// rewritten to TargetURL for host/scheme, but the actual transport dials
	// the provided socket path using network "unix".
	UnixSocketPath string
	// H2cEnabled enables HTTP/2 cleartext (h2c) when the upstream speaks it.
	H2cEnabled bool
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
		Transport:    createProxyTransport(opts),
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

func createProxyTransport(opts Options) http.RoundTripper {
	// Start from the default transport for sane defaults.
	base := http.DefaultTransport.(*http.Transport).Clone()
	base.DisableCompression = true

	// If a UNIX socket is provided, always prefer it and keep HTTP/1.1 semantics.
	// HTTP/2 over unix sockets is uncommon and not targeted here.
	socketPath := normalizeUnixSocketPath(opts.UnixSocketPath)
	if socketPath != "" {
		base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}
		return base
	}

	// Enable HTTP/2 cleartext (h2c) via prior knowledge when explicitly opted-in
	// and only for non-TLS upstreams.
	if opts.H2cEnabled && opts.TargetURL != nil && opts.TargetURL.Scheme == "http" {
		return &http2.Transport{
			AllowHTTP:          true,
			DisableCompression: true,
			// Prior-knowledge: dial raw TCP and speak HTTP/2 without TLS or upgrade.
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		}
	}

	// Default HTTP/1.1 (with TLS ALPN-driven h2 when applicable).
	return base
}

// normalizeUnixSocketPath accepts common forms and returns a clean absolute
// filesystem path for use with net.Dial("unix", path). Supported inputs:
//   - "/tmp/puma.sock"
//   - "unix:///tmp/puma.sock"
//   - "unix://tmp/puma.sock"
//
// Any other string returns as-is if it already looks like a path.
func normalizeUnixSocketPath(in string) string {
	if in == "" {
		return ""
	}
	s := strings.TrimSpace(in)
	if strings.HasPrefix(s, "unix://") {
		s = strings.TrimPrefix(s, "unix://")
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
		return s
	}
	return s
}
