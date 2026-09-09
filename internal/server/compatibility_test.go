package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/database"
	"thrust_oauth2id/internal/routers"
)

func TestSpongeRoutesAndRailsCompatibility(t *testing.T) {
	var upstreamCalls atomic.Int32
	body := strings.Repeat("Rails response ", 300)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalls.Add(1)
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-Origin-ID", r.Header.Get("X-Request-ID"))
		w.Header().Set("X-Origin-Start", r.Header.Get("X-Request-Start"))
		w.Header().Set("X-Request-ID", "upstream-must-not-replace-id")
		if r.URL.Path == "/asset" {
			w.Header().Set("Cache-Control", "public, max-age=60")
		}
		if r.URL.Path == "/login" {
			w.Header().Set("Set-Cookie", "session=secret")
		}
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(origin.Close)
	cfg := &config.Config{
		App:  config.App{Env: "prod"},
		HTTP: config.HTTP{GzipEnabled: true, GzipJitter: 32, GzipDisableOnAuth: true, AddRequestStartHeader: true},
		Proxy: config.Proxy{Enabled: true, TargetURL: origin.URL, XSendfileEnabled: true,
			Cache: config.Cache{Enabled: true, CapacityBytes: 1 << 20, MaxItemSizeBytes: 1 << 16, MaxResponseBodyBytes: 1 << 16}},
		Database: config.Database{Driver: "sqlite", Sqlite: config.Sqlite{DBFile: filepath.Join(t.TempDir(), "test.sqlite3")}},
	}
	config.Set(cfg)
	database.InitDB()
	t.Cleanup(func() { _ = database.CloseDB() })
	h := newHTTPHandler(routers.NewRouter(), cfg.HTTP)
	doRequest := func(path, cookie string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Accept-Encoding", "gzip")
		r.Header.Set("X-Request-ID", "untrusted")
		r.Header.Set("Connection", "X-Request-ID, X-Request-Start")
		if cookie != "" {
			r.Header.Set("Cookie", cookie)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		require.NotEmpty(t, w.Header().Get("X-Request-ID"))
		require.NotEqual(t, "untrusted", w.Header().Get("X-Request-ID"))
		return w
	}
	first := doRequest("/asset", "")
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, "miss", first.Header().Get("X-Cache"))
	require.Equal(t, first.Header().Get("X-Request-ID"), first.Header().Get("X-Origin-ID"))
	require.NotEmpty(t, first.Header().Get("X-Origin-Start"))
	require.Equal(t, "gzip", first.Header().Get("Content-Encoding"))
	require.NotZero(t, first.Body.Bytes()[3]&0x10)
	reader, err := gzip.NewReader(first.Body)
	require.NoError(t, err)
	decoded, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, body, string(decoded))
	second := doRequest("/asset", "session=secret")
	require.Equal(t, "hit", second.Header().Get("X-Cache"))
	require.Empty(t, second.Header().Get("Content-Encoding"), "guard must also protect cached responses")
	require.Equal(t, body, second.Body.String())
	require.NotEqual(t, first.Header().Get("X-Request-ID"), second.Header().Get("X-Request-ID"))
	require.EqualValues(t, 1, upstreamCalls.Load())
	login := doRequest("/login", "")
	require.Empty(t, login.Header().Get("Content-Encoding"), "Rails Set-Cookie must disable gzip before headers are written")
	require.Equal(t, body, login.Body.String())
	beforeNative := upstreamCalls.Load()
	require.Equal(t, http.StatusOK, doRequest("/health", "").Code)
	apiResponse := doRequest("/api/v1/users/invalid-id", "")
	require.NotEqual(t, body, apiResponse.Body.String())
	require.Equal(t, beforeNative, upstreamCalls.Load(), "registered Sponge APIs must never fall through to Rails")

	// A trusted front proxy's request ID must reach both Rails and the caller.
	cfg.Proxy.ForwardHeaders = true
	h = newHTTPHandler(routers.NewRouter(), cfg.HTTP)
	r := httptest.NewRequest("GET", "/rails", nil)
	r.Header.Set("X-Request-ID", "trusted-front-proxy")
	r.Header.Set("Connection", "X-Request-ID, X-Request-Start")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, "trusted-front-proxy", w.Header().Get("X-Request-ID"))
	require.Equal(t, "trusted-front-proxy", w.Header().Get("X-Origin-ID"))
}
