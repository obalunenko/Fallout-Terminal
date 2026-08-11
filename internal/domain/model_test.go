package domain

import (
	"bytes"
	"encoding/json"
	"os"
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

func deepJSONEqual(left, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return bytes.Equal(leftJSON, rightJSON)
}
