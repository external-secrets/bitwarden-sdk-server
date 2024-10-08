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

package bitwarden

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bitwarden/sdk-go"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This is a valid test token that has been generated and then revoked. The important
// bit is that it is in valid format. If ever this needs to be changed, it has to be
// regenerated by a valid bitwarden secret manager machine account.
const testToken = "0.0f0a8a7e-d737-498d-be6a-b1930064c31c.Rjqj7Vt7ZcDADtpbuzgx0hqbNJFPho:RZbls0Ka2gwIVKaLvSG3eA=="

type testClient struct{}

var _ sdk.BitwardenClientInterface = &testClient{}

type testSecrets struct {
	sdk.SecretsInterface
}

var _ sdk.SecretsInterface = &testSecrets{}

func (t *testClient) AccessTokenLogin(accessToken string, statePath *string) error {
	return nil
}

func (t *testClient) Projects() sdk.ProjectsInterface {
	return &sdk.Projects{}
}

func (t *testClient) Secrets() sdk.SecretsInterface {
	return nil
}

func (t *testClient) Close() {}

func (t *testClient) Generators() sdk.GeneratorsInterface {
	return nil
}

var testBitwardenClient = func(testClient *testClient) func(apiURL, identityURL *string) (sdk.BitwardenClientInterface, error) {
	return func(apiURL, identityURL *string) (sdk.BitwardenClientInterface, error) {
		return testClient, nil
	}
}

func TestWardenWithToken(t *testing.T) {
	prevBitwardenClient := newBitwardenClientFn
	mockClient := &testClient{}
	newBitwardenClientFn = testBitwardenClient(mockClient)
	defer func() {
		newBitwardenClientFn = prevBitwardenClient
	}()

	bitwardenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer bitwardenServer.Close()

	// this is our warden test mock
	r := chi.NewRouter()
	r.Use(Warden)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(r)
	defer server.Close()

	url := server.URL + "/"
	client := server.Client()
	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer([]byte(`{}`)))
	require.NoError(t, err)
	req.Header.Set(WardenHeaderAccessToken, testToken)
	req.Header.Set(WardenHeaderAPIURL, bitwardenServer.URL)
	req.Header.Set(WardenHeaderIdentityURL, bitwardenServer.URL)
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "test", string(content))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
