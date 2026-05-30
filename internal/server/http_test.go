package server

import (
	"net/http"
	"strings"
	"testing"

	"thrust_oauth2id/internal/config"
)

func TestNewServerSelectsSpongeTLSMode(t *testing.T) {
	tests := []struct {
		name       string
		tls        config.TLS
		wantScheme string
	}{
		{
			name:       "http",
			tls:        config.TLS{},
			wantScheme: "http",
		},
		{
			name:       "self-signed",
			tls:        config.TLS{EnableMode: "self-signed"},
			wantScheme: "https",
		},
		{
			name: "encrypt",
			tls: config.TLS{
				EnableMode: "encrypt",
				Domain:     "example.com",
				Email:      "admin@example.com",
			},
			wantScheme: "https",
		},
		{
			name: "external",
			tls: config.TLS{
				EnableMode: "external",
				CertFile:   "cert.pem",
				KeyFile:    "key.pem",
			},
			wantScheme: "https",
		},
		{
			name: "remote-api",
			tls: config.TLS{
				EnableMode: "remote-api",
				RemoteAPI:  config.RemoteAPI{URL: "https://certs.example.com/current"},
			},
			wantScheme: "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newServer(&http.Server{}, config.HTTP{TLS: tt.tls})
			if got := srv.Scheme(); got != tt.wantScheme {
				t.Fatalf("Scheme() = %q, want %q", got, tt.wantScheme)
			}
		})
	}
}

func TestNewHTTPServerAddressUsesHTTPSPortOnlyWhenTLSEnabled(t *testing.T) {
	handler := http.NewServeMux()

	tests := []struct {
		name     string
		cfg      config.HTTP
		wantAddr string
	}{
		{
			name: "http",
			cfg: config.HTTP{
				Port:      8080,
				HTTPSPort: 8443,
			},
			wantAddr: ":8080",
		},
		{
			name: "tls",
			cfg: config.HTTP{
				Port:      8080,
				HTTPSPort: 8443,
				TLS: config.TLS{
					EnableMode: "external",
					CertFile:   "cert.pem",
					KeyFile:    "key.pem",
				},
			},
			wantAddr: ":8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewHTTPServer(tt.cfg, WithHTTPHandler(handler))
			if got := srv.String(); !strings.Contains(got, tt.wantAddr) {
				t.Fatalf("String() = %q, want address %q", got, tt.wantAddr)
			}
		})
	}
}

func TestInvalidTLSModeFailsBeforeListen(t *testing.T) {
	srv := newServer(&http.Server{}, config.HTTP{TLS: config.TLS{EnableMode: "bogus"}})

	err := srv.Run()
	if err == nil {
		t.Fatal("Run() error is nil, want unsupported mode error")
	}
	if !strings.Contains(err.Error(), `unsupported tls enableMode "bogus"`) {
		t.Fatalf("Run() error = %q, want unsupported mode error", err.Error())
	}
}
