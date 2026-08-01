package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptSecretRoundTrip(t *testing.T) {
	originalSecret := JWTSecret
	JWTSecret = "test-encryption-key"
	t.Cleanup(func() { JWTSecret = originalSecret })

	encrypted, err := EncryptSecret("sensitive-token")
	require.NoError(t, err)
	require.NotEqual(t, "sensitive-token", encrypted)
	require.NotContains(t, encrypted, "sensitive-token")

	decrypted, err := DecryptSecret(encrypted)
	require.NoError(t, err)
	require.Equal(t, "sensitive-token", decrypted)
}

func TestDecryptSecretRejectsWrongKey(t *testing.T) {
	originalSecret := JWTSecret
	t.Cleanup(func() { JWTSecret = originalSecret })

	JWTSecret = "first-key"
	encrypted, err := EncryptSecret("sensitive-token")
	require.NoError(t, err)

	JWTSecret = "second-key"
	_, err = DecryptSecret(encrypted)
	require.Error(t, err)
}
