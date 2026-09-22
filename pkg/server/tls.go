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
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
)

var (
	cipherIDs  = map[string]uint16{}
	versionIDs = map[string]uint16{}
	curveNames = map[string]tls.CurveID{}
)

func init() {
	// See https://pkg.go.dev/crypto/tls#CipherSuites for cipher names.
	for _, cs := range append(tls.CipherSuites(), tls.InsecureCipherSuites()...) {
		cipherIDs[cs.Name] = cs.ID
	}
	// See https://pkg.go.dev/crypto/tls#VersionName for version names.
	for _, v := range []uint16{tls.VersionTLS10, tls.VersionTLS11, tls.VersionTLS12, tls.VersionTLS13} {
		versionIDs[strings.TrimPrefix(tls.VersionName(v), "TLS ")] = v
	}
	// See https://pkg.go.dev/crypto/tls#CurveID for curve names.
	for _, c := range []tls.CurveID{tls.CurveP256, tls.CurveP384, tls.CurveP521, tls.X25519, tls.X25519MLKEM768} {
		curveNames[c.String()] = c
		curveNames[strconv.FormatUint(uint64(c), 10)] = c
	}
}

// TLSConfig builds the server TLS profile. Insecure mode returns nil without
// applying TLS flags. Invalid explicit values return an error. Empty cipher,
// min-version, and curve settings leave Go defaults in place. HTTP/2 is
// enabled or disabled on the HTTP server, not via NextProtos; when HTTP/2 is
// on, CipherSuites must include an HTTP/2 MTI cipher unless MinVersion is 1.3.
func (s *Server) TLSConfig() (*tls.Config, error) {
	if s.Insecure {
		return nil, nil
	}

	cfg, err := buildTLSConfig(s.TLSCiphers, s.TLSMinVersion, s.TLSCurvePreferences)
	if err != nil {
		return nil, err
	}
	if !s.DisableHTTP2 {
		if err := checkHTTP2Ciphers(cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// checkHTTP2Ciphers mirrors net/http/internal/http2.Server.Configure: an
// explicit TLS 1.0–1.2 CipherSuites list must include AES_128_GCM_SHA256.
// TLS 1.3 ignores CipherSuites, so MinVersion >= 1.3 skips the check.
func checkHTTP2Ciphers(cfg *tls.Config) error {
	if cfg.CipherSuites != nil && cfg.MinVersion < tls.VersionTLS13 {
		haveRequired := false
		for _, cs := range cfg.CipherSuites {
			switch cs {
			case tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
				haveRequired = true
			}
		}
		if !haveRequired {
			return fmt.Errorf("http2: TLSConfig.CipherSuites is missing an HTTP/2-required AES_128_GCM_SHA256 cipher " +
				"(need at least one of TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 or TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256)")
		}
	}

	return nil
}

func buildTLSConfig(ciphers, minVer string, curves []string) (*tls.Config, error) {
	cfg := &tls.Config{}

	ciphers = strings.TrimSpace(ciphers)
	if ciphers != "" {
		ids, err := parseCipherSuites(ciphers)
		if err != nil {
			return nil, fmt.Errorf("unable to parse tls ciphers: %w", err)
		}
		if len(ids) > 0 {
			cfg.CipherSuites = ids
		}
	}

	minVer = strings.TrimSpace(minVer)
	if minVer != "" {
		ver, ok := versionIDs[minVer]
		if !ok {
			return nil, fmt.Errorf("unable to parse tls min version: unsupported TLS minimum version %q; valid values are 1.0, 1.1, 1.2, 1.3", minVer)
		}
		cfg.MinVersion = ver
	}

	prefs, err := parseCurvePreferences(curves)
	if err != nil {
		return nil, fmt.Errorf("unable to parse tls curve preferences: %w", err)
	}
	if len(prefs) > 0 {
		cfg.CurvePreferences = prefs
	}

	return cfg, nil
}

func parseCipherSuites(cipherListString string) ([]uint16, error) {
	parts := strings.Split(cipherListString, ",")
	ret := make([]uint16, 0, len(parts))
	for _, c := range parts {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		id, ok := cipherIDs[c]
		if !ok {
			return nil, fmt.Errorf("cipher %q was not found", c)
		}
		ret = append(ret, id)
	}

	return ret, nil
}

func parseCurvePreferences(names []string) ([]tls.CurveID, error) {
	out := make([]tls.CurveID, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		id, err := parseCurveID(name)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, nil
	}

	return out, nil
}

func parseCurveID(name string) (tls.CurveID, error) {
	if id, ok := curveNames[name]; ok {
		return id, nil
	}

	return 0, fmt.Errorf("unknown tls curve preference %q (use a name like CurveP256, X25519, or a decimal CurveID)", name)
}
