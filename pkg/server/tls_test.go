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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildTLSConfig(t *testing.T) {
	tests := []struct {
		name                 string
		ciphers              string
		minVer               string
		curves               []string
		wantErr              bool
		wantMinVersion       uint16
		wantCipherSuites     []uint16
		wantCurvePreferences []tls.CurveID
	}{
		{
			name: "all empty uses Go defaults",
		},
		{
			name:           "empty minVersion does not set MinVersion",
			minVer:         "",
			wantMinVersion: 0,
		},
		{
			name:           "explicit minVersion sets MinVersion",
			minVer:         "1.3",
			wantMinVersion: tls.VersionTLS13,
		},
		{
			name:           "explicit minVersion 1.2",
			minVer:         "1.2",
			wantMinVersion: tls.VersionTLS12,
		},
		{
			name:             "valid cipher suites are applied",
			ciphers:          "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			wantCipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name:             "cipher suite names with spaces are trimmed",
			ciphers:          "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			wantCipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name:    "invalid cipher suite returns error",
			ciphers: "NOT_A_REAL_CIPHER",
			wantErr: true,
		},
		{
			name:                 "valid curve preferences are applied",
			curves:               []string{"X25519", "CurveP256"},
			wantCurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		},
		{
			name:    "invalid curve preference returns error",
			curves:  []string{"not-a-curve"},
			wantErr: true,
		},
		{
			name:    "invalid minVersion returns error",
			minVer:  "1.4",
			wantErr: true,
		},
		{
			name:                 "all settings together",
			ciphers:              "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			minVer:               "1.2",
			curves:               []string{"X25519"},
			wantMinVersion:       tls.VersionTLS12,
			wantCipherSuites:     []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
			wantCurvePreferences: []tls.CurveID{tls.X25519},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := buildTLSConfig(tt.ciphers, tt.minVer, tt.curves)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Equal(t, tt.wantMinVersion, cfg.MinVersion)
			require.Equal(t, tt.wantCipherSuites, cfg.CipherSuites)
			require.Equal(t, tt.wantCurvePreferences, cfg.CurvePreferences)
			require.Empty(t, cfg.NextProtos)
		})
	}
}

func TestParseCurvePreferences(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []tls.CurveID
		wantErr bool
	}{
		{
			name:  "well-known curve names",
			input: []string{"X25519", "CurveP256"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256},
		},
		{
			name:  "whitespace is trimmed",
			input: []string{" X25519 ", "CurveP256"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256},
		},
		{
			name:  "nil input uses Go defaults",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty input uses Go defaults",
			input: []string{},
			want:  nil,
		},
		{
			name:    "unknown curve name",
			input:   []string{"not-a-curve"},
			wantErr: true,
		},
		{
			name:  "empty curve entries are skipped",
			input: []string{"X25519", ""},
			want:  []tls.CurveID{tls.X25519},
		},
		{
			name:  "decimal curve ID",
			input: []string{"29"},
			want:  []tls.CurveID{tls.X25519},
		},
		{
			name:  "all four standard curves",
			input: []string{"X25519", "CurveP256", "CurveP384", "CurveP521"},
			want:  []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384, tls.CurveP521},
		},
		{
			name:    "numeric curve ID larger than uint16",
			input:   []string{"65536"},
			wantErr: true,
		},
		{
			name:    "unsupported decimal curve ID",
			input:   []string{"9999"},
			wantErr: true,
		},
		{
			name:  "valid post-quantum hybrid curve by number",
			input: []string{"4588"},
			want:  []tls.CurveID{tls.X25519MLKEM768},
		},
		{
			name:  "post-quantum hybrid curve by name",
			input: []string{"X25519MLKEM768"},
			want:  []tls.CurveID{tls.X25519MLKEM768},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCurvePreferences(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestTLSConfigInsecure(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "no tls flags",
			cfg:  Config{Insecure: true},
		},
		{
			name: "invalid min version is ignored",
			cfg:  Config{Insecure: true, TLSMinVersion: "1.4"},
		},
		{
			name: "invalid cipher is ignored",
			cfg:  Config{Insecure: true, TLSCiphers: "NOT_A_REAL_CIPHER"},
		},
		{
			name: "invalid curve is ignored",
			cfg:  Config{Insecure: true, TLSCurvePreferences: []string{"not-a-curve"}},
		},
		{
			name: "valid tls flags are ignored",
			cfg: Config{
				Insecure:            true,
				TLSMinVersion:       "1.3",
				TLSCiphers:          "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				TLSCurvePreferences: []string{"X25519"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewServer(tt.cfg).TLSConfig()
			require.NoError(t, err)
			require.Nil(t, cfg)
		})
	}
}

func TestTLSConfigSecure(t *testing.T) {
	tests := []struct {
		name                 string
		cfg                  Config
		wantErr              bool
		errContains          string
		wantMinVersion       uint16
		wantCipherSuites     []uint16
		wantCurvePreferences []tls.CurveID
	}{
		{
			name: "defaults leave Go TLS settings unset",
		},
		{
			name:           "min version 1.0",
			cfg:            Config{TLSMinVersion: "1.0"},
			wantMinVersion: tls.VersionTLS10,
		},
		{
			name:           "min version 1.1",
			cfg:            Config{TLSMinVersion: "1.1"},
			wantMinVersion: tls.VersionTLS11,
		},
		{
			name:           "min version 1.2",
			cfg:            Config{TLSMinVersion: "1.2"},
			wantMinVersion: tls.VersionTLS12,
		},
		{
			name:           "min version 1.3",
			cfg:            Config{TLSMinVersion: "1.3"},
			wantMinVersion: tls.VersionTLS13,
		},
		{
			name:           "min version with surrounding whitespace",
			cfg:            Config{TLSMinVersion: " 1.3 "},
			wantMinVersion: tls.VersionTLS13,
		},
		{
			name:        "invalid min version",
			cfg:         Config{TLSMinVersion: "1.4"},
			wantErr:     true,
			errContains: `unsupported TLS minimum version "1.4"`,
		},
		{
			name:             "single cipher suite",
			cfg:              Config{TLSCiphers: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
			wantCipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name: "cipher suite names with spaces are trimmed",
			cfg:  Config{TLSCiphers: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"},
			wantCipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		},
		{
			name: "multiple cipher suites preserve order",
			cfg:  Config{TLSCiphers: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"},
			wantCipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		},
		{
			name:        "invalid cipher suite",
			cfg:         Config{TLSCiphers: "NOT_A_REAL_CIPHER"},
			wantErr:     true,
			errContains: `cipher "NOT_A_REAL_CIPHER" was not found`,
		},
		{
			name:             "insecure cipher suite from stdlib is accepted when HTTP/2 is off",
			cfg:              Config{DisableHTTP2: true, TLSCiphers: "TLS_RSA_WITH_AES_128_CBC_SHA"},
			wantCipherSuites: []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA},
		},
		{
			name:        "HTTP/2 rejects TLS 1.2 cipher list missing AES_128_GCM_SHA256",
			cfg:         Config{TLSCiphers: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
			wantErr:     true,
			errContains: "HTTP/2-required AES_128_GCM_SHA256",
		},
		{
			name:             "HTTP/2 allows AES_256_GCM when min version is 1.3",
			cfg:              Config{TLSMinVersion: "1.3", TLSCiphers: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
			wantMinVersion:   tls.VersionTLS13,
			wantCipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
		},
		{
			name:             "HTTP/2 accepts ECDSA MTI cipher",
			cfg:              Config{TLSCiphers: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"},
			wantCipherSuites: []uint16{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256},
		},
		{
			name:             "HTTP/2 off allows AES_256_GCM without MTI cipher",
			cfg:              Config{DisableHTTP2: true, TLSCiphers: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
			wantCipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384},
		},
		{
			name:                 "curve names",
			cfg:                  Config{TLSCurvePreferences: []string{"X25519", "CurveP256"}},
			wantCurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
		},
		{
			name:                 "curve names with whitespace",
			cfg:                  Config{TLSCurvePreferences: []string{" CurveP256 ", "CurveP384"}},
			wantCurvePreferences: []tls.CurveID{tls.CurveP256, tls.CurveP384},
		},
		{
			name:                 "empty curve entries are skipped",
			cfg:                  Config{TLSCurvePreferences: []string{"", "X25519", ""}},
			wantCurvePreferences: []tls.CurveID{tls.X25519},
		},
		{
			name:                 "decimal curve ID",
			cfg:                  Config{TLSCurvePreferences: []string{"29"}},
			wantCurvePreferences: []tls.CurveID{tls.X25519},
		},
		{
			name:        "invalid curve name",
			cfg:         Config{TLSCurvePreferences: []string{"not-a-curve"}},
			wantErr:     true,
			errContains: `unknown tls curve preference "not-a-curve"`,
		},
		{
			name:           "http2 flag does not set NextProtos",
			cfg:            Config{DisableHTTP2: true, TLSMinVersion: "1.3"},
			wantMinVersion: tls.VersionTLS13,
		},
		{
			name: "all tls flags together",
			cfg: Config{
				TLSMinVersion:       "1.2",
				TLSCiphers:          "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				TLSCurvePreferences: []string{"X25519"},
			},
			wantMinVersion:       tls.VersionTLS12,
			wantCipherSuites:     []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
			wantCurvePreferences: []tls.CurveID{tls.X25519},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := NewServer(tt.cfg).TLSConfig()
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, cfg)
				if tt.errContains != "" {
					require.ErrorContains(t, err, tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cfg)
			require.Equal(t, tt.wantMinVersion, cfg.MinVersion)
			require.Equal(t, tt.wantCipherSuites, cfg.CipherSuites)
			require.Equal(t, tt.wantCurvePreferences, cfg.CurvePreferences)
			require.Empty(t, cfg.NextProtos)
		})
	}
}
