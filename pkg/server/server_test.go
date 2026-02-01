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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bitwarden/sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/external-secrets/bitwarden-sdk-server/pkg/bitwarden"
)

type mockClient struct {
	secrets *mockSecrets
}

var _ sdk.BitwardenClientInterface = &mockClient{}

func (m *mockClient) AccessTokenLogin(accessToken string, statePath *string) error { return nil }
func (m *mockClient) Projects() sdk.ProjectsInterface                              { return nil }
func (m *mockClient) Secrets() sdk.SecretsInterface                                { return m.secrets }
func (m *mockClient) Close()                                                       {}
func (m *mockClient) Generators() sdk.GeneratorsInterface                          { return nil }

type mockSecrets struct {
	getResp      *sdk.SecretResponse
	getErr       error
	getByIDsResp *sdk.SecretsResponse
	getByIDsErr  error
	listResp     *sdk.SecretIdentifiersResponse
	listErr      error
	deleteResp   *sdk.SecretsDeleteResponse
	deleteErr    error
	createResp   *sdk.SecretResponse
	createErr    error
	updateResp   *sdk.SecretResponse
	updateErr    error
	syncResp     *sdk.SecretsSyncResponse
	syncErr      error
}

var _ sdk.SecretsInterface = &mockSecrets{}

func (m *mockSecrets) Get(id string) (*sdk.SecretResponse, error) {
	return m.getResp, m.getErr
}

func (m *mockSecrets) GetByIDS(ids []string) (*sdk.SecretsResponse, error) {
	return m.getByIDsResp, m.getByIDsErr
}

func (m *mockSecrets) List(orgID string) (*sdk.SecretIdentifiersResponse, error) {
	return m.listResp, m.listErr
}

func (m *mockSecrets) Delete(ids []string) (*sdk.SecretsDeleteResponse, error) {
	return m.deleteResp, m.deleteErr
}

func (m *mockSecrets) Create(key, value, note, orgID string, projectIDs []string) (*sdk.SecretResponse, error) {
	return m.createResp, m.createErr
}

func (m *mockSecrets) Update(id, key, value, note, orgID string, projectIDs []string) (*sdk.SecretResponse, error) {
	return m.updateResp, m.updateErr
}

func (m *mockSecrets) Sync(orgID string, lastSyncedDate *time.Time) (*sdk.SecretsSyncResponse, error) {
	return m.syncResp, m.syncErr
}

func TestNewServer(t *testing.T) {
	cfg := Config{
		Addr:     ":8080",
		Insecure: true,
	}
	s := NewServer(cfg)
	assert.Equal(t, cfg.Addr, s.Addr)
	assert.True(t, s.Insecure)
}

func TestReadyEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ready"))
	})
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ready", w.Body.String())
}

func TestLiveEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/live", http.NoBody)
	w := httptest.NewRecorder()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("live"))
	})
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "live", w.Body.String())
}

func TestGetSecretHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		client         *mockClient
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{"id": "test-id"}`,
			client: &mockClient{
				secrets: &mockSecrets{
					getResp: &sdk.SecretResponse{
						ID:    "test-id",
						Key:   "test-key",
						Value: "test-value",
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "secrets api error",
			body: `{"id": "test-id"}`,
			client: &mockClient{
				secrets: &mockSecrets{
					getErr: errors.New("secret not found"),
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "failed to get secret: secret not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(Config{})
			req := httptest.NewRequest(http.MethodGet, "/secret", bytes.NewBufferString(tt.body))
			ctx := context.WithValue(req.Context(), bitwarden.ContextClientKey, tt.client)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			s.getSecretHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestGetByIdsSecretHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		client         *mockClient
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{"ids": ["id1", "id2"]}`,
			client: &mockClient{
				secrets: &mockSecrets{
					getByIDsResp: &sdk.SecretsResponse{
						Data: []sdk.SecretResponse{
							{ID: "id1", Key: "key1"},
							{ID: "id2", Key: "key2"},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "secrets api error",
			body: `{"ids": ["id1"]}`,
			client: &mockClient{
				secrets: &mockSecrets{
					getByIDsErr: errors.New("failed to fetch"),
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "failed to get secrets: failed to fetch\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(Config{})
			req := httptest.NewRequest(http.MethodGet, "/secrets-by-ids", bytes.NewBufferString(tt.body))
			ctx := context.WithValue(req.Context(), bitwarden.ContextClientKey, tt.client)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			s.getByIdsSecretHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestListSecretsHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		client         *mockClient
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{"organizationId": "org-1"}`,
			client: &mockClient{
				secrets: &mockSecrets{
					listResp: &sdk.SecretIdentifiersResponse{
						Data: []sdk.SecretIdentifierResponse{
							{ID: "id1", Key: "key1"},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "list error",
			body: `{"organizationId": "org-1"}`,
			client: &mockClient{
				secrets: &mockSecrets{
					listErr: errors.New("list failed"),
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "failed to get secret: list failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(Config{})
			req := httptest.NewRequest(http.MethodGet, "/secrets", bytes.NewBufferString(tt.body))
			ctx := context.WithValue(req.Context(), bitwarden.ContextClientKey, tt.client)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			s.listSecretsHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestDeleteSecretHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		client         *mockClient
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{"ids": ["id1"]}`,
			client: &mockClient{
				secrets: &mockSecrets{
					deleteResp: &sdk.SecretsDeleteResponse{
						Data: []sdk.SecretDeleteResponse{
							{ID: "id1"},
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "delete error",
			body: `{"ids": ["id1"]}`,
			client: &mockClient{
				secrets: &mockSecrets{
					deleteErr: errors.New("delete failed"),
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "delete failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(Config{})
			req := httptest.NewRequest(http.MethodDelete, "/secret", bytes.NewBufferString(tt.body))
			ctx := context.WithValue(req.Context(), bitwarden.ContextClientKey, tt.client)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			s.deleteSecretHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestCreateSecretHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		client         *mockClient
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{"key": "new-key", "value": "new-value", "note": "note", "organizationId": "org-1", "projectIds": ["proj-1"]}`,
			client: &mockClient{
				secrets: &mockSecrets{
					createResp: &sdk.SecretResponse{
						ID:    "new-id",
						Key:   "new-key",
						Value: "new-value",
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "create error",
			body: `{"key": "new-key", "value": "new-value", "organizationId": "org-1"}`,
			client: &mockClient{
				secrets: &mockSecrets{
					createErr: errors.New("create failed"),
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "failed to create secret: create failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(Config{})
			req := httptest.NewRequest(http.MethodPost, "/secret", bytes.NewBufferString(tt.body))
			ctx := context.WithValue(req.Context(), bitwarden.ContextClientKey, tt.client)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			s.createSecretHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestUpdateSecretHandler(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		client         *mockClient
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "success",
			body: `{"id": "id-1", "key": "updated-key", "value": "updated-value", "note": "note", "organizationId": "org-1", "projectIds": ["proj-1"]}`,
			client: &mockClient{
				secrets: &mockSecrets{
					updateResp: &sdk.SecretResponse{
						ID:    "id-1",
						Key:   "updated-key",
						Value: "updated-value",
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "update error",
			body: `{"id": "id-1", "key": "updated-key", "value": "updated-value", "organizationId": "org-1"}`,
			client: &mockClient{
				secrets: &mockSecrets{
					updateErr: errors.New("update failed"),
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "failed to update secret: update failed\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(Config{})
			req := httptest.NewRequest(http.MethodPut, "/secret", bytes.NewBufferString(tt.body))
			ctx := context.WithValue(req.Context(), bitwarden.ContextClientKey, tt.client)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			s.updateSecretHandler(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedBody != "" {
				assert.Equal(t, tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestGetClient(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		ctx         context.Context
		expectError bool
		errorMsg    string
	}{
		{
			name:        "missing client in context",
			body:        `{}`,
			ctx:         context.Background(),
			expectError: true,
			errorMsg:    "missing client in context",
		},
		{
			name:        "invalid client type in context",
			body:        `{}`,
			ctx:         context.WithValue(context.Background(), bitwarden.ContextClientKey, "not-a-client"),
			expectError: true,
			errorMsg:    "invalid client in context",
		},
		{
			name:        "invalid json body",
			body:        `{invalid`,
			ctx:         context.WithValue(context.Background(), bitwarden.ContextClientKey, &mockClient{}),
			expectError: true,
		},
		{
			name:        "success",
			body:        `{"id": "test"}`,
			ctx:         context.WithValue(context.Background(), bitwarden.ContextClientKey, &mockClient{}),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(Config{})
			req := httptest.NewRequest(http.MethodGet, "/", bytes.NewBufferString(tt.body))
			req = req.WithContext(tt.ctx)

			var resp sdk.SecretGetRequest
			client, err := s.getClient(req, &resp)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, client)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
			}
		})
	}
}

func TestHandleResponse(t *testing.T) {
	tests := []struct {
		name           string
		response       any
		expectedStatus int
	}{
		{
			name: "success",
			response: &sdk.SecretResponse{
				ID:    "id-1",
				Key:   "key-1",
				Value: "value-1",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(Config{})
			w := httptest.NewRecorder()

			s.handleResponse(tt.response, w)

			var resp sdk.SecretResponse
			err := json.NewDecoder(w.Body).Decode(&resp)
			require.NoError(t, err)

			expected := tt.response.(*sdk.SecretResponse)
			assert.Equal(t, expected.ID, resp.ID)
			assert.Equal(t, expected.Key, resp.Key)
			assert.Equal(t, expected.Value, resp.Value)
		})
	}
}

func TestHandleResponseMarshalError(t *testing.T) {
	s := NewServer(Config{})
	w := httptest.NewRecorder()

	s.handleResponse(make(chan int), w)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandlerWithInvalidBody(t *testing.T) {
	s := NewServer(Config{})
	client := &mockClient{secrets: &mockSecrets{}}

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		method  string
		path    string
	}{
		{"getSecretHandler", s.getSecretHandler, http.MethodGet, "/secret"},
		{"getByIdsSecretHandler", s.getByIdsSecretHandler, http.MethodGet, "/secrets-by-ids"},
		{"listSecretsHandler", s.listSecretsHandler, http.MethodGet, "/secrets"},
		{"deleteSecretHandler", s.deleteSecretHandler, http.MethodDelete, "/secret"},
		{"createSecretHandler", s.createSecretHandler, http.MethodPost, "/secret"},
		{"updateSecretHandler", s.updateSecretHandler, http.MethodPut, "/secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_invalid_json", func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString("{invalid"))
			ctx := context.WithValue(req.Context(), bitwarden.ContextClientKey, client)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			assert.True(t, w.Code >= 400)
		})
	}
}

func TestHandlerWithNoClientInContext(t *testing.T) {
	s := NewServer(Config{})

	tests := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
		method  string
		path    string
	}{
		{"getSecretHandler", s.getSecretHandler, http.MethodGet, "/secret"},
		{"getByIdsSecretHandler", s.getByIdsSecretHandler, http.MethodGet, "/secrets-by-ids"},
		{"listSecretsHandler", s.listSecretsHandler, http.MethodGet, "/secrets"},
		{"deleteSecretHandler", s.deleteSecretHandler, http.MethodDelete, "/secret"},
		{"createSecretHandler", s.createSecretHandler, http.MethodPost, "/secret"},
		{"updateSecretHandler", s.updateSecretHandler, http.MethodPut, "/secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_no_client", func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString("{}"))
			w := httptest.NewRecorder()

			tt.handler(w, req)

			assert.True(t, w.Code >= 400)
		})
	}
}
