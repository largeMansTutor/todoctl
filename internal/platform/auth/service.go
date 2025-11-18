package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
)

// Service performs API key authorization with optional per-key labels (e.g. issuer).
// It supports plaintext keys and bcrypt-hashed keys. If hashed entries exist, the
// caller must send both X-API-Key-ID and X-API-Key so we can look up the correct hash.
type Service struct {
	allowedPlain map[string]string // key -> label/issuer
	allowedHash  map[string]hashed // id -> hash+label
}

type hashed struct {
	hash  string
	label string
}

// New initializes an auth service from a list of allowed keys and labels. The keys
// should be full secrets (not hashes) for exact match. Labels are optional and used
// only for logging/audit.
func New(keys map[string]string) *Service {
	clean := make(map[string]string, len(keys))
	for k, v := range keys {
		trimmed := strings.TrimSpace(k)
		if trimmed == "" {
			continue
		}
		clean[trimmed] = strings.TrimSpace(v)
	}
	if len(clean) == 0 {
		return nil
	}
	return &Service{allowedPlain: clean}
}

// NewHashed initialises an auth service with bcrypt-hashed keys keyed by id.
func NewHashed(entries map[string]string) *Service {
	if len(entries) == 0 {
		return nil
	}
	clean := make(map[string]hashed, len(entries))
	for id, h := range entries {
		id = strings.TrimSpace(id)
		if id == "" || strings.TrimSpace(h) == "" {
			continue
		}
		clean[id] = hashed{hash: strings.TrimSpace(h), label: id}
	}
	if len(clean) == 0 {
		return nil
	}
	return &Service{allowedHash: clean}
}

// Authorize returns the label (if any) for a valid key, or empty string if unauthorized.
func (s *Service) Authorize(key string, keyID ...string) (string, bool) {
	if s == nil {
		return "", false
	}
	// hashed path if we have hashes and a key ID
	if len(s.allowedHash) > 0 && len(keyID) > 0 && keyID[0] != "" && key != "" {
		if h, ok := s.allowedHash[keyID[0]]; ok {
			sum := sha256.Sum256([]byte(key))
			expected, err := hex.DecodeString(h.hash)
			if err == nil && len(expected) == len(sum) && subtle.ConstantTimeCompare(expected, sum[:]) == 1 {
				return h.label, true
			}
		}
	}
	if key == "" {
		return "", false
	}
	for allowed, label := range s.allowedPlain {
		if subtle.ConstantTimeCompare([]byte(allowed), []byte(key)) == 1 {
			return label, true
		}
	}
	return "", false
}

// AddHashed allows adding a bcrypt hash keyed by id. Useful in tests.
func (s *Service) AddHashed(id, hash, label string) error {
	if s.allowedHash == nil {
		s.allowedHash = make(map[string]hashed)
	}
	if id == "" || hash == "" {
		return errors.New("id and hash required")
	}
	s.allowedHash[id] = hashed{hash: hash, label: label}
	return nil
}
