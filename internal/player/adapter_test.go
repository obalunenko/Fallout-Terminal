package player

import (
	"strings"
	"testing"

	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestGeneratedMutationFingerprintsAreDeterministicProcedureQualifiedAndUnknownAware(t *testing.T) {
	navigate := &playerv1.NavigateRequest{
		RecognitionHandle: "recognition-1", RequestId: "request-1", BroadcastId: "broadcast-1", TerminalId: "terminal-1",
		Action: &playerv1.NavigateRequest_Enter{Enter: &playerv1.NavigateEnter{NodeId: "docs"}},
	}
	first, err := NavigateFromProto(navigate)
	require.NoError(t, err)
	second, err := NavigateFromProto(proto.Clone(navigate).(*playerv1.NavigateRequest))
	require.NoError(t, err)
	require.Equal(t, first.Command.PayloadFingerprint, second.Command.PayloadFingerprint)
	require.Len(t, first.Command.PayloadFingerprint, 64)
	require.NotContains(t, first.Command.PayloadFingerprint, "recognition-1")

	changed := proto.Clone(navigate).(*playerv1.NavigateRequest)
	changed.Action = &playerv1.NavigateRequest_Entry{Entry: &playerv1.NavigateEntry{NodeId: "docs"}}
	changedMutation, err := NavigateFromProto(changed)
	require.NoError(t, err)
	require.NotEqual(t, first.Command.PayloadFingerprint, changedMutation.Command.PayloadFingerprint)

	withUnknown := proto.Clone(navigate).(*playerv1.NavigateRequest)
	withUnknown.ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 0x01})
	unknownMutation, err := NavigateFromProto(withUnknown)
	require.NoError(t, err)
	require.NotEqual(t, first.Command.PayloadFingerprint, unknownMutation.Command.PayloadFingerprint)

	guess, err := GuessFromProto(&playerv1.GuessRequest{
		RecognitionHandle: "recognition-1", RequestId: "request-1", BroadcastId: "broadcast-1", TerminalId: "terminal-1",
		Target: &playerv1.GuessRequest_WordId{WordId: "docs"},
	})
	require.NoError(t, err)
	require.NotEqual(t, first.Command.PayloadFingerprint, guess.Command.PayloadFingerprint)
	require.False(t, strings.Contains(guess.Command.PayloadFingerprint, "Guess"))
}
