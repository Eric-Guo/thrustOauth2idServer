package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/go-dev-frame/sponge/pkg/conf"
)

// Load reads Sponge configuration with Thruster defaults and environment
// overrides. Keep startup policy outside the file rewritten by update-config.
func Load(configFile string) error {
	cfg := &Config{HTTP: HTTP{GzipJitter: 32}}
	if err := conf.Parse(configFile, cfg); err != nil {
		return err
	}
	if err := applyThrusterEnvironment(cfg); err != nil {
		return err
	}
	Set(cfg)
	return nil
}

// Keep the upstream names for the compression and header-trust controls ported
// from Thruster. YAML remains the application's primary Sponge configuration.
func applyThrusterEnvironment(cfg *Config) error {
	for _, setting := range []struct {
		name   string
		target *bool
	}{
		{"GZIP_COMPRESSION_ENABLED", &cfg.HTTP.GzipEnabled},
		{"GZIP_COMPRESSION_DISABLE_ON_AUTH", &cfg.HTTP.GzipDisableOnAuth},
		{"FORWARD_HEADERS", &cfg.Proxy.ForwardHeaders},
	} {
		if raw, ok := thrusterEnv(setting.name); ok {
			value, err := strconv.ParseBool(raw)
			if err != nil {
				return fmt.Errorf("%s: %w", setting.name, err)
			}
			*setting.target = value
		}
	}
	if raw, ok := thrusterEnv("GZIP_COMPRESSION_JITTER"); ok {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("GZIP_COMPRESSION_JITTER: %w", err)
		}
		cfg.HTTP.GzipJitter = value
	}
	return nil
}

func thrusterEnv(name string) (string, bool) {
	if value, ok := os.LookupEnv("THRUSTER_" + name); ok {
		return value, true
	}
	return os.LookupEnv(name)
}
