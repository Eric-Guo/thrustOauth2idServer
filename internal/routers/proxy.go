package routers

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/go-dev-frame/sponge/pkg/logger"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/proxy"
	proxcache "thrust_oauth2id/internal/proxy/cache"
)

func registerReverseProxy(r *gin.Engine) {
	proxyCfg := config.Get().Proxy
	if !proxyCfg.Enabled {
		return
	}

	targetURL, err := url.Parse(proxyCfg.TargetURL)
	if err != nil {
		targetErr := err
		logger.Fatal("invalid proxy target url", logger.String("target", proxyCfg.TargetURL), logger.Err(targetErr))
		return
	}

	reverseProxy := proxy.NewReverseProxy(proxy.Options{
		TargetURL:      targetURL,
		BadGatewayPage: proxyCfg.BadGatewayPage,
		ForwardHeaders: proxyCfg.ForwardHeaders,
	})

	logger.Info("reverse proxy enabled", logger.String("target", targetURL.String()), logger.Bool("forward_headers", proxyCfg.ForwardHeaders), logger.String("bad_gateway_page", proxyCfg.BadGatewayPage))

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
