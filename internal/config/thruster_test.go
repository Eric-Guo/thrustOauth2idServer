package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressionConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name, yaml string
		jitter     int
	}{
		{"default", "http:\n  gzipEnabled: true\n", 32},
		{"disabled", "http:\n  gzipJitter: 0\n", 0},
		{"custom", "http:\n  gzipJitter: 64\n", 64},
	} {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), tt.name+".yml")
			require.NoError(t, os.WriteFile(file, []byte(tt.yaml), 0600))
			require.NoError(t, Load(file))
			require.Equal(t, tt.jitter, Get().HTTP.GzipJitter)
		})
	}
}

func TestThrusterEnvironmentPrecedence(t *testing.T) {
	t.Setenv("GZIP_COMPRESSION_JITTER", "64")
	t.Setenv("THRUSTER_GZIP_COMPRESSION_JITTER", "0")
	t.Setenv("GZIP_COMPRESSION_DISABLE_ON_AUTH", "false")
	t.Setenv("THRUSTER_GZIP_COMPRESSION_DISABLE_ON_AUTH", "true")
	t.Setenv("GZIP_COMPRESSION_ENABLED", "1")
	t.Setenv("THRUSTER_FORWARD_HEADERS", "false")
	cfg := &Config{Proxy: Proxy{ForwardHeaders: true}}
	require.NoError(t, applyThrusterEnvironment(cfg))
	require.Zero(t, cfg.HTTP.GzipJitter)
	require.True(t, cfg.HTTP.GzipDisableOnAuth)
	require.True(t, cfg.HTTP.GzipEnabled)
	require.False(t, cfg.Proxy.ForwardHeaders)
	t.Setenv("THRUSTER_GZIP_COMPRESSION_JITTER", "invalid")
	require.Error(t, applyThrusterEnvironment(cfg))
}
