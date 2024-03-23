package http

import (
	ctx "context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/derezzolution/platform/config"
	"github.com/derezzolution/platform/http/middleware"
	"github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

type ServerOptions struct {
	InitializeRoutesFunc func(r *mux.Router)
	Middlware            []alice.Constructor
}

type Server struct {
	config *config.Http
	server *http.Server
	name   string // Name of server (used in logging)
}

func NewServer(name string, httpConfig *config.Http, initializeRoutesFunc func(r *mux.Router)) *Server {
	return NewServerWithOptions(name, httpConfig, &ServerOptions{
		InitializeRoutesFunc: initializeRoutesFunc,
	})
}

func NewServerWithOptions(name string, httpConfig *config.Http, serverOptions *ServerOptions) *Server {
	server := &Server{
		config: httpConfig,
		server: newHttpServer(httpConfig),
		name:   name,
	}
	http.Handle("/", createHttpHandler(serverOptions))
	return server
}

// Serve is the entry-point for the http package. This takes a service, sets up
// http server (as a function of the config) adds routes.
func (s *Server) Serve() {
	go func() {
		s.Logf("started, listeners open")
		var err error
		if s.config.TLSEnable {
			err = s.server.ListenAndServeTLS(s.config.TLSCRT, s.config.TLSKey)
		} else {
			err = s.server.ListenAndServe()
		}
		if err != http.ErrServerClosed {
			s.Logf("unexpected listen and serve response: %s", err)
		}
	}()
}

// Shuts down the http server waiting for active connections to complete.
func (s *Server) Shutdown() error {
	s.Logf("shutting down, closing open listners and waiting for active " +
		"connections to complete")
	err := s.server.Shutdown(ctx.Background())
	if err != nil {
		s.Logf("error shutting down: %s", err)
	}
	s.Logf("shut down complete, open listners and active connections " +
		"terminated")
	return err
}

func (s *Server) fullName() string {
	return fmt.Sprintf("%s-http[%d]", s.name, s.config.Port)
}

func (s *Server) Logf(pattern string, args ...interface{}) {
	log.Printf("%s: "+pattern,
		append([]interface{}{s.fullName()}, args...)...)
}

// newHttpServer creates a new HTTP Server configured with TLS defaults.
//
// Note: Even though we have TLSConfig specified here, it's simply ignored if
// we're not calling ListenAndServeTLS.
//
// Notes:
// https://blog.gopheracademy.com/advent-2016/exposing-go-on-the-internet/
// https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
func newHttpServer(config *config.Http) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  120 * time.Second,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS10,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			},
		},
	}
}

// Creates a standard http handler with core middleware for all http services.
func createHttpHandler(serverOptions *ServerOptions) http.Handler {
	r := mux.NewRouter()
	serverOptions.InitializeRoutesFunc(r)
	return context.ClearHandler(
		alice.New(
			middleware.ThrottleHandler,
			handlers.CompressHandler,
			handlers.CORS(
				handlers.AllowedMethods([]string{"OPTIONS", "DELETE", "GET", "HEAD", "POST", "PUT"}),
				handlers.AllowedHeaders([]string{"Authorization", "Content-Type"}),
			)).Append(serverOptions.Middlware...).Then(r))
}
