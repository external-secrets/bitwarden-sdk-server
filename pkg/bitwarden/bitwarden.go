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
	"github.com/bitwarden/sdk-go"
)

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
func Login(req *LoginRequest) error {
	// Configuring the URLS is optional, set them to nil to use the default values
	apiURL := defaultAPIURL
	identityURL := defaultIdentityURL

	// TODO: Cache the client... or the session?
	bitwardenClient, err := sdk.NewBitwardenClient(&apiURL, &identityURL)
	if err != nil {
		return err
	}

	defer bitwardenClient.Close()

	var statePath string
	if req.StatePath == "" {
		statePath = defaultStatePath
	}

	return bitwardenClient.AccessTokenLogin(req.AccessToken, &statePath)
}
