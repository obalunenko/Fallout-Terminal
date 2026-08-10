package domain

import (
	"bytes"
	"encoding/json"
	"os"
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

func deepJSONEqual(left, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return bytes.Equal(leftJSON, rightJSON)
}
