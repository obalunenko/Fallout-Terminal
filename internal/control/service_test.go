package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/hack"
	"github.com/obalunenko/Fallout-Terminal/internal/live"
	"github.com/obalunenko/Fallout-Terminal/internal/nav"
	"github.com/obalunenko/Fallout-Terminal/internal/testutil"
)

// These tests intentionally exercise the package-private transaction seam.
// Story commands build on this seam, while commit remains the single place
// that assigns revisions, detaches effects, and orders their publication.

func TestServiceUsesInjectedOpaqueIDSourceDeterministically(t *testing.T) {
	ids := &sequenceIDSource{values: []string{
		"opaque-session-id",
		"opaque-browser-token",
		"opaque-character-id",
	}}
	service := New(Config{IDs: ids})

	for index, want := range ids.values {
		if got := service.nextID(); got != want {
			t.Fatalf("nextID() call %d = %q, want injected opaque value %q", index+1, got, want)
		}
	}
}

func TestCommitAdvancesRevisionOnlyForAcceptedTransitions(t *testing.T) {
	var effects []Effect
	service := New(Config{Enqueue: func(effect Effect) {
		effects = append(effects, effect)
	}})

	first := service.commit(func(*domain.ProcessRuntime) transition {
		return transition{accepted: true, effects: []Effect{{Live: testLiveState("first")}}}
	})
	rejected := service.commit(func(*domain.ProcessRuntime) transition {
		return transition{accepted: false, effects: []Effect{{Live: testLiveState("rejected")}}}
	})
	second := service.commit(func(*domain.ProcessRuntime) transition {
		return transition{accepted: true, effects: []Effect{{Live: testLiveState("second")}}}
	})

	if !first.accepted || first.revision != 1 {
		t.Fatalf("first commit = %#v, want accepted revision 1", first)
	}
	if rejected.accepted || rejected.revision != 1 {
		t.Fatalf("rejected commit = %#v, want rejected at unchanged revision 1", rejected)
	}
	if !second.accepted || second.revision != 2 {
		t.Fatalf("second commit = %#v, want accepted revision 2", second)
	}

	wantRevisions := []uint64{1, 1, 2}
	gotRevisions := make([]uint64, 0, len(effects))
	for _, effect := range effects {
		gotRevisions = append(gotRevisions, effect.Revision)
	}
	if !reflect.DeepEqual(gotRevisions, wantRevisions) {
		t.Fatalf("effect revisions = %v, want %v", gotRevisions, wantRevisions)
	}
}

func TestCommitDetachesEffectsBeforeEnqueue(t *testing.T) {
	var enqueued Effect
	service := New(Config{Enqueue: func(effect Effect) {
		enqueued = effect
	}})
	produced := Effect{Live: testLiveState("canonical")}

	result := service.commit(func(*domain.ProcessRuntime) transition {
		return transition{accepted: true, effects: []Effect{produced}}
	})
	if !result.accepted || result.revision != 1 {
		t.Fatalf("commit() = %#v, want accepted revision 1", result)
	}

	produced.Live.TerminalName = "mutated producer"
	produced.Live.Tree.Children[0].Name = "mutated child"
	produced.Live.Nav.Path[0] = "mutated path"
	produced.Live.Hack.Log[0] = "mutated log"
	produced.Live.Hack.Columns[0].Words[0].ID = "mutated word"

	if enqueued.Revision != 1 {
		t.Fatalf("enqueued revision = %d, want 1", enqueued.Revision)
	}
	want := testLiveState("canonical")
	if !reflect.DeepEqual(enqueued.Live, want) {
		t.Fatalf("enqueued effect aliases its producer\ngot:  %#v\nwant: %#v", enqueued.Live, want)
	}
}

func TestCommitEnqueuesBeforeUnlocking(t *testing.T) {
	firstEnqueueStarted := make(chan struct{})
	releaseFirstEnqueue := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	var revisions []uint64

	service := New(Config{Enqueue: func(effect Effect) {
		mu.Lock()
		revisions = append(revisions, effect.Revision)
		mu.Unlock()

		if effect.Live != nil && effect.Live.TerminalID == "first" {
			once.Do(func() { close(firstEnqueueStarted) })
			<-releaseFirstEnqueue
		}
	}})

	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		service.commit(func(*domain.ProcessRuntime) transition {
			return transition{accepted: true, effects: []Effect{{Live: testLiveState("first")}}}
		})
	}()

	select {
	case <-firstEnqueueStarted:
	case <-time.After(time.Second):
		t.Fatal("first effect was not enqueued")
	}

	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		service.commit(func(*domain.ProcessRuntime) transition {
			return transition{accepted: true, effects: []Effect{{Live: testLiveState("second")}}}
		})
	}()

	select {
	case <-secondDone:
		t.Fatal("second transition committed while the first effect enqueue still held the transaction boundary")
	case <-time.After(50 * time.Millisecond):
	}

	close(releaseFirstEnqueue)
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("first transition did not finish after its enqueue was released")
	}
	select {
	case <-secondDone:
	case <-time.After(time.Second):
		t.Fatal("second transition did not finish after the first transaction unlocked")
	}

	mu.Lock()
	got := append([]uint64(nil), revisions...)
	mu.Unlock()
	if want := []uint64{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("enqueued revisions = %v, want %v", got, want)
	}
}

func TestRosterCreationAndFreshBroadcastSelection(t *testing.T) {
	service := newUS1Service()

	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatalf("AddCharacter(Mara) error = %v", err)
	}
	state, err = service.AddCharacter("Boone")
	if err != nil {
		t.Fatalf("AddCharacter(Boone) error = %v", err)
	}
	if len(state.Roster) != 2 || state.Roster[0].Name != "Mara" || state.Roster[1].Name != "Boone" {
		t.Fatalf("roster after creation = %#v, want Mara then Boone", state.Roster)
	}
	if state.Roster[0].ID == "" || state.Roster[1].ID == "" || state.Roster[0].ID == state.Roster[1].ID {
		t.Fatalf("roster IDs are not distinct opaque values: %#v", state.Roster)
	}

	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatalf("StartBroadcast() error = %v", err)
	}
	if state.Broadcast == nil || state.Broadcast.ID == "" || state.Broadcast.ControllerSessionID != nil {
		t.Fatalf("fresh broadcast = %#v, want new ID with no controller", state.Broadcast)
	}
	for _, character := range state.Roster {
		if character.ClaimedBySessionID != nil {
			t.Fatalf("fresh broadcast retained claim %#v", character)
		}
	}

	identity := service.CreateSession(domain.ConnectionID("connection-1"))
	if identity.SessionID == "" || identity.BrowserToken == "" || identity.State == nil || identity.State.FallbackName == "" {
		t.Fatalf("CreateSession() = %#v, want opaque identity, token, and fallback state", identity)
	}
	result := service.SelectCharacter(CharacterSelection{
		SessionID:   identity.SessionID,
		RequestID:   "select-1",
		BroadcastID: state.Broadcast.ID,
		CharacterID: state.Roster[0].ID,
	})
	if !result.Accepted {
		t.Fatalf("fresh SelectCharacter() = %#v, want accepted", result)
	}

	selected := service.Snapshot()
	if selected.Broadcast == nil || selected.Broadcast.ControllerSessionID == nil || *selected.Broadcast.ControllerSessionID != identity.SessionID {
		t.Fatalf("initial controller = %#v, want %q", selected.Broadcast, identity.SessionID)
	}
	assertExclusiveClaimInvariants(t, selected)
	if got := masterSession(t, selected, identity.SessionID); got.Role != domain.PlayerRoleActive || got.Character == nil || got.Character.ID != state.Roster[0].ID {
		t.Fatalf("selected session = %#v, want active Mara assignment", got)
	}
}

func TestConcurrentSameCharacterClaimHasExactlyOneWinnerAcross100Trials(t *testing.T) {
	for trial := 0; trial < 100; trial++ {
		service := newUS1Service()
		state, err := service.AddCharacter("Mara")
		if err != nil {
			t.Fatalf("trial %d AddCharacter() error = %v", trial, err)
		}
		state, err = service.StartBroadcast()
		if err != nil {
			t.Fatalf("trial %d StartBroadcast() error = %v", trial, err)
		}
		first := service.CreateSession(domain.ConnectionID(fmt.Sprintf("trial-%d-first", trial)))
		second := service.CreateSession(domain.ConnectionID(fmt.Sprintf("trial-%d-second", trial)))
		selection := func(identity SessionIdentity, requestID string) domain.ActionResult {
			return service.SelectCharacter(CharacterSelection{
				SessionID: identity.SessionID, RequestID: requestID,
				BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[0].ID,
			})
		}

		start := make(chan struct{})
		results := make(chan domain.ActionResult, 2)
		var workers sync.WaitGroup
		for index, candidate := range []SessionIdentity{first, second} {
			workers.Add(1)
			go func(index int, candidate SessionIdentity) {
				defer workers.Done()
				<-start
				results <- selection(candidate, fmt.Sprintf("trial-%d-request-%d", trial, index))
			}(index, candidate)
		}
		close(start)
		workers.Wait()
		close(results)

		accepted := 0
		for result := range results {
			if result.Accepted {
				accepted++
			}
		}
		if accepted != 1 {
			t.Fatalf("trial %d accepted claims = %d, want exactly 1", trial, accepted)
		}
		snapshot := service.Snapshot()
		assertExclusiveClaimInvariants(t, snapshot)
		if claimedRosterCount(snapshot) != 1 || activeSessionCount(snapshot) != 1 {
			t.Fatalf("trial %d state = %#v, want one claim and one controller", trial, snapshot)
		}
	}
}

func TestConcurrentDifferentFirstAssignmentsChooseExactlyOneControllerAcross100Trials(t *testing.T) {
	for trial := 0; trial < 100; trial++ {
		service := newUS1Service()
		state, err := service.AddCharacter("Mara")
		if err != nil {
			t.Fatalf("trial %d AddCharacter(Mara) error = %v", trial, err)
		}
		state, err = service.AddCharacter("Boone")
		if err != nil {
			t.Fatalf("trial %d AddCharacter(Boone) error = %v", trial, err)
		}
		state, err = service.StartBroadcast()
		if err != nil {
			t.Fatalf("trial %d StartBroadcast() error = %v", trial, err)
		}
		first := service.CreateSession(domain.ConnectionID(fmt.Sprintf("trial-%d-first", trial)))
		second := service.CreateSession(domain.ConnectionID(fmt.Sprintf("trial-%d-second", trial)))

		start := make(chan struct{})
		results := make(chan domain.ActionResult, 2)
		var workers sync.WaitGroup
		for index, candidate := range []struct {
			identity    SessionIdentity
			characterID domain.CharacterID
		}{{first, state.Roster[0].ID}, {second, state.Roster[1].ID}} {
			workers.Add(1)
			go func(index int, candidate SessionIdentity, characterID domain.CharacterID) {
				defer workers.Done()
				<-start
				results <- service.SelectCharacter(CharacterSelection{
					SessionID: candidate.SessionID, RequestID: fmt.Sprintf("trial-%d-request-%d", trial, index),
					BroadcastID: state.Broadcast.ID, CharacterID: characterID,
				})
			}(index, candidate.identity, candidate.characterID)
		}
		close(start)
		workers.Wait()
		close(results)

		for result := range results {
			if !result.Accepted {
				t.Fatalf("trial %d different-character selection rejected: %#v", trial, result)
			}
		}
		snapshot := service.Snapshot()
		assertExclusiveClaimInvariants(t, snapshot)
		if claimedRosterCount(snapshot) != 2 || activeSessionCount(snapshot) != 1 || observerSessionCount(snapshot) != 1 {
			t.Fatalf("trial %d roles/claims = %#v, want two claims with one active and one observer", trial, snapshot)
		}
	}
}

func TestSessionCannotClaimTwoCharactersAndCharacterCannotHaveTwoSessions(t *testing.T) {
	service := newUS1Service()
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.AddCharacter("Boone")
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	first := service.CreateSession("connection-first")
	second := service.CreateSession("connection-second")

	if result := service.SelectCharacter(CharacterSelection{
		SessionID: first.SessionID, RequestID: "first-mara",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[0].ID,
	}); !result.Accepted {
		t.Fatalf("first claim = %#v, want accepted", result)
	}
	if result := service.SelectCharacter(CharacterSelection{
		SessionID: first.SessionID, RequestID: "first-boone",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[1].ID,
	}); result.Accepted {
		t.Fatalf("same session second claim = %#v, want rejected", result)
	}
	if result := service.SelectCharacter(CharacterSelection{
		SessionID: second.SessionID, RequestID: "second-mara",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[0].ID,
	}); result.Accepted {
		t.Fatalf("same character second session claim = %#v, want rejected", result)
	}

	snapshot := service.Snapshot()
	assertExclusiveClaimInvariants(t, snapshot)
	if claimedRosterCount(snapshot) != 1 || masterSession(t, snapshot, first.SessionID).Character.ID != state.Roster[0].ID || masterSession(t, snapshot, second.SessionID).Character != nil {
		t.Fatalf("rejected claims changed assignments: %#v", snapshot)
	}
}

func TestPlayerActionAuthorizationRejectsWithoutTerminalMutationOrRandomness(t *testing.T) {
	tests := []struct {
		name       string
		connection func(us2Fixture) domain.ConnectionID
		mutate     func(*testing.T, us2Fixture)
		terminalID string
		wantReason domain.ActionReason
	}{
		{
			name: "observer", connection: func(fixture us2Fixture) domain.ConnectionID { return fixture.observerConnection },
			wantReason: domain.ActionReasonNotController,
		},
		{
			name: "unassigned", connection: func(fixture us2Fixture) domain.ConnectionID { return fixture.unassignedConnection },
			wantReason: domain.ActionReasonUnassigned,
		},
		{
			name: "unknown", connection: func(us2Fixture) domain.ConnectionID { return "connection-unknown" },
			wantReason: domain.ActionReasonInvalidSession,
		},
		{
			name: "stale terminal", connection: func(fixture us2Fixture) domain.ConnectionID { return fixture.controllerConnection },
			terminalID: "terminal-stale", wantReason: domain.ActionReasonStaleTerminal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runtime := &recordingTerminalRuntime{}
			fixture := newUS2Fixture(t, runtime)
			if test.mutate != nil {
				test.mutate(t, fixture)
			}
			terminalID := test.terminalID
			if terminalID == "" {
				terminalID = fixture.terminalID
			}
			requestID := "reject-" + test.name
			beforeTerminal := canonicalTerminalBytes(t, fixture.service, fixture.terminalID)
			beforeRevision := fixture.service.Revision()
			beforeCalls := runtime.Calls()
			beforeRandom := runtime.RandomCalls()

			fixture.service.DispatchPlayerAction(test.connection(fixture), domain.RuntimeCommand{
				RequestID: requestID, BroadcastID: fixture.broadcastID, TerminalID: terminalID,
				Kind: domain.RuntimeCommandHackPattern, PatternID: "opaque-current-pattern",
			})

			result := actionResultForRequest(t, fixture.effects, requestID)
			if result.Accepted || result.Reason != test.wantReason || result.Revision != beforeRevision {
				t.Fatalf("DispatchPlayerAction() result = %#v, want rejected %q at revision %d", result, test.wantReason, beforeRevision)
			}
			if got := canonicalTerminalBytes(t, fixture.service, fixture.terminalID); !reflect.DeepEqual(got, beforeTerminal) {
				t.Fatalf("rejected action changed canonical terminal bytes\nbefore: %s\nafter:  %s", beforeTerminal, got)
			}
			if fixture.service.Revision() != beforeRevision || runtime.Calls() != beforeCalls || runtime.RandomCalls() != beforeRandom {
				t.Fatalf("rejected action changed revision/runtime/RNG: revision %d->%d calls %d->%d RNG %d->%d",
					beforeRevision, fixture.service.Revision(), beforeCalls, runtime.Calls(), beforeRandom, runtime.RandomCalls())
			}
		})
	}
}

func TestControllerActionIsAuthorizedAndObserverStateRemainsCanonical(t *testing.T) {
	runtime := &recordingTerminalRuntime{}
	fixture := newUS2Fixture(t, runtime)
	beforeRevision := fixture.service.Revision()

	fixture.service.DispatchPlayerAction(fixture.controllerConnection, domain.RuntimeCommand{
		RequestID: "controller-nav", BroadcastID: fixture.broadcastID, TerminalID: fixture.terminalID,
		Kind: domain.RuntimeCommandNavAction, Action: "enter", NodeID: "docs",
	})

	result := actionResultForRequest(t, fixture.effects, "controller-nav")
	if !result.Accepted || result.Reason != domain.ActionReasonAccepted || result.Revision != beforeRevision+1 {
		t.Fatalf("controller action result = %#v, want accepted revision %d", result, beforeRevision+1)
	}
	terminal := canonicalTerminal(t, fixture.service, fixture.terminalID)
	if !reflect.DeepEqual(terminal.Nav.Path, []string{"root", "docs"}) || runtime.Calls() != 1 {
		t.Fatalf("controller navigation = %#v, runtime calls = %d", terminal.Nav, runtime.Calls())
	}
	observer, ok := fixture.service.PlayerSnapshot(fixture.observerSession)
	if !ok || observer.Role != domain.PlayerRoleObserver {
		t.Fatalf("observer state changed after controller action: %#v, ok=%t", observer, ok)
	}
}

func TestDuplicatePlayerActionFingerprintNeverMutatesTwice(t *testing.T) {
	runtime := &recordingTerminalRuntime{}
	fixture := newUS2Fixture(t, runtime)
	command := domain.RuntimeCommand{
		RequestID: "duplicate-nav", BroadcastID: fixture.broadcastID, TerminalID: fixture.terminalID,
		Kind: domain.RuntimeCommandNavAction, Action: "enter", NodeID: "docs",
	}

	fixture.service.DispatchPlayerAction(fixture.controllerConnection, command)
	first := actionResultForRequest(t, fixture.effects, command.RequestID)
	if !first.Accepted {
		t.Fatalf("first action = %#v, want accepted", first)
	}
	afterFirst := canonicalTerminalBytes(t, fixture.service, fixture.terminalID)
	afterFirstRevision := fixture.service.Revision()
	afterFirstCalls := runtime.Calls()

	fixture.service.DispatchPlayerAction(fixture.controllerConnection, command)
	replayed := actionResultForRequest(t, fixture.effects, command.RequestID)
	if !reflect.DeepEqual(replayed, first) {
		t.Fatalf("exact duplicate result = %#v, want cached %#v", replayed, first)
	}
	if runtime.Calls() != afterFirstCalls || fixture.service.Revision() != afterFirstRevision || !reflect.DeepEqual(canonicalTerminalBytes(t, fixture.service, fixture.terminalID), afterFirst) {
		t.Fatal("exact duplicate repeated canonical mutation")
	}

	different := command
	different.NodeID = "other-node"
	fixture.service.DispatchPlayerAction(fixture.controllerConnection, different)
	conflict := actionResultForRequest(t, fixture.effects, command.RequestID)
	if conflict.Accepted || conflict.Reason != domain.ActionReasonDuplicate || conflict.Revision != afterFirstRevision {
		t.Fatalf("different duplicate fingerprint = %#v, want duplicate rejection", conflict)
	}
	if runtime.Calls() != afterFirstCalls || fixture.service.Revision() != afterFirstRevision || !reflect.DeepEqual(canonicalTerminalBytes(t, fixture.service, fixture.terminalID), afterFirst) {
		t.Fatal("different duplicate fingerprint changed canonical state")
	}
}

func TestPlayerActionAndControllerReassignmentFollowCoordinatorOrder(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	runtime := &recordingTerminalRuntime{started: started, release: release}
	fixture := newUS2Fixture(t, runtime)

	actionDone := make(chan struct{})
	go func() {
		defer close(actionDone)
		fixture.service.DispatchPlayerAction(fixture.controllerConnection, domain.RuntimeCommand{
			RequestID: "before-reassign", BroadcastID: fixture.broadcastID, TerminalID: fixture.terminalID,
			Kind: domain.RuntimeCommandNavAction, Action: "enter", NodeID: "docs",
		})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("controller action did not enter runtime boundary")
	}

	reassigned := make(chan struct{})
	go func() {
		defer close(reassigned)
		setControllerForTest(fixture.service, fixture.observerSession)
	}()
	select {
	case <-reassigned:
		t.Fatal("controller reassignment overtook an action already inside the coordinator transaction")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	select {
	case <-actionDone:
	case <-time.After(time.Second):
		t.Fatal("ordered controller action did not finish")
	}
	select {
	case <-reassigned:
	case <-time.After(time.Second):
		t.Fatal("controller reassignment did not follow the completed action")
	}
	if result := actionResultForRequest(t, fixture.effects, "before-reassign"); !result.Accepted {
		t.Fatalf("action ordered before reassignment = %#v, want accepted", result)
	}

	before := canonicalTerminalBytes(t, fixture.service, fixture.terminalID)
	beforeCalls := runtime.Calls()
	beforeRevision := fixture.service.Revision()
	fixture.service.DispatchPlayerAction(fixture.controllerConnection, domain.RuntimeCommand{
		RequestID: "after-reassign", BroadcastID: fixture.broadcastID, TerminalID: fixture.terminalID,
		Kind: domain.RuntimeCommandNavAction, Action: "back",
	})
	result := actionResultForRequest(t, fixture.effects, "after-reassign")
	if result.Accepted || result.Reason != domain.ActionReasonNotController || result.Revision != beforeRevision {
		t.Fatalf("former controller action after reassignment = %#v, want not-controller rejection", result)
	}
	if runtime.Calls() != beforeCalls || !reflect.DeepEqual(canonicalTerminalBytes(t, fixture.service, fixture.terminalID), before) {
		t.Fatal("action ordered after reassignment mutated canonical terminal")
	}
}

func TestSetActiveControllerRequiresConnectedAssignedObserverAndPreservesCanonicalRuntime(t *testing.T) {
	runtime := &recordingTerminalRuntime{}
	fixture := newUS2Fixture(t, runtime)
	beforeState := fixture.service.Snapshot()
	beforeTerminal := canonicalTerminalBytes(t, fixture.service, fixture.terminalID)
	beforeAssignments := masterAssignments(beforeState)
	beforeRevision := fixture.service.Revision()

	state, err := fixture.service.SetActiveController(fixture.observerSession)
	if err != nil {
		t.Fatalf("SetActiveController(eligible observer) error = %v", err)
	}
	if state == nil || state.Revision != beforeRevision+1 {
		t.Fatalf("SetActiveController() state = %#v, want revision %d", state, beforeRevision+1)
	}
	assertExactlyOneController(t, state, fixture.observerSession)
	if got := masterSession(t, state, fixture.controllerSession).Role; got != domain.PlayerRoleObserver {
		t.Fatalf("former controller role = %q, want observer", got)
	}
	if !reflect.DeepEqual(masterAssignments(state), beforeAssignments) {
		t.Fatalf("reassignment changed character assignments\nbefore: %#v\nafter:  %#v", beforeAssignments, masterAssignments(state))
	}
	if got := canonicalTerminalBytes(t, fixture.service, fixture.terminalID); !reflect.DeepEqual(got, beforeTerminal) {
		t.Fatalf("reassignment changed terminal/private puzzle bytes\nbefore: %s\nafter:  %s", beforeTerminal, got)
	}
	if runtime.Calls() != 0 || runtime.RandomCalls() != 0 {
		t.Fatalf("reassignment entered gameplay runtime: calls=%d RNG=%d", runtime.Calls(), runtime.RandomCalls())
	}
}

func TestSetActiveControllerRejectsEveryIneligibleTargetWithoutMutation(t *testing.T) {
	t.Run("unknown logical session", func(t *testing.T) {
		fixture := newUS2Fixture(t, &recordingTerminalRuntime{})
		assertControllerReassignmentRejected(t, fixture.service, "unknown-session", fixture.terminalID)
	})

	t.Run("unassigned logical session", func(t *testing.T) {
		fixture := newUS2Fixture(t, &recordingTerminalRuntime{})
		assertControllerReassignmentRejected(t, fixture.service, fixture.unassignedSession, fixture.terminalID)
	})

	t.Run("disconnected assigned observer", func(t *testing.T) {
		fixture := newUS2Fixture(t, &recordingTerminalRuntime{})
		fixture.service.DetachConnection(fixture.observerConnection)
		assertControllerReassignmentRejected(t, fixture.service, fixture.observerSession, fixture.terminalID)
	})

	t.Run("no current broadcast", func(t *testing.T) {
		service := New(Config{IDs: &counterIDSource{}})
		session := service.CreateSession("no-broadcast-connection")
		assertControllerReassignmentRejected(t, service, session.SessionID, "")
	})
}

func TestActionAndSetActiveControllerHaveOneCoordinatorOrderAcross100Interleavings(t *testing.T) {
	for trial := 0; trial < 100; trial++ {
		t.Run(fmt.Sprintf("trial-%03d", trial), func(t *testing.T) {
			actionFirst := trial%2 == 0
			runtime := &recordingTerminalRuntime{}
			if actionFirst {
				runtime.started = make(chan struct{})
				runtime.release = make(chan struct{})
			}
			fixture := newUS2Fixture(t, runtime)
			beforeState := fixture.service.Snapshot()
			beforeAssignments := masterAssignments(beforeState)
			beforeTerminal := canonicalTerminalBytes(t, fixture.service, fixture.terminalID)
			requestID := domain.RequestID(fmt.Sprintf("reassign-race-%03d", trial))
			command := domain.RuntimeCommand{
				RequestID: requestID, BroadcastID: fixture.broadcastID, TerminalID: fixture.terminalID,
				Kind: domain.RuntimeCommandNavAction, Action: "enter", NodeID: "docs",
			}

			type reassignmentResult struct {
				state *domain.MasterCoordinationState
				err   error
			}
			reassigned := make(chan reassignmentResult, 1)
			actionDone := make(chan struct{})
			if actionFirst {
				go func() {
					fixture.service.DispatchPlayerAction(fixture.controllerConnection, command)
					close(actionDone)
				}()
				waitForSignal(t, runtime.started, "former-controller action to enter runtime")
				go func() {
					state, err := fixture.service.SetActiveController(fixture.observerSession)
					reassigned <- reassignmentResult{state: state, err: err}
				}()
				close(runtime.release)
			} else {
				gateStarted := make(chan struct{})
				gateRelease := make(chan struct{})
				originalEnqueue := fixture.service.enqueue
				var gateOnce sync.Once
				fixture.service.enqueue = func(effect Effect) {
					if effect.Master != nil && effect.Master.Broadcast != nil && effect.Master.Broadcast.ControllerSessionID != nil && *effect.Master.Broadcast.ControllerSessionID == fixture.observerSession {
						gateOnce.Do(func() {
							close(gateStarted)
							<-gateRelease
						})
					}
					originalEnqueue(effect)
				}
				go func() {
					state, err := fixture.service.SetActiveController(fixture.observerSession)
					reassigned <- reassignmentResult{state: state, err: err}
				}()
				waitForSignal(t, gateStarted, "controller reassignment to enter ordered publication")
				go func() {
					fixture.service.DispatchPlayerAction(fixture.controllerConnection, command)
					close(actionDone)
				}()
				close(gateRelease)
			}

			waitForSignal(t, actionDone, "interleaved former-controller action")
			var reassignment reassignmentResult
			select {
			case reassignment = <-reassigned:
			case <-time.After(time.Second):
				t.Fatal("interleaved controller reassignment did not finish")
			}
			if reassignment.err != nil || reassignment.state == nil {
				t.Fatalf("SetActiveController() = state %#v error %v", reassignment.state, reassignment.err)
			}
			assertExactlyOneController(t, reassignment.state, fixture.observerSession)
			if !reflect.DeepEqual(masterAssignments(reassignment.state), beforeAssignments) {
				t.Fatalf("trial %d changed assignments: got %#v want %#v", trial, masterAssignments(reassignment.state), beforeAssignments)
			}

			result := actionResultForRequest(t, fixture.effects, string(requestID))
			if actionFirst {
				if !result.Accepted || result.Reason != domain.ActionReasonAccepted || runtime.Calls() != 1 {
					t.Fatalf("action-before-reassignment result = %#v runtime calls=%d", result, runtime.Calls())
				}
				if got := canonicalTerminal(t, fixture.service, fixture.terminalID).Nav.Path; !reflect.DeepEqual(got, []string{"root", "docs"}) {
					t.Fatalf("accepted ordered action navigation = %#v", got)
				}
			} else {
				if result.Accepted || result.Reason != domain.ActionReasonNotController || runtime.Calls() != 0 {
					t.Fatalf("reassignment-before-action result = %#v runtime calls=%d", result, runtime.Calls())
				}
				if got := canonicalTerminalBytes(t, fixture.service, fixture.terminalID); !reflect.DeepEqual(got, beforeTerminal) {
					t.Fatalf("rejected former-controller action changed terminal\nbefore: %s\nafter:  %s", beforeTerminal, got)
				}
			}

			beforeRejected := canonicalTerminalBytes(t, fixture.service, fixture.terminalID)
			beforeRejectedCalls := runtime.Calls()
			beforeRejectedRevision := fixture.service.Revision()
			fixture.service.DispatchPlayerAction(fixture.controllerConnection, domain.RuntimeCommand{
				RequestID: domain.RequestID(fmt.Sprintf("former-controller-%03d", trial)), BroadcastID: fixture.broadcastID,
				TerminalID: fixture.terminalID, Kind: domain.RuntimeCommandNavAction, Action: "back",
			})
			formerResult := actionResultForRequest(t, fixture.effects, fmt.Sprintf("former-controller-%03d", trial))
			if formerResult.Accepted || formerResult.Reason != domain.ActionReasonNotController || formerResult.Revision != beforeRejectedRevision {
				t.Fatalf("former controller retry = %#v, want not-controller at revision %d", formerResult, beforeRejectedRevision)
			}
			if runtime.Calls() != beforeRejectedCalls || !reflect.DeepEqual(canonicalTerminalBytes(t, fixture.service, fixture.terminalID), beforeRejected) {
				t.Fatal("former-controller rejection changed canonical terminal/private puzzle")
			}
		})
	}
}

func assertControllerReassignmentRejected(t *testing.T, service *Service, sessionID domain.LogicalSessionID, terminalID string) {
	t.Helper()
	before := service.Snapshot()
	beforeRevision := service.Revision()
	var beforeTerminal []byte
	if terminalID != "" {
		beforeTerminal = canonicalTerminalBytes(t, service, terminalID)
	}
	state, err := service.SetActiveController(sessionID)
	if err == nil {
		t.Fatalf("SetActiveController(%q) unexpectedly succeeded with %#v", sessionID, state)
	}
	if !reflect.DeepEqual(state, before) || !reflect.DeepEqual(service.Snapshot(), before) || service.Revision() != beforeRevision {
		t.Fatalf("rejected reassignment changed authoritative state\nbefore: %#v\nresult: %#v\nafter:  %#v", before, state, service.Snapshot())
	}
	if terminalID != "" && !reflect.DeepEqual(canonicalTerminalBytes(t, service, terminalID), beforeTerminal) {
		t.Fatal("rejected reassignment changed terminal/private puzzle")
	}
}

func assertExactlyOneController(t *testing.T, state *domain.MasterCoordinationState, want domain.LogicalSessionID) {
	t.Helper()
	if state == nil || state.Broadcast == nil || state.Broadcast.ControllerSessionID == nil || *state.Broadcast.ControllerSessionID != want {
		t.Fatalf("controller state = %#v, want %q", state, want)
	}
	active := 0
	for _, session := range state.Sessions {
		if session.Role == domain.PlayerRoleActive {
			active++
			if session.ID != want {
				t.Fatalf("unexpected active session %q, want %q", session.ID, want)
			}
		}
	}
	if active != 1 {
		t.Fatalf("active controller count = %d, want exactly 1 in %#v", active, state.Sessions)
	}
}

func masterAssignments(state *domain.MasterCoordinationState) map[domain.LogicalSessionID]domain.CharacterID {
	assignments := make(map[domain.LogicalSessionID]domain.CharacterID)
	if state == nil {
		return assignments
	}
	for _, session := range state.Sessions {
		if session.Character != nil {
			assignments[session.ID] = session.Character.ID
		}
	}
	return assignments
}

func waitForSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func TestAcceptedPlayerActionsPreserveNavigationAndHackingOutcomes(t *testing.T) {
	runtime := &recordingTerminalRuntime{}
	fixture := newUS2Fixture(t, runtime)
	beforeNav := canonicalTerminal(t, fixture.service, fixture.terminalID)
	wantNav := nav.ApplyAction(beforeNav.Nav, beforeNav.Tree, "enter", "docs")

	fixture.service.DispatchPlayerAction(fixture.controllerConnection, domain.RuntimeCommand{
		RequestID: "outcome-nav", BroadcastID: fixture.broadcastID, TerminalID: fixture.terminalID,
		Kind: domain.RuntimeCommandNavAction, Action: "enter", NodeID: "docs",
	})
	if result := actionResultForRequest(t, fixture.effects, "outcome-nav"); !result.Accepted {
		t.Fatalf("navigation result = %#v, want accepted", result)
	}
	if got := canonicalTerminal(t, fixture.service, fixture.terminalID).Nav; !reflect.DeepEqual(got, wantNav) {
		t.Fatalf("coordinated navigation = %#v, want unchanged gameplay result %#v", got, wantNav)
	}

	beforeHack := cloneHackState(canonicalTerminal(t, fixture.service, fixture.terminalID).Hack)
	wantHack := cloneHackState(beforeHack)
	hack.ApplyGuess(wantHack, "candidate-wrong")
	fixture.service.DispatchPlayerAction(fixture.controllerConnection, domain.RuntimeCommand{
		RequestID: "outcome-guess", BroadcastID: fixture.broadcastID, TerminalID: fixture.terminalID,
		Kind: domain.RuntimeCommandHackGuess, TargetID: "candidate-wrong",
	})
	if result := actionResultForRequest(t, fixture.effects, "outcome-guess"); !result.Accepted {
		t.Fatalf("hacking result = %#v, want accepted", result)
	}
	if got := canonicalTerminal(t, fixture.service, fixture.terminalID).Hack; !reflect.DeepEqual(got, wantHack) {
		t.Fatalf("coordinated hacking outcome changed\ngot:  %#v\nwant: %#v", got, wantHack)
	}
}

func TestAttachConnectionAbsentAndUnknownTokensIssueUniqueReplacementIdentities(t *testing.T) {
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{IDs: &counterIDSource{}, Enqueue: effects.Enqueue})

	firstToken, first := service.AttachConnection("connection-first", "")
	secondToken, second := service.AttachConnection("connection-second", "")
	unknownToken, unknown := service.AttachConnection("connection-unknown", "client-stale-token")
	repeatedUnknownToken, repeatedUnknown := service.AttachConnection("connection-repeated-unknown", "client-stale-token")

	states := []*domain.PlayerState{first, second, unknown, repeatedUnknown}
	tokens := []domain.BrowserToken{firstToken, secondToken, unknownToken, repeatedUnknownToken}
	seenTokens := make(map[domain.BrowserToken]struct{}, len(tokens))
	seenSessions := make(map[domain.LogicalSessionID]struct{}, len(states))
	seenFallbacks := make(map[string]struct{}, len(states))
	for index, state := range states {
		if state == nil || state.SessionID == "" || state.FallbackName == "" {
			t.Fatalf("AttachConnection() state %d = %#v, want fresh logical identity", index, state)
		}
		if tokens[index] == "" || tokens[index] == "client-stale-token" {
			t.Fatalf("AttachConnection() token %d = %q, want server-issued replacement", index, tokens[index])
		}
		if _, duplicate := seenTokens[tokens[index]]; duplicate {
			t.Fatalf("replacement browser token %q was reused", tokens[index])
		}
		if _, duplicate := seenSessions[state.SessionID]; duplicate {
			t.Fatalf("unrecognized attachment reused logical session %q", state.SessionID)
		}
		if _, duplicate := seenFallbacks[state.FallbackName]; duplicate {
			t.Fatalf("fresh logical sessions reused fallback name %q", state.FallbackName)
		}
		seenTokens[tokens[index]] = struct{}{}
		seenSessions[state.SessionID] = struct{}{}
		seenFallbacks[state.FallbackName] = struct{}{}
	}
	if masterEffectCount(effects.Values()) != 4 {
		t.Fatalf("fresh first-connection master effects = %d, want 4", masterEffectCount(effects.Values()))
	}
}

func TestKnownTokenReusesStableSessionAcrossFirstAndLastPresenceTransitions(t *testing.T) {
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{IDs: &counterIDSource{}, Enqueue: effects.Enqueue})
	firstConnection := domain.ConnectionID("connection-first")
	token, initial := service.AttachConnection(firstConnection, "")
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	selection := service.SelectCharacter(CharacterSelection{
		ConnectionID: firstConnection, SessionID: initial.SessionID, RequestID: "select-mara",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[0].ID,
	})
	if !selection.Accepted {
		t.Fatalf("initial claim = %#v", selection)
	}
	claimed := service.Snapshot()
	fallback := masterSession(t, claimed, initial.SessionID).FallbackName

	service.DetachConnection(firstConnection)
	disconnected := service.Snapshot()
	disconnectedSession := masterSession(t, disconnected, initial.SessionID)
	if disconnectedSession.Connected || disconnectedSession.Character == nil || disconnectedSession.Role != domain.PlayerRoleActive {
		t.Fatalf("last detach changed stable claim/role: %#v", disconnectedSession)
	}
	baseline := effects.Calls()

	secondConnection := domain.ConnectionID("connection-second")
	returnedToken, reconnected := service.AttachConnection(secondConnection, token)
	if returnedToken != token || reconnected == nil || reconnected.SessionID != initial.SessionID || reconnected.FallbackName != fallback {
		t.Fatalf("known-token reconnect = token %q state %#v, want stable %q/%q", returnedToken, reconnected, token, initial.SessionID)
	}
	if reconnected.Character == nil || reconnected.Character.ID != state.Roster[0].ID || reconnected.Role != domain.PlayerRoleActive {
		t.Fatalf("known-token reconnect lost assignment/role: %#v", reconnected)
	}
	if got := masterEffectCount(effects.Values()[baseline:]); got != 1 {
		t.Fatalf("disconnected-to-connected transition emitted %d master effects, want 1", got)
	}

	baseline = effects.Calls()
	thirdConnection := domain.ConnectionID("connection-third")
	thirdToken, third := service.AttachConnection(thirdConnection, token)
	if thirdToken != token || third == nil || third.SessionID != initial.SessionID {
		t.Fatalf("second tab = token %q state %#v, want same logical session", thirdToken, third)
	}
	if got := masterEffectCount(effects.Values()[baseline:]); got != 0 {
		t.Fatalf("additional connection emitted %d presence effects, want 0", got)
	}

	baseline = effects.Calls()
	service.DetachConnection(secondConnection)
	if !masterSession(t, service.Snapshot(), initial.SessionID).Connected {
		t.Fatal("closing one of two connections marked the logical session disconnected")
	}
	if got := masterEffectCount(effects.Values()[baseline:]); got != 0 {
		t.Fatalf("non-final detach emitted %d presence effects, want 0", got)
	}

	baseline = effects.Calls()
	service.DetachConnection(thirdConnection)
	final := masterSession(t, service.Snapshot(), initial.SessionID)
	if final.Connected || final.Character == nil || final.Character.ID != state.Roster[0].ID || final.Role != domain.PlayerRoleActive {
		t.Fatalf("final detach changed stable logical-session state: %#v", final)
	}
	if got := masterEffectCount(effects.Values()[baseline:]); got != 1 {
		t.Fatalf("final detach emitted %d presence effects, want 1", got)
	}
}

func TestUnrecognizedSessionAttachmentDoesNotReleaseExistingClaim(t *testing.T) {
	service := New(Config{IDs: &counterIDSource{}})
	ownerConnection := domain.ConnectionID("connection-owner")
	ownerToken, owner := service.AttachConnection(ownerConnection, "")
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: ownerConnection, SessionID: owner.SessionID, RequestID: "select-owner",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[0].ID,
	}); !result.Accepted {
		t.Fatalf("owner selection = %#v", result)
	}
	service.DetachConnection(ownerConnection)

	replacementToken, newcomer := service.AttachConnection("connection-newcomer", "unknown-after-restart")
	if replacementToken == "" || replacementToken == "unknown-after-restart" || replacementToken == ownerToken || newcomer == nil || newcomer.SessionID == owner.SessionID {
		t.Fatalf("unrecognized attachment = token %q state %#v", replacementToken, newcomer)
	}
	snapshot := service.Snapshot()
	ownerState := masterSession(t, snapshot, owner.SessionID)
	newcomerState := masterSession(t, snapshot, newcomer.SessionID)
	if ownerState.Character == nil || ownerState.Character.ID != state.Roster[0].ID || ownerState.Role != domain.PlayerRoleActive {
		t.Fatalf("unrecognized session released or demoted existing owner: %#v", ownerState)
	}
	if newcomerState.Character != nil || newcomerState.Role != domain.PlayerRoleUnassigned {
		t.Fatalf("unrecognized newcomer inherited claim: %#v", newcomerState)
	}
	if snapshot.Roster[0].ClaimedBySessionID == nil || *snapshot.Roster[0].ClaimedBySessionID != owner.SessionID {
		t.Fatalf("roster claim moved after unrecognized attachment: %#v", snapshot.Roster)
	}
}

func TestFinalDetachRetainsObserverAndControllerClaimsWithoutPromotionOrRuntimeMutation(t *testing.T) {
	runtime := &recordingTerminalRuntime{}
	fixture := newUS2Fixture(t, runtime)
	terminalBefore := canonicalTerminalBytes(t, fixture.service, fixture.terminalID)
	assignmentsBefore := masterAssignments(fixture.service.Snapshot())

	baseline := fixture.effects.Calls()
	observerRevision := fixture.service.Revision()
	fixture.service.DetachConnection(fixture.observerConnection)
	observerDetached := fixture.service.Snapshot()
	if observerDetached.Revision != observerRevision+1 {
		t.Fatalf("observer final detach revision = %d, want %d", observerDetached.Revision, observerRevision+1)
	}
	observer := masterSession(t, observerDetached, fixture.observerSession)
	if observer.Connected || observer.Character == nil || observer.Role != domain.PlayerRoleObserver {
		t.Fatalf("detached observer lost stable claim/role: %#v", observer)
	}
	assertExactlyOneController(t, observerDetached, fixture.controllerSession)
	assertPresenceOnlyEffects(t, fixture.effects.Values()[baseline:], observerDetached.Revision)

	baseline = fixture.effects.Calls()
	controllerRevision := fixture.service.Revision()
	fixture.service.DetachConnection(fixture.controllerConnection)
	controllerDetached := fixture.service.Snapshot()
	if controllerDetached.Revision != controllerRevision+1 {
		t.Fatalf("controller final detach revision = %d, want %d", controllerDetached.Revision, controllerRevision+1)
	}
	controller := masterSession(t, controllerDetached, fixture.controllerSession)
	if controller.Connected || controller.Character == nil || controller.Role != domain.PlayerRoleActive {
		t.Fatalf("detached controller lost stable claim/designation: %#v", controller)
	}
	if got := masterSession(t, controllerDetached, fixture.observerSession).Role; got != domain.PlayerRoleObserver {
		t.Fatalf("observer role after controller disconnect = %q, want no automatic promotion", got)
	}
	assertExactlyOneController(t, controllerDetached, fixture.controllerSession)
	assertPresenceOnlyEffects(t, fixture.effects.Values()[baseline:], controllerDetached.Revision)
	if !reflect.DeepEqual(masterAssignments(controllerDetached), assignmentsBefore) {
		t.Fatalf("final detaches changed claims\nbefore: %#v\nafter:  %#v", assignmentsBefore, masterAssignments(controllerDetached))
	}
	if got := canonicalTerminalBytes(t, fixture.service, fixture.terminalID); !reflect.DeepEqual(got, terminalBefore) {
		t.Fatalf("final detaches changed terminal/private puzzle\nbefore: %s\nafter:  %s", terminalBefore, got)
	}
	if runtime.Calls() != 0 || runtime.RandomCalls() != 0 {
		t.Fatalf("presence transitions entered gameplay runtime: calls=%d RNG=%d", runtime.Calls(), runtime.RandomCalls())
	}
}

func TestControllerReconnectBeforeAndAfterReassignmentRestoresAuthoritativeRoleOnly(t *testing.T) {
	runtime := &recordingTerminalRuntime{}
	fixture := newUS2Fixture(t, runtime)
	terminalBefore := canonicalTerminalBytes(t, fixture.service, fixture.terminalID)
	assignmentsBefore := masterAssignments(fixture.service.Snapshot())

	fixture.service.DetachConnection(fixture.controllerConnection)
	reconnectConnection := domain.ConnectionID("connection-controller-reconnect")
	returnedToken, reconnected := fixture.service.AttachConnection(reconnectConnection, fixture.controllerToken)
	if returnedToken != fixture.controllerToken || reconnected == nil || reconnected.SessionID != fixture.controllerSession || reconnected.Character == nil || reconnected.Role != domain.PlayerRoleActive {
		t.Fatalf("unchanged controller reconnect = token %q state %#v", returnedToken, reconnected)
	}
	assertExactlyOneController(t, fixture.service.Snapshot(), fixture.controllerSession)

	fixture.service.DetachConnection(reconnectConnection)
	state, err := fixture.service.SetActiveController(fixture.observerSession)
	if err != nil {
		t.Fatalf("reassign while former controller disconnected: %v", err)
	}
	assertExactlyOneController(t, state, fixture.observerSession)

	secondReconnect := domain.ConnectionID("connection-former-controller-return")
	returnedToken, formerController := fixture.service.AttachConnection(secondReconnect, fixture.controllerToken)
	if returnedToken != fixture.controllerToken || formerController == nil || formerController.SessionID != fixture.controllerSession {
		t.Fatalf("former-controller reconnect identity = token %q state %#v", returnedToken, formerController)
	}
	if formerController.Character == nil || formerController.Role != domain.PlayerRoleObserver || formerController.Phase != domain.PlayerPhaseObserving {
		t.Fatalf("reassigned former-controller reconnect = %#v, want assigned observer", formerController)
	}
	final := fixture.service.Snapshot()
	assertExactlyOneController(t, final, fixture.observerSession)
	if !reflect.DeepEqual(masterAssignments(final), assignmentsBefore) {
		t.Fatalf("disconnect/reconnect/reassignment changed claims\nbefore: %#v\nafter:  %#v", assignmentsBefore, masterAssignments(final))
	}
	if got := canonicalTerminalBytes(t, fixture.service, fixture.terminalID); !reflect.DeepEqual(got, terminalBefore) {
		t.Fatalf("disconnect/reconnect/reassignment changed terminal/private puzzle\nbefore: %s\nafter:  %s", terminalBefore, got)
	}
	if runtime.Calls() != 0 || runtime.RandomCalls() != 0 {
		t.Fatalf("disconnect lifecycle entered gameplay runtime: calls=%d RNG=%d", runtime.Calls(), runtime.RandomCalls())
	}
}

func TestDirectTerminalActivationClearAndLateAssignmentPreserveBroadcastIdentity(t *testing.T) {
	actions := &recordingTerminalRuntime{}
	terminals := &recordingTerminalLifecycle{}
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{IDs: &counterIDSource{}, Enqueue: effects.Enqueue, Runtime: actions, Terminals: terminals})

	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	maraID := state.Roster[0].ID
	state, err = service.AddCharacter("Boone")
	if err != nil {
		t.Fatal(err)
	}
	booneID := state.Roster[1].ID
	state, err = service.AddCharacter("Arcade")
	if err != nil {
		t.Fatal(err)
	}
	arcadeID := state.Roster[2].ID
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	broadcastID := state.Broadcast.ID
	controller := service.CreateSession("switch-controller")
	observer := service.CreateSession("switch-observer")
	late := service.CreateSession("switch-late")
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: "switch-controller", SessionID: controller.SessionID, RequestID: "switch-select-controller",
		BroadcastID: broadcastID, CharacterID: maraID,
	}); !result.Accepted {
		t.Fatalf("controller selection = %#v", result)
	}
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: "switch-observer", SessionID: observer.SessionID, RequestID: "switch-select-observer",
		BroadcastID: broadcastID, CharacterID: booneID,
	}); !result.Accepted {
		t.Fatalf("observer selection = %#v", result)
	}
	identityBefore := service.Snapshot()
	assignmentsBefore := masterAssignments(identityBefore)

	terminalA := terminalTarget("terminal-a", "Overseer A")
	state, err = service.RequestTerminalActivation(terminalA)
	if err != nil {
		t.Fatalf("RequestTerminalActivation(A) error = %v", err)
	}
	assertActiveTerminalAndIdentity(t, state, "terminal-a", broadcastID, controller.SessionID, assignmentsBefore)
	controllerState, ok := service.PlayerSnapshot(controller.SessionID)
	if !ok || controllerState.ActiveTerminalID != "terminal-a" || controllerState.Phase != domain.PlayerPhaseControlling {
		t.Fatalf("controller after activation A = %#v, ok=%t", controllerState, ok)
	}
	terminalARuntime := canonicalTerminal(t, service, "terminal-a")

	terminalB := terminalTarget("terminal-b", "Overseer B")
	state, err = service.RequestTerminalActivation(terminalB)
	if err != nil {
		t.Fatalf("direct completed-puzzle activation B error = %v", err)
	}
	assertActiveTerminalAndIdentity(t, state, "terminal-b", broadcastID, controller.SessionID, assignmentsBefore)
	slots := canonicalTerminalSlots(t, service)
	if len(slots) != 2 || slots["terminal-a"] == nil || slots["terminal-b"] == nil {
		t.Fatalf("direct switch runtime slots = %#v, want owned A and B checkpoints", slots)
	}
	if slots["terminal-a"].Lifecycle != domain.TerminalLifecycleSuspended || slots["terminal-b"].Lifecycle != domain.TerminalLifecycleActive {
		t.Fatalf("direct switch lifecycle = A %q B %q", slots["terminal-a"].Lifecycle, slots["terminal-b"].Lifecycle)
	}
	suspendedA := canonicalTerminal(t, service, "terminal-a")
	terminalARuntime.Lifecycle = ""
	suspendedA.Lifecycle = ""
	if !reflect.DeepEqual(suspendedA, terminalARuntime) {
		t.Fatalf("switching away changed completed source gameplay runtime\nbefore: %#v\nafter:  %#v", terminalARuntime, suspendedA)
	}

	restoredA := terminalTarget("terminal-a", "Overseer A Updated")
	state, err = service.RequestTerminalActivation(restoredA)
	if err != nil {
		t.Fatalf("RequestTerminalActivation(restored A) error = %v", err)
	}
	assertActiveTerminalAndIdentity(t, state, "terminal-a", broadcastID, controller.SessionID, assignmentsBefore)
	if creates, updates, _ := terminals.Calls(); creates != 2 || updates != 1 {
		t.Fatalf("runtime lifecycle calls = create/update %d/%d, want 2/1", creates, updates)
	}
	restored := canonicalTerminal(t, service, "terminal-a")
	if restored.TerminalName != "Overseer A Updated" || restored.Hack == nil || !restored.Hack.Solved || restored.Hack.GenerationID != "generation-terminal-a" {
		t.Fatalf("restored checkpoint = %#v, want updated authored metadata and preserved completed puzzle", restored)
	}

	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: "switch-late", SessionID: late.SessionID, RequestID: "switch-select-late",
		BroadcastID: broadcastID, CharacterID: arcadeID,
	}); !result.Accepted {
		t.Fatalf("late assignment = %#v", result)
	}
	lateState, ok := service.PlayerSnapshot(late.SessionID)
	if !ok || lateState.Character == nil || lateState.ActiveTerminalID != "terminal-a" || lateState.Phase != domain.PlayerPhaseObserving {
		t.Fatalf("late assignee current-terminal snapshot = %#v, ok=%t", lateState, ok)
	}
	assignmentsWithLate := masterAssignments(service.Snapshot())

	state, err = service.RequestTerminalClear()
	if err != nil {
		t.Fatalf("direct completed-puzzle clear error = %v", err)
	}
	if state.Broadcast == nil || state.Broadcast.ActiveTerminalID != nil || state.Broadcast.ID != broadcastID {
		t.Fatalf("cleared terminal state = %#v, want active broadcast with nil terminal", state.Broadcast)
	}
	assertExactlyOneController(t, state, controller.SessionID)
	if !reflect.DeepEqual(masterAssignments(state), assignmentsWithLate) {
		t.Fatalf("terminal clear changed claims: got %#v want %#v", masterAssignments(state), assignmentsWithLate)
	}
	for _, sessionID := range []domain.LogicalSessionID{controller.SessionID, observer.SessionID, late.SessionID} {
		player, exists := service.PlayerSnapshot(sessionID)
		if !exists || player.Character == nil || player.ActiveTerminalID != "" || player.Phase != domain.PlayerPhaseWaiting {
			t.Fatalf("assigned session %q after terminal clear = %#v, exists=%t", sessionID, player, exists)
		}
	}
	for terminalID, slot := range canonicalTerminalSlots(t, service) {
		if slot.Lifecycle != domain.TerminalLifecycleSuspended {
			t.Fatalf("cleared runtime slot %q lifecycle = %q, want suspended", terminalID, slot.Lifecycle)
		}
	}
	if !hasClearEffectAtRevision(effects.Values(), state.Revision) {
		t.Fatalf("terminal clear emitted no revision-%d canonical clear effect", state.Revision)
	}
}

func TestInactiveAndClearedTerminalActionsAreRejectedWithoutTouchingRuntimeSlots(t *testing.T) {
	actions := &recordingTerminalRuntime{}
	terminals := &recordingTerminalLifecycle{}
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{IDs: &counterIDSource{}, Enqueue: effects.Enqueue, Runtime: actions, Terminals: terminals})
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	characterID := state.Roster[0].ID
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	connectionID := domain.ConnectionID("inactive-controller")
	controller := service.CreateSession(connectionID)
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: connectionID, SessionID: controller.SessionID, RequestID: "inactive-select",
		BroadcastID: state.Broadcast.ID, CharacterID: characterID,
	}); !result.Accepted {
		t.Fatalf("controller selection = %#v", result)
	}
	if _, err = service.RequestTerminalActivation(terminalTarget("terminal-a", "A")); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RequestTerminalActivation(terminalTarget("terminal-b", "B")); err != nil {
		t.Fatal(err)
	}

	assertRejectedTerminalAction := func(requestID string, terminalID string) {
		t.Helper()
		beforeSlots := canonicalTerminalSlotBytes(t, service)
		beforeRevision := service.Revision()
		beforeCalls := actions.Calls()
		service.DispatchPlayerAction(connectionID, domain.RuntimeCommand{
			RequestID: domain.RequestID(requestID), BroadcastID: state.Broadcast.ID, TerminalID: terminalID,
			Kind: domain.RuntimeCommandNavAction, Action: "enter", NodeID: "docs",
		})
		result := actionResultForRequest(t, effects, requestID)
		if result.Accepted || result.Reason != domain.ActionReasonStaleTerminal || result.Revision != beforeRevision {
			t.Fatalf("inactive action %q result = %#v, want stale-terminal revision %d", terminalID, result, beforeRevision)
		}
		if actions.Calls() != beforeCalls || !reflect.DeepEqual(canonicalTerminalSlotBytes(t, service), beforeSlots) {
			t.Fatalf("inactive action %q touched gameplay/runtime slots", terminalID)
		}
	}

	assertRejectedTerminalAction("inactive-a", "terminal-a")
	if _, err = service.RequestTerminalClear(); err != nil {
		t.Fatal(err)
	}
	assertRejectedTerminalAction("cleared-b", "terminal-b")
}

func TestUnfinishedTerminalSwitchPreserveKeepsSourceActionableAndRestoresExactCheckpoint(t *testing.T) {
	fixture := newUS8SwitchFixture(t)
	sourceBefore := canonicalTerminalBytes(t, fixture.service, "terminal-a")
	revisionBefore := fixture.service.Revision()

	state, err := fixture.service.RequestTerminalActivation(terminalTarget("terminal-b", "Terminal B"))
	if err != nil {
		t.Fatalf("unfinished switch request error = %v", err)
	}
	assertPendingSwitch(t, state, fixture.broadcastID, "terminal-a", "terminal-b")
	if state.Revision != revisionBefore+1 || state.PendingSwitch.SwitchID == "" || state.PendingSwitch.SwitchID == domain.SwitchID("terminal-a") || state.PendingSwitch.SwitchID == domain.SwitchID("terminal-b") {
		t.Fatalf("pending switch identity/revision = %#v, want opaque token at %d", state.PendingSwitch, revisionBefore+1)
	}
	switchID := state.PendingSwitch.SwitchID
	if got := canonicalTerminalBytes(t, fixture.service, "terminal-a"); !reflect.DeepEqual(got, sourceBefore) {
		t.Fatalf("decision-required request changed source checkpoint\nbefore: %s\nafter:  %s", sourceBefore, got)
	}
	if slots := canonicalTerminalSlots(t, fixture.service); len(slots) != 1 || slots["terminal-b"] != nil {
		t.Fatalf("decision-required request created target prematurely: %#v", slots)
	}

	fixture.service.DispatchPlayerAction(fixture.connectionID, domain.RuntimeCommand{
		RequestID: "pending-source-action", BroadcastID: fixture.broadcastID, TerminalID: "terminal-a",
		Kind: domain.RuntimeCommandNavAction, Action: "enter", NodeID: "docs",
	})
	if result := actionResultForRequest(t, fixture.effects, "pending-source-action"); !result.Accepted {
		t.Fatalf("source action while switch pending = %#v, want accepted", result)
	}
	sourceAfterAction := canonicalTerminalBytes(t, fixture.service, "terminal-a")
	if reflect.DeepEqual(sourceAfterAction, sourceBefore) {
		t.Fatal("accepted source action while pending did not update canonical checkpoint")
	}

	state, err = fixture.service.ResolveTerminalSwitch(switchID, domain.TerminalSwitchPreserve)
	if err != nil {
		t.Fatalf("ResolveTerminalSwitch(preserve) error = %v", err)
	}
	if state.PendingSwitch != nil || state.Broadcast.ActiveTerminalID == nil || *state.Broadcast.ActiveTerminalID != "terminal-b" {
		t.Fatalf("preserve resolution state = %#v", state)
	}
	slots := canonicalTerminalSlots(t, fixture.service)
	if slots["terminal-a"] == nil || slots["terminal-a"].Lifecycle != domain.TerminalLifecycleSuspended || slots["terminal-b"] == nil || slots["terminal-b"].Lifecycle != domain.TerminalLifecycleActive {
		t.Fatalf("preserve runtime slots = %#v", slots)
	}
	fixture.service.commit(func(runtime *domain.ProcessRuntime) transition {
		runtime.Broadcast.TerminalRuntimes["terminal-b"].Hack.Solved = true
		return transition{accepted: true}
	})

	state, err = fixture.service.RequestTerminalActivation(terminalTarget("terminal-a", "Terminal A Updated"))
	if err != nil {
		t.Fatalf("reactivate preserved A error = %v", err)
	}
	if state.Broadcast.ActiveTerminalID == nil || *state.Broadcast.ActiveTerminalID != "terminal-a" {
		t.Fatalf("reactivated preserved source state = %#v", state.Broadcast)
	}
	restored := canonicalTerminal(t, fixture.service, "terminal-a")
	if restored.TerminalName != "Terminal A Updated" || restored.Nav.Path[len(restored.Nav.Path)-1] != "docs" || restored.Hack == nil || restored.Hack.GenerationID != "generation-terminal-a-1" {
		t.Fatalf("reactivated preserved checkpoint = %#v", restored)
	}
	if suspends, reactivates, discards := fixture.terminals.DecisionCalls(); suspends != 1 || reactivates != 1 || discards != 0 {
		t.Fatalf("preserve lifecycle calls suspend/reactivate/discard = %d/%d/%d", suspends, reactivates, discards)
	}
}

func TestUnfinishedTerminalSwitchCancelDiscardStaleAndDeletionGuards(t *testing.T) {
	t.Run("cancel", func(t *testing.T) {
		fixture := newUS8SwitchFixture(t)
		sourceBefore := canonicalTerminalBytes(t, fixture.service, "terminal-a")
		state, err := fixture.service.RequestTerminalClear()
		if err != nil {
			t.Fatalf("unfinished clear request error = %v", err)
		}
		assertPendingSwitch(t, state, fixture.broadcastID, "terminal-a", "")
		state, err = fixture.service.ResolveTerminalSwitch(state.PendingSwitch.SwitchID, domain.TerminalSwitchCancel)
		if err != nil {
			t.Fatalf("ResolveTerminalSwitch(cancel) error = %v", err)
		}
		if state.PendingSwitch != nil || state.Broadcast.ActiveTerminalID == nil || *state.Broadcast.ActiveTerminalID != "terminal-a" {
			t.Fatalf("cancel resolution state = %#v", state)
		}
		if got := canonicalTerminalBytes(t, fixture.service, "terminal-a"); !reflect.DeepEqual(got, sourceBefore) {
			t.Fatalf("cancel changed source checkpoint\nbefore: %s\nafter:  %s", sourceBefore, got)
		}
	})

	t.Run("discard and inactive rejection", func(t *testing.T) {
		fixture := newUS8SwitchFixture(t)
		firstGeneration := canonicalTerminal(t, fixture.service, "terminal-a").Hack.GenerationID
		state, err := fixture.service.RequestTerminalActivation(terminalTarget("terminal-b", "Terminal B"))
		if err != nil {
			t.Fatal(err)
		}
		state, err = fixture.service.ResolveTerminalSwitch(state.PendingSwitch.SwitchID, domain.TerminalSwitchDiscard)
		if err != nil {
			t.Fatalf("ResolveTerminalSwitch(discard) error = %v", err)
		}
		if state.PendingSwitch != nil || state.Broadcast.ActiveTerminalID == nil || *state.Broadcast.ActiveTerminalID != "terminal-b" {
			t.Fatalf("discard resolution state = %#v", state)
		}
		if slots := canonicalTerminalSlots(t, fixture.service); slots["terminal-a"] != nil {
			t.Fatalf("discard retained source runtime: %#v", slots)
		}
		beforeCalls := fixture.actions.Calls()
		beforeSlots := canonicalTerminalSlotBytes(t, fixture.service)
		fixture.service.DispatchPlayerAction(fixture.connectionID, domain.RuntimeCommand{
			RequestID: "discarded-source-action", BroadcastID: fixture.broadcastID, TerminalID: "terminal-a",
			Kind: domain.RuntimeCommandNavAction, Action: "back",
		})
		result := actionResultForRequest(t, fixture.effects, "discarded-source-action")
		if result.Accepted || result.Reason != domain.ActionReasonStaleTerminal || fixture.actions.Calls() != beforeCalls || !reflect.DeepEqual(canonicalTerminalSlotBytes(t, fixture.service), beforeSlots) {
			t.Fatalf("discarded source action = %#v calls=%d", result, fixture.actions.Calls())
		}

		fixture.service.commit(func(runtime *domain.ProcessRuntime) transition {
			runtime.Broadcast.TerminalRuntimes["terminal-b"].Hack.Solved = true
			return transition{accepted: true}
		})
		if _, err = fixture.service.RequestTerminalActivation(terminalTarget("terminal-a", "Terminal A Fresh")); err != nil {
			t.Fatal(err)
		}
		fresh := canonicalTerminal(t, fixture.service, "terminal-a")
		if fresh.Hack == nil || fresh.Hack.GenerationID == firstGeneration {
			t.Fatalf("discarded source was not freshly generated: old %q new %#v", firstGeneration, fresh.Hack)
		}
	})

	t.Run("stale and deletion guards", func(t *testing.T) {
		fixture := newUS8SwitchFixture(t)
		state, err := fixture.service.RequestTerminalActivation(terminalTarget("terminal-b", "Terminal B"))
		if err != nil {
			t.Fatal(err)
		}
		switchID := state.PendingSwitch.SwitchID
		before := fixture.service.Snapshot()
		beforeSlots := canonicalTerminalSlotBytes(t, fixture.service)
		if _, err = fixture.service.ResolveTerminalSwitch("unknown-switch", domain.TerminalSwitchPreserve); err == nil {
			t.Fatal("unknown switch decision unexpectedly resolved")
		}
		if !reflect.DeepEqual(fixture.service.Snapshot(), before) || !reflect.DeepEqual(canonicalTerminalSlotBytes(t, fixture.service), beforeSlots) {
			t.Fatal("stale decision refusal changed canonical state")
		}
		if err = fixture.service.CanDeleteTerminal("terminal-a"); err == nil {
			t.Fatal("active source terminal deletion was allowed")
		}
		if _, err = fixture.service.ResolveTerminalSwitch(switchID, domain.TerminalSwitchPreserve); err != nil {
			t.Fatal(err)
		}
		if err = fixture.service.CanDeleteTerminal("terminal-a"); err == nil {
			t.Fatal("preserved suspended terminal deletion was allowed")
		}
		if err = fixture.service.CanDeleteTerminal("terminal-unowned"); err != nil {
			t.Fatalf("unowned terminal deletion guard error = %v", err)
		}
	})
}

func TestResetFailedHackAtomicallyReplacesOnlyActiveSlotAndSerializesDuplicates(t *testing.T) {
	fixture := newUS8SwitchFixture(t)
	fixture.service.commit(func(runtime *domain.ProcessRuntime) transition {
		active := runtime.Broadcast.TerminalRuntimes["terminal-a"]
		active.Hack.AttemptsLeft = 0
		active.Hack.Failed = true
		active.Hack.Log = []string{"TERMINAL LOCKED"}
		runtime.Broadcast.TerminalRuntimes["terminal-observer"] = testTerminalRuntime("terminal-observer")
		runtime.Broadcast.TerminalRuntimes["terminal-observer"].Lifecycle = domain.TerminalLifecycleSuspended
		return transition{accepted: true}
	})

	before := fixture.service.Snapshot()
	assignmentsBefore := masterAssignments(before)
	otherBefore := canonicalTerminalBytes(t, fixture.service, "terminal-observer")
	oldGeneration := canonicalTerminal(t, fixture.service, "terminal-a").Hack.GenerationID
	latest := terminalTarget("terminal-a", "Terminal A Latest")
	latest.HackLevel = 2
	latest.IntroText = "LATEST INTRO"
	revisionBefore := fixture.service.Revision()

	state, err := fixture.service.ResetFailedHack(latest)
	if err != nil {
		t.Fatalf("ResetFailedHack() error = %v", err)
	}
	if state.Revision != revisionBefore+1 || state.Broadcast == nil || state.Broadcast.ID != fixture.broadcastID || state.Broadcast.ActiveTerminalID == nil || *state.Broadcast.ActiveTerminalID != "terminal-a" {
		t.Fatalf("reset state = %#v", state)
	}
	if !reflect.DeepEqual(masterAssignments(state), assignmentsBefore) || state.Broadcast.ControllerSessionID == nil || before.Broadcast.ControllerSessionID == nil || *state.Broadcast.ControllerSessionID != *before.Broadcast.ControllerSessionID || !reflect.DeepEqual(state.Sessions, before.Sessions) || !reflect.DeepEqual(state.Roster, before.Roster) {
		t.Fatalf("reset changed identity/ownership state\nbefore=%#v\nafter=%#v", before, state)
	}
	if got := canonicalTerminalBytes(t, fixture.service, "terminal-observer"); !reflect.DeepEqual(got, otherBefore) {
		t.Fatalf("reset changed unrelated runtime\nbefore=%s\nafter=%s", otherBefore, got)
	}
	fresh := canonicalTerminal(t, fixture.service, "terminal-a")
	if fresh.Hack == nil || fresh.Hack.GenerationID == oldGeneration || fresh.Hack.Level != 2 || fresh.Hack.AttemptsLeft != fresh.Hack.AttemptsMax || fresh.Hack.Failed || fresh.Hack.Solved || len(fresh.Hack.Log) != 0 || fresh.TerminalName != latest.TerminalName || fresh.IntroText != latest.IntroText {
		t.Fatalf("fresh active runtime = %#v", fresh)
	}
	if _, _, resets := fixture.terminals.DecisionCalls(); resets != 1 {
		t.Fatalf("reset lifecycle calls = %d, want 1", resets)
	}
	if !hasLiveEffectAtRevision(fixture.effects.Values(), state.Revision, "terminal-a") {
		t.Fatalf("reset emitted no live effect at revision %d", state.Revision)
	}

	afterFirst := canonicalTerminalSlotBytes(t, fixture.service)
	if duplicate, duplicateErr := fixture.service.ResetFailedHack(latest); duplicateErr == nil || duplicate.Revision != state.Revision {
		t.Fatalf("duplicate reset = state %#v error %v", duplicate, duplicateErr)
	}
	if fixture.service.Revision() != state.Revision || !reflect.DeepEqual(canonicalTerminalSlotBytes(t, fixture.service), afterFirst) {
		t.Fatal("duplicate reset mutated canonical state")
	}
	wrong := latest
	wrong.TerminalID = "terminal-observer"
	if stale, staleErr := fixture.service.ResetFailedHack(wrong); staleErr == nil || stale.Revision != state.Revision {
		t.Fatalf("stale reset = state %#v error %v", stale, staleErr)
	}
}

func TestResetFailedHackSerializesConcurrentDuplicateRequests(t *testing.T) {
	fixture := newUS8SwitchFixture(t)
	fixture.service.commit(func(runtime *domain.ProcessRuntime) transition {
		active := runtime.Broadcast.TerminalRuntimes["terminal-a"]
		active.Hack.AttemptsLeft = 0
		active.Hack.Failed = true
		return transition{accepted: true}
	})
	target := terminalTarget("terminal-a", "Terminal A Retry")
	revisionBefore := fixture.service.Revision()

	const callers = 32
	results := make(chan error, callers)
	var group sync.WaitGroup
	group.Add(callers)
	for range callers {
		go func() {
			defer group.Done()
			_, err := fixture.service.ResetFailedHack(target)
			results <- err
		}()
	}
	group.Wait()
	close(results)

	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		}
	}
	if accepted != 1 || fixture.service.Revision() != revisionBefore+1 {
		t.Fatalf("concurrent resets accepted=%d revision=%d, want 1/%d", accepted, fixture.service.Revision(), revisionBefore+1)
	}
	fresh := canonicalTerminal(t, fixture.service, "terminal-a")
	if fresh.Hack == nil || fresh.Hack.Failed || fresh.Hack.Solved || fresh.Hack.AttemptsLeft != fresh.Hack.AttemptsMax {
		t.Fatalf("serialized reset result = %#v", fresh)
	}
	if _, _, resets := fixture.terminals.DecisionCalls(); resets != 1 {
		t.Fatalf("concurrent reset lifecycle calls = %d, want 1", resets)
	}
}

func TestCurrentLiveForSessionResumesCoordinatorOwnedRuntimeWithoutRegeneration(t *testing.T) {
	liveService := live.New(nil, nil)
	service := New(Config{IDs: &counterIDSource{}, Runtime: liveService, Terminals: liveService})
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	controller := service.CreateSession("resume-controller")
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: "resume-controller", SessionID: controller.SessionID, RequestID: "resume-select",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[0].ID,
	}); !result.Accepted {
		t.Fatalf("SelectCharacter() = %#v", result)
	}
	if _, err = service.RequestTerminalActivation(terminalTarget("terminal-resume", "Resume Terminal")); err != nil {
		t.Fatal(err)
	}

	want, revision, ok := service.CurrentLiveForSession(controller.SessionID)
	if !ok || want == nil || want.TerminalID != "terminal-resume" || want.Hack == nil {
		t.Fatalf("CurrentLiveForSession() = %#v, %d, %v", want, revision, ok)
	}
	want.TerminalName = "mutated detached projection"
	want.Hack.AttemptsLeft = -1
	canonical, canonicalRevision, ok := service.CurrentLiveForSession(controller.SessionID)
	if !ok || canonical == nil || canonical.TerminalName != "Resume Terminal" || canonical.Hack.AttemptsLeft < 0 || canonicalRevision != revision {
		t.Fatalf("detached current live = %#v revision=%d ok=%v", canonical, canonicalRevision, ok)
	}

	returnedToken, resumed := service.AttachConnection("resume-new-tab", controller.BrowserToken)
	if returnedToken != controller.BrowserToken || resumed == nil || resumed.SessionID != controller.SessionID || resumed.Role != domain.PlayerRoleActive || resumed.ActiveTerminalID != "terminal-resume" {
		t.Fatalf("recognized resume = token %q state %#v", returnedToken, resumed)
	}
	resumedLive, _, ok := service.CurrentLiveForSession(resumed.SessionID)
	if !ok || !reflect.DeepEqual(resumedLive, canonical) {
		t.Fatalf("resumed live regenerated or drifted\nwant=%#v\ngot=%#v", canonical, resumedLive)
	}

	unassigned := service.CreateSession("resume-unassigned")
	if leaked, _, available := service.CurrentLiveForSession(unassigned.SessionID); available || leaked != nil {
		t.Fatalf("unassigned session received coordinator runtime: %#v", leaked)
	}
}

func TestForceHackSuccessMutatesOnlyCoordinatorOwnedActiveRuntime(t *testing.T) {
	liveService := live.New(nil, nil)
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{
		IDs: &counterIDSource{}, Enqueue: effects.Enqueue, Runtime: liveService,
		Terminals: liveService, TrustedHack: liveService,
	})
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	controller := service.CreateSession("force-controller")
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: "force-controller", SessionID: controller.SessionID, RequestID: "force-select",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[0].ID,
	}); !result.Accepted {
		t.Fatalf("SelectCharacter() = %#v", result)
	}
	if _, err = service.RequestTerminalActivation(terminalTarget("terminal-force", "Force Terminal")); err != nil {
		t.Fatal(err)
	}
	before, beforeRevision, ok := service.CurrentLiveForSession(controller.SessionID)
	if !ok || before == nil || before.Hack == nil || before.Hack.Solved || before.Hack.Failed {
		t.Fatalf("force precondition = %#v", before)
	}
	if legacy := liveService.Snapshot(); legacy != nil {
		t.Fatalf("legacy live slot unexpectedly owns coordinator runtime: %#v", legacy)
	}

	forced, accepted := service.ForceHackSuccess()
	if !accepted || forced == nil || !forced.Solved || forced.Failed || forced.AttemptsLeft != before.Hack.AttemptsLeft {
		t.Fatalf("ForceHackSuccess() = %#v, %v", forced, accepted)
	}
	after, afterRevision, ok := service.CurrentLiveForSession(controller.SessionID)
	if !ok || after == nil || after.Hack == nil || !after.Hack.Solved || after.Hack.Failed || afterRevision != beforeRevision+1 {
		t.Fatalf("forced canonical runtime = %#v revision=%d", after, afterRevision)
	}
	if legacy := liveService.Snapshot(); legacy != nil {
		t.Fatalf("trusted force populated legacy live slot: %#v", legacy)
	}
	if !hasLiveEffectAtRevision(effects.Values(), afterRevision, "terminal-force") {
		t.Fatalf("trusted force emitted no complete live projection at revision %d", afterRevision)
	}
	liveEffects := 0
	for _, effect := range effects.Values() {
		if effect.Revision == afterRevision && effect.Live != nil {
			liveEffects++
		}
	}
	if liveEffects != 1 {
		t.Fatalf("trusted force emitted %d live projections at revision %d, want 1", liveEffects, afterRevision)
	}
	if duplicate, duplicateOK := service.ForceHackSuccess(); duplicateOK || duplicate != nil || service.Revision() != afterRevision {
		t.Fatalf("duplicate force = %#v, %v revision=%d", duplicate, duplicateOK, service.Revision())
	}

	service.commit(func(runtime *domain.ProcessRuntime) transition {
		terminal := activeTerminalRuntime(runtime.Broadcast)
		terminal.Hack.Solved = false
		terminal.Hack.Failed = true
		terminal.Hack.AttemptsLeft = 0
		return transition{accepted: true}
	})
	failedRevision := service.Revision()
	if failed, failedOK := service.ForceHackSuccess(); failedOK || failed != nil || service.Revision() != failedRevision {
		t.Fatalf("failed-puzzle force = %#v, %v revision=%d", failed, failedOK, service.Revision())
	}
}

func hasLiveEffectAtRevision(effects []Effect, revision uint64, terminalID string) bool {
	for _, effect := range effects {
		if effect.Revision == revision && effect.Live != nil && effect.Live.TerminalID == terminalID {
			return true
		}
	}
	return false
}

func TestEndAndRestartBroadcastClearsEpochStateWhileRetainingProcessIdentity(t *testing.T) {
	actions := &recordingTerminalRuntime{}
	terminals := newRecordingDecisionTerminalLifecycle()
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{IDs: &counterIDSource{}, Enqueue: effects.Enqueue, Runtime: actions, Terminals: terminals})

	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	maraID := state.Roster[0].ID
	state, err = service.AddCharacter("Boone")
	if err != nil {
		t.Fatal(err)
	}
	booneID := state.Roster[1].ID
	rosterBefore := append([]domain.MasterRosterEntry(nil), state.Roster...)
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	firstBroadcastID := state.Broadcast.ID
	firstConnection := domain.ConnectionID("lifetime-first")
	secondConnection := domain.ConnectionID("lifetime-second")
	first := service.CreateSession(firstConnection)
	second := service.CreateSession(secondConnection)
	if _, err = service.RenameLogicalSession(first.SessionID, "TABLET LEFT"); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RenameLogicalSession(second.SessionID, "TABLET RIGHT"); err != nil {
		t.Fatal(err)
	}
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: firstConnection, SessionID: first.SessionID, RequestID: "lifetime-first-request",
		BroadcastID: firstBroadcastID, CharacterID: maraID,
	}); !result.Accepted {
		t.Fatalf("first-broadcast controller selection = %#v", result)
	}
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: secondConnection, SessionID: second.SessionID, RequestID: "lifetime-second-request",
		BroadcastID: firstBroadcastID, CharacterID: booneID,
	}); !result.Accepted {
		t.Fatalf("first-broadcast observer selection = %#v", result)
	}
	if _, err = service.RequestTerminalActivation(terminalTarget("lifetime-terminal-a", "Terminal A")); err != nil {
		t.Fatal(err)
	}
	state, err = service.RequestTerminalActivation(terminalTarget("lifetime-terminal-b", "Terminal B"))
	if err != nil {
		t.Fatal(err)
	}
	assertPendingSwitch(t, state, firstBroadcastID, "lifetime-terminal-a", "lifetime-terminal-b")
	if cacheCount := requestCacheCount(t, service); cacheCount < 2 {
		t.Fatalf("populated first-broadcast request cache count = %d, want at least 2", cacheCount)
	}
	if slots := canonicalTerminalSlots(t, service); len(slots) == 0 {
		t.Fatal("populated first broadcast has no coordinator-owned terminal runtime")
	}

	baseline := effects.Calls()
	beforeEndRevision := service.Revision()
	ended, err := service.EndBroadcast()
	if err != nil {
		t.Fatalf("EndBroadcast() error = %v", err)
	}
	if ended == nil || ended.Revision != beforeEndRevision+1 || ended.Broadcast != nil || ended.PendingSwitch != nil {
		t.Fatalf("EndBroadcast() state = %#v, want no broadcast at revision %d", ended, beforeEndRevision+1)
	}
	if !reflect.DeepEqual(masterRosterIdentities(ended), masterRosterIdentities(&domain.MasterCoordinationState{Roster: rosterBefore})) {
		t.Fatalf("broadcast end changed retained roster\nbefore: %#v\nafter:  %#v", rosterBefore, ended.Roster)
	}
	firstEnded := masterSession(t, ended, first.SessionID)
	secondEnded := masterSession(t, ended, second.SessionID)
	if firstEnded.FallbackName != "TABLET LEFT" || secondEnded.FallbackName != "TABLET RIGHT" || !firstEnded.Connected || !secondEnded.Connected {
		t.Fatalf("broadcast end changed retained session identity/presence: first %#v second %#v", firstEnded, secondEnded)
	}
	for _, session := range []domain.MasterSessionEntry{firstEnded, secondEnded} {
		if session.Character != nil || session.Role != domain.PlayerRoleUnassigned {
			t.Fatalf("broadcast end retained assignment/controller role: %#v", session)
		}
	}
	for _, character := range ended.Roster {
		if character.ClaimedBySessionID != nil {
			t.Fatalf("broadcast end retained roster claim: %#v", character)
		}
	}
	assertEndedRuntimeRoot(t, service)
	assertBroadcastEndEffects(t, effects.Values()[baseline:], ended.Revision, first.SessionID, second.SessionID)

	returnedToken, reattached := service.AttachConnection("lifetime-first-reopen", first.BrowserToken)
	if returnedToken != first.BrowserToken || reattached == nil || reattached.SessionID != first.SessionID || reattached.FallbackName != "TABLET LEFT" || reattached.Character != nil || reattached.Phase != domain.PlayerPhaseNoBroadcast {
		t.Fatalf("same-process post-end token/session = token %q state %#v", returnedToken, reattached)
	}

	secondBroadcast, err := service.StartBroadcast()
	if err != nil {
		t.Fatalf("second StartBroadcast() error = %v", err)
	}
	if secondBroadcast.Broadcast == nil || secondBroadcast.Broadcast.ID == "" || secondBroadcast.Broadcast.ID == firstBroadcastID {
		t.Fatalf("second broadcast ID = %#v, want fresh from %q", secondBroadcast.Broadcast, firstBroadcastID)
	}
	if secondBroadcast.Broadcast.ControllerSessionID != nil || secondBroadcast.Broadcast.ActiveTerminalID != nil || secondBroadcast.PendingSwitch != nil {
		t.Fatalf("fresh broadcast retained controller/terminal/pending state: %#v", secondBroadcast)
	}
	if !reflect.DeepEqual(masterRosterIdentities(secondBroadcast), masterRosterIdentities(ended)) {
		t.Fatalf("second broadcast changed roster identities: %#v vs %#v", secondBroadcast.Roster, ended.Roster)
	}
	for _, sessionID := range []domain.LogicalSessionID{first.SessionID, second.SessionID} {
		player, ok := service.PlayerSnapshot(sessionID)
		if !ok || player.Character != nil || player.Role != domain.PlayerRoleUnassigned || player.Phase != domain.PlayerPhaseSelecting {
			t.Fatalf("fresh broadcast session %q = %#v, ok=%t", sessionID, player, ok)
		}
	}
	if cacheCount := requestCacheCount(t, service); cacheCount != 0 {
		t.Fatalf("fresh broadcast retained %d prior request results", cacheCount)
	}

	// Reusing an old request ID with a new-broadcast payload proves that the
	// per-broadcast cache was discarded instead of replaying the old result.
	newController := service.SelectCharacter(CharacterSelection{
		ConnectionID: secondConnection, SessionID: second.SessionID, RequestID: "lifetime-second-request",
		BroadcastID: secondBroadcast.Broadcast.ID, CharacterID: maraID,
	})
	if !newController.Accepted {
		t.Fatalf("fresh-broadcast reused request ID = %#v, want new accepted selection", newController)
	}
	final := service.Snapshot()
	assertExactlyOneController(t, final, second.SessionID)
	if got := masterSession(t, final, first.SessionID); got.Character != nil || got.Role != domain.PlayerRoleUnassigned {
		t.Fatalf("old controller retained ownership in fresh broadcast: %#v", got)
	}
}

func requestCacheCount(t *testing.T, service *Service) int {
	t.Helper()
	service.mu.RLock()
	defer service.mu.RUnlock()
	count := 0
	for _, session := range service.runtime.SessionsByID {
		if session != nil {
			count += len(session.RequestResults)
		}
	}
	return count
}

func masterRosterIdentities(state *domain.MasterCoordinationState) []domain.PlayerCharacter {
	identities := make([]domain.PlayerCharacter, 0)
	if state == nil {
		return identities
	}
	for _, character := range state.Roster {
		identities = append(identities, domain.PlayerCharacter{ID: character.ID, Name: character.Name})
	}
	return identities
}

func assertEndedRuntimeRoot(t *testing.T, service *Service) {
	t.Helper()
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.runtime.Broadcast != nil || service.runtime.PendingSwitch != nil {
		t.Fatalf("ended runtime retained broadcast/pending switch: %#v", service.runtime)
	}
	for sessionID, session := range service.runtime.SessionsByID {
		if session == nil || len(session.RequestResults) != 0 {
			t.Fatalf("ended runtime session %q request cache = %#v", sessionID, session)
		}
	}
}

func assertBroadcastEndEffects(t *testing.T, effects []Effect, revision uint64, sessionIDs ...domain.LogicalSessionID) {
	t.Helper()
	if masterEffectCount(effects) != 1 || !hasClearEffectAtRevision(effects, revision) {
		t.Fatalf("broadcast end effects = %#v, want one master and terminal clear at revision %d", effects, revision)
	}
	seenPlayers := make(map[domain.LogicalSessionID]bool)
	for _, effect := range effects {
		if effect.Revision != revision {
			t.Fatalf("broadcast end effect revision = %d, want %d", effect.Revision, revision)
		}
		if effect.Player != nil {
			if effect.Player.Character != nil || effect.Player.Role != domain.PlayerRoleUnassigned || effect.Player.Phase != domain.PlayerPhaseNoBroadcast {
				t.Fatalf("broadcast end player effect = %#v", effect.Player)
			}
			seenPlayers[effect.SessionID] = true
		}
	}
	for _, sessionID := range sessionIDs {
		if !seenPlayers[sessionID] {
			t.Fatalf("broadcast end omitted player context for session %q", sessionID)
		}
	}
}

type us8SwitchFixture struct {
	service      *Service
	effects      *testutil.FakeOrderedEffectSink[Effect]
	actions      *recordingTerminalRuntime
	terminals    *recordingDecisionTerminalLifecycle
	broadcastID  domain.BroadcastID
	connectionID domain.ConnectionID
}

func newUS8SwitchFixture(t *testing.T) us8SwitchFixture {
	t.Helper()
	actions := &recordingTerminalRuntime{}
	terminals := newRecordingDecisionTerminalLifecycle()
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{IDs: &counterIDSource{}, Enqueue: effects.Enqueue, Runtime: actions, Terminals: terminals})
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	characterID := state.Roster[0].ID
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	connectionID := domain.ConnectionID("decision-controller")
	controller := service.CreateSession(connectionID)
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: connectionID, SessionID: controller.SessionID, RequestID: "decision-select",
		BroadcastID: state.Broadcast.ID, CharacterID: characterID,
	}); !result.Accepted {
		t.Fatalf("controller selection = %#v", result)
	}
	if _, err = service.RequestTerminalActivation(terminalTarget("terminal-a", "Terminal A")); err != nil {
		t.Fatal(err)
	}
	return us8SwitchFixture{service: service, effects: effects, actions: actions, terminals: terminals, broadcastID: state.Broadcast.ID, connectionID: connectionID}
}

func assertPendingSwitch(t *testing.T, state *domain.MasterCoordinationState, broadcastID domain.BroadcastID, sourceTerminalID string, targetTerminalID string) {
	t.Helper()
	if state == nil || state.Broadcast == nil || state.Broadcast.ID != broadcastID || state.Broadcast.ActiveTerminalID == nil || *state.Broadcast.ActiveTerminalID != sourceTerminalID || state.PendingSwitch == nil {
		t.Fatalf("pending switch state = %#v", state)
	}
	if state.PendingSwitch.BroadcastID != broadcastID || state.PendingSwitch.SourceTerminalID != sourceTerminalID {
		t.Fatalf("pending switch source/broadcast = %#v", state.PendingSwitch)
	}
	if targetTerminalID == "" {
		if state.PendingSwitch.TargetTerminalID != nil {
			t.Fatalf("pending clear target = %#v, want nil", state.PendingSwitch.TargetTerminalID)
		}
	} else if state.PendingSwitch.TargetTerminalID == nil || *state.PendingSwitch.TargetTerminalID != targetTerminalID {
		t.Fatalf("pending activation target = %#v, want %q", state.PendingSwitch.TargetTerminalID, targetTerminalID)
	}
}

type recordingDecisionTerminalLifecycle struct {
	recordingTerminalLifecycle
	muDecision  sync.Mutex
	generations map[string]int
	suspends    int
	reactivates int
	discards    int
}

type terminalDecisionLifecycleContract interface {
	TerminalRuntimeLifecycle
	SuspendRuntime(*domain.TerminalRuntime)
	ReactivateRuntime(*domain.TerminalRuntime, domain.TerminalTarget) *domain.PublicLiveState
	DiscardRuntime(domain.TerminalTarget) (*domain.TerminalRuntime, *domain.PublicLiveState)
	ResetFailedHack(*domain.TerminalRuntime, domain.TerminalTarget) (*domain.TerminalRuntime, *domain.PublicLiveState)
}

var _ terminalDecisionLifecycleContract = (*recordingDecisionTerminalLifecycle)(nil)

func newRecordingDecisionTerminalLifecycle() *recordingDecisionTerminalLifecycle {
	return &recordingDecisionTerminalLifecycle{generations: make(map[string]int)}
}

func (lifecycle *recordingDecisionTerminalLifecycle) CreateRuntime(target domain.TerminalTarget) (*domain.TerminalRuntime, *domain.PublicLiveState) {
	lifecycle.muDecision.Lock()
	lifecycle.generations[target.TerminalID]++
	generation := lifecycle.generations[target.TerminalID]
	lifecycle.muDecision.Unlock()
	runtime, _ := lifecycle.recordingTerminalLifecycle.CreateRuntime(target)
	runtime.Hack.Solved = false
	runtime.Hack.Failed = false
	runtime.Hack.Level = target.HackLevel
	runtime.Hack.AttemptsLeft = runtime.Hack.AttemptsMax
	runtime.Hack.Log = []string{}
	runtime.Hack.GenerationID = fmt.Sprintf("generation-%s-%d", target.TerminalID, generation)
	return runtime, publicTerminalRuntime(runtime)
}

func (lifecycle *recordingDecisionTerminalLifecycle) SuspendRuntime(runtime *domain.TerminalRuntime) {
	lifecycle.muDecision.Lock()
	lifecycle.suspends++
	lifecycle.muDecision.Unlock()
	runtime.Lifecycle = domain.TerminalLifecycleSuspended
}

func (lifecycle *recordingDecisionTerminalLifecycle) ReactivateRuntime(runtime *domain.TerminalRuntime, target domain.TerminalTarget) *domain.PublicLiveState {
	lifecycle.muDecision.Lock()
	lifecycle.reactivates++
	lifecycle.muDecision.Unlock()
	runtime.Lifecycle = domain.TerminalLifecycleActive
	return lifecycle.UpdateRuntime(runtime, target)
}

func (lifecycle *recordingDecisionTerminalLifecycle) DiscardRuntime(target domain.TerminalTarget) (*domain.TerminalRuntime, *domain.PublicLiveState) {
	lifecycle.muDecision.Lock()
	lifecycle.discards++
	lifecycle.muDecision.Unlock()
	return lifecycle.CreateRuntime(target)
}

func (lifecycle *recordingDecisionTerminalLifecycle) ResetFailedHack(runtime *domain.TerminalRuntime, target domain.TerminalTarget) (*domain.TerminalRuntime, *domain.PublicLiveState) {
	if runtime == nil || runtime.TerminalID != target.TerminalID || runtime.Hack == nil || !runtime.Hack.Failed || runtime.Hack.Solved {
		return nil, nil
	}
	lifecycle.muDecision.Lock()
	lifecycle.discards++
	lifecycle.muDecision.Unlock()
	return lifecycle.CreateRuntime(target)
}

func (lifecycle *recordingDecisionTerminalLifecycle) DecisionCalls() (int, int, int) {
	lifecycle.muDecision.Lock()
	defer lifecycle.muDecision.Unlock()
	return lifecycle.suspends, lifecycle.reactivates, lifecycle.discards
}

func terminalTarget(id string, name string) domain.TerminalTarget {
	return domain.TerminalTarget{
		TerminalID: id, TerminalName: name, HackLevel: 1, IntroText: "WELCOME " + id,
		Tree: testTerminalRuntime(id).Tree,
	}
}

func assertActiveTerminalAndIdentity(t *testing.T, state *domain.MasterCoordinationState, terminalID string, broadcastID domain.BroadcastID, controllerID domain.LogicalSessionID, assignments map[domain.LogicalSessionID]domain.CharacterID) {
	t.Helper()
	if state == nil || state.Broadcast == nil || state.Broadcast.ID != broadcastID || state.Broadcast.ActiveTerminalID == nil || *state.Broadcast.ActiveTerminalID != terminalID {
		t.Fatalf("active terminal state = %#v, want broadcast %q terminal %q", state, broadcastID, terminalID)
	}
	assertExactlyOneController(t, state, controllerID)
	if !reflect.DeepEqual(masterAssignments(state), assignments) {
		t.Fatalf("terminal transition changed assignments: got %#v want %#v", masterAssignments(state), assignments)
	}
}

func canonicalTerminalSlots(t *testing.T, service *Service) map[string]*domain.TerminalRuntime {
	t.Helper()
	service.mu.RLock()
	defer service.mu.RUnlock()
	result := make(map[string]*domain.TerminalRuntime)
	if service.runtime.Broadcast == nil {
		return result
	}
	for terminalID, runtime := range service.runtime.Broadcast.TerminalRuntimes {
		result[terminalID] = cloneTerminalRuntime(runtime)
	}
	return result
}

func canonicalTerminalSlotBytes(t *testing.T, service *Service) map[string][]byte {
	t.Helper()
	result := make(map[string][]byte)
	for terminalID := range canonicalTerminalSlots(t, service) {
		result[terminalID] = canonicalTerminalBytes(t, service, terminalID)
	}
	return result
}

func hasClearEffectAtRevision(effects []Effect, revision uint64) bool {
	for _, effect := range effects {
		if effect.Revision == revision && effect.ClearLiveTerminal && effect.Live == nil {
			return true
		}
	}
	return false
}

type recordingTerminalLifecycle struct {
	mu       sync.Mutex
	creates  int
	updates  int
	projects int
}

func (lifecycle *recordingTerminalLifecycle) CreateRuntime(target domain.TerminalTarget) (*domain.TerminalRuntime, *domain.PublicLiveState) {
	lifecycle.mu.Lock()
	lifecycle.creates++
	lifecycle.mu.Unlock()
	runtime := testTerminalRuntime(target.TerminalID)
	runtime.TerminalName = target.TerminalName
	runtime.Tree = cloneContentNode(target.Tree)
	runtime.HackLevel = target.HackLevel
	runtime.IntroText = target.IntroText
	runtime.Hack.GenerationID = "generation-" + target.TerminalID
	runtime.Hack.Solved = true
	runtime.Hack.Log = []string{"ACCESS GRANTED"}
	return runtime, publicTerminalRuntime(runtime)
}

func (lifecycle *recordingTerminalLifecycle) UpdateRuntime(runtime *domain.TerminalRuntime, target domain.TerminalTarget) *domain.PublicLiveState {
	lifecycle.mu.Lock()
	lifecycle.updates++
	lifecycle.mu.Unlock()
	runtime.TerminalName = target.TerminalName
	runtime.Tree = cloneContentNode(target.Tree)
	runtime.HackLevel = target.HackLevel
	runtime.IntroText = target.IntroText
	runtime.Nav = nav.Revalidate(runtime.Nav, runtime.Tree)
	return publicTerminalRuntime(runtime)
}

func (lifecycle *recordingTerminalLifecycle) ProjectRuntime(runtime *domain.TerminalRuntime) *domain.PublicLiveState {
	lifecycle.mu.Lock()
	lifecycle.projects++
	lifecycle.mu.Unlock()
	return publicTerminalRuntime(runtime)
}

func (lifecycle *recordingTerminalLifecycle) Calls() (int, int, int) {
	lifecycle.mu.Lock()
	defer lifecycle.mu.Unlock()
	return lifecycle.creates, lifecycle.updates, lifecycle.projects
}

func assertPresenceOnlyEffects(t *testing.T, effects []Effect, revision uint64) {
	t.Helper()
	if masterEffectCount(effects) != 1 {
		t.Fatalf("presence transition master effects = %d, want exactly 1 in %#v", masterEffectCount(effects), effects)
	}
	for _, effect := range effects {
		if effect.Revision != revision {
			t.Fatalf("presence effect revision = %d, want %d", effect.Revision, revision)
		}
		if effect.Live != nil || effect.Hack != nil || effect.Result != nil || effect.ClearLiveTerminal || effect.TerminalID != "" || effect.ConnectionID != "" {
			t.Fatalf("presence transition emitted gameplay/result payload: %#v", effect)
		}
		if effect.Master == nil && effect.Player == nil {
			t.Fatalf("presence transition emitted empty effect: %#v", effect)
		}
	}
}

func TestGameMasterRosterAndAssignmentCorrectionsPreserveRuntime(t *testing.T) {
	runtime := &recordingTerminalRuntime{}
	service := New(Config{IDs: &counterIDSource{}, Runtime: runtime})
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	maraID := state.Roster[0].ID
	state, err = service.AddCharacter("Boone")
	if err != nil {
		t.Fatal(err)
	}
	booneID := state.Roster[1].ID
	state, err = service.AddCharacter("Mara")
	if err != nil {
		t.Fatalf("duplicate character names must remain valid: %v", err)
	}
	if len(state.Roster) != 3 || state.Roster[2].Name != "Mara" || state.Roster[2].ID == maraID {
		t.Fatalf("duplicate-name roster entry = %#v, want a distinct stable identity", state.Roster)
	}
	duplicateMaraID := state.Roster[2].ID

	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	first := service.CreateSession("gm-first")
	second := service.CreateSession("gm-second")
	third := service.CreateSession("gm-third")
	terminalID := "terminal-roster-neutral"
	service.commit(func(root *domain.ProcessRuntime) transition {
		root.Broadcast.ActiveTerminalID = &terminalID
		root.Broadcast.TerminalRuntimes[terminalID] = testTerminalRuntime(terminalID)
		return transition{accepted: true}
	})
	runtimeBefore := canonicalTerminalBytes(t, service, terminalID)

	state, err = service.RenameCharacter(maraID, "  Mara Voss  ")
	if err != nil {
		t.Fatalf("RenameCharacter() error = %v", err)
	}
	if state.Roster[0].ID != maraID || state.Roster[0].Name != "Mara Voss" {
		t.Fatalf("renamed roster entry = %#v, want stable ID and trimmed name", state.Roster[0])
	}
	assertRejectedCoordinationMutation(t, service, func() error {
		_, renameErr := service.RenameCharacter(maraID, strings.Repeat("x", 81))
		return renameErr
	})

	state, err = service.RenameLogicalSession(first.SessionID, "  TABLET LEFT  ")
	if err != nil {
		t.Fatalf("RenameLogicalSession() error = %v", err)
	}
	if got := masterSession(t, state, first.SessionID).FallbackName; got != "TABLET LEFT" {
		t.Fatalf("renamed fallback = %q, want TABLET LEFT", got)
	}
	assertRejectedCoordinationMutation(t, service, func() error {
		_, renameErr := service.RenameLogicalSession(second.SessionID, "TABLET LEFT")
		return renameErr
	})
	assertRejectedCoordinationMutation(t, service, func() error {
		_, renameErr := service.RenameLogicalSession(second.SessionID, "   ")
		return renameErr
	})

	state, err = service.AssignCharacter(first.SessionID, maraID)
	if err != nil {
		t.Fatalf("AssignCharacter(first) error = %v", err)
	}
	if state.Broadcast.ControllerSessionID == nil || *state.Broadcast.ControllerSessionID != first.SessionID {
		t.Fatalf("first GM assignment controller = %#v, want %q", state.Broadcast, first.SessionID)
	}
	state, err = service.AssignCharacter(second.SessionID, booneID)
	if err != nil {
		t.Fatalf("AssignCharacter(second) error = %v", err)
	}
	assertExclusiveClaimInvariants(t, state)
	if got := masterSession(t, state, second.SessionID).Role; got != domain.PlayerRoleObserver {
		t.Fatalf("second GM assignment role = %q, want observer", got)
	}
	assertRejectedCoordinationMutation(t, service, func() error {
		_, assignErr := service.AssignCharacter(third.SessionID, booneID)
		return assignErr
	})
	assertRejectedCoordinationMutation(t, service, func() error {
		_, deleteErr := service.DeleteCharacter(maraID)
		return deleteErr
	})

	state, err = service.MoveCharacter(maraID, third.SessionID)
	if err != nil {
		t.Fatalf("MoveCharacter() error = %v", err)
	}
	assertExclusiveClaimInvariants(t, state)
	if state.Broadcast.ControllerSessionID != nil {
		t.Fatalf("moving the controller character retained or promoted control: %#v", state.Broadcast)
	}
	if got := masterSession(t, state, first.SessionID); got.Character != nil || got.Role != domain.PlayerRoleUnassigned {
		t.Fatalf("former owner after move = %#v, want unassigned", got)
	}
	if got := masterSession(t, state, third.SessionID); got.Character == nil || got.Character.ID != maraID || got.Role != domain.PlayerRoleObserver {
		t.Fatalf("move destination = %#v, want observer owning stable Mara ID", got)
	}

	state, err = service.ReleaseCharacter(second.SessionID)
	if err != nil {
		t.Fatalf("ReleaseCharacter() error = %v", err)
	}
	assertExclusiveClaimInvariants(t, state)
	if got := masterSession(t, state, second.SessionID); got.Character != nil || got.Role != domain.PlayerRoleUnassigned {
		t.Fatalf("released session = %#v, want unassigned", got)
	}
	state, err = service.DeleteCharacter(booneID)
	if err != nil {
		t.Fatalf("DeleteCharacter(unclaimed) error = %v", err)
	}
	for _, rosterEntry := range state.Roster {
		if rosterEntry.ID == booneID {
			t.Fatalf("deleted character remains in roster: %#v", state.Roster)
		}
	}
	if state.Roster[1].ID != duplicateMaraID {
		t.Fatalf("delete changed surviving stable roster order: %#v", state.Roster)
	}

	if runtime.RandomCalls() != 0 {
		t.Fatalf("roster/assignment commands consumed runtime randomness %d times", runtime.RandomCalls())
	}
	if got := canonicalTerminalBytes(t, service, terminalID); !reflect.DeepEqual(got, runtimeBefore) {
		t.Fatalf("roster/assignment commands mutated canonical terminal/puzzle\nbefore: %s\nafter:  %s", runtimeBefore, got)
	}
}

func assertRejectedCoordinationMutation(t *testing.T, service *Service, command func() error) {
	t.Helper()
	before := service.Snapshot()
	if err := command(); err == nil {
		t.Fatal("coordination command unexpectedly succeeded")
	}
	after := service.Snapshot()
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("rejected coordination command changed state\nbefore: %#v\nafter:  %#v", before, after)
	}
}

func masterEffectCount(effects []Effect) int {
	count := 0
	for _, effect := range effects {
		if effect.Master != nil {
			count++
		}
	}
	return count
}

type us2Fixture struct {
	service              *Service
	effects              *testutil.FakeOrderedEffectSink[Effect]
	broadcastID          domain.BroadcastID
	terminalID           string
	controllerConnection domain.ConnectionID
	observerConnection   domain.ConnectionID
	unassignedConnection domain.ConnectionID
	controllerSession    domain.LogicalSessionID
	observerSession      domain.LogicalSessionID
	unassignedSession    domain.LogicalSessionID
	controllerToken      domain.BrowserToken
	observerToken        domain.BrowserToken
}

func newUS2Fixture(t *testing.T, runtime *recordingTerminalRuntime) us2Fixture {
	t.Helper()
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{IDs: &counterIDSource{}, Enqueue: effects.Enqueue, Runtime: runtime})
	state, err := service.AddCharacter("Mara")
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.AddCharacter("Boone")
	if err != nil {
		t.Fatal(err)
	}
	state, err = service.StartBroadcast()
	if err != nil {
		t.Fatal(err)
	}
	controllerConnection := domain.ConnectionID("connection-controller")
	observerConnection := domain.ConnectionID("connection-observer")
	unassignedConnection := domain.ConnectionID("connection-unassigned")
	controller := service.CreateSession(controllerConnection)
	observer := service.CreateSession(observerConnection)
	unassigned := service.CreateSession(unassignedConnection)
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: controllerConnection, SessionID: controller.SessionID, RequestID: "select-controller",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[0].ID,
	}); !result.Accepted {
		t.Fatalf("controller selection = %#v", result)
	}
	if result := service.SelectCharacter(CharacterSelection{
		ConnectionID: observerConnection, SessionID: observer.SessionID, RequestID: "select-observer",
		BroadcastID: state.Broadcast.ID, CharacterID: state.Roster[1].ID,
	}); !result.Accepted {
		t.Fatalf("observer selection = %#v", result)
	}
	terminalID := "terminal-1"
	service.commit(func(root *domain.ProcessRuntime) transition {
		root.Broadcast.ActiveTerminalID = &terminalID
		root.Broadcast.TerminalRuntimes[terminalID] = testTerminalRuntime(terminalID)
		return transition{accepted: true}
	})
	return us2Fixture{
		service: service, effects: effects, broadcastID: state.Broadcast.ID, terminalID: terminalID,
		controllerConnection: controllerConnection, observerConnection: observerConnection, unassignedConnection: unassignedConnection,
		controllerSession: controller.SessionID, observerSession: observer.SessionID, unassignedSession: unassigned.SessionID,
		controllerToken: controller.BrowserToken, observerToken: observer.BrowserToken,
	}
}

type recordingTerminalRuntime struct {
	mu sync.Mutex

	calls       []domain.RuntimeCommand
	randomCalls int
	started     chan struct{}
	release     chan struct{}
	startOnce   sync.Once
}

func (runtime *recordingTerminalRuntime) Apply(state *domain.TerminalRuntime, command domain.RuntimeCommand) (*domain.PublicLiveState, bool) {
	runtime.mu.Lock()
	runtime.calls = append(runtime.calls, command)
	if command.Kind == domain.RuntimeCommandHackPattern {
		runtime.randomCalls++
	}
	started := runtime.started
	release := runtime.release
	runtime.mu.Unlock()
	if started != nil {
		runtime.startOnce.Do(func() {
			close(started)
			<-release
		})
	}

	switch command.Kind {
	case domain.RuntimeCommandNavAction:
		state.Nav = nav.ApplyAction(state.Nav, state.Tree, command.Action, command.NodeID)
	case domain.RuntimeCommandHackGuess:
		if state.Hack == nil || state.Hack.Solved || state.Hack.Failed {
			return nil, false
		}
		hack.ApplyGuess(state.Hack, command.TargetID)
	case domain.RuntimeCommandHackPattern:
		return nil, false
	default:
		return nil, false
	}
	return publicTerminalRuntime(state), true
}

func (runtime *recordingTerminalRuntime) Calls() int {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return len(runtime.calls)
}

func (runtime *recordingTerminalRuntime) RandomCalls() int {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.randomCalls
}

func actionResultForRequest(t *testing.T, effects *testutil.FakeOrderedEffectSink[Effect], requestID string) domain.ActionResult {
	t.Helper()
	recorded := effects.Values()
	for index := len(recorded) - 1; index >= 0; index-- {
		if recorded[index].Result != nil && recorded[index].Result.RequestID == requestID {
			return *recorded[index].Result
		}
	}
	t.Fatalf("no action result effect for request %q", requestID)
	return domain.ActionResult{}
}

func canonicalTerminalBytes(t *testing.T, service *Service, terminalID string) []byte {
	t.Helper()
	terminal := canonicalTerminal(t, service, terminalID)
	var usedPatterns []domain.HackPatternIdentity
	var privateHack *canonicalHackSnapshot
	if terminal.Hack != nil {
		hackState := terminal.Hack
		for identity := range hackState.UsedPatterns {
			usedPatterns = append(usedPatterns, identity)
		}
		sort.Slice(usedPatterns, func(left, right int) bool {
			if usedPatterns[left].GenerationID != usedPatterns[right].GenerationID {
				return usedPatterns[left].GenerationID < usedPatterns[right].GenerationID
			}
			if usedPatterns[left].Row != usedPatterns[right].Row {
				return usedPatterns[left].Row < usedPatterns[right].Row
			}
			if usedPatterns[left].Start != usedPatterns[right].Start {
				return usedPatterns[left].Start < usedPatterns[right].Start
			}
			return usedPatterns[left].End < usedPatterns[right].End
		})
		privateHack = &canonicalHackSnapshot{
			GenerationID: hackState.GenerationID, Level: hackState.Level, WordLength: hackState.WordLength,
			AttemptsMax: hackState.AttemptsMax, AttemptsLeft: hackState.AttemptsLeft,
			SecretWord: hackState.SecretWord, WordsByID: hackState.WordsByID, UsedPatterns: usedPatterns,
			Solved: hackState.Solved, Failed: hackState.Failed, Log: hackState.Log, Columns: hackState.Columns,
		}
		terminal.Hack = nil
	}
	encoded, err := json.Marshal(struct {
		Runtime *domain.TerminalRuntime
		Hack    *canonicalHackSnapshot
	}{Runtime: terminal, Hack: privateHack})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

type canonicalHackSnapshot struct {
	GenerationID string
	Level        int
	WordLength   int
	AttemptsMax  int
	AttemptsLeft int
	SecretWord   string
	WordsByID    map[string]domain.HackCandidate
	UsedPatterns []domain.HackPatternIdentity
	Solved       bool
	Failed       bool
	Log          []string
	Columns      []domain.HackColumn
}

func canonicalTerminal(t *testing.T, service *Service, terminalID string) *domain.TerminalRuntime {
	t.Helper()
	service.mu.RLock()
	defer service.mu.RUnlock()
	if service.runtime.Broadcast == nil || service.runtime.Broadcast.TerminalRuntimes[terminalID] == nil {
		t.Fatalf("canonical terminal %q is absent", terminalID)
	}
	return cloneTerminalRuntime(service.runtime.Broadcast.TerminalRuntimes[terminalID])
}

func setControllerForTest(service *Service, sessionID domain.LogicalSessionID) {
	service.commit(func(runtime *domain.ProcessRuntime) transition {
		controller := sessionID
		runtime.Broadcast.ControllerSessionID = &controller
		return transition{accepted: true}
	})
}

func testTerminalRuntime(terminalID string) *domain.TerminalRuntime {
	return &domain.TerminalRuntime{
		TerminalID: terminalID, TerminalName: "Overseer", Lifecycle: domain.TerminalLifecycleActive,
		Tree: domain.ContentNode{
			ID: "root", Type: domain.NodeFolder, Name: "ROOT",
			Children: []domain.ContentNode{{
				ID: "docs", Type: domain.NodeFolder, Name: "DOCS",
			}},
		},
		Nav:       domain.NavState{Path: []string{"root"}, Mode: "list"},
		HackLevel: 1,
		Hack: &domain.HackState{
			GenerationID: "generation-1", Level: 1, WordLength: 5,
			AttemptsMax: 4, AttemptsLeft: 4, SecretWord: "ALPHA",
			WordsByID: map[string]domain.HackCandidate{
				"candidate-secret": {Text: "ALPHA"},
				"candidate-wrong":  {Text: "BRAVO"},
			},
			UsedPatterns: make(map[domain.HackPatternIdentity]struct{}),
			Log:          []string{},
		},
	}
}

func publicTerminalRuntime(state *domain.TerminalRuntime) *domain.PublicLiveState {
	return &domain.PublicLiveState{
		TerminalID: state.TerminalID, TerminalName: state.TerminalName,
		Tree: state.Tree, HackLevel: state.HackLevel, IntroText: state.IntroText,
		Nav: state.Nav, Hack: hack.PublicState(state.Hack),
	}
}

func newUS1Service() *Service {
	return New(Config{IDs: &counterIDSource{}})
}

func assertExclusiveClaimInvariants(t *testing.T, state *domain.MasterCoordinationState) {
	t.Helper()
	claimBySession := make(map[domain.LogicalSessionID]domain.CharacterID)
	for _, character := range state.Roster {
		if character.ClaimedBySessionID == nil {
			continue
		}
		if previous, duplicate := claimBySession[*character.ClaimedBySessionID]; duplicate {
			t.Fatalf("session %q claims both %q and %q", *character.ClaimedBySessionID, previous, character.ID)
		}
		claimBySession[*character.ClaimedBySessionID] = character.ID
	}
	for _, session := range state.Sessions {
		claimed, hasClaim := claimBySession[session.ID]
		if session.Character == nil && hasClaim {
			t.Fatalf("roster claim for %q is missing from session projection", session.ID)
		}
		if session.Character != nil && (!hasClaim || claimed != session.Character.ID) {
			t.Fatalf("session claim for %q disagrees with roster: session=%#v roster=%#v", session.ID, session.Character, claimBySession)
		}
	}
	if state.Broadcast != nil && state.Broadcast.ControllerSessionID != nil {
		controller := masterSession(t, state, *state.Broadcast.ControllerSessionID)
		if controller.Character == nil || controller.Role != domain.PlayerRoleActive {
			t.Fatalf("controller is not an active assigned session: %#v", controller)
		}
	}
}

func masterSession(t *testing.T, state *domain.MasterCoordinationState, sessionID domain.LogicalSessionID) domain.MasterSessionEntry {
	t.Helper()
	for _, session := range state.Sessions {
		if session.ID == sessionID {
			return session
		}
	}
	t.Fatalf("master state has no logical session %q", sessionID)
	return domain.MasterSessionEntry{}
}

func claimedRosterCount(state *domain.MasterCoordinationState) int {
	count := 0
	for _, character := range state.Roster {
		if character.ClaimedBySessionID != nil {
			count++
		}
	}
	return count
}

func activeSessionCount(state *domain.MasterCoordinationState) int {
	return sessionRoleCount(state, domain.PlayerRoleActive)
}

func observerSessionCount(state *domain.MasterCoordinationState) int {
	return sessionRoleCount(state, domain.PlayerRoleObserver)
}

func sessionRoleCount(state *domain.MasterCoordinationState, role domain.PlayerRole) int {
	count := 0
	for _, session := range state.Sessions {
		if session.Role == role {
			count++
		}
	}
	return count
}

type counterIDSource struct {
	next atomic.Uint64
}

func (source *counterIDSource) Next() string {
	return fmt.Sprintf("opaque-%d", source.next.Add(1))
}

type sequenceIDSource struct {
	mu     sync.Mutex
	values []string
	next   int
}

func (source *sequenceIDSource) Next() string {
	source.mu.Lock()
	defer source.mu.Unlock()
	if source.next >= len(source.values) {
		panic("sequence ID source exhausted")
	}
	value := source.values[source.next]
	source.next++
	return value
}

func testLiveState(id string) *domain.PublicLiveState {
	return &domain.PublicLiveState{
		TerminalID:   id,
		TerminalName: "canonical",
		Tree: domain.ContentNode{
			ID:   "root",
			Type: domain.NodeFolder,
			Name: "ROOT",
			Children: []domain.ContentNode{{
				ID:   "docs",
				Type: domain.NodeFolder,
				Name: "DOCS",
			}},
		},
		Nav: domain.NavState{Path: []string{"root"}, Mode: "list"},
		Hack: &domain.PublicHackState{
			Log: []string{"ENTRY DENIED"},
			Columns: []domain.HackColumn{{
				Words: []domain.HackWord{{ID: "candidate-1", Start: 0, Length: 5}},
			}},
		},
	}
}

func TestPlayerConfigRosterInstallAndSaveBeforePublication(t *testing.T) {
	t.Parallel()

	store := &fakeRosterStore{}
	effects := testutil.NewFakeOrderedEffectSink[Effect]()
	service := New(Config{IDs: &counterIDSource{}, Enqueue: effects.Enqueue, RosterStore: store})

	if _, err := service.AddCharacter("No Store Yet"); err == nil {
		t.Fatal("AddCharacter() without an active player config succeeded")
	}
	if got := service.Snapshot(); len(got.Roster) != 0 || got.Revision != 0 {
		t.Fatalf("failed add changed state: %#v", got)
	}

	handle := domain.PlayerConfigHandle{Path: "/Campaigns/players.json", Version: 1, Name: "Players"}
	emptyInstalled, err := service.InstallPlayerConfig(handle, []domain.CharacterRosterEntry{})
	if err != nil {
		t.Fatalf("InstallPlayerConfig() empty roster error = %v", err)
	}
	if emptyInstalled.PlayerConfig == nil || len(emptyInstalled.Roster) != 0 {
		t.Fatalf("installed empty player config = %#v", emptyInstalled)
	}
	beforeNil := service.Snapshot()
	if _, err := service.InstallPlayerConfig(handle, nil); err == nil || err.Error() != "roster must be an array" {
		t.Fatalf("InstallPlayerConfig() nil roster error = %v, want roster array validation", err)
	}
	if afterNil := service.Snapshot(); !reflect.DeepEqual(afterNil, beforeNil) {
		t.Fatalf("nil roster changed coordinator\nbefore=%#v\nafter=%#v", beforeNil, afterNil)
	}

	roster := []domain.CharacterRosterEntry{{ID: "mara", Name: "Mara"}, {ID: "boone", Name: "Boone"}}
	installed, err := service.InstallPlayerConfig(handle, roster)
	if err != nil {
		t.Fatalf("InstallPlayerConfig() error = %v", err)
	}
	if installed.PlayerConfig == nil || installed.PlayerConfig.Name != "Players" || len(installed.Roster) != 2 {
		t.Fatalf("installed state = %#v", installed)
	}

	store.fail = true
	before := service.Snapshot()
	effectsBefore := effects.Calls()
	if _, err := service.RenameCharacter("mara", "Mara Voss"); err == nil {
		t.Fatal("RenameCharacter() with failed persistence succeeded")
	}
	if after := service.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("failed persistence changed coordinator\nbefore=%#v\nafter=%#v", before, after)
	}
	if effects.Calls() != effectsBefore {
		t.Fatalf("failed persistence published %d effects", effects.Calls()-effectsBefore)
	}

	store.fail = false
	renamed, err := service.RenameCharacter("mara", "Mara Voss")
	if err != nil || renamed.Roster[0].Name != "Mara Voss" {
		t.Fatalf("RenameCharacter() = state %#v, error %v", renamed, err)
	}
	if len(store.saves) != 1 || store.saves[0].Roster[0].Name != "Mara Voss" {
		t.Fatalf("persisted candidates = %#v", store.saves)
	}
	added, err := service.AddCharacter("Arcade")
	if err != nil || len(added.Roster) != 3 {
		t.Fatalf("AddCharacter() = state %#v, error %v", added, err)
	}
	deleted, err := service.DeleteCharacter("boone")
	if err != nil || len(deleted.Roster) != 2 {
		t.Fatalf("DeleteCharacter() = state %#v, error %v", deleted, err)
	}
	if len(store.saves) != 3 {
		t.Fatalf("successful roster mutations saved %d candidates, want 3", len(store.saves))
	}
	if got := store.saves[2].Roster; len(got) != 2 || got[0].ID != "mara" || got[1].Name != "Arcade" {
		t.Fatalf("final persisted ordered roster = %#v", got)
	}
}

func TestPlayerConfigReplacementRequiresNoBroadcastAndPreservesRuntimeOnFailure(t *testing.T) {
	t.Parallel()

	store := &fakeRosterStore{}
	service := New(Config{IDs: &counterIDSource{}, RosterStore: store})
	handle := domain.PlayerConfigHandle{Path: "/Campaigns/players.json", Version: 1, Name: "Players"}
	if _, err := service.InstallPlayerConfig(handle, []domain.CharacterRosterEntry{{ID: "mara", Name: "Mara"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartBroadcast(); err != nil {
		t.Fatal(err)
	}
	before := service.Snapshot()
	if _, err := service.InstallPlayerConfig(domain.PlayerConfigHandle{Path: "/Campaigns/other.json", Version: 1, Name: "Other"}, nil); err == nil {
		t.Fatal("InstallPlayerConfig() during broadcast succeeded")
	}
	if after := service.Snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("rejected replacement changed state\nbefore=%#v\nafter=%#v", before, after)
	}
}

type fakeRosterStore struct {
	fail  bool
	saves []domain.PlayerConfig
}

func (store *fakeRosterStore) Save(handle domain.PlayerConfigHandle, roster []domain.CharacterRosterEntry) error {
	if store.fail {
		return errors.New("injected player-config write failure")
	}
	store.saves = append(store.saves, domain.PlayerConfig{
		Version: handle.Version,
		Name:    handle.Name,
		Roster:  append([]domain.CharacterRosterEntry(nil), roster...),
	})
	return nil
}
