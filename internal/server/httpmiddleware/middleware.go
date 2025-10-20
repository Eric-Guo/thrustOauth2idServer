package httpmiddleware

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	ginmiddleware "github.com/go-dev-frame/sponge/pkg/gin/middleware"
	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/klauspost/compress/gzhttp"
)

// Options configures the optional HTTP middleware that can wrap the Gin engine.
type Options struct {
	AddRequestStartHeader bool
	GzipEnabled           bool
	LogRequests           bool
	MaxRequestBodyBytes   int
}

// Wrap decorates the provided handler with the optional middleware configured in opts.
func Wrap(handler http.Handler, opts Options) http.Handler {
	if handler == nil {
		return nil
	}

	if opts.AddRequestStartHeader {
		handler = newRequestStartMiddleware(handler)
	}

	if opts.GzipEnabled {
		handler = gzhttp.GzipHandler(handler)
	}

	if opts.MaxRequestBodyBytes > 0 {
		handler = http.MaxBytesHandler(handler, int64(opts.MaxRequestBodyBytes))
	}

	if opts.LogRequests {
		handler = newLoggingMiddleware(handler)
	}

	return handler
}

func newRequestStartMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-Start") == "" {
			timestamp := time.Now().UnixMilli()
			r.Header.Set("X-Request-Start", fmt.Sprintf("t=%d", timestamp))
		}
		next.ServeHTTP(w, r)
	})
}

func newLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		writer := newResponseRecorder(w)
		started := time.Now()

		next.ServeHTTP(writer, r)

		elapsed := time.Since(started)
		reqContentType := r.Header.Get("Content-Type")
		respContentType := writer.Header().Get("Content-Type")
		cache := writer.Header().Get("X-Cache")
		remoteAddr := r.Header.Get("X-Forwarded-For")
		if remoteAddr == "" {
			remoteAddr = r.RemoteAddr
		}
		requestIDHeader := r.Header.Get(ginmiddleware.HeaderXRequestIDKey)

		fields := []logger.Field{
			logger.String("path", r.URL.Path),
			logger.Int("status", writer.Status()),
			logger.Int64("dur", elapsed.Milliseconds()),
			logger.String("method", r.Method),
			logger.Int64("req_content_length", r.ContentLength),
			logger.String("req_content_type", reqContentType),
			logger.Int64("resp_content_length", writer.BytesWritten()),
			logger.String("resp_content_type", respContentType),
			logger.String("remote_addr", remoteAddr),
			logger.String("user_agent", r.Header.Get("User-Agent")),
			logger.String("cache", cache),
			logger.String("query", r.URL.RawQuery),
		}

		if requestIDHeader != "" {
			fields = append(fields, logger.String("request_id", requestIDHeader))
		}

		logger.Info("request", fields...)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
}

// WriteHeader captures the HTTP status code before forwarding it to the underlying writer.
func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write counts the bytes written and proxies the call.
func (r *responseRecorder) Write(p []byte) (int, error) {
	if r.statusCode == 0 {
		r.statusCode = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(p)
	r.bytesWritten += int64(n)
	return n, err
}

// Flush delegates to the underlying writer when supported.
func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack allows websocket upgrades to continue working when supported.
func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not implement http.Hijacker")
	}

	conn, rw, err := hijacker.Hijack()
	if err == nil {
		r.statusCode = http.StatusSwitchingProtocols
	}
	return conn, rw, err
}

// Status returns the captured status code.
func (r *responseRecorder) Status() int {
	if r.statusCode == 0 {
		return http.StatusOK
	}
	return r.statusCode
}

// BytesWritten returns the number of bytes written to the response body.
func (r *responseRecorder) BytesWritten() int64 {
	return r.bytesWritten
}
