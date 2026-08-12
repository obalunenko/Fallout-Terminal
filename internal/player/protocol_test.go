package player

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/hack"
)

func TestDecodeClientMessageAcceptsExactProtocol(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  ClientMessage
	}{
		{name: "new session hello", input: `{"type":"SESSION_HELLO"}`, want: ClientMessage{Type: MessageSessionHello}},
		{name: "recognized session hello", input: `{"type":"SESSION_HELLO","browserToken":"token-1"}`, want: ClientMessage{Type: MessageSessionHello, BrowserToken: "token-1"}},
		{name: "character selection", input: `{"type":"CHARACTER_SELECT","requestId":"r1","broadcastId":"b1","characterId":"c1"}`, want: ClientMessage{Type: MessageCharacterSelect, RequestID: "r1", BroadcastID: "b1", CharacterID: "c1"}},
		{name: "navigation back", input: `{"type":"NAV_ACTION","requestId":"r2","broadcastId":"b1","terminalId":"t1","action":"back"}`, want: ClientMessage{Type: MessageNavAction, RequestID: "r2", BroadcastID: "b1", TerminalID: "t1", Action: "back"}},
		{name: "navigation enter", input: `{"type":"NAV_ACTION","requestId":"r3","broadcastId":"b1","terminalId":"t1","action":"enter","nodeId":"folder-1"}`, want: ClientMessage{Type: MessageNavAction, RequestID: "r3", BroadcastID: "b1", TerminalID: "t1", Action: "enter", NodeID: "folder-1"}},
		{name: "navigation command", input: `{"type":"NAV_ACTION","requestId":"r4","broadcastId":"b1","terminalId":"t1","action":"command","nodeId":"command-1"}`, want: ClientMessage{Type: MessageNavAction, RequestID: "r4", BroadcastID: "b1", TerminalID: "t1", Action: "command", NodeID: "command-1"}},
		{name: "navigation entry", input: `{"type":"NAV_ACTION","requestId":"r5","broadcastId":"b1","terminalId":"t1","action":"entry","nodeId":"entry-1"}`, want: ClientMessage{Type: MessageNavAction, RequestID: "r5", BroadcastID: "b1", TerminalID: "t1", Action: "entry", NodeID: "entry-1"}},
		{name: "word guess", input: `{"type":"HACK_GUESS","requestId":"r6","broadcastId":"b1","terminalId":"t1","targetId":"A1"}`, want: ClientMessage{Type: MessageHackGuess, RequestID: "r6", BroadcastID: "b1", TerminalID: "t1", TargetID: "A1"}},
		{name: "filler guess", input: `{"type":"HACK_GUESS","requestId":"r7","broadcastId":"b1","terminalId":"t1","targetId":"0:0"}`, want: ClientMessage{Type: MessageHackGuess, RequestID: "r7", BroadcastID: "b1", TerminalID: "t1", TargetID: "0:0"}},
		{name: "generation-bound special pattern", input: `{"type":"HACK_PATTERN","requestId":"r8","broadcastId":"b1","terminalId":"t1","patternId":"Z2VuZXJhdGlvbi0xADAAMQA0"}`, want: ClientMessage{Type: MessageHackPattern, RequestID: "r8", BroadcastID: "b1", TerminalID: "t1", PatternID: "Z2VuZXJhdGlvbi0xADAAMQA0"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := DecodeClientMessage(strings.NewReader(test.input))
			if err != nil {
				t.Fatalf("DecodeClientMessage() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("DecodeClientMessage() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestDecodeClientMessageRejectsMalformedOrUnsupportedObjects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "malformed", input: `{"type":`},
		{name: "array", input: `[{"type":"HACK_ADMIN"}]`},
		{name: "null", input: `null`},
		{name: "scalar", input: `"HACK_ADMIN"`},
		{name: "missing type", input: `{}`},
		{name: "non-string type", input: `{"type":7}`},
		{name: "unknown type", input: `{"type":"DELETE_SESSION"}`},
		{name: "removed administrator type", input: `{"type":"HACK_ADMIN"}`},
		{name: "server type from client", input: `{"type":"TERMINAL_LIVE"}`},
		{name: "unknown field", input: `{"type":"HACK_ADMIN","canonicalState":true}`},
		{name: "duplicate field", input: `{"type":"HACK_ADMIN","type":"HACK_GUESS","targetId":"A1"}`},
		{name: "second object", input: `{"type":"HACK_ADMIN"}{"type":"HACK_ADMIN"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if message, err := DecodeClientMessage(strings.NewReader(test.input)); err == nil {
				t.Fatalf("DecodeClientMessage() = %#v, want error", message)
			}
		})
	}
}

func TestDecodeClientMessageValidatesActionAndTargetFields(t *testing.T) {
	t.Parallel()

	invalid := []string{
		`{"type":"SESSION_HELLO","browserToken":""}`,
		`{"type":"SESSION_HELLO","browserToken":"token","sessionId":"client-selected"}`,
		`{"type":"CHARACTER_SELECT","requestId":"r1","broadcastId":"b1"}`,
		`{"type":"CHARACTER_SELECT","requestId":"r1","broadcastId":"b1","characterId":""}`,
		`{"type":"NAV_ACTION"}`,
		`{"type":"NAV_ACTION","action":7}`,
		`{"type":"NAV_ACTION","action":"delete","nodeId":"folder-1"}`,
		`{"type":"NAV_ACTION","action":"enter"}`,
		`{"type":"NAV_ACTION","action":"command","nodeId":""}`,
		`{"type":"NAV_ACTION","action":"entry","nodeId":"   "}`,
		`{"type":"HACK_GUESS"}`,
		`{"type":"HACK_GUESS","targetId":1}`,
		`{"type":"HACK_GUESS","targetId":""}`,
		`{"type":"HACK_GUESS","targetId":"   "}`,
		`{"type":"HACK_PATTERN"}`,
		`{"type":"HACK_PATTERN","patternId":1}`,
		`{"type":"HACK_PATTERN","patternId":""}`,
		`{"type":"HACK_PATTERN","patternId":"   "}`,
		`{"type":"HACK_PATTERN","patternId":"opaque","targetId":"A1"}`,
		`{"type":"HACK_PATTERN","patternId":"opaque","generationId":"client-supplied"}`,
		`{"type":"HACK_PATTERN","patternId":"first","patternId":"second"}`,
		`{"type":"HACK_ADMIN","targetId":"A1"}`,
		`{"type":"NAV_ACTION","requestId":"r1","broadcastId":"b1","terminalId":"t1","action":"back","requestId":"r2"}`,
		`{"type":"HACK_GUESS","requestId":"r1","broadcastId":"b1","terminalId":"t1","targetId":"A1","controller":true}`,
	}

	for _, input := range invalid {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if message, err := DecodeClientMessage(strings.NewReader(input)); err == nil {
				t.Fatalf("DecodeClientMessage() = %#v, want validation error", message)
			}
		})
	}
}

func TestSessionAndActionEnvelopeConstructors(t *testing.T) {
	t.Parallel()

	state := &domain.PlayerState{
		Revision:         7,
		SessionID:        "session-1",
		FallbackName:     "PLAYER 1",
		Role:             domain.PlayerRoleUnassigned,
		Phase:            domain.PlayerPhaseSelecting,
		BroadcastID:      "broadcast-1",
		ActiveTerminalID: "terminal-1",
		Roster: []domain.PlayerRosterEntry{
			{ID: "character-1", Name: "Mara", Status: domain.RosterStatusAvailable},
		},
	}
	welcome := NewSessionWelcomeEnvelope("browser-token", state)
	assertJSONEqual(t, marshalEnvelope(t, welcome), []byte(`{
		"type":"SESSION_WELCOME",
		"browserToken":"browser-token",
		"state":{"revision":7,"sessionId":"session-1","fallbackName":"PLAYER 1","character":null,"role":"unassigned","phase":"selecting","broadcastId":"broadcast-1","activeTerminalId":"terminal-1","roster":[{"id":"character-1","name":"Mara","status":"available"}]}
	}`))

	assertJSONEqual(t, marshalEnvelope(t, NewPlayerStateEnvelope(state)), []byte(`{
		"type":"PLAYER_STATE",
		"state":{"revision":7,"sessionId":"session-1","fallbackName":"PLAYER 1","character":null,"role":"unassigned","phase":"selecting","broadcastId":"broadcast-1","activeTerminalId":"terminal-1","roster":[{"id":"character-1","name":"Mara","status":"available"}]}
	}`))

	result := domain.ActionResult{RequestID: "request-1", Accepted: false, Reason: domain.ActionReasonNotController, Revision: 7}
	assertJSONEqual(t, marshalEnvelope(t, NewActionResultEnvelope(result)), []byte(`{
		"type":"ACTION_RESULT","requestId":"request-1","accepted":false,"reason":"not-controller","revision":7
	}`))
}

func TestUserStoryOneHandshakeSelectionAndStateShapes(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name  string
		input string
		want  ClientMessage
	}{
		{
			name:  "first-use hello",
			input: `{"type":"SESSION_HELLO"}`,
			want:  ClientMessage{Type: MessageSessionHello},
		},
		{
			name:  "recognized-profile hello",
			input: `{"type":"SESSION_HELLO","browserToken":"opaque-browser-token"}`,
			want:  ClientMessage{Type: MessageSessionHello, BrowserToken: "opaque-browser-token"},
		},
		{
			name:  "broadcast-scoped character selection",
			input: `{"type":"CHARACTER_SELECT","requestId":"selection-1","broadcastId":"broadcast-7","characterId":"character-2"}`,
			want: ClientMessage{
				Type: MessageCharacterSelect, RequestID: "selection-1", BroadcastID: "broadcast-7", CharacterID: "character-2",
			},
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := DecodeClientMessage(strings.NewReader(test.input))
			if err != nil {
				t.Fatalf("DecodeClientMessage() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("DecodeClientMessage() = %#v, want %#v", got, test.want)
			}
		})
	}

	unassigned := &domain.PlayerState{
		Revision:         17,
		SessionID:        "session-3",
		FallbackName:     "PLAYER 3",
		Role:             domain.PlayerRoleUnassigned,
		Phase:            domain.PlayerPhaseSelecting,
		BroadcastID:      "broadcast-7",
		ActiveTerminalID: "terminal-1",
		Roster: []domain.PlayerRosterEntry{
			{ID: "character-1", Name: "Mara", Status: domain.RosterStatusAvailable},
			{ID: "character-2", Name: "Boone", Status: domain.RosterStatusClaimed},
		},
	}
	assertJSONEqual(t, marshalEnvelope(t, NewSessionWelcomeEnvelope("opaque-browser-token", unassigned)), []byte(`{
		"type":"SESSION_WELCOME",
		"browserToken":"opaque-browser-token",
		"state":{
			"revision":17,
			"sessionId":"session-3",
			"fallbackName":"PLAYER 3",
			"character":null,
			"role":"unassigned",
			"phase":"selecting",
			"broadcastId":"broadcast-7",
			"activeTerminalId":"terminal-1",
			"roster":[
				{"id":"character-1","name":"Mara","status":"available"},
				{"id":"character-2","name":"Boone","status":"claimed"}
			]
		}
	}`))

	assigned := &domain.PlayerState{
		Revision:         18,
		SessionID:        "session-3",
		FallbackName:     "PLAYER 3",
		Character:        &domain.PlayerCharacter{ID: "character-1", Name: "Mara"},
		Role:             domain.PlayerRoleActive,
		Phase:            domain.PlayerPhaseControlling,
		BroadcastID:      "broadcast-7",
		ActiveTerminalID: "terminal-1",
		Roster: []domain.PlayerRosterEntry{
			{ID: "character-1", Name: "Mara", Status: domain.RosterStatusClaimed},
			{ID: "character-2", Name: "Boone", Status: domain.RosterStatusClaimed},
		},
	}
	assertJSONEqual(t, marshalEnvelope(t, NewPlayerStateEnvelope(assigned)), []byte(`{
		"type":"PLAYER_STATE",
		"state":{
			"revision":18,
			"sessionId":"session-3",
			"fallbackName":"PLAYER 3",
			"character":{"id":"character-1","name":"Mara"},
			"role":"active",
			"phase":"controlling",
			"broadcastId":"broadcast-7",
			"activeTerminalId":"terminal-1",
			"roster":[
				{"id":"character-1","name":"Mara","status":"claimed"},
				{"id":"character-2","name":"Boone","status":"claimed"}
			]
		}
	}`))

	assertJSONEqual(t, marshalEnvelope(t, NewActionResultEnvelope(domain.ActionResult{
		RequestID: "selection-1", Accepted: true, Reason: domain.ActionReason("accepted"), Revision: 18,
	})), []byte(`{"type":"ACTION_RESULT","requestId":"selection-1","accepted":true,"reason":"accepted","revision":18}`))
	assertJSONEqual(t, marshalEnvelope(t, NewActionResultEnvelope(domain.ActionResult{
		RequestID: "selection-stale", Accepted: false, Reason: domain.ActionReasonStaleBroadcast, Revision: 18,
	})), []byte(`{"type":"ACTION_RESULT","requestId":"selection-stale","accepted":false,"reason":"stale-broadcast","revision":18}`))
}

func TestUserStoryOnePlayerRosterProjectionIsPrivate(t *testing.T) {
	t.Parallel()

	state := &domain.PlayerState{
		Revision:     21,
		SessionID:    "session-1",
		FallbackName: "TABLET LEFT",
		Role:         domain.PlayerRoleUnassigned,
		Phase:        domain.PlayerPhaseSelecting,
		BroadcastID:  "broadcast-8",
		Roster: []domain.PlayerRosterEntry{
			{ID: "character-1", Name: "Mara", Status: domain.RosterStatusClaimed},
		},
	}
	raw := marshalEnvelope(t, NewPlayerStateEnvelope(state))
	assertJSONFieldsAbsent(t, raw,
		"browserToken", "claimedBySessionId", "claimantSessionId", "connected",
		"connectionId", "connectionIds", "controllerSessionId", "pendingSwitch",
		"sessions", "secretWord", "wordsById", "usedPatterns",
	)
	assertJSONEqual(t, raw, []byte(`{
		"type":"PLAYER_STATE",
		"state":{
			"revision":21,
			"sessionId":"session-1",
			"fallbackName":"TABLET LEFT",
			"character":null,
			"role":"unassigned",
			"phase":"selecting",
			"broadcastId":"broadcast-8",
			"activeTerminalId":null,
			"roster":[{"id":"character-1","name":"Mara","status":"claimed"}]
		}
	}`))
}

func TestUserStoryOneRejectsUnknownAndDuplicateHandshakeOrSelectionFields(t *testing.T) {
	t.Parallel()

	for _, input := range []string{
		`{"type":"SESSION_HELLO","browserToken":"token","sessionId":"client-chosen"}`,
		`{"type":"SESSION_HELLO","browserToken":"first","browserToken":"second"}`,
		`{"type":"SESSION_HELLO","type":"CHARACTER_SELECT"}`,
		`{"type":"CHARACTER_SELECT","requestId":"r1","broadcastId":"b1","characterId":"c1","sessionId":"s1"}`,
		`{"type":"CHARACTER_SELECT","requestId":"r1","requestId":"r2","broadcastId":"b1","characterId":"c1"}`,
		`{"type":"CHARACTER_SELECT","requestId":"r1","broadcastId":"b1","broadcastId":"b2","characterId":"c1"}`,
		`{"type":"CHARACTER_SELECT","requestId":"r1","broadcastId":"b1","characterId":"c1","characterId":"c2"}`,
		`{"type":"CHARACTER_SELECT","requestId":"r1","broadcastId":"b1","characterId":"c1","controller":true}`,
	} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			if message, err := DecodeClientMessage(strings.NewReader(input)); err == nil {
				t.Fatalf("DecodeClientMessage() = %#v, want strict-field error", message)
			}
		})
	}
}

func TestDecodeClientMessageEnforcesReadLimit(t *testing.T) {
	t.Parallel()

	payload := fmt.Sprintf(`{"type":"HACK_GUESS","targetId":"%s"}`, strings.Repeat("A", MaxClientMessageBytes))
	message, err := DecodeClientMessage(strings.NewReader(payload))
	if !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("DecodeClientMessage() = %#v, %v; want ErrMessageTooLarge", message, err)
	}
}

func TestServerEnvelopeConstructorsMatchGoldenProtocol(t *testing.T) {
	t.Parallel()

	t.Run("TERMINAL_LIVE", func(t *testing.T) {
		var state domain.PublicLiveState
		fixture := readProtocolFixture(t, "terminal-live.json")
		mustUnmarshal(t, fixture, &state)
		expected := jsonWithFields(t, fixture, map[string]any{"revision": float64(9)})
		assertJSONEqual(t, marshalEnvelope(t, NewTerminalLiveEnvelope(9, &state)), expected)
	})

	t.Run("TERMINAL_UPDATE", func(t *testing.T) {
		var state domain.PublicLiveState
		fixture := readProtocolFixture(t, "terminal-update.json")
		mustUnmarshal(t, fixture, &state)
		state.TerminalID = "t_demo1"
		expected := jsonWithFields(t, fixture, map[string]any{"revision": float64(10), "terminalId": "t_demo1"})
		assertJSONEqual(t, marshalEnvelope(t, NewTerminalUpdateEnvelope(10, &state)), expected)
	})

	t.Run("TERMINAL_CLEAR", func(t *testing.T) {
		fixture := readProtocolFixture(t, "terminal-clear.json")
		expected := jsonWithFields(t, fixture, map[string]any{"revision": float64(11)})
		assertJSONEqual(t, marshalEnvelope(t, NewTerminalClearEnvelope(11)), expected)
	})

	t.Run("NAV_STATE", func(t *testing.T) {
		var payload struct {
			Nav domain.NavState `json:"nav"`
		}
		fixture := readProtocolFixture(t, "nav-state.json")
		mustUnmarshal(t, fixture, &payload)
		expected := jsonWithFields(t, fixture, map[string]any{"revision": float64(12), "terminalId": "t_demo1"})
		assertJSONEqual(t, marshalEnvelope(t, NewNavStateEnvelope(12, "t_demo1", &payload.Nav)), expected)
	})

	t.Run("HACK_STATE", func(t *testing.T) {
		var payload struct {
			Hack domain.PublicHackState `json:"hack"`
		}
		fixture := readProtocolFixture(t, "hack-state.json")
		mustUnmarshal(t, fixture, &payload)
		expected := jsonWithFields(t, fixture, map[string]any{"revision": float64(13), "terminalId": "t_demo1"})
		assertJSONEqual(t, marshalEnvelope(t, NewHackStateEnvelope(13, "t_demo1", &payload.Hack)), expected)
	})
}

func TestPlayerEnvelopesNeverMarshalPrivateHackFields(t *testing.T) {
	t.Parallel()

	private := &domain.HackState{
		GenerationID: "generation-private",
		Level:        2,
		WordLength:   5,
		AttemptsMax:  4,
		AttemptsLeft: 4,
		SecretWord:   "VAULT",
		WordsByID:    map[string]domain.HackCandidate{"A1": {Text: "VAULT"}},
		UsedPatterns: map[domain.HackPatternIdentity]struct{}{{GenerationID: "generation-private", Row: 0, Start: 0, End: 1}: {}},
		Columns:      []domain.HackColumn{},
		Log:          []string{},
	}
	public := hack.PublicState(private)
	live := &domain.PublicLiveState{
		TerminalID: "terminal-1", TerminalName: "Overseer", HackLevel: 2,
		Tree: domain.ContentNode{ID: "root", Type: domain.NodeFolder, Name: "ROOT", Children: []domain.ContentNode{}},
		Nav:  domain.NavState{Path: []string{"root"}, Mode: "list"},
		Hack: public,
	}

	for name, envelope := range map[string]any{
		"terminal live": NewTerminalLiveEnvelope(1, live),
		"hack state":    NewHackStateEnvelope(1, "terminal-1", public),
	} {
		raw := marshalEnvelope(t, envelope)
		for _, forbidden := range []string{"secretWord", "wordsById", "usedPatterns", "adminModeUsed", "isAdmin"} {
			if strings.Contains(string(raw), forbidden) {
				t.Errorf("%s leaked %q: %s", name, forbidden, raw)
			}
		}
	}
}

func jsonWithFields(t *testing.T, raw []byte, fields map[string]any) []byte {
	t.Helper()
	var object map[string]any
	mustUnmarshal(t, raw, &object)
	for name, value := range fields {
		object[name] = value
	}
	updated, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func readProtocolFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("..", "testutil", "testdata", "protocol", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}

func mustUnmarshal(t *testing.T, raw []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("json.Unmarshal(%s): %v", raw, err)
	}
}

func marshalEnvelope(t *testing.T, envelope any) []byte {
	t.Helper()
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", envelope, err)
	}
	return raw
}

func assertJSONEqual(t *testing.T, actual, expected []byte) {
	t.Helper()
	var actualValue any
	var expectedValue any
	mustUnmarshal(t, actual, &actualValue)
	mustUnmarshal(t, expected, &expectedValue)
	if !reflect.DeepEqual(actualValue, expectedValue) {
		t.Fatalf("JSON mismatch\nactual:   %s\nexpected: %s", actual, expected)
	}
}

func assertJSONFieldsAbsent(t *testing.T, raw []byte, forbidden ...string) {
	t.Helper()
	var value any
	mustUnmarshal(t, raw, &value)
	for _, field := range forbidden {
		if jsonContainsField(value, field) {
			t.Errorf("JSON exposes forbidden field %q: %s", field, raw)
		}
	}
}

func jsonContainsField(value any, forbidden string) bool {
	switch value := value.(type) {
	case map[string]any:
		for field, nested := range value {
			if field == forbidden || jsonContainsField(nested, forbidden) {
				return true
			}
		}
	case []any:
		for _, nested := range value {
			if jsonContainsField(nested, forbidden) {
				return true
			}
		}
	}
	return false
}
