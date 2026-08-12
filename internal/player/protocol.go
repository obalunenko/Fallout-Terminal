// Package player owns the HTTP and WebSocket boundary for the retained player UI.
package player

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
)

const (
	// MaxClientMessageBytes bounds player-controlled WebSocket input before it
	// reaches JSON decoding or canonical live-state services.
	MaxClientMessageBytes = 4 << 10

	MessageSessionHello    = "SESSION_HELLO"
	MessageCharacterSelect = "CHARACTER_SELECT"
	MessageNavAction       = "NAV_ACTION"
	MessageHackGuess       = "HACK_GUESS"
	MessageHackPattern     = "HACK_PATTERN"
	MessageTerminalLive    = "TERMINAL_LIVE"
	MessageTerminalUpdate  = "TERMINAL_UPDATE"
	MessageTerminalClear   = "TERMINAL_CLEAR"
	MessageNavState        = "NAV_STATE"
	MessageHackState       = "HACK_STATE"
	MessageSessionWelcome  = "SESSION_WELCOME"
	MessagePlayerState     = "PLAYER_STATE"
	MessageActionResult    = "ACTION_RESULT"
)

// ErrMessageTooLarge identifies an inbound player message that exceeded the
// configured read limit. Callers may map it to an appropriate WebSocket close
// status without exposing raw input.
var ErrMessageTooLarge = errors.New("player message exceeds read limit")

// ClientMessage is the complete typed player-to-server protocol. Only fields
// applicable to Type are populated.
type ClientMessage struct {
	Type         string
	BrowserToken string
	RequestID    string
	BroadcastID  string
	TerminalID   string
	CharacterID  string
	Action       string
	NodeID       string
	TargetID     string
	PatternID    string
}

// DecodeClientMessage reads exactly one bounded JSON object and validates its
// fields before it can reach canonical state. Unknown and duplicate fields are
// rejected to keep this privileged mutation boundary explicit.
func DecodeClientMessage(reader io.Reader) (ClientMessage, error) {
	if reader == nil {
		return ClientMessage{}, errors.New("decode player message: reader is nil")
	}

	data, err := io.ReadAll(io.LimitReader(reader, MaxClientMessageBytes+1))
	if err != nil {
		return ClientMessage{}, fmt.Errorf("read player message: %w", err)
	}
	if len(data) > MaxClientMessageBytes {
		return ClientMessage{}, fmt.Errorf("%w: maximum is %d bytes", ErrMessageTooLarge, MaxClientMessageBytes)
	}

	fields, err := decodeStrictObject(data)
	if err != nil {
		return ClientMessage{}, fmt.Errorf("decode player message: %w", err)
	}
	typeName, err := requiredString(fields, "type")
	if err != nil {
		return ClientMessage{}, fmt.Errorf("decode player message: %w", err)
	}

	switch typeName {
	case MessageSessionHello:
		return decodeSessionHello(fields)
	case MessageCharacterSelect:
		return decodeCharacterSelect(fields)
	case MessageNavAction:
		return decodeNavAction(fields)
	case MessageHackGuess:
		return decodeHackGuess(fields)
	case MessageHackPattern:
		return decodeHackPattern(fields)
	default:
		return ClientMessage{}, fmt.Errorf("unsupported player message type %q", typeName)
	}
}

func decodeSessionHello(fields map[string]json.RawMessage) (ClientMessage, error) {
	if err := allowFields(fields, "type", "browserToken"); err != nil {
		return ClientMessage{}, err
	}
	message := ClientMessage{Type: MessageSessionHello}
	if raw, exists := fields["browserToken"]; exists {
		token, err := nonBlankString(raw, "browserToken")
		if err != nil {
			return ClientMessage{}, err
		}
		message.BrowserToken = token
	}
	return message, nil
}

func decodeCharacterSelect(fields map[string]json.RawMessage) (ClientMessage, error) {
	if err := allowFields(fields, "type", "requestId", "broadcastId", "characterId"); err != nil {
		return ClientMessage{}, err
	}
	requestID, broadcastID, err := requiredRequest(fields)
	if err != nil {
		return ClientMessage{}, err
	}
	characterID, err := requiredString(fields, "characterId")
	if err != nil {
		return ClientMessage{}, err
	}
	return ClientMessage{Type: MessageCharacterSelect, RequestID: requestID, BroadcastID: broadcastID, CharacterID: characterID}, nil
}

func decodeStrictObject(data []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	start, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := start.(json.Delim); !ok || delimiter != '{' {
		return nil, errors.New("message must be a JSON object")
	}

	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		name, ok := token.(string)
		if !ok {
			return nil, errors.New("object field name must be a string")
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, fmt.Errorf("duplicate field %q", name)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields[name] = value
	}

	end, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delimiter, ok := end.(json.Delim); !ok || delimiter != '}' {
		return nil, errors.New("message object is not terminated")
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("message contains a trailing JSON value")
		}
		return nil, err
	}
	return fields, nil
}

func decodeNavAction(fields map[string]json.RawMessage) (ClientMessage, error) {
	if err := allowFields(fields, "type", "requestId", "broadcastId", "terminalId", "action", "nodeId"); err != nil {
		return ClientMessage{}, err
	}
	requestID, broadcastID, terminalID, err := requiredTerminalRequest(fields)
	if err != nil {
		return ClientMessage{}, err
	}
	action, err := requiredString(fields, "action")
	if err != nil {
		return ClientMessage{}, err
	}
	message := ClientMessage{Type: MessageNavAction, RequestID: requestID, BroadcastID: broadcastID, TerminalID: terminalID, Action: action}

	switch action {
	case "back":
		if raw, exists := fields["nodeId"]; exists {
			message.NodeID, err = nonBlankString(raw, "nodeId")
		}
	case "enter", "command", "entry":
		message.NodeID, err = requiredString(fields, "nodeId")
	default:
		return ClientMessage{}, fmt.Errorf("unsupported navigation action %q", action)
	}
	if err != nil {
		return ClientMessage{}, err
	}
	return message, nil
}

func decodeHackGuess(fields map[string]json.RawMessage) (ClientMessage, error) {
	if err := allowFields(fields, "type", "requestId", "broadcastId", "terminalId", "targetId"); err != nil {
		return ClientMessage{}, err
	}
	requestID, broadcastID, terminalID, err := requiredTerminalRequest(fields)
	if err != nil {
		return ClientMessage{}, err
	}
	targetID, err := requiredString(fields, "targetId")
	if err != nil {
		return ClientMessage{}, err
	}
	return ClientMessage{Type: MessageHackGuess, RequestID: requestID, BroadcastID: broadcastID, TerminalID: terminalID, TargetID: targetID}, nil
}

func decodeHackPattern(fields map[string]json.RawMessage) (ClientMessage, error) {
	if err := allowFields(fields, "type", "requestId", "broadcastId", "terminalId", "patternId"); err != nil {
		return ClientMessage{}, err
	}
	requestID, broadcastID, terminalID, err := requiredTerminalRequest(fields)
	if err != nil {
		return ClientMessage{}, err
	}
	// patternId is an opaque server-issued identity that binds the active
	// puzzle generation to one inclusive rendered-row coordinate pair. The
	// player protocol echoes it and never reconstructs coordinates itself.
	patternID, err := requiredString(fields, "patternId")
	if err != nil {
		return ClientMessage{}, err
	}
	return ClientMessage{Type: MessageHackPattern, RequestID: requestID, BroadcastID: broadcastID, TerminalID: terminalID, PatternID: patternID}, nil
}

func requiredRequest(fields map[string]json.RawMessage) (string, string, error) {
	requestID, err := requiredString(fields, "requestId")
	if err != nil {
		return "", "", err
	}
	broadcastID, err := requiredString(fields, "broadcastId")
	if err != nil {
		return "", "", err
	}
	return requestID, broadcastID, nil
}

func requiredTerminalRequest(fields map[string]json.RawMessage) (string, string, string, error) {
	requestID, broadcastID, err := requiredRequest(fields)
	if err != nil {
		return "", "", "", err
	}
	terminalID, err := requiredString(fields, "terminalId")
	if err != nil {
		return "", "", "", err
	}
	return requestID, broadcastID, terminalID, nil
}

func allowFields(fields map[string]json.RawMessage, names ...string) error {
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		allowed[name] = struct{}{}
	}
	for name := range fields {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("field %q is not allowed for this message type", name)
		}
	}
	return nil
}

func requiredString(fields map[string]json.RawMessage, name string) (string, error) {
	raw, exists := fields[name]
	if !exists {
		return "", fmt.Errorf("field %q is required", name)
	}
	return nonBlankString(raw, name)
}

func nonBlankString(raw json.RawMessage, name string) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("field %q must be a string", name)
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("field %q must not be blank", name)
	}
	return value, nil
}

// TerminalLiveEnvelope is the complete reconnect/set-live projection.
type TerminalLiveEnvelope struct {
	Type         string                  `json:"type"`
	Revision     uint64                  `json:"revision"`
	TerminalID   string                  `json:"terminalId"`
	TerminalName string                  `json:"terminalName"`
	Tree         domain.ContentNode      `json:"tree"`
	HackLevel    int                     `json:"hackLevel"`
	IntroText    string                  `json:"introText"`
	Hack         *domain.PublicHackState `json:"hack"`
	Nav          domain.NavState         `json:"nav"`
}

// TerminalUpdateEnvelope replaces public content without resetting identity
// or the current hacking puzzle.
type TerminalUpdateEnvelope struct {
	Type       string             `json:"type"`
	Revision   uint64             `json:"revision"`
	TerminalID string             `json:"terminalId"`
	Tree       domain.ContentNode `json:"tree"`
	IntroText  string             `json:"introText"`
	Nav        domain.NavState    `json:"nav"`
}

// TerminalClearEnvelope announces that no terminal is currently live.
type TerminalClearEnvelope struct {
	Type     string `json:"type"`
	Revision uint64 `json:"revision"`
}

// NavStateEnvelope broadcasts the authoritative shared navigation position.
type NavStateEnvelope struct {
	Type       string          `json:"type"`
	Revision   uint64          `json:"revision"`
	TerminalID string          `json:"terminalId"`
	Nav        domain.NavState `json:"nav"`
}

// HackStateEnvelope broadcasts only the public hacking projection.
type HackStateEnvelope struct {
	Type       string                  `json:"type"`
	Revision   uint64                  `json:"revision"`
	TerminalID string                  `json:"terminalId"`
	Hack       *domain.PublicHackState `json:"hack"`
}

// SessionWelcomeEnvelope completes the mandatory connection handshake.
type SessionWelcomeEnvelope struct {
	Type         string              `json:"type"`
	BrowserToken string              `json:"browserToken"`
	State        *domain.PlayerState `json:"state"`
}

// PlayerStateEnvelope carries one complete personalized coordination snapshot.
type PlayerStateEnvelope struct {
	Type  string              `json:"type"`
	State *domain.PlayerState `json:"state"`
}

// ActionResultEnvelope completes one correlated player request.
type ActionResultEnvelope struct {
	Type      string              `json:"type"`
	RequestID string              `json:"requestId"`
	Accepted  bool                `json:"accepted"`
	Reason    domain.ActionReason `json:"reason"`
	Revision  uint64              `json:"revision"`
}

// NewTerminalLiveEnvelope constructs the exact TERMINAL_LIVE wire object.
func NewTerminalLiveEnvelope(revision uint64, state *domain.PublicLiveState) TerminalLiveEnvelope {
	envelope := TerminalLiveEnvelope{Type: MessageTerminalLive, Revision: revision}
	if state == nil {
		return envelope
	}
	envelope.TerminalID = state.TerminalID
	envelope.TerminalName = state.TerminalName
	envelope.Tree = state.Tree
	envelope.HackLevel = state.HackLevel
	envelope.IntroText = state.IntroText
	envelope.Hack = state.Hack
	envelope.Nav = state.Nav
	return envelope
}

// NewTerminalUpdateEnvelope constructs the exact TERMINAL_UPDATE wire object.
func NewTerminalUpdateEnvelope(revision uint64, state *domain.PublicLiveState) TerminalUpdateEnvelope {
	envelope := TerminalUpdateEnvelope{Type: MessageTerminalUpdate, Revision: revision}
	if state == nil {
		return envelope
	}
	envelope.Tree = state.Tree
	envelope.TerminalID = state.TerminalID
	envelope.IntroText = state.IntroText
	envelope.Nav = state.Nav
	return envelope
}

// NewTerminalClearEnvelope constructs the exact TERMINAL_CLEAR wire object.
func NewTerminalClearEnvelope(revision uint64) TerminalClearEnvelope {
	return TerminalClearEnvelope{Type: MessageTerminalClear, Revision: revision}
}

// NewNavStateEnvelope constructs the exact NAV_STATE wire object.
func NewNavStateEnvelope(revision uint64, terminalID string, state *domain.NavState) NavStateEnvelope {
	envelope := NavStateEnvelope{Type: MessageNavState, Revision: revision, TerminalID: terminalID}
	if state != nil {
		envelope.Nav = *state
	}
	return envelope
}

// NewHackStateEnvelope constructs the exact HACK_STATE wire object.
func NewHackStateEnvelope(revision uint64, terminalID string, state *domain.PublicHackState) HackStateEnvelope {
	return HackStateEnvelope{Type: MessageHackState, Revision: revision, TerminalID: terminalID, Hack: state}
}

// NewSessionWelcomeEnvelope constructs the exact SESSION_WELCOME object.
func NewSessionWelcomeEnvelope(browserToken string, state *domain.PlayerState) SessionWelcomeEnvelope {
	return SessionWelcomeEnvelope{Type: MessageSessionWelcome, BrowserToken: browserToken, State: domain.ClonePlayerState(state)}
}

// NewPlayerStateEnvelope constructs the exact PLAYER_STATE object.
func NewPlayerStateEnvelope(state *domain.PlayerState) PlayerStateEnvelope {
	return PlayerStateEnvelope{Type: MessagePlayerState, State: domain.ClonePlayerState(state)}
}

// NewActionResultEnvelope constructs the exact ACTION_RESULT object.
func NewActionResultEnvelope(result domain.ActionResult) ActionResultEnvelope {
	return ActionResultEnvelope{Type: MessageActionResult, RequestID: result.RequestID, Accepted: result.Accepted, Reason: result.Reason, Revision: result.Revision}
}
