package main

import (
	"strings"

	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/auth"
	"github.com/thetrollfarmercodes/todoctl/todo/internal/platform/config"
)

// authFromConfig builds an auth.Service using either multiple allowed keys
// (APP_ALLOWED_API_KEYS formatted as "key:label,key2:label2"), hashed keys
// (APP_ALLOWED_API_KEYS_HASHED formatted as "id:bcryptHash(:label)") or the legacy
// APP_API_KEY when present. If nothing is provided, it returns nil to keep
// auth optional for local dev.
func authFromConfig(cfg *config.Config) *auth.Service {
	keyMap := make(map[string]string)
	if cfg.AllowedAPIKeys != "" {
		entries := strings.Split(cfg.AllowedAPIKeys, ",")
		for _, e := range entries {
			parts := strings.SplitN(strings.TrimSpace(e), ":", 2)
			if len(parts) == 0 || parts[0] == "" {
				continue
			}
			label := ""
			if len(parts) == 2 {
				label = strings.TrimSpace(parts[1])
			}
			keyMap[parts[0]] = label
		}
	}
	if cfg.APIKey != "" {
		keyMap[cfg.APIKey] = "default"
	}
	var svc *auth.Service
	if cfg.AllowedAPIKeysHashed != "" {
		hashMap := make(map[string]string)
		entries := strings.Split(cfg.AllowedAPIKeysHashed, ",")
		for _, e := range entries {
			parts := strings.Split(strings.TrimSpace(e), ":")
			// id:sha256Hex or id:sha256Hex:label
			if len(parts) < 2 {
				continue
			}
			id := strings.TrimSpace(parts[0])
			hash := strings.TrimSpace(parts[1])
			if id == "" || hash == "" {
				continue
			}
			label := id
			if len(parts) >= 3 {
				label = strings.TrimSpace(strings.Join(parts[2:], ":"))
			}
			hashMap[id] = hash
			if svc == nil {
				svc = auth.NewHashed(hashMap)
			}
			_ = svc.AddHashed(id, hash, label)
		}
	}
	if svc == nil && len(keyMap) > 0 {
		svc = auth.New(keyMap)
	}
	return svc
}
