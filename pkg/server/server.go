/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/external-secrets/bitwarden-sdk-server/pkg/bitwarden"
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
	r.Get(api+"/get/secret", func(w http.ResponseWriter, r *http.Request) {
		bitwarden.GetSecret()
		_, _ = w.Write([]byte("welcome"))
	})

	srv := &http.Server{Addr: s.Addr, Handler: r, ReadTimeout: 5 * time.Second}
	s.server = srv

	if s.Insecure {
		slog.Info("starting to listen on http", "addr", s.Addr)
		return srv.ListenAndServe()
	}

	return srv.ListenAndServeTLS(s.CertFile, s.KeyFile)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
