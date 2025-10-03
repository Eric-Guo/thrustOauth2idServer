package routers

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/go-dev-frame/sponge/pkg/logger"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/proxy"
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

	ginHandler := func(c *gin.Context) {
		reverseProxy.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}

	r.NoRoute(ginHandler)
	r.NoMethod(ginHandler)
}
