package platform

import (
	"context"
	"errors"

	"github.com/obalunenko/Fallout-Terminal/internal/tunnel"
)

const keychainServiceBase = "com.vaulttec.fallout-terminal"

var (
	errCredentialNotFound      = errors.New("credential item not found")
	errCredentialLocked        = errors.New("credential store locked")
	errCredentialDenied        = errors.New("credential store access denied")
	errCredentialUnavailable   = errors.New("credential store unavailable")
	errCredentialUserCancelled = errors.New("credential store interaction cancelled")
)

type credentialBackend interface {
	Presence(context.Context, string, string) (bool, error)
	Update(context.Context, string, string, []byte) error
	Add(context.Context, string, string, []byte) error
	Delete(context.Context, string, string) error
	Read(context.Context, string, string) ([]byte, error)
}

type KeychainSecretStore struct {
	service string
	backend credentialBackend
}

func KeychainServiceName(production bool) string {
	if production {
		return keychainServiceBase + ".public-access"
	}
	return keychainServiceBase + ".dev.public-access"
}

func KeychainServiceNameForSigning(production bool, _ string) string {
	return KeychainServiceName(production)
}

func NewKeychainSecretStore(production bool, backend credentialBackend) *KeychainSecretStore {
	if backend == nil {
		backend = defaultCredentialBackend()
	}
	return &KeychainSecretStore{service: KeychainServiceName(production), backend: backend}
}

func NewPlatformKeychainSecretStore(production bool) tunnel.SecretStore {
	return NewKeychainSecretStore(production, nil)
}

func (store *KeychainSecretStore) Presence(ctx context.Context, ref tunnel.SecretRef) (tunnel.SecretPresence, error) {
	account, err := keychainAccount(ref)
	if err != nil {
		return tunnel.SecretUnknown, err
	}
	if store == nil || store.backend == nil {
		return tunnel.SecretUnknown, tunnel.ErrSecretStoreUnavailable
	}
	present, err := store.backend.Presence(ctx, store.service, account)
	if err != nil {
		return tunnel.SecretUnknown, mapCredentialError(err)
	}
	if present {
		return tunnel.SecretPresent, nil
	}
	return tunnel.SecretAbsent, nil
}

func (store *KeychainSecretStore) Replace(ctx context.Context, ref tunnel.SecretRef, value []byte) error {
	account, err := keychainAccount(ref)
	if err != nil {
		return err
	}
	if store == nil || store.backend == nil {
		return tunnel.ErrSecretStoreUnavailable
	}
	temporary := append([]byte(nil), value...)
	defer clear(temporary)
	err = store.backend.Update(ctx, store.service, account, temporary)
	if errors.Is(err, errCredentialNotFound) {
		err = store.backend.Add(ctx, store.service, account, temporary)
	}
	if err != nil {
		return mapCredentialError(err)
	}
	return nil
}

func (store *KeychainSecretStore) Delete(ctx context.Context, ref tunnel.SecretRef) error {
	account, err := keychainAccount(ref)
	if err != nil {
		return err
	}
	if store == nil || store.backend == nil {
		return tunnel.ErrSecretStoreUnavailable
	}
	err = store.backend.Delete(ctx, store.service, account)
	if errors.Is(err, errCredentialNotFound) {
		return nil
	}
	if err != nil {
		return mapCredentialError(err)
	}
	return nil
}

func (store *KeychainSecretStore) WithSecrets(ctx context.Context, refs []tunnel.SecretRef, callback func(*tunnel.SecretUse) error) error {
	if store == nil || store.backend == nil || callback == nil {
		return tunnel.ErrSecretStoreUnavailable
	}
	use := &tunnel.SecretUse{}
	defer use.Clear()
	for _, ref := range refs {
		account, err := keychainAccount(ref)
		if err != nil {
			return err
		}
		value, err := store.backend.Read(ctx, store.service, account)
		if err != nil {
			return mapCredentialError(err)
		}
		switch ref {
		case tunnel.ProviderAccountToken:
			clear(use.ProviderToken)
			use.ProviderToken = value
		case tunnel.PlayerBasicAuthPassword:
			clear(use.PlayerPassword)
			use.PlayerPassword = value
		}
	}
	return callback(use)
}

func keychainAccount(ref tunnel.SecretRef) (string, error) {
	if !ref.Valid() {
		return "", errors.New("invalid secret reference")
	}
	return ref.Account(), nil
}

func mapCredentialError(err error) error {
	switch {
	case errors.Is(err, errCredentialNotFound):
		return tunnel.ErrSecretStoreUnavailable
	case errors.Is(err, errCredentialLocked):
		return tunnel.ErrSecretStoreLocked
	case errors.Is(err, errCredentialDenied):
		return tunnel.ErrSecretStoreDenied
	case errors.Is(err, errCredentialUserCancelled):
		return tunnel.ErrSecretStoreUserCancelled
	default:
		return tunnel.ErrSecretStoreUnavailable
	}
}

var _ tunnel.SecretStore = (*KeychainSecretStore)(nil)
