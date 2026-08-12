package main

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	controlservice "github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	liveservice "github.com/obalunenko/Fallout-Terminal/internal/live"
	playerconfigservice "github.com/obalunenko/Fallout-Terminal/internal/playerconfig"
	sessionservice "github.com/obalunenko/Fallout-Terminal/internal/session"
	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
)

func TestApplicationStartsPlayerBeforePublishingReady(t *testing.T) {
	recorder := &callRecorder{}
	player := &recordingPlayerServer{
		recorder: recorder,
		info: domain.ServerInfo{
			IP: "192.0.2.10", Port: 3690, URL: "http://192.0.2.10:3690",
		},
	}
	events := &recordingEventSink{recorder: recorder}
	desktop := &recordingDesktop{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{
		Player:  player,
		Events:  events,
		Desktop: desktop,
	})

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got, want := recorder.Calls(), []string{"player:start", "event:server-info", "desktop:ready"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("startup calls = %v, want %v", got, want)
	}
	status := app.GetRuntimeStatus()
	if status.ServerInfo == nil || status.ServerInfo.Port != 3690 || status.StartupError != "" {
		t.Fatalf("runtime status = %#v", status)
	}
}

func TestPlayerConfigCommandsAssociateBeforeInstallingRoster(t *testing.T) {
	t.Parallel()

	coordination := &recordingPlayerConfigCoordination{recordingCoordinationService: recordingCoordinationService{state: &domain.MasterCoordinationState{Roster: []domain.MasterRosterEntry{}, Sessions: []domain.MasterSessionEntry{}}}}
	sessions := &recordingPlayerConfigSession{snapshot: sessionservice.ActiveSession{Path: "/Campaigns/game.json", Session: &domain.Session{Version: 1, Name: "Game", Terminals: []domain.Terminal{}}}}
	configs := &recordingPlayerConfigService{next: playerconfigservice.Result{
		OK: true, FilePath: "/Campaigns/players/shared.json",
		Config: &domain.PlayerConfig{Version: 1, Name: "Shared", Roster: []domain.CharacterRosterEntry{{ID: "mara", Name: "Mara"}}},
	}}
	app := NewAppWithDependencies(AppDependencies{Sessions: sessions, PlayerConfigs: configs, Coordination: coordination})

	result := app.OpenPlayerConfig()
	if !result.OK || result.Config == nil || result.State == nil || len(result.State.Roster) != 1 {
		t.Fatalf("OpenPlayerConfig() = %#v", result)
	}
	if got, want := sessions.associations, []string{"/Campaigns/players/shared.json"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("session associations = %#v, want %#v", got, want)
	}
	if got, want := coordination.installs, []string{"Shared:mara"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("coordinator installs = %#v, want %#v", got, want)
	}
}

func TestNewPlayerConfigInstallsEmptyRosterAndPersistsFirstCharacter(t *testing.T) {
	t.Parallel()

	fileSystem := testutil.NewFakeFileSystem()
	target := "/Campaigns/players/empty.json"
	configs := playerconfigservice.NewService(
		playerconfigservice.NewStorage(fileSystem),
		&testutil.FakeDialog{SaveResult: target},
		"/Campaigns",
	)
	coordination := controlservice.New(controlservice.Config{RosterStore: configs})
	sessions := &recordingPlayerConfigSession{snapshot: sessionservice.ActiveSession{
		Path: "/Campaigns/game.json",
		Session: &domain.Session{
			Version:   1,
			Name:      "Game",
			Terminals: []domain.Terminal{},
		},
	}}
	app := NewAppWithDependencies(AppDependencies{
		Sessions:      sessions,
		PlayerConfigs: configs,
		Coordination:  coordination,
	})

	created := app.NewPlayerConfig()
	if !created.OK || created.Error != "" || created.Config == nil || created.Session == nil || created.State == nil {
		t.Fatalf("NewPlayerConfig() = %#v", created)
	}
	if created.Config.FilePath != target || created.Session.PlayerConfig == "" || created.State.PlayerConfig == nil || len(created.State.Roster) != 0 {
		t.Fatalf("new empty player config was not associated and installed: %#v", created)
	}

	added := app.AddCharacter("Mara")
	if !added.OK || added.State == nil || len(added.State.Roster) != 1 || added.State.Roster[0].Name != "Mara" {
		t.Fatalf("AddCharacter() after empty config = %#v", added)
	}
	stored, ok := fileSystem.File(target)
	if !ok {
		t.Fatalf("player config was not written to %q", target)
	}
	persisted, err := domain.DecodePlayerConfig(stored)
	if err != nil {
		t.Fatalf("DecodePlayerConfig() after first add: %v", err)
	}
	if len(persisted.Roster) != 1 || persisted.Roster[0].Name != "Mara" {
		t.Fatalf("persisted roster after first add = %#v", persisted.Roster)
	}
}

func TestApplicationStartsProtectedTunnelAfterLocalReadinessAndPublishesBothAddresses(t *testing.T) {
	recorder := &callRecorder{}
	events := &recordingEventSink{recorder: recorder}
	local := domain.ServerInfo{
		IP: "192.0.2.10", Port: 3690, URL: "http://192.0.2.10:3690", LocalURL: "http://127.0.0.1:3690",
	}
	tunnel := &recordingTunnel{
		recorder: recorder,
		info: domain.ServerInfo{
			URL: "https://players.example.test", LocalURL: "http://untrusted.invalid", Tunnel: true,
		},
	}
	app := NewAppWithDependencies(AppDependencies{
		Player:        &recordingPlayerServer{recorder: recorder, info: local},
		Tunnel:        tunnel,
		TunnelEnabled: true,
		Events:        events,
		Desktop:       &recordingDesktop{recorder: recorder},
	})

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if got, want := recorder.Calls(), []string{
		"player:start", "event:server-info", "desktop:ready", "tunnel:start", "event:server-info",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("public startup calls = %v, want %v", got, want)
	}
	status := app.GetRuntimeStatus()
	if status.ServerInfo == nil || status.ServerInfo.URL != "https://players.example.test" || status.ServerInfo.LocalURL != local.URL || !status.ServerInfo.Tunnel {
		t.Fatalf("public runtime status = %#v, want protected public and trusted local addresses", status)
	}
	if status.StartupError != "" || status.ServerInfo.TunnelError != "" {
		t.Fatalf("successful public runtime status retained an error: %#v", status)
	}
	records := events.Records()
	if len(records) != 2 {
		t.Fatalf("server-info events = %#v, want local then public", records)
	}
	first, firstOK := records[0].Payload.(domain.ServerInfo)
	second, secondOK := records[1].Payload.(domain.ServerInfo)
	if !firstOK || first.URL != local.URL || first.Tunnel || !secondOK || second.URL != "https://players.example.test" || second.LocalURL != local.URL || !second.Tunnel {
		t.Fatalf("server-info transition = %#v, want local then protected public", records)
	}

	recorder.Reset()
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if got, want := recorder.Calls(), []string{"tunnel:stop", "player:stop", "desktop:close"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("public shutdown calls = %v, want %v", got, want)
	}
}

func TestApplicationRejectsUnsafeTunnelAddressAndStopsAcquiredTunnel(t *testing.T) {
	for _, unsafeURL := range []string{
		"http://players.example.test",
		"https://overseer:vault-password@players.example.test",
		"not a URL",
	} {
		t.Run(unsafeURL, func(t *testing.T) {
			recorder := &callRecorder{}
			local := domain.ServerInfo{IP: "192.0.2.10", Port: 3690, URL: "http://192.0.2.10:3690"}
			app := NewAppWithDependencies(AppDependencies{
				Player: &recordingPlayerServer{recorder: recorder, info: local},
				Tunnel: &recordingTunnel{
					recorder: recorder,
					info:     domain.ServerInfo{URL: unsafeURL, Tunnel: true},
				},
				TunnelEnabled: true,
				Events:        &recordingEventSink{recorder: recorder},
				Desktop:       &recordingDesktop{recorder: recorder},
			})

			if err := app.Start(context.Background()); err != nil {
				t.Fatalf("Start() error = %v, want safe local fallback", err)
			}
			if got, want := recorder.Calls(), []string{
				"player:start", "event:server-info", "desktop:ready", "tunnel:start", "tunnel:stop", "event:server-info",
			}; !reflect.DeepEqual(got, want) {
				t.Fatalf("unsafe-public startup calls = %v, want %v", got, want)
			}
			status := app.GetRuntimeStatus()
			if status.ServerInfo == nil || status.ServerInfo.URL != local.URL || status.ServerInfo.Tunnel || status.ServerInfo.TunnelError != tunnelAddressFailureMessage {
				t.Fatalf("unsafe-public status = %#v, want local fallback", status)
			}
			if strings.Contains(status.StartupError, unsafeURL) || strings.Contains(status.ServerInfo.TunnelError, unsafeURL) {
				t.Fatalf("unsafe public address leaked into status: %#v", status)
			}
		})
	}
}

func TestApplicationUnwindsPartialStartup(t *testing.T) {
	recorder := &callRecorder{}
	player := &recordingPlayerServer{
		recorder: recorder,
		info:     domain.ServerInfo{IP: "127.0.0.1", Port: 3690, URL: "http://127.0.0.1:3690"},
	}
	app := NewAppWithDependencies(AppDependencies{
		Player: player,
		Events: &recordingEventSink{recorder: recorder, err: errors.New("bridge unavailable")},
		Desktop: &recordingDesktop{
			recorder: recorder,
		},
	})

	err := app.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "bridge") {
		t.Fatalf("Start() error = %v, want actionable bridge error", err)
	}
	if got, want := recorder.Calls(), []string{"player:start", "event:server-info", "player:stop"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("partial-start calls = %v, want %v", got, want)
	}
	status := app.GetRuntimeStatus()
	if status.ServerInfo != nil || !strings.Contains(status.StartupError, "bridge") {
		t.Fatalf("failure status = %#v", status)
	}
}

func TestApplicationShutdownIsReverseOrderedAndIdempotent(t *testing.T) {
	recorder := &callRecorder{}
	app := NewAppWithDependencies(AppDependencies{
		Sessions: &recordingSessionService{recorder: recorder},
		Player: &recordingPlayerServer{
			recorder: recorder,
			info:     domain.ServerInfo{IP: "127.0.0.1", Port: 3690, URL: "http://127.0.0.1:3690"},
		},
		Tunnel:        &recordingTunnel{recorder: recorder, info: domain.ServerInfo{URL: "https://public.example", Tunnel: true}},
		TunnelEnabled: true,
		Events:        &recordingEventSink{recorder: recorder},
		Desktop:       &recordingDesktop{recorder: recorder},
	})
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	recorder.Reset()
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}
	if got, want := recorder.Calls(), []string{"tunnel:stop", "player:stop", "session:shutdown", "desktop:close"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("shutdown calls = %v, want %v", got, want)
	}
}

func TestInvalidPublicConfigurationStartsZeroTunnelProcesses(t *testing.T) {
	recorder := &callRecorder{}
	tunnel := &invalidPublicTunnel{
		recorder: recorder,
		err:      errors.New("public credentials are missing or invalid"),
	}
	app := NewAppWithDependencies(AppDependencies{
		Player: &recordingPlayerServer{
			recorder: recorder,
			info:     domain.ServerInfo{IP: "192.0.2.10", Port: 3690, URL: "http://192.0.2.10:3690"},
		},
		Tunnel:        tunnel,
		TunnelEnabled: true,
		Events:        &recordingEventSink{recorder: recorder},
		Desktop:       &recordingDesktop{recorder: recorder},
	})

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want local readiness despite invalid public configuration", err)
	}
	if tunnel.validationCalls != 1 {
		t.Fatalf("tunnel validation calls = %d, want 1", tunnel.validationCalls)
	}
	if tunnel.processStarts != 0 {
		t.Fatalf("tunnel process starts = %d, want 0 for invalid credentials", tunnel.processStarts)
	}
	if got, want := recorder.Calls(), []string{
		"player:start", "event:server-info", "desktop:ready", "tunnel:validate", "event:server-info",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid-public startup calls = %v, want %v", got, want)
	}
}

func TestInvalidPublicConfigurationPreservesLocalReadinessAndNonSecretStatus(t *testing.T) {
	const (
		localURL       = "http://192.0.2.10:3690"
		secretUsername = "overseer-private"
		secretPassword = "vault-door-password"
	)
	recorder := &callRecorder{}
	events := &recordingEventSink{recorder: recorder}
	tunnel := &invalidPublicTunnel{
		recorder: recorder,
		err:      errors.New("authentication rejected for " + secretUsername + ":" + secretPassword),
	}
	app := NewAppWithDependencies(AppDependencies{
		Player: &recordingPlayerServer{
			recorder: recorder,
			info: domain.ServerInfo{
				IP: "192.0.2.10", Port: 3690, URL: localURL, LocalURL: "http://127.0.0.1:3690",
			},
		},
		Tunnel:        tunnel,
		TunnelEnabled: true,
		Events:        events,
		Desktop:       &recordingDesktop{recorder: recorder},
	})

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v, want usable local mode", err)
	}
	status := app.GetRuntimeStatus()
	if status.ServerInfo == nil || status.ServerInfo.URL != localURL || status.ServerInfo.Tunnel {
		t.Fatalf("runtime server info = %#v, want unchanged local readiness", status.ServerInfo)
	}
	if status.ServerInfo.TunnelError == "" || status.StartupError == "" {
		t.Fatalf("runtime status = %#v, want actionable public-access failure", status)
	}
	combinedStatus := status.StartupError + " " + status.ServerInfo.TunnelError
	for _, expected := range []string{"public", "credential", "executable", "network"} {
		if !strings.Contains(strings.ToLower(combinedStatus), expected) {
			t.Errorf("public failure status %q is missing actionable term %q", combinedStatus, expected)
		}
	}
	for _, secret := range []string{secretUsername, secretPassword} {
		if strings.Contains(combinedStatus, secret) {
			t.Errorf("public failure status disclosed secret %q", secret)
		}
	}

	records := events.Records()
	if len(records) < 2 {
		t.Fatalf("server-info event records = %#v, want local and failure status", records)
	}
	lastInfo, ok := records[len(records)-1].Payload.(domain.ServerInfo)
	if !ok || lastInfo.URL != localURL || lastInfo.Tunnel || lastInfo.TunnelError == "" {
		t.Fatalf("last server-info event = %#v, want usable local URL plus tunnel error", records[len(records)-1])
	}

	recorder.Reset()
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if got, want := recorder.Calls(), []string{"player:stop", "desktop:close"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("invalid-public shutdown calls = %v, want %v", got, want)
	}
	if tunnel.stopCalls != 0 {
		t.Fatalf("tunnel stop calls = %d, want 0 because no process was acquired", tunnel.stopCalls)
	}
}

func TestApplicationPlayerStartFailureNeverReportsReady(t *testing.T) {
	recorder := &callRecorder{}
	app := NewAppWithDependencies(AppDependencies{
		Player:  &recordingPlayerServer{recorder: recorder, startErr: errors.New("port 3690 is already in use")},
		Events:  &recordingEventSink{recorder: recorder},
		Desktop: &recordingDesktop{recorder: recorder},
	})

	err := app.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "3690") {
		t.Fatalf("Start() error = %v, want port detail", err)
	}
	if got, want := recorder.Calls(), []string{"player:start"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("failed startup calls = %v, want %v", got, want)
	}
	if status := app.GetRuntimeStatus(); status.ServerInfo != nil || !strings.Contains(status.StartupError, "3690") {
		t.Fatalf("failure status = %#v", status)
	}
}

func TestBridgeRejectsInvalidLivePayloadsBeforeMutation(t *testing.T) {
	coordination := &recordingTerminalCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: &domain.MasterCoordinationState{
			Revision: 1, Broadcast: &domain.MasterBroadcastState{ID: "broadcast-1"},
		}},
	}
	app := NewAppWithDependencies(AppDependencies{Coordination: coordination})

	activationResult := app.RequestTerminalActivation(LiveTerminalPayload{
		TerminalID:   "terminal-1",
		TerminalName: "Overseer",
		Tree: domain.ContentNode{
			ID: "root", Type: domain.NodeCommand, Name: "not a folder", Text: "invalid root",
		},
		HackLevel: 1,
	})
	if activationResult.OK || activationResult.Error == "" {
		t.Fatalf("RequestTerminalActivation(invalid) = %#v, want structured validation error", activationResult)
	}

	updateResult := app.UpdateLiveTerminal(LiveUpdatePayload{
		Tree: domain.ContentNode{ID: "root", Type: "script", Name: "unsupported"},
	})
	if updateResult.OK || updateResult.Error == "" {
		t.Fatalf("UpdateLiveTerminal(invalid) = %#v, want structured validation error", updateResult)
	}
	if len(coordination.targets) != 0 || coordination.updateCalls != 0 {
		t.Fatalf("invalid live payloads reached coordinator: activations=%d updates=%d", len(coordination.targets), coordination.updateCalls)
	}
}

func TestRuntimeStatusReturnsCompleteDetachedSnapshot(t *testing.T) {
	app := NewAppWithDependencies(AppDependencies{})
	app.serverInfo = &domain.ServerInfo{
		IP: "192.0.2.10", Port: 3690, URL: "http://192.0.2.10:3690",
	}
	app.clientCount = 4
	app.hackState = &domain.PublicHackState{
		Level: 2, AttemptsMax: 4, AttemptsLeft: 3, Log: []string{"ENTRY DENIED"},
		Columns: []domain.HackColumn{{Addresses: []string{"0xF000"}, Text: "...."}},
	}
	app.saveState = "saving"
	app.requestedRevision = 8
	app.savedRevision = 7

	status := app.GetRuntimeStatus()
	if status.ServerInfo == nil || status.ServerInfo.URL != "http://192.0.2.10:3690" {
		t.Fatalf("RuntimeStatus.ServerInfo = %#v", status.ServerInfo)
	}
	if status.ClientCount != 4 || status.HackState == nil || status.HackState.AttemptsLeft != 3 {
		t.Fatalf("RuntimeStatus bridge state = %#v", status)
	}
	if status.SaveState != "saving" || status.RequestedRevision != 8 || status.SavedRevision != 7 {
		t.Fatalf("RuntimeStatus save state = %#v", status)
	}

	status.ServerInfo.URL = "mutated"
	status.HackState.Log[0] = "mutated"
	status.HackState.Columns[0].Addresses[0] = "mutated"
	again := app.GetRuntimeStatus()
	if again.ServerInfo.URL == "mutated" || again.HackState.Log[0] == "mutated" || again.HackState.Columns[0].Addresses[0] == "mutated" {
		t.Fatalf("GetRuntimeStatus returned aliases into canonical state: %#v", again)
	}
}

func TestCoordinationBridgeAddsCharacterStartsBroadcastAndReplaysDetachedState(t *testing.T) {
	recorder := &callRecorder{}
	events := &recordingEventSink{recorder: recorder}
	service := &recordingCoordinationService{
		state: &domain.MasterCoordinationState{Revision: 4},
		addState: &domain.MasterCoordinationState{
			Revision: 5,
			Roster:   []domain.MasterRosterEntry{{ID: "character-1", Name: "Mara"}},
		},
		startState: &domain.MasterCoordinationState{
			Revision: 6,
			Roster:   []domain.MasterRosterEntry{{ID: "character-1", Name: "Mara"}},
			Broadcast: &domain.MasterBroadcastState{
				ID: "broadcast-1",
			},
		},
	}
	app := NewAppWithDependencies(AppDependencies{Coordination: service, Events: events})

	initial := app.GetRuntimeStatus()
	if initial.CoordinationState == nil || initial.CoordinationState.Revision != 4 {
		t.Fatalf("initial coordination status = %#v, want replayable revision 4", initial.CoordinationState)
	}

	added := app.AddCharacter("  Mara  ")
	if !added.OK || added.Error != "" || added.State == nil || added.State.Revision != 5 {
		t.Fatalf("AddCharacter() = %#v, want accepted revision 5", added)
	}
	if got, want := service.addNames, []string{"Mara"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("coordinator AddCharacter calls = %v, want %v", got, want)
	}
	added.State.Roster[0].Name = "mutated result"

	started := app.StartBroadcast()
	if !started.OK || started.Error != "" || started.State == nil || started.State.Broadcast == nil || started.State.Broadcast.ID != "broadcast-1" {
		t.Fatalf("StartBroadcast() = %#v, want accepted broadcast", started)
	}
	started.State.Roster[0].Name = "mutated start result"

	status := app.GetRuntimeStatus()
	if status.CoordinationState == nil || status.CoordinationState.Revision != 6 || status.CoordinationState.Roster[0].Name != "Mara" {
		t.Fatalf("detached coordination status = %#v", status.CoordinationState)
	}
	records := events.Records()
	if len(records) != 2 || records[0].Name != "coordination-state" || records[1].Name != "coordination-state" {
		t.Fatalf("coordination events = %#v, want add then start snapshots", records)
	}
	last, ok := records[1].Payload.(*domain.MasterCoordinationState)
	if !ok || last == nil || last.Broadcast == nil || last.Broadcast.ID != "broadcast-1" {
		t.Fatalf("last coordination event = %#v", records[1])
	}
	last.Roster[0].Name = "mutated event"
	if replay := app.GetRuntimeStatus().CoordinationState; replay == nil || replay.Roster[0].Name != "Mara" {
		t.Fatalf("event payload aliased runtime status: %#v", replay)
	}
}

func TestCoordinationBridgeRejectsInvalidOrFailedCommandsWithoutPartialState(t *testing.T) {
	recorder := &callRecorder{}
	service := &recordingCoordinationService{
		state:    &domain.MasterCoordinationState{Revision: 9, Roster: []domain.MasterRosterEntry{{ID: "character-1", Name: "Mara"}}},
		addErr:   errors.New("roster unavailable"),
		startErr: errors.New("broadcast already active"),
	}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: service,
		Events:       &recordingEventSink{recorder: recorder},
	})

	invalid := app.AddCharacter("   ")
	if invalid.OK || invalid.Error == "" || len(service.addNames) != 0 {
		t.Fatalf("AddCharacter(blank) = %#v, calls = %v", invalid, service.addNames)
	}
	failedAdd := app.AddCharacter("Boone")
	if failedAdd.OK || !strings.Contains(failedAdd.Error, "roster") {
		t.Fatalf("AddCharacter(failed) = %#v", failedAdd)
	}
	failedStart := app.StartBroadcast()
	if failedStart.OK || !strings.Contains(failedStart.Error, "already") {
		t.Fatalf("StartBroadcast(failed) = %#v", failedStart)
	}

	status := app.GetRuntimeStatus()
	if status.CoordinationState == nil || status.CoordinationState.Revision != 9 || len(status.CoordinationState.Roster) != 1 {
		t.Fatalf("failed commands partially changed status: %#v", status.CoordinationState)
	}
	if records := recorder.Calls(); len(records) != 0 {
		t.Fatalf("failed commands emitted coordination events: %v", records)
	}
}

func TestBroadcastLifecycleBridgeEndsRestartsReplaysAndDisposesWithoutDurableMutation(t *testing.T) {
	recorder := &callRecorder{}
	activeID := "terminal-1"
	coordination := &recordingBroadcastLifecycleService{
		recordingCoordinationService: recordingCoordinationService{state: &domain.MasterCoordinationState{
			Revision: 80, Roster: []domain.MasterRosterEntry{{ID: "character-1", Name: "Mara"}},
			Broadcast: &domain.MasterBroadcastState{ID: "broadcast-1", ActiveTerminalID: &activeID},
		}},
		endState: &domain.MasterCoordinationState{Revision: 81, Roster: []domain.MasterRosterEntry{{ID: "character-1", Name: "Mara"}}},
	}
	coordination.startState = &domain.MasterCoordinationState{
		Revision: 82, Roster: []domain.MasterRosterEntry{{ID: "character-1", Name: "Mara"}},
		Broadcast: &domain.MasterBroadcastState{ID: "broadcast-2"},
	}
	durable := &recordingSessionService{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: coordination, Sessions: durable, Events: &recordingEventSink{recorder: recorder},
	})

	ended := app.EndBroadcast()
	if !ended.OK || ended.Error != "" || ended.State == nil || ended.State.Broadcast != nil || ended.State.Revision != 81 {
		t.Fatalf("EndBroadcast() = %#v", ended)
	}
	restarted := app.StartBroadcast()
	if !restarted.OK || restarted.State == nil || restarted.State.Broadcast == nil || restarted.State.Broadcast.ID != "broadcast-2" {
		t.Fatalf("second StartBroadcast() = %#v", restarted)
	}
	if status := app.GetRuntimeStatus(); status.CoordinationState == nil || status.CoordinationState.Broadcast == nil || status.CoordinationState.Broadcast.ID != "broadcast-2" {
		t.Fatalf("broadcast lifecycle replay status = %#v", status)
	}
	if coordination.endCalls != 1 || coordination.startCalls != 1 {
		t.Fatalf("lifecycle calls end/start = %d/%d", coordination.endCalls, coordination.startCalls)
	}

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if coordination.shutdownCalls != 1 {
		t.Fatalf("coordination shutdown calls = %d, want 1", coordination.shutdownCalls)
	}
	if durable.shutdownCalls != 1 {
		t.Fatalf("durable session shutdown calls = %d, want 1 without mutation commands", durable.shutdownCalls)
	}
}

func TestCoordinationBridgeValidatesRosterSessionAndAssignmentCorrections(t *testing.T) {
	recorder := &callRecorder{}
	initial := &domain.MasterCoordinationState{
		Revision: 20,
		Roster: []domain.MasterRosterEntry{
			{ID: "character-1", Name: "Mara"},
			{ID: "character-2", Name: "Boone"},
		},
		Sessions: []domain.MasterSessionEntry{
			{ID: "session-1", FallbackName: "DEVICE 1", Connected: true},
			{ID: "session-2", FallbackName: "DEVICE 2", Connected: true},
		},
		Broadcast: &domain.MasterBroadcastState{ID: "broadcast-1"},
	}
	service := &recordingCorrectionCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: initial},
	}
	events := &recordingEventSink{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: service,
		Sessions:     &recordingSessionService{recorder: recorder},
		Events:       events,
	})

	commands := []struct {
		name string
		run  func() CoordinationCommandResult
	}{
		{"rename-character", func() CoordinationCommandResult {
			return app.RenameCharacter(CharacterRenamePayload{CharacterID: "character-1", Name: "  Mara Voss  "})
		}},
		{"delete-character", func() CoordinationCommandResult { return app.DeleteCharacter("character-2") }},
		{"rename-session", func() CoordinationCommandResult {
			return app.RenameLogicalSession(LogicalSessionRenamePayload{SessionID: "session-1", FallbackName: "  TABLET LEFT  "})
		}},
		{"assign-character", func() CoordinationCommandResult {
			return app.AssignCharacter(AssignmentPayload{SessionID: "session-1", CharacterID: "character-1"})
		}},
		{"release-character", func() CoordinationCommandResult { return app.ReleaseCharacter("session-1") }},
		{"move-character", func() CoordinationCommandResult {
			return app.MoveCharacter(MoveCharacterPayload{CharacterID: "character-1", ToSessionID: "session-2"})
		}},
	}
	for _, command := range commands {
		service.nextRevision++
		result := command.run()
		if !result.OK || result.Error != "" || result.State == nil || result.State.Revision != uint64(20+service.nextRevision) {
			t.Fatalf("%s result = %#v", command.name, result)
		}
		result.State.Roster[0].Name = "mutated result"
		if got := app.GetRuntimeStatus().CoordinationState.Roster[0].Name; got == "mutated result" {
			t.Fatalf("%s returned an aliased coordination state", command.name)
		}
	}

	wantCalls := []string{
		"rename-character:character-1:Mara Voss",
		"delete-character:character-2",
		"rename-session:session-1:TABLET LEFT",
		"assign-character:session-1:character-1",
		"release-character:session-1",
		"move-character:character-1:session-2",
	}
	if !reflect.DeepEqual(service.calls, wantCalls) {
		t.Fatalf("coordination correction calls = %v, want %v", service.calls, wantCalls)
	}
	if got := recorder.Calls(); len(got) != len(commands) {
		t.Fatalf("coordination event/save calls = %v, want only %d coordination events", got, len(commands))
	}
	for _, call := range recorder.Calls() {
		if call != "event:coordination-state" {
			t.Fatalf("coordination correction touched durable session path: %v", recorder.Calls())
		}
	}

	eventState, ok := events.Records()[0].Payload.(*domain.MasterCoordinationState)
	if !ok || eventState == nil {
		t.Fatalf("first correction event payload = %#v", events.Records()[0].Payload)
	}
	eventState.Roster[0].Name = "mutated event"
	if got := app.GetRuntimeStatus().CoordinationState.Roster[0].Name; got == "mutated event" {
		t.Fatal("coordination event payload aliases replayable runtime status")
	}

	before := app.GetRuntimeStatus().CoordinationState
	service.failCommand = "move-character"
	service.commandErr = errors.New("destination session already has a character")
	rejected := app.MoveCharacter(MoveCharacterPayload{CharacterID: "character-1", ToSessionID: "session-2"})
	if rejected.OK || !strings.Contains(rejected.Error, "already") || !reflect.DeepEqual(rejected.State, before) {
		t.Fatalf("conflicting MoveCharacter() = %#v, want unchanged authoritative snapshot", rejected)
	}
	if !reflect.DeepEqual(app.GetRuntimeStatus().CoordinationState, before) {
		t.Fatal("conflicting correction changed replayable coordination state")
	}

	invalidCalls := len(service.calls)
	invalid := []CoordinationCommandResult{
		app.RenameCharacter(CharacterRenamePayload{CharacterID: "", Name: "Mara"}),
		app.RenameLogicalSession(LogicalSessionRenamePayload{SessionID: "session-1", FallbackName: "  "}),
		app.AssignCharacter(AssignmentPayload{SessionID: "", CharacterID: "character-1"}),
		app.ReleaseCharacter("   "),
		app.MoveCharacter(MoveCharacterPayload{CharacterID: "character-1", ToSessionID: ""}),
	}
	for index, result := range invalid {
		if result.OK || result.Error == "" {
			t.Fatalf("invalid correction %d = %#v, want validation refusal", index, result)
		}
	}
	if len(service.calls) != invalidCalls {
		t.Fatalf("invalid payloads reached coordinator: %v", service.calls[invalidCalls:])
	}
}

func TestCoordinationBridgeValidatesAndPublishesActiveControllerReassignment(t *testing.T) {
	order := &callRecorder{}
	firstID := domain.LogicalSessionID("session-1")
	secondID := domain.LogicalSessionID("session-2")
	service := &recordingCorrectionCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: &domain.MasterCoordinationState{
			Revision: 30,
			Sessions: []domain.MasterSessionEntry{
				{ID: firstID, FallbackName: "DEVICE 1", Connected: true, Character: &domain.PlayerCharacter{ID: "character-1", Name: "Mara"}, Role: domain.PlayerRoleActive},
				{ID: secondID, FallbackName: "DEVICE 2", Connected: true, Character: &domain.PlayerCharacter{ID: "character-2", Name: "Boone"}, Role: domain.PlayerRoleObserver},
			},
			Broadcast: &domain.MasterBroadcastState{ID: "broadcast-1", ControllerSessionID: &firstID},
		}},
		order: order,
	}
	events := &recordingEventSink{recorder: order}
	app := NewAppWithDependencies(AppDependencies{Coordination: service, Events: events})

	result := app.SetActiveController(string(secondID))
	if !result.OK || result.Error != "" || result.State == nil || result.State.Revision != 31 {
		t.Fatalf("SetActiveController(second) = %#v", result)
	}
	if result.State.Broadcast == nil || result.State.Broadcast.ControllerSessionID == nil || *result.State.Broadcast.ControllerSessionID != secondID {
		t.Fatalf("reassigned broadcast = %#v, want controller %q", result.State.Broadcast, secondID)
	}
	if got := masterSessionEntryForAppTest(t, result.State, firstID).Role; got != domain.PlayerRoleObserver {
		t.Fatalf("former controller role = %q, want observer", got)
	}
	if got := masterSessionEntryForAppTest(t, result.State, secondID).Role; got != domain.PlayerRoleActive {
		t.Fatalf("new controller role = %q, want active", got)
	}
	if got, want := order.Calls(), []string{"coordinator:set-controller:session-2", "event:coordination-state"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("reassignment order = %v, want %v", got, want)
	}

	result.State.Sessions[0].FallbackName = "mutated result"
	eventState, ok := events.Records()[0].Payload.(*domain.MasterCoordinationState)
	if !ok || eventState == nil {
		t.Fatalf("controller event = %#v", events.Records())
	}
	eventState.Sessions[1].FallbackName = "mutated event"
	status := app.GetRuntimeStatus().CoordinationState
	if status.Sessions[0].FallbackName == "mutated result" || status.Sessions[1].FallbackName == "mutated event" {
		t.Fatalf("controller result/event alias replay status: %#v", status)
	}

	before := app.GetRuntimeStatus().CoordinationState
	service.failCommand = "set-active-controller"
	service.commandErr = errors.New("controller must be connected and assigned")
	rejected := app.SetActiveController(string(firstID))
	if rejected.OK || !strings.Contains(rejected.Error, "connected") || !reflect.DeepEqual(rejected.State, before) {
		t.Fatalf("ineligible SetActiveController() = %#v", rejected)
	}
	if len(events.Records()) != 1 || !reflect.DeepEqual(app.GetRuntimeStatus().CoordinationState, before) {
		t.Fatal("ineligible reassignment published or changed the authoritative snapshot")
	}

	callsBeforeBlank := len(service.calls)
	blank := app.SetActiveController("   ")
	if blank.OK || blank.Error == "" || len(service.calls) != callsBeforeBlank {
		t.Fatalf("blank SetActiveController() = %#v calls=%v", blank, service.calls)
	}
}

func TestCoordinationStatusReplaysDisconnectedControllerWithoutChangingClaimOrRole(t *testing.T) {
	controllerID := domain.LogicalSessionID("session-controller")
	characterID := domain.CharacterID("character-mara")
	state := &domain.MasterCoordinationState{
		Revision: 41,
		Roster:   []domain.MasterRosterEntry{{ID: characterID, Name: "Mara", ClaimedBySessionID: &controllerID}},
		Sessions: []domain.MasterSessionEntry{{
			ID: controllerID, FallbackName: "TABLET LEFT", Connected: false,
			Character: &domain.PlayerCharacter{ID: characterID, Name: "Mara"}, Role: domain.PlayerRoleActive,
		}},
		Broadcast: &domain.MasterBroadcastState{ID: "broadcast-1", ControllerSessionID: &controllerID},
	}
	events := &recordingEventSink{recorder: &callRecorder{}}
	service := &recordingCoordinationService{state: state}
	app := NewAppWithDependencies(AppDependencies{Coordination: service, Events: events})

	initial := app.GetRuntimeStatus().CoordinationState
	assertDisconnectedControllerSnapshot(t, initial, controllerID, characterID)
	app.publishCoordinationState(state)
	replay := app.GetRuntimeStatus().CoordinationState
	assertDisconnectedControllerSnapshot(t, replay, controllerID, characterID)
	records := events.Records()
	if len(records) != 1 {
		t.Fatalf("presence events = %#v, want one coordination snapshot", records)
	}
	published, ok := records[0].Payload.(*domain.MasterCoordinationState)
	if !ok {
		t.Fatalf("presence payload = %#v", records[0].Payload)
	}
	assertDisconnectedControllerSnapshot(t, published, controllerID, characterID)
	published.Sessions[0].Connected = true
	published.Roster[0].Name = "mutated"
	assertDisconnectedControllerSnapshot(t, app.GetRuntimeStatus().CoordinationState, controllerID, characterID)
}

func assertDisconnectedControllerSnapshot(t *testing.T, state *domain.MasterCoordinationState, sessionID domain.LogicalSessionID, characterID domain.CharacterID) {
	t.Helper()
	if state == nil || state.Broadcast == nil || state.Broadcast.ControllerSessionID == nil || *state.Broadcast.ControllerSessionID != sessionID {
		t.Fatalf("disconnected controller broadcast = %#v", state)
	}
	session := masterSessionEntryForAppTest(t, state, sessionID)
	if session.Connected || session.Role != domain.PlayerRoleActive || session.Character == nil || session.Character.ID != characterID {
		t.Fatalf("disconnected controller session = %#v", session)
	}
	if len(state.Roster) != 1 || state.Roster[0].ClaimedBySessionID == nil || *state.Roster[0].ClaimedBySessionID != sessionID {
		t.Fatalf("disconnected controller claim = %#v", state.Roster)
	}
}

func masterSessionEntryForAppTest(t *testing.T, state *domain.MasterCoordinationState, sessionID domain.LogicalSessionID) domain.MasterSessionEntry {
	t.Helper()
	for _, session := range state.Sessions {
		if session.ID == sessionID {
			return session
		}
	}
	t.Fatalf("coordination state has no session %q", sessionID)
	return domain.MasterSessionEntry{}
}

func TestCoordinationBridgeOrdersTerminalActivationClearAndUpdateWithoutLegacyMutation(t *testing.T) {
	recorder := &callRecorder{}
	initial := &domain.MasterCoordinationState{Revision: 40, Broadcast: &domain.MasterBroadcastState{ID: "broadcast-terminal-bridge"}}
	coordination := &recordingTerminalCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: initial},
		order:                        recorder,
	}
	legacy := &recordingLiveService{
		setState: &domain.PublicLiveState{TerminalID: "legacy-terminal"}, updateState: &domain.PublicLiveState{TerminalID: "legacy-terminal"},
	}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: coordination, Live: legacy,
		Player: &recordingPlayerServer{recorder: recorder}, Events: &recordingEventSink{recorder: recorder},
	})
	app.hackState = &domain.PublicHackState{Level: 1, AttemptsMax: 4, AttemptsLeft: 2}
	tree := domain.ContentNode{
		ID: "root", Type: domain.NodeFolder, Name: "ROOT",
		Children: []domain.ContentNode{{ID: "docs", Type: domain.NodeFolder, Name: "DOCS"}},
	}

	activated := app.RequestTerminalActivation(LiveTerminalPayload{
		TerminalID: "  terminal-1  ", TerminalName: "  Overseer  ", Tree: tree, HackLevel: 2, IntroText: "WELCOME",
	})
	if !activated.OK || activated.Error != "" || activated.Status != "activated" || activated.SwitchID != "" || activated.State == nil || activated.State.Revision != 41 {
		t.Fatalf("RequestTerminalActivation() = %#v", activated)
	}
	if activated.State.Broadcast == nil || activated.State.Broadcast.ActiveTerminalID == nil || *activated.State.Broadcast.ActiveTerminalID != "terminal-1" {
		t.Fatalf("activation authoritative state = %#v", activated.State)
	}
	if len(coordination.targets) != 1 || coordination.targets[0].TerminalID != "terminal-1" || coordination.targets[0].TerminalName != "Overseer" {
		t.Fatalf("activation payload was not trimmed before coordinator: %#v", coordination.targets)
	}

	intro := "UPDATED INTRO"
	updated := app.UpdateLiveTerminal(LiveUpdatePayload{Tree: tree, IntroText: &intro})
	if !updated.OK || updated.Error != "" || updated.State == nil || updated.State.Revision != 42 {
		t.Fatalf("UpdateLiveTerminal() = %#v, want authoritative revision 42", updated)
	}
	if coordination.updateCalls != 1 || coordination.updateIntro == nil || *coordination.updateIntro != intro || !reflect.DeepEqual(coordination.updateTree, tree) {
		t.Fatalf("ordered update payload = calls %d tree %#v intro %#v", coordination.updateCalls, coordination.updateTree, coordination.updateIntro)
	}

	cleared := app.RequestTerminalClear()
	if !cleared.OK || cleared.Error != "" || cleared.Status != "cleared" || cleared.SwitchID != "" || cleared.State == nil || cleared.State.Revision != 43 {
		t.Fatalf("RequestTerminalClear() = %#v", cleared)
	}
	if cleared.State.Broadcast == nil || cleared.State.Broadcast.ActiveTerminalID != nil {
		t.Fatalf("clear authoritative state = %#v", cleared.State)
	}

	wantOrder := []string{
		"coordinator:request-terminal-activation:terminal-1", "event:coordination-state",
		"coordinator:update-live-terminal:terminal-1", "event:coordination-state",
		"coordinator:request-terminal-clear", "event:coordination-state",
	}
	if got := recorder.Calls(); !reflect.DeepEqual(got, wantOrder) {
		t.Fatalf("terminal coordination order = %v, want %v", got, wantOrder)
	}
	if legacy.setCalls != 0 || legacy.updateCalls != 0 || legacy.clearCalls != 0 {
		t.Fatalf("coordinated commands mutated legacy live state: set/update/clear=%d/%d/%d", legacy.setCalls, legacy.updateCalls, legacy.clearCalls)
	}
	status := app.GetRuntimeStatus()
	if status.CoordinationState == nil || status.CoordinationState.Revision != 43 || status.HackState == nil || status.HackState.AttemptsLeft != 2 {
		t.Fatalf("terminal runtime status = %#v, want revision 43 and unchanged hack mirror", status)
	}
	cleared.State.Revision = 999
	if app.GetRuntimeStatus().CoordinationState.Revision != 43 {
		t.Fatal("terminal command result aliases replayable coordination state")
	}
}

func TestCoordinationBridgeRejectsInvalidOrFailedTerminalRequestsWithoutOptimisticState(t *testing.T) {
	recorder := &callRecorder{}
	initial := &domain.MasterCoordinationState{Revision: 50, Broadcast: &domain.MasterBroadcastState{ID: "broadcast-terminal-refusal"}}
	coordination := &recordingTerminalCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: initial}, order: recorder,
	}
	legacy := &recordingLiveService{}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: coordination, Live: legacy,
		Player: &recordingPlayerServer{recorder: recorder}, Events: &recordingEventSink{recorder: recorder},
	})

	invalidTree := domain.ContentNode{ID: "root", Type: domain.NodeCommand, Name: "not a folder"}
	invalid := []TerminalSwitchCommandResult{
		app.RequestTerminalActivation(LiveTerminalPayload{TerminalID: " ", TerminalName: "Overseer", Tree: invalidTree}),
		app.RequestTerminalActivation(LiveTerminalPayload{TerminalID: "terminal-1", TerminalName: " ", Tree: invalidTree}),
		app.RequestTerminalActivation(LiveTerminalPayload{TerminalID: "terminal-1", TerminalName: "Overseer", Tree: invalidTree}),
	}
	for index, result := range invalid {
		if result.OK || result.Error == "" || result.State == nil || result.State.Revision != 50 {
			t.Fatalf("invalid terminal activation %d = %#v", index, result)
		}
	}
	if len(coordination.targets) != 0 || len(recorder.Calls()) != 0 {
		t.Fatalf("invalid payload reached coordinator/publication: targets=%#v calls=%v", coordination.targets, recorder.Calls())
	}

	coordination.commandErr = errors.New("active terminal has an unfinished puzzle")
	validTree := domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{}}
	rejected := app.RequestTerminalActivation(LiveTerminalPayload{
		TerminalID: "terminal-2", TerminalName: "Overseer 2", Tree: validTree, HackLevel: 1,
	})
	if rejected.OK || !strings.Contains(rejected.Error, "unfinished") || rejected.Status != "" || rejected.State == nil || rejected.State.Revision != 50 {
		t.Fatalf("failed terminal activation = %#v", rejected)
	}
	if status := app.GetRuntimeStatus().CoordinationState; !reflect.DeepEqual(status, initial) {
		t.Fatalf("failed activation changed replay state: %#v", status)
	}
	if got := recorder.Calls(); !reflect.DeepEqual(got, []string{"coordinator:request-terminal-activation:terminal-2"}) {
		t.Fatalf("failed activation publications = %v", got)
	}
	if legacy.setCalls != 0 || legacy.updateCalls != 0 || legacy.clearCalls != 0 {
		t.Fatalf("failed request touched legacy live: set/update/clear=%d/%d/%d", legacy.setCalls, legacy.updateCalls, legacy.clearCalls)
	}
}

func TestResetFailedHackValidatesPrivatePayloadAndReturnsAuthoritativeState(t *testing.T) {
	recorder := &callRecorder{}
	initial := &domain.MasterCoordinationState{
		Revision:  70,
		Broadcast: &domain.MasterBroadcastState{ID: "broadcast-reset", ActiveTerminalID: appStringPointer("terminal-1")},
	}
	coordination := &recordingTerminalCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: initial}, order: recorder,
	}
	legacy := &recordingLiveService{}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: coordination, Live: legacy, Events: &recordingEventSink{recorder: recorder},
	})
	tree := domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{}}

	invalid := app.ResetFailedHack(LiveTerminalPayload{TerminalID: " ", TerminalName: "Overseer", Tree: tree, HackLevel: 2})
	if invalid.OK || invalid.Error == "" || invalid.State == nil || invalid.State.Revision != 70 || len(coordination.resetTargets) != 0 {
		t.Fatalf("invalid ResetFailedHack() = %#v targets=%#v", invalid, coordination.resetTargets)
	}

	result := app.ResetFailedHack(LiveTerminalPayload{
		TerminalID: "  terminal-1 ", TerminalName: " Overseer Latest ", Tree: tree, HackLevel: 2, IntroText: "LATEST",
	})
	if !result.OK || result.Error != "" || result.State == nil || result.State.Revision != 71 {
		t.Fatalf("ResetFailedHack() = %#v", result)
	}
	if len(coordination.resetTargets) != 1 || coordination.resetTargets[0].TerminalID != "terminal-1" || coordination.resetTargets[0].TerminalName != "Overseer Latest" || coordination.resetTargets[0].HackLevel != 2 {
		t.Fatalf("validated reset payloads = %#v", coordination.resetTargets)
	}
	if got := recorder.Calls(); !reflect.DeepEqual(got, []string{"coordinator:reset-failed-hack:terminal-1", "event:coordination-state"}) {
		t.Fatalf("reset order = %v", got)
	}
	if legacy.setCalls != 0 || legacy.updateCalls != 0 || legacy.clearCalls != 0 || legacy.forceCalls != 0 {
		t.Fatalf("reset bypassed coordinator through legacy live service: %#v", legacy)
	}

	coordination.commandErr = errors.New("active hacking puzzle is not failed")
	rejected := app.ResetFailedHack(LiveTerminalPayload{
		TerminalID: "terminal-1", TerminalName: "Overseer Latest", Tree: tree, HackLevel: 2, IntroText: "LATEST",
	})
	if rejected.OK || !strings.Contains(rejected.Error, "not failed") || rejected.State == nil || rejected.State.Revision != 71 || app.GetRuntimeStatus().CoordinationState.Revision != 71 {
		t.Fatalf("ineligible ResetFailedHack() = %#v", rejected)
	}
}

func TestTerminalSwitchBridgeReturnsDecisionShapeAndResolvesValidatedChoices(t *testing.T) {
	recorder := &callRecorder{}
	initial := &domain.MasterCoordinationState{
		Revision:  60,
		Broadcast: &domain.MasterBroadcastState{ID: "broadcast-decision", ActiveTerminalID: appStringPointer("terminal-1")},
	}
	coordination := &recordingTerminalCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: initial},
		order:                        recorder, decisionRequired: true, nextSwitchID: "opaque-switch-1",
	}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: coordination, Events: &recordingEventSink{recorder: recorder},
	})
	tree := domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{}}

	pending := app.RequestTerminalActivation(LiveTerminalPayload{
		TerminalID: "terminal-2", TerminalName: "Archive", Tree: tree, HackLevel: 2,
	})
	if !pending.OK || pending.Error != "" || pending.Status != "decision-required" || pending.SwitchID != "opaque-switch-1" || pending.State == nil {
		t.Fatalf("decision-required activation = %#v", pending)
	}
	if pending.State.PendingSwitch == nil || pending.State.PendingSwitch.SwitchID != pending.SwitchID || pending.State.Broadcast.ActiveTerminalID == nil || *pending.State.Broadcast.ActiveTerminalID != "terminal-1" {
		t.Fatalf("pending activation changed source or omitted switch metadata: %#v", pending.State)
	}

	resolved := app.ResolveTerminalSwitch(TerminalSwitchDecisionPayload{
		SwitchID: pending.SwitchID, Decision: domain.TerminalSwitchPreserve,
	})
	if !resolved.OK || resolved.Error != "" || resolved.Status != "activated" || resolved.SwitchID != "" || resolved.State == nil || resolved.State.PendingSwitch != nil {
		t.Fatalf("preserve resolution = %#v", resolved)
	}
	if got := coordination.decisions; !reflect.DeepEqual(got, []recordedTerminalDecision{{SwitchID: "opaque-switch-1", Decision: domain.TerminalSwitchPreserve}}) {
		t.Fatalf("coordinator decisions = %#v", got)
	}
	if got := recorder.Calls(); !reflect.DeepEqual(got, []string{
		"coordinator:request-terminal-activation:terminal-2", "event:coordination-state",
		"coordinator:resolve-terminal-switch:opaque-switch-1:preserve", "event:coordination-state",
	}) {
		t.Fatalf("decision bridge ordering = %v", got)
	}

	for _, decision := range []domain.TerminalSwitchChoice{domain.TerminalSwitchDiscard, domain.TerminalSwitchCancel} {
		coordination.decisionRequired = true
		coordination.nextSwitchID = domain.SwitchID("opaque-" + string(decision))
		request := app.RequestTerminalClear()
		if request.Status != "decision-required" || request.SwitchID == "" {
			t.Fatalf("clear %s decision request = %#v", decision, request)
		}
		result := app.ResolveTerminalSwitch(TerminalSwitchDecisionPayload{SwitchID: request.SwitchID, Decision: decision})
		wantStatus := "cleared"
		if decision == domain.TerminalSwitchCancel {
			wantStatus = "cancelled"
		}
		if !result.OK || result.Status != wantStatus || result.SwitchID != "" {
			t.Fatalf("resolve %s = %#v, want status %q", decision, result, wantStatus)
		}
		if decision != domain.TerminalSwitchCancel {
			active := "terminal-1"
			coordination.state.Broadcast.ActiveTerminalID = &active
		}
	}
}

func TestTerminalSwitchBridgeRejectsInvalidAndStaleDecisionButKeepsTrustedForceSuccessEligible(t *testing.T) {
	initial := &domain.MasterCoordinationState{
		Revision:  70,
		Broadcast: &domain.MasterBroadcastState{ID: "broadcast-decision", ActiveTerminalID: appStringPointer("terminal-1")},
		PendingSwitch: &domain.MasterPendingSwitch{
			SwitchID: "switch-current", BroadcastID: "broadcast-decision", SourceTerminalID: "terminal-1",
		},
	}
	coordination := &recordingTerminalCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: initial},
		commandErr:                   errors.New("terminal switch decision is stale"),
		forceState:                   &domain.PublicHackState{Level: 2, AttemptsMax: 4, AttemptsLeft: 3},
	}
	app := NewAppWithDependencies(AppDependencies{Coordination: coordination})

	invalid := []TerminalSwitchDecisionPayload{
		{SwitchID: "", Decision: domain.TerminalSwitchPreserve},
		{SwitchID: "switch-current", Decision: ""},
		{SwitchID: "switch-current", Decision: "restart"},
	}
	for index, payload := range invalid {
		result := app.ResolveTerminalSwitch(payload)
		if result.OK || result.Error == "" || result.State == nil || result.State.Revision != 70 {
			t.Fatalf("invalid decision %d = %#v", index, result)
		}
	}
	if len(coordination.decisions) != 0 {
		t.Fatalf("invalid decisions reached coordinator: %#v", coordination.decisions)
	}

	stale := app.ResolveTerminalSwitch(TerminalSwitchDecisionPayload{SwitchID: "switch-old", Decision: domain.TerminalSwitchDiscard})
	if stale.OK || !strings.Contains(stale.Error, "stale") || stale.Status != "" || stale.SwitchID != "" || stale.State == nil || !reflect.DeepEqual(stale.State, initial) {
		t.Fatalf("stale decision = %#v", stale)
	}
	stale.State.Revision = 999
	if app.GetRuntimeStatus().CoordinationState.Revision != 70 {
		t.Fatal("stale switch result aliases replay state")
	}

	forced := app.ForceHackSuccess()
	if !forced.OK || coordination.forceCalls != 1 {
		t.Fatalf("ForceHackSuccess() while decision pending = %#v, calls %d", forced, coordination.forceCalls)
	}
}

func appStringPointer(value string) *string { return &value }

func TestDOMReadyReplaysCurrentBridgeEvents(t *testing.T) {
	recorder := &callRecorder{}
	events := &recordingEventSink{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{Events: events})
	app.serverInfo = &domain.ServerInfo{IP: "127.0.0.1", Port: 3690, URL: "http://127.0.0.1:3690"}
	app.clientCount = 5
	app.hackState = &domain.PublicHackState{Level: 3, AttemptsMax: 4, AttemptsLeft: 2}

	app.domReady(context.Background())

	if got, want := recorder.Calls(), []string{"event:server-info", "event:client-count", "event:hack-state", "event:coordination-state"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("DOM-ready events = %v, want %v", got, want)
	}
	records := events.Records()
	if info, ok := records[0].Payload.(domain.ServerInfo); !ok || info.Port != 3690 {
		t.Fatalf("server-info payload = %#v", records[0].Payload)
	}
	if count, ok := records[1].Payload.(int); !ok || count != 5 {
		t.Fatalf("client-count payload = %#v", records[1].Payload)
	}
	if hackState, ok := records[2].Payload.(*domain.PublicHackState); !ok || hackState == nil || hackState.AttemptsLeft != 2 {
		t.Fatalf("hack-state payload = %#v", records[2].Payload)
	}
	coordinationState, ok := records[3].Payload.(*domain.MasterCoordinationState)
	if records[3].Name != coordinationStateEvent || !ok || coordinationState != nil {
		t.Fatalf("coordination-state payload = %#v, want nil replay without coordinator", records[3].Payload)
	}
}

func TestOpenURLAllowsOnlyHTTPAndHTTPS(t *testing.T) {
	browser := &testutil.FakeBrowser{}
	app := NewAppWithDependencies(AppDependencies{Browser: browser})

	for _, rawURL := range []string{
		"file:///tmp/session.json",
		"javascript:alert(1)",
		"mailto:overseer@example.test",
		"not a URL",
		"http://[::1",
	} {
		result := app.OpenURL(rawURL)
		if result.OK || result.Error == "" {
			t.Errorf("OpenURL(%q) = %#v, want structured rejection", rawURL, result)
		}
	}

	for _, rawURL := range []string{
		"http://127.0.0.1:3690/",
		"https://players.example.test/session",
	} {
		if result := app.OpenURL(rawURL); !result.OK || result.Error != "" {
			t.Errorf("OpenURL(%q) = %#v, want success", rawURL, result)
		}
	}
	if got, want := browser.URLs(), []string{
		"http://127.0.0.1:3690/",
		"https://players.example.test/session",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("browser URLs = %v, want %v", got, want)
	}
}

func TestBridgeCoordinatorActivationAndLifecycleCleanup(t *testing.T) {
	recorder := &callRecorder{}
	live := &recordingLiveService{}
	coordination := &recordingTerminalCoordinationService{
		recordingCoordinationService: recordingCoordinationService{state: &domain.MasterCoordinationState{
			Revision: 1, Broadcast: &domain.MasterBroadcastState{ID: "broadcast-1"},
		}},
		order: recorder,
	}
	events := &recordingEventSink{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{
		Sessions: &recordingSessionService{recorder: recorder},
		Player: &recordingPlayerServer{
			recorder: recorder,
			info:     domain.ServerInfo{IP: "127.0.0.1", Port: 3690, URL: "http://127.0.0.1:3690"},
		},
		Desktop:      &recordingDesktop{recorder: recorder},
		Events:       events,
		Live:         live,
		Coordination: coordination,
	})
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	result := app.RequestTerminalActivation(LiveTerminalPayload{
		TerminalID:   "terminal-1",
		TerminalName: "Overseer",
		Tree: domain.ContentNode{
			ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{},
		},
		HackLevel: 1,
		IntroText: "ROBCO INDUSTRIES UNIFIED OPERATING SYSTEM",
	})
	if !result.OK || result.Error != "" {
		t.Fatalf("RequestTerminalActivation(valid) = %#v", result)
	}
	records := events.Records()
	last := records[len(records)-1]
	if last.Name != coordinationStateEvent {
		t.Fatalf("last bridge event = %#v, want coordination-state", last)
	}
	coordinationState, ok := last.Payload.(*domain.MasterCoordinationState)
	if !ok || coordinationState == nil || coordinationState.Broadcast == nil || coordinationState.Broadcast.ActiveTerminalID == nil || *coordinationState.Broadcast.ActiveTerminalID != "terminal-1" {
		t.Fatalf("coordination-state payload = %#v", last.Payload)
	}

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if live.clearCalls != 1 {
		t.Fatalf("live Clear calls after shutdown = %d, want 1", live.clearCalls)
	}
}

func TestForceHackSuccessPublishesSolvedStateWithoutSpendingAttempt(t *testing.T) {
	recorder := &callRecorder{}
	events := &recordingEventSink{recorder: recorder}
	live := &recordingLiveService{
		forceState: &domain.PublicHackState{
			Level: 2, AttemptsMax: 4, AttemptsLeft: 2, Solved: true,
			Patterns: []domain.PublicHackPattern{{ID: "opaque-generation-pattern", Row: 0, Start: 0, End: 1, Used: false}},
		},
	}
	app := NewAppWithDependencies(AppDependencies{
		Live: live, Player: &recordingPlayerServer{recorder: recorder}, Events: events,
	})

	if result := app.ForceHackSuccess(); !result.OK {
		t.Fatalf("ForceHackSuccess() = %#v", result)
	}
	if live.forceCalls != 1 {
		t.Fatalf("ForceHackSuccess calls = %d, want 1", live.forceCalls)
	}
	records := events.Records()
	if len(records) != 1 {
		t.Fatalf("hack-state event count = %d, want 1", len(records))
	}
	state, ok := records[0].Payload.(*domain.PublicHackState)
	if !ok || state == nil || !state.Solved || state.AttemptsLeft != 2 || len(state.Patterns) != 1 {
		t.Fatalf("forced public state = %#v", records[0].Payload)
	}
}

func TestForceHackSuccessRejectsIneligiblePuzzleWithoutPublication(t *testing.T) {
	recorder := &callRecorder{}
	live := &recordingLiveService{}
	app := NewAppWithDependencies(AppDependencies{
		Live: live, Player: &recordingPlayerServer{recorder: recorder}, Events: &recordingEventSink{recorder: recorder},
	})

	result := app.ForceHackSuccess()
	if result.OK || result.Error == "" {
		t.Fatalf("ForceHackSuccess() = %#v, want structured ineligible rejection", result)
	}
	if live.forceCalls != 1 || len(recorder.Calls()) != 0 {
		t.Fatalf("ineligible force calls=%d publications=%v, want one validation and no publication", live.forceCalls, recorder.Calls())
	}
}

func TestForceHackSuccessPrefersOrderedCoordinatorAndPublishesHackStatus(t *testing.T) {
	recorder := &callRecorder{}
	coordination := &recordingCoordinatedHackService{
		recordingCoordinationService: recordingCoordinationService{state: &domain.MasterCoordinationState{Revision: 8}},
		forceState: &domain.PublicHackState{
			Level: 2, AttemptsMax: 4, AttemptsLeft: 2, Solved: true,
			Log: []string{"Exact private completion"},
		},
	}
	legacy := &recordingLiveService{forceState: &domain.PublicHackState{Solved: true}}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: coordination,
		Live:         legacy,
		Events:       &recordingEventSink{recorder: recorder},
	})

	if result := app.ForceHackSuccess(); !result.OK || result.Error != "" {
		t.Fatalf("ForceHackSuccess() = %#v, want ordered success", result)
	}
	if coordination.forceCalls != 1 || legacy.forceCalls != 0 {
		t.Fatalf("force calls coordinator=%d legacy=%d, want 1/0", coordination.forceCalls, legacy.forceCalls)
	}
	status := app.GetRuntimeStatus()
	if status.HackState == nil || !status.HackState.Solved || status.HackState.AttemptsLeft != 2 {
		t.Fatalf("ordered force status = %#v", status.HackState)
	}
	records := recorder.Calls()
	if !reflect.DeepEqual(records, []string{"event:hack-state"}) {
		t.Fatalf("ordered force publications = %v, want one hack-state event", records)
	}
}

func TestForceHackSuccessDoesNotBypassCoordinatorRejection(t *testing.T) {
	recorder := &callRecorder{}
	coordination := &recordingCoordinatedHackService{
		recordingCoordinationService: recordingCoordinationService{state: &domain.MasterCoordinationState{Revision: 8}},
	}
	legacy := &recordingLiveService{forceState: &domain.PublicHackState{Solved: true}}
	app := NewAppWithDependencies(AppDependencies{
		Coordination: coordination,
		Live:         legacy,
		Events:       &recordingEventSink{recorder: recorder},
	})

	result := app.ForceHackSuccess()
	if result.OK || result.Error == "" {
		t.Fatalf("ForceHackSuccess() = %#v, want coordinator rejection", result)
	}
	if coordination.forceCalls != 1 || legacy.forceCalls != 0 || len(recorder.Calls()) != 0 {
		t.Fatalf("rejected ordered force calls coordinator=%d legacy=%d events=%v", coordination.forceCalls, legacy.forceCalls, recorder.Calls())
	}
}

func TestForceHackSuccessUsesProductionCoordinatorOwnedRuntime(t *testing.T) {
	liveService := liveservice.New(nil, nil)
	var effects []controlservice.Effect
	coordination := controlservice.New(controlservice.Config{
		Runtime: liveService, Terminals: liveService, TrustedHack: liveService,
		Enqueue: func(effect controlservice.Effect) { effects = append(effects, effect) },
	})
	if _, err := coordination.StartBroadcast(); err != nil {
		t.Fatal(err)
	}
	if _, err := coordination.RequestTerminalActivation(domain.TerminalTarget{
		TerminalID: "terminal-force-app", TerminalName: "Force App", HackLevel: 1, IntroText: "WELCOME",
		Tree: domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT"},
	}); err != nil {
		t.Fatal(err)
	}
	if legacy := liveService.Snapshot(); legacy != nil {
		t.Fatalf("legacy live slot unexpectedly owns production runtime: %#v", legacy)
	}
	beforeRevision := coordination.Revision()
	app := NewAppWithDependencies(AppDependencies{
		Coordination: coordination,
		Live:         liveService,
		Events:       &recordingEventSink{recorder: &callRecorder{}},
	})

	if result := app.ForceHackSuccess(); !result.OK || result.Error != "" {
		t.Fatalf("ForceHackSuccess() = %#v", result)
	}
	if coordination.Revision() != beforeRevision+1 {
		t.Fatalf("force revision = %d, want %d", coordination.Revision(), beforeRevision+1)
	}
	status := app.GetRuntimeStatus()
	if status.HackState == nil || !status.HackState.Solved || status.HackState.Failed {
		t.Fatalf("app hack status = %#v", status.HackState)
	}
	var published *domain.PublicLiveState
	for _, effect := range effects {
		if effect.Revision == coordination.Revision() && effect.Live != nil {
			published = effect.Live
		}
	}
	if published == nil || published.TerminalID != "terminal-force-app" || published.Hack == nil || !published.Hack.Solved {
		t.Fatalf("coordinator publication = %#v", published)
	}
	if legacy := liveService.Snapshot(); legacy != nil {
		t.Fatalf("trusted app force populated legacy live slot: %#v", legacy)
	}
	if result := app.ForceHackSuccess(); result.OK || result.Error == "" || coordination.Revision() != beforeRevision+1 {
		t.Fatalf("repeated ForceHackSuccess() = %#v revision=%d", result, coordination.Revision())
	}
}

func TestPlayerCallbacksEmitAndRetainDetachedPublicStatus(t *testing.T) {
	recorder := &callRecorder{}
	events := &recordingEventSink{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{Events: events})
	hackState := &domain.PublicHackState{
		Level: 3, AttemptsMax: 4, AttemptsLeft: 2,
		Log:      []string{"ENTRY DENIED"},
		Patterns: []domain.PublicHackPattern{{ID: "opaque-generation-pattern", Row: 0, Start: 0, End: 1}},
	}

	app.updateClientCount(6)
	app.updateHackState(hackState)
	hackState.AttemptsLeft = 0
	hackState.Log[0] = "MUTATED"
	hackState.Patterns[0].ID = "mutated"
	hackState.Patterns[0].Row = 99
	hackState.Patterns[0].Used = true

	if got, want := recorder.Calls(), []string{"event:client-count", "event:hack-state"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("player callback events = %v, want %v", got, want)
	}
	status := app.GetRuntimeStatus()
	if status.ClientCount != 6 || status.HackState == nil || status.HackState.AttemptsLeft != 2 || status.HackState.Log[0] != "ENTRY DENIED" || status.HackState.Patterns[0].ID != "opaque-generation-pattern" || status.HackState.Patterns[0].Row != 0 || status.HackState.Patterns[0].Used {
		t.Fatalf("detached player callback status = %#v", status)
	}
}

type callRecorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *callRecorder) Add(call string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

func (r *callRecorder) Calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func (r *callRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = nil
}

type recordingPlayerServer struct {
	recorder *callRecorder
	info     domain.ServerInfo
	startErr error
}

type recordingSessionService struct {
	recorder      *callRecorder
	shutdownCalls int
}

type recordingPlayerConfigSession struct {
	recordingSessionService
	snapshot     sessionservice.ActiveSession
	associations []string
}

func (service *recordingPlayerConfigSession) Snapshot() sessionservice.ActiveSession {
	return service.snapshot
}

func (service *recordingPlayerConfigSession) AssociatePlayerConfig(_ context.Context, path string) sessionservice.SessionResult {
	service.associations = append(service.associations, path)
	if service.snapshot.Session == nil {
		return sessionservice.SessionResult{Error: "no active session"}
	}
	copy := *service.snapshot.Session
	copy.PlayerConfig = "players/shared.json"
	service.snapshot.Session = &copy
	return sessionservice.SessionResult{OK: true, FilePath: service.snapshot.Path, Session: &copy}
}

type recordingPlayerConfigService struct {
	next playerconfigservice.Result
}

func (service *recordingPlayerConfigService) Create(context.Context) playerconfigservice.Result {
	return service.next
}
func (service *recordingPlayerConfigService) Open(context.Context) playerconfigservice.Result {
	return service.next
}
func (service *recordingPlayerConfigService) LoadReferenced(string, string) playerconfigservice.Result {
	return service.next
}

type recordingPlayerConfigCoordination struct {
	recordingCoordinationService
	installs []string
}

func (service *recordingPlayerConfigCoordination) InstallPlayerConfig(handle domain.PlayerConfigHandle, roster []domain.CharacterRosterEntry) (*domain.MasterCoordinationState, error) {
	entry := handle.Name + ":"
	if len(roster) > 0 {
		entry += string(roster[0].ID)
	}
	service.installs = append(service.installs, entry)
	state := domain.CloneMasterCoordinationState(service.state)
	state.PlayerConfig = &domain.PlayerConfigMetadata{Status: "loaded", Name: handle.Name, FilePath: handle.Path, Version: handle.Version}
	state.Roster = make([]domain.MasterRosterEntry, len(roster))
	for index, character := range roster {
		state.Roster[index] = domain.MasterRosterEntry{ID: character.ID, Name: character.Name}
	}
	service.state = state
	return domain.CloneMasterCoordinationState(state), nil
}

func (service *recordingPlayerConfigCoordination) ClearPlayerConfig() (*domain.MasterCoordinationState, error) {
	state := domain.CloneMasterCoordinationState(service.state)
	state.PlayerConfig = nil
	state.Roster = []domain.MasterRosterEntry{}
	service.state = state
	return domain.CloneMasterCoordinationState(state), nil
}

func (service *recordingSessionService) Shutdown(context.Context) error {
	service.shutdownCalls++
	if service.recorder != nil {
		service.recorder.Add("session:shutdown")
	}
	return nil
}

type recordingLiveService struct {
	setState    *domain.PublicLiveState
	updateState *domain.PublicLiveState
	forceState  *domain.PublicHackState
	setCalls    int
	updateCalls int
	clearCalls  int
	forceCalls  int
}

type recordingCoordinationService struct {
	state      *domain.MasterCoordinationState
	addState   *domain.MasterCoordinationState
	startState *domain.MasterCoordinationState
	addErr     error
	startErr   error
	addNames   []string
	startCalls int
}

type recordingCoordinatedHackService struct {
	recordingCoordinationService
	forceState *domain.PublicHackState
	forceCalls int
}

type recordingTerminalCoordinationService struct {
	recordingCoordinationService
	order            *callRecorder
	targets          []domain.TerminalTarget
	clearCalls       int
	updateCalls      int
	updateTree       domain.ContentNode
	updateIntro      *string
	commandErr       error
	decisionRequired bool
	nextSwitchID     domain.SwitchID
	decisions        []recordedTerminalDecision
	forceState       *domain.PublicHackState
	forceCalls       int
	resetTargets     []domain.TerminalTarget
}

type recordingBroadcastLifecycleService struct {
	recordingCoordinationService
	endState      *domain.MasterCoordinationState
	endErr        error
	endCalls      int
	shutdownCalls int
}

func (service *recordingBroadcastLifecycleService) EndBroadcast() (*domain.MasterCoordinationState, error) {
	service.endCalls++
	if service.endErr != nil {
		return domain.CloneMasterCoordinationState(service.state), service.endErr
	}
	service.state = domain.CloneMasterCoordinationState(service.endState)
	return domain.CloneMasterCoordinationState(service.state), nil
}

func (service *recordingBroadcastLifecycleService) Shutdown() { service.shutdownCalls++ }

type recordedTerminalDecision struct {
	SwitchID domain.SwitchID
	Decision domain.TerminalSwitchChoice
}

func (service *recordingTerminalCoordinationService) RequestTerminalActivation(target domain.TerminalTarget) (*domain.MasterCoordinationState, error) {
	service.targets = append(service.targets, target)
	if service.order != nil {
		service.order.Add("coordinator:request-terminal-activation:" + target.TerminalID)
	}
	if service.commandErr != nil {
		return domain.CloneMasterCoordinationState(service.state), service.commandErr
	}
	state := domain.CloneMasterCoordinationState(service.state)
	state.Revision++
	if service.decisionRequired {
		state.PendingSwitch = &domain.MasterPendingSwitch{
			SwitchID: service.nextSwitchID, BroadcastID: state.Broadcast.ID,
			SourceTerminalID: *state.Broadcast.ActiveTerminalID, TargetTerminalID: appStringPointer(target.TerminalID),
		}
		service.state = state
		return domain.CloneMasterCoordinationState(state), nil
	}
	activeTerminalID := target.TerminalID
	state.Broadcast.ActiveTerminalID = &activeTerminalID
	service.state = state
	return domain.CloneMasterCoordinationState(state), nil
}

func (service *recordingTerminalCoordinationService) RequestTerminalClear() (*domain.MasterCoordinationState, error) {
	service.clearCalls++
	if service.order != nil {
		service.order.Add("coordinator:request-terminal-clear")
	}
	if service.commandErr != nil {
		return domain.CloneMasterCoordinationState(service.state), service.commandErr
	}
	state := domain.CloneMasterCoordinationState(service.state)
	state.Revision++
	if service.decisionRequired {
		state.PendingSwitch = &domain.MasterPendingSwitch{
			SwitchID: service.nextSwitchID, BroadcastID: state.Broadcast.ID,
			SourceTerminalID: *state.Broadcast.ActiveTerminalID,
		}
		service.state = state
		return domain.CloneMasterCoordinationState(state), nil
	}
	state.Broadcast.ActiveTerminalID = nil
	service.state = state
	return domain.CloneMasterCoordinationState(state), nil
}

func (service *recordingTerminalCoordinationService) ResetFailedHack(target domain.TerminalTarget) (*domain.MasterCoordinationState, error) {
	service.resetTargets = append(service.resetTargets, target)
	if service.order != nil {
		service.order.Add("coordinator:reset-failed-hack:" + target.TerminalID)
	}
	if service.commandErr != nil {
		return domain.CloneMasterCoordinationState(service.state), service.commandErr
	}
	state := domain.CloneMasterCoordinationState(service.state)
	state.Revision++
	service.state = state
	return domain.CloneMasterCoordinationState(state), nil
}

func (service *recordingTerminalCoordinationService) ResolveTerminalSwitch(switchID domain.SwitchID, decision domain.TerminalSwitchChoice) (*domain.MasterCoordinationState, error) {
	if service.order != nil {
		service.order.Add("coordinator:resolve-terminal-switch:" + string(switchID) + ":" + string(decision))
	}
	if service.commandErr != nil {
		return domain.CloneMasterCoordinationState(service.state), service.commandErr
	}
	service.decisions = append(service.decisions, recordedTerminalDecision{SwitchID: switchID, Decision: decision})
	state := domain.CloneMasterCoordinationState(service.state)
	state.Revision++
	pending := state.PendingSwitch
	state.PendingSwitch = nil
	if decision != domain.TerminalSwitchCancel && pending != nil {
		state.Broadcast.ActiveTerminalID = pending.TargetTerminalID
	}
	service.state = state
	service.decisionRequired = false
	return domain.CloneMasterCoordinationState(state), nil
}

func (service *recordingTerminalCoordinationService) ForceHackSuccess() (*domain.PublicHackState, bool) {
	service.forceCalls++
	if service.forceState == nil {
		return nil, false
	}
	return clonePublicHackState(service.forceState), true
}

func (service *recordingTerminalCoordinationService) UpdateLiveTerminal(tree domain.ContentNode, introText *string) (*domain.MasterCoordinationState, error) {
	service.updateCalls++
	service.updateTree = tree
	if introText != nil {
		intro := *introText
		service.updateIntro = &intro
	}
	terminalID := ""
	if service.state != nil && service.state.Broadcast != nil && service.state.Broadcast.ActiveTerminalID != nil {
		terminalID = *service.state.Broadcast.ActiveTerminalID
	}
	if service.order != nil {
		service.order.Add("coordinator:update-live-terminal:" + terminalID)
	}
	if service.commandErr != nil {
		return domain.CloneMasterCoordinationState(service.state), service.commandErr
	}
	state := domain.CloneMasterCoordinationState(service.state)
	state.Revision++
	service.state = state
	return domain.CloneMasterCoordinationState(state), nil
}

type recordingCorrectionCoordinationService struct {
	recordingCoordinationService
	calls        []string
	nextRevision int
	failCommand  string
	commandErr   error
	order        *callRecorder
}

func (service *recordingCorrectionCoordinationService) correction(command string) (*domain.MasterCoordinationState, error) {
	if service.failCommand == command {
		return domain.CloneMasterCoordinationState(service.state), service.commandErr
	}
	state := domain.CloneMasterCoordinationState(service.state)
	state.Revision = uint64(20 + service.nextRevision)
	service.state = state
	return domain.CloneMasterCoordinationState(state), nil
}

func (service *recordingCorrectionCoordinationService) RenameCharacter(characterID domain.CharacterID, name string) (*domain.MasterCoordinationState, error) {
	service.calls = append(service.calls, fmt.Sprintf("rename-character:%s:%s", characterID, name))
	return service.correction("rename-character")
}

func (service *recordingCorrectionCoordinationService) DeleteCharacter(characterID domain.CharacterID) (*domain.MasterCoordinationState, error) {
	service.calls = append(service.calls, "delete-character:"+string(characterID))
	return service.correction("delete-character")
}

func (service *recordingCorrectionCoordinationService) RenameLogicalSession(sessionID domain.LogicalSessionID, fallbackName string) (*domain.MasterCoordinationState, error) {
	service.calls = append(service.calls, fmt.Sprintf("rename-session:%s:%s", sessionID, fallbackName))
	return service.correction("rename-session")
}

func (service *recordingCorrectionCoordinationService) AssignCharacter(sessionID domain.LogicalSessionID, characterID domain.CharacterID) (*domain.MasterCoordinationState, error) {
	service.calls = append(service.calls, fmt.Sprintf("assign-character:%s:%s", sessionID, characterID))
	return service.correction("assign-character")
}

func (service *recordingCorrectionCoordinationService) ReleaseCharacter(sessionID domain.LogicalSessionID) (*domain.MasterCoordinationState, error) {
	service.calls = append(service.calls, "release-character:"+string(sessionID))
	return service.correction("release-character")
}

func (service *recordingCorrectionCoordinationService) MoveCharacter(characterID domain.CharacterID, toSessionID domain.LogicalSessionID) (*domain.MasterCoordinationState, error) {
	service.calls = append(service.calls, fmt.Sprintf("move-character:%s:%s", characterID, toSessionID))
	return service.correction("move-character")
}

func (service *recordingCorrectionCoordinationService) SetActiveController(sessionID domain.LogicalSessionID) (*domain.MasterCoordinationState, error) {
	service.calls = append(service.calls, "set-controller:"+string(sessionID))
	if service.order != nil {
		service.order.Add("coordinator:set-controller:" + string(sessionID))
	}
	if service.failCommand == "set-active-controller" {
		return domain.CloneMasterCoordinationState(service.state), service.commandErr
	}
	state := domain.CloneMasterCoordinationState(service.state)
	state.Revision++
	controller := sessionID
	state.Broadcast.ControllerSessionID = &controller
	for index := range state.Sessions {
		if state.Sessions[index].Character == nil {
			state.Sessions[index].Role = domain.PlayerRoleUnassigned
		} else if state.Sessions[index].ID == sessionID {
			state.Sessions[index].Role = domain.PlayerRoleActive
		} else {
			state.Sessions[index].Role = domain.PlayerRoleObserver
		}
	}
	service.state = state
	return domain.CloneMasterCoordinationState(state), nil
}

func (service *recordingCoordinatedHackService) ForceHackSuccess() (*domain.PublicHackState, bool) {
	service.forceCalls++
	if service.forceState == nil {
		return nil, false
	}
	clone := *service.forceState
	clone.Log = append([]string(nil), service.forceState.Log...)
	clone.Columns = append([]domain.HackColumn(nil), service.forceState.Columns...)
	clone.Patterns = append([]domain.PublicHackPattern(nil), service.forceState.Patterns...)
	return &clone, true
}

func (service *recordingCoordinationService) Snapshot() *domain.MasterCoordinationState {
	return domain.CloneMasterCoordinationState(service.state)
}

func (service *recordingCoordinationService) AddCharacter(name string) (*domain.MasterCoordinationState, error) {
	service.addNames = append(service.addNames, name)
	if service.addErr != nil {
		return domain.CloneMasterCoordinationState(service.state), service.addErr
	}
	service.state = domain.CloneMasterCoordinationState(service.addState)
	return domain.CloneMasterCoordinationState(service.state), nil
}

func (service *recordingCoordinationService) StartBroadcast() (*domain.MasterCoordinationState, error) {
	service.startCalls++
	if service.startErr != nil {
		return domain.CloneMasterCoordinationState(service.state), service.startErr
	}
	service.state = domain.CloneMasterCoordinationState(service.startState)
	return domain.CloneMasterCoordinationState(service.state), nil
}

func (service *recordingLiveService) Set(string, string, domain.ContentNode, int, string) *domain.PublicLiveState {
	service.setCalls++
	return clonePublicLiveStateForTest(service.setState)
}

func (service *recordingLiveService) Update(domain.ContentNode, *string) (*domain.PublicLiveState, bool) {
	service.updateCalls++
	return clonePublicLiveStateForTest(service.updateState), service.updateState != nil
}

func (service *recordingLiveService) Clear() {
	service.clearCalls++
}

func (service *recordingLiveService) Snapshot() *domain.PublicLiveState {
	return clonePublicLiveStateForTest(service.setState)
}

func (service *recordingLiveService) ForceHackSuccess() (*domain.PublicHackState, bool) {
	service.forceCalls++
	if service.forceState == nil {
		return nil, false
	}
	clone := *service.forceState
	return &clone, true
}

func clonePublicLiveStateForTest(state *domain.PublicLiveState) *domain.PublicLiveState {
	if state == nil {
		return nil
	}
	clone := *state
	if state.Hack != nil {
		hackClone := *state.Hack
		clone.Hack = &hackClone
	}
	return &clone
}

func (server *recordingPlayerServer) Start(context.Context) (domain.ServerInfo, error) {
	server.recorder.Add("player:start")
	return server.info, server.startErr
}

func (server *recordingPlayerServer) Stop(context.Context) error {
	server.recorder.Add("player:stop")
	return nil
}

func (server *recordingPlayerServer) PublishLive() {
	server.recorder.Add("player:publish-live")
}

func (server *recordingPlayerServer) PublishUpdate() {
	server.recorder.Add("player:publish-update")
}

func (server *recordingPlayerServer) PublishClear() {
	server.recorder.Add("player:publish-clear")
}

func (server *recordingPlayerServer) PublishHack() {
	server.recorder.Add("player:publish-hack")
}

type recordingTunnel struct {
	recorder *callRecorder
	info     domain.ServerInfo
}

type invalidPublicTunnel struct {
	recorder        *callRecorder
	err             error
	validationCalls int
	processStarts   int
	stopCalls       int
}

func (tunnel *invalidPublicTunnel) Start(context.Context) (domain.ServerInfo, error) {
	tunnel.recorder.Add("tunnel:validate")
	tunnel.validationCalls++
	return domain.ServerInfo{}, tunnel.err
}

func (tunnel *invalidPublicTunnel) Stop(context.Context) error {
	tunnel.recorder.Add("tunnel:stop")
	tunnel.stopCalls++
	return nil
}

func (tunnel *recordingTunnel) Start(context.Context) (domain.ServerInfo, error) {
	tunnel.recorder.Add("tunnel:start")
	return tunnel.info, nil
}

func (tunnel *recordingTunnel) Stop(context.Context) error {
	tunnel.recorder.Add("tunnel:stop")
	return nil
}

type recordingEventSink struct {
	recorder *callRecorder
	err      error
	mu       sync.Mutex
	records  []eventRecord
}

type eventRecord struct {
	Name    string
	Payload any
}

func (sink *recordingEventSink) Emit(name string, payload any) error {
	sink.recorder.Add("event:" + name)
	sink.mu.Lock()
	sink.records = append(sink.records, eventRecord{Name: name, Payload: payload})
	sink.mu.Unlock()
	return sink.err
}

func (sink *recordingEventSink) Records() []eventRecord {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	return append([]eventRecord(nil), sink.records...)
}

type recordingDesktop struct {
	recorder *callRecorder
}

func (desktop *recordingDesktop) Ready(context.Context) error {
	desktop.recorder.Add("desktop:ready")
	return nil
}

func (desktop *recordingDesktop) Close(context.Context) error {
	desktop.recorder.Add("desktop:close")
	return nil
}
