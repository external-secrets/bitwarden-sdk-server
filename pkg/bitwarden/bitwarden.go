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
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bitwarden/sdk-go"
)

type contextKey string

var ContextClientKey contextKey = "warden-client"

// Default Settings.
const (
	defaultAPIURL      = "https://api.bitwarden.com"
	defaultIdentityURL = "https://identity.bitwarden.com"
	defaultStatePath   = ".bitwarden-state"
)

// Defined Header Keys.
const (
	WardenHeaderAccessToken = "Warden-Access-Token"
	WardenHeaderStatePath   = "Warden-State-Path"
	WardenHeaderAPIURL      = "Warden-Api-Url"
	WardenHeaderIdentityURL = "Warden-Identity-Url"
)

// RequestBase contains optional API_URL and IDENTITY_URL values. If not defined,
// defaults are used always.
type RequestBase struct {
	APIURL      string `yaml:"apiUrl,omitempty"`
	IdentityURL string `yaml:"identityUrl,omitempty"`
}

// LoginRequest defines bitwarden login details to Secrets Manager.
type LoginRequest struct {
	*RequestBase `yaml:",inline,omitempty"`

	AccessToken string `yaml:"accessToken"`
	StatePath   string `yaml:"statePath,omitempty"`
}

// setOrDefault returns a value if not empty, otherwise a default.
func setOrDefault(v, def string) string {
	if v != "" {
		return v
	}

	return def
}

// newBitwardenClientFn is used to overwrite how to initialize the bitwarden client.
var newBitwardenClientFn = sdk.NewBitwardenClient

// Login creates a session for further Bitwarden requests.
// Note: I don't like returning the interface, but that's what
// the client returns.
func Login(req *LoginRequest) (sdk.BitwardenClientInterface, error) {
	// Configuring the URLS is optional, set them to nil to use the default values
	apiURL := setOrDefault(req.APIURL, defaultAPIURL)
	identityURL := setOrDefault(req.IdentityURL, defaultIdentityURL)
	statePath := setOrDefault(req.StatePath, defaultStatePath)

	// Client is closed in the calling handlers.
	slog.Debug("constructed client with api and identity url", "api", apiURL, "identityUrl", identityURL, "statePath", statePath)
	bitwardenClient, err := newBitwardenClientFn(&apiURL, &identityURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	if err := bitwardenClient.AccessTokenLogin(req.AccessToken, &statePath); err != nil {
		return nil, fmt.Errorf("bitwarden login: %w", err)
	}

	return bitwardenClient, nil
}

// Warden is a middleware to use with the bitwarden API.
// Header used by the Warden:
// Warden-Access-Token: <token>
// Warden-State-Path: <state-path>
// Warden-Api-Url: <url>
// Warden-Identity-Url: <url>
// Put the client into the context and so if a context contains our client
// we know that calls are authenticated.
func Warden(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(WardenHeaderAccessToken)
		if token == "" {
			http.Error(w, "Missing Warden access token", http.StatusUnauthorized)

			return
		}

		loginRequest := &LoginRequest{
			RequestBase: &RequestBase{
				APIURL:      r.Header.Get(WardenHeaderAPIURL),
				IdentityURL: r.Header.Get(WardenHeaderIdentityURL),
			},
			AccessToken: token,
			StatePath:   r.Header.Get(WardenHeaderStatePath),
		}

		// Make sure every request gets its own client that it will close after it's done.
		client, err := Login(loginRequest)
		if err != nil {
			http.Error(w, "failed to login to bitwarden using access token: "+err.Error(), http.StatusBadRequest)

			return
		}
		defer client.Close()

		ctx := context.WithValue(r.Context(), ContextClientKey, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
