package tunnel

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicAccessPreferencesValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		preferences PublicAccessPreferences
		wantDomain  string
		wantErr     string
	}{
		{name: "safe defaults", preferences: DefaultPublicAccessPreferences()},
		{name: "normalized domain", preferences: PublicAccessPreferences{Version: 1, Username: " players ", ReservedDomain: "Vault.Example."}, wantDomain: "vault.example"},
		{name: "blank domain requests random URL", preferences: PublicAccessPreferences{Version: 1, Username: "players", ReservedDomain: "  "}},
		{name: "future version", preferences: PublicAccessPreferences{Version: 2, Username: "players"}, wantErr: "version"},
		{name: "empty username", preferences: PublicAccessPreferences{Version: 1}, wantErr: "username"},
		{name: "username newline", preferences: PublicAccessPreferences{Version: 1, Username: "play\ners"}, wantErr: "username"},
		{name: "domain scheme", preferences: PublicAccessPreferences{Version: 1, Username: "players", ReservedDomain: "https://vault.example"}, wantErr: "domain"},
		{name: "domain port", preferences: PublicAccessPreferences{Version: 1, Username: "players", ReservedDomain: "vault.example:443"}, wantErr: "domain"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			normalized, err := test.preferences.Normalized()
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantDomain, normalized.ReservedDomain)
			assert.Equal(t, "players", normalized.Username)
		})
	}
}

func TestPublicAccessLifecycleAndStatusInvariants(t *testing.T) {
	t.Parallel()

	for _, state := range []LifecycleState{LifecycleDisabled, LifecycleStarting, LifecycleReady, LifecycleStopping, LifecycleFailed} {
		assert.True(t, state.Valid(), state)
	}
	assert.False(t, LifecycleState(0).Valid())
	assert.False(t, LifecycleState(99).Valid())

	tests := []struct {
		name    string
		status  PublicAccessStatus
		wantErr string
	}{
		{name: "disabled", status: PublicAccessStatus{State: LifecycleDisabled, Generation: 1}},
		{name: "ready", status: PublicAccessStatus{State: LifecycleReady, Generation: 2, SettingsRevision: 3, PublicURL: "https://vault.example"}},
		{name: "failed", status: PublicAccessStatus{State: LifecycleFailed, Generation: 3, ErrorCategory: ErrorProviderFailure, ErrorMessage: "Provider unavailable."}},
		{name: "ready without URL", status: PublicAccessStatus{State: LifecycleReady, Generation: 2}, wantErr: "URL"},
		{name: "URL while stopped", status: PublicAccessStatus{State: LifecycleDisabled, Generation: 2, PublicURL: "https://vault.example"}, wantErr: "URL"},
		{name: "error outside failed", status: PublicAccessStatus{State: LifecycleStarting, ErrorCategory: ErrorTimeout}, wantErr: "error"},
		{name: "unclassified failure", status: PublicAccessStatus{State: LifecycleFailed}, wantErr: "category"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.status.Validate()
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestPublicAccessErrorCategoriesAreClosedAndRedacted(t *testing.T) {
	t.Parallel()

	categories := []ErrorCategory{
		ErrorValidation, ErrorSettingsCorrupt, ErrorSecretStoreLocked, ErrorSecretStoreDenied,
		ErrorSecretStoreUnavailable, ErrorCredentialMissing, ErrorProviderAuthentication,
		ErrorDomainUnavailable, ErrorNetworkUnavailable, ErrorTimeout, ErrorProviderFailure,
		ErrorShutdownTimeout, ErrorConflict,
	}
	for _, category := range categories {
		assert.True(t, category.Valid(), category)
		assert.NotEmpty(t, category.SafeMessage())
	}
	assert.False(t, ErrorCategory(0).Valid())
	assert.False(t, ErrorCategory(99).Valid())
}

func TestNormalizeEndpointURLRequiresExactHTTPSAuthority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		raw            string
		reservedDomain string
		wantURL        string
		wantHost       string
		wantErr        bool
	}{
		{name: "random host", raw: "https://Random.Ngrok.App./", wantURL: "https://random.ngrok.app", wantHost: "random.ngrok.app"},
		{name: "exact reserved", raw: "https://vault.example", reservedDomain: "vault.example", wantURL: "https://vault.example", wantHost: "vault.example"},
		{name: "wrong reserved", raw: "https://other.example", reservedDomain: "vault.example", wantErr: true},
		{name: "http", raw: "http://vault.example", wantErr: true},
		{name: "userinfo", raw: "https://user@vault.example", wantErr: true},
		{name: "path", raw: "https://vault.example/player", wantErr: true},
		{name: "query", raw: "https://vault.example?q=1", wantErr: true},
		{name: "fragment", raw: "https://vault.example#player", wantErr: true},
		{name: "port", raw: "https://vault.example:443", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			canonicalURL, host, err := NormalizeEndpointURL(test.raw, test.reservedDomain)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantURL, canonicalURL)
			assert.Equal(t, test.wantHost, host)
		})
	}
}

func TestIntentVersionRequiresMonotonicGenerationAndExactRevision(t *testing.T) {
	t.Parallel()

	current := IntentVersion{Generation: 9, SettingsRevision: 4}
	assert.True(t, current.Matches(IntentVersion{Generation: 9, SettingsRevision: 4}))
	assert.False(t, current.Matches(IntentVersion{Generation: 8, SettingsRevision: 4}))
	assert.False(t, current.Matches(IntentVersion{Generation: 9, SettingsRevision: 3}))
	assert.True(t, IntentVersion{Generation: 10, SettingsRevision: 4}.NewerThan(current))
	assert.False(t, IntentVersion{Generation: 9, SettingsRevision: 5}.NewerThan(current), "settings writes do not substitute for lifecycle generation")
}

func TestSecretAndProviderRequestTypesHaveNoFormattingSurface(t *testing.T) {
	t.Parallel()

	stringer := reflect.TypeFor[fmt.Stringer]()
	goStringer := reflect.TypeFor[fmt.GoStringer]()
	for _, value := range []any{
		TunnelStartRequest{}, SecretUse{},
	} {
		typeOf := reflect.TypeOf(value)
		assert.False(t, typeOf.Implements(stringer), "%s must not implement String", typeOf)
		assert.False(t, typeOf.Implements(goStringer), "%s must not implement GoString", typeOf)
	}

	request := TunnelStartRequest{
		UpstreamURL:  "http://127.0.0.1:41000",
		AccountToken: []byte("ephemeral-test-value"),
		Timeout:      30 * time.Second,
	}
	request.Clear()
	assert.Equal(t, []byte(nil), request.AccountToken)
	requestType := reflect.TypeOf(request)
	fieldNames := make([]string, 0, requestType.NumField())
	for index := range requestType.NumField() {
		fieldNames = append(fieldNames, requestType.Field(index).Name)
	}
	assert.Equal(t, []string{"UpstreamURL", "ReservedDomain", "AccountToken", "Timeout"}, fieldNames)
}
