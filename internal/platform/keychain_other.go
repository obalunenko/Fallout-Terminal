//go:build !darwin

package platform

import (
	"context"

	"github.com/obalunenko/Fallout-Terminal/internal/tunnel"
)

type unavailableCredentialBackend struct{}

func defaultCredentialBackend() credentialBackend { return unavailableCredentialBackend{} }
func (unavailableCredentialBackend) Presence(context.Context, string, string) (bool, error) {
	return false, tunnel.ErrSecretStoreUnavailable
}
func (unavailableCredentialBackend) Update(context.Context, string, string, []byte) error {
	return tunnel.ErrSecretStoreUnavailable
}
func (unavailableCredentialBackend) Add(context.Context, string, string, []byte) error {
	return tunnel.ErrSecretStoreUnavailable
}
func (unavailableCredentialBackend) Delete(context.Context, string, string) error {
	return tunnel.ErrSecretStoreUnavailable
}
func (unavailableCredentialBackend) Read(context.Context, string, string) ([]byte, error) {
	return nil, tunnel.ErrSecretStoreUnavailable
}
