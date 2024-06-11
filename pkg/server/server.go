package server

import (
	"context"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

const (
	api = "/rest/api/1"
)

type Config struct {
	Insecure bool
	Debug    bool
	Addr     string
	KeyFile  string
	CertFile string
}

// Server defines a server which runs and accepts requests.
type Server struct {
	Config

	server *http.Server
}

func NewServer(cfg Config) *Server {
	return &Server{Config: cfg}
}

func (s *Server) Run(_ context.Context) error {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get(api, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("welcome"))
	})

	srv := &http.Server{Addr: s.Addr, Handler: r}
	s.server = srv

	if s.Insecure {
		return srv.ListenAndServe()
	}

	return srv.ListenAndServeTLS(s.CertFile, s.KeyFile)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
