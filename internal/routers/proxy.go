package routers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/go-dev-frame/sponge/pkg/logger"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/proxy"
	proxcache "thrust_oauth2id/internal/proxy/cache"
)

func registerReverseProxy(r *gin.Engine) {
	cfg := config.Get()
	proxyCfg := cfg.Proxy
	if !proxyCfg.Enabled {
		return
	}

	targetURLStr := proxyCfg.TargetURL
	if targetURLStr == "" && cfg.Upstream.Enabled {
		if cfg.Upstream.TargetBindSocket != "" {
			// When a UNIX socket is configured, we still need a valid HTTP URL
			// for request rewriting; the transport will dial the socket.
			targetURLStr = "http://localhost"
		} else {
			port := cfg.Upstream.TargetPort
			if port == 0 {
				port = 3000
			}
			targetURLStr = fmt.Sprintf("http://127.0.0.1:%d", port)
		}
	}

	targetURL, err := url.Parse(targetURLStr)
	if err != nil {
		targetErr := err
		logger.Fatal("invalid proxy target url", logger.String("target", targetURLStr), logger.Err(targetErr))
		return
	}

	// Prefer dialing via UNIX socket when the upstream advertises one. This avoids
	// TCP self-loops when the HTTP server and upstream share a port.
	var unixSocketPath string
	if cfg.Upstream.Enabled && cfg.Upstream.TargetBindSocket != "" {
		unixSocketPath = cfg.Upstream.TargetBindSocket
	}

	reverseProxy := proxy.NewReverseProxy(proxy.Options{
		TargetURL:      targetURL,
		BadGatewayPage: proxyCfg.BadGatewayPage,
		ForwardHeaders: proxyCfg.ForwardHeaders,
		UnixSocketPath: unixSocketPath,
	})

	loggerFields := []logger.Field{
		logger.String("target", targetURL.String()),
		logger.Bool("forward_headers", proxyCfg.ForwardHeaders),
		logger.String("bad_gateway_page", proxyCfg.BadGatewayPage),
	}
	if unixSocketPath != "" {
		loggerFields = append(loggerFields, logger.String("unix_socket", unixSocketPath))
	}

	logger.Info("reverse proxy enabled", loggerFields...)

	var handler http.Handler = reverseProxy

	if proxyCfg.Cache.Enabled {
		capacity := proxyCfg.Cache.CapacityBytes
		maxItemSize := proxyCfg.Cache.MaxItemSizeBytes
		maxBodySize := proxyCfg.Cache.MaxResponseBodyBytes
		if maxBodySize <= 0 {
			maxBodySize = maxItemSize
		}

		if capacity > 0 && maxItemSize > 0 && maxBodySize > 0 {
			cache := proxcache.NewMemoryCache(capacity, maxItemSize)
			handler = proxcache.NewCacheHandler(cache, maxBodySize, handler)
			logger.Info(
				"reverse proxy cache enabled",
				logger.Int("capacity_bytes", capacity),
				logger.Int("max_item_size_bytes", maxItemSize),
				logger.Int("max_body_size_bytes", maxBodySize),
			)
		} else {
			logger.Warn(
				"reverse proxy cache disabled due to invalid configuration",
				logger.Int("capacity_bytes", capacity),
				logger.Int("max_item_size_bytes", maxItemSize),
				logger.Int("max_body_size_bytes", maxBodySize),
			)
		}
	}

	handler = proxy.NewSendfileHandler(proxyCfg.XSendfileEnabled, handler)
	if proxyCfg.XSendfileEnabled {
		logger.Info("reverse proxy x-sendfile enabled")
	}

	ginHandler := func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}

	r.NoRoute(ginHandler)
	r.NoMethod(ginHandler)
}
