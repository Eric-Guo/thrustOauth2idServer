package routers

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/proxykit"
	"golang.org/x/net/http2"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/proxycache"
	"thrust_oauth2id/internal/proxysendfile"
)

func registerProxyKit(r *gin.Engine) {
	cfg := config.Get()
	proxyCfg := cfg.Proxy
	if !proxyCfg.Enabled {
		return
	}

	targets, unixSocketPath := resolveProxyTargets(cfg)
	if len(targets) == 0 {
		logger.Fatal(
			"proxy target url not configured",
			logger.Bool("proxy_enabled", proxyCfg.Enabled),
			logger.Bool("upstream_enabled", cfg.Upstream.Enabled),
		)
		return
	}

	manager := proxykit.NewRouteManager()
	backends, err := proxykit.ParseBackends("/", targets)
	if err != nil {
		logger.Fatal("invalid proxy target url", logger.Err(err))
		return
	}

	errorHandler := newProxyErrorHandler(proxyCfg.BadGatewayPage)
	for _, backend := range backends {
		rp := backend.ReverseProxy()
		rp.ErrorHandler = errorHandler
		rp.Transport = createProxyTransport(backend.URL, unixSocketPath, proxyCfg.H2cEnabled)
		applyForwardHeaderPolicy(rp, proxyCfg.ForwardHeaders)
	}

	proxykit.StartHealthChecks(backends, buildHealthCheckConfig(proxyCfg.HealthCheck))

	balancer := selectProxyBalancer(proxyCfg.Strategy, backends)
	route, err := manager.AddRoute("/", balancer)
	if err != nil {
		logger.Fatal("failed to add proxy route", logger.String("prefix", "/"), logger.Err(err))
		return
	}

	handler := http.Handler(route.Proxy)
	handler = wrapWithCache(proxyCfg.Cache, handler)
	handler = proxysendfile.NewSendfileHandler(proxyCfg.XSendfileEnabled, handler)

	ginHandler := func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}

	r.NoRoute(ginHandler)
	r.NoMethod(ginHandler)

	if proxyCfg.Management.Enabled {
		registerProxyManagementRoutes(r, proxyCfg.Management.BasePath, manager)
	}

	loggerProxyKitInfo(proxyCfg, targets, unixSocketPath)
}

func loggerProxyKitInfo(proxyCfg config.Proxy, targets []string, unixSocketPath string) {
	logFields := []logger.Field{
		logger.String("strategy", strings.ToLower(proxyCfg.Strategy)),
		logger.String("targets", strings.Join(targets, ",")),
		logger.Bool("forward_headers", proxyCfg.ForwardHeaders),
		logger.String("bad_gateway_page", proxyCfg.BadGatewayPage),
		logger.Bool("h2c_enabled", proxyCfg.H2cEnabled),
		logger.Bool("cache_enabled", proxyCfg.Cache.Enabled),
		logger.Bool("x_sendfile_enabled", proxyCfg.XSendfileEnabled),
	}
	if unixSocketPath != "" {
		logFields = append(logFields, logger.String("unix_socket", unixSocketPath))
	}
	if proxyCfg.Management.Enabled {
		logFields = append(logFields,
			logger.String("management_base_path", normalizeManagementBasePath(proxyCfg.Management.BasePath)),
		)
	}

	logger.Info("proxykit reverse proxy enabled", logFields...)
}

func resolveProxyTargets(cfg *config.Config) ([]string, string) {
	var targets []string
	if cfg.Proxy.TargetURL != "" {
		targets = append(targets, cfg.Proxy.TargetURL)
	}

	derivedFromSocket := false
	if len(targets) == 0 {
		if cfg.Upstream.TargetBindSocket != "" {
			targets = append(targets, "http://localhost")
			derivedFromSocket = true
		} else if cfg.Upstream.Enabled {
			port := cfg.Upstream.TargetPort
			if port == 0 {
				port = 3000
			}
			targets = append(targets, fmt.Sprintf("http://127.0.0.1:%d", port))
		}
	}

	targets = normalizeTargets(targets)

	var unixSocketPath string
	if cfg.Upstream.TargetBindSocket != "" && (cfg.Upstream.Enabled || derivedFromSocket) {
		unixSocketPath = cfg.Upstream.TargetBindSocket
	}
	return targets, normalizeUnixSocketPath(unixSocketPath)
}

func normalizeTargets(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, target := range in {
		t := strings.TrimSpace(target)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func newProxyErrorHandler(badGatewayPage string) func(http.ResponseWriter, *http.Request, error) {
	var content []byte
	if badGatewayPage != "" {
		data, err := os.ReadFile(badGatewayPage)
		if err != nil {
			logger.Debug("no custom 502 page found", logger.String("path", badGatewayPage))
		} else {
			content = data
		}
	}

	return func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Info("unable to proxy request", logger.String("path", r.URL.Path), logger.Err(err))

		if isRequestEntityTooLarge(err) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		if len(content) > 0 {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write(content)
			return
		}

		w.WriteHeader(http.StatusBadGateway)
	}
}

func isRequestEntityTooLarge(err error) bool {
	var maxBytesError *http.MaxBytesError
	return errors.As(err, &maxBytesError)
}

func createProxyTransport(target *url.URL, unixSocketPath string, h2cEnabled bool) http.RoundTripper {
	if unixSocketPath != "" {
		base := http.DefaultTransport.(*http.Transport).Clone()
		base.DisableCompression = true
		base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", unixSocketPath)
		}
		return base
	}

	if h2cEnabled && target != nil && target.Scheme == "http" {
		return &http2.Transport{
			AllowHTTP:          true,
			DisableCompression: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, addr)
			},
		}
	}

	base := http.DefaultTransport.(*http.Transport).Clone()
	base.DisableCompression = true
	return base
}

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
	}
	return s
}

func applyForwardHeaderPolicy(rp *httputil.ReverseProxy, forwardHeaders bool) {
	if rp == nil || !forwardHeaders {
		return
	}

	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		incomingForwardedFor := append([]string(nil), req.Header["X-Forwarded-For"]...)
		incomingForwardedHost := req.Header.Get("X-Forwarded-Host")
		incomingForwardedProto := req.Header.Get("X-Forwarded-Proto")

		if len(incomingForwardedFor) > 0 {
			req.Header["X-Forwarded-For"] = incomingForwardedFor
		}

		originalDirector(req)

		if incomingForwardedHost != "" {
			req.Header.Set("X-Forwarded-Host", incomingForwardedHost)
		}
		if incomingForwardedProto != "" {
			req.Header.Set("X-Forwarded-Proto", incomingForwardedProto)
		}
	}
}

func buildHealthCheckConfig(cfg config.HealthCheck) proxykit.HealthCheckConfig {
	return proxykit.HealthCheckConfig{
		Interval: time.Duration(cfg.IntervalSeconds) * time.Second,
		Timeout:  time.Duration(cfg.TimeoutSeconds) * time.Second,
	}
}

func selectProxyBalancer(strategy string, backends []*proxykit.Backend) proxykit.Balancer {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "least_connections":
		return proxykit.NewLeastConnections(backends)
	case "ip_hash":
		return proxykit.NewIPHash(backends)
	default:
		return proxykit.NewRoundRobin(backends)
	}
}

func wrapWithCache(cfg config.Cache, next http.Handler) http.Handler {
	if !cfg.Enabled {
		return next
	}

	capacity := cfg.CapacityBytes
	maxItemSize := cfg.MaxItemSizeBytes
	maxBodySize := cfg.MaxResponseBodyBytes
	if maxBodySize <= 0 {
		maxBodySize = maxItemSize
	}

	if capacity > 0 && maxItemSize > 0 && maxBodySize > 0 {
		cache := proxycache.NewMemoryCache(capacity, maxItemSize)
		logger.Info(
			"reverse proxy cache enabled",
			logger.Int("capacity_bytes", capacity),
			logger.Int("max_item_size_bytes", maxItemSize),
			logger.Int("max_body_size_bytes", maxBodySize),
		)
		return proxycache.NewCacheHandler(cache, maxBodySize, next)
	}

	logger.Warn(
		"reverse proxy cache disabled due to invalid configuration",
		logger.Int("capacity_bytes", capacity),
		logger.Int("max_item_size_bytes", maxItemSize),
		logger.Int("max_body_size_bytes", maxBodySize),
	)
	return next
}

func registerProxyManagementRoutes(r *gin.Engine, basePath string, manager *proxykit.RouteManager) {
	path := normalizeManagementBasePath(basePath)
	group := r.Group(path)
	group.POST("/endpoints/add", gin.WrapF(manager.HandleAddBackends))
	group.POST("/endpoints/remove", gin.WrapF(manager.HandleRemoveBackends))
	group.GET("/endpoints/list", gin.WrapF(manager.HandleListBackends))
	group.GET("/endpoints", gin.WrapF(manager.HandleGetBackend))
}

func normalizeManagementBasePath(in string) string {
	p := strings.TrimSpace(in)
	if p == "" {
		return "/proxykit"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}
