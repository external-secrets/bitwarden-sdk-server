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
	"os"

	"github.com/bitwarden/sdk-go"
)

func GetSecret() {
	// Configuring the URLS is optional, set them to nil to use the default values
	apiURL := os.Getenv("API_URL")
	identityURL := os.Getenv("IDENTITY_URL")

	bitwardenClient, err := sdk.NewBitwardenClient(&apiURL, &identityURL)
	if err != nil {
		panic(err)
	}

	defer bitwardenClient.Close()

	accessToken := os.Getenv("ACCESS_TOKEN")
	// Configuring the statePath is optional, pass nil
	// in AccessTokenLogin() to not use state
	statePath := os.Getenv("STATE_PATH")

	if err := bitwardenClient.AccessTokenLogin(accessToken, &statePath); err != nil {
		panic(err)
	}
}
