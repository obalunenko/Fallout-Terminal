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
	Version      int                        `json:"version"`
	Name         string                     `json:"name"`
	PlayerConfig string                     `json:"playerConfig,omitempty"`
	Terminals    []Terminal                 `json:"terminals"`
	Extra        map[string]json.RawMessage `json:"-"`
}

// PlayerConfig is the durable version-1 authored player roster. It is stored
// separately from Session so runtime recognition, claims, and terminal state
// cannot cross the persistence boundary.
type PlayerConfig struct {
	Version int                    `json:"version"`
	Name    string                 `json:"name"`
	Roster  []CharacterRosterEntry `json:"roster"`
}

// PlayerConfigHandle is the private active-file identity used for atomic
// roster saves. Path is exposed only to the trusted desktop projection.
type PlayerConfigHandle struct {
	Path    string
	Version int
	Name    string
}

// PlayerConfigMetadata is the detached game-master view of the active config.
type PlayerConfigMetadata struct {
	Status   string `json:"status"`
	Name     string `json:"name"`
	FilePath string `json:"filePath"`
	Version  int    `json:"version"`
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
	ID     string `json:"id"`
	Start  int    `json:"start"`
	Length int    `json:"length"`
}

// HackColumn is one 192-character public hacking column.
type HackColumn struct {
	Addresses []string   `json:"addresses"`
	Text      string     `json:"text"`
	Words     []HackWord `json:"words"`
}

// HackCandidate is private lookup data for a placed hacking word.
type HackCandidate struct {
	Text string
}

// HackPatternIdentity is the complete one-use identity of a bracket span.
// Row, Start, and End are rendered-row coordinates; GenerationID prevents a
// delayed action from targeting coincident coordinates in a later puzzle.
type HackPatternIdentity struct {
	GenerationID string
	Row          int
	Start        int
	End          int
}

// HackPattern is one valid bracket span derived from the current board text.
// Storage coordinates and Pair remain private discovery metadata.
type HackPattern struct {
	Identity      HackPatternIdentity
	ColumnIndex   int
	AbsoluteStart int
	AbsoluteEnd   int
	Pair          string
}

// PublicHackPattern is the client-safe projection of one current bracket span.
type PublicHackPattern struct {
	ID    string `json:"id"`
	Row   int    `json:"row"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Used  bool   `json:"used"`
}

// HackState is the canonical private hacking aggregate.
type HackState struct {
	GenerationID string
	Level        int
	WordLength   int
	AttemptsMax  int
	AttemptsLeft int
	SecretWord   string
	WordsByID    map[string]HackCandidate
	UsedPatterns map[HackPatternIdentity]struct{}
	Solved       bool
	Failed       bool
	Log          []string
	Columns      []HackColumn
}

// PublicHackState is the only hacking representation permitted at a client boundary.
type PublicHackState struct {
	Level        int                 `json:"level"`
	WordLength   int                 `json:"wordLength"`
	AttemptsMax  int                 `json:"attemptsMax"`
	AttemptsLeft int                 `json:"attemptsLeft"`
	Solved       bool                `json:"solved"`
	Failed       bool                `json:"failed"`
	Log          []string            `json:"log"`
	Columns      []HackColumn        `json:"columns"`
	Patterns     []PublicHackPattern `json:"patterns"`
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

// LogicalSessionID identifies one browser profile for the lifetime of a server process.
type LogicalSessionID string

// BrowserToken is an opaque, process-local recognition handle. It is private
// coordinator state and must never appear in a public projection.
type BrowserToken string

// RecognitionHandle is the public name for the opaque process-local browser
// recognition value. BrowserToken remains as a compatibility alias while the
// legacy transport is removed.
type RecognitionHandle = BrowserToken

// ConnectionID identifies one concrete public stream.
type ConnectionID string

// PhysicalStream is detached identity metadata for one active subscription.
// Queue, cancellation, and synchronization objects remain transport-owned and
// are deliberately not part of this serializable boundary value.
type PhysicalStream struct {
	ID        ConnectionID
	SessionID LogicalSessionID
}

// CharacterID identifies one process-local roster entry.
type CharacterID string

// BroadcastID identifies one live-broadcast lifetime.
type BroadcastID string

// SwitchID identifies one pending terminal-switch decision.
type SwitchID string

// RequestID correlates one player command with its authoritative result.
// It remains wire-compatible with the browser-generated string identifier.
type RequestID = string

// RuntimeCommandKind identifies one supported shared player command after decoding.
type RuntimeCommandKind string

const (
	RuntimeCommandSelectCharacter RuntimeCommandKind = "select-character"
	RuntimeCommandNavigate        RuntimeCommandKind = "navigate"
	RuntimeCommandGuess           RuntimeCommandKind = "guess"
	RuntimeCommandActivatePattern RuntimeCommandKind = "activate-pattern"
)

// PlayerRole is a logical session's current broadcast-wide authority.
type PlayerRole string

const (
	PlayerRoleUnassigned PlayerRole = "unassigned"
	PlayerRoleActive     PlayerRole = "active"
	PlayerRoleObserver   PlayerRole = "observer"
)

// PlayerPhase determines which authoritative player surface is currently visible.
type PlayerPhase string

const (
	PlayerPhaseNoBroadcast PlayerPhase = "no-broadcast"
	PlayerPhaseSelecting   PlayerPhase = "selecting"
	PlayerPhaseWaiting     PlayerPhase = "waiting"
	PlayerPhaseControlling PlayerPhase = "controlling"
	PlayerPhaseObserving   PlayerPhase = "observing"
)

// RosterStatus is the player-safe availability of a roster entry.
type RosterStatus string

const (
	RosterStatusAvailable RosterStatus = "available"
	RosterStatusClaimed   RosterStatus = "claimed"
)

// TerminalLifecycle distinguishes the active runtime from an exact suspended checkpoint.
type TerminalLifecycle string

const (
	TerminalLifecycleActive    TerminalLifecycle = "active"
	TerminalLifecycleSuspended TerminalLifecycle = "suspended"
)

// TerminalSwitchChoice is the game master's explicit unfinished-puzzle decision.
type TerminalSwitchChoice string

const (
	TerminalSwitchPreserve TerminalSwitchChoice = "preserve"
	TerminalSwitchDiscard  TerminalSwitchChoice = "discard"
	TerminalSwitchCancel   TerminalSwitchChoice = "cancel"
)

// ActionReason is a stable public explanation for a player-command outcome.
type ActionReason string

const (
	ActionReasonAccepted               ActionReason = "accepted"
	ActionReasonInvalidSession         ActionReason = "invalid-session"
	ActionReasonStaleBroadcast         ActionReason = "stale-broadcast"
	ActionReasonUnassigned             ActionReason = "unassigned"
	ActionReasonNotController          ActionReason = "not-controller"
	ActionReasonControllerDisconnected ActionReason = "controller-disconnected"
	ActionReasonStaleTerminal          ActionReason = "stale-terminal"
	ActionReasonInvalidAction          ActionReason = "invalid-action"
	ActionReasonConflict               ActionReason = "conflict"
	ActionReasonDuplicate              ActionReason = "duplicate"
)

// ActionResult is the authoritative outcome of one correlated player command.
type ActionResult struct {
	RequestID RequestID    `json:"requestId"`
	Accepted  bool         `json:"accepted"`
	Reason    ActionReason `json:"reason"`
	Revision  uint64       `json:"revision"`
}

// RuntimeCommand is the transport-independent form of a shared player request.
// Only fields relevant to its command kind are populated.
type RuntimeCommand struct {
	RequestID          RequestID
	BroadcastID        BroadcastID
	TerminalID         string
	Kind               RuntimeCommandKind
	Action             string
	NodeID             string
	TargetID           string
	PatternID          string
	PayloadFingerprint string
}

// RequestResultRecord retains enough information to make request replay idempotent.
type RequestResultRecord struct {
	Fingerprint string
	Result      ActionResult
}

// RequestReplayRecord is the complete detached value retained by the bounded
// Connect mutation replay cache. It carries no transport request object.
type RequestReplayRecord struct {
	RequestID          RequestID
	Procedure          string
	PayloadFingerprint string
	Result             ActionResult
	Revision           uint64
}

// BrowserRecognition is the private mapping from an opaque browser token to a
// process-local logical session.
type BrowserRecognition struct {
	BrowserToken BrowserToken
	SessionID    LogicalSessionID
}

// LogicalSession is canonical process-local browser-profile state.
type LogicalSession struct {
	ID             LogicalSessionID
	FallbackName   string
	ConnectionIDs  map[ConnectionID]struct{}
	RequestResults map[RequestID]RequestResultRecord
}

// CharacterRosterEntry is one stable process-local player identity option.
type CharacterRosterEntry struct {
	ID   CharacterID `json:"id"`
	Name string      `json:"name"`
}

// CharacterAssignment is one broadcast-scoped exclusive claim.
type CharacterAssignment struct {
	BroadcastID BroadcastID
	SessionID   LogicalSessionID
	CharacterID CharacterID
}

// ControllerAssignment designates the one assigned session allowed to mutate shared state.
type ControllerAssignment struct {
	SessionID LogicalSessionID
}

// TerminalRuntime is an exact canonical active or suspended terminal checkpoint.
type TerminalRuntime struct {
	TerminalID   string
	TerminalName string
	Tree         ContentNode
	HackLevel    int
	IntroText    string
	Nav          NavState
	Hack         *HackState
	Lifecycle    TerminalLifecycle
}

// LiveBroadcast owns all state whose lifetime ends with the current broadcast.
type LiveBroadcast struct {
	ID                   BroadcastID
	AssignmentsBySession map[LogicalSessionID]CharacterID
	SessionByCharacter   map[CharacterID]LogicalSessionID
	ControllerSessionID  *LogicalSessionID
	ActiveTerminalID     *string
	TerminalRuntimes     map[string]*TerminalRuntime
}

// TerminalTarget is the validated authored payload retained by a pending switch.
type TerminalTarget struct {
	TerminalID   string
	TerminalName string
	Tree         ContentNode
	HackLevel    int
	IntroText    string
}

// TerminalSwitchDecision keeps a switch request ordered against the source runtime.
// A nil Target requests clearing the active terminal while retaining the broadcast.
type TerminalSwitchDecision struct {
	ID               SwitchID
	BroadcastID      BroadcastID
	SourceTerminalID string
	Target           *TerminalTarget
}

// ProcessRuntime is the private canonical root owned by the coordination service.
// It is intentionally unrelated to the durable version-1 Session document.
type ProcessRuntime struct {
	Revision                uint64
	SessionsByID            map[LogicalSessionID]*LogicalSession
	SessionIDByBrowserToken map[BrowserToken]LogicalSessionID
	RosterByID              map[CharacterID]*CharacterRosterEntry
	RosterOrder             []CharacterID
	ActivePlayerConfig      *PlayerConfigHandle
	Broadcast               *LiveBroadcast
	PendingSwitch           *TerminalSwitchDecision
}

// PlayerCharacter is the assigned identity visible at a projection boundary.
type PlayerCharacter struct {
	ID   CharacterID `json:"id"`
	Name string      `json:"name"`
}

// PlayerRosterEntry contains availability without claimant or presence details.
type PlayerRosterEntry struct {
	ID     CharacterID  `json:"id"`
	Name   string       `json:"name"`
	Status RosterStatus `json:"status"`
}

// PlayerState is one complete personalized, secret-free browser projection.
// Empty broadcast and terminal IDs marshal as null to match the frozen protocol.
type PlayerState struct {
	Revision         uint64              `json:"revision"`
	SessionID        LogicalSessionID    `json:"sessionId"`
	FallbackName     string              `json:"fallbackName"`
	Character        *PlayerCharacter    `json:"character"`
	Role             PlayerRole          `json:"role"`
	Phase            PlayerPhase         `json:"phase"`
	BroadcastID      BroadcastID         `json:"-"`
	ActiveTerminalID string              `json:"-"`
	Roster           []PlayerRosterEntry `json:"roster"`
}

// TerminalPresentation is an exclusive detached public terminal projection.
// Live is non-nil for a complete active terminal; NoLiveTerminal is true for
// the explicit empty variant. Adapters reject every other combination.
type TerminalPresentation struct {
	Live           *PublicLiveState
	NoLiveTerminal bool
}

// PersonalizedSnapshot is the mandatory first value for every subscription.
type PersonalizedSnapshot struct {
	RecognitionHandle RecognitionHandle
	Revision          uint64
	PlayerState       *PlayerState
	Terminal          TerminalPresentation
}

// CompoundUpdate is one complete personalized publication for a committed
// revision. Nil components mean unchanged, never clear or partial patch.
type CompoundUpdate struct {
	Revision uint64
	Player   *PlayerState
	Terminal *TerminalPresentation
	Nav      *NavState
	Hack     *PublicHackState
}

// BrowserPendingAction tracks the two independent acknowledgements required
// before an accepted browser action is no longer pending.
type BrowserPendingAction struct {
	RequestID      RequestID
	Result         *ActionResult
	StreamRevision uint64
}

// SoundCategory is one stable allowlisted same-origin asset group.
type SoundCategory string

const (
	SoundCategoryAmbient    SoundCategory = "ambient"
	SoundCategoryHackGood   SoundCategory = "hack-good"
	SoundCategoryHackBad    SoundCategory = "hack-bad"
	SoundCategoryMenuFocus  SoundCategory = "menu-focus"
	SoundCategorySingle     SoundCategory = "single"
	SoundCategoryMultiple   SoundCategory = "multiple"
	SoundCategoryEnter      SoundCategory = "enter"
	SoundCategoryCharscroll SoundCategory = "charscroll"
)

// SoundManifest contains only sorted safe relative same-origin asset paths.
type SoundManifest struct {
	Category SoundCategory
	Assets   []string
}

// MarshalJSON preserves nullable identifiers while keeping convenient typed
// scalar fields for coordinator and protocol construction.
func (state PlayerState) MarshalJSON() ([]byte, error) {
	var broadcastID *BroadcastID
	if state.BroadcastID != "" {
		value := state.BroadcastID
		broadcastID = &value
	}
	var activeTerminalID *string
	if state.ActiveTerminalID != "" {
		value := state.ActiveTerminalID
		activeTerminalID = &value
	}
	return json.Marshal(struct {
		Revision         uint64              `json:"revision"`
		SessionID        LogicalSessionID    `json:"sessionId"`
		FallbackName     string              `json:"fallbackName"`
		Character        *PlayerCharacter    `json:"character"`
		Role             PlayerRole          `json:"role"`
		Phase            PlayerPhase         `json:"phase"`
		BroadcastID      *BroadcastID        `json:"broadcastId"`
		ActiveTerminalID *string             `json:"activeTerminalId"`
		Roster           []PlayerRosterEntry `json:"roster"`
	}{
		Revision:         state.Revision,
		SessionID:        state.SessionID,
		FallbackName:     state.FallbackName,
		Character:        state.Character,
		Role:             state.Role,
		Phase:            state.Phase,
		BroadcastID:      broadcastID,
		ActiveTerminalID: activeTerminalID,
		Roster:           state.Roster,
	})
}

// MasterRosterEntry is the game-master view of one roster claim.
type MasterRosterEntry struct {
	ID                 CharacterID       `json:"id"`
	Name               string            `json:"name"`
	ClaimedBySessionID *LogicalSessionID `json:"claimedBySessionId"`
}

// MasterSessionEntry is the game-master view of one recognized logical session.
type MasterSessionEntry struct {
	ID           LogicalSessionID `json:"id"`
	FallbackName string           `json:"fallbackName"`
	Connected    bool             `json:"connected"`
	Character    *PlayerCharacter `json:"character"`
	Role         PlayerRole       `json:"role"`
}

// MasterBroadcastState is the game-master view of the current broadcast epoch.
type MasterBroadcastState struct {
	ID                  BroadcastID       `json:"id"`
	ControllerSessionID *LogicalSessionID `json:"controllerSessionId"`
	ActiveTerminalID    *string           `json:"activeTerminalId"`
}

// MasterPendingSwitch is the non-secret metadata for one pending switch decision.
type MasterPendingSwitch struct {
	SwitchID         SwitchID    `json:"switchId"`
	BroadcastID      BroadcastID `json:"broadcastId"`
	SourceTerminalID string      `json:"sourceTerminalId"`
	TargetTerminalID *string     `json:"targetTerminalId"`
}

// MasterCoordinationState is one detached private-desktop projection.
type MasterCoordinationState struct {
	Revision      uint64                `json:"revision"`
	PlayerConfig  *PlayerConfigMetadata `json:"playerConfig"`
	Roster        []MasterRosterEntry   `json:"roster"`
	Sessions      []MasterSessionEntry  `json:"sessions"`
	Broadcast     *MasterBroadcastState `json:"broadcast"`
	PendingSwitch *MasterPendingSwitch  `json:"pendingSwitch"`
}

// CloneMasterCoordinationState returns a deeply detached desktop projection.
func CloneMasterCoordinationState(state *MasterCoordinationState) *MasterCoordinationState {
	if state == nil {
		return nil
	}
	clone := *state
	if state.PlayerConfig != nil {
		value := *state.PlayerConfig
		clone.PlayerConfig = &value
	}
	clone.Roster = append([]MasterRosterEntry(nil), state.Roster...)
	for index := range clone.Roster {
		clone.Roster[index].ClaimedBySessionID = cloneLogicalSessionID(state.Roster[index].ClaimedBySessionID)
	}
	clone.Sessions = append([]MasterSessionEntry(nil), state.Sessions...)
	for index := range clone.Sessions {
		clone.Sessions[index].Character = clonePlayerCharacter(state.Sessions[index].Character)
	}
	if state.Broadcast != nil {
		broadcast := *state.Broadcast
		broadcast.ControllerSessionID = cloneLogicalSessionID(state.Broadcast.ControllerSessionID)
		broadcast.ActiveTerminalID = cloneString(state.Broadcast.ActiveTerminalID)
		clone.Broadcast = &broadcast
	}
	if state.PendingSwitch != nil {
		pending := *state.PendingSwitch
		pending.TargetTerminalID = cloneString(state.PendingSwitch.TargetTerminalID)
		clone.PendingSwitch = &pending
	}
	return &clone
}

// ClonePlayerState returns a deeply detached personalized browser projection.
func ClonePlayerState(state *PlayerState) *PlayerState {
	if state == nil {
		return nil
	}
	clone := *state
	clone.Character = clonePlayerCharacter(state.Character)
	clone.Roster = append([]PlayerRosterEntry(nil), state.Roster...)
	return &clone
}

// CloneTerminalPresentation returns a deeply detached terminal variant.
func CloneTerminalPresentation(presentation TerminalPresentation) TerminalPresentation {
	return TerminalPresentation{
		Live:           clonePublicLiveState(presentation.Live),
		NoLiveTerminal: presentation.NoLiveTerminal,
	}
}

// ClonePersonalizedSnapshot returns a deeply detached first-stream value.
func ClonePersonalizedSnapshot(snapshot *PersonalizedSnapshot) *PersonalizedSnapshot {
	if snapshot == nil {
		return nil
	}
	clone := *snapshot
	clone.PlayerState = ClonePlayerState(snapshot.PlayerState)
	clone.Terminal = CloneTerminalPresentation(snapshot.Terminal)
	return &clone
}

// CloneCompoundUpdate returns a deeply detached authoritative publication.
func CloneCompoundUpdate(update *CompoundUpdate) *CompoundUpdate {
	if update == nil {
		return nil
	}
	clone := *update
	clone.Player = ClonePlayerState(update.Player)
	if update.Terminal != nil {
		terminal := CloneTerminalPresentation(*update.Terminal)
		clone.Terminal = &terminal
	}
	if update.Nav != nil {
		nav := *update.Nav
		nav.Path = append([]string(nil), update.Nav.Path...)
		nav.ViewEntryID = cloneString(update.Nav.ViewEntryID)
		nav.CommandNodeID = cloneString(update.Nav.CommandNodeID)
		clone.Nav = &nav
	}
	clone.Hack = clonePublicHackState(update.Hack)
	return &clone
}

func clonePublicLiveState(state *PublicLiveState) *PublicLiveState {
	if state == nil {
		return nil
	}
	clone := *state
	clone.Tree = cloneContentNode(state.Tree)
	clone.Nav.Path = append([]string(nil), state.Nav.Path...)
	clone.Nav.ViewEntryID = cloneString(state.Nav.ViewEntryID)
	clone.Nav.CommandNodeID = cloneString(state.Nav.CommandNodeID)
	clone.Hack = clonePublicHackState(state.Hack)
	return &clone
}

func cloneContentNode(node ContentNode) ContentNode {
	clone := node
	clone.Extra = cloneRawMessages(node.Extra)
	clone.Children = make([]ContentNode, len(node.Children))
	for index := range node.Children {
		clone.Children[index] = cloneContentNode(node.Children[index])
	}
	return clone
}

func cloneRawMessages(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return nil
	}
	clone := make(map[string]json.RawMessage, len(values))
	for key, value := range values {
		clone[key] = append(json.RawMessage(nil), value...)
	}
	return clone
}

func clonePublicHackState(state *PublicHackState) *PublicHackState {
	if state == nil {
		return nil
	}
	clone := *state
	clone.Log = append([]string(nil), state.Log...)
	clone.Patterns = append([]PublicHackPattern(nil), state.Patterns...)
	clone.Columns = make([]HackColumn, len(state.Columns))
	for index, column := range state.Columns {
		clone.Columns[index] = HackColumn{
			Addresses: append([]string(nil), column.Addresses...),
			Text:      column.Text,
			Words:     append([]HackWord(nil), column.Words...),
		}
	}
	return &clone
}

func cloneLogicalSessionID(value *LogicalSessionID) *LogicalSessionID {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func clonePlayerCharacter(value *PlayerCharacter) *PlayerCharacter {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
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
