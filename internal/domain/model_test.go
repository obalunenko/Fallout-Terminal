package domain

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestDecodeEncodeSessionV1Fixture(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../testutil/testdata/session-v1.json")
	if err != nil {
		t.Fatal(err)
	}

	session, err := DecodeSession(raw)
	if err != nil {
		t.Fatalf("DecodeSession() error = %v", err)
	}
	if session.Version != 1 || len(session.Terminals) == 0 {
		t.Fatalf("decoded fixture = %#v", session)
	}

	encoded, err := EncodeSession(session)
	if err != nil {
		t.Fatalf("EncodeSession() error = %v", err)
	}
	if !bytes.HasSuffix(encoded, []byte("\n")) {
		t.Fatal("EncodeSession() must include a final newline")
	}

	var got, want any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if !deepJSONEqual(got, want) {
		t.Fatalf("semantic round trip changed fixture\ngot:  %s\nwant: %s", encoded, raw)
	}
}

func TestUnknownFieldsRoundTrip(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
  "version": 1,
  "name": "Extras",
  "campaignNote": {"keep": true},
  "terminals": [{
    "id": "t1",
    "name": "Terminal",
    "hackLevel": 0,
    "introText": "",
    "terminalNote": 42,
    "root": {
      "id": "root",
      "type": "folder",
      "name": "ROOT",
      "nodeNote": [1, 2],
      "children": []
    }
  }]
}`)

	session, err := DecodeSession(raw)
	if err != nil {
		t.Fatalf("DecodeSession() error = %v", err)
	}
	encoded, err := EncodeSession(session)
	if err != nil {
		t.Fatalf("EncodeSession() error = %v", err)
	}

	for _, field := range []string{"campaignNote", "terminalNote", "nodeNote"} {
		if !bytes.Contains(encoded, []byte(`"`+field+`"`)) {
			t.Errorf("round trip dropped %s: %s", field, encoded)
		}
	}
}

func TestVersionOneSessionNeverPersistsRuntimeHackAggregate(t *testing.T) {
	t.Parallel()

	session := Session{
		Version: 1,
		Name:    "Runtime boundary",
		Terminals: []Terminal{{
			ID: "terminal-1", Name: "Overseer", HackLevel: 3, IntroText: "WELCOME",
			Root: ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{}},
		}},
	}
	// This aggregate deliberately contains every category of process-local
	// hacking state. It is not part of Session or Terminal and therefore has no
	// path into the version-1 document.
	runtime := &HackState{
		GenerationID: "generation-runtime-only",
		Level:        3, WordLength: 6, AttemptsMax: 4, AttemptsLeft: 2,
		SecretWord: "CIPHER",
		WordsByID:  map[string]HackCandidate{"A1": {Text: "CIPHER"}, "A2": {Text: "BUNKER"}},
		UsedPatterns: map[HackPatternIdentity]struct{}{{
			GenerationID: "generation-runtime-only", Row: 4, Start: 2, End: 7,
		}: {}},
		Log:     []string{"Ложное слово удалено."},
		Columns: []HackColumn{{Text: "..CIPHER....", Words: []HackWord{{ID: "A1", Start: 2, Length: 6}}}},
	}
	if runtime.GenerationID == "" || runtime.AttemptsLeft == runtime.AttemptsMax {
		t.Fatal("runtime fixture did not contain progressed puzzle state")
	}

	encoded, err := EncodeSession(session)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{
		"generationId", "generation-runtime-only", "patterns", "usedPatterns",
		"removedDuds", "attemptsMax", "attemptsLeft", "outcomes", "unlocked",
		"puzzleSeed", "secretWord", "wordsById", "CIPHER", "Ложное слово удалено.",
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Errorf("version-1 session persisted runtime hacking value %q: %s", forbidden, encoded)
		}
	}

	decoded, err := DecodeSession(encoded)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := EncodeSession(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, reencoded) {
		t.Fatalf("version-1 round trip changed runtime-free document\nfirst: %s\nagain: %s", encoded, reencoded)
	}
}

func TestRuntimeCoordinationProjectionClonesDetachDeeply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		projection          any
		clone               func(any) any
		requiredFields      map[string]reflect.Kind
		forbiddenJSONFields []string
	}{
		{
			name:       "master coordination state",
			projection: &MasterCoordinationState{},
			clone: func(value any) any {
				return CloneMasterCoordinationState(value.(*MasterCoordinationState))
			},
			requiredFields: map[string]reflect.Kind{
				"Roster":        reflect.Slice,
				"Sessions":      reflect.Slice,
				"Broadcast":     reflect.Pointer,
				"PendingSwitch": reflect.Pointer,
			},
			forbiddenJSONFields: []string{
				"browserToken", "connectionId", "connectionIds", "requestResults",
				"secretWord", "wordsById", "usedPatterns", "hack",
			},
		},
		{
			name:       "personalized player state",
			projection: &PlayerState{},
			clone: func(value any) any {
				return ClonePlayerState(value.(*PlayerState))
			},
			requiredFields: map[string]reflect.Kind{
				"Character":        reflect.Pointer,
				"BroadcastID":      reflect.String,
				"ActiveTerminalID": reflect.String,
				"Roster":           reflect.Slice,
			},
			forbiddenJSONFields: []string{
				"browserToken", "sessions", "claimedBySessionId", "connected",
				"connectionId", "connectionIds", "requestResults", "pendingSwitch",
				"secretWord", "wordsById", "usedPatterns", "hack",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			projection := reflect.ValueOf(test.projection).Elem()
			for fieldName, wantKind := range test.requiredFields {
				field := projection.FieldByName(fieldName)
				if !field.IsValid() {
					t.Fatalf("%T is missing required projection field %s", test.projection, fieldName)
				}
				if field.Kind() != wantKind {
					t.Fatalf("%T.%s kind = %s, want %s", test.projection, fieldName, field.Kind(), wantKind)
				}
			}

			seedProjection(t, projection, test.name, 0)
			before, err := json.Marshal(test.projection)
			if err != nil {
				t.Fatalf("marshal seeded projection: %v", err)
			}
			var publicProjection any
			if err := json.Unmarshal(before, &publicProjection); err != nil {
				t.Fatalf("decode seeded projection: %v", err)
			}
			for _, forbidden := range test.forbiddenJSONFields {
				if containsJSONField(publicProjection, forbidden) {
					t.Errorf("projection exposes private field %q: %s", forbidden, before)
				}
			}

			clone := test.clone(test.projection)
			if !reflect.DeepEqual(test.projection, clone) {
				t.Fatalf("clone changed projection\nsource: %#v\nclone:  %#v", test.projection, clone)
			}
			if !mutateProjectionReferences(reflect.ValueOf(clone).Elem(), false) {
				t.Fatal("projection fixture contains no nested mutable references")
			}
			if reflect.DeepEqual(test.projection, clone) {
				t.Fatal("mutating clone did not change it")
			}

			after, err := json.Marshal(test.projection)
			if err != nil {
				t.Fatalf("marshal source after clone mutation: %v", err)
			}
			if !bytes.Equal(after, before) {
				t.Fatalf("clone retained an alias into its source\nbefore: %s\nafter:  %s", before, after)
			}
		})
	}
}

func TestVersionOneSessionJSONContainsOnlyDurableAuthoredFields(t *testing.T) {
	t.Parallel()

	session := Session{
		Version: 1,
		Name:    "Persistence boundary",
		Terminals: []Terminal{{
			ID:        "durable-terminal",
			Name:      "Overseer",
			HackLevel: 4,
			IntroText: "WELCOME",
			Root: ContentNode{
				ID:   "root",
				Type: NodeFolder,
				Name: "ROOT",
				Children: []ContentNode{{
					ID: "entry", Type: NodeEntry, Name: "STATUS", Description: "Authored content",
				}},
			},
		}},
	}

	encoded, err := EncodeSession(session)
	if err != nil {
		t.Fatalf("EncodeSession() error = %v", err)
	}

	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("decode encoded session: %v", err)
	}
	assertJSONFieldSet(t, document, "session", "name", "terminals", "version")

	terminals, ok := document["terminals"].([]any)
	if !ok || len(terminals) != 1 {
		t.Fatalf("terminals = %#v, want one terminal", document["terminals"])
	}
	terminal, ok := terminals[0].(map[string]any)
	if !ok {
		t.Fatalf("terminal = %#v, want object", terminals[0])
	}
	assertJSONFieldSet(t, terminal, "terminal", "hackLevel", "id", "introText", "name", "root")

	for _, forbidden := range []string{
		"browserToken", "sessionId", "logicalSessionId", "sessions", "fallbackName",
		"connectionId", "connectionIds", "connected", "roster", "characterId",
		"claimedBySessionId", "assignmentsBySession", "sessionByCharacter", "role",
		"controllerSessionId", "broadcastId", "activeTerminalId", "revision",
		"pendingSwitch", "switchId", "terminalRuntimes", "lifecycle", "nav", "hack",
		"generationId", "secretWord", "wordsById", "usedPatterns", "attemptsLeft",
		"board", "patterns", "log", "outcome",
	} {
		if containsJSONField(document, forbidden) {
			t.Errorf("version-1 session JSON contains process-local field %q: %s", forbidden, encoded)
		}
	}
}

func seedProjection(t *testing.T, value reflect.Value, path string, depth int) {
	t.Helper()
	if depth > 12 {
		t.Fatalf("projection type is unexpectedly recursive at %s", path)
	}

	switch value.Kind() {
	case reflect.Pointer:
		value.Set(reflect.New(value.Type().Elem()))
		seedProjection(t, value.Elem(), path+"*", depth+1)
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Field(index)
			if field.CanSet() {
				seedProjection(t, field, path+"."+value.Type().Field(index).Name, depth+1)
			}
		}
	case reflect.Slice:
		value.Set(reflect.MakeSlice(value.Type(), 1, 1))
		seedProjection(t, value.Index(0), path+"[0]", depth+1)
	case reflect.Map:
		value.Set(reflect.MakeMap(value.Type()))
		key := reflect.New(value.Type().Key()).Elem()
		seedProjection(t, key, path+".key", depth+1)
		element := reflect.New(value.Type().Elem()).Elem()
		seedProjection(t, element, path+".value", depth+1)
		value.SetMapIndex(key, element)
	case reflect.String:
		value.SetString(path)
	case reflect.Bool:
		value.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value.SetInt(int64(depth + 1))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value.SetUint(uint64(depth + 1))
	case reflect.Float32, reflect.Float64:
		value.SetFloat(float64(depth) + 0.5)
	}
}

func mutateProjectionReferences(value reflect.Value, behindReference bool) bool {
	mutated := false
	switch value.Kind() {
	case reflect.Pointer:
		if !value.IsNil() {
			return mutateProjectionReferences(value.Elem(), true)
		}
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Field(index)
			if field.CanSet() && mutateProjectionReferences(field, behindReference) {
				mutated = true
			}
		}
	case reflect.Slice:
		for index := 0; index < value.Len(); index++ {
			if mutateProjectionReferences(value.Index(index), true) {
				mutated = true
			}
		}
	case reflect.Map:
		iterator := value.MapRange()
		for iterator.Next() {
			element := reflect.New(value.Type().Elem()).Elem()
			element.Set(iterator.Value())
			if mutateProjectionReferences(element, true) {
				value.SetMapIndex(iterator.Key(), element)
				mutated = true
			}
		}
	case reflect.String:
		if behindReference {
			value.SetString(value.String() + "-mutated")
			mutated = true
		}
	case reflect.Bool:
		if behindReference {
			value.SetBool(!value.Bool())
			mutated = true
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if behindReference {
			value.SetInt(value.Int() + 1)
			mutated = true
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if behindReference {
			value.SetUint(value.Uint() + 1)
			mutated = true
		}
	case reflect.Float32, reflect.Float64:
		if behindReference {
			value.SetFloat(value.Float() + 1)
			mutated = true
		}
	}
	return mutated
}

func assertJSONFieldSet(t *testing.T, object map[string]any, location string, want ...string) {
	t.Helper()
	got := make([]string, 0, len(object))
	for field := range object {
		got = append(got, field)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s fields = %v, want %v", location, got, want)
	}
}

func containsJSONField(value any, forbidden string) bool {
	switch value := value.(type) {
	case map[string]any:
		for field, nested := range value {
			if field == forbidden || containsJSONField(nested, forbidden) {
				return true
			}
		}
	case []any:
		for _, nested := range value {
			if containsJSONField(nested, forbidden) {
				return true
			}
		}
	}
	return false
}

func deepJSONEqual(left, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return bytes.Equal(leftJSON, rightJSON)
}

func TestVersionOneEncodingIsInvariantAcrossCompleteProcessRuntimeActivity(t *testing.T) {
	session := Session{
		Version: 1, Name: "Runtime-neutral campaign",
		Terminals: []Terminal{{
			ID: "terminal-1", Name: "Overseer", HackLevel: 2, IntroText: "WELCOME",
			Root: ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{}},
		}},
	}
	before, err := EncodeSession(session)
	if err != nil {
		t.Fatal(err)
	}
	controller := LogicalSessionID("session-1")
	activeTerminal := "terminal-1"
	_ = ProcessRuntime{
		Revision: 99,
		SessionsByID: map[LogicalSessionID]*LogicalSession{
			controller: {ID: controller, FallbackName: "TABLET LEFT", ConnectionIDs: map[ConnectionID]struct{}{"connection-1": {}}, RequestResults: map[RequestID]RequestResultRecord{}},
		},
		SessionIDByBrowserToken: map[BrowserToken]LogicalSessionID{"opaque-token": controller},
		RosterByID:              map[CharacterID]*CharacterRosterEntry{"character-1": {ID: "character-1", Name: "Mara"}},
		RosterOrder:             []CharacterID{"character-1"},
		Broadcast: &LiveBroadcast{
			ID: "broadcast-1", AssignmentsBySession: map[LogicalSessionID]CharacterID{controller: "character-1"},
			SessionByCharacter: map[CharacterID]LogicalSessionID{"character-1": controller}, ControllerSessionID: &controller,
			ActiveTerminalID: &activeTerminal, TerminalRuntimes: map[string]*TerminalRuntime{
				"terminal-1": {TerminalID: "terminal-1", TerminalName: "Overseer", HackLevel: 2, Lifecycle: TerminalLifecycleActive, Hack: &HackState{GenerationID: "private-generation", SecretWord: "SECRET"}},
			},
		},
		PendingSwitch: &TerminalSwitchDecision{ID: "switch-1", BroadcastID: "broadcast-1", SourceTerminalID: "terminal-1"},
	}
	after, err := EncodeSession(session)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("runtime activity changed version-1 encoding:\nbefore %s\nafter  %s", before, after)
	}
	for _, forbidden := range []string{"browserToken", "fallbackName", "connection", "broadcast", "controller", "claim", "pendingSwitch", "generation", "secretWord", "terminalRuntimes"} {
		if strings.Contains(string(after), forbidden) {
			t.Fatalf("durable encoding leaked runtime field %q: %s", forbidden, after)
		}
	}
	decoded, err := DecodeSession(after)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Terminals) != 1 || decoded.Terminals[0].ID != "terminal-1" || decoded.Terminals[0].HackLevel != 2 {
		t.Fatalf("durable authored terminal changed after runtime activity: %#v", decoded)
	}
}

func TestSessionPlayerConfigReferenceIsOptionalAndRoundTrips(t *testing.T) {
	t.Parallel()

	legacy := validSessionForPlayerConfigTest()
	encoded, err := EncodeSession(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(`"playerConfig"`)) {
		t.Fatalf("legacy session unexpectedly gained playerConfig: %s", encoded)
	}

	legacy.PlayerConfig = filepath.Join("players", "vault-13.json")
	encoded, err = EncodeSession(legacy)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSession(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.PlayerConfig != legacy.PlayerConfig {
		t.Fatalf("playerConfig = %q, want %q", decoded.PlayerConfig, legacy.PlayerConfig)
	}
}

func TestPlayerConfigV1StrictValidationAndStableEncoding(t *testing.T) {
	t.Parallel()

	empty := PlayerConfig{Version: 1, Name: "Empty Players", Roster: []CharacterRosterEntry{}}
	emptyEncoded, err := EncodePlayerConfig(empty)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(emptyEncoded, []byte(`"roster": []`)) {
		t.Fatalf("empty player config must encode roster as an array: %s", emptyEncoded)
	}
	emptyDecoded, err := DecodePlayerConfig(emptyEncoded)
	if err != nil {
		t.Fatal(err)
	}
	if emptyDecoded.Roster == nil {
		t.Fatal("empty player config round trip produced a nil roster")
	}

	config := PlayerConfig{
		Version: 1,
		Name:    "Vault 13 Players",
		Roster: []CharacterRosterEntry{
			{ID: "mara", Name: "Mara"},
			{ID: "boone", Name: "Boone"},
		},
	}
	encoded, err := EncodePlayerConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodePlayerConfig(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, config) {
		t.Fatalf("round trip = %#v, want %#v", decoded, config)
	}
	if !bytes.HasSuffix(encoded, []byte("\n")) {
		t.Fatal("player config must end with a newline")
	}
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	assertJSONFieldSet(t, document, "player config", "name", "roster", "version")
	for _, forbidden := range []string{
		"browserToken", "sessionId", "connectionId", "connected", "claimedBySessionId",
		"controllerSessionId", "broadcastId", "revision", "requestId", "activeTerminalId",
		"nav", "hack", "secretWord", "attemptsLeft", "patterns", "log", "outcome",
	} {
		if containsJSONField(document, forbidden) {
			t.Errorf("player config contains runtime field %q: %s", forbidden, encoded)
		}
	}

	invalid := []string{
		`{"version":2,"name":"Players","roster":[]}`,
		`{"version":1,"name":" ","roster":[]}`,
		`{"version":1,"name":"Players","roster":null}`,
		`{"version":1,"name":"Players","roster":[{"id":"same","name":"One"},{"id":"same","name":"Two"}]}`,
		`{"version":1,"name":"Players","roster":[],"browserToken":"secret"}`,
	}
	for _, raw := range invalid {
		if _, err := DecodePlayerConfig([]byte(raw)); err == nil {
			t.Errorf("DecodePlayerConfig(%s) unexpectedly succeeded", raw)
		}
	}
}

func validSessionForPlayerConfigTest() Session {
	return Session{
		Version: 1,
		Name:    "Campaign",
		Terminals: []Terminal{{
			ID: "terminal-1", Name: "Terminal", HackLevel: 0,
			Root: ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{}},
		}},
	}
}
