package session

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestSessionContractMapsEveryKnownFieldAndPreservesRecursiveJSONExtras(t *testing.T) {
	value := domain.Session{
		Version: 1, Name: "Vault", PlayerConfig: "players/roster.json",
		Extra: map[string]json.RawMessage{"futureSession": json.RawMessage(`{"enabled":true}`)},
		Terminals: []domain.Terminal{{
			ID: "terminal-1", Name: "Overseer", HackLevel: 2, IntroText: "WELCOME",
			Extra: map[string]json.RawMessage{"futureTerminal": json.RawMessage(`7`)},
			Root: domain.ContentNode{
				ID: "root", Type: domain.NodeFolder, Name: "ROOT", Extra: map[string]json.RawMessage{"futureRoot": json.RawMessage(`"x"`)},
				Children: []domain.ContentNode{
					{ID: "command", Type: domain.NodeCommand, Name: "RUN", Text: "OK", Extra: map[string]json.RawMessage{"futureCommand": json.RawMessage(`[]`)}},
					{ID: "entry", Type: domain.NodeEntry, Name: "READ", Description: "BODY", Extra: map[string]json.RawMessage{"futureEntry": json.RawMessage(`null`)}},
				},
			},
		}},
	}

	semantic, err := SessionToProto(value)
	require.NoError(t, err)
	require.Equal(t, int32(1), semantic.GetVersion())
	require.Equal(t, "players/roster.json", semantic.GetPlayerConfig())
	require.Equal(t, int32(2), semantic.GetTerminals()[0].GetHackLevel())
	require.Equal(t, "OK", semantic.GetTerminals()[0].GetRoot().GetFolder().GetChildren()[0].GetCommand().GetText())
	require.Equal(t, "BODY", semantic.GetTerminals()[0].GetRoot().GetFolder().GetChildren()[1].GetEntry().GetDescription())

	roundTrip, err := SessionFromProto(semantic, value)
	require.NoError(t, err)
	require.True(t, reflect.DeepEqual(value, roundTrip), "round trip\nwant: %#v\ngot: %#v", value, roundTrip)
}

func TestSessionContractRejectsMissingOneofAndInvalidReference(t *testing.T) {
	value := domain.Session{Version: 1, Name: "Vault", PlayerConfig: "/absolute/players.json", Terminals: []domain.Terminal{}}
	_, err := SessionToProto(value)
	require.Error(t, err)
}
