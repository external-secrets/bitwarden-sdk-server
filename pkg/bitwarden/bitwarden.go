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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bitwarden/sdk-go/v2"
)

type contextKey string

var ContextClientKey contextKey = "warden-client"

// Default Settings.
const (
	defaultAPIURL      = "https://api.bitwarden.com"
	defaultIdentityURL = "https://identity.bitwarden.com"
)

// DefaultStatePath uses distroless nonroot image's /home/nonroot as 0700 owned by uid 65532.
// This is writable without a volume being mounted normally. So, we assume that as default.
const DefaultStatePath = "/home/nonroot/.bitwarden-state"

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
func Login(req *LoginRequest, statePathDefault string) (sdk.BitwardenClientInterface, error) {
	// Configuring the URLS is optional, set them to nil to use the default values
	apiURL := setOrDefault(req.APIURL, defaultAPIURL)
	identityURL := setOrDefault(req.IdentityURL, defaultIdentityURL)

	statePath := stateFilePath(req.StatePath, req.AccessToken, statePathDefault)

	// A nil state path tells the SDK to keep the session in memory only.
	var state *string
	if statePath != "" {
		state = new(statePath)
	}

	// Client is closed in the calling handlers.
	slog.Debug("constructed client with api and identity url", "api", apiURL, "identityUrl", identityURL, "statePath", statePath)
	bitwardenClient, err := newBitwardenClientFn(&apiURL, &identityURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	if err := accessTokenLogin(bitwardenClient, req.AccessToken, state); err != nil {
		bitwardenClient.Close()

		return nil, fmt.Errorf("bitwarden login: %w", err)
	}

	return bitwardenClient, nil
}

// loginLockTimeout bounds how long a login waits for another login that writes the
// same state file. The SDK's AccessTokenLogin takes no context and sets no HTTP
// timeout, and the identity URL is caller-supplied, so a login can stall for as long
// as that endpoint withholds a response. A waiter returns an error on timeout rather
// than accessing the same file without the lock.
const loginLockTimeout = 10 * time.Second

// stateFileLocks serializes logins that write the same state file, so concurrent
// cold-start requests for one token log in once instead of each missing the file, and
// no login reads the file while another is writing it in place. Locks are keyed by the
// resolved state path, i.e. one per token, so a login that stalls on a caller-supplied
// identity URL only delays other requests for the same token, never unrelated ones.
// Entries are reference-counted and removed when idle; stalled logins retain their entries.
var stateFileLocks = newPathLocks()

type pathLocks struct {
	mu    sync.Mutex
	locks map[string]*pathLock
}

type pathLock struct {
	// ch is a buffered channel of size one used as a mutex, so acquisition can time out.
	ch   chan struct{}
	refs int
}

func newPathLocks() *pathLocks {
	return &pathLocks{locks: map[string]*pathLock{}}
}

// acquire takes the lock for path, waiting at most timeout. It always returns a release
// func and reports whether the lock was actually taken. The caller must release the
// reference even when acquisition times out.
func (p *pathLocks) acquire(path string, timeout time.Duration) (release func(), locked bool) {
	p.mu.Lock()
	l := p.locks[path]
	if l == nil {
		l = &pathLock{ch: make(chan struct{}, 1)}
		p.locks[path] = l
	}
	l.refs++
	p.mu.Unlock()

	drop := func() {
		p.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(p.locks, path)
		}
		p.mu.Unlock()
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case l.ch <- struct{}{}:
		return func() { <-l.ch; drop() }, true
	case <-timer.C:
		return drop, false
	}
}

func accessTokenLogin(client sdk.BitwardenClientInterface, accessToken string, state *string) error {
	if state == nil {
		return client.AccessTokenLogin(accessToken, nil)
	}

	release, locked := stateFileLocks.acquire(*state, loginLockTimeout)
	defer release()
	if !locked {
		return fmt.Errorf("timed out waiting for state file login lock")
	}

	return client.AccessTokenLogin(accessToken, state)
}

// stateFilePath picks the state file for a request. The SDK encrypts the state file with
// a key from the access token, so tokens that share a file keep overwriting each other's
// session and every switch costs a new login. Without a header each token gets its own file
// next to the configured state path, named after the token's ID like the bws CLI does.
// An empty statePathDefault without a header keeps the session in memory.
func stateFilePath(header, accessToken, statePathDefault string) string {
	if header != "" {
		return header
	}

	if statePathDefault == "" {
		return ""
	}

	return statePathDefault + "-" + stateFileID(accessToken)
}

// accessTokenIDPattern matches the version and ID of an access token, "0.<uuid>.<secret>:<key>".
var accessTokenIDPattern = regexp.MustCompile(`^0\.([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.`)

// stateFileID returns the access token's ID, or a SHA-256 of the token if it is in a
// format this server does not know.
func stateFileID(accessToken string) string {
	if m := accessTokenIDPattern.FindStringSubmatch(accessToken); m != nil {
		return strings.ToLower(m[1])
	}

	sum := sha256.Sum256([]byte(accessToken))

	return hex.EncodeToString(sum[:])
}

// NewWarden builds a middleware to use with the BitWarden API.
// Header used by the Warden:
// Warden-Access-Token: <token>
// Warden-State-Path: <state-path>
// Warden-Api-Url: <url>
// Warden-Identity-Url: <url>
// Put the client into the context and so if a context contains our client
// we know that calls are authenticated.
func NewWarden(statePath string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return warden(next, statePath)
	}
}

func warden(next http.Handler, statePath string) http.Handler {
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
		client, err := Login(loginRequest, statePath)
		if err != nil {
			http.Error(w, "failed to login to bitwarden using access token: "+err.Error(), http.StatusBadRequest)

			return
		}
		defer client.Close()

		ctx := context.WithValue(r.Context(), ContextClientKey, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// EnsureStatePath creates the state file's directory and checks if it is writable.
// Without a usable state file every request re-authenticates. This could be problematic
// for rate-limiters.
func EnsureStatePath(statePath string) error {
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create state directory %q: %w", dir, err)
	}

	probe := filepath.Join(dir, ".write-probe")
	if err := os.WriteFile(probe, []byte{}, 0o600); err != nil {
		return fmt.Errorf("state directory %q is not writable: %w", dir, err)
	}

	if err := os.Remove(probe); err != nil {
		return fmt.Errorf("failed to clean up write probe in %q: %w", dir, err)
	}

	return nil
}
