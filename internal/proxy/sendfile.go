package proxy

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/go-dev-frame/sponge/pkg/logger"
)

// SendfileHandler converts X-Sendfile headers into direct file responses when enabled.
type SendfileHandler struct {
	enabled bool
	next    http.Handler
}

// NewSendfileHandler wraps the provided handler with X-Sendfile support.
func NewSendfileHandler(enabled bool, next http.Handler) *SendfileHandler {
	return &SendfileHandler{enabled: enabled, next: next}
}

// ServeHTTP sets up X-Sendfile translation when enabled before delegating to the next handler.
func (h *SendfileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.enabled {
		r.Header.Set("X-Sendfile-Type", "X-Sendfile")
		w = &sendfileWriter{ResponseWriter: w, request: r}
	} else {
		r.Header.Del("X-Sendfile-Type")
	}

	h.next.ServeHTTP(w, r)
}

type sendfileWriter struct {
	http.ResponseWriter
	request       *http.Request
	headerWritten bool
	sendingFile   bool
}

func (w *sendfileWriter) Write(b []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}

	if w.sendingFile {
		return 0, http.ErrBodyNotAllowed
	}

	return w.ResponseWriter.Write(b)
}

func (w *sendfileWriter) WriteHeader(statusCode int) {
	filename := w.ResponseWriter.Header().Get("X-Sendfile")
	w.ResponseWriter.Header().Del("X-Sendfile")

	w.sendingFile = filename != ""
	w.headerWritten = true

	if w.sendingFile {
		w.serveFile(filename)
	} else {
		w.ResponseWriter.WriteHeader(statusCode)
	}
}

func (w *sendfileWriter) serveFile(filename string) {
	logger.Debug("x-sendfile sending file", logger.String("path", filename))

	w.setContentLength(filename)
	http.ServeFile(w.ResponseWriter, w.request, filename)
}

func (w *sendfileWriter) setContentLength(filename string) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		w.ResponseWriter.Header().Del("Content-Length")
		return
	}

	w.ResponseWriter.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
}

func (w *sendfileWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("ResponseWriter does not implement http.Hijacker")
	}

	return hijacker.Hijack()
}

func (w *sendfileWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *sendfileWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}
