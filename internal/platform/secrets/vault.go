package secrets

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// VaultProvider fetches secrets from HashiCorp Vault KV v2. It is intentionally
// minimal to avoid heavy dependencies and keeps HTTP client simple. It expects
// a token-based auth and reads under mount/path, e.g. mount "secret", path "app/config".
type VaultProvider struct {
	addr    string
	token   string
	mount   string
	path    string
	client  *http.Client
	timeout time.Duration
}

type Option func(*VaultProvider)

// WithTimeout overrides the HTTP timeout.
func WithTimeout(d time.Duration) Option {
	return func(v *VaultProvider) { v.timeout = d }
}

// NewVaultProvider constructs a provider. The addr must include scheme and host,
// e.g. http://127.0.0.1:8200. Mount defaults to "secret" if empty.
func NewVaultProvider(addr, token, mount, path string, opts ...Option) (*VaultProvider, error) {
	if addr == "" || token == "" || path == "" {
		return nil, errors.New("vault addr, token, and path are required")
	}
	if mount == "" {
		mount = "secret"
	}
	vp := &VaultProvider{
		addr:    strings.TrimSuffix(addr, "/"),
		token:   token,
		mount:   strings.Trim(mount, "/"),
		path:    strings.Trim(path, "/"),
		timeout: 5 * time.Second,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
	for _, o := range opts {
		o(vp)
	}
	vp.client.Timeout = vp.timeout
	return vp, nil
}

// Fetch returns the secrets stored at the configured path. Keys are returned as a
// simple string map (KV v2 "data" values must be strings to be used here).
func (v *VaultProvider) Fetch() (map[string]string, error) {
	url := fmt.Sprintf("%s/v1/%s/data/%s", v.addr, v.mount, v.path)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Vault-Token", v.token)
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vault: unexpected status %d: %s", resp.StatusCode, string(body))
	}
	var wrap struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&wrap); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(wrap.Data.Data))
	for k, v := range wrap.Data.Data {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out, nil
}
