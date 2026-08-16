package platform

import (
	"context"
	"errors"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/tunnel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeychainServiceNamesAreStableAndSigningIndependent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "com.vaulttec.fallout-terminal.public-access", KeychainServiceName(true))
	assert.Equal(t, "com.vaulttec.fallout-terminal.dev.public-access", KeychainServiceName(false))
	assert.Equal(t, KeychainServiceName(true), KeychainServiceNameForSigning(true, "Developer ID Application: Example"))
	assert.Equal(t, KeychainServiceName(false), KeychainServiceNameForSigning(false, "unsigned"))
}

func TestKeychainPresenceUsesAttributesOnlyAndFixedAccounts(t *testing.T) {
	t.Parallel()

	backend := newFakeCredentialBackend()
	backend.present[tunnel.ProviderAccountTokenAccount] = true
	store := NewKeychainSecretStore(false, backend)

	presence, err := store.Presence(t.Context(), tunnel.ProviderAccountToken)
	require.NoError(t, err)
	assert.Equal(t, tunnel.SecretPresent, presence)
	presence, err = store.Presence(t.Context(), tunnel.PlayerBasicAuthPassword)
	require.NoError(t, err)
	assert.Equal(t, tunnel.SecretAbsent, presence)
	assert.Equal(t, []string{tunnel.ProviderAccountTokenAccount, tunnel.PlayerPasswordAccount}, backend.presenceAccounts)
	assert.Empty(t, backend.readAccounts, "presence must never request secret data")
}

func TestKeychainReplaceUpdateAddDeleteAndNotFoundSemantics(t *testing.T) {
	t.Parallel()

	backend := newFakeCredentialBackend()
	backend.updateErrors[tunnel.ProviderAccountTokenAccount] = errCredentialNotFound
	store := NewKeychainSecretStore(true, backend)

	input := []byte("synthetic-provider-value")
	require.NoError(t, store.Replace(t.Context(), tunnel.ProviderAccountToken, input))
	assert.Equal(t, []string{tunnel.ProviderAccountTokenAccount}, backend.updateAccounts)
	assert.Equal(t, []string{tunnel.ProviderAccountTokenAccount}, backend.addAccounts)
	assert.Equal(t, []byte("synthetic-provider-value"), input)

	delete(backend.updateErrors, tunnel.ProviderAccountTokenAccount)
	require.NoError(t, store.Replace(t.Context(), tunnel.ProviderAccountToken, []byte("replacement-value")))
	assert.Len(t, backend.addAccounts, 1, "existing items update without a duplicate add")

	backend.deleteErrors[tunnel.ProviderAccountTokenAccount] = errCredentialNotFound
	require.NoError(t, store.Delete(t.Context(), tunnel.ProviderAccountToken))
	require.Error(t, store.Replace(t.Context(), tunnel.SecretRef(99), input))
}

func TestKeychainScopedReadClearsReturnedBuffers(t *testing.T) {
	t.Parallel()

	backend := newFakeCredentialBackend()
	backend.values[tunnel.ProviderAccountTokenAccount] = []byte("synthetic-provider-value")
	backend.values[tunnel.PlayerPasswordAccount] = []byte("synthetic-player-value")
	store := NewKeychainSecretStore(false, backend)

	var captured *tunnel.SecretUse
	require.NoError(t, store.WithSecrets(t.Context(), []tunnel.SecretRef{
		tunnel.ProviderAccountToken, tunnel.PlayerBasicAuthPassword,
	}, func(use *tunnel.SecretUse) error {
		captured = use
		assert.NotEmpty(t, use.ProviderToken)
		assert.NotEmpty(t, use.PlayerPassword)
		return nil
	}))
	require.NotNil(t, captured)
	assert.Empty(t, captured.ProviderToken)
	assert.Empty(t, captured.PlayerPassword)
	assert.Equal(t, []string{tunnel.ProviderAccountTokenAccount, tunnel.PlayerPasswordAccount}, backend.readAccounts)
}

func TestKeychainFailuresMapToStableSecretFreeCategories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		backend error
		want    error
	}{
		{name: "locked", backend: errCredentialLocked, want: tunnel.ErrSecretStoreLocked},
		{name: "denied", backend: errCredentialDenied, want: tunnel.ErrSecretStoreDenied},
		{name: "unavailable", backend: errCredentialUnavailable, want: tunnel.ErrSecretStoreUnavailable},
		{name: "cancelled", backend: errCredentialUserCancelled, want: tunnel.ErrSecretStoreUserCancelled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			backend := newFakeCredentialBackend()
			backend.presenceErrors[tunnel.ProviderAccountTokenAccount] = test.backend
			store := NewKeychainSecretStore(false, backend)
			presence, err := store.Presence(t.Context(), tunnel.ProviderAccountToken)
			assert.Equal(t, tunnel.SecretUnknown, presence)
			require.ErrorIs(t, err, test.want)
			assert.NotContains(t, err.Error(), "synthetic-provider-value")
			assert.NotContains(t, err.Error(), "OSStatus")
		})
	}
}

type fakeCredentialBackend struct {
	present map[string]bool
	values  map[string][]byte

	presenceErrors map[string]error
	updateErrors   map[string]error
	addErrors      map[string]error
	deleteErrors   map[string]error
	readErrors     map[string]error

	presenceAccounts []string
	updateAccounts   []string
	addAccounts      []string
	deleteAccounts   []string
	readAccounts     []string
}

func newFakeCredentialBackend() *fakeCredentialBackend {
	return &fakeCredentialBackend{
		present: make(map[string]bool), values: make(map[string][]byte),
		presenceErrors: make(map[string]error), updateErrors: make(map[string]error),
		addErrors: make(map[string]error), deleteErrors: make(map[string]error), readErrors: make(map[string]error),
	}
}

func (backend *fakeCredentialBackend) Presence(_ context.Context, _, account string) (bool, error) {
	backend.presenceAccounts = append(backend.presenceAccounts, account)
	return backend.present[account], backend.presenceErrors[account]
}

func (backend *fakeCredentialBackend) Update(_ context.Context, _, account string, value []byte) error {
	backend.updateAccounts = append(backend.updateAccounts, account)
	if err := backend.updateErrors[account]; err != nil {
		return err
	}
	backend.values[account] = append([]byte(nil), value...)
	backend.present[account] = true
	return nil
}

func (backend *fakeCredentialBackend) Add(_ context.Context, _, account string, value []byte) error {
	backend.addAccounts = append(backend.addAccounts, account)
	if err := backend.addErrors[account]; err != nil {
		return err
	}
	backend.values[account] = append([]byte(nil), value...)
	backend.present[account] = true
	return nil
}

func (backend *fakeCredentialBackend) Delete(_ context.Context, _, account string) error {
	backend.deleteAccounts = append(backend.deleteAccounts, account)
	if err := backend.deleteErrors[account]; err != nil {
		return err
	}
	delete(backend.values, account)
	delete(backend.present, account)
	return nil
}

func (backend *fakeCredentialBackend) Read(_ context.Context, _, account string) ([]byte, error) {
	backend.readAccounts = append(backend.readAccounts, account)
	if err := backend.readErrors[account]; err != nil {
		return nil, err
	}
	value, ok := backend.values[account]
	if !ok {
		return nil, errCredentialNotFound
	}
	return append([]byte(nil), value...), nil
}

var _ credentialBackend = (*fakeCredentialBackend)(nil)
var _ = errors.Is
