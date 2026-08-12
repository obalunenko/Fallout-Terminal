package player

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"

	"github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/live"
)

func TestServerFirstHandshakeIssuesUniqueIdentityAndRefreshesPersonalizedRosterAfterSelectionConflict(t *testing.T) {
	coordinator := newUS1CoordinatorStub([]domain.PlayerRosterEntry{
		{ID: "character-mara", Name: "Mara", Status: domain.RosterStatusAvailable},
		{ID: "character-boone", Name: "Boone", Status: domain.RosterStatusAvailable},
	}, nil)
	server := startUS1TestServer(t, coordinator, live.New(nil, nil))

	first := dialPlayer(t, server.Info().URL)
	second := dialPlayer(t, server.Info().URL)

	writeJSON(t, first, map[string]any{"type": MessageSessionHello})
	firstWelcome := readMessage(t, first)
	assertUS1Welcome(t, firstWelcome, "broadcast-us1", "")

	writeJSON(t, second, map[string]any{"type": MessageSessionHello})
	secondWelcome := readMessage(t, second)
	assertUS1Welcome(t, secondWelcome, "broadcast-us1", "")

	if firstWelcome.BrowserToken == secondWelcome.BrowserToken {
		t.Fatalf("fresh handshakes reused browser token %q", firstWelcome.BrowserToken)
	}
	if firstWelcome.State.SessionID == secondWelcome.State.SessionID {
		t.Fatalf("fresh handshakes reused logical session %q", firstWelcome.State.SessionID)
	}
	if firstWelcome.State.FallbackName == secondWelcome.State.FallbackName {
		t.Fatalf("fresh handshakes reused fallback name %q", firstWelcome.State.FallbackName)
	}

	writeJSON(t, first, map[string]any{
		"type": MessageCharacterSelect, "requestId": "select-first",
		"broadcastId": "broadcast-us1", "characterId": "character-mara",
	})
	firstState := readMessage(t, first)
	secondRosterRefresh := readMessage(t, second)
	firstResult := readMessage(t, first)

	assertUS1AssignedState(t, firstState, MessagePlayerState, "character-mara", domain.PlayerRoleActive, domain.PlayerPhaseWaiting, "")
	assertUS1RosterClaimed(t, secondRosterRefresh, "character-mara")
	if secondRosterRefresh.State.Character != nil || secondRosterRefresh.State.Role != domain.PlayerRoleUnassigned || secondRosterRefresh.State.Phase != domain.PlayerPhaseSelecting {
		t.Fatalf("other claimant personalized refresh = %#v, want unassigned selection state", secondRosterRefresh.State)
	}
	assertUS1ActionResult(t, firstResult, "select-first", true, domain.ActionReason("accepted"))

	writeJSON(t, second, map[string]any{
		"type": MessageCharacterSelect, "requestId": "select-conflict",
		"broadcastId": "broadcast-us1", "characterId": "character-mara",
	})
	conflictState := readMessage(t, second)
	conflictResult := readMessage(t, second)
	assertUS1RosterClaimed(t, conflictState, "character-mara")
	if conflictState.State.Character != nil || conflictState.State.Role != domain.PlayerRoleUnassigned || conflictState.State.Phase != domain.PlayerPhaseSelecting {
		t.Fatalf("conflicting claimant state = %#v, want unchanged unassigned selection state", conflictState.State)
	}
	assertUS1ActionResult(t, conflictResult, "select-conflict", false, domain.ActionReasonConflict)
}

func TestServerAssignedSessionReceivesCurrentTerminalOnlyAfterHandshakeAndSuccessfulSelection(t *testing.T) {
	liveService := live.New(nil, nil)
	liveState := liveService.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	coordinator := newUS1CoordinatorStub([]domain.PlayerRosterEntry{
		{ID: "character-mara", Name: "Mara", Status: domain.RosterStatusAvailable},
	}, liveState)
	server := startUS1TestServer(t, coordinator, liveService)
	connection := dialPlayer(t, server.Info().URL)

	writeJSON(t, connection, map[string]any{"type": MessageSessionHello})
	welcome := readMessage(t, connection)
	assertUS1Welcome(t, welcome, "broadcast-us1", "terminal-1")
	if welcome.State.Phase != domain.PlayerPhaseSelecting || welcome.State.Character != nil {
		t.Fatalf("pre-selection welcome = %#v, want selection gated before terminal delivery", welcome.State)
	}

	writeJSON(t, connection, map[string]any{
		"type": MessageCharacterSelect, "requestId": "select-terminal",
		"broadcastId": "broadcast-us1", "characterId": "character-mara",
	})
	assigned := readMessage(t, connection)
	terminal := readMessage(t, connection)
	result := readMessage(t, connection)

	assertUS1AssignedState(t, assigned, MessagePlayerState, "character-mara", domain.PlayerRoleActive, domain.PlayerPhaseControlling, "terminal-1")
	if terminal.Type != MessageTerminalLive || terminal.TerminalID != "terminal-1" || terminal.Revision != assigned.State.Revision {
		t.Fatalf("assigned terminal delivery = %#v, want revision %d terminal-1", terminal, assigned.State.Revision)
	}
	assertUS1ActionResult(t, result, "select-terminal", true, domain.ActionReason("accepted"))
	if result.Revision != terminal.Revision {
		t.Fatalf("selection result revision = %d, want delivered state revision %d", result.Revision, terminal.Revision)
	}
}

func TestServerThreeTabsShareOneLogicalSessionUntilTheFinalClose(t *testing.T) {
	liveService := live.New(nil, nil)
	liveState := liveService.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	coordinator := newUS3CoordinatorStub(liveState)
	server := startUS3TestServer(t, coordinator, liveService)

	first, firstWelcome := connectUS3Player(t, server.Info().URL, "")
	writeJSON(t, first, map[string]any{
		"type": MessageCharacterSelect, "requestId": "select-us3",
		"broadcastId": "broadcast-us3", "characterId": "character-mara",
	})
	assigned := readMessage(t, first)
	terminal := readMessage(t, first)
	result := readMessage(t, first)
	assertUS1AssignedState(t, assigned, MessagePlayerState, "character-mara", domain.PlayerRoleActive, domain.PlayerPhaseControlling, "terminal-1")
	if terminal.Type != MessageTerminalLive || !result.Accepted {
		t.Fatalf("initial assignment messages = terminal %#v result %#v", terminal, result)
	}

	second, secondWelcome := connectUS3Player(t, server.Info().URL, domain.BrowserToken(firstWelcome.BrowserToken))
	third, thirdWelcome := connectUS3Player(t, server.Info().URL, domain.BrowserToken(firstWelcome.BrowserToken))
	for index, welcome := range []serverMessage{secondWelcome, thirdWelcome} {
		if welcome.BrowserToken != firstWelcome.BrowserToken || welcome.State == nil || welcome.State.SessionID != firstWelcome.State.SessionID {
			t.Fatalf("tab %d welcome = %#v, want shared token/session", index+2, welcome)
		}
		if welcome.State.Character == nil || welcome.State.Role != domain.PlayerRoleActive || welcome.State.Phase != domain.PlayerPhaseControlling {
			t.Fatalf("tab %d lost claim/control: %#v", index+2, welcome.State)
		}
		if snapshot := readMessage(t, []*websocket.Conn{second, third}[index]); snapshot.Type != MessageTerminalLive || snapshot.TerminalID != "terminal-1" {
			t.Fatalf("tab %d reconnect terminal = %#v", index+2, snapshot)
		}
	}

	waitForUS3Connections(t, coordinator, firstWelcome.State.SessionID, 3)
	_ = second.Close(websocket.StatusNormalClosure, "close one tab")
	waitForUS3Connections(t, coordinator, firstWelcome.State.SessionID, 2)
	if !coordinator.connected(firstWelcome.State.SessionID) {
		t.Fatal("one tab close marked shared logical session absent")
	}
	_ = third.Close(websocket.StatusNormalClosure, "close second tab")
	waitForUS3Connections(t, coordinator, firstWelcome.State.SessionID, 1)
	if !coordinator.connected(firstWelcome.State.SessionID) {
		t.Fatal("second tab close marked remaining logical session absent")
	}
	_ = first.Close(websocket.StatusNormalClosure, "close final tab")
	waitForUS3Connections(t, coordinator, firstWelcome.State.SessionID, 0)
	if coordinator.connected(firstWelcome.State.SessionID) {
		t.Fatal("final tab close retained logical presence")
	}
}

func TestServerReconnectAndUnknownTokensRestoreOnlyRecognizedLogicalSession(t *testing.T) {
	liveService := live.New(nil, nil)
	liveState := liveService.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	coordinator := newUS3CoordinatorStub(liveState)
	server := startUS3TestServer(t, coordinator, liveService)

	first, welcome := connectUS3Player(t, server.Info().URL, "")
	writeJSON(t, first, map[string]any{
		"type": MessageCharacterSelect, "requestId": "select-reconnect",
		"broadcastId": "broadcast-us3", "characterId": "character-mara",
	})
	readMessage(t, first)
	readMessage(t, first)
	readMessage(t, first)
	_ = first.Close(websocket.StatusNormalClosure, "refresh")
	waitForUS3Connections(t, coordinator, welcome.State.SessionID, 0)

	reconnected, reconnectWelcome := connectUS3Player(t, server.Info().URL, domain.BrowserToken(welcome.BrowserToken))
	if reconnectWelcome.BrowserToken != welcome.BrowserToken || reconnectWelcome.State == nil || reconnectWelcome.State.SessionID != welcome.State.SessionID {
		t.Fatalf("recognized reconnect = %#v, want original token/session", reconnectWelcome)
	}
	if reconnectWelcome.State.Character == nil || reconnectWelcome.State.Role != domain.PlayerRoleActive {
		t.Fatalf("recognized reconnect lost assignment/control: %#v", reconnectWelcome.State)
	}
	if snapshot := readMessage(t, reconnected); snapshot.Type != MessageTerminalLive || snapshot.TerminalID != "terminal-1" {
		t.Fatalf("recognized reconnect canonical snapshot = %#v", snapshot)
	}

	unknown, unknownWelcome := connectUS3Player(t, server.Info().URL, "unknown-prior-process-token")
	defer unknown.CloseNow()
	if unknownWelcome.BrowserToken == "unknown-prior-process-token" || unknownWelcome.BrowserToken == welcome.BrowserToken {
		t.Fatalf("unknown token was not replaced: %#v", unknownWelcome)
	}
	if unknownWelcome.State == nil || unknownWelcome.State.SessionID == welcome.State.SessionID || unknownWelcome.State.Character != nil {
		t.Fatalf("unknown token inherited recognized state: %#v", unknownWelcome.State)
	}

	other, otherWelcome := connectUS3Player(t, server.Info().URL, "")
	defer other.CloseNow()
	if otherWelcome.BrowserToken == welcome.BrowserToken || otherWelcome.BrowserToken == unknownWelcome.BrowserToken || otherWelcome.State.SessionID == welcome.State.SessionID {
		t.Fatalf("different profile reused another identity: %#v", otherWelcome)
	}

	restarted := newUS3CoordinatorStub(liveState)
	restartedServer := startUS3TestServer(t, restarted, liveService)
	stale, staleWelcome := connectUS3Player(t, restartedServer.Info().URL, domain.BrowserToken(welcome.BrowserToken))
	defer stale.CloseNow()
	if staleWelcome.BrowserToken == welcome.BrowserToken || staleWelcome.State == nil || staleWelcome.State.SessionID == welcome.State.SessionID || staleWelcome.State.Character != nil {
		t.Fatalf("prior-process token restored stale state after restart: %#v", staleWelcome)
	}
}

func TestServerRosterAndAssignmentEffectsFanOutCompletePrivatePlayerStates(t *testing.T) {
	coordinator := newUS4CoordinatorStub()
	server := startUS4TestServer(t, coordinator)

	first, firstWelcome := connectUS3Player(t, server.Info().URL, "")
	sibling, siblingWelcome := connectUS3Player(t, server.Info().URL, domain.BrowserToken(firstWelcome.BrowserToken))
	other, otherWelcome := connectUS3Player(t, server.Info().URL, "")
	if siblingWelcome.State.SessionID != firstWelcome.State.SessionID || otherWelcome.State.SessionID == firstWelcome.State.SessionID {
		t.Fatalf("US4 session setup = first %#v sibling %#v other %#v", firstWelcome.State, siblingWelcome.State, otherWelcome.State)
	}

	ownerID := firstWelcome.State.SessionID
	targetID := otherWelcome.State.SessionID
	if !coordinator.AssignCharacter(ownerID) {
		t.Fatal("initial game-master assignment was rejected")
	}
	assignedFirst, assignedFirstJSON := readMessageWithPayload(t, first)
	assignedSibling := readMessage(t, sibling)
	assignedOther, assignedOtherJSON := readMessageWithPayload(t, other)
	assertUS4Assigned(t, assignedFirst, ownerID, "Mara", domain.PlayerRoleActive)
	if !reflect.DeepEqual(assignedFirst, assignedSibling) {
		t.Fatalf("same-session tabs diverged after assignment: first %#v sibling %#v", assignedFirst, assignedSibling)
	}
	assertUS4Selecting(t, assignedOther, targetID, "Mara", domain.RosterStatusClaimed)
	assertPrivatePlayerStateJSON(t, assignedFirstJSON, targetID, otherWelcome.State.FallbackName)
	assertPrivatePlayerStateJSON(t, assignedOtherJSON, ownerID, firstWelcome.State.FallbackName)

	if !coordinator.RenameCharacter("Mara Vance") {
		t.Fatal("character rename was rejected")
	}
	renamedFirst := readMessage(t, first)
	renamedSibling := readMessage(t, sibling)
	renamedOther := readMessage(t, other)
	for index, message := range []serverMessage{renamedFirst, renamedSibling, renamedOther} {
		if message.State == nil || len(message.State.Roster) != 1 || message.State.Roster[0].Name != "Mara Vance" {
			t.Fatalf("tab %d roster rename = %#v", index, message)
		}
	}
	if renamedFirst.State.Character == nil || renamedFirst.State.Character.Name != "Mara Vance" {
		t.Fatalf("assigned identity did not follow stable character rename: %#v", renamedFirst.State)
	}

	if !coordinator.RenameLogicalSession(ownerID, "PIP-BOY ALPHA") {
		t.Fatal("logical-session rename was rejected")
	}
	for index, connection := range []*websocket.Conn{first, sibling} {
		message := readMessage(t, connection)
		if message.State == nil || message.State.SessionID != ownerID || message.State.FallbackName != "PIP-BOY ALPHA" || message.State.Character == nil {
			t.Fatalf("same-session tab %d fallback rename = %#v", index, message)
		}
	}

	if !coordinator.MoveCharacter(targetID) {
		t.Fatal("atomic character transfer was rejected")
	}
	releasedFirst, releasedFirstJSON := readMessageWithPayload(t, first)
	releasedSibling := readMessage(t, sibling)
	transferredOther, transferredOtherJSON := readMessageWithPayload(t, other)
	assertUS4Selecting(t, releasedFirst, ownerID, "Mara Vance", domain.RosterStatusClaimed)
	if !reflect.DeepEqual(releasedFirst, releasedSibling) {
		t.Fatalf("same-session tabs diverged after transfer: first %#v sibling %#v", releasedFirst, releasedSibling)
	}
	assertUS4Assigned(t, transferredOther, targetID, "Mara Vance", domain.PlayerRoleObserver)
	assertPrivatePlayerStateJSON(t, releasedFirstJSON, targetID, otherWelcome.State.FallbackName)
	assertPrivatePlayerStateJSON(t, transferredOtherJSON, ownerID, "PIP-BOY ALPHA")

	revisionBeforeDelete, effectsBeforeDelete := coordinator.counters()
	if coordinator.DeleteCharacter() {
		t.Fatal("claimed character deletion was accepted")
	}
	if revision, effects := coordinator.counters(); revision != revisionBeforeDelete || effects != effectsBeforeDelete {
		t.Fatalf("claimed delete changed coordinator publication: revision/effects = %d/%d, want %d/%d", revision, effects, revisionBeforeDelete, effectsBeforeDelete)
	}

	if !coordinator.ReleaseCharacter(targetID) {
		t.Fatal("game-master release was rejected")
	}
	for index, connection := range []*websocket.Conn{first, sibling, other} {
		message, payload := readMessageWithPayload(t, connection)
		sessionID := ownerID
		privateSessionID := targetID
		privateFallback := otherWelcome.State.FallbackName
		if index == 2 {
			sessionID = targetID
			privateSessionID = ownerID
			privateFallback = "PIP-BOY ALPHA"
		}
		assertUS4Selecting(t, message, sessionID, "Mara Vance", domain.RosterStatusAvailable)
		assertPrivatePlayerStateJSON(t, payload, privateSessionID, privateFallback)
	}
}

func assertUS4Assigned(t *testing.T, message serverMessage, sessionID domain.LogicalSessionID, characterName string, role domain.PlayerRole) {
	t.Helper()
	if message.Type != MessagePlayerState || message.State == nil || message.State.SessionID != sessionID || message.State.Character == nil || message.State.Character.Name != characterName || message.State.Role != role {
		t.Fatalf("assigned player state = %#v, want %q assigned to %q as %q", message, sessionID, characterName, role)
	}
	if len(message.State.Roster) != 1 || message.State.Roster[0].Name != characterName || message.State.Roster[0].Status != domain.RosterStatusClaimed {
		t.Fatalf("assigned roster = %#v, want claimed %q", message.State.Roster, characterName)
	}
}

func assertUS4Selecting(t *testing.T, message serverMessage, sessionID domain.LogicalSessionID, characterName string, status domain.RosterStatus) {
	t.Helper()
	if message.Type != MessagePlayerState || message.State == nil || message.State.SessionID != sessionID || message.State.Character != nil || message.State.Role != domain.PlayerRoleUnassigned || message.State.Phase != domain.PlayerPhaseSelecting {
		t.Fatalf("selection player state = %#v, want unassigned session %q", message, sessionID)
	}
	if len(message.State.Roster) != 1 || message.State.Roster[0].Name != characterName || message.State.Roster[0].Status != status {
		t.Fatalf("selection roster = %#v, want %q/%q", message.State.Roster, characterName, status)
	}
}

func assertPrivatePlayerStateJSON(t *testing.T, payload []byte, privateSessionID domain.LogicalSessionID, privateFallback string) {
	t.Helper()
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode player envelope %s: %v", payload, err)
	}
	for key := range envelope {
		if key != "type" && key != "state" {
			t.Fatalf("player envelope exposed unexpected key %q in %s", key, payload)
		}
	}
	var state map[string]json.RawMessage
	if err := json.Unmarshal(envelope["state"], &state); err != nil {
		t.Fatalf("decode player state %s: %v", envelope["state"], err)
	}
	allowedStateKeys := map[string]struct{}{
		"revision": {}, "sessionId": {}, "fallbackName": {}, "character": {}, "role": {}, "phase": {},
		"broadcastId": {}, "activeTerminalId": {}, "roster": {},
	}
	for key := range state {
		if _, allowed := allowedStateKeys[key]; !allowed {
			t.Fatalf("player state exposed private key %q in %s", key, payload)
		}
	}
	var roster []map[string]json.RawMessage
	if err := json.Unmarshal(state["roster"], &roster); err != nil {
		t.Fatalf("decode player roster %s: %v", state["roster"], err)
	}
	for _, entry := range roster {
		for key := range entry {
			if key != "id" && key != "name" && key != "status" {
				t.Fatalf("player roster exposed claimant/presence key %q in %s", key, payload)
			}
		}
	}
	if privateSessionID != "" && bytes.Contains(payload, []byte(privateSessionID)) {
		t.Fatalf("player JSON exposed another logical session %q: %s", privateSessionID, payload)
	}
	if privateFallback != "" && bytes.Contains(payload, []byte(privateFallback)) {
		t.Fatalf("player JSON exposed another session fallback %q: %s", privateFallback, payload)
	}
}

type us3CoordinatorStub struct {
	mu sync.Mutex

	revision     uint64
	next         int
	process      uint64
	live         *domain.PublicLiveState
	sessions     map[domain.LogicalSessionID]*us3Session
	byToken      map[domain.BrowserToken]domain.LogicalSessionID
	byConnection map[domain.ConnectionID]domain.LogicalSessionID
	claim        domain.LogicalSessionID
	enqueue      func(control.Effect)
}

var us3ProcessSequence atomic.Uint64

type us3Session struct {
	id          domain.LogicalSessionID
	token       domain.BrowserToken
	fallback    string
	connections map[domain.ConnectionID]struct{}
	assigned    bool
}

func newUS3CoordinatorStub(liveState *domain.PublicLiveState) *us3CoordinatorStub {
	return &us3CoordinatorStub{
		live: liveState, process: us3ProcessSequence.Add(1), sessions: make(map[domain.LogicalSessionID]*us3Session),
		byToken:      make(map[domain.BrowserToken]domain.LogicalSessionID),
		byConnection: make(map[domain.ConnectionID]domain.LogicalSessionID),
	}
}

func (stub *us3CoordinatorStub) AttachConnection(connectionID domain.ConnectionID, token domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	session := stub.sessions[stub.byToken[token]]
	if token == "" || session == nil {
		stub.next++
		session = &us3Session{
			id:       domain.LogicalSessionID(fmt.Sprintf("us3-%d-session-%d", stub.process, stub.next)),
			token:    domain.BrowserToken(fmt.Sprintf("us3-%d-token-%d", stub.process, stub.next)),
			fallback: fmt.Sprintf("PLAYER %d", stub.next), connections: make(map[domain.ConnectionID]struct{}),
		}
		stub.sessions[session.id] = session
		stub.byToken[session.token] = session.id
	}
	session.connections[connectionID] = struct{}{}
	stub.byConnection[connectionID] = session.id
	return session.token, stub.playerState(session)
}

func (stub *us3CoordinatorStub) DetachConnection(connectionID domain.ConnectionID) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	sessionID := stub.byConnection[connectionID]
	delete(stub.byConnection, connectionID)
	if session := stub.sessions[sessionID]; session != nil {
		delete(session.connections, connectionID)
	}
}

func (stub *us3CoordinatorStub) SelectCharacter(connectionID domain.ConnectionID, requestID string, broadcastID domain.BroadcastID, characterID domain.CharacterID) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	session := stub.sessions[stub.byConnection[connectionID]]
	if session == nil || broadcastID != "broadcast-us3" || characterID != "character-mara" || stub.claim != "" {
		stub.emit(control.Effect{ConnectionID: connectionID, Result: &domain.ActionResult{RequestID: requestID, Reason: domain.ActionReasonConflict, Revision: stub.revision}})
		return
	}
	stub.revision++
	session.assigned = true
	stub.claim = session.id
	stub.emit(control.Effect{SessionID: session.id, Player: stub.playerState(session)})
	stub.emit(control.Effect{SessionID: session.id, Live: stub.live})
	stub.emit(control.Effect{ConnectionID: connectionID, Result: &domain.ActionResult{RequestID: requestID, Accepted: true, Reason: domain.ActionReasonAccepted, Revision: stub.revision}})
}

func (stub *us3CoordinatorStub) playerState(session *us3Session) *domain.PlayerState {
	state := &domain.PlayerState{
		Revision: stub.revision, SessionID: session.id, FallbackName: session.fallback,
		Role: domain.PlayerRoleUnassigned, Phase: domain.PlayerPhaseSelecting,
		BroadcastID: "broadcast-us3", ActiveTerminalID: "terminal-1",
		Roster: []domain.PlayerRosterEntry{{ID: "character-mara", Name: "Mara", Status: domain.RosterStatusAvailable}},
	}
	if stub.claim != "" {
		state.Roster[0].Status = domain.RosterStatusClaimed
	}
	if session.assigned {
		state.Character = &domain.PlayerCharacter{ID: "character-mara", Name: "Mara"}
		state.Role = domain.PlayerRoleActive
		state.Phase = domain.PlayerPhaseControlling
	}
	return state
}

func (stub *us3CoordinatorStub) emit(effect control.Effect) {
	effect.Revision = stub.revision
	if stub.enqueue != nil {
		stub.enqueue(effect)
	}
}

func (stub *us3CoordinatorStub) connections(sessionID domain.LogicalSessionID) int {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if session := stub.sessions[sessionID]; session != nil {
		return len(session.connections)
	}
	return 0
}

func (stub *us3CoordinatorStub) connected(sessionID domain.LogicalSessionID) bool {
	return stub.connections(sessionID) > 0
}

func startUS3TestServer(t *testing.T, coordinator *us3CoordinatorStub, liveService *live.Service) *Server {
	t.Helper()
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")}}
	server, err := NewServer(Config{Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: liveService, Coordinator: coordinator})
	if err != nil {
		t.Fatal(err)
	}
	coordinator.enqueue = server.PublishCoordinationEffect
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

func connectUS3Player(t *testing.T, rawURL string, token domain.BrowserToken) (*websocket.Conn, serverMessage) {
	t.Helper()
	connection := dialPlayer(t, rawURL)
	hello := map[string]any{"type": MessageSessionHello}
	if token != "" {
		hello["browserToken"] = token
	}
	writeJSON(t, connection, hello)
	return connection, readMessage(t, connection)
}

func waitForUS3Connections(t *testing.T, coordinator *us3CoordinatorStub, sessionID domain.LogicalSessionID, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if coordinator.connections(sessionID) == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("session %q connections = %d, want %d", sessionID, coordinator.connections(sessionID), want)
}

// us4CoordinatorStub models the complete per-session projections produced by
// trusted roster and assignment operations. The server remains responsible
// only for routing each detached effect to every concrete tab of that session.
type us4CoordinatorStub struct {
	mu sync.Mutex

	revision     uint64
	next         int
	effectCount  int
	characterID  domain.CharacterID
	character    string
	controller   domain.LogicalSessionID
	sessions     map[domain.LogicalSessionID]*us4Session
	byToken      map[domain.BrowserToken]domain.LogicalSessionID
	byConnection map[domain.ConnectionID]domain.LogicalSessionID
	enqueue      func(control.Effect)
}

type us4Session struct {
	id          domain.LogicalSessionID
	token       domain.BrowserToken
	fallback    string
	characterID domain.CharacterID
	connections map[domain.ConnectionID]struct{}
}

func newUS4CoordinatorStub() *us4CoordinatorStub {
	return &us4CoordinatorStub{
		characterID: "character-mara", character: "Mara",
		sessions: make(map[domain.LogicalSessionID]*us4Session), byToken: make(map[domain.BrowserToken]domain.LogicalSessionID),
		byConnection: make(map[domain.ConnectionID]domain.LogicalSessionID),
	}
}

func (stub *us4CoordinatorStub) AttachConnection(connectionID domain.ConnectionID, token domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	session := stub.sessions[stub.byToken[token]]
	if token == "" || session == nil {
		stub.next++
		session = &us4Session{
			id: domain.LogicalSessionID(fmt.Sprintf("us4-session-%d", stub.next)), token: domain.BrowserToken(fmt.Sprintf("us4-token-%d", stub.next)),
			fallback: fmt.Sprintf("PLAYER %d", stub.next), connections: make(map[domain.ConnectionID]struct{}),
		}
		stub.sessions[session.id] = session
		stub.byToken[session.token] = session.id
	}
	session.connections[connectionID] = struct{}{}
	stub.byConnection[connectionID] = session.id
	return session.token, stub.playerStateLocked(session)
}

func (stub *us4CoordinatorStub) DetachConnection(connectionID domain.ConnectionID) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	sessionID := stub.byConnection[connectionID]
	delete(stub.byConnection, connectionID)
	if session := stub.sessions[sessionID]; session != nil {
		delete(session.connections, connectionID)
	}
}

func (*us4CoordinatorStub) SelectCharacter(domain.ConnectionID, string, domain.BroadcastID, domain.CharacterID) {
}

func (stub *us4CoordinatorStub) AssignCharacter(sessionID domain.LogicalSessionID) bool {
	return stub.change(func() bool {
		session := stub.sessions[sessionID]
		if session == nil || session.characterID != "" || stub.claimantLocked() != nil || stub.characterID == "" {
			return false
		}
		session.characterID = stub.characterID
		if stub.controller == "" {
			stub.controller = sessionID
		}
		return true
	}, true)
}

func (stub *us4CoordinatorStub) RenameCharacter(name string) bool {
	return stub.change(func() bool {
		if stub.characterID == "" || name == "" || name == stub.character {
			return false
		}
		stub.character = name
		return true
	}, true)
}

func (stub *us4CoordinatorStub) RenameLogicalSession(sessionID domain.LogicalSessionID, name string) bool {
	return stub.change(func() bool {
		session := stub.sessions[sessionID]
		if session == nil || name == "" || name == session.fallback {
			return false
		}
		session.fallback = name
		return true
	}, false, sessionID)
}

func (stub *us4CoordinatorStub) MoveCharacter(targetID domain.LogicalSessionID) bool {
	return stub.change(func() bool {
		target := stub.sessions[targetID]
		owner := stub.claimantLocked()
		if target == nil || owner == nil || owner.id == targetID || target.characterID != "" {
			return false
		}
		owner.characterID = ""
		target.characterID = stub.characterID
		if stub.controller == owner.id {
			stub.controller = ""
		}
		return true
	}, true)
}

func (stub *us4CoordinatorStub) DeleteCharacter() bool {
	return stub.change(func() bool {
		if stub.characterID == "" || stub.claimantLocked() != nil {
			return false
		}
		stub.characterID = ""
		stub.character = ""
		return true
	}, true)
}

func (stub *us4CoordinatorStub) ReleaseCharacter(sessionID domain.LogicalSessionID) bool {
	return stub.change(func() bool {
		session := stub.sessions[sessionID]
		if session == nil || session.characterID == "" {
			return false
		}
		session.characterID = ""
		if stub.controller == sessionID {
			stub.controller = ""
		}
		return true
	}, true)
}

func (stub *us4CoordinatorStub) change(mutate func() bool, allSessions bool, sessionIDs ...domain.LogicalSessionID) bool {
	stub.mu.Lock()
	if !mutate() {
		stub.mu.Unlock()
		return false
	}
	stub.revision++
	if allSessions {
		sessionIDs = sessionIDs[:0]
		for sessionID := range stub.sessions {
			sessionIDs = append(sessionIDs, sessionID)
		}
	}
	sort.Slice(sessionIDs, func(left, right int) bool { return sessionIDs[left] < sessionIDs[right] })
	effects := make([]control.Effect, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		if session := stub.sessions[sessionID]; session != nil {
			effects = append(effects, control.Effect{Revision: stub.revision, SessionID: sessionID, Player: stub.playerStateLocked(session)})
		}
	}
	stub.effectCount += len(effects)
	enqueue := stub.enqueue
	stub.mu.Unlock()
	for _, effect := range effects {
		if enqueue != nil {
			enqueue(effect)
		}
	}
	return true
}

func (stub *us4CoordinatorStub) claimantLocked() *us4Session {
	for _, session := range stub.sessions {
		if session != nil && session.characterID == stub.characterID && stub.characterID != "" {
			return session
		}
	}
	return nil
}

func (stub *us4CoordinatorStub) playerStateLocked(session *us4Session) *domain.PlayerState {
	state := &domain.PlayerState{
		Revision: stub.revision, SessionID: session.id, FallbackName: session.fallback,
		Role: domain.PlayerRoleUnassigned, Phase: domain.PlayerPhaseSelecting, BroadcastID: "broadcast-us4",
	}
	if stub.characterID != "" {
		status := domain.RosterStatusAvailable
		if stub.claimantLocked() != nil {
			status = domain.RosterStatusClaimed
		}
		state.Roster = []domain.PlayerRosterEntry{{ID: stub.characterID, Name: stub.character, Status: status}}
	}
	if session.characterID != "" {
		state.Character = &domain.PlayerCharacter{ID: stub.characterID, Name: stub.character}
		state.Role = domain.PlayerRoleObserver
		state.Phase = domain.PlayerPhaseWaiting
		if stub.controller == session.id {
			state.Role = domain.PlayerRoleActive
		}
	}
	return state
}

func (stub *us4CoordinatorStub) counters() (uint64, int) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	return stub.revision, stub.effectCount
}

func startUS4TestServer(t *testing.T, coordinator *us4CoordinatorStub) *Server {
	t.Helper()
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")}}
	server, err := NewServer(Config{Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: live.New(nil, nil), Coordinator: coordinator})
	if err != nil {
		t.Fatal(err)
	}
	coordinator.enqueue = server.PublishCoordinationEffect
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

// us1CoordinatorStub defines the transport seam expected by T022 without
// implementing it in the player server. It models only the already-frozen US1
// coordinator contract so these remain server integration tests rather than a
// second copy of internal/control.Service's concurrency suite.
type us1CoordinatorStub struct {
	mu sync.Mutex

	revision   uint64
	next       int
	broadcast  domain.BroadcastID
	roster     []domain.PlayerRosterEntry
	live       *domain.PublicLiveState
	sessions   map[domain.ConnectionID]*us1Session
	claimedBy  map[domain.CharacterID]domain.LogicalSessionID
	controller domain.LogicalSessionID
	enqueue    func(control.Effect)
}

type us1Session struct {
	id           domain.LogicalSessionID
	browserToken domain.BrowserToken
	fallbackName string
	characterID  domain.CharacterID
}

func newUS1CoordinatorStub(roster []domain.PlayerRosterEntry, liveState *domain.PublicLiveState) *us1CoordinatorStub {
	return &us1CoordinatorStub{
		broadcast: "broadcast-us1",
		roster:    append([]domain.PlayerRosterEntry(nil), roster...),
		live:      liveState,
		sessions:  make(map[domain.ConnectionID]*us1Session),
		claimedBy: make(map[domain.CharacterID]domain.LogicalSessionID),
	}
}

func (stub *us1CoordinatorStub) AttachConnection(connectionID domain.ConnectionID, _ domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	stub.next++
	stub.revision++
	session := &us1Session{
		id:           domain.LogicalSessionID(fmt.Sprintf("session-%d", stub.next)),
		browserToken: domain.BrowserToken(fmt.Sprintf("browser-token-%d", stub.next)),
		fallbackName: fmt.Sprintf("PLAYER %d", stub.next),
	}
	stub.sessions[connectionID] = session
	return session.browserToken, stub.playerState(session)
}

func (stub *us1CoordinatorStub) DetachConnection(connectionID domain.ConnectionID) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	delete(stub.sessions, connectionID)
}

func (stub *us1CoordinatorStub) SelectCharacter(connectionID domain.ConnectionID, requestID string, broadcastID domain.BroadcastID, characterID domain.CharacterID) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	session := stub.sessions[connectionID]
	if session == nil || broadcastID != stub.broadcast {
		stub.emit(control.Effect{ConnectionID: connectionID, Result: &domain.ActionResult{
			RequestID: requestID, Reason: domain.ActionReasonInvalidSession, Revision: stub.revision,
		}})
		return
	}
	if _, claimed := stub.claimedBy[characterID]; claimed || session.characterID != "" {
		state := stub.playerState(session)
		stub.emit(control.Effect{SessionID: session.id, Player: state})
		stub.emit(control.Effect{ConnectionID: connectionID, Result: &domain.ActionResult{
			RequestID: requestID, Reason: domain.ActionReasonConflict, Revision: stub.revision,
		}})
		return
	}

	stub.revision++
	session.characterID = characterID
	stub.claimedBy[characterID] = session.id
	if stub.controller == "" {
		stub.controller = session.id
	}
	for _, current := range stub.sessions {
		stub.emit(control.Effect{SessionID: current.id, Player: stub.playerState(current)})
	}
	if stub.live != nil {
		stub.emit(control.Effect{SessionID: session.id, Live: stub.live})
	}
	stub.emit(control.Effect{ConnectionID: connectionID, Result: &domain.ActionResult{
		RequestID: requestID, Accepted: true, Reason: domain.ActionReason("accepted"), Revision: stub.revision,
	}})
}

func (stub *us1CoordinatorStub) playerState(session *us1Session) *domain.PlayerState {
	state := &domain.PlayerState{
		Revision: stub.revision, SessionID: session.id, FallbackName: session.fallbackName,
		Role: domain.PlayerRoleUnassigned, Phase: domain.PlayerPhaseSelecting,
		BroadcastID: stub.broadcast, Roster: stub.playerRoster(),
	}
	if stub.live != nil {
		state.ActiveTerminalID = stub.live.TerminalID
	}
	if session.characterID == "" {
		return state
	}
	for _, character := range stub.roster {
		if character.ID == session.characterID {
			state.Character = &domain.PlayerCharacter{ID: character.ID, Name: character.Name}
			break
		}
	}
	if session.id == stub.controller {
		state.Role = domain.PlayerRoleActive
	} else {
		state.Role = domain.PlayerRoleObserver
	}
	if state.ActiveTerminalID == "" {
		state.Phase = domain.PlayerPhaseWaiting
	} else if state.Role == domain.PlayerRoleActive {
		state.Phase = domain.PlayerPhaseControlling
	} else {
		state.Phase = domain.PlayerPhaseObserving
	}
	return state
}

func (stub *us1CoordinatorStub) playerRoster() []domain.PlayerRosterEntry {
	roster := make([]domain.PlayerRosterEntry, len(stub.roster))
	copy(roster, stub.roster)
	for index := range roster {
		if _, claimed := stub.claimedBy[roster[index].ID]; claimed {
			roster[index].Status = domain.RosterStatusClaimed
		} else {
			roster[index].Status = domain.RosterStatusAvailable
		}
	}
	return roster
}

func (stub *us1CoordinatorStub) emit(effect control.Effect) {
	effect.Revision = stub.revision
	if stub.enqueue != nil {
		stub.enqueue(effect)
	}
}

func startUS1TestServer(t *testing.T, coordinator *us1CoordinatorStub, liveService *live.Service) *Server {
	t.Helper()
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")},
	}
	server, err := NewServer(Config{
		Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: liveService,
		Coordinator: coordinator,
	})
	if err != nil {
		t.Fatal(err)
	}
	coordinator.enqueue = server.PublishCoordinationEffect
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

func assertUS1Welcome(t *testing.T, message serverMessage, broadcastID domain.BroadcastID, terminalID string) {
	t.Helper()
	if message.Type != MessageSessionWelcome || message.BrowserToken == "" || message.State == nil {
		t.Fatalf("handshake response = %#v, want SESSION_WELCOME with token and state", message)
	}
	if message.State.SessionID == "" || message.State.FallbackName == "" {
		t.Fatalf("welcome identity = %#v, want nonblank session and fallback name", message.State)
	}
	if message.State.BroadcastID != broadcastID || message.State.ActiveTerminalID != terminalID {
		t.Fatalf("welcome broadcast state = %#v, want broadcast=%q terminal=%q", message.State, broadcastID, terminalID)
	}
	if message.State.Role != domain.PlayerRoleUnassigned || message.State.Phase != domain.PlayerPhaseSelecting || message.State.Character != nil {
		t.Fatalf("welcome player state = %#v, want unassigned selection", message.State)
	}
}

func assertUS1AssignedState(t *testing.T, message serverMessage, messageType string, characterID domain.CharacterID, role domain.PlayerRole, phase domain.PlayerPhase, terminalID string) {
	t.Helper()
	if message.Type != messageType || message.State == nil || message.State.Character == nil {
		t.Fatalf("assigned state envelope = %#v", message)
	}
	if message.State.Character.ID != characterID || message.State.Role != role || message.State.Phase != phase || message.State.ActiveTerminalID != terminalID {
		t.Fatalf("assigned player state = %#v, want character=%q role=%q phase=%q terminal=%q", message.State, characterID, role, phase, terminalID)
	}
	assertUS1RosterClaimed(t, message, characterID)
}

func assertUS1RosterClaimed(t *testing.T, message serverMessage, characterID domain.CharacterID) {
	t.Helper()
	if message.Type != MessagePlayerState || message.State == nil {
		t.Fatalf("roster refresh = %#v, want PLAYER_STATE", message)
	}
	for _, character := range message.State.Roster {
		if character.ID == characterID {
			if character.Status != domain.RosterStatusClaimed {
				t.Fatalf("roster character %q status = %q, want claimed", characterID, character.Status)
			}
			return
		}
	}
	t.Fatalf("roster refresh omitted character %q: %#v", characterID, message.State.Roster)
}

func assertUS1ActionResult(t *testing.T, message serverMessage, requestID string, accepted bool, reason domain.ActionReason) {
	t.Helper()
	if message.Type != MessageActionResult || message.RequestID != requestID || message.Accepted != accepted || message.Reason != reason {
		t.Fatalf("action result = %#v, want request=%q accepted=%t reason=%q", message, requestID, accepted, reason)
	}
}

func TestCoordinatedServerConvergesFourThroughSevenAssignedClientsInRevisionOrder(t *testing.T) {
	for clientTotal := 4; clientTotal <= 7; clientTotal++ {
		t.Run(fmt.Sprintf("%d_clients", clientTotal), func(t *testing.T) {
			liveService := live.New(nil, nil)
			liveService.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
			coordinator := newUS2CoordinatorStub(liveService)
			server := startUS2TestServer(t, coordinator, 16)

			clients := make([]*websocket.Conn, 0, clientTotal)
			for index := 0; index < clientTotal; index++ {
				connection, welcome := connectUS2Player(t, server.Info().URL, "")
				clients = append(clients, connection)
				wantRole := domain.PlayerRoleObserver
				if index == 0 {
					wantRole = domain.PlayerRoleActive
				}
				if welcome.State == nil || welcome.State.Role != wantRole {
					t.Fatalf("client %d role = %#v, want %q", index, welcome.State, wantRole)
				}
			}

			previousRevision := uint64(0)
			requests := []map[string]any{
				{
					"type": MessageNavAction, "requestId": "nav-enter", "broadcastId": us2BroadcastID,
					"terminalId": "terminal-1", "action": "enter", "nodeId": "docs",
				},
				{
					"type": MessageNavAction, "requestId": "nav-back", "broadcastId": us2BroadcastID,
					"terminalId": "terminal-1", "action": "back",
				},
			}
			for requestIndex, request := range requests {
				writeJSON(t, clients[0], request)
				canonical := readUS2Message(t, clients[0])
				if canonical.Type != MessageTerminalLive || canonical.Nav == nil {
					t.Fatalf("controller canonical message %d = %#v, want revisioned terminal state", requestIndex, canonical)
				}
				for clientIndex := 1; clientIndex < len(clients); clientIndex++ {
					message := readUS2Message(t, clients[clientIndex])
					if !reflect.DeepEqual(message, canonical) {
						t.Fatalf("request %d client %d diverged: got %#v, canonical %#v", requestIndex, clientIndex, message, canonical)
					}
				}
				result := readUS2Message(t, clients[0])
				assertUS1ActionResult(t, result, request["requestId"].(string), true, domain.ActionReasonAccepted)
				if result.Revision != canonical.Revision || canonical.Revision <= previousRevision {
					t.Fatalf("request %d revisions: state=%d result=%d previous=%d", requestIndex, canonical.Revision, result.Revision, previousRevision)
				}
				previousRevision = canonical.Revision
			}
		})
	}
}

func TestCoordinatedServerTargetsResultToInitiatingTabAndRejectsCraftedObserverAction(t *testing.T) {
	liveService := live.New(nil, nil)
	liveService.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	coordinator := newUS2CoordinatorStub(liveService)
	server := startUS2TestServer(t, coordinator, 16)

	controller, controllerWelcome := connectUS2Player(t, server.Info().URL, "")
	controllerSibling, siblingWelcome := connectUS2Player(t, server.Info().URL, domain.BrowserToken(controllerWelcome.BrowserToken))
	observer, observerWelcome := connectUS2Player(t, server.Info().URL, "")
	if controllerWelcome.State == nil || siblingWelcome.State == nil || controllerWelcome.State.SessionID != siblingWelcome.State.SessionID {
		t.Fatalf("same-token tabs did not share a logical session: first=%#v sibling=%#v", controllerWelcome.State, siblingWelcome.State)
	}
	if observerWelcome.State == nil || observerWelcome.State.Role != domain.PlayerRoleObserver {
		t.Fatalf("observer welcome = %#v, want observer", observerWelcome.State)
	}

	writeJSON(t, controller, map[string]any{
		"type": MessageNavAction, "requestId": "controller-enter", "broadcastId": us2BroadcastID,
		"terminalId": "terminal-1", "action": "enter", "nodeId": "docs",
	})
	controllerState := readUS2Message(t, controller)
	siblingState := readUS2Message(t, controllerSibling)
	observerState := readUS2Message(t, observer)
	if !reflect.DeepEqual(siblingState, controllerState) || !reflect.DeepEqual(observerState, controllerState) {
		t.Fatalf("accepted action did not converge: controller=%#v sibling=%#v observer=%#v", controllerState, siblingState, observerState)
	}
	controllerResult := readUS2Message(t, controller)
	assertUS1ActionResult(t, controllerResult, "controller-enter", true, domain.ActionReasonAccepted)
	if controllerResult.Revision != controllerState.Revision {
		t.Fatalf("controller result revision = %d, want state revision %d", controllerResult.Revision, controllerState.Revision)
	}

	writeJSON(t, observer, map[string]any{
		"type": MessageNavAction, "requestId": "crafted-observer-back", "broadcastId": us2BroadcastID,
		"terminalId": "terminal-1", "action": "back",
	})
	observerResult := readUS2Message(t, observer)
	assertUS1ActionResult(t, observerResult, "crafted-observer-back", false, domain.ActionReasonNotController)
	if observerResult.Revision != controllerState.Revision {
		t.Fatalf("observer rejection revision = %d, want unchanged %d", observerResult.Revision, controllerState.Revision)
	}
	assertNoPlayerMessage(t, controller)
	assertNoPlayerMessage(t, controllerSibling)
	if got := coordinator.acceptedMutations(); got != 1 {
		t.Fatalf("accepted canonical mutations = %d, want 1", got)
	}
}

func TestCoordinatedServerDeduplicatesOneUsePatternRequests(t *testing.T) {
	liveService := live.New(&serverRandom{values: []int{1, 0}}, serverWords{})
	liveState := liveService.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	coordinator := newUS2CoordinatorStub(liveService)
	server := startUS2TestServer(t, coordinator, 16)
	controller, _ := connectUS2Player(t, server.Info().URL, "")
	observer, _ := connectUS2Player(t, server.Info().URL, "")
	if liveState.Hack == nil || len(liveState.Hack.Patterns) == 0 {
		t.Fatal("fixture has no one-use hacking pattern")
	}
	patternID := liveState.Hack.Patterns[0].ID
	request := map[string]any{
		"type": MessageHackPattern, "requestId": "one-use-pattern", "broadcastId": us2BroadcastID,
		"terminalId": "terminal-1", "patternId": patternID,
	}

	writeJSON(t, controller, request)
	writeJSON(t, controller, request)
	acceptedState := readUS2Message(t, controller)
	observerState := readUS2Message(t, observer)
	if !reflect.DeepEqual(observerState, acceptedState) || !publicPatternUsed(acceptedState.Hack, patternID) {
		t.Fatalf("accepted one-use state did not converge: controller=%#v observer=%#v", acceptedState, observerState)
	}
	firstResult := readUS2Message(t, controller)
	replayedResult := readUS2Message(t, controller)
	assertUS1ActionResult(t, firstResult, "one-use-pattern", true, domain.ActionReasonAccepted)
	if !reflect.DeepEqual(replayedResult, firstResult) {
		t.Fatalf("exact replay result = %#v, want cached %#v", replayedResult, firstResult)
	}
	if firstResult.Revision != acceptedState.Revision {
		t.Fatalf("one-use result revision = %d, want state revision %d", firstResult.Revision, acceptedState.Revision)
	}
	if got := coordinator.acceptedMutations(); got != 1 {
		t.Fatalf("one-use request mutated canonical state %d times, want 1", got)
	}
	assertNoPlayerMessage(t, observer)
}

func TestCoordinatedServerSlowObserverDoesNotBlockControllerOrFastObserver(t *testing.T) {
	liveService := live.New(nil, nil)
	// Queue pressure, not payload serialization cost, is the subject here. A
	// compact terminal keeps the assertion meaningful under the race detector.
	liveService.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	coordinator := newUS2CoordinatorStub(liveService)
	// Keep the controller's own two-message-per-action burst below its queue
	// capacity under the race detector; this test isolates a non-reading
	// observer, while the dedicated bounded-queue test covers overflow closure.
	server := startUS2TestServer(t, coordinator, 64)
	controller, _ := connectUS2Player(t, server.Info().URL, "")
	slowObserver, _ := connectUS2Player(t, server.Info().URL, "")
	defer slowObserver.CloseNow()
	fastObserver, _ := connectUS2Player(t, server.Info().URL, "")

	controllerMessages := pumpUS2Messages(controller)
	fastMessages := pumpUS2Messages(fastObserver)
	const actionTotal = 20
	started := time.Now()
	for index := 0; index < actionTotal; index++ {
		action := "enter"
		request := map[string]any{
			"type": MessageNavAction, "requestId": fmt.Sprintf("slow-isolation-%d", index), "broadcastId": us2BroadcastID,
			"terminalId": "terminal-1", "action": action, "nodeId": "docs",
		}
		if index%2 == 1 {
			request["action"] = "back"
			delete(request, "nodeId")
		}
		writeJSON(t, controller, request)
	}
	coordinator.waitForDispatches(t, actionTotal)
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("coordinated actions waited %s for a slow observer", elapsed)
	}

	wantRevision := coordinator.currentRevision()
	waitForUS2Revision(t, controllerMessages, MessageActionResult, wantRevision)
	waitForUS2Revision(t, fastMessages, MessageTerminalLive, wantRevision)
	if got := coordinator.acceptedMutations(); got != actionTotal {
		t.Fatalf("accepted canonical mutations = %d, want %d", got, actionTotal)
	}
}

func TestCoordinatedServerDispatchesActiveControllerHackingThroughControlService(t *testing.T) {
	liveService := live.New(&serverRandom{values: []int{1, 0}}, serverWords{})
	coordinator := newUS9CoordinatorStub(liveService)
	server := startUS9TestServer(t, coordinator, liveService)

	_, err := coordinator.service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	characterID := coordinator.characterID(t)
	broadcast, err := coordinator.service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.service.RequestTerminalActivation(domain.TerminalTarget{
		TerminalID: "terminal-1", TerminalName: "Overseer", Tree: serverTree(), HackLevel: 1, IntroText: "WELCOME",
	}); err != nil {
		t.Fatal(err)
	}

	controller, _ := connectUS9Player(t, server.Info().URL, "")
	defer controller.CloseNow()
	writeJSON(t, controller, map[string]any{
		"type": MessageCharacterSelect, "requestId": "select-controller",
		"broadcastId": broadcast.Broadcast.ID, "characterId": characterID,
	})
	assigned := readUS2Message(t, controller)
	liveMessage := readUS2Message(t, controller)
	selectionResult := readUS2Message(t, controller)
	assertUS1AssignedState(t, assigned, MessagePlayerState, characterID, domain.PlayerRoleActive, domain.PlayerPhaseControlling, "terminal-1")
	assertUS1ActionResult(t, selectionResult, "select-controller", true, domain.ActionReasonAccepted)
	if liveMessage.Type != MessageTerminalLive || liveMessage.Hack == nil {
		t.Fatalf("assigned active terminal = %#v, want unfinished hacking state", liveMessage)
	}
	if liveMessage.Hack.Log == nil {
		t.Fatal("production-shaped puzzle encoded an empty hacking log as null")
	}
	targetID := firstCandidateID(liveMessage.Hack)
	if targetID == "" {
		t.Fatal("production-shaped puzzle has no password candidate")
	}

	writeJSON(t, controller, map[string]any{
		"type": MessageHackGuess, "requestId": "active-password",
		"broadcastId": broadcast.Broadcast.ID, "terminalId": "terminal-1", "targetId": targetID,
	})
	hackState := readUS2Message(t, controller)
	result := readUS2Message(t, controller)
	if hackState.Type != MessageTerminalLive || hackState.Hack == nil {
		t.Fatalf("active password projection = %#v, want revisioned terminal state", hackState)
	}
	assertUS1ActionResult(t, result, "active-password", true, domain.ActionReasonAccepted)
	if result.Revision != hackState.Revision || result.Revision <= selectionResult.Revision {
		t.Fatalf("active password revisions: state=%d result=%d selection=%d", hackState.Revision, result.Revision, selectionResult.Revision)
	}
}

func TestCoordinatedServerPublishesFreshFailedHackResetToActiveAndObserver(t *testing.T) {
	liveService := live.New(&serverRandom{values: []int{1, 0}}, serverWords{})
	coordinator := newUS9CoordinatorStub(liveService)
	server := startUS9TestServer(t, coordinator, liveService)

	first, err := coordinator.service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	second, err := coordinator.service.AddCharacter("Boone")
	if err != nil {
		t.Fatal(err)
	}
	broadcast, err := coordinator.service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	active, activeWelcome := connectUS9Player(t, server.Info().URL, "")
	observer, observerWelcome := connectUS9Player(t, server.Info().URL, "")
	defer active.CloseNow()
	defer observer.CloseNow()
	activeMessages := pumpUS2Messages(active)
	observerMessages := pumpUS2Messages(observer)
	if _, err := coordinator.service.AssignCharacter(activeWelcome.State.SessionID, first.Roster[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.service.AssignCharacter(observerWelcome.State.SessionID, second.Roster[1].ID); err != nil {
		t.Fatal(err)
	}
	target := domain.TerminalTarget{TerminalID: "terminal-reset", TerminalName: "Reset", Tree: serverTree(), HackLevel: 1, IntroText: "OLD"}
	activated, err := coordinator.service.RequestTerminalActivation(target)
	if err != nil {
		t.Fatal(err)
	}
	oldActive := waitForUS9TerminalRevision(t, activeMessages, activated.Revision)
	oldObserver := waitForUS9TerminalRevision(t, observerMessages, activated.Revision)
	if oldActive.Hack == nil || !reflect.DeepEqual(oldActive.Hack, oldObserver.Hack) {
		t.Fatalf("initial assigned puzzles diverged: active=%#v observer=%#v", oldActive.Hack, oldObserver.Hack)
	}

	for index, filler := range safeFillerTargets(oldActive.Hack, oldActive.Hack.AttemptsMax) {
		requestID := fmt.Sprintf("fail-%d", index)
		writeJSON(t, active, map[string]any{
			"type": MessageHackGuess, "requestId": requestID, "broadcastId": broadcast.Broadcast.ID,
			"terminalId": target.TerminalID, "targetId": filler,
		})
		failedActive := waitForUS9TerminalAfter(t, activeMessages, oldActive.Revision)
		failedObserver := waitForUS9TerminalAfter(t, observerMessages, oldObserver.Revision)
		oldActive, oldObserver = failedActive, failedObserver
		waitForUS9ActionResult(t, activeMessages, requestID, failedActive.Revision)
	}
	if oldActive.Hack == nil || !oldActive.Hack.Failed || oldActive.Hack.AttemptsLeft != 0 || !reflect.DeepEqual(oldActive.Hack, oldObserver.Hack) {
		t.Fatalf("failed projections diverged: active=%#v observer=%#v", oldActive.Hack, oldObserver.Hack)
	}

	latest := target
	latest.TerminalName = "Reset Latest"
	latest.HackLevel = 2
	latest.IntroText = "LATEST"
	reset, err := coordinator.service.ResetFailedHack(latest)
	if err != nil {
		t.Fatal(err)
	}
	freshActive := waitForUS9TerminalRevision(t, activeMessages, reset.Revision)
	freshObserver := waitForUS9TerminalRevision(t, observerMessages, reset.Revision)
	if freshActive.Hack == nil || freshActive.Hack.Failed || freshActive.Hack.Solved || freshActive.Hack.Level != 2 || freshActive.Hack.AttemptsLeft != freshActive.Hack.AttemptsMax || len(freshActive.Hack.Log) != 0 {
		t.Fatalf("fresh active projection = %#v", freshActive)
	}
	if !reflect.DeepEqual(freshActive, freshObserver) || freshActive.TerminalID != target.TerminalID || freshActive.TerminalName != latest.TerminalName || freshActive.IntroText != latest.IntroText {
		t.Fatalf("reset projections diverged: active=%#v observer=%#v", freshActive, freshObserver)
	}
	if reset.Broadcast == nil || reset.Broadcast.ID != broadcast.Broadcast.ID || reset.Broadcast.ActiveTerminalID == nil || *reset.Broadcast.ActiveTerminalID != target.TerminalID {
		t.Fatalf("reset changed broadcast or terminal identity: %#v", reset)
	}
}

func waitForUS9TerminalRevision(t *testing.T, messages <-chan serverMessage, revision uint64) serverMessage {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case message := <-messages:
			if message.Type == MessageTerminalLive && message.Revision == revision {
				return message
			}
		case <-deadline:
			t.Fatalf("TERMINAL_LIVE revision %d timed out", revision)
		}
	}
}

func waitForUS9TerminalAfter(t *testing.T, messages <-chan serverMessage, revision uint64) serverMessage {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case message := <-messages:
			if message.Type == MessageTerminalLive && message.Revision > revision {
				return message
			}
		case <-deadline:
			t.Fatalf("TERMINAL_LIVE after revision %d timed out", revision)
		}
	}
}

func waitForUS9ActionResult(t *testing.T, messages <-chan serverMessage, requestID string, revision uint64) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case message := <-messages:
			if message.Type == MessageActionResult && message.RequestID == requestID {
				if !message.Accepted || message.Revision != revision {
					t.Fatalf("action result = %#v", message)
				}
				return
			}
		case <-deadline:
			t.Fatalf("ACTION_RESULT %q timed out", requestID)
		}
	}
}

func safeFillerTargets(state *domain.PublicHackState, count int) []string {
	if state == nil || len(state.Columns) == 0 {
		return nil
	}
	blocked := make(map[int]bool)
	for _, word := range state.Columns[0].Words {
		for index := word.Start; index < word.Start+word.Length; index++ {
			blocked[index] = true
		}
	}
	for _, pattern := range state.Patterns {
		if pattern.Row < 16 {
			blocked[pattern.Row*12+pattern.Start] = true
		}
	}
	result := make([]string, 0, count)
	for index := 0; index < len(state.Columns[0].Text) && len(result) < count; index++ {
		if !blocked[index] {
			result = append(result, fmt.Sprintf("0:%d", index))
		}
	}
	return result
}

func TestCoordinatedServerReassignmentFansOneRoleRevisionToEverySessionTab(t *testing.T) {
	liveService := live.New(&serverRandom{values: []int{1, 0}}, serverWords{})
	liveService.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	coordinator := newUS2CoordinatorStub(liveService)
	server := startUS2TestServer(t, coordinator, 16)

	former, formerWelcome := connectUS2Player(t, server.Info().URL, "")
	formerSibling, formerSiblingWelcome := connectUS2Player(t, server.Info().URL, domain.BrowserToken(formerWelcome.BrowserToken))
	next, nextWelcome := connectUS2Player(t, server.Info().URL, "")
	nextSibling, nextSiblingWelcome := connectUS2Player(t, server.Info().URL, domain.BrowserToken(nextWelcome.BrowserToken))
	if formerSiblingWelcome.State.SessionID != formerWelcome.State.SessionID || nextSiblingWelcome.State.SessionID != nextWelcome.State.SessionID {
		t.Fatalf("same-token tabs did not share sessions: former=%#v/%#v next=%#v/%#v", formerWelcome.State, formerSiblingWelcome.State, nextWelcome.State, nextSiblingWelcome.State)
	}
	before := liveService.Snapshot()

	revision, ok := coordinator.SetActiveController(nextWelcome.State.SessionID)
	if !ok {
		t.Fatal("test coordinator rejected connected assigned observer")
	}
	for index, connection := range []*websocket.Conn{former, formerSibling} {
		assertUS5RoleChange(t, connection, revision, domain.PlayerRoleObserver, formerWelcome.State.Character.ID)
		_ = index
	}
	for _, connection := range []*websocket.Conn{next, nextSibling} {
		assertUS5RoleChange(t, connection, revision, domain.PlayerRoleActive, nextWelcome.State.Character.ID)
	}
	if after := liveService.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("controller reassignment changed canonical terminal\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func TestCoordinatedServerOrdersActionsBeforeAndAfterConcurrentReassignment(t *testing.T) {
	liveService := live.New(nil, nil)
	liveService.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	coordinator := newUS2CoordinatorStub(liveService)
	server := startUS2TestServer(t, coordinator, 16)
	former, formerWelcome := connectUS2Player(t, server.Info().URL, "")
	next, nextWelcome := connectUS2Player(t, server.Info().URL, "")

	actionStarted := make(chan struct{})
	actionRelease := make(chan struct{})
	coordinator.blockNextAction(actionStarted, actionRelease)
	writeJSON(t, former, map[string]any{
		"type": MessageNavAction, "requestId": "before-reassignment", "broadcastId": us2BroadcastID,
		"terminalId": "terminal-1", "action": "enter", "nodeId": "docs",
	})
	select {
	case <-actionStarted:
	case <-time.After(time.Second):
		t.Fatal("former-controller action did not enter coordinator order")
	}

	reassigned := make(chan uint64, 1)
	go func() {
		revision, _ := coordinator.SetActiveController(nextWelcome.State.SessionID)
		reassigned <- revision
	}()
	select {
	case revision := <-reassigned:
		t.Fatalf("reassignment revision %d overtook action already inside coordinator", revision)
	case <-time.After(50 * time.Millisecond):
	}
	close(actionRelease)

	actionLive := readUS2Message(t, former)
	actionResult := readUS2Message(t, former)
	assertUS1ActionResult(t, actionResult, "before-reassignment", true, domain.ActionReasonAccepted)
	if actionLive.Type != MessageTerminalLive || actionLive.Revision != actionResult.Revision || !reflect.DeepEqual(actionLive.Nav.Path, []string{"root", "docs"}) {
		t.Fatalf("action ordered before reassignment = live %#v result %#v", actionLive, actionResult)
	}
	nextActionLive := readUS2Message(t, next)
	if !reflect.DeepEqual(nextActionLive, actionLive) {
		t.Fatalf("observer did not receive pre-reassignment canonical action: got %#v want %#v", nextActionLive, actionLive)
	}

	var reassignmentRevision uint64
	select {
	case reassignmentRevision = <-reassigned:
	case <-time.After(time.Second):
		t.Fatal("reassignment did not follow released action")
	}
	if reassignmentRevision <= actionResult.Revision {
		t.Fatalf("reassignment revision = %d, want after action revision %d", reassignmentRevision, actionResult.Revision)
	}
	assertUS5RoleChange(t, former, reassignmentRevision, domain.PlayerRoleObserver, formerWelcome.State.Character.ID)
	assertUS5RoleChange(t, next, reassignmentRevision, domain.PlayerRoleActive, nextWelcome.State.Character.ID)

	writeJSON(t, former, map[string]any{
		"type": MessageNavAction, "requestId": "after-reassignment", "broadcastId": us2BroadcastID,
		"terminalId": "terminal-1", "action": "back",
	})
	rejected := readUS2Message(t, former)
	assertUS1ActionResult(t, rejected, "after-reassignment", false, domain.ActionReasonNotController)
	if rejected.Revision != reassignmentRevision {
		t.Fatalf("former-controller rejection revision = %d, want %d", rejected.Revision, reassignmentRevision)
	}
	assertNoPlayerMessage(t, next)
	if got := coordinator.acceptedMutations(); got != 1 {
		t.Fatalf("accepted terminal mutations = %d, want only the pre-reassignment action", got)
	}
}

func TestCoordinatedServerControllerMultiTabCloseRetainsControlUntilReconnect(t *testing.T) {
	liveService := live.New(&serverRandom{values: []int{1, 0}}, serverWords{})
	canonical := liveService.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	coordinator := newUS2CoordinatorStub(liveService)
	server := startUS2TestServer(t, coordinator, 16)

	first, firstWelcome := connectUS2Player(t, server.Info().URL, "")
	second, secondWelcome := connectUS2Player(t, server.Info().URL, domain.BrowserToken(firstWelcome.BrowserToken))
	third, thirdWelcome := connectUS2Player(t, server.Info().URL, domain.BrowserToken(firstWelcome.BrowserToken))
	observer, observerWelcome := connectUS2Player(t, server.Info().URL, "")
	if secondWelcome.State.SessionID != firstWelcome.State.SessionID || thirdWelcome.State.SessionID != firstWelcome.State.SessionID {
		t.Fatalf("controller tabs did not share one session: %#v %#v %#v", firstWelcome.State, secondWelcome.State, thirdWelcome.State)
	}
	controllerID := firstWelcome.State.SessionID
	observerID := observerWelcome.State.SessionID

	_ = second.Close(websocket.StatusNormalClosure, "close controller tab two")
	coordinator.waitForConnections(t, controllerID, 2)
	if role := coordinator.roleForSession(observerID); role != domain.PlayerRoleObserver {
		t.Fatalf("observer role after non-final close = %q, want observer", role)
	}
	_ = first.Close(websocket.StatusNormalClosure, "close controller tab one")
	coordinator.waitForConnections(t, controllerID, 1)
	_ = third.Close(websocket.StatusNormalClosure, "close final controller tab")
	coordinator.waitForConnections(t, controllerID, 0)
	if got := coordinator.controllerSession(); got != controllerID {
		t.Fatalf("final close changed controller = %q, want retained %q", got, controllerID)
	}
	if role := coordinator.roleForSession(observerID); role != domain.PlayerRoleObserver {
		t.Fatalf("observer was promoted after controller disconnect: role=%q", role)
	}
	if after := liveService.Snapshot(); !reflect.DeepEqual(after, canonical) {
		t.Fatalf("controller disconnect changed terminal/puzzle\nbefore: %#v\nafter:  %#v", canonical, after)
	}

	reconnected, welcome := connectUS2Player(t, server.Info().URL, domain.BrowserToken(firstWelcome.BrowserToken))
	defer reconnected.CloseNow()
	if welcome.State.SessionID != controllerID || welcome.State.Role != domain.PlayerRoleActive || welcome.State.Character == nil {
		t.Fatalf("unchanged controller reconnect = %#v, want retained active assignment", welcome.State)
	}
	if after := liveService.Snapshot(); !reflect.DeepEqual(after, canonical) {
		t.Fatalf("controller reconnect regenerated terminal/puzzle\nbefore: %#v\nafter:  %#v", canonical, after)
	}
	defer observer.CloseNow()
}

func TestCoordinatedServerDisconnectedFormerControllerReconnectsAsObserverAfterReassignment(t *testing.T) {
	liveService := live.New(&serverRandom{values: []int{1, 0}}, serverWords{})
	canonical := liveService.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	coordinator := newUS2CoordinatorStub(liveService)
	server := startUS2TestServer(t, coordinator, 16)

	former, formerWelcome := connectUS2Player(t, server.Info().URL, "")
	next, nextWelcome := connectUS2Player(t, server.Info().URL, "")
	formerID := formerWelcome.State.SessionID
	_ = former.Close(websocket.StatusNormalClosure, "disconnect former controller")
	coordinator.waitForConnections(t, formerID, 0)

	revision, ok := coordinator.SetActiveController(nextWelcome.State.SessionID)
	if !ok {
		t.Fatal("reassignment to connected assigned observer failed")
	}
	assertUS5RoleChange(t, next, revision, domain.PlayerRoleActive, nextWelcome.State.Character.ID)
	if got := coordinator.controllerSession(); got != nextWelcome.State.SessionID {
		t.Fatalf("controller after reassignment = %q, want %q", got, nextWelcome.State.SessionID)
	}
	if after := liveService.Snapshot(); !reflect.DeepEqual(after, canonical) {
		t.Fatalf("reassignment after disconnect changed terminal/puzzle\nbefore: %#v\nafter:  %#v", canonical, after)
	}

	reconnected, welcome := connectUS2Player(t, server.Info().URL, domain.BrowserToken(formerWelcome.BrowserToken))
	defer reconnected.CloseNow()
	defer next.CloseNow()
	if welcome.State.SessionID != formerID || welcome.State.Role != domain.PlayerRoleObserver || welcome.State.Phase != domain.PlayerPhaseObserving || welcome.State.Character == nil {
		t.Fatalf("reassigned former-controller reconnect = %#v, want retained observer assignment", welcome.State)
	}
	if after := liveService.Snapshot(); !reflect.DeepEqual(after, canonical) {
		t.Fatalf("former-controller reconnect cleared/regenerated terminal\nbefore: %#v\nafter:  %#v", canonical, after)
	}
}

func TestCoordinatedServerFollowsTenTerminalSwitchesLateAssignmentReconnectAndStaleAction(t *testing.T) {
	liveService := live.New(nil, nil)
	coordinator := newUS7CoordinatorStub(liveService)
	server := startUS7TestServer(t, coordinator)

	controller, controllerWelcome := connectUS7Player(t, server.Info().URL, "")
	observer, observerWelcome := connectUS7Player(t, server.Info().URL, "")
	late, lateWelcome := connectUS7Player(t, server.Info().URL, "")
	coordinator.Assign(controllerWelcome.State.SessionID)
	assertUS7PlayerState(t, controller, domain.PlayerRoleActive, domain.PlayerPhaseWaiting, "", coordinator.currentRevision())
	coordinator.Assign(observerWelcome.State.SessionID)
	assertUS7PlayerState(t, observer, domain.PlayerRoleObserver, domain.PlayerPhaseWaiting, "", coordinator.currentRevision())

	targets := []string{"terminal-a", "terminal-b", "", "terminal-a", "terminal-b", "terminal-a", "", "terminal-b", "terminal-a", "terminal-b"}
	previousRevision := coordinator.currentRevision()
	previousTerminal := ""
	for index, target := range targets {
		revision := coordinator.Switch(target)
		if revision <= previousRevision {
			t.Fatalf("switch %d revision = %d, want greater than %d", index+1, revision, previousRevision)
		}
		for clientIndex, connection := range []*websocket.Conn{controller, observer} {
			role := domain.PlayerRoleObserver
			if clientIndex == 0 {
				role = domain.PlayerRoleActive
			}
			phase := domain.PlayerPhaseWaiting
			if target != "" && role == domain.PlayerRoleActive {
				phase = domain.PlayerPhaseControlling
			} else if target != "" {
				phase = domain.PlayerPhaseObserving
			}
			assertUS7PlayerState(t, connection, role, phase, target, revision)
			terminal := readUS2Message(t, connection)
			if target == "" {
				if terminal.Type != MessageTerminalClear || terminal.Revision != revision {
					t.Fatalf("switch %d client %d clear = %#v, want revision %d", index+1, clientIndex, terminal, revision)
				}
			} else if terminal.Type != MessageTerminalLive || terminal.Revision != revision || terminal.TerminalID != target {
				t.Fatalf("switch %d client %d live = %#v, want %q revision %d", index+1, clientIndex, terminal, target, revision)
			}
		}
		previousRevision = revision
		previousTerminal = target
	}
	if previousTerminal == "" {
		t.Fatal("fixture must finish on an active terminal")
	}

	coordinator.Assign(lateWelcome.State.SessionID)
	lateRevision := coordinator.currentRevision()
	assertUS7PlayerState(t, late, domain.PlayerRoleObserver, domain.PlayerPhaseObserving, previousTerminal, lateRevision)
	lateLive := readUS2Message(t, late)
	if lateLive.Type != MessageTerminalLive || lateLive.TerminalID != previousTerminal || lateLive.Revision != lateRevision {
		t.Fatalf("late assignment terminal = %#v, want %q revision %d", lateLive, previousTerminal, lateRevision)
	}

	_ = late.Close(websocket.StatusNormalClosure, "reconnect late assignment")
	coordinator.waitForConnections(t, lateWelcome.State.SessionID, 0)
	reconnected, reconnectWelcome := connectUS7Player(t, server.Info().URL, domain.BrowserToken(lateWelcome.BrowserToken))
	defer reconnected.CloseNow()
	if reconnectWelcome.State.SessionID != lateWelcome.State.SessionID || reconnectWelcome.State.Role != domain.PlayerRoleObserver || reconnectWelcome.State.ActiveTerminalID != previousTerminal {
		t.Fatalf("late-assignee reconnect welcome = %#v", reconnectWelcome.State)
	}
	reconnectLive := readUS2Message(t, reconnected)
	if reconnectLive.Type != MessageTerminalLive || reconnectLive.TerminalID != previousTerminal || reconnectLive.Revision != reconnectWelcome.State.Revision {
		t.Fatalf("late-assignee reconnect terminal = %#v", reconnectLive)
	}

	staleTerminal := "terminal-a"
	if staleTerminal == previousTerminal {
		staleTerminal = "terminal-b"
	}
	writeJSON(t, controller, map[string]any{
		"type": MessageNavAction, "requestId": "stale-after-switch", "broadcastId": us7BroadcastID,
		"terminalId": staleTerminal, "action": "back",
	})
	result := readUS2Message(t, controller)
	assertUS1ActionResult(t, result, "stale-after-switch", false, domain.ActionReasonStaleTerminal)
	if result.Revision != lateRevision {
		t.Fatalf("stale-terminal result revision = %d, want current %d", result.Revision, lateRevision)
	}
	defer controller.CloseNow()
	defer observer.CloseNow()
}

func TestCoordinatedServerUnfinishedPuzzlePreserveDiscardCancelFanout(t *testing.T) {
	liveService := live.New(&serverRandom{values: []int{1, 0, 2, 3}}, serverWords{})
	source := liveService.Set("terminal-a", "Terminal A", serverTree(), 1, "SOURCE")
	if source == nil || source.Hack == nil {
		t.Fatal("unfinished source fixture has no public puzzle")
	}
	coordinator := newUS8CoordinatorStub(liveService, source)
	server := startUS8TestServer(t, coordinator)
	controller, _ := connectUS8Player(t, server.Info().URL, "")
	observer, _ := connectUS8Player(t, server.Info().URL, "")
	controllerMessages := pumpUS2Messages(controller)
	observerMessages := pumpUS2Messages(observer)

	coordinator.RequestSwitch("")
	assertNoUS8Message(t, controllerMessages, "pending clear reached controller")
	assertNoUS8Message(t, observerMessages, "pending clear reached observer")
	coordinator.Resolve(domain.TerminalSwitchCancel)
	assertNoUS8Message(t, controllerMessages, "cancel changed controller terminal")
	assertNoUS8Message(t, observerMessages, "cancel changed observer terminal")
	if current := coordinator.currentLive(); !sameUS8JSON(current, source) {
		t.Fatalf("cancel changed source puzzle\nsource:  %#v\ncurrent: %#v", source, current)
	}

	coordinator.RequestSwitch("terminal-b")
	assertNoUS8Message(t, controllerMessages, "pending preserve switched controller early")
	assertNoUS8Message(t, observerMessages, "pending preserve switched observer early")
	preserveRevision := coordinator.Resolve(domain.TerminalSwitchPreserve)
	readUS8Transition(t, controllerMessages, "terminal-b", preserveRevision, domain.PlayerRoleActive)
	readUS8Transition(t, observerMessages, "terminal-b", preserveRevision, domain.PlayerRoleObserver)

	restoreRevision := coordinator.ActivateSource()
	restoredController := readUS8Transition(t, controllerMessages, "terminal-a", restoreRevision, domain.PlayerRoleActive)
	restoredObserver := readUS8Transition(t, observerMessages, "terminal-a", restoreRevision, domain.PlayerRoleObserver)
	if !sameUS8JSON(restoredController.Hack, source.Hack) || !sameUS8JSON(restoredObserver.Hack, source.Hack) {
		t.Fatalf("preserved public puzzle was not restored exactly\nsource: %#v\ncontroller: %#v\nobserver: %#v", source.Hack, restoredController.Hack, restoredObserver.Hack)
	}

	coordinator.RequestSwitch("terminal-b")
	discardRevision := coordinator.Resolve(domain.TerminalSwitchDiscard)
	readUS8Transition(t, controllerMessages, "terminal-b", discardRevision, domain.PlayerRoleActive)
	readUS8Transition(t, observerMessages, "terminal-b", discardRevision, domain.PlayerRoleObserver)
	freshRevision := coordinator.ActivateSource()
	freshController := readUS8Transition(t, controllerMessages, "terminal-a", freshRevision, domain.PlayerRoleActive)
	freshObserver := readUS8Transition(t, observerMessages, "terminal-a", freshRevision, domain.PlayerRoleObserver)
	if freshController.Hack == nil || freshObserver.Hack == nil || sameUS8JSON(freshController.Hack, source.Hack) || !sameUS8JSON(freshController.Hack, freshObserver.Hack) {
		t.Fatalf("discard did not fan out one fresh public puzzle\nsource: %#v\ncontroller: %#v\nobserver: %#v", source.Hack, freshController.Hack, freshObserver.Hack)
	}

	writeJSON(t, controller, map[string]any{
		"type": MessageHackGuess, "requestId": "inactive-terminal-b", "broadcastId": us8BroadcastID,
		"terminalId": "terminal-b", "targetId": "crafted-inactive-target",
	})
	result := readUS8Message(t, controllerMessages)
	assertUS1ActionResult(t, result, "inactive-terminal-b", false, domain.ActionReasonStaleTerminal)
	if result.Revision != freshRevision {
		t.Fatalf("inactive-terminal result revision = %d, want %d", result.Revision, freshRevision)
	}
	assertNoUS8Message(t, observerMessages, "inactive crafted request reached observer")
	controller.CloseNow()
	observer.CloseNow()
}

func TestCoordinatedServerBroadcastLifetimeEndReselectStaleAndFreshProcess(t *testing.T) {
	liveService := live.New(nil, nil)
	coordinator := newUS9CoordinatorStub(liveService)
	server := startUS9TestServer(t, coordinator, liveService)

	state, err := coordinator.service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	characterID := state.Roster[0].ID
	firstBroadcast, err := coordinator.service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	firstBroadcastID := firstBroadcast.Broadcast.ID

	player, welcome := connectUS9Player(t, server.Info().URL, "")
	originalToken := domain.BrowserToken(welcome.BrowserToken)
	originalSessionID := welcome.State.SessionID
	originalFallback := welcome.State.FallbackName
	assertUS9UnassignedContext(t, welcome, firstBroadcastID, domain.PlayerPhaseSelecting, characterID)

	writeJSON(t, player, map[string]any{
		"type": MessageCharacterSelect, "requestId": "us9-first-selection",
		"broadcastId": firstBroadcastID, "characterId": characterID,
	})
	assigned := readUS2Message(t, player)
	assignedResult := readUS2Message(t, player)
	assertUS1AssignedState(t, assigned, MessagePlayerState, characterID, domain.PlayerRoleActive, domain.PlayerPhaseWaiting, "")
	assertUS1ActionResult(t, assignedResult, "us9-first-selection", true, domain.ActionReasonAccepted)

	if _, err := coordinator.service.RequestTerminalActivation(domain.TerminalTarget{
		TerminalID: "terminal-us9", TerminalName: "US9", Tree: serverTree(), IntroText: "WELCOME",
	}); err != nil {
		t.Fatal(err)
	}
	liveContext := readUS2Message(t, player)
	liveTerminal := readUS2Message(t, player)
	assertUS1AssignedState(t, liveContext, MessagePlayerState, characterID, domain.PlayerRoleActive, domain.PlayerPhaseControlling, "terminal-us9")
	if liveTerminal.Type != MessageTerminalLive || liveTerminal.TerminalID != "terminal-us9" || liveTerminal.Revision != liveContext.State.Revision {
		t.Fatalf("activated terminal = %#v, want terminal-us9 revision %d", liveTerminal, liveContext.State.Revision)
	}

	ended, err := coordinator.service.EndBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	endContext := readUS2Message(t, player)
	endClear := readUS2Message(t, player)
	assertUS9UnassignedContext(t, endContext, "", domain.PlayerPhaseNoBroadcast, characterID)
	if endContext.State.SessionID != originalSessionID || endContext.State.FallbackName != originalFallback {
		t.Fatalf("end-broadcast identity = %#v, want session %q fallback %q", endContext.State, originalSessionID, originalFallback)
	}
	if endClear.Type != MessageTerminalClear || endClear.Revision != ended.Revision || endContext.State.Revision != ended.Revision {
		t.Fatalf("end-broadcast ordering = context %#v clear %#v, want revision %d", endContext, endClear, ended.Revision)
	}

	writeJSON(t, player, map[string]any{
		"type": MessageNavAction, "requestId": "us9-old-action", "broadcastId": firstBroadcastID,
		"terminalId": "terminal-us9", "action": "back",
	})
	staleAction := readUS2Message(t, player)
	assertUS1ActionResult(t, staleAction, "us9-old-action", false, domain.ActionReasonStaleBroadcast)

	secondBroadcast, err := coordinator.service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	secondBroadcastID := secondBroadcast.Broadcast.ID
	if secondBroadcastID == firstBroadcastID {
		t.Fatalf("second broadcast reused ID %q", secondBroadcastID)
	}
	secondContext := readUS2Message(t, player)
	assertUS9UnassignedContext(t, secondContext, secondBroadcastID, domain.PlayerPhaseSelecting, characterID)
	if secondContext.State.SessionID != originalSessionID || secondContext.State.FallbackName != originalFallback {
		t.Fatalf("same-process broadcast lost recognized identity: %#v", secondContext.State)
	}

	writeJSON(t, player, map[string]any{
		"type": MessageCharacterSelect, "requestId": "us9-stale-selection",
		"broadcastId": firstBroadcastID, "characterId": characterID,
	})
	staleSelectionContext := readUS2Message(t, player)
	staleSelectionResult := readUS2Message(t, player)
	assertUS9UnassignedContext(t, staleSelectionContext, secondBroadcastID, domain.PlayerPhaseSelecting, characterID)
	assertUS1ActionResult(t, staleSelectionResult, "us9-stale-selection", false, domain.ActionReasonStaleBroadcast)

	writeJSON(t, player, map[string]any{
		"type": MessageCharacterSelect, "requestId": "us9-second-selection",
		"broadcastId": secondBroadcastID, "characterId": characterID,
	})
	reassigned := readUS2Message(t, player)
	reassignedResult := readUS2Message(t, player)
	assertUS1AssignedState(t, reassigned, MessagePlayerState, characterID, domain.PlayerRoleActive, domain.PlayerPhaseWaiting, "")
	assertUS1ActionResult(t, reassignedResult, "us9-second-selection", true, domain.ActionReasonAccepted)

	restartedLive := live.New(nil, nil)
	restartedCoordinator := newUS9CoordinatorStub(restartedLive)
	restartedServer := startUS9TestServer(t, restartedCoordinator, restartedLive)
	if _, err := restartedCoordinator.service.AddCharacter("Mara"); err != nil {
		t.Fatal(err)
	}
	restartedBroadcast, err := restartedCoordinator.service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	restartedPlayer, restartedWelcome := connectUS9Player(t, restartedServer.Info().URL, originalToken)
	defer restartedPlayer.CloseNow()
	if restartedWelcome.BrowserToken == string(originalToken) || restartedWelcome.State.SessionID == originalSessionID {
		t.Fatalf("fresh process recognized prior token/session: %#v", restartedWelcome)
	}
	assertUS9UnassignedContext(t, restartedWelcome, restartedBroadcast.Broadcast.ID, domain.PlayerPhaseSelecting, restartedCoordinator.characterID(t))
	defer player.CloseNow()
}

func TestCoordinatedServerBroadcastEndClearsActiveAndObserverPlayersAtOneRevision(t *testing.T) {
	liveService := live.New(nil, nil)
	coordinator := newUS9CoordinatorStub(liveService)
	server := startUS9TestServer(t, coordinator, liveService)

	firstRoster, err := coordinator.service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	secondRoster, err := coordinator.service.AddCharacter("Boone")
	if err != nil {
		t.Fatal(err)
	}
	if len(secondRoster.Roster) != 2 {
		t.Fatalf("roster = %#v, want two retained characters", secondRoster.Roster)
	}
	if _, err := coordinator.service.StartBroadcast(); err != nil {
		t.Fatal(err)
	}

	active, activeWelcome := connectUS9Player(t, server.Info().URL, "")
	observer, observerWelcome := connectUS9Player(t, server.Info().URL, "")
	defer active.CloseNow()
	defer observer.CloseNow()
	activeMessages := pumpUS2Messages(active)
	observerMessages := pumpUS2Messages(observer)

	if _, err := coordinator.service.AssignCharacter(activeWelcome.State.SessionID, firstRoster.Roster[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.service.AssignCharacter(observerWelcome.State.SessionID, secondRoster.Roster[1].ID); err != nil {
		t.Fatal(err)
	}
	activated, err := coordinator.service.RequestTerminalActivation(domain.TerminalTarget{
		TerminalID: "terminal-bug-004", TerminalName: "BUG-004", Tree: serverTree(), IntroText: "LIVE",
	})
	if err != nil {
		t.Fatal(err)
	}

	waitForState := func(messages <-chan serverMessage, revision uint64) serverMessage {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			select {
			case message, ok := <-messages:
				if !ok {
					t.Fatalf("connection closed before PLAYER_STATE revision %d", revision)
				}
				if message.Type == MessagePlayerState && message.State != nil && message.State.Revision == revision {
					return message
				}
			case <-deadline:
				t.Fatalf("PLAYER_STATE revision %d timed out", revision)
			}
		}
	}
	waitForTerminal := func(messages <-chan serverMessage, messageType string, revision uint64) serverMessage {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			select {
			case message, ok := <-messages:
				if !ok {
					t.Fatalf("connection closed before %s revision %d", messageType, revision)
				}
				if message.Type == messageType && message.Revision == revision {
					return message
				}
			case <-deadline:
				t.Fatalf("%s revision %d timed out", messageType, revision)
			}
		}
	}

	activeLive := waitForState(activeMessages, activated.Revision)
	observerLive := waitForState(observerMessages, activated.Revision)
	if activeLive.State.Role != domain.PlayerRoleActive || activeLive.State.Phase != domain.PlayerPhaseControlling {
		t.Fatalf("active state before end = %#v", activeLive.State)
	}
	if observerLive.State.Role != domain.PlayerRoleObserver || observerLive.State.Phase != domain.PlayerPhaseObserving {
		t.Fatalf("observer state before end = %#v", observerLive.State)
	}
	waitForTerminal(activeMessages, MessageTerminalLive, activated.Revision)
	waitForTerminal(observerMessages, MessageTerminalLive, activated.Revision)

	ended, err := coordinator.service.EndBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	for _, player := range []struct {
		name     string
		messages <-chan serverMessage
		welcome  serverMessage
	}{
		{name: "active", messages: activeMessages, welcome: activeWelcome},
		{name: "observer", messages: observerMessages, welcome: observerWelcome},
	} {
		context := waitForState(player.messages, ended.Revision)
		if context.State.Phase != domain.PlayerPhaseNoBroadcast || context.State.Role != domain.PlayerRoleUnassigned || context.State.Character != nil || context.State.BroadcastID != "" || context.State.ActiveTerminalID != "" {
			t.Fatalf("%s end context = %#v", player.name, context.State)
		}
		if context.State.SessionID != player.welcome.State.SessionID || context.State.FallbackName != player.welcome.State.FallbackName {
			t.Fatalf("%s retained identity = %#v, want session %q fallback %q", player.name, context.State, player.welcome.State.SessionID, player.welcome.State.FallbackName)
		}
		clear := waitForTerminal(player.messages, MessageTerminalClear, ended.Revision)
		if clear.Revision != context.State.Revision {
			t.Fatalf("%s clear revision = %d, context revision = %d", player.name, clear.Revision, context.State.Revision)
		}
	}

	if ended.Broadcast != nil || len(ended.Roster) != 2 || len(ended.Sessions) != 2 {
		t.Fatalf("end state = %#v, want no broadcast with retained roster and sessions", ended)
	}
	for _, session := range ended.Sessions {
		if session.Character != nil || session.Role != domain.PlayerRoleUnassigned {
			t.Fatalf("end state retained assignment or controller: %#v", session)
		}
	}
}

func TestCoordinatedServerResumesCanonicalRuntimeForNewTabAndReconnect(t *testing.T) {
	liveService := live.New(nil, nil)
	coordinator := newUS9CoordinatorStub(liveService)
	server := startUS9TestServer(t, coordinator, liveService)
	roster, err := coordinator.service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	broadcast, err := coordinator.service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}

	active, welcome := connectUS9Player(t, server.Info().URL, "")
	if _, err = coordinator.service.AssignCharacter(welcome.State.SessionID, roster.Roster[0].ID); err != nil {
		t.Fatal(err)
	}
	readUntilServerMessage(t, active, func(message serverMessage) bool {
		return message.Type == MessagePlayerState && message.State != nil && message.State.Character != nil
	})
	activated, err := coordinator.service.RequestTerminalActivation(domain.TerminalTarget{
		TerminalID: "terminal-resume", TerminalName: "Resume Terminal", Tree: serverTree(), HackLevel: 1, IntroText: "RESUME",
	})
	if err != nil {
		t.Fatal(err)
	}
	current := readUntilServerMessage(t, active, func(message serverMessage) bool {
		return message.Type == MessageTerminalLive && message.Revision == activated.Revision
	})
	targetID := firstCandidateID(current.Hack)
	if targetID == "" {
		t.Fatalf("active runtime has no candidate: %#v", current.Hack)
	}
	writeJSON(t, active, map[string]any{
		"type": MessageHackGuess, "requestId": "resume-mutation", "broadcastId": broadcast.Broadcast.ID,
		"terminalId": "terminal-resume", "targetId": targetID,
	})
	mutated := readUntilServerMessage(t, active, func(message serverMessage) bool {
		return message.Type == MessageTerminalLive && message.Revision > activated.Revision
	})
	readUntilServerMessage(t, active, func(message serverMessage) bool {
		return message.Type == MessageActionResult && message.RequestID == "resume-mutation"
	})

	newTab, tabWelcome := connectUS9Player(t, server.Info().URL, domain.BrowserToken(welcome.BrowserToken))
	if tabWelcome.State.SessionID != welcome.State.SessionID || tabWelcome.State.Role != domain.PlayerRoleActive || tabWelcome.State.ActiveTerminalID != "terminal-resume" {
		t.Fatalf("new-tab welcome lost identity/role/terminal: %#v", tabWelcome.State)
	}
	tabLive := readUntilServerMessage(t, newTab, func(message serverMessage) bool { return message.Type == MessageTerminalLive })
	if tabLive.TerminalID != mutated.TerminalID || !reflect.DeepEqual(tabLive.Nav, mutated.Nav) || !reflect.DeepEqual(tabLive.Hack, mutated.Hack) {
		t.Fatalf("new-tab runtime regenerated or drifted\nwant=%#v\ngot=%#v", mutated, tabLive)
	}

	active.CloseNow()
	newTab.CloseNow()
	waitForLogicalSessionPresence(t, coordinator.service, welcome.State.SessionID, false)
	reconnected, reconnectWelcome := connectUS9Player(t, server.Info().URL, domain.BrowserToken(welcome.BrowserToken))
	defer reconnected.CloseNow()
	if reconnectWelcome.State.SessionID != welcome.State.SessionID || reconnectWelcome.State.Role != domain.PlayerRoleActive || reconnectWelcome.State.ActiveTerminalID != "terminal-resume" {
		t.Fatalf("reconnect welcome lost identity/role/terminal: %#v", reconnectWelcome.State)
	}
	reconnectLive := readUntilServerMessage(t, reconnected, func(message serverMessage) bool { return message.Type == MessageTerminalLive })
	if reconnectLive.TerminalID != mutated.TerminalID || !reflect.DeepEqual(reconnectLive.Nav, mutated.Nav) || !reflect.DeepEqual(reconnectLive.Hack, mutated.Hack) {
		t.Fatalf("reconnect runtime regenerated or drifted\nwant=%#v\ngot=%#v", mutated, reconnectLive)
	}
}

func TestCoordinatedForceHackSuccessPublishesOneSolvedCanonicalProjection(t *testing.T) {
	liveService := live.New(nil, nil)
	coordinator := newUS9CoordinatorStub(liveService)
	server := startUS9TestServer(t, coordinator, liveService)
	firstRoster, err := coordinator.service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	secondRoster, err := coordinator.service.AddCharacter("Boone")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = coordinator.service.StartBroadcast(); err != nil {
		t.Fatal(err)
	}
	active, activeWelcome := connectUS9Player(t, server.Info().URL, "")
	observer, observerWelcome := connectUS9Player(t, server.Info().URL, "")
	defer active.CloseNow()
	defer observer.CloseNow()
	if _, err = coordinator.service.AssignCharacter(activeWelcome.State.SessionID, firstRoster.Roster[0].ID); err != nil {
		t.Fatal(err)
	}
	readUntilServerMessage(t, active, func(message serverMessage) bool { return message.Type == MessagePlayerState })
	readUntilServerMessage(t, observer, func(message serverMessage) bool { return message.Type == MessagePlayerState })
	if _, err = coordinator.service.AssignCharacter(observerWelcome.State.SessionID, secondRoster.Roster[1].ID); err != nil {
		t.Fatal(err)
	}
	readUntilServerMessage(t, active, func(message serverMessage) bool { return message.Type == MessagePlayerState })
	readUntilServerMessage(t, observer, func(message serverMessage) bool {
		return message.Type == MessagePlayerState && message.State != nil && message.State.Role == domain.PlayerRoleObserver
	})
	activated, err := coordinator.service.RequestTerminalActivation(domain.TerminalTarget{
		TerminalID: "terminal-force", TerminalName: "Force Terminal", Tree: serverTree(), HackLevel: 1, IntroText: "FORCE",
	})
	if err != nil {
		t.Fatal(err)
	}
	activeBefore := readUntilServerMessage(t, active, func(message serverMessage) bool {
		return message.Type == MessageTerminalLive && message.Revision == activated.Revision
	})
	observerBefore := readUntilServerMessage(t, observer, func(message serverMessage) bool {
		return message.Type == MessageTerminalLive && message.Revision == activated.Revision
	})
	if activeBefore.Hack == nil || observerBefore.Hack == nil || activeBefore.Hack.Solved || activeBefore.Hack.Failed {
		t.Fatalf("force precondition active=%#v observer=%#v", activeBefore.Hack, observerBefore.Hack)
	}

	forced, ok := coordinator.service.ForceHackSuccess()
	if !ok || forced == nil || !forced.Solved || forced.AttemptsLeft != activeBefore.Hack.AttemptsLeft {
		t.Fatalf("ForceHackSuccess() = %#v, %v", forced, ok)
	}
	forceRevision := coordinator.service.Revision()
	activeSolved := readUntilServerMessage(t, active, func(message serverMessage) bool {
		return message.Type == MessageTerminalLive && message.Revision == forceRevision
	})
	observerSolved := readUntilServerMessage(t, observer, func(message serverMessage) bool {
		return message.Type == MessageTerminalLive && message.Revision == forceRevision
	})
	if activeSolved.Hack == nil || observerSolved.Hack == nil || !activeSolved.Hack.Solved || !observerSolved.Hack.Solved ||
		activeSolved.Hack.AttemptsLeft != activeBefore.Hack.AttemptsLeft || !reflect.DeepEqual(activeSolved.Hack, observerSolved.Hack) {
		t.Fatalf("trusted force did not converge active=%#v observer=%#v", activeSolved.Hack, observerSolved.Hack)
	}
	if repeated, repeatedOK := coordinator.service.ForceHackSuccess(); repeatedOK || repeated != nil || coordinator.service.Revision() != forceRevision {
		t.Fatalf("repeated force = %#v, %v revision=%d", repeated, repeatedOK, coordinator.service.Revision())
	}
}

type us9CoordinatorStub struct {
	service *control.Service

	mu                  sync.RWMutex
	sessionByConnection map[domain.ConnectionID]domain.LogicalSessionID
	publish             func(control.Effect)
}

var us9ProcessSequence atomic.Uint64

type us9IDSource struct {
	process uint64
	next    atomic.Uint64
}

func (source *us9IDSource) Next() string {
	return fmt.Sprintf("us9-process-%d-id-%d", source.process, source.next.Add(1))
}

func newUS9CoordinatorStub(liveService *live.Service) *us9CoordinatorStub {
	stub := &us9CoordinatorStub{sessionByConnection: make(map[domain.ConnectionID]domain.LogicalSessionID)}
	stub.service = control.New(control.Config{
		IDs: &us9IDSource{process: us9ProcessSequence.Add(1)}, Runtime: liveService, Terminals: liveService, TrustedHack: liveService,
		Enqueue: func(effect control.Effect) {
			stub.mu.RLock()
			publish := stub.publish
			stub.mu.RUnlock()
			if publish != nil {
				publish(effect)
			}
		},
	})
	return stub
}

func (stub *us9CoordinatorStub) AttachConnection(connectionID domain.ConnectionID, token domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	returnedToken, state := stub.service.AttachConnection(connectionID, token)
	if state != nil {
		stub.mu.Lock()
		stub.sessionByConnection[connectionID] = state.SessionID
		stub.mu.Unlock()
	}
	return returnedToken, state
}

func (stub *us9CoordinatorStub) DetachConnection(connectionID domain.ConnectionID) {
	stub.mu.Lock()
	delete(stub.sessionByConnection, connectionID)
	stub.mu.Unlock()
	stub.service.DetachConnection(connectionID)
}

func (stub *us9CoordinatorStub) SelectCharacter(connectionID domain.ConnectionID, requestID string, broadcastID domain.BroadcastID, characterID domain.CharacterID) {
	stub.mu.RLock()
	sessionID := stub.sessionByConnection[connectionID]
	stub.mu.RUnlock()
	stub.service.SelectCharacter(control.CharacterSelection{
		ConnectionID: connectionID, SessionID: sessionID, RequestID: domain.RequestID(requestID),
		BroadcastID: broadcastID, CharacterID: characterID,
	})
}

func (stub *us9CoordinatorStub) DispatchPlayerAction(connectionID domain.ConnectionID, command domain.RuntimeCommand) {
	stub.service.DispatchPlayerAction(connectionID, command)
}

func (stub *us9CoordinatorStub) CurrentLiveForSession(sessionID domain.LogicalSessionID) (*domain.PublicLiveState, uint64, bool) {
	return stub.service.CurrentLiveForSession(sessionID)
}

func (stub *us9CoordinatorStub) characterID(t *testing.T) domain.CharacterID {
	t.Helper()
	state := stub.service.Snapshot()
	if len(state.Roster) != 1 {
		t.Fatalf("US9 roster = %#v, want one entry", state.Roster)
	}
	return state.Roster[0].ID
}

func startUS9TestServer(t *testing.T, coordinator *us9CoordinatorStub, liveService *live.Service) *Server {
	t.Helper()
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")}}
	server, err := NewServer(Config{Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: liveService, Coordinator: coordinator})
	if err != nil {
		t.Fatal(err)
	}
	coordinator.mu.Lock()
	coordinator.publish = server.PublishCoordinationEffect
	coordinator.mu.Unlock()
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

func connectUS9Player(t *testing.T, rawURL string, token domain.BrowserToken) (*websocket.Conn, serverMessage) {
	t.Helper()
	connection := dialPlayer(t, rawURL)
	hello := map[string]any{"type": MessageSessionHello}
	if token != "" {
		hello["browserToken"] = token
	}
	writeJSON(t, connection, hello)
	welcome := readUS2Message(t, connection)
	if welcome.Type != MessageSessionWelcome || welcome.BrowserToken == "" || welcome.State == nil {
		t.Fatalf("US9 handshake = %#v", welcome)
	}
	return connection, welcome
}

func readUntilServerMessage(t *testing.T, connection *websocket.Conn, matches func(serverMessage) bool) serverMessage {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		message := readUS2Message(t, connection)
		if matches(message) {
			return message
		}
	}
	t.Fatal("timed out waiting for coordinated server message")
	return serverMessage{}
}

func waitForLogicalSessionPresence(t *testing.T, service *control.Service, sessionID domain.LogicalSessionID, want bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := service.Snapshot()
		for _, session := range state.Sessions {
			if session.ID == sessionID && session.Connected == want {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("logical session %q connected state did not become %v", sessionID, want)
}

func assertUS9UnassignedContext(t *testing.T, message serverMessage, broadcastID domain.BroadcastID, phase domain.PlayerPhase, characterID domain.CharacterID) {
	t.Helper()
	if message.State == nil || message.State.Character != nil || message.State.Role != domain.PlayerRoleUnassigned || message.State.Phase != phase || message.State.BroadcastID != broadcastID || message.State.ActiveTerminalID != "" {
		t.Fatalf("US9 unassigned context = %#v, want broadcast=%q phase=%q", message, broadcastID, phase)
	}
	for _, entry := range message.State.Roster {
		if entry.ID == characterID && entry.Status == domain.RosterStatusAvailable {
			return
		}
	}
	t.Fatalf("US9 context omitted available character %q: %#v", characterID, message.State.Roster)
}

const us8BroadcastID domain.BroadcastID = "broadcast-us8"

type us8CoordinatorStub struct {
	mu sync.Mutex

	live         *live.Service
	revision     uint64
	next         int
	active       string
	pending      *string
	controller   domain.LogicalSessionID
	current      *domain.PublicLiveState
	preserved    *domain.PublicLiveState
	byToken      map[domain.BrowserToken]*us8Session
	byConnection map[domain.ConnectionID]*us8Session
	enqueue      func(control.Effect)
}

type us8Session struct {
	id          domain.LogicalSessionID
	token       domain.BrowserToken
	character   domain.PlayerCharacter
	connections map[domain.ConnectionID]struct{}
}

func newUS8CoordinatorStub(liveService *live.Service, source *domain.PublicLiveState) *us8CoordinatorStub {
	return &us8CoordinatorStub{
		live: liveService, revision: 80, active: "terminal-a", current: cloneUS8Live(source),
		byToken: make(map[domain.BrowserToken]*us8Session), byConnection: make(map[domain.ConnectionID]*us8Session),
	}
}

func (stub *us8CoordinatorStub) AttachConnection(connectionID domain.ConnectionID, token domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	session := stub.byToken[token]
	if token == "" || session == nil {
		stub.next++
		session = &us8Session{
			id: domain.LogicalSessionID(fmt.Sprintf("us8-session-%d", stub.next)), token: domain.BrowserToken(fmt.Sprintf("us8-token-%d", stub.next)),
			character:   domain.PlayerCharacter{ID: domain.CharacterID(fmt.Sprintf("us8-character-%d", stub.next)), Name: fmt.Sprintf("Character %d", stub.next)},
			connections: make(map[domain.ConnectionID]struct{}),
		}
		stub.byToken[session.token] = session
		if stub.controller == "" {
			stub.controller = session.id
		}
	}
	session.connections[connectionID] = struct{}{}
	stub.byConnection[connectionID] = session
	return session.token, stub.playerStateLocked(session)
}

func (stub *us8CoordinatorStub) DetachConnection(connectionID domain.ConnectionID) {
	stub.mu.Lock()
	if session := stub.byConnection[connectionID]; session != nil {
		delete(session.connections, connectionID)
	}
	delete(stub.byConnection, connectionID)
	stub.mu.Unlock()
}

func (*us8CoordinatorStub) SelectCharacter(domain.ConnectionID, string, domain.BroadcastID, domain.CharacterID) {
}

func (stub *us8CoordinatorStub) DispatchPlayerAction(connectionID domain.ConnectionID, command domain.RuntimeCommand) {
	stub.mu.Lock()
	reason := domain.ActionReasonInvalidSession
	if session := stub.byConnection[connectionID]; session != nil {
		switch {
		case session.id != stub.controller:
			reason = domain.ActionReasonNotController
		case command.BroadcastID != us8BroadcastID:
			reason = domain.ActionReasonStaleBroadcast
		case command.TerminalID != stub.active:
			reason = domain.ActionReasonStaleTerminal
		default:
			reason = domain.ActionReasonInvalidAction
		}
	}
	result := domain.ActionResult{RequestID: command.RequestID, Reason: reason, Revision: stub.revision}
	revision := stub.revision
	enqueue := stub.enqueue
	stub.mu.Unlock()
	if enqueue != nil {
		enqueue(control.Effect{Revision: revision, ConnectionID: connectionID, Result: &result})
	}
}

func (stub *us8CoordinatorStub) RequestSwitch(target string) {
	stub.mu.Lock()
	stub.pending = &target
	stub.mu.Unlock()
}

func (stub *us8CoordinatorStub) Resolve(choice domain.TerminalSwitchChoice) uint64 {
	stub.mu.Lock()
	if stub.pending == nil {
		revision := stub.revision
		stub.mu.Unlock()
		return revision
	}
	target := *stub.pending
	stub.pending = nil
	if choice == domain.TerminalSwitchCancel {
		stub.revision++
		revision := stub.revision
		stub.mu.Unlock()
		return revision
	}
	if choice == domain.TerminalSwitchPreserve {
		stub.preserved = cloneUS8Live(stub.current)
	} else {
		stub.preserved = nil
	}
	stub.mu.Unlock()
	return stub.activateTarget(target)
}

func (stub *us8CoordinatorStub) ActivateSource() uint64 {
	stub.mu.Lock()
	preserved := cloneUS8Live(stub.preserved)
	stub.mu.Unlock()
	if preserved != nil {
		return stub.publishActive(preserved)
	}
	fresh := stub.live.Set("terminal-a", "Terminal A", serverTree(), 1, "SOURCE")
	return stub.publishActive(fresh)
}

func (stub *us8CoordinatorStub) activateTarget(target string) uint64 {
	if target == "" {
		stub.live.Clear()
		return stub.publishActive(nil)
	}
	state := stub.live.Set(target, "Terminal B", serverTree(), 0, "TARGET")
	return stub.publishActive(state)
}

func (stub *us8CoordinatorStub) publishActive(state *domain.PublicLiveState) uint64 {
	stub.mu.Lock()
	stub.current = cloneUS8Live(state)
	stub.active = ""
	if state != nil {
		stub.active = state.TerminalID
	}
	stub.revision++
	revision := stub.revision
	sessions := make([]*us8Session, 0, len(stub.byToken))
	for _, session := range stub.byToken {
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(left, right int) bool { return sessions[left].id < sessions[right].id })
	effects := make([]control.Effect, 0, len(sessions)+1)
	for _, session := range sessions {
		effects = append(effects, control.Effect{Revision: revision, SessionID: session.id, Player: stub.playerStateLocked(session)})
	}
	if state == nil {
		effects = append(effects, control.Effect{Revision: revision, ClearLiveTerminal: true})
	} else {
		effects = append(effects, control.Effect{Revision: revision, Live: cloneUS8Live(state)})
	}
	enqueue := stub.enqueue
	stub.mu.Unlock()
	for _, effect := range effects {
		if enqueue != nil {
			enqueue(effect)
		}
	}
	return revision
}

func (stub *us8CoordinatorStub) playerStateLocked(session *us8Session) *domain.PlayerState {
	role := domain.PlayerRoleObserver
	phase := domain.PlayerPhaseWaiting
	if session.id == stub.controller {
		role = domain.PlayerRoleActive
	}
	if stub.active != "" && role == domain.PlayerRoleActive {
		phase = domain.PlayerPhaseControlling
	} else if stub.active != "" {
		phase = domain.PlayerPhaseObserving
	}
	return &domain.PlayerState{
		Revision: stub.revision, SessionID: session.id, FallbackName: string(session.id),
		Character: &domain.PlayerCharacter{ID: session.character.ID, Name: session.character.Name},
		Role:      role, Phase: phase, BroadcastID: us8BroadcastID, ActiveTerminalID: stub.active,
	}
}

func (stub *us8CoordinatorStub) currentLive() *domain.PublicLiveState {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	return cloneUS8Live(stub.current)
}

func cloneUS8Live(state *domain.PublicLiveState) *domain.PublicLiveState {
	if state == nil {
		return nil
	}
	payload, _ := json.Marshal(state)
	var clone domain.PublicLiveState
	_ = json.Unmarshal(payload, &clone)
	return &clone
}

func sameUS8JSON(left, right any) bool {
	leftJSON, leftErr := json.Marshal(left)
	rightJSON, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftJSON, rightJSON)
}

func startUS8TestServer(t *testing.T, coordinator *us8CoordinatorStub) *Server {
	t.Helper()
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")}}
	server, err := NewServer(Config{Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: coordinator.live, Coordinator: coordinator})
	if err != nil {
		t.Fatal(err)
	}
	coordinator.enqueue = server.PublishCoordinationEffect
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

func connectUS8Player(t *testing.T, rawURL string, token domain.BrowserToken) (*websocket.Conn, serverMessage) {
	t.Helper()
	connection := dialPlayer(t, rawURL)
	hello := map[string]any{"type": MessageSessionHello}
	if token != "" {
		hello["browserToken"] = token
	}
	writeJSON(t, connection, hello)
	welcome := readUS2Message(t, connection)
	if welcome.Type != MessageSessionWelcome || welcome.State == nil {
		t.Fatalf("US8 handshake = %#v", welcome)
	}
	initial := readUS2Message(t, connection)
	if initial.Type != MessageTerminalLive || initial.TerminalID != "terminal-a" {
		t.Fatalf("US8 initial terminal = %#v", initial)
	}
	return connection, welcome
}

func readUS8Transition(t *testing.T, messages <-chan serverMessage, terminalID string, revision uint64, role domain.PlayerRole) serverMessage {
	t.Helper()
	player := readUS8Message(t, messages)
	if player.Type != MessagePlayerState || player.State == nil || player.State.Revision != revision || player.State.Role != role || player.State.ActiveTerminalID != terminalID {
		t.Fatalf("US8 player transition = %#v, want terminal=%q role=%q revision=%d", player, terminalID, role, revision)
	}
	terminal := readUS8Message(t, messages)
	if terminal.Type != MessageTerminalLive || terminal.Revision != revision || terminal.TerminalID != terminalID {
		t.Fatalf("US8 terminal transition = %#v, want terminal=%q revision=%d", terminal, terminalID, revision)
	}
	return terminal
}

func readUS8Message(t *testing.T, messages <-chan serverMessage) serverMessage {
	t.Helper()
	select {
	case message, ok := <-messages:
		if !ok {
			t.Fatal("US8 connection closed before expected message")
		}
		return message
	case <-time.After(time.Second):
		t.Fatal("US8 message timed out")
	}
	return serverMessage{}
}

func assertNoUS8Message(t *testing.T, messages <-chan serverMessage, failure string) {
	t.Helper()
	select {
	case message := <-messages:
		t.Fatalf("%s: %#v", failure, message)
	case <-time.After(50 * time.Millisecond):
	}
}

const us7BroadcastID domain.BroadcastID = "broadcast-us7"

type us7CoordinatorStub struct {
	mu sync.Mutex

	live         *live.Service
	revision     uint64
	next         int
	active       string
	controller   domain.LogicalSessionID
	byToken      map[domain.BrowserToken]*us7Session
	byConnection map[domain.ConnectionID]*us7Session
	enqueue      func(control.Effect)
}

type us7Session struct {
	id          domain.LogicalSessionID
	token       domain.BrowserToken
	fallback    string
	character   domain.PlayerCharacter
	assigned    bool
	connections map[domain.ConnectionID]struct{}
}

func newUS7CoordinatorStub(liveService *live.Service) *us7CoordinatorStub {
	return &us7CoordinatorStub{
		live: liveService, revision: 40,
		byToken: make(map[domain.BrowserToken]*us7Session), byConnection: make(map[domain.ConnectionID]*us7Session),
	}
}

func (stub *us7CoordinatorStub) AttachConnection(connectionID domain.ConnectionID, token domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	session := stub.byToken[token]
	if token == "" || session == nil {
		stub.next++
		session = &us7Session{
			id: domain.LogicalSessionID(fmt.Sprintf("us7-session-%d", stub.next)), token: domain.BrowserToken(fmt.Sprintf("us7-token-%d", stub.next)),
			fallback: fmt.Sprintf("PLAYER %d", stub.next), character: domain.PlayerCharacter{ID: domain.CharacterID(fmt.Sprintf("us7-character-%d", stub.next)), Name: fmt.Sprintf("Character %d", stub.next)},
			connections: make(map[domain.ConnectionID]struct{}),
		}
		stub.byToken[session.token] = session
	}
	session.connections[connectionID] = struct{}{}
	stub.byConnection[connectionID] = session
	return session.token, stub.playerStateLocked(session)
}

func (stub *us7CoordinatorStub) DetachConnection(connectionID domain.ConnectionID) {
	stub.mu.Lock()
	if session := stub.byConnection[connectionID]; session != nil {
		delete(session.connections, connectionID)
	}
	delete(stub.byConnection, connectionID)
	stub.mu.Unlock()
}

func (*us7CoordinatorStub) SelectCharacter(domain.ConnectionID, string, domain.BroadcastID, domain.CharacterID) {
}

func (stub *us7CoordinatorStub) DispatchPlayerAction(connectionID domain.ConnectionID, command domain.RuntimeCommand) {
	stub.mu.Lock()
	session := stub.byConnection[connectionID]
	reason := domain.ActionReasonInvalidSession
	if session != nil {
		switch {
		case !session.assigned:
			reason = domain.ActionReasonUnassigned
		case session.id != stub.controller:
			reason = domain.ActionReasonNotController
		case command.BroadcastID != us7BroadcastID:
			reason = domain.ActionReasonStaleBroadcast
		case command.TerminalID != stub.active:
			reason = domain.ActionReasonStaleTerminal
		default:
			reason = domain.ActionReasonInvalidAction
		}
	}
	result := domain.ActionResult{RequestID: command.RequestID, Reason: reason, Revision: stub.revision}
	enqueue := stub.enqueue
	revision := stub.revision
	stub.mu.Unlock()
	if enqueue != nil {
		enqueue(control.Effect{Revision: revision, ConnectionID: connectionID, Result: &result})
	}
}

func (stub *us7CoordinatorStub) Assign(sessionID domain.LogicalSessionID) {
	stub.mu.Lock()
	var target *us7Session
	for _, session := range stub.byToken {
		if session.id == sessionID {
			target = session
			break
		}
	}
	if target == nil || target.assigned {
		stub.mu.Unlock()
		return
	}
	target.assigned = true
	if stub.controller == "" {
		stub.controller = target.id
	}
	stub.revision++
	revision := stub.revision
	state := stub.playerStateLocked(target)
	liveState := stub.live.Snapshot()
	enqueue := stub.enqueue
	stub.mu.Unlock()
	if enqueue != nil {
		enqueue(control.Effect{Revision: revision, SessionID: target.id, Player: state})
		if liveState != nil {
			enqueue(control.Effect{Revision: revision, SessionID: target.id, Live: liveState})
		}
	}
}

func (stub *us7CoordinatorStub) Switch(terminalID string) uint64 {
	if terminalID == "" {
		stub.live.Clear()
	} else {
		stub.live.Set(terminalID, terminalID, serverTree(), 0, "WELCOME")
	}
	stub.mu.Lock()
	stub.active = terminalID
	stub.revision++
	revision := stub.revision
	sessions := make([]*us7Session, 0, len(stub.byToken))
	for _, session := range stub.byToken {
		if session.assigned {
			sessions = append(sessions, session)
		}
	}
	sort.Slice(sessions, func(left, right int) bool { return sessions[left].id < sessions[right].id })
	effects := make([]control.Effect, 0, len(sessions)+1)
	for _, session := range sessions {
		effects = append(effects, control.Effect{Revision: revision, SessionID: session.id, Player: stub.playerStateLocked(session)})
	}
	if terminalID == "" {
		effects = append(effects, control.Effect{Revision: revision, ClearLiveTerminal: true})
	} else {
		effects = append(effects, control.Effect{Revision: revision, Live: stub.live.Snapshot()})
	}
	enqueue := stub.enqueue
	stub.mu.Unlock()
	for _, effect := range effects {
		if enqueue != nil {
			enqueue(effect)
		}
	}
	return revision
}

func (stub *us7CoordinatorStub) playerStateLocked(session *us7Session) *domain.PlayerState {
	state := &domain.PlayerState{
		Revision: stub.revision, SessionID: session.id, FallbackName: session.fallback,
		Role: domain.PlayerRoleUnassigned, Phase: domain.PlayerPhaseSelecting, BroadcastID: us7BroadcastID,
		ActiveTerminalID: stub.active,
	}
	if !session.assigned {
		return state
	}
	state.Character = &domain.PlayerCharacter{ID: session.character.ID, Name: session.character.Name}
	state.Role = domain.PlayerRoleObserver
	state.Phase = domain.PlayerPhaseWaiting
	if session.id == stub.controller {
		state.Role = domain.PlayerRoleActive
	}
	if stub.active != "" && state.Role == domain.PlayerRoleActive {
		state.Phase = domain.PlayerPhaseControlling
	} else if stub.active != "" {
		state.Phase = domain.PlayerPhaseObserving
	}
	return state
}

func (stub *us7CoordinatorStub) currentRevision() uint64 {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	return stub.revision
}

func (stub *us7CoordinatorStub) waitForConnections(t *testing.T, sessionID domain.LogicalSessionID, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		stub.mu.Lock()
		got := 0
		for _, session := range stub.byToken {
			if session.id == sessionID {
				got = len(session.connections)
				break
			}
		}
		stub.mu.Unlock()
		if got == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("session %q connection count did not reach %d", sessionID, want)
}

func startUS7TestServer(t *testing.T, coordinator *us7CoordinatorStub) *Server {
	t.Helper()
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")}}
	server, err := NewServer(Config{Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: coordinator.live, Coordinator: coordinator})
	if err != nil {
		t.Fatal(err)
	}
	coordinator.enqueue = server.PublishCoordinationEffect
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

func connectUS7Player(t *testing.T, rawURL string, token domain.BrowserToken) (*websocket.Conn, serverMessage) {
	t.Helper()
	connection := dialPlayer(t, rawURL)
	hello := map[string]any{"type": MessageSessionHello}
	if token != "" {
		hello["browserToken"] = token
	}
	writeJSON(t, connection, hello)
	welcome := readUS2Message(t, connection)
	if welcome.Type != MessageSessionWelcome || welcome.State == nil {
		t.Fatalf("US7 handshake = %#v", welcome)
	}
	return connection, welcome
}

func assertUS7PlayerState(t *testing.T, connection *websocket.Conn, role domain.PlayerRole, phase domain.PlayerPhase, terminalID string, revision uint64) {
	t.Helper()
	message := readUS2Message(t, connection)
	if message.Type != MessagePlayerState || message.State == nil || message.State.Role != role || message.State.Phase != phase || message.State.ActiveTerminalID != terminalID || message.State.Revision != revision {
		t.Fatalf("US7 player state = %#v, want role=%q phase=%q terminal=%q revision=%d", message, role, phase, terminalID, revision)
	}
}

func assertUS5RoleChange(t *testing.T, connection *websocket.Conn, revision uint64, role domain.PlayerRole, characterID domain.CharacterID) {
	t.Helper()
	state := readUS2Message(t, connection)
	phase := domain.PlayerPhaseObserving
	if role == domain.PlayerRoleActive {
		phase = domain.PlayerPhaseControlling
	}
	assertUS1AssignedState(t, state, MessagePlayerState, characterID, role, phase, "terminal-1")
	if state.State.Revision != revision {
		t.Fatalf("role change revision = %d, want %d", state.State.Revision, revision)
	}
}

const us2BroadcastID domain.BroadcastID = "broadcast-us2"

// us2PlayerCoordinator is the sender-aware transport seam expected from T035.
// Server tests use a deterministic coordinator double so they cover socket
// dispatch, ordered fanout, and concrete-connection result targeting without
// duplicating the canonical authorization suite in internal/control.
type us2PlayerCoordinator interface {
	PlayerCoordinator
	DispatchPlayerAction(domain.ConnectionID, domain.RuntimeCommand)
}

var _ us2PlayerCoordinator = (*us2CoordinatorStub)(nil)

type us2CoordinatorStub struct {
	mu sync.Mutex

	live                 *live.Service
	revision             uint64
	nextSession          int
	controller           domain.LogicalSessionID
	sessionsByConnection map[domain.ConnectionID]*us2Session
	sessionsByToken      map[domain.BrowserToken]*us2Session
	accepted             int
	dispatched           chan struct{}
	actionStarted        chan struct{}
	actionRelease        <-chan struct{}
	enqueue              func(control.Effect)
}

type us2Session struct {
	id           domain.LogicalSessionID
	browserToken domain.BrowserToken
	fallbackName string
	character    domain.PlayerCharacter
	requests     map[domain.RequestID]domain.RequestResultRecord
	connections  map[domain.ConnectionID]struct{}
}

func newUS2CoordinatorStub(liveService *live.Service) *us2CoordinatorStub {
	return &us2CoordinatorStub{
		live:                 liveService,
		revision:             10,
		sessionsByConnection: make(map[domain.ConnectionID]*us2Session),
		sessionsByToken:      make(map[domain.BrowserToken]*us2Session),
		dispatched:           make(chan struct{}, 128),
	}
}

func (stub *us2CoordinatorStub) AttachConnection(connectionID domain.ConnectionID, token domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	session := stub.sessionsByToken[token]
	if token == "" || session == nil {
		stub.nextSession++
		session = &us2Session{
			id:           domain.LogicalSessionID(fmt.Sprintf("us2-session-%d", stub.nextSession)),
			browserToken: domain.BrowserToken(fmt.Sprintf("us2-token-%d", stub.nextSession)),
			fallbackName: fmt.Sprintf("PLAYER %d", stub.nextSession),
			character: domain.PlayerCharacter{
				ID: domain.CharacterID(fmt.Sprintf("character-%d", stub.nextSession)), Name: fmt.Sprintf("Character %d", stub.nextSession),
			},
			requests:    make(map[domain.RequestID]domain.RequestResultRecord),
			connections: make(map[domain.ConnectionID]struct{}),
		}
		stub.sessionsByToken[session.browserToken] = session
		if stub.controller == "" {
			stub.controller = session.id
		}
	}
	session.connections[connectionID] = struct{}{}
	stub.sessionsByConnection[connectionID] = session
	return session.browserToken, stub.playerState(session)
}

func (stub *us2CoordinatorStub) DetachConnection(connectionID domain.ConnectionID) {
	stub.mu.Lock()
	if session := stub.sessionsByConnection[connectionID]; session != nil {
		delete(session.connections, connectionID)
	}
	delete(stub.sessionsByConnection, connectionID)
	stub.mu.Unlock()
}

func (stub *us2CoordinatorStub) SelectCharacter(domain.ConnectionID, string, domain.BroadcastID, domain.CharacterID) {
}

func (stub *us2CoordinatorStub) DispatchPlayerAction(connectionID domain.ConnectionID, command domain.RuntimeCommand) {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if stub.actionStarted != nil {
		started := stub.actionStarted
		release := stub.actionRelease
		stub.actionStarted = nil
		stub.actionRelease = nil
		close(started)
		<-release
	}
	select {
	case stub.dispatched <- struct{}{}:
	default:
	}

	session := stub.sessionsByConnection[connectionID]
	result := domain.ActionResult{RequestID: command.RequestID, Revision: stub.revision}
	if session == nil {
		result.Reason = domain.ActionReasonInvalidSession
		stub.emit(control.Effect{ConnectionID: connectionID, Result: &result})
		return
	}
	fingerprint := fmt.Sprintf("%#v", command)
	if cached, exists := session.requests[command.RequestID]; exists {
		if cached.Fingerprint == fingerprint {
			result = cached.Result
		} else {
			result.Reason = domain.ActionReasonDuplicate
		}
		stub.emit(control.Effect{ConnectionID: connectionID, SessionID: session.id, Result: &result})
		return
	}
	if session.id != stub.controller {
		result.Reason = domain.ActionReasonNotController
		session.requests[command.RequestID] = domain.RequestResultRecord{Fingerprint: fingerprint, Result: result}
		stub.emit(control.Effect{ConnectionID: connectionID, SessionID: session.id, Result: &result})
		return
	}
	if command.BroadcastID != us2BroadcastID {
		result.Reason = domain.ActionReasonStaleBroadcast
	} else if snapshot := stub.live.Snapshot(); snapshot == nil || command.TerminalID != snapshot.TerminalID {
		result.Reason = domain.ActionReasonStaleTerminal
	} else if !stub.apply(command) {
		result.Reason = domain.ActionReasonInvalidAction
	} else {
		stub.revision++
		stub.accepted++
		result.Accepted = true
		result.Reason = domain.ActionReasonAccepted
		result.Revision = stub.revision
	}
	session.requests[command.RequestID] = domain.RequestResultRecord{Fingerprint: fingerprint, Result: result}
	if result.Accepted {
		stub.emit(control.Effect{Live: stub.live.Snapshot()})
	}
	stub.emit(control.Effect{ConnectionID: connectionID, SessionID: session.id, Result: &result})
}

// SetActiveController models the T062 command and emits the detached per-
// session effects that T065 must fan to every concrete connection. It shares
// the same mutex as DispatchPlayerAction so the socket tests can prove the
// authoritative before/after order without implementing production logic.
func (stub *us2CoordinatorStub) SetActiveController(sessionID domain.LogicalSessionID) (uint64, bool) {
	stub.mu.Lock()
	var target *us2Session
	for _, session := range stub.sessionsByToken {
		if session.id == sessionID {
			target = session
			break
		}
	}
	if target == nil || len(target.connections) == 0 || target.id == stub.controller {
		revision := stub.revision
		stub.mu.Unlock()
		return revision, false
	}
	stub.controller = target.id
	stub.revision++
	revision := stub.revision
	sessions := make(map[domain.LogicalSessionID]*us2Session)
	for _, session := range stub.sessionsByToken {
		sessions[session.id] = session
	}
	ids := make([]domain.LogicalSessionID, 0, len(sessions))
	for id := range sessions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(left, right int) bool { return ids[left] < ids[right] })
	effects := make([]control.Effect, 0, len(ids))
	for _, id := range ids {
		effects = append(effects, control.Effect{Revision: revision, SessionID: id, Player: stub.playerState(sessions[id])})
	}
	enqueue := stub.enqueue
	stub.mu.Unlock()
	for _, effect := range effects {
		if enqueue != nil {
			enqueue(effect)
		}
	}
	return revision, true
}

func (stub *us2CoordinatorStub) blockNextAction(started chan struct{}, release <-chan struct{}) {
	stub.mu.Lock()
	stub.actionStarted = started
	stub.actionRelease = release
	stub.mu.Unlock()
}

func (stub *us2CoordinatorStub) waitForConnections(t *testing.T, sessionID domain.LogicalSessionID, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		stub.mu.Lock()
		got := 0
		for _, session := range stub.sessionsByToken {
			if session.id == sessionID {
				got = len(session.connections)
				break
			}
		}
		stub.mu.Unlock()
		if got == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("session %q connection count did not reach %d", sessionID, want)
}

func (stub *us2CoordinatorStub) controllerSession() domain.LogicalSessionID {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	return stub.controller
}

func (stub *us2CoordinatorStub) roleForSession(sessionID domain.LogicalSessionID) domain.PlayerRole {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	if sessionID == stub.controller {
		return domain.PlayerRoleActive
	}
	return domain.PlayerRoleObserver
}

func (stub *us2CoordinatorStub) apply(command domain.RuntimeCommand) bool {
	switch command.Kind {
	case domain.RuntimeCommandNavAction:
		_, ok := stub.live.ApplyNav(command.Action, command.NodeID)
		return ok
	case domain.RuntimeCommandHackGuess:
		_, ok := stub.live.ApplyHackGuess(command.TargetID)
		return ok
	case domain.RuntimeCommandHackPattern:
		applied := false
		stub.live.ApplyHackPattern(command.PatternID, func(*domain.PublicHackState) { applied = true })
		return applied
	default:
		return false
	}
}

func (stub *us2CoordinatorStub) playerState(session *us2Session) *domain.PlayerState {
	snapshot := stub.live.Snapshot()
	terminalID := ""
	if snapshot != nil {
		terminalID = snapshot.TerminalID
	}
	role := domain.PlayerRoleObserver
	phase := domain.PlayerPhaseObserving
	if session.id == stub.controller {
		role = domain.PlayerRoleActive
		phase = domain.PlayerPhaseControlling
	}
	return &domain.PlayerState{
		Revision: stub.revision, SessionID: session.id, FallbackName: session.fallbackName,
		Character: &domain.PlayerCharacter{ID: session.character.ID, Name: session.character.Name},
		Role:      role, Phase: phase, BroadcastID: us2BroadcastID, ActiveTerminalID: terminalID,
		Roster: []domain.PlayerRosterEntry{{ID: session.character.ID, Name: session.character.Name, Status: domain.RosterStatusClaimed}},
	}
}

func (stub *us2CoordinatorStub) emit(effect control.Effect) {
	effect.Revision = stub.revision
	if stub.enqueue != nil {
		stub.enqueue(effect)
	}
}

func (stub *us2CoordinatorStub) acceptedMutations() int {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	return stub.accepted
}

func (stub *us2CoordinatorStub) currentRevision() uint64 {
	stub.mu.Lock()
	defer stub.mu.Unlock()
	return stub.revision
}

func (stub *us2CoordinatorStub) waitForDispatches(t *testing.T, want int) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for count := 0; count < want; count++ {
		select {
		case <-stub.dispatched:
		case <-deadline:
			t.Fatalf("coordinator dispatched %d actions, want %d", count, want)
		}
	}
}

func startUS2TestServer(t *testing.T, coordinator *us2CoordinatorStub, queueSize int) *Server {
	t.Helper()
	assets := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")}}
	server, err := NewServer(Config{
		Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: coordinator.live,
		Coordinator: coordinator, QueueSize: queueSize,
	})
	if err != nil {
		t.Fatal(err)
	}
	coordinator.enqueue = server.PublishCoordinationEffect
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

func connectUS2Player(t *testing.T, rawURL string, token domain.BrowserToken) (*websocket.Conn, serverMessage) {
	t.Helper()
	connection := dialPlayer(t, rawURL)
	hello := map[string]any{"type": MessageSessionHello}
	if token != "" {
		hello["browserToken"] = token
	}
	writeJSON(t, connection, hello)
	welcome := readUS2Message(t, connection)
	if welcome.Type != MessageSessionWelcome || welcome.State == nil || welcome.BrowserToken == "" {
		t.Fatalf("coordinated handshake = %#v", welcome)
	}
	liveMessage := readUS2Message(t, connection)
	if liveMessage.Type != MessageTerminalLive || liveMessage.Revision != welcome.State.Revision {
		t.Fatalf("initial coordinated terminal = %#v, want revision %d", liveMessage, welcome.State.Revision)
	}
	return connection, welcome
}

func readUS2Message(t *testing.T, connection *websocket.Conn) serverMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()
	messageType, payload, err := connection.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageText {
		t.Fatalf("message type = %v, want text", messageType)
	}
	var message serverMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatalf("decode %s: %v", payload, err)
	}
	return message
}

func pumpUS2Messages(connection *websocket.Conn) <-chan serverMessage {
	messages := make(chan serverMessage, 128)
	go func() {
		defer close(messages)
		for {
			_, payload, err := connection.Read(context.Background())
			if err != nil {
				return
			}
			var message serverMessage
			if json.Unmarshal(payload, &message) == nil {
				messages <- message
			}
		}
	}()
	return messages
}

func waitForUS2Revision(t *testing.T, messages <-chan serverMessage, messageType string, revision uint64) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case message, ok := <-messages:
			if !ok {
				t.Fatalf("connection closed before %s revision %d", messageType, revision)
			}
			if message.Type == messageType && message.Revision == revision {
				return
			}
		case <-deadline:
			t.Fatalf("did not receive %s revision %d", messageType, revision)
		}
	}
}

func TestServerConvergesFourThroughSevenClients(t *testing.T) {
	for clientTotal := 4; clientTotal <= 7; clientTotal++ {
		t.Run(fmt.Sprintf("%d_clients", clientTotal), func(t *testing.T) {
			service := live.New(&serverRandom{values: []int{0, 1, 2, 3}}, serverWords{})
			service.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
			counts := &countRecorder{}
			server := startTestServer(t, service, counts.Add, 16)

			clients := make([]*websocket.Conn, 0, clientTotal)
			patternID := ""
			for range clientTotal {
				connection := dialPlayer(t, server.Info().URL)
				clients = append(clients, connection)
				message := readMessage(t, connection)
				if message.Type != MessageTerminalLive || message.Nav == nil || message.Hack == nil || len(message.Hack.Patterns) == 0 || !reflect.DeepEqual(message.Nav.Path, []string{"root"}) {
					t.Fatalf("initial message = %#v", message)
				}
				if patternID == "" {
					patternID = message.Hack.Patterns[0].ID
				}
			}
			waitForCount(t, counts, clientTotal)

			writeJSON(t, clients[0], map[string]any{"type": MessageHackPattern, "patternId": patternID})
			patternState := readConvergedMessages(t, clients, MessageHackState)
			if !publicPatternUsed(patternState.Hack, patternID) {
				t.Fatalf("pattern state did not converge as used: %#v", patternState.Hack)
			}
			candidateID := firstCandidateID(patternState.Hack)
			if candidateID == "" {
				t.Fatal("pattern state contained no guessable candidate")
			}
			writeJSON(t, clients[1], map[string]any{"type": MessageHackGuess, "targetId": candidateID})
			readConvergedMessages(t, clients, MessageHackState)

			inDocs := false
			for actionIndex := 0; actionIndex < 23; actionIndex++ {
				request := map[string]any{"type": MessageNavAction, "action": "enter", "nodeId": "docs"}
				wantPath := []string{"root", "docs"}
				if inDocs {
					request = map[string]any{"type": MessageNavAction, "action": "back"}
					wantPath = []string{"root"}
				}
				writeJSON(t, clients[actionIndex%len(clients)], request)
				message := readConvergedMessages(t, clients, MessageNavState)
				if message.Nav == nil || !reflect.DeepEqual(message.Nav.Path, wantPath) {
					t.Fatalf("action %d navigation = %#v, want path %v", actionIndex+3, message, wantPath)
				}
				inDocs = !inDocs
			}
		})
	}
}

func TestReconnectReceivesCurrentStateAndClearRemovesIt(t *testing.T) {
	service := live.New(nil, nil)
	service.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	server := startTestServer(t, service, nil, 8)

	first := dialPlayer(t, server.Info().URL)
	second := dialPlayer(t, server.Info().URL)
	readMessage(t, first)
	readMessage(t, second)
	writeJSON(t, first, map[string]any{"type": MessageNavAction, "action": "enter", "nodeId": "docs"})
	readMessage(t, first)
	readMessage(t, second)
	_ = second.Close(websocket.StatusNormalClosure, "reconnect")

	writeJSON(t, first, map[string]any{"type": MessageNavAction, "action": "entry", "nodeId": "report"})
	if message := readMessage(t, first); message.Nav == nil || message.Nav.Mode != "entry" {
		t.Fatalf("current client navigation = %#v", message)
	}

	reconnected := dialPlayer(t, server.Info().URL)
	defer reconnected.CloseNow()
	snapshot := readMessage(t, reconnected)
	if snapshot.Type != MessageTerminalLive || snapshot.Nav == nil || snapshot.Nav.Mode != "entry" || !reflect.DeepEqual(snapshot.Nav.Path, []string{"root", "docs"}) {
		t.Fatalf("reconnect snapshot = %#v", snapshot)
	}

	service.Clear()
	server.PublishClear()
	if message := readMessage(t, first); message.Type != MessageTerminalClear {
		t.Fatalf("first clear = %#v", message)
	}
	if message := readMessage(t, reconnected); message.Type != MessageTerminalClear {
		t.Fatalf("reconnected clear = %#v", message)
	}

	afterClear := dialPlayer(t, server.Info().URL)
	defer afterClear.CloseNow()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, _, err := afterClear.Read(ctx); err == nil {
		t.Fatal("connection after clear received a stale live snapshot")
	}
}

func TestSlowClientDoesNotBlockBroadcastOrFastClient(t *testing.T) {
	service := live.New(nil, nil)
	service.Set("terminal-1", "Overseer", serverTreeWithLargeBody(), 0, "WELCOME")
	server := startTestServer(t, service, nil, 8)

	slow := dialPlayer(t, server.Info().URL)
	defer slow.CloseNow()
	readMessage(t, slow)
	fast := dialPlayer(t, server.Info().URL)
	defer fast.CloseNow()
	readMessage(t, fast)

	fastMessages := make(chan serverMessage, 64)
	fastDone := make(chan struct{})
	go func() {
		defer close(fastDone)
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, payload, err := fast.Read(ctx)
			cancel()
			if err != nil {
				return
			}
			var message serverMessage
			if json.Unmarshal(payload, &message) == nil {
				fastMessages <- message
			}
		}
	}()

	for index := 0; index < 16; index++ {
		intro := fmt.Sprintf("update-%d", index)
		if _, ok := service.Update(serverTreeWithLargeBody(), &intro); !ok {
			t.Fatal("live update unexpectedly failed")
		}
		started := time.Now()
		server.PublishUpdate()
		// The race detector makes the deliberately large JSON fixture's local
		// marshal cost substantially slower on loaded CI hosts. A blocked socket
		// write would take the connection write timeout, so this ceiling still
		// proves broadcast is queue-bound without conflating instrumentation cost.
		if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
			t.Fatalf("broadcast %d waited %s for a slow client", index, elapsed)
		}
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case message := <-fastMessages:
			if message.Type == MessageTerminalUpdate {
				return
			}
		case <-deadline:
			t.Fatal("fast client did not receive an update while a slow client was isolated")
		}
	}
}

func TestGracefulShutdownClosesClientsAndListenerIdempotently(t *testing.T) {
	service := live.New(nil, nil)
	service.Set("terminal-1", "Overseer", serverTree(), 0, "WELCOME")
	server := startTestServer(t, service, nil, 4)
	info := server.Info()
	connection := dialPlayer(t, info.URL)
	readMessage(t, connection)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("second Stop() error = %v", err)
	}
	if _, _, err := connection.Read(ctx); err == nil {
		t.Fatal("client remained open after graceful shutdown")
	}
	if _, response, err := websocket.Dial(ctx, websocketURL(info.URL), nil); err == nil {
		t.Fatal("listener accepted a connection after shutdown")
	} else if response != nil {
		response.Body.Close()
	}
}

func TestServerRejectsPortAlreadyOwnedOnConfiguredIPv4Interface(t *testing.T) {
	holder, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	port := holder.Addr().(*net.TCPAddr).Port
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")},
	}
	server, err := NewServer(Config{
		Address: fmt.Sprintf("0.0.0.0:%d", port), Assets: fs.FS(assets), Live: live.New(nil, nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Start(context.Background()); err == nil {
		_ = server.Stop(context.Background())
		t.Fatal("Start() acquired an IPv6 wildcard while the configured IPv4 port was already owned")
	}
}

func TestPlayerPatternActionPublishesPublicCallback(t *testing.T) {
	service := live.New(&serverRandom{values: []int{0}}, serverWords{})
	service.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	states := make(chan *domain.PublicHackState, 1)
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")},
	}
	server, err := NewServer(Config{
		Address: "127.0.0.1:0", Assets: fs.FS(assets), Live: service,
		OnHackState: func(state *domain.PublicHackState) { states <- state },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})

	connection := dialPlayer(t, server.Info().URL)
	snapshot := readMessage(t, connection)
	if snapshot.Type != MessageTerminalLive || snapshot.Hack == nil || len(snapshot.Hack.Patterns) == 0 {
		t.Fatalf("initial message = %#v", snapshot)
	}
	patternID := snapshot.Hack.Patterns[0].ID
	writeJSON(t, connection, map[string]any{"type": MessageHackPattern, "patternId": patternID})
	if message := readMessage(t, connection); message.Type != MessageHackState {
		t.Fatalf("hack broadcast = %#v", message)
	}
	select {
	case state := <-states:
		if state == nil || state.Level != 1 || !publicPatternUsed(state, patternID) {
			t.Fatalf("hack callback = %#v", state)
		}
	case <-time.After(time.Second):
		t.Fatal("player hack action did not reach public callback")
	}
}

func TestRejectedPatternDoesNotBroadcastAndReconnectReceivesCurrentState(t *testing.T) {
	service := live.New(&serverRandom{values: []int{1, 0}}, serverWords{})
	service.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	server := startTestServer(t, service, nil, 8)

	first := dialPlayer(t, server.Info().URL)
	defer first.CloseNow()
	second := dialPlayer(t, server.Info().URL)
	initialFirst := readMessage(t, first)
	readMessage(t, second)
	if initialFirst.Hack == nil || len(initialFirst.Hack.Patterns) == 0 {
		t.Fatalf("initial pattern state = %#v", initialFirst)
	}
	patternID := initialFirst.Hack.Patterns[0].ID
	writeJSON(t, first, map[string]any{"type": MessageHackPattern, "patternId": patternID})
	accepted := readConvergedMessages(t, []*websocket.Conn{first, second}, MessageHackState)
	if !publicPatternUsed(accepted.Hack, patternID) {
		t.Fatalf("accepted state = %#v", accepted.Hack)
	}

	_ = second.Close(websocket.StatusNormalClosure, "reconnect")
	reconnected := dialPlayer(t, server.Info().URL)
	defer reconnected.CloseNow()
	snapshot := readMessage(t, reconnected)
	if snapshot.Type != MessageTerminalLive || !reflect.DeepEqual(snapshot.Hack, accepted.Hack) {
		t.Fatalf("reconnect hack state = %#v, want %#v", snapshot.Hack, accepted.Hack)
	}

	if _, ok := service.ForceHackSuccess(); !ok {
		t.Fatal("ForceHackSuccess() rejected active puzzle")
	}
	server.PublishHack()
	readMessage(t, first)
	readMessage(t, reconnected)
	writeJSON(t, first, map[string]any{"type": MessageHackPattern, "patternId": patternID})
	assertNoPlayerMessage(t, first)
	assertNoPlayerMessage(t, reconnected)
}

func TestStaleAndConcurrentDuplicatePatternsPublishExactlyOneCurrentTransition(t *testing.T) {
	service := live.New(&serverRandom{values: []int{1, 99}}, serverWords{})
	old := service.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	server := startTestServer(t, service, nil, 8)
	staleSender := dialPlayer(t, server.Info().URL)
	readMessage(t, staleSender)
	staleID := old.Hack.Patterns[0].ID

	current := service.Set("terminal-1", "Overseer", serverTree(), 1, "WELCOME")
	server.PublishLive()
	readMessage(t, staleSender)
	currentID := current.Hack.Patterns[0].ID
	if currentID == staleID {
		t.Fatal("fresh puzzle reused the stale opaque pattern identity")
	}

	writeJSON(t, staleSender, map[string]any{"type": MessageHackPattern, "patternId": staleID})
	assertNoPlayerMessage(t, staleSender)
	staleSender.CloseNow()

	first := dialPlayer(t, server.Info().URL)
	defer first.CloseNow()
	second := dialPlayer(t, server.Info().URL)
	defer second.CloseNow()
	readMessage(t, first)
	readMessage(t, second)

	writeJSON(t, first, map[string]any{"type": MessageHackPattern, "patternId": currentID})
	writeJSON(t, second, map[string]any{"type": MessageHackPattern, "patternId": currentID})
	accepted := readConvergedMessages(t, []*websocket.Conn{first, second}, MessageHackState)
	if !publicPatternUsed(accepted.Hack, currentID) {
		t.Fatalf("accepted concurrent state = %#v", accepted.Hack)
	}
	reconnected := dialPlayer(t, server.Info().URL)
	defer reconnected.CloseNow()
	snapshot := readMessage(t, reconnected)
	if snapshot.Type != MessageTerminalLive || !reflect.DeepEqual(snapshot.Hack, accepted.Hack) {
		t.Fatalf("reconnect state = %#v, want accepted canonical state %#v", snapshot.Hack, accepted.Hack)
	}
	assertNoPlayerMessage(t, first)
	assertNoPlayerMessage(t, second)
}

func assertNoPlayerMessage(t *testing.T, connection *websocket.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, _, err := connection.Read(ctx); err == nil {
		t.Fatal("unexpected player broadcast")
	}
}

func publicPatternUsed(state *domain.PublicHackState, patternID string) bool {
	if state == nil {
		return false
	}
	for _, pattern := range state.Patterns {
		if pattern.ID == patternID {
			return pattern.Used
		}
	}
	return false
}

type serverMessage struct {
	Type         string                  `json:"type"`
	BrowserToken string                  `json:"browserToken,omitempty"`
	State        *us1PlayerState         `json:"state,omitempty"`
	RequestID    string                  `json:"requestId,omitempty"`
	Accepted     bool                    `json:"accepted,omitempty"`
	Reason       domain.ActionReason     `json:"reason,omitempty"`
	Revision     uint64                  `json:"revision,omitempty"`
	TerminalID   string                  `json:"terminalId,omitempty"`
	TerminalName string                  `json:"terminalName,omitempty"`
	IntroText    string                  `json:"introText,omitempty"`
	Nav          *domain.NavState        `json:"nav,omitempty"`
	Hack         *domain.PublicHackState `json:"hack,omitempty"`
}

type us1PlayerState struct {
	Revision         uint64                     `json:"revision"`
	SessionID        domain.LogicalSessionID    `json:"sessionId"`
	FallbackName     string                     `json:"fallbackName"`
	Character        *domain.PlayerCharacter    `json:"character"`
	Role             domain.PlayerRole          `json:"role"`
	Phase            domain.PlayerPhase         `json:"phase"`
	BroadcastID      domain.BroadcastID         `json:"broadcastId"`
	ActiveTerminalID string                     `json:"activeTerminalId"`
	Roster           []domain.PlayerRosterEntry `json:"roster"`
}

func readConvergedMessages(t *testing.T, clients []*websocket.Conn, wantType string) serverMessage {
	t.Helper()
	var canonical serverMessage
	for index, connection := range clients {
		message := readMessage(t, connection)
		if message.Type != wantType {
			t.Fatalf("client %d message type = %q, want %q", index, message.Type, wantType)
		}
		if index == 0 {
			canonical = message
			continue
		}
		if !reflect.DeepEqual(message, canonical) {
			t.Fatalf("client %d diverged: got %#v, canonical %#v", index, message, canonical)
		}
	}
	return canonical
}

func firstCandidateID(state *domain.PublicHackState) string {
	if state == nil {
		return ""
	}
	for _, column := range state.Columns {
		for _, word := range column.Words {
			return word.ID
		}
	}
	return ""
}

func startTestServer(t *testing.T, service *live.Service, onCount func(int), queueSize int) *Server {
	t.Helper()
	assets := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>Player</title>")},
	}
	server, err := NewServer(Config{
		Address:       "127.0.0.1:0",
		Assets:        fs.FS(assets),
		Live:          service,
		QueueSize:     queueSize,
		OnClientCount: onCount,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = server.Stop(ctx)
	})
	return server
}

func dialPlayer(t *testing.T, rawURL string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	origin, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	connection, response, err := websocket.Dial(ctx, websocketURL(rawURL), &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{origin.Scheme + "://" + origin.Host}},
	})
	if err != nil {
		if response != nil {
			response.Body.Close()
		}
		t.Fatal(err)
	}
	connection.SetReadLimit(8 << 20)
	t.Cleanup(func() { connection.CloseNow() })
	return connection
}

func readMessage(t *testing.T, connection *websocket.Conn) serverMessage {
	t.Helper()
	message, _ := readMessageWithPayload(t, connection)
	return message
}

func readMessageWithPayload(t *testing.T, connection *websocket.Conn) (serverMessage, []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	messageType, payload, err := connection.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if messageType != websocket.MessageText {
		t.Fatalf("message type = %v, want text", messageType)
	}
	var message serverMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatalf("decode %s: %v", payload, err)
	}
	return message, payload
}

func writeJSON(t *testing.T, connection *websocket.Conn, value any) {
	t.Helper()
	if message, ok := value.(map[string]any); ok {
		switch message["type"] {
		case MessageNavAction, MessageHackGuess, MessageHackPattern:
			if _, exists := message["requestId"]; !exists {
				message["requestId"] = "legacy-request"
			}
			if _, exists := message["broadcastId"]; !exists {
				message["broadcastId"] = "legacy-broadcast"
			}
			if _, exists := message["terminalId"]; !exists {
				message["terminalId"] = "legacy-terminal"
			}
		}
	}
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeClientMessage(bytes.NewReader(payload)); err != nil {
		t.Fatalf("test attempted invalid player message %s: %v", payload, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := connection.Write(ctx, websocket.MessageText, payload); err != nil {
		t.Fatal(err)
	}
}

func websocketURL(rawURL string) string {
	return "ws" + rawURL[len("http"):]
}

type countRecorder struct {
	mu     sync.Mutex
	counts []int
}

func (recorder *countRecorder) Add(count int) {
	recorder.mu.Lock()
	recorder.counts = append(recorder.counts, count)
	recorder.mu.Unlock()
}

func (recorder *countRecorder) Last() int {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(recorder.counts) == 0 {
		return 0
	}
	return recorder.counts[len(recorder.counts)-1]
}

func waitForCount(t *testing.T, recorder *countRecorder, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if recorder.Last() == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("client count = %d, want %d", recorder.Last(), want)
}

func serverTree() domain.ContentNode {
	return domain.ContentNode{
		ID: "root", Type: domain.NodeFolder, Name: "ROOT",
		Children: []domain.ContentNode{{
			ID: "docs", Type: domain.NodeFolder, Name: "DOCS",
			Children: []domain.ContentNode{
				{ID: "report", Type: domain.NodeEntry, Name: "REPORT", Description: "Report"},
			},
		}},
	}
}

func serverTreeWithLargeBody() domain.ContentNode {
	tree := serverTree()
	tree.Children[0].Children = append(tree.Children[0].Children, domain.ContentNode{
		ID: "large", Type: domain.NodeEntry, Name: "LARGE", Description: string(make([]byte, 32*1024)),
	})
	return tree
}

type serverRandom struct {
	values []int
	index  int
}

func (random *serverRandom) Intn(limit int) int {
	if limit <= 0 || len(random.values) == 0 {
		return 0
	}
	value := random.values[random.index%len(random.values)]
	random.index++
	if value < 0 {
		value = -value
	}
	return value % limit
}

type serverWords struct{}

func (serverWords) PickWords(length, count int) []string {
	words := make([]string, count)
	for index := range words {
		word := fmt.Sprintf("%0*d", length, index)
		words[index] = word[len(word)-length:]
	}
	return words
}
