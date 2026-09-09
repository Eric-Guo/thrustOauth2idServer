// Package initial is the package that starts the service to initialize the service, including
// the initialization configuration, service configuration, connecting to the database, and
// resource release needed when shutting down the service.
package initial

import (
	"strconv"
	"time"

	ginAuth "github.com/go-dev-frame/sponge/pkg/gin/middleware/auth"

	"github.com/go-dev-frame/sponge/pkg/logger"
	"github.com/go-dev-frame/sponge/pkg/stat"
	"github.com/go-dev-frame/sponge/pkg/tracer"

	"thrust_oauth2id/configs"
	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/database"
)

// InitApp initial app configuration
func InitApp(configFile, version string) {
	initConfig(configFile, version)
	cfg := config.Get()

	// initializing log
	_, err := logger.Init(
		logger.WithLevel(cfg.Logger.Level),
		logger.WithFormat(cfg.Logger.Format),
		logger.WithSave(
			cfg.Logger.IsSave,
			//logger.WithFileName(cfg.Logger.LogFileConfig.Filename),
			//logger.WithFileMaxSize(cfg.Logger.LogFileConfig.MaxSize),
			//logger.WithFileMaxBackups(cfg.Logger.LogFileConfig.MaxBackups),
			//logger.WithFileMaxAge(cfg.Logger.LogFileConfig.MaxAge),
			//logger.WithFileIsCompression(cfg.Logger.LogFileConfig.IsCompression),
		),
	)
	if err != nil {
		panic(err)
	}
	logger.Debug(config.Show())
	logger.Info("[logger] was initialized")

	// initializing tracing
	if cfg.App.EnableTrace {
		tracer.InitWithConfig(
			cfg.App.Name,
			cfg.App.Env,
			cfg.App.Version,
			cfg.Jaeger.AgentHost,
			strconv.Itoa(cfg.Jaeger.AgentPort),
			cfg.App.TracingSamplingRate,
		)
		logger.Info("[tracer] was initialized")
	}

	// initializing the print system and process resources
	if cfg.App.EnableStat {
		stat.Init(
			stat.WithLog(logger.Get()),
			stat.WithAlarm(), // invalid if it is windows, the default threshold for cpu and memory is 0.8, you can modify them
			stat.WithPrintField(logger.String("service_name", cfg.App.Name), logger.String("host", cfg.App.Host)),
		)
		logger.Info("[resource statistics] was initialized")
	}

	// initializing database
	database.InitDB()
	logger.Infof("[%s] was initialized", cfg.Database.Driver)
	database.InitCache(cfg.App.CacheType)
	if cfg.App.CacheType != "" {
		logger.Infof("[%s] was initialized", cfg.App.CacheType)
	}
	if cfg.JWT.SigningKey != "" && cfg.JWT.SigningKey != "change-me" {
		ginAuth.InitAuth([]byte(cfg.JWT.SigningKey), time.Duration(cfg.JWT.Expire)*time.Second)
	}
}

func initConfig(configFile, version string) {
	getConfigFromLocal(configFile)

	if version != "" {
		config.Get().App.Version = version
	}
}

// get configuration from local configuration file
func getConfigFromLocal(configFile string) {
	if configFile == "" {
		configFile = configs.Location("thrustOauth2idServer.yml")
	}
	err := config.Load(configFile)
	if err != nil {
		panic("init config error: " + err.Error())
	}
}
