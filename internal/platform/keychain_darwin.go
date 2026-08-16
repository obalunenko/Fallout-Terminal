//go:build darwin

package platform

import (
	"context"
	"errors"

	"github.com/keybase/go-keychain"
)

type darwinCredentialBackend struct{}

func defaultCredentialBackend() credentialBackend { return darwinCredentialBackend{} }

func newDarwinKeychainSecretStoreWithService(service string) *KeychainSecretStore {
	return &KeychainSecretStore{service: service, backend: darwinCredentialBackend{}}
}

func (darwinCredentialBackend) Presence(ctx context.Context, service, account string) (bool, error) {
	if err := contextError(ctx); err != nil {
		return false, err
	}
	query := genericPasswordQuery(service, account)
	query.SetReturnAttributes(true)
	results, err := keychain.QueryItem(query)
	if err != nil {
		return false, translateDarwinCredentialError(err)
	}
	return len(results) > 0, nil
}

func (darwinCredentialBackend) Update(ctx context.Context, service, account string, value []byte) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	query := genericPasswordQuery(service, account)
	update := keychain.NewItem()
	update.SetData(value)
	return translateDarwinCredentialError(keychain.UpdateItem(query, update))
}

func (darwinCredentialBackend) Add(ctx context.Context, service, account string, value []byte) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	item := keychain.NewGenericPassword(service, account, "Fallout Terminal public access", value, "")
	item.SetSynchronizable(keychain.SynchronizableNo)
	return translateDarwinCredentialError(keychain.AddItem(item))
}

func (darwinCredentialBackend) Delete(ctx context.Context, service, account string) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	return translateDarwinCredentialError(keychain.DeleteItem(genericPasswordQuery(service, account)))
}

func (darwinCredentialBackend) Read(ctx context.Context, service, account string) ([]byte, error) {
	if err := contextError(ctx); err != nil {
		return nil, err
	}
	query := genericPasswordQuery(service, account)
	query.SetReturnData(true)
	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, translateDarwinCredentialError(err)
	}
	if len(results) != 1 || len(results[0].Data) == 0 {
		return nil, errCredentialNotFound
	}
	return append([]byte(nil), results[0].Data...), nil
}

func genericPasswordQuery(service, account string) keychain.Item {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(service)
	query.SetAccount(account)
	query.SetSynchronizable(keychain.SynchronizableNo)
	query.SetMatchLimit(keychain.MatchLimitOne)
	return query
}

func translateDarwinCredentialError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, keychain.ErrorItemNotFound):
		return errCredentialNotFound
	case errors.Is(err, keychain.ErrorInteractionNotAllowed):
		return errCredentialLocked
	case errors.Is(err, keychain.ErrorAuthFailed), errors.Is(err, keychain.ErrorNoAccessForItem), errors.Is(err, keychain.ErrorReadOnly):
		return errCredentialDenied
	case errors.Is(err, keychain.ErrorUserCanceled):
		return errCredentialUserCancelled
	default:
		return errCredentialUnavailable
	}
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

var _ credentialBackend = darwinCredentialBackend{}
