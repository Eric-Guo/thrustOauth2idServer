package initial

import (
	"context"
	"time"

	"github.com/go-dev-frame/sponge/pkg/app"
	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/tracer"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/database"
)

// Close releasing resources after service exit
func Close(servers []app.IServer) []app.Close {
	var closes []app.Close

	// close server
	// Stop the supervised upstream before draining the HTTP listener, so parent
	// signals reach Rails immediately and its final exit status is available.
	for i := len(servers) - 1; i >= 0; i-- {
		closes = append(closes, servers[i].Stop)
	}

	// close database
	closes = append(closes, func() error {
		return database.CloseDB()
	})

	// close redis
	if config.Get().App.CacheType == "redis" {
		closes = append(closes, func() error {
			return database.CloseRedis()
		})
	}

	// close tracing
	if config.Get().App.EnableTrace {
		closes = append(closes, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			return tracer.Close(ctx)
		})
	}

	// close logger
	closes = append(closes, func() error {
		return logger.Sync()
	})

	return closes
}
