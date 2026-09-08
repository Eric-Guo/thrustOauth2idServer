package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-dev-frame/sponge/pkg/logger"

	"github.com/gin-gonic/gin"

	"github.com/go-dev-frame/sponge/pkg/app"
	"github.com/go-dev-frame/sponge/pkg/httpsrv"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/routers"
)

var _ app.IServer = (*httpServer)(nil)

type httpServer struct {
	addr   string
	server *httpsrv.Server
}

// Start http service
func (s *httpServer) Start() error {
	if err := s.server.Run(); err != nil {
		return fmt.Errorf("run %s service error: %v", s.server.Scheme(), err)
	}
	return nil
}

// Stop http service
func (s *httpServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// String comment
func (s *httpServer) String() string {
	return s.server.Scheme() + " service address is " + s.addr
}

func newServer(server *http.Server, cfg config.HTTP) *httpsrv.Server {
	mode := normalizeTLSMode(cfg.TLS.EnableMode)
	switch httpsrv.Mode(mode) {
	case httpsrv.ModeTLSSelfSigned:
		logger.Info("tls enabled with sponge self-signed mode", logger.String("addr", server.Addr))
		return httpsrv.New(server, newTLSSelfSignedConfig(cfg.TLS))

	case httpsrv.ModeTLSEncrypt:
		logger.Info("tls enabled with sponge encrypt mode", logger.String("addr", server.Addr), logger.Any("domains", tlsDomains(cfg.TLS)))
		return httpsrv.New(server, newTLSEncryptConfig(cfg))

	case httpsrv.ModeTLSExternal:
		logger.Info("tls enabled with sponge external mode", logger.String("addr", server.Addr), logger.String("cert_file", cfg.TLS.CertFile))
		return httpsrv.New(server, httpsrv.NewTLSExternalConfig(strings.TrimSpace(cfg.TLS.CertFile), strings.TrimSpace(cfg.TLS.KeyFile)))

	case httpsrv.ModeRemoteAPI:
		logger.Info("tls enabled with sponge remote-api mode", logger.String("addr", server.Addr), logger.String("url", cfg.TLS.RemoteAPI.URL))
		return httpsrv.New(server, newTLSRemoteAPIConfig(cfg.TLS))

	case "":
		logger.Info("tls disabled", logger.String("addr", server.Addr))
		return httpsrv.New(server)

	default:
		return httpsrv.New(server, invalidTLSMode(mode))
	}
}

func newTLSSelfSignedConfig(tls config.TLS) *httpsrv.TLSSelfSignedConfig {
	var opts []httpsrv.TLSSelfSignedOption
	if cacheDir := strings.TrimSpace(tls.StoragePath); cacheDir != "" {
		opts = append(opts, httpsrv.WithTLSSelfSignedCacheDir(cacheDir))
	}
	return httpsrv.NewTLSSelfSignedConfig(opts...)
}

func newTLSEncryptConfig(cfg config.HTTP) *httpsrv.TLSAutoEncryptConfig {
	var opts []httpsrv.TLSEncryptOption
	domains := tlsDomains(cfg.TLS)
	if cacheDir := strings.TrimSpace(cfg.TLS.StoragePath); cacheDir != "" {
		opts = append(opts, httpsrv.WithTLSEncryptCacheDir(cacheDir))
	}
	if len(domains) > 1 {
		opts = append(opts, httpsrv.WithTLSEncryptDomains(domains[1:]...))
	}
	if cfg.TLS.RedirectHTTP {
		if cfg.Port > 0 {
			opts = append(opts, httpsrv.WithTLSEncryptEnableRedirect(fmt.Sprintf(":%d", cfg.Port)))
		} else {
			opts = append(opts, httpsrv.WithTLSEncryptEnableRedirect())
		}
		opts = append(opts, httpsrv.WithTLSEncryptRedirectHTTPSPort(cfg.HTTPSPort))
	}
	return httpsrv.NewTLSEAutoEncryptConfig(primaryTLSDomain(domains), strings.TrimSpace(cfg.TLS.Email), opts...)
}

func newTLSRemoteAPIConfig(tls config.TLS) *httpsrv.TLSRemoteAPIConfig {
	var opts []httpsrv.TLSRemoteAPIOption
	if len(tls.RemoteAPI.Headers) > 0 {
		opts = append(opts, httpsrv.WithTLSRemoteAPIHeaders(tls.RemoteAPI.Headers))
	}
	if tls.RemoteAPI.Timeout > 0 {
		opts = append(opts, httpsrv.WithTLSRemoteAPITimeout(time.Duration(tls.RemoteAPI.Timeout)*time.Second))
	}
	if cacheDir := strings.TrimSpace(tls.StoragePath); cacheDir != "" {
		opts = append(opts, httpsrv.WithTLSRemoteAPICacheDir(cacheDir))
	}
	return httpsrv.NewTLSRemoteAPIConfig(strings.TrimSpace(tls.RemoteAPI.URL), opts...)
}

func serviceAddr(cfg config.HTTP) string {
	port := cfg.Port
	if normalizeTLSMode(cfg.TLS.EnableMode) != "" && cfg.HTTPSPort > 0 {
		port = cfg.HTTPSPort
	}
	return fmt.Sprintf(":%d", port)
}

func secondsToDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func normalizeTLSMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "off", "none", "disabled":
		return ""
	case "self_signed":
		return string(httpsrv.ModeTLSSelfSigned)
	case "letsencrypt", "lets-encrypt", "auto":
		return string(httpsrv.ModeTLSEncrypt)
	case "remote_api":
		return string(httpsrv.ModeRemoteAPI)
	default:
		return strings.ToLower(strings.TrimSpace(mode))
	}
}

func primaryTLSDomain(domains []string) string {
	if len(domains) == 0 {
		return ""
	}
	return domains[0]
}

func tlsDomains(tls config.TLS) []string {
	seen := map[string]struct{}{}
	domains := make([]string, 0, len(tls.Domains)+1)
	if domain := strings.TrimSpace(tls.Domain); domain != "" {
		domains = append(domains, domain)
		seen[domain] = struct{}{}
	}
	for _, domain := range tls.Domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		domains = append(domains, domain)
		seen[domain] = struct{}{}
	}
	return domains
}

type invalidTLSMode string

func (m invalidTLSMode) Validate() error {
	return fmt.Errorf("unsupported tls enableMode %q", string(m))
}

func (m invalidTLSMode) Run(_ *http.Server) error {
	return m.Validate()
}

// NewHTTPServer creates a new http server
func NewHTTPServer(addr string, opts ...HTTPOption) app.IServer {
	o := defaultHTTPOptions()
	o.apply(opts...)
	cfg := config.Get().HTTP
	if o.tls != nil {
		cfg.TLS = *o.tls
	}
	if normalizeTLSMode(cfg.TLS.EnableMode) != "" && cfg.HTTPSPort > 0 {
		addr = serviceAddr(cfg)
	}

	if o.isProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := o.handler
	if router == nil {
		router = routers.NewRouter()
	}
	server := &http.Server{
		Addr: addr,
		Handler: httpsrv.WrapHandler(router, httpsrv.MiddlewareOptions{
			AddRequestStartHeader: cfg.AddRequestStartHeader, GzipEnabled: cfg.GzipEnabled,
			LogRequests: cfg.LogRequests, MaxRequestBodyBytes: cfg.MaxRequestBodyBytes,
		}),
		ReadTimeout:    secondsToDuration(cfg.ReadTimeout),
		WriteTimeout:   secondsToDuration(cfg.WriteTimeout),
		IdleTimeout:    secondsToDuration(cfg.IdleTimeout),
		MaxHeaderBytes: 1 << 20,
	}

	return &httpServer{
		addr:   addr,
		server: newServer(server, cfg),
	}
}
