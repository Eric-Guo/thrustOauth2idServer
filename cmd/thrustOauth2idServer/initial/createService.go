package initial

import (
	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/server"

	"github.com/go-dev-frame/sponge/pkg/app"
)

// CreateServices create http service
func CreateServices() []app.IServer {
	var cfg = config.Get()
	var servers []app.IServer

	// create a http service
	httpServer := server.NewHTTPServer(cfg.HTTP,
		server.WithHTTPIsProd(cfg.App.Env == "prod"),
	)
	servers = append(servers, httpServer)

	return servers
}
