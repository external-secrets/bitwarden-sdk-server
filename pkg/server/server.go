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
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/bitwarden/sdk-go"
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
	r.Use(middleware.Recoverer)
	r.Use(bitwarden.Warden)
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ready"))
	})
	r.Get("/live", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("live"))
	})

	// The header will always contain the right credentials.
	r.Get(api+"/secret", s.getSecretHandler)
	r.Delete(api+"/secret", s.deleteSecretHandler)
	r.Post(api+"/secret", s.createSecretHandler)

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

// GetSecretRequest is the format in which we required secrets to be requested in.
type GetSecretRequest struct {
	SecretID string `json:"secretId"`
}

func (s *Server) getSecretHandler(w http.ResponseWriter, r *http.Request) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}
	defer r.Body.Close()

	request := &GetSecretRequest{}
	if err := json.Unmarshal(content, request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	client := r.Context().Value(bitwarden.ContextClientKey)
	if client == nil {
		http.Error(w, "missing client in context, login error", http.StatusInternalServerError)

		return
	}

	c := client.(sdk.BitwardenClientInterface)
	secretResponse, err := c.Secrets().Get(request.SecretID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	body, err := json.Marshal(secretResponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if _, err := w.Write(body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) deleteSecretHandler(w http.ResponseWriter, _ *http.Request) {
}

func (s *Server) createSecretHandler(w http.ResponseWriter, _ *http.Request) {
}
