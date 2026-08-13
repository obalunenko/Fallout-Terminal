package playerconfig

import (
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestPlayerConfigContractMapsEveryKnownField(t *testing.T) {
	value := domain.PlayerConfig{Version: 1, Name: "Vault", Roster: []domain.CharacterRosterEntry{
		{ID: "character-1", Name: "Mara"}, {ID: "character-2", Name: "Boone"},
	}}
	semantic, err := PlayerConfigToProto(value)
	require.NoError(t, err)
	require.Equal(t, int32(1), semantic.GetVersion())
	require.Equal(t, "character-2", semantic.GetRoster()[1].GetId())
	roundTrip, err := PlayerConfigFromProto(semantic)
	require.NoError(t, err)
	require.Equal(t, value, roundTrip)
}

func TestPlayerConfigContractRetainsStrictVersionIdentityAndDuplicateValidation(t *testing.T) {
	tests := []domain.PlayerConfig{
		{Version: 2, Name: "Vault", Roster: []domain.CharacterRosterEntry{}},
		{Version: 1, Name: "Vault", Roster: nil},
		{Version: 1, Name: "Vault", Roster: []domain.CharacterRosterEntry{{ID: "same", Name: "Mara"}, {ID: "same", Name: "Boone"}}},
	}
	for _, value := range tests {
		_, err := PlayerConfigToProto(value)
		require.Error(t, err)
	}
}
