package proxy

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/go-dev-frame/sponge/pkg/logger"
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
		Transport:    createProxyTransport(opts.UnixSocketPath),
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

func createProxyTransport(unixSocketRaw string) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableCompression = true
	socketPath := normalizeUnixSocketPath(unixSocketRaw)
	if socketPath != "" {
		// Route all outbound requests to the given UNIX socket, ignoring TCP host:port.
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		}
		// HTTP/2 over unix domain sockets is unusual; keep defaults which already
		// negotiate HTTP/1.1 for non-TLS transports.
	}
	return transport
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
