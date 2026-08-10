// Package domain contains transport-independent Fallout Terminal models.
package domain

import "encoding/json"

const (
	// NodeFolder identifies a recursive folder node.
	NodeFolder = "folder"
	// NodeCommand identifies a command/output leaf node.
	NodeCommand = "command"
	// NodeEntry identifies a descriptive entry leaf node.
	NodeEntry = "entry"
)

// Session is the durable version-1 campaign document.
type Session struct {
	Version   int                        `json:"version"`
	Name      string                     `json:"name"`
	Terminals []Terminal                 `json:"terminals"`
	Extra     map[string]json.RawMessage `json:"-"`
}

// Terminal is one durable authoring and broadcast target.
type Terminal struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	HackLevel int                        `json:"hackLevel"`
	IntroText string                     `json:"introText"`
	Root      ContentNode                `json:"root"`
	Extra     map[string]json.RawMessage `json:"-"`
}

// ContentNode is a tagged folder, command, or entry node.
type ContentNode struct {
	ID          string                     `json:"id"`
	Type        string                     `json:"type"`
	Name        string                     `json:"name"`
	Children    []ContentNode              `json:"children,omitempty"`
	Text        string                     `json:"text,omitempty"`
	Description string                     `json:"description,omitempty"`
	Extra       map[string]json.RawMessage `json:"-"`
}

// NavState is the shared server-authoritative player position.
type NavState struct {
	Path          []string `json:"path"`
	Mode          string   `json:"mode"`
	ViewEntryID   *string  `json:"viewEntryId"`
	CommandNodeID *string  `json:"commandNodeId"`
}

// HackWord describes a visible word placement in a hacking column.
type HackWord struct {
	ID      string `json:"id"`
	Start   int    `json:"start"`
	Length  int    `json:"length"`
	IsAdmin bool   `json:"isAdmin"`
}

// HackColumn is one 192-character public hacking column.
type HackColumn struct {
	Addresses []string   `json:"addresses"`
	Text      string     `json:"text"`
	Words     []HackWord `json:"words"`
}

// HackCandidate is private lookup data for a placed hacking word.
type HackCandidate struct {
	Text    string
	IsAdmin bool
}

// HackState is the canonical private hacking aggregate.
type HackState struct {
	Level         int
	WordLength    int
	AttemptsMax   int
	AttemptsLeft  int
	SecretWord    string
	WordsByID     map[string]HackCandidate
	AdminModeUsed bool
	Solved        bool
	Failed        bool
	Log           []string
	Columns       []HackColumn
}

// PublicHackState is the only hacking representation permitted at a client boundary.
type PublicHackState struct {
	Level        int          `json:"level"`
	WordLength   int          `json:"wordLength"`
	AttemptsMax  int          `json:"attemptsMax"`
	AttemptsLeft int          `json:"attemptsLeft"`
	Solved       bool         `json:"solved"`
	Failed       bool         `json:"failed"`
	Log          []string     `json:"log"`
	Columns      []HackColumn `json:"columns"`
}

// LiveState is the private process-local canonical broadcast state.
type LiveState struct {
	TerminalID   string
	TerminalName string
	Tree         ContentNode
	HackLevel    int
	IntroText    string
	Nav          NavState
	Hack         *HackState
}

// PublicLiveState is the immutable client-facing live snapshot.
type PublicLiveState struct {
	TerminalID   string           `json:"terminalId"`
	TerminalName string           `json:"terminalName"`
	Tree         ContentNode      `json:"tree"`
	HackLevel    int              `json:"hackLevel"`
	IntroText    string           `json:"introText"`
	Nav          NavState         `json:"nav"`
	Hack         *PublicHackState `json:"hack"`
}

// ServerInfo is safe status displayed to the game master.
type ServerInfo struct {
	IP          string `json:"ip"`
	Port        int    `json:"port"`
	URL         string `json:"url"`
	LocalURL    string `json:"localUrl,omitempty"`
	Tunnel      bool   `json:"tunnel"`
	TunnelError string `json:"tunnelError,omitempty"`
}
