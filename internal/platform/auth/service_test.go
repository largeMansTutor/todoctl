package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected bool
	}{
		{name: "returns nil when empty", input: map[string]string{}, expected: false},
		{name: "trims keys and values", input: map[string]string{" key ": " label "}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := New(tt.input)
			if tt.expected {
				require.NotNil(t, svc)
			} else {
				assert.Nil(t, svc)
			}
		})
	}
}

func TestAuthorizePlain(t *testing.T) {
	svc := New(map[string]string{"secret": "label"})
	label, ok := svc.Authorize("secret")
	assert.True(t, ok)
	assert.Equal(t, "label", label)

	_, ok = svc.Authorize("bad")
	assert.False(t, ok)

	_, ok = (*Service)(nil).Authorize("secret")
	assert.False(t, ok)
}

func TestAuthorizeHashed(t *testing.T) {
	raw := "super-secret"
	sum := sha256.Sum256([]byte(raw))
	svc := NewHashed(map[string]string{"id1": hex.EncodeToString(sum[:])})

	label, ok := svc.Authorize(raw, "id1")
	assert.True(t, ok)
	assert.Equal(t, "id1", label)

	_, ok = svc.Authorize("wrong", "id1")
	assert.False(t, ok)

	_, ok = svc.Authorize(raw, "unknown")
	assert.False(t, ok)
}

func TestAddHashed(t *testing.T) {
	svc := New(map[string]string{"secret": ""})
	err := svc.AddHashed("", "", "")
	assert.Error(t, err)

	sum := sha256.Sum256([]byte("another"))
	err = svc.AddHashed("id2", hex.EncodeToString(sum[:]), "lbl")
	require.NoError(t, err)

	_, ok := svc.Authorize("another", "id2")
	assert.True(t, ok)
}
