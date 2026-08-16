package tunnel

import (
	"context"
	"errors"
)

const (
	DevelopmentNgrokAuthtokenEnvironment = "FALLOUT_NGROK_AUTHTOKEN"
	DevelopmentNgrokDomainEnvironment    = "FALLOUT_NGROK_RESERVED_DOMAIN"
	DevelopmentPlayerUsernameEnvironment = "FALLOUT_PUBLIC_TEST_USERNAME"
	DevelopmentPlayerPasswordEnvironment = "FALLOUT_PUBLIC_TEST_PASSWORD"
)

// EnvironmentLookup is deliberately narrower than process environment access:
// the development adapter asks only for the four approved names above.
type EnvironmentLookup func(string) (string, bool)

// DevelopmentTestPublicAccessOverride overlays process-local test values onto
// the existing settings and scoped-secret boundaries. It owns no persistence
// and is not selected by packaged production composition.
type DevelopmentTestPublicAccessOverride struct {
	settings PublicAccessSettings
	secrets  SecretStore
	lookup   EnvironmentLookup
}

func NewDevelopmentTestPublicAccessOverride(
	settings PublicAccessSettings,
	secrets SecretStore,
	lookup EnvironmentLookup,
) *DevelopmentTestPublicAccessOverride {
	return &DevelopmentTestPublicAccessOverride{settings: settings, secrets: secrets, lookup: lookup}
}

func (override *DevelopmentTestPublicAccessOverride) Load() (PublicAccessPreferences, error) {
	if override == nil || override.settings == nil {
		return DefaultPublicAccessPreferences(), errors.New("public-access settings are unavailable")
	}
	preferences, err := override.settings.Load()
	if err != nil {
		return preferences, err
	}
	if domain, ok := override.nonEmpty(DevelopmentNgrokDomainEnvironment); ok {
		preferences.ReservedDomain = domain
	}
	if username, ok := override.nonEmpty(DevelopmentPlayerUsernameEnvironment); ok {
		preferences.Username = username
	}
	normalized, err := preferences.Normalized()
	if err != nil {
		return DefaultPublicAccessPreferences(), errors.New(ErrorValidation.SafeMessage())
	}
	return normalized, nil
}

func (override *DevelopmentTestPublicAccessOverride) Save(preferences PublicAccessPreferences) error {
	if override == nil || override.settings == nil {
		return errors.New("public-access settings are unavailable")
	}
	return override.settings.Save(preferences)
}

// SaveForMutation keeps environment-supplied visible values ephemeral for
// non-Save commands while still committing revision and presence metadata.
func (override *DevelopmentTestPublicAccessOverride) SaveForMutation(preferences PublicAccessPreferences, persistVisibleOverrides bool) error {
	if override == nil || override.settings == nil {
		return errors.New("public-access settings are unavailable")
	}
	if persistVisibleOverrides {
		return override.settings.Save(preferences)
	}
	domainOverridden := false
	usernameOverridden := false
	if _, ok := override.nonEmpty(DevelopmentNgrokDomainEnvironment); ok {
		domainOverridden = true
	}
	if _, ok := override.nonEmpty(DevelopmentPlayerUsernameEnvironment); ok {
		usernameOverridden = true
	}
	if !domainOverridden && !usernameOverridden {
		return override.settings.Save(preferences)
	}
	stored, err := override.settings.Load()
	if err != nil {
		return err
	}
	persisted := preferences
	if domainOverridden {
		persisted.ReservedDomain = stored.ReservedDomain
	}
	if usernameOverridden {
		persisted.Username = stored.Username
	}
	return override.settings.Save(persisted)
}

func (override *DevelopmentTestPublicAccessOverride) Presence(ctx context.Context, ref SecretRef) (SecretPresence, error) {
	name, ok := developmentSecretEnvironment(ref)
	if !ok {
		return SecretUnknown, errors.New("invalid secret reference")
	}
	if _, present := override.nonEmpty(name); present {
		return SecretPresent, nil
	}
	if override == nil || override.secrets == nil {
		return SecretUnknown, ErrSecretStoreUnavailable
	}
	return override.secrets.Presence(ctx, ref)
}

func (override *DevelopmentTestPublicAccessOverride) Replace(ctx context.Context, ref SecretRef, value []byte) error {
	if override == nil || override.secrets == nil {
		return ErrSecretStoreUnavailable
	}
	return override.secrets.Replace(ctx, ref, value)
}

func (override *DevelopmentTestPublicAccessOverride) Delete(ctx context.Context, ref SecretRef) error {
	if override == nil || override.secrets == nil {
		return ErrSecretStoreUnavailable
	}
	return override.secrets.Delete(ctx, ref)
}

func (override *DevelopmentTestPublicAccessOverride) WithSecrets(
	ctx context.Context,
	refs []SecretRef,
	callback func(*SecretUse) error,
) error {
	if override == nil || callback == nil {
		return ErrSecretStoreUnavailable
	}
	use := &SecretUse{}
	defer use.Clear()
	fallback := make([]SecretRef, 0, len(refs))
	for _, ref := range refs {
		name, ok := developmentSecretEnvironment(ref)
		if !ok {
			return errors.New("invalid secret reference")
		}
		value, present := override.nonEmpty(name)
		if !present {
			fallback = append(fallback, ref)
			continue
		}
		switch ref {
		case ProviderAccountToken:
			use.ProviderToken = []byte(value)
		case PlayerBasicAuthPassword:
			use.PlayerPassword = []byte(value)
		}
	}
	if len(fallback) == 0 {
		return callback(use)
	}
	if override.secrets == nil {
		return ErrSecretStoreUnavailable
	}
	return override.secrets.WithSecrets(ctx, fallback, func(stored *SecretUse) error {
		if stored == nil {
			return ErrSecretStoreUnavailable
		}
		for _, ref := range fallback {
			switch ref {
			case ProviderAccountToken:
				use.ProviderToken = append(use.ProviderToken[:0], stored.ProviderToken...)
			case PlayerBasicAuthPassword:
				use.PlayerPassword = append(use.PlayerPassword[:0], stored.PlayerPassword...)
			}
		}
		return callback(use)
	})
}

func (override *DevelopmentTestPublicAccessOverride) nonEmpty(name string) (string, bool) {
	if override == nil || override.lookup == nil {
		return "", false
	}
	value, ok := override.lookup(name)
	return value, ok && value != ""
}

func developmentSecretEnvironment(ref SecretRef) (string, bool) {
	switch ref {
	case ProviderAccountToken:
		return DevelopmentNgrokAuthtokenEnvironment, true
	case PlayerBasicAuthPassword:
		return DevelopmentPlayerPasswordEnvironment, true
	default:
		return "", false
	}
}

var _ PublicAccessSettings = (*DevelopmentTestPublicAccessOverride)(nil)
var _ SecretStore = (*DevelopmentTestPublicAccessOverride)(nil)
