package player

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/go-cmp/cmp"
	"github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	"github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1/playerv1connect"
	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestConnectSubscribeBeginsWithCompleteSnapshotAndSelectsCharacter(t *testing.T) {
	var service *ConnectService
	coordinator := newConnectTestCoordinator(t, func(effect control.Effect) {
		if service != nil {
			service.PublishEffect(effect)
		}
	})
	var err error
	service, err = NewConnectService(ConnectServiceConfig{Coordinator: coordinator})
	require.NoError(t, err)

	path, handler := playerv1connect.NewPlayerServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := playerv1connect.NewPlayerServiceClient(server.Client(), server.URL)
	streamContext, cancelStream := context.WithCancel(t.Context())
	t.Cleanup(cancelStream)
	stream, err := client.Subscribe(streamContext, connect.NewRequest(&playerv1.SubscribeRequest{}))
	require.NoError(t, err)
	require.True(t, stream.Receive())
	first := stream.Msg()
	snapshot := first.GetSnapshot()
	require.NotNil(t, snapshot)
	require.NotEmpty(t, snapshot.GetRecognitionHandle())
	require.NotNil(t, snapshot.GetPlayerState())
	require.NotNil(t, snapshot.GetTerminalPresentation().GetNoLiveTerminal())
	require.Nil(t, first.GetUpdate())

	wantPlayer := &playerv1.PlayerState{
		LogicalSessionId: snapshot.GetPlayerState().GetLogicalSessionId(),
		FallbackName:     snapshot.GetPlayerState().GetFallbackName(),
		Role:             playerv1.PlayerRole_PLAYER_ROLE_UNASSIGNED,
		Phase:            playerv1.PlayerPhase_PLAYER_PHASE_SELECTING,
		BroadcastId:      pointerTo("broadcast-1"),
		Roster: []*playerv1.RosterEntry{{
			CharacterId:  "character-1",
			DisplayName:  "Lucy",
			Availability: playerv1.RosterAvailability_ROSTER_AVAILABILITY_AVAILABLE,
		}},
	}
	require.Empty(t, cmp.Diff(wantPlayer, snapshot.GetPlayerState(), protocmp.Transform()))

	response, err := client.SelectCharacter(t.Context(), connect.NewRequest(&playerv1.SelectCharacterRequest{
		RecognitionHandle: snapshot.GetRecognitionHandle(),
		RequestId:         "request-1",
		BroadcastId:       "broadcast-1",
		CharacterId:       "character-1",
	}))
	require.NoError(t, err)
	require.True(t, response.Msg.GetAccepted())
	require.Equal(t, playerv1.ActionReason_ACTION_REASON_ACCEPTED, response.Msg.GetReason())
	require.Greater(t, response.Msg.GetRevision(), snapshot.GetRevision())
	require.True(t, stream.Receive())
	update := stream.Msg().GetUpdate()
	require.NotNil(t, update)
	require.Equal(t, response.Msg.GetRevision(), update.GetRevision())
	require.NotNil(t, update.GetPlayerState())
}

func TestConnectSubscribeHandleMatrixRejectsInvalidWithoutCanonicalEffects(t *testing.T) {
	tests := []struct {
		name   string
		handle string
	}{
		{name: "blank", handle: ""},
		{name: "whitespace", handle: "not valid"},
		{name: "oversized", handle: strings.Repeat("a", domain.MaxRecognitionHandleBytes+1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			coordinator := newConnectTestCoordinator(t)
			beforeRevision := coordinator.Revision()
			service, err := NewConnectService(ConnectServiceConfig{Coordinator: coordinator})
			require.NoError(t, err)
			path, handler := playerv1connect.NewPlayerServiceHandler(service)
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)
			client := playerv1connect.NewPlayerServiceClient(server.Client(), server.URL)

			stream, err := client.Subscribe(t.Context(), connect.NewRequest(&playerv1.SubscribeRequest{RecognitionHandle: &test.handle}))
			if err == nil {
				require.False(t, stream.Receive())
				err = stream.Err()
			}
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.Equal(t, beforeRevision, coordinator.Revision())
			require.Zero(t, coordinator.ActiveStreamCount())
		})
	}
}

func TestTypedSharedActionHandlersRejectUnassignedSessionWithoutMutation(t *testing.T) {
	coordinator := newConnectTestCoordinator(t)
	snapshot, err := coordinator.AttachSubscription("connect-test-stream", nil)
	require.NoError(t, err)
	service, err := NewConnectService(ConnectServiceConfig{Coordinator: coordinator})
	require.NoError(t, err)
	beforeRevision := coordinator.Revision()

	tests := []struct {
		name string
		call func() (*connect.Response[playerv1.ActionResult], error)
	}{
		{
			name: "navigate",
			call: func() (*connect.Response[playerv1.ActionResult], error) {
				return service.Navigate(t.Context(), connect.NewRequest(&playerv1.NavigateRequest{
					RecognitionHandle: string(snapshot.RecognitionHandle), RequestId: "navigate-1",
					BroadcastId: "broadcast-1", TerminalId: "terminal-1",
					Action: &playerv1.NavigateRequest_Back{Back: &playerv1.NavigateBack{}},
				}))
			},
		},
		{
			name: "guess",
			call: func() (*connect.Response[playerv1.ActionResult], error) {
				return service.Guess(t.Context(), connect.NewRequest(&playerv1.GuessRequest{
					RecognitionHandle: string(snapshot.RecognitionHandle), RequestId: "guess-1",
					BroadcastId: "broadcast-1", TerminalId: "terminal-1",
					Target: &playerv1.GuessRequest_WordId{WordId: "word-1"},
				}))
			},
		},
		{
			name: "activate pattern",
			call: func() (*connect.Response[playerv1.ActionResult], error) {
				return service.ActivatePattern(t.Context(), connect.NewRequest(&playerv1.ActivatePatternRequest{
					RecognitionHandle: string(snapshot.RecognitionHandle), RequestId: "pattern-1",
					BroadcastId: "broadcast-1", TerminalId: "terminal-1", PatternId: "opaque-pattern-1",
				}))
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, err := test.call()
			require.NoError(t, err)
			require.False(t, response.Msg.GetAccepted())
			require.Equal(t, playerv1.ActionReason_ACTION_REASON_UNASSIGNED, response.Msg.GetReason())
			require.Equal(t, beforeRevision, response.Msg.GetRevision())
		})
	}
	require.Equal(t, beforeRevision, coordinator.Revision())
}

func TestTypedHandlersClassifyStructuralDomainAndBoundaryErrorsSafely(t *testing.T) {
	coordinator := newConnectTestCoordinator(t)
	service, err := NewConnectService(ConnectServiceConfig{Coordinator: coordinator})
	require.NoError(t, err)
	beforeRevision := coordinator.Revision()
	secret := "private-recognition-handle"

	structural := []struct {
		name string
		call func() error
	}{
		{
			name: "blank recognition",
			call: func() error {
				_, err := service.Navigate(t.Context(), connect.NewRequest(&playerv1.NavigateRequest{
					RequestId: "request-1", BroadcastId: "broadcast-1", TerminalId: "terminal-1",
					Action: &playerv1.NavigateRequest_Back{Back: &playerv1.NavigateBack{}},
				}))
				return err
			},
		},
		{
			name: "oversized request identity",
			call: func() error {
				_, err := service.SelectCharacter(t.Context(), connect.NewRequest(&playerv1.SelectCharacterRequest{
					RecognitionHandle: secret, RequestId: strings.Repeat("r", domain.MaxRequestIDBytes+1),
					BroadcastId: "broadcast-1", CharacterId: "character-1",
				}))
				return err
			},
		},
		{
			name: "missing navigation variant",
			call: func() error {
				_, err := service.Navigate(t.Context(), connect.NewRequest(&playerv1.NavigateRequest{
					RecognitionHandle: secret, RequestId: "request-2", BroadcastId: "broadcast-1", TerminalId: "terminal-1",
				}))
				return err
			},
		},
		{
			name: "illegal filler coordinates",
			call: func() error {
				_, err := service.Guess(t.Context(), connect.NewRequest(&playerv1.GuessRequest{
					RecognitionHandle: secret, RequestId: "request-3", BroadcastId: "broadcast-1", TerminalId: "terminal-1",
					Target: &playerv1.GuessRequest_Filler{Filler: &playerv1.FillerTarget{Column: 2}},
				}))
				return err
			},
		},
	}
	for _, test := range structural {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()
			require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
			require.NotContains(t, err.Error(), secret)
			require.Equal(t, beforeRevision, coordinator.Revision())
		})
	}

	unknown, err := service.ActivatePattern(t.Context(), connect.NewRequest(&playerv1.ActivatePatternRequest{
		RecognitionHandle: "unknown-but-well-formed", RequestId: "request-4", BroadcastId: "broadcast-1",
		TerminalId: "terminal-1", PatternId: "pattern-1",
	}))
	require.NoError(t, err)
	require.False(t, unknown.Msg.GetAccepted())
	require.Equal(t, playerv1.ActionReason_ACTION_REASON_INVALID_SESSION, unknown.Msg.GetReason())
	require.Equal(t, beforeRevision, coordinator.Revision())

	require.Equal(t, playerv1.ActionReason_ACTION_REASON_UNSPECIFIED, ActionResultToProto(domain.ActionResult{Reason: "private-reason"}).GetReason())
	require.Equal(t, connect.CodeCanceled, connect.CodeOf(mapStreamError(context.Canceled)))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(mapStreamError(errors.New("private dependency detail"))))
	require.NotContains(t, mapStreamError(errors.New("private dependency detail")).Error(), "private dependency detail")
	require.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(publicConnectError(ErrResourceExhausted)))
}

func TestPublicDescriptorAndProceduresExcludeEveryPrivateDesktopCapability(t *testing.T) {
	var symbols []string
	var collectMessages func(protoreflect.MessageDescriptors)
	collectMessages = func(messages protoreflect.MessageDescriptors) {
		for index := 0; index < messages.Len(); index++ {
			message := messages.Get(index)
			symbols = append(symbols, string(message.FullName()))
			collectMessages(message.Messages())
		}
	}
	file := playerv1.File_fallout_terminal_player_v1_player_proto
	collectMessages(file.Messages())
	for index := 0; index < file.Services().Len(); index++ {
		service := file.Services().Get(index)
		symbols = append(symbols, string(service.FullName()))
		for methodIndex := 0; methodIndex < service.Methods().Len(); methodIndex++ {
			symbols = append(symbols, string(service.Methods().Get(methodIndex).FullName()))
		}
	}
	publicSurface := strings.ToLower(strings.Join(symbols, "\n") + playerv1connect.PlayerServiceName +
		playerv1connect.PlayerServiceSubscribeProcedure + playerv1connect.PlayerServiceSelectCharacterProcedure +
		playerv1connect.PlayerServiceNavigateProcedure + playerv1connect.PlayerServiceGuessProcedure +
		playerv1connect.PlayerServiceActivatePatternProcedure + playerv1connect.PlayerServiceSoundManifestProcedure)
	for _, forbidden := range []string{
		"desktop", "dialog", "openurl", "forcehacksuccess", "resetfailedhack", "runtimestatus",
		"serverinformation", "credential", "secretword", "logicalsessionstate", "coordinationstate",
	} {
		require.NotContains(t, publicSurface, forbidden)
	}
}

func newConnectTestCoordinator(t *testing.T, publish ...func(control.Effect)) *control.Service {
	t.Helper()
	ids := testutil.NewFakeOpaqueIDSource("broadcast-1", "session-1", "recognition-1")
	var enqueue func(control.Effect)
	if len(publish) > 0 {
		enqueue = publish[0]
	}
	coordinator := control.New(control.Config{IDs: ids, Enqueue: enqueue})
	_, err := coordinator.InstallPlayerConfig(domain.PlayerConfigHandle{Path: "/private/player.json", Version: 1, Name: "Vault 33"}, []domain.CharacterRosterEntry{{ID: "character-1", Name: "Lucy"}})
	require.NoError(t, err)
	_, err = coordinator.StartBroadcast()
	require.NoError(t, err)
	return coordinator
}

func pointerTo[T any](value T) *T {
	return &value
}
