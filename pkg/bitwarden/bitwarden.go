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
	"net/http"

	"github.com/bitwarden/sdk-go"
)

type contextKey string

var contextClientKey contextKey = "warden-client"

const (
	defaultAPIURL      = "https://api.bitwarden.com"
	defaultIdentityURL = "https://identity.bitwarden.com"
	defaultStatePath   = ".bitwarden-state"
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

// Login creates a session for further Bitwarden requests.
// Note: I don't like returning the interface, but that's what
// the client returns.
func Login(req *LoginRequest) (sdk.BitwardenClientInterface, error) {
	// Configuring the URLS is optional, set them to nil to use the default values
	apiURL := defaultAPIURL
	identityURL := defaultIdentityURL

	// TODO: Cache the client... or the session?
	bitwardenClient, err := sdk.NewBitwardenClient(&apiURL, &identityURL)
	if err != nil {
		return nil, err
	}

	defer bitwardenClient.Close()

	var statePath string
	if req.StatePath == "" {
		statePath = defaultStatePath
	}

	if err := bitwardenClient.AccessTokenLogin(req.AccessToken, &statePath); err != nil {
		return nil, fmt.Errorf("bitwarden login: %w", err)
	}

	return bitwardenClient, nil
}

// Warden is a middleware to use with the bitwarden API.
// Header used by the Warden:
// warden-access-token: <token>
// warden-state-path: <state-path>
// warden-api-url: <url>
// warden-identity-url: <url>
// Put the client into the context and so if a context contains our client
// we know that calls are authenticated.
func Warden(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if ctx.Value(contextClientKey) != nil {
			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		ctx = context.WithValue(ctx, contextClientKey, nil)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
