package domain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

var (
	sessionFields  = fieldSet("version", "name", "terminals")
	terminalFields = fieldSet("id", "name", "hackLevel", "introText", "root")
	nodeFields     = fieldSet("id", "type", "name", "children", "text", "description")
)

// DecodeSession decodes a version-1 document while retaining compatible unknown fields.
func DecodeSession(data []byte) (Session, error) {
	var session Session
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&session); err != nil {
		return Session{}, fmt.Errorf("decode session: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Session{}, fmt.Errorf("decode session: trailing JSON value")
		}
		return Session{}, fmt.Errorf("decode session: %w", err)
	}
	if err := ValidateSession(session); err != nil {
		return Session{}, fmt.Errorf("validate session: %w", err)
	}
	return session, nil
}

// EncodeSession emits stable, human-readable version-1 JSON with a final newline.
func EncodeSession(session Session) ([]byte, error) {
	if err := ValidateSession(session); err != nil {
		return nil, fmt.Errorf("validate session: %w", err)
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode session: %w", err)
	}
	return append(data, '\n'), nil
}

// UnmarshalJSON retains unknown top-level session fields.
func (s *Session) UnmarshalJSON(data []byte) error {
	type alias Session
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	s.Extra = extrasFrom(data, sessionFields)
	decoded.Extra = s.Extra
	*s = Session(decoded)
	return nil
}

// MarshalJSON restores unknown top-level session fields.
func (s Session) MarshalJSON() ([]byte, error) {
	type alias Session
	return marshalWithExtras(alias(s), s.Extra, sessionFields)
}

// UnmarshalJSON retains unknown terminal fields.
func (t *Terminal) UnmarshalJSON(data []byte) error {
	type alias Terminal
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	t.Extra = extrasFrom(data, terminalFields)
	decoded.Extra = t.Extra
	*t = Terminal(decoded)
	return nil
}

// MarshalJSON restores unknown terminal fields.
func (t Terminal) MarshalJSON() ([]byte, error) {
	type alias Terminal
	return marshalWithExtras(alias(t), t.Extra, terminalFields)
}

// UnmarshalJSON retains unknown fields on known content-node variants.
func (n *ContentNode) UnmarshalJSON(data []byte) error {
	type alias ContentNode
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	n.Extra = extrasFrom(data, nodeFields)
	decoded.Extra = n.Extra
	*n = ContentNode(decoded)
	return nil
}

// MarshalJSON restores unknown content-node fields and preserves folder children arrays.
func (n ContentNode) MarshalJSON() ([]byte, error) {
	type alias ContentNode
	data, err := marshalWithExtras(alias(n), n.Extra, nodeFields)
	if err != nil || n.Type != NodeFolder {
		return data, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	children, err := json.Marshal(n.Children)
	if err != nil {
		return nil, err
	}
	raw["children"] = children
	return json.Marshal(raw)
}

func fieldSet(fields ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		set[field] = struct{}{}
	}
	return set
}

func extrasFrom(data []byte, known map[string]struct{}) map[string]json.RawMessage {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return nil
	}
	for field := range known {
		delete(raw, field)
	}
	if len(raw) == 0 {
		return nil
	}
	return raw
}

func marshalWithExtras(value any, extras map[string]json.RawMessage, known map[string]struct{}) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(extras) == 0 {
		return data, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for field, value := range extras {
		if _, exists := known[field]; exists {
			return nil, fmt.Errorf("extra field %q shadows a known field", field)
		}
		raw[field] = value
	}
	return json.Marshal(raw)
}
