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
	"errors"
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
	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ready"))
	})
	r.Get("/live", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("live"))
	})

	warden := chi.NewRouter()
	warden.Use(bitwarden.Warden)

	// The header will always contain the right credentials.
	warden.Get("/secret", s.getSecretHandler)
	warden.Get("/secrets", s.listSecretsHandler)
	warden.Get("/secrets-by-ids", s.getByIdsSecretHandler)
	warden.Delete("/secret", s.deleteSecretHandler)
	warden.Post("/secret", s.createSecretHandler)
	warden.Put("/secret", s.updateSecretHandler)

	r.Mount(api, warden)

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

func (s *Server) getSecretHandler(w http.ResponseWriter, r *http.Request) {
	request := &sdk.SecretGetRequest{}
	c, err := s.getClient(r, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	defer c.Close()

	secretResponse, err := c.Secrets().Get(request.ID)
	if err != nil {
		http.Error(w, "failed to get secret: "+err.Error(), http.StatusBadRequest)

		return
	}

	s.handleResponse(secretResponse, w)
}

func (s *Server) getByIdsSecretHandler(w http.ResponseWriter, r *http.Request) {
	request := &sdk.SecretsGetRequest{}
	c, err := s.getClient(r, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	defer c.Close()

	secretResponse, err := c.Secrets().GetByIDS(request.IDS)
	if err != nil {
		http.Error(w, "failed to get secrets: "+err.Error(), http.StatusBadRequest)

		return
	}

	s.handleResponse(secretResponse, w)
}

func (s *Server) listSecretsHandler(w http.ResponseWriter, r *http.Request) {
	request := &sdk.SecretIdentifiersRequest{}
	c, err := s.getClient(r, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	defer c.Close()

	secretResponse, err := c.Secrets().List(request.OrganizationID)
	if err != nil {
		http.Error(w, "failed to get secret: "+err.Error(), http.StatusBadRequest)

		return
	}

	s.handleResponse(secretResponse, w)
}

func (s *Server) deleteSecretHandler(w http.ResponseWriter, r *http.Request) {
	request := &sdk.SecretsDeleteRequest{}
	c, err := s.getClient(r, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}
	defer c.Close()

	response, err := c.Secrets().Delete(request.IDS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	s.handleResponse(response, w)
}

func (s *Server) createSecretHandler(w http.ResponseWriter, r *http.Request) {
	request := &sdk.SecretCreateRequest{}
	c, err := s.getClient(r, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}
	defer c.Close()

	response, err := c.Secrets().Create(request.Key, request.Value, request.Note, request.OrganizationID, request.ProjectIDS)
	if err != nil {
		http.Error(w, "failed to create secret: "+err.Error(), http.StatusBadRequest)

		return
	}

	s.handleResponse(response, w)
}

func (s *Server) updateSecretHandler(w http.ResponseWriter, r *http.Request) {
	request := &sdk.SecretPutRequest{}
	c, err := s.getClient(r, &request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}
	defer c.Close()

	response, err := c.Secrets().Update(request.ID, request.Key, request.Value, request.Note, request.OrganizationID, request.ProjectIDS)
	if err != nil {
		http.Error(w, "failed to update secret: "+err.Error(), http.StatusBadRequest)

		return
	}

	s.handleResponse(response, w)
}

func (s *Server) getClient(r *http.Request, response any) (sdk.BitwardenClientInterface, error) {
	content, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	if err := json.Unmarshal(content, response); err != nil {
		return nil, err
	}

	client := r.Context().Value(bitwarden.ContextClientKey)
	if client == nil {
		return nil, errors.New("missing client in context, login error")
	}

	c, ok := client.(sdk.BitwardenClientInterface)
	if !ok {
		return nil, errors.New("invalid client in context, login error")
	}

	return c, nil
}

func (s *Server) handleResponse(response any, w http.ResponseWriter) {
	body, err := json.Marshal(response)
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
