package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/go-dev-frame/sponge/pkg/app"
	"github.com/go-dev-frame/sponge/pkg/logger"

	"thrust_oauth2id/internal/config"
	"thrust_oauth2id/internal/routers"
	"thrust_oauth2id/internal/server/httpmiddleware"
)

var _ app.IServer = (*httpServer)(nil)

type httpServer struct {
	httpAddr    string
	httpsAddr   string
	httpServer  *http.Server
	httpsServer *http.Server
	tlsEnabled  bool
}

var (
	serveHTTP  = listenAndServe
	serveHTTPS = listenAndServeTLS
)

// Start http/https service
func (s *httpServer) Start() error {
	if s.tlsEnabled {
		errCh := make(chan error, 2)

		go func() {
			errCh <- serveHTTP(s.httpServer)
		}()

		go func() {
			errCh <- serveHTTPS(s.httpsServer)
		}()

		completed := 0
		for completed < 2 {
			if err := <-errCh; err != nil {
				s.shutdownListenersOnStartError()
				return err
			}
			completed++
		}

		return nil
	}

	if err := serveHTTP(s.httpServer); err != nil {
		return err
	}

	return nil
}

// Stop http/https service
func (s *httpServer) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var firstErr error
	if err := s.httpServer.Shutdown(ctx); err != nil {
		firstErr = err
	}

	if s.tlsEnabled && s.httpsServer != nil {
		if err := s.httpsServer.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			if firstErr == nil {
				firstErr = err
			} else {
				logger.Error("https server shutdown reported additional error", logger.Err(err))
			}
		}
	}

	return firstErr
}

func (s *httpServer) shutdownListenersOnStartError() {
	shutdown := func(name string, srv *http.Server) {
		if srv == nil {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server shutdown after startup error", logger.String("server", name), logger.Err(err))
		}
	}

	shutdown("http", s.httpServer)
	if s.tlsEnabled {
		shutdown("https", s.httpsServer)
	}
}

// String provides a human readable description of listener addresses.
func (s *httpServer) String() string {
	if s.tlsEnabled {
		return fmt.Sprintf("http service redirecting on %s and https service address %s", s.httpAddr, s.httpsAddr)
	}
	return "http service address " + s.httpAddr
}

// NewHTTPServer creates an HTTP server with optional automatic TLS.
func NewHTTPServer(cfg config.HTTP, opts ...HTTPOption) app.IServer {
	o := defaultHTTPOptions()
	o.apply(opts...)

	if o.isProd {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	appHandler := o.handler
	if appHandler == nil {
		appHandler = routers.NewRouter()
	}

	appHandler = httpmiddleware.Wrap(appHandler, httpmiddleware.Options{
		AddRequestStartHeader: cfg.AddRequestStartHeader,
		GzipEnabled:           cfg.GzipEnabled,
		LogRequests:           cfg.LogRequests,
		MaxRequestBodyBytes:   cfg.MaxRequestBodyBytes,
	})

	readTimeout := secondsToDuration(cfg.ReadTimeout)
	writeTimeout := secondsToDuration(cfg.WriteTimeout)
	idleTimeout := secondsToDuration(cfg.IdleTimeout)

	httpSrv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Port),
		Handler:        appHandler,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	domains := filterDomains(cfg.TLS.Domains)
	tlsEnabled := len(domains) > 0

	var (
		httpsSrv  *http.Server
		httpsAddr string
	)
	if tlsEnabled {
		manager := buildAutocertManager(cfg, domains)
		httpSrv.Handler = manager.HTTPHandler(httpRedirectHandler(cfg.HTTPSPort))

		httpsSrv = &http.Server{
			Addr:           fmt.Sprintf(":%d", cfg.HTTPSPort),
			Handler:        appHandler,
			ReadTimeout:    readTimeout,
			WriteTimeout:   writeTimeout,
			IdleTimeout:    idleTimeout,
			MaxHeaderBytes: 1 << 20,
			TLSConfig:      manager.TLSConfig(),
		}
		httpsAddr = httpsSrv.Addr

		logger.Info("automatic TLS enabled", logger.String("http_addr", httpSrv.Addr), logger.String("https_addr", httpsSrv.Addr), logger.Any("domains", domains))
	} else {
		logger.Info("automatic TLS disabled", logger.String("http_addr", httpSrv.Addr))
	}

	return &httpServer{
		httpAddr:    httpSrv.Addr,
		httpsAddr:   httpsAddr,
		httpServer:  httpSrv,
		httpsServer: httpsSrv,
		tlsEnabled:  tlsEnabled,
	}
}

func listenAndServe(server *http.Server) error {
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen server error: %w", err)
	}
	return nil
}

func listenAndServeTLS(server *http.Server) error {
	if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen tls server error: %w", err)
	}
	return nil
}

func secondsToDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func filterDomains(domains []string) []string {
	filtered := make([]string, 0, len(domains))
	for _, domain := range domains {
		cleaned := strings.TrimSpace(domain)
		if cleaned != "" {
			filtered = append(filtered, cleaned)
		}
	}
	return filtered
}

func buildAutocertManager(cfg config.HTTP, domains []string) *autocert.Manager {
	client := &acme.Client{DirectoryURL: cfg.TLS.AcmeDirectory}
	binding := externalAccountBinding(cfg.TLS.Eab)

	if binding == nil {
		logger.Debug("http server: initializing autocert manager without EAB")
	} else {
		logger.Debug("http server: initializing autocert manager with EAB")
	}

	return &autocert.Manager{
		Cache:                  autocert.DirCache(cfg.TLS.StoragePath),
		Client:                 client,
		ExternalAccountBinding: binding,
		HostPolicy:             autocert.HostWhitelist(domains...),
		Prompt:                 autocert.AcceptTOS,
	}
}

func externalAccountBinding(eab config.Eab) *acme.ExternalAccountBinding {
	kid := strings.TrimSpace(eab.Kid)
	secret := strings.TrimSpace(eab.HmacKey)
	if kid == "" || secret == "" {
		return nil
	}

	key, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		logger.Error("failed to decode EAB HMAC key", logger.Err(err))
		return nil
	}

	return &acme.ExternalAccountBinding{KID: kid, Key: key}
}

func httpRedirectHandler(httpsPort int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")

		host := strings.TrimSpace(r.Host)
		if parsedHost, _, err := net.SplitHostPort(host); err == nil && parsedHost != "" {
			host = parsedHost
		}

		if host == "" && r.URL != nil {
			host = r.URL.Host
		}

		targetHost := host
		if host != "" && httpsPort > 0 && httpsPort != 443 {
			targetHost = net.JoinHostPort(host, strconv.Itoa(httpsPort))
		}

		target := "https://" + targetHost + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}
