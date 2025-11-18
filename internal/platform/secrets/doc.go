// Package secrets provides minimal clients for external secret managers. It is
// intentionally slim to avoid heavy dependencies while allowing production
// builds to fetch sensitive configuration from supported backends (e.g. Vault).
package secrets
