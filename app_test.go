package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
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
	live := &recordingLiveService{}
	app := NewAppWithDependencies(AppDependencies{Live: live})

	setResult := app.SetLiveTerminal(LiveTerminalPayload{
		TerminalID:   "terminal-1",
		TerminalName: "Overseer",
		Tree: domain.ContentNode{
			ID: "root", Type: domain.NodeCommand, Name: "not a folder", Text: "invalid root",
		},
		HackLevel: 1,
	})
	if setResult.OK || setResult.Error == "" {
		t.Fatalf("SetLiveTerminal(invalid) = %#v, want structured validation error", setResult)
	}

	updateResult := app.UpdateLiveTerminal(LiveUpdatePayload{
		Tree: domain.ContentNode{ID: "root", Type: "script", Name: "unsupported"},
	})
	if updateResult.OK || updateResult.Error == "" {
		t.Fatalf("UpdateLiveTerminal(invalid) = %#v, want structured validation error", updateResult)
	}
	if live.setCalls != 0 || live.updateCalls != 0 {
		t.Fatalf("invalid live payloads reached canonical service: set=%d update=%d", live.setCalls, live.updateCalls)
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

func TestDOMReadyReplaysCurrentBridgeEvents(t *testing.T) {
	recorder := &callRecorder{}
	events := &recordingEventSink{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{Events: events})
	app.serverInfo = &domain.ServerInfo{IP: "127.0.0.1", Port: 3690, URL: "http://127.0.0.1:3690"}
	app.clientCount = 5
	app.hackState = &domain.PublicHackState{Level: 3, AttemptsMax: 4, AttemptsLeft: 2}

	app.domReady(context.Background())

	if got, want := recorder.Calls(), []string{"event:server-info", "event:client-count", "event:hack-state"}; !reflect.DeepEqual(got, want) {
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

func TestBridgeLiveLifecycleEmitsPublicStateAndCleansUp(t *testing.T) {
	recorder := &callRecorder{}
	live := &recordingLiveService{
		setState: &domain.PublicLiveState{
			TerminalID: "terminal-1",
			Hack:       &domain.PublicHackState{Level: 1, AttemptsMax: 4, AttemptsLeft: 4},
		},
	}
	events := &recordingEventSink{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{
		Sessions: &recordingSessionService{recorder: recorder},
		Player: &recordingPlayerServer{
			recorder: recorder,
			info:     domain.ServerInfo{IP: "127.0.0.1", Port: 3690, URL: "http://127.0.0.1:3690"},
		},
		Desktop: &recordingDesktop{recorder: recorder},
		Events:  events,
		Live:    live,
	})
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	result := app.SetLiveTerminal(LiveTerminalPayload{
		TerminalID:   "terminal-1",
		TerminalName: "Overseer",
		Tree: domain.ContentNode{
			ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{},
		},
		HackLevel: 1,
		IntroText: "ROBCO INDUSTRIES UNIFIED OPERATING SYSTEM",
	})
	if !result.OK || result.Error != "" {
		t.Fatalf("SetLiveTerminal(valid) = %#v", result)
	}
	records := events.Records()
	last := records[len(records)-1]
	if last.Name != "hack-state" {
		t.Fatalf("last bridge event = %#v, want hack-state", last)
	}
	hackState, ok := last.Payload.(*domain.PublicHackState)
	if !ok || hackState == nil || hackState.AttemptsLeft != 4 {
		t.Fatalf("hack-state payload = %#v", last.Payload)
	}

	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if live.clearCalls != 1 {
		t.Fatalf("live Clear calls after shutdown = %d, want 1", live.clearCalls)
	}
}

func TestBridgeLiveCommandsPublishPlayerAndPublicHackState(t *testing.T) {
	recorder := &callRecorder{}
	player := &recordingPlayerServer{recorder: recorder}
	live := &recordingLiveService{
		setState: &domain.PublicLiveState{
			TerminalID: "terminal-1",
			Hack:       &domain.PublicHackState{Level: 2, AttemptsMax: 4, AttemptsLeft: 4},
		},
		updateState: &domain.PublicLiveState{
			TerminalID: "terminal-1",
			Hack:       &domain.PublicHackState{Level: 2, AttemptsMax: 4, AttemptsLeft: 3},
		},
		forceState: &domain.PublicHackState{Level: 2, AttemptsMax: 4, AttemptsLeft: 3, Solved: true},
	}
	app := NewAppWithDependencies(AppDependencies{
		Live:   live,
		Player: player,
		Events: &recordingEventSink{recorder: recorder},
	})
	tree := domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{}}

	if result := app.SetLiveTerminal(LiveTerminalPayload{
		TerminalID: "terminal-1", TerminalName: "Overseer", Tree: tree, HackLevel: 2,
	}); !result.OK {
		t.Fatalf("SetLiveTerminal() = %#v", result)
	}
	if result := app.UpdateLiveTerminal(LiveUpdatePayload{Tree: tree}); !result.OK {
		t.Fatalf("UpdateLiveTerminal() = %#v", result)
	}
	if result := app.ForceHackSuccess(); !result.OK {
		t.Fatalf("ForceHackSuccess() = %#v", result)
	}
	if result := app.ClearLiveTerminal(); !result.OK {
		t.Fatalf("ClearLiveTerminal() = %#v", result)
	}

	want := []string{
		"player:publish-live", "event:hack-state",
		"player:publish-update", "event:hack-state",
		"player:publish-hack", "event:hack-state",
		"player:publish-clear", "event:hack-state",
	}
	if got := recorder.Calls(); !reflect.DeepEqual(got, want) {
		t.Fatalf("live publication calls = %v, want %v", got, want)
	}
	status := app.GetRuntimeStatus()
	if status.HackState != nil {
		t.Fatalf("hack state after clear = %#v, want nil", status.HackState)
	}
}

func TestPlayerCallbacksEmitAndRetainDetachedPublicStatus(t *testing.T) {
	recorder := &callRecorder{}
	events := &recordingEventSink{recorder: recorder}
	app := NewAppWithDependencies(AppDependencies{Events: events})
	hackState := &domain.PublicHackState{
		Level: 3, AttemptsMax: 4, AttemptsLeft: 2,
		Log: []string{"ENTRY DENIED"},
	}

	app.updateClientCount(6)
	app.updateHackState(hackState)
	hackState.AttemptsLeft = 0
	hackState.Log[0] = "MUTATED"

	if got, want := recorder.Calls(), []string{"event:client-count", "event:hack-state"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("player callback events = %v, want %v", got, want)
	}
	status := app.GetRuntimeStatus()
	if status.ClientCount != 6 || status.HackState == nil || status.HackState.AttemptsLeft != 2 || status.HackState.Log[0] != "ENTRY DENIED" {
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
	recorder *callRecorder
}

func (service *recordingSessionService) Shutdown(context.Context) error {
	service.recorder.Add("session:shutdown")
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
