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
		{name: "navigation back", input: `{"type":"NAV_ACTION","action":"back"}`, want: ClientMessage{Type: "NAV_ACTION", Action: "back"}},
		{name: "navigation enter", input: `{"type":"NAV_ACTION","action":"enter","nodeId":"folder-1"}`, want: ClientMessage{Type: "NAV_ACTION", Action: "enter", NodeID: "folder-1"}},
		{name: "navigation command", input: `{"type":"NAV_ACTION","action":"command","nodeId":"command-1"}`, want: ClientMessage{Type: "NAV_ACTION", Action: "command", NodeID: "command-1"}},
		{name: "navigation entry", input: `{"type":"NAV_ACTION","action":"entry","nodeId":"entry-1"}`, want: ClientMessage{Type: "NAV_ACTION", Action: "entry", NodeID: "entry-1"}},
		{name: "word guess", input: `{"type":"HACK_GUESS","targetId":"A1"}`, want: ClientMessage{Type: "HACK_GUESS", TargetID: "A1"}},
		{name: "filler guess", input: `{"type":"HACK_GUESS","targetId":"0:0"}`, want: ClientMessage{Type: "HACK_GUESS", TargetID: "0:0"}},
		{name: "stale guess remains syntactically eligible", input: `{"type":"HACK_GUESS","targetId":"stale"}`, want: ClientMessage{Type: "HACK_GUESS", TargetID: "stale"}},
		{name: "special pattern", input: `{"type":"HACK_PATTERN","patternId":"0:17:23"}`, want: ClientMessage{Type: "HACK_PATTERN", PatternID: "0:17:23"}},
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
		`{"type":"HACK_PATTERN","patternId":"0:17:23","targetId":"A1"}`,
		`{"type":"HACK_ADMIN","targetId":"A1"}`,
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
		expected := readProtocolFixture(t, "terminal-live.json")
		mustUnmarshal(t, expected, &state)
		assertJSONEqual(t, marshalEnvelope(t, NewTerminalLiveEnvelope(&state)), expected)
	})

	t.Run("TERMINAL_UPDATE", func(t *testing.T) {
		var state domain.PublicLiveState
		expected := readProtocolFixture(t, "terminal-update.json")
		mustUnmarshal(t, expected, &state)
		assertJSONEqual(t, marshalEnvelope(t, NewTerminalUpdateEnvelope(&state)), expected)
	})

	t.Run("TERMINAL_CLEAR", func(t *testing.T) {
		expected := readProtocolFixture(t, "terminal-clear.json")
		assertJSONEqual(t, marshalEnvelope(t, NewTerminalClearEnvelope()), expected)
	})

	t.Run("NAV_STATE", func(t *testing.T) {
		var payload struct {
			Nav domain.NavState `json:"nav"`
		}
		expected := readProtocolFixture(t, "nav-state.json")
		mustUnmarshal(t, expected, &payload)
		assertJSONEqual(t, marshalEnvelope(t, NewNavStateEnvelope(&payload.Nav)), expected)
	})

	t.Run("HACK_STATE", func(t *testing.T) {
		var payload struct {
			Hack domain.PublicHackState `json:"hack"`
		}
		expected := readProtocolFixture(t, "hack-state.json")
		mustUnmarshal(t, expected, &payload)
		assertJSONEqual(t, marshalEnvelope(t, NewHackStateEnvelope(&payload.Hack)), expected)
	})
}

func TestPlayerEnvelopesNeverMarshalPrivateHackFields(t *testing.T) {
	t.Parallel()

	private := &domain.HackState{
		Level:        2,
		WordLength:   5,
		AttemptsMax:  4,
		AttemptsLeft: 4,
		SecretWord:   "VAULT",
		WordsByID:    map[string]domain.HackCandidate{"A1": {Text: "VAULT"}},
		UsedPatterns: map[string]struct{}{"0:0:1": {}},
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
		"terminal live": NewTerminalLiveEnvelope(live),
		"hack state":    NewHackStateEnvelope(public),
	} {
		raw := marshalEnvelope(t, envelope)
		for _, forbidden := range []string{"secretWord", "wordsById", "usedPatterns", "adminModeUsed", "isAdmin"} {
			if strings.Contains(string(raw), forbidden) {
				t.Errorf("%s leaked %q: %s", name, forbidden, raw)
			}
		}
	}
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
