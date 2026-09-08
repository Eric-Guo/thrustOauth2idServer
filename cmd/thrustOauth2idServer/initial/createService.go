package initial

import (
	"net/url"
	"strconv"

	"github.com/go-dev-frame/sponge/pkg/app"
	"github.com/go-dev-frame/sponge/pkg/logger"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/server"
)

// CreateServices create http service
func CreateServices() []app.IServer {
	var cfg = config.Get()
	var servers []app.IServer

	// create a http service
	httpServer := server.NewHTTPServer(":"+strconv.Itoa(cfg.HTTP.Port),
		server.WithHTTPIsProd(cfg.App.Env == "prod"),
	)
	servers = append(servers, httpServer)

	if cfg.Upstream.Enabled {
		if cfg.Upstream.Command == "" {
			logger.Fatal("upstream enabled but command not configured")
		}

		// If a unix socket is configured, do not derive or set TargetPort to avoid conflicts.
		if cfg.Upstream.TargetBindSocket == "" && cfg.Upstream.TargetPort == 0 {
			cfg.Upstream.TargetPort = deriveTargetPort(cfg.Proxy.TargetURL)
		}

		servers = append(servers, app.NewUpstreamServer(app.UpstreamConfig{
			Enabled: cfg.Upstream.Enabled, Command: cfg.Upstream.Command, Args: cfg.Upstream.Args,
			WorkingDirectory: cfg.Upstream.WorkingDirectory, TargetPort: cfg.Upstream.TargetPort,
			TargetBindSocket: cfg.Upstream.TargetBindSocket, StopSignal: cfg.Upstream.StopSignal, Env: cfg.Upstream.Env,
		}))
	}

	return servers
}

func deriveTargetPort(rawURL string) int {
	if rawURL != "" {
		u, err := url.Parse(rawURL)
		if err == nil {
			if portStr := u.Port(); portStr != "" {
				if port, convErr := strconv.Atoi(portStr); convErr == nil {
					return port
				}
			}
		}
	}

	return 3000
}
