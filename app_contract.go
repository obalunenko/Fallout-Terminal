package main

import (
	"fmt"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	persistencev1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/persistence/v1"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	privatev1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/private/v1"
	sessionservice "github.com/obalunenko/Fallout-Terminal/internal/session"
)

// The private protobuf graph governs trusted desktop semantics only. These
// adapters are invoked inside App while Wails continues carrying the existing
// native DTOs; no protobuf bytes, ProtoJSON, or generic envelope crosses Wails.

func routeAddCharacterRequest(name string) string {
	return (&privatev1.AddCharacterRequest{DisplayName: name}).GetDisplayName()
}

func routeRenameCharacterRequest(payload CharacterRenamePayload) CharacterRenamePayload {
	semantic := &privatev1.RenameCharacterRequest{CharacterId: string(payload.CharacterID), DisplayName: payload.Name}
	return CharacterRenamePayload{CharacterID: domain.CharacterID(semantic.GetCharacterId()), Name: semantic.GetDisplayName()}
}

func routeDeleteCharacterRequest(characterID string) string {
	return (&privatev1.DeleteCharacterRequest{CharacterId: characterID}).GetCharacterId()
}

func routeRenameLogicalSessionRequest(payload LogicalSessionRenamePayload) LogicalSessionRenamePayload {
	semantic := &privatev1.RenameLogicalSessionRequest{LogicalSessionId: string(payload.SessionID), FallbackName: payload.FallbackName}
	return LogicalSessionRenamePayload{SessionID: domain.LogicalSessionID(semantic.GetLogicalSessionId()), FallbackName: semantic.GetFallbackName()}
}

func routeAssignCharacterRequest(payload AssignmentPayload) AssignmentPayload {
	semantic := &privatev1.AssignCharacterRequest{LogicalSessionId: string(payload.SessionID), CharacterId: string(payload.CharacterID)}
	return AssignmentPayload{SessionID: domain.LogicalSessionID(semantic.GetLogicalSessionId()), CharacterID: domain.CharacterID(semantic.GetCharacterId())}
}

func routeReleaseCharacterRequest(sessionID string) string {
	return (&privatev1.ReleaseCharacterRequest{LogicalSessionId: sessionID}).GetLogicalSessionId()
}

func routeMoveCharacterRequest(payload MoveCharacterPayload) MoveCharacterPayload {
	semantic := &privatev1.MoveCharacterRequest{CharacterId: string(payload.CharacterID), DestinationSessionId: string(payload.ToSessionID)}
	return MoveCharacterPayload{CharacterID: domain.CharacterID(semantic.GetCharacterId()), ToSessionID: domain.LogicalSessionID(semantic.GetDestinationSessionId())}
}

func routeSetActiveControllerRequest(sessionID string) string {
	return (&privatev1.SetActiveControllerRequest{LogicalSessionId: sessionID}).GetLogicalSessionId()
}

func routeOpenURLRequest(rawURL string) string {
	return (&privatev1.OpenUrlRequest{Url: rawURL}).GetUrl()
}

func routeTerminalSwitchDecisionRequest(payload TerminalSwitchDecisionPayload) (TerminalSwitchDecisionPayload, error) {
	choice, err := terminalSwitchChoiceToPrivate(payload.Decision)
	if err != nil {
		return payload, err
	}
	semantic := &privatev1.TerminalSwitchDecisionRequest{SwitchId: string(payload.SwitchID), Choice: choice}
	return TerminalSwitchDecisionPayload{SwitchID: domain.SwitchID(semantic.GetSwitchId()), Decision: terminalSwitchChoiceFromPrivate(semantic.GetChoice())}, nil
}

func routeTerminalActivationRequest(payload LiveTerminalPayload, reset bool) (LiveTerminalPayload, error) {
	tree, err := contentNodeToPrivate(payload.Tree)
	if err != nil {
		return payload, err
	}
	if reset {
		semantic := &privatev1.ResetFailedHackRequest{
			TerminalId: payload.TerminalID, TerminalName: payload.TerminalName, Tree: tree,
			HackLevel: int32(payload.HackLevel), IntroText: payload.IntroText,
		}
		return liveTerminalFromPrivate(semantic.GetTerminalId(), semantic.GetTerminalName(), semantic.GetTree(), semantic.GetHackLevel(), semantic.GetIntroText(), payload.Tree)
	}
	semantic := &privatev1.TerminalActivationRequest{
		TerminalId: payload.TerminalID, TerminalName: payload.TerminalName, Tree: tree,
		HackLevel: int32(payload.HackLevel), IntroText: payload.IntroText,
	}
	return liveTerminalFromPrivate(semantic.GetTerminalId(), semantic.GetTerminalName(), semantic.GetTree(), semantic.GetHackLevel(), semantic.GetIntroText(), payload.Tree)
}

func routeLiveTerminalUpdateRequest(payload LiveUpdatePayload) (LiveUpdatePayload, error) {
	tree, err := contentNodeToPrivate(payload.Tree)
	if err != nil {
		return payload, err
	}
	semantic := &privatev1.LiveTerminalUpdateRequest{Tree: tree, IntroText: payload.IntroText}
	routedTree, err := contentNodeFromPrivate(semantic.GetTree(), payload.Tree)
	if err != nil {
		return payload, err
	}
	result := LiveUpdatePayload{Tree: routedTree}
	if semantic.IntroText != nil {
		value := semantic.GetIntroText()
		result.IntroText = &value
	}
	return result, nil
}

func liveTerminalFromPrivate(id, name string, tree *persistencev1.ContentNode, level int32, intro string, template domain.ContentNode) (LiveTerminalPayload, error) {
	routedTree, err := contentNodeFromPrivate(tree, template)
	if err != nil {
		return LiveTerminalPayload{}, err
	}
	return LiveTerminalPayload{TerminalID: id, TerminalName: name, Tree: routedTree, HackLevel: int(level), IntroText: intro}, nil
}

func contentNodeToPrivate(node domain.ContentNode) (*persistencev1.ContentNode, error) {
	node = cloneTreeForBridgeValidation(node)
	semantic, err := sessionservice.SessionToProto(domain.Session{
		Version: 1, Name: "private bridge", Terminals: []domain.Terminal{{ID: "bridge", Name: "bridge", Root: node}},
	})
	if err != nil {
		return nil, err
	}
	return semantic.GetTerminals()[0].GetRoot(), nil
}

func contentNodeFromPrivate(node *persistencev1.ContentNode, template domain.ContentNode) (domain.ContentNode, error) {
	semantic := &persistencev1.Session{Version: 1, Name: "private bridge", Terminals: []*persistencev1.Terminal{{Id: "bridge", Name: "bridge", Root: node}}}
	routed, err := sessionservice.SessionFromProto(semantic, domain.Session{
		Version: 1, Name: "private bridge", Terminals: []domain.Terminal{{ID: "bridge", Name: "bridge", Root: template}},
	})
	if err != nil {
		return domain.ContentNode{}, err
	}
	result := routed.Terminals[0].Root
	restoreContentNodeShape(&result, template)
	return result, nil
}

func restoreContentNodeShape(node *domain.ContentNode, template domain.ContentNode) {
	if node == nil {
		return
	}
	if template.Children == nil && len(node.Children) == 0 {
		node.Children = nil
	}
	for index := range node.Children {
		if index < len(template.Children) {
			restoreContentNodeShape(&node.Children[index], template.Children[index])
		}
	}
}

func runtimeStatusToPrivate(status RuntimeStatus) *privatev1.RuntimeStatus {
	result := &privatev1.RuntimeStatus{
		ClientCount: uint32(max(status.ClientCount, 0)), SaveState: status.SaveState,
		RequestedRevision: status.RequestedRevision,
		CoordinationState: coordinationStateToPrivate(status.CoordinationState),
	}
	if status.ServerInfo != nil {
		result.ServerInfo = serverInfoToPrivate(*status.ServerInfo)
	}
	if status.HackState != nil {
		result.HackState = publicHackToPrivate(status.HackState)
	}
	if status.StartupError != "" {
		result.StartupError = &status.StartupError
	}
	if status.SavedRevision != 0 {
		result.SavedRevision = &status.SavedRevision
	}
	return result
}

func runtimeStatusFromPrivate(status *privatev1.RuntimeStatus) RuntimeStatus {
	if status == nil {
		return RuntimeStatus{}
	}
	result := RuntimeStatus{
		ClientCount:       int(status.GetClientCount()),
		HackState:         publicHackFromPrivate(status.GetHackState()),
		SaveState:         status.GetSaveState(),
		RequestedRevision: status.GetRequestedRevision(),
		CoordinationState: coordinationStateFromPrivate(status.GetCoordinationState()),
	}
	if status.ServerInfo != nil {
		value := serverInfoFromPrivate(status.ServerInfo)
		result.ServerInfo = &value
	}
	if status.StartupError != nil {
		result.StartupError = status.GetStartupError()
	}
	if status.SavedRevision != nil {
		result.SavedRevision = status.GetSavedRevision()
	}
	return result
}

func routeRuntimeStatus(status RuntimeStatus) RuntimeStatus {
	return runtimeStatusFromPrivate(runtimeStatusToPrivate(status))
}

func commandResultToPrivate(result CommandResult) *privatev1.CommandResult {
	semantic := &privatev1.CommandResult{Ok: result.OK}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	return semantic
}

func routeCommandResult(result CommandResult) CommandResult {
	semantic := commandResultToPrivate(result)
	return CommandResult{OK: semantic.GetOk(), Error: semantic.GetError()}
}

func routeSessionOperationResult(result sessionservice.SessionResult) sessionservice.SessionResult {
	semantic := &privatev1.SessionOperationResult{Ok: result.OK, Canceled: result.Canceled}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	if result.FilePath != "" {
		semantic.FilePath = &result.FilePath
	}
	if result.Session != nil {
		semantic.Session, _ = sessionservice.SessionToProto(*result.Session)
	}
	routed := sessionservice.SessionResult{OK: semantic.GetOk(), Canceled: semantic.GetCanceled(), Error: semantic.GetError(), FilePath: semantic.GetFilePath()}
	if semantic.Session != nil {
		template := domain.Session{}
		if result.Session != nil {
			template = *result.Session
		}
		if value, err := sessionservice.SessionFromProto(semantic.Session, template); err == nil {
			routed.Session = &value
		}
	}
	return routed
}

func routeSaveSessionResult(result sessionservice.SaveResult) sessionservice.SaveResult {
	semantic := &privatev1.SaveSessionResult{Ok: result.OK, RequestedRevision: result.RequestedRevision}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	if result.SavedRevision != 0 {
		semantic.SavedRevision = &result.SavedRevision
	}
	return sessionservice.SaveResult{
		OK: semantic.GetOk(), Error: semantic.GetError(), RequestedRevision: semantic.GetRequestedRevision(), SavedRevision: semantic.GetSavedRevision(),
	}
}

func routePlayerConfigResult(result PlayerConfigCommandResult) PlayerConfigCommandResult {
	semantic := &privatev1.PlayerConfigOperationResult{Ok: result.OK, Canceled: result.Canceled, State: coordinationStateToPrivate(result.State)}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	if result.Session != nil {
		semantic.Session, _ = sessionservice.SessionToProto(*result.Session)
	}
	if result.Config != nil {
		semantic.PlayerConfigMetadata = playerConfigMetadataToPrivate(result.Config)
	}
	routed := PlayerConfigCommandResult{
		OK: semantic.GetOk(), Canceled: semantic.GetCanceled(), Error: semantic.GetError(),
		Config: playerConfigMetadataFromPrivate(semantic.GetPlayerConfigMetadata()),
		State:  coordinationStateFromPrivate(semantic.GetState()),
	}
	if semantic.Session != nil {
		template := domain.Session{}
		if result.Session != nil {
			template = *result.Session
		}
		if value, err := sessionservice.SessionFromProto(semantic.Session, template); err == nil {
			routed.Session = &value
		}
	}
	return routed
}

func coordinationResultToPrivate(result CoordinationCommandResult) *privatev1.CoordinationResult {
	semantic := &privatev1.CoordinationResult{Ok: result.OK, State: coordinationStateToPrivate(result.State)}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	return semantic
}

func routeCoordinationResult(result CoordinationCommandResult) CoordinationCommandResult {
	semantic := coordinationResultToPrivate(result)
	return CoordinationCommandResult{OK: semantic.GetOk(), Error: semantic.GetError(), State: coordinationStateFromPrivate(semantic.GetState())}
}

func terminalSwitchResultToPrivate(result TerminalSwitchCommandResult) *privatev1.TerminalSwitchResult {
	semantic := &privatev1.TerminalSwitchResult{Ok: result.OK, State: coordinationStateToPrivate(result.State), Status: terminalSwitchStatusToPrivate(result.Status)}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	if result.SwitchID != "" {
		value := string(result.SwitchID)
		semantic.SwitchId = &value
	}
	return semantic
}

func routeTerminalSwitchResult(result TerminalSwitchCommandResult) TerminalSwitchCommandResult {
	semantic := terminalSwitchResultToPrivate(result)
	return TerminalSwitchCommandResult{
		OK: semantic.GetOk(), Error: semantic.GetError(), Status: terminalSwitchStatusFromPrivate(semantic.GetStatus()),
		SwitchID: domain.SwitchID(semantic.GetSwitchId()), State: coordinationStateFromPrivate(semantic.GetState()),
	}
}

func routeServerInfoEvent(info domain.ServerInfo) domain.ServerInfo {
	semantic := &privatev1.ServerInformationEvent{ServerInfo: serverInfoToPrivate(info)}
	return serverInfoFromPrivate(semantic.GetServerInfo())
}

func routeClientCountEvent(count int) int {
	semantic := &privatev1.ClientCountEvent{ClientCount: uint32(max(count, 0))}
	return int(semantic.GetClientCount())
}

func routeHackStateEvent(state *domain.PublicHackState) *domain.PublicHackState {
	semantic := &privatev1.HackStateEvent{HackState: publicHackToPrivate(state)}
	return publicHackFromPrivate(semantic.GetHackState())
}

func routeCoordinationEvent(state *domain.MasterCoordinationState) *domain.MasterCoordinationState {
	semantic := &privatev1.CoordinationStateEvent{CoordinationState: coordinationStateToPrivate(state)}
	return coordinationStateFromPrivate(semantic.GetCoordinationState())
}

func coordinationStateToPrivate(state *domain.MasterCoordinationState) *privatev1.CoordinationState {
	if state == nil {
		return nil
	}
	result := &privatev1.CoordinationState{
		Roster:          make([]*privatev1.CharacterState, 0, len(state.Roster)),
		LogicalSessions: make([]*privatev1.LogicalSessionState, 0, len(state.Sessions)),
		Revision:        state.Revision,
		PlayerConfig:    playerConfigMetadataToPrivate(state.PlayerConfig),
	}
	for _, entry := range state.Roster {
		mapped := &privatev1.CharacterState{CharacterId: string(entry.ID), DisplayName: entry.Name}
		if entry.ClaimedBySessionID != nil {
			value := string(*entry.ClaimedBySessionID)
			mapped.LogicalSessionId = &value
		}
		result.Roster = append(result.Roster, mapped)
	}
	for _, session := range state.Sessions {
		mapped := &privatev1.LogicalSessionState{LogicalSessionId: string(session.ID), FallbackName: session.FallbackName, Connected: session.Connected, Role: playerRoleToPrivate(session.Role)}
		if session.Connected {
			mapped.ActiveStreams = 1
		}
		if session.Character != nil {
			value := string(session.Character.ID)
			mapped.CharacterId = &value
		}
		result.LogicalSessions = append(result.LogicalSessions, mapped)
	}
	if state.Broadcast != nil {
		result.Broadcast = &privatev1.BroadcastState{BroadcastId: string(state.Broadcast.ID), Revision: state.Revision}
		if state.Broadcast.ControllerSessionID != nil {
			value := string(*state.Broadcast.ControllerSessionID)
			result.Broadcast.ActiveControllerSessionId = &value
		}
		if state.Broadcast.ActiveTerminalID != nil {
			value := *state.Broadcast.ActiveTerminalID
			result.Broadcast.ActiveTerminalId = &value
		}
	}
	if state.PendingSwitch != nil {
		targetID := ""
		if state.PendingSwitch.TargetTerminalID != nil {
			targetID = *state.PendingSwitch.TargetTerminalID
		}
		result.PendingTerminalSwitch = &privatev1.PendingTerminalSwitch{
			SwitchId: string(state.PendingSwitch.SwitchID), TerminalId: targetID,
			BroadcastId: string(state.PendingSwitch.BroadcastID), SourceTerminalId: state.PendingSwitch.SourceTerminalID,
		}
		if state.PendingSwitch.TargetTerminalID != nil {
			value := *state.PendingSwitch.TargetTerminalID
			result.PendingTerminalSwitch.TargetTerminalId = &value
		}
	}
	return result
}

func coordinationStateFromPrivate(state *privatev1.CoordinationState) *domain.MasterCoordinationState {
	if state == nil {
		return nil
	}
	result := &domain.MasterCoordinationState{Revision: state.GetRevision(), PlayerConfig: playerConfigMetadataFromPrivate(state.GetPlayerConfig())}
	for _, entry := range state.GetRoster() {
		mapped := domain.MasterRosterEntry{ID: domain.CharacterID(entry.GetCharacterId()), Name: entry.GetDisplayName()}
		if entry.LogicalSessionId != nil {
			value := domain.LogicalSessionID(entry.GetLogicalSessionId())
			mapped.ClaimedBySessionID = &value
		}
		result.Roster = append(result.Roster, mapped)
	}
	for _, session := range state.GetLogicalSessions() {
		mapped := domain.MasterSessionEntry{
			ID: domain.LogicalSessionID(session.GetLogicalSessionId()), FallbackName: session.GetFallbackName(),
			Connected: session.GetConnected(), Role: playerRoleFromPrivate(session.GetRole()),
		}
		if session.CharacterId != nil {
			characterID := domain.CharacterID(session.GetCharacterId())
			mapped.Character = &domain.PlayerCharacter{ID: characterID}
			for _, roster := range result.Roster {
				if roster.ID == characterID {
					mapped.Character.Name = roster.Name
					break
				}
			}
		}
		result.Sessions = append(result.Sessions, mapped)
	}
	if state.Broadcast != nil {
		result.Broadcast = &domain.MasterBroadcastState{ID: domain.BroadcastID(state.Broadcast.GetBroadcastId())}
		if state.Broadcast.ActiveControllerSessionId != nil {
			value := domain.LogicalSessionID(state.Broadcast.GetActiveControllerSessionId())
			result.Broadcast.ControllerSessionID = &value
		}
		if state.Broadcast.ActiveTerminalId != nil {
			value := state.Broadcast.GetActiveTerminalId()
			result.Broadcast.ActiveTerminalID = &value
		}
	}
	if state.PendingTerminalSwitch != nil {
		pending := state.PendingTerminalSwitch
		result.PendingSwitch = &domain.MasterPendingSwitch{
			SwitchID: domain.SwitchID(pending.GetSwitchId()), BroadcastID: domain.BroadcastID(pending.GetBroadcastId()),
			SourceTerminalID: pending.GetSourceTerminalId(),
		}
		if pending.TargetTerminalId != nil {
			value := pending.GetTargetTerminalId()
			result.PendingSwitch.TargetTerminalID = &value
		}
	}
	return result
}

func serverInfoToPrivate(info domain.ServerInfo) *privatev1.ServerInformation {
	localURL := info.LocalURL
	if localURL == "" && !info.Tunnel {
		localURL = info.URL
	}
	result := &privatev1.ServerInformation{LocalUrl: localURL, TunnelEnabled: info.Tunnel, Ip: info.IP, Port: int32(info.Port), Url: info.URL}
	if info.Tunnel && info.URL != "" {
		result.PublicUrl = &info.URL
	}
	if info.TunnelError != "" {
		result.TunnelError = &info.TunnelError
	}
	return result
}

func serverInfoFromPrivate(info *privatev1.ServerInformation) domain.ServerInfo {
	if info == nil {
		return domain.ServerInfo{}
	}
	result := domain.ServerInfo{IP: info.GetIp(), Port: int(info.GetPort()), URL: info.GetUrl(), LocalURL: info.GetLocalUrl(), Tunnel: info.GetTunnelEnabled(), TunnelError: info.GetTunnelError()}
	return result
}

func playerConfigMetadataToPrivate(metadata *domain.PlayerConfigMetadata) *privatev1.PlayerConfigMetadata {
	if metadata == nil {
		return nil
	}
	return &privatev1.PlayerConfigMetadata{Status: metadata.Status, FilePath: metadata.FilePath, Version: int32(metadata.Version), Name: metadata.Name}
}

func playerConfigMetadataFromPrivate(metadata *privatev1.PlayerConfigMetadata) *domain.PlayerConfigMetadata {
	if metadata == nil {
		return nil
	}
	return &domain.PlayerConfigMetadata{Status: metadata.GetStatus(), FilePath: metadata.GetFilePath(), Version: int(metadata.GetVersion()), Name: metadata.GetName()}
}

func publicHackToPrivate(state *domain.PublicHackState) *playerv1.PublicHackState {
	if state == nil {
		return nil
	}
	result := &playerv1.PublicHackState{
		Level: int32(state.Level), WordLength: int32(state.WordLength), AttemptsMax: int32(state.AttemptsMax),
		AttemptsLeft: int32(state.AttemptsLeft), Solved: state.Solved, Failed: state.Failed, Log: append([]string(nil), state.Log...),
	}
	for _, column := range state.Columns {
		mapped := &playerv1.PublicHackColumn{Addresses: append([]string(nil), column.Addresses...), Text: column.Text}
		for _, word := range column.Words {
			mapped.Words = append(mapped.Words, &playerv1.PublicHackWord{Id: word.ID, Start: int32(word.Start), Length: int32(word.Length)})
		}
		result.Columns = append(result.Columns, mapped)
	}
	for _, pattern := range state.Patterns {
		result.Patterns = append(result.Patterns, &playerv1.PublicHackPattern{PatternId: pattern.ID, Row: int32(pattern.Row), Start: int32(pattern.Start), End: int32(pattern.End), Used: pattern.Used})
	}
	return result
}

func publicHackFromPrivate(state *playerv1.PublicHackState) *domain.PublicHackState {
	if state == nil {
		return nil
	}
	result := &domain.PublicHackState{
		Level: int(state.GetLevel()), WordLength: int(state.GetWordLength()), AttemptsMax: int(state.GetAttemptsMax()),
		AttemptsLeft: int(state.GetAttemptsLeft()), Solved: state.GetSolved(), Failed: state.GetFailed(), Log: append([]string(nil), state.GetLog()...),
	}
	for _, column := range state.GetColumns() {
		mapped := domain.HackColumn{Addresses: append([]string(nil), column.GetAddresses()...), Text: column.GetText()}
		for _, word := range column.GetWords() {
			mapped.Words = append(mapped.Words, domain.HackWord{ID: word.GetId(), Start: int(word.GetStart()), Length: int(word.GetLength())})
		}
		result.Columns = append(result.Columns, mapped)
	}
	for _, pattern := range state.GetPatterns() {
		result.Patterns = append(result.Patterns, domain.PublicHackPattern{ID: pattern.GetPatternId(), Row: int(pattern.GetRow()), Start: int(pattern.GetStart()), End: int(pattern.GetEnd()), Used: pattern.GetUsed()})
	}
	return result
}

func playerRoleToPrivate(role domain.PlayerRole) playerv1.PlayerRole {
	switch role {
	case domain.PlayerRoleUnassigned:
		return playerv1.PlayerRole_PLAYER_ROLE_UNASSIGNED
	case domain.PlayerRoleActive:
		return playerv1.PlayerRole_PLAYER_ROLE_ACTIVE
	case domain.PlayerRoleObserver:
		return playerv1.PlayerRole_PLAYER_ROLE_OBSERVER
	default:
		return playerv1.PlayerRole_PLAYER_ROLE_UNSPECIFIED
	}
}

func playerRoleFromPrivate(role playerv1.PlayerRole) domain.PlayerRole {
	switch role {
	case playerv1.PlayerRole_PLAYER_ROLE_UNASSIGNED:
		return domain.PlayerRoleUnassigned
	case playerv1.PlayerRole_PLAYER_ROLE_ACTIVE:
		return domain.PlayerRoleActive
	case playerv1.PlayerRole_PLAYER_ROLE_OBSERVER:
		return domain.PlayerRoleObserver
	default:
		return ""
	}
}

func terminalSwitchStatusToPrivate(status string) privatev1.TerminalSwitchStatus {
	switch status {
	case "activated":
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_ACTIVATED
	case "cleared":
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_CLEARED
	case "pending":
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_PENDING
	case "preserved":
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_PRESERVED
	case "discarded":
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_DISCARDED
	case "cancelled", "canceled":
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_CANCELED
	case "decision-required":
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_DECISION_REQUIRED
	default:
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_UNSPECIFIED
	}
}

func terminalSwitchStatusFromPrivate(status privatev1.TerminalSwitchStatus) string {
	switch status {
	case privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_ACTIVATED:
		return "activated"
	case privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_CLEARED:
		return "cleared"
	case privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_PENDING:
		return "pending"
	case privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_PRESERVED:
		return "preserved"
	case privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_DISCARDED:
		return "discarded"
	case privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_CANCELED:
		return "cancelled"
	case privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_DECISION_REQUIRED:
		return "decision-required"
	default:
		return ""
	}
}

func terminalSwitchChoiceToPrivate(choice domain.TerminalSwitchChoice) (privatev1.TerminalSwitchChoice, error) {
	switch choice {
	case domain.TerminalSwitchPreserve:
		return privatev1.TerminalSwitchChoice_TERMINAL_SWITCH_CHOICE_PRESERVE, nil
	case domain.TerminalSwitchDiscard:
		return privatev1.TerminalSwitchChoice_TERMINAL_SWITCH_CHOICE_DISCARD, nil
	case domain.TerminalSwitchCancel:
		return privatev1.TerminalSwitchChoice_TERMINAL_SWITCH_CHOICE_CANCEL, nil
	default:
		return privatev1.TerminalSwitchChoice_TERMINAL_SWITCH_CHOICE_UNSPECIFIED, fmt.Errorf("unsupported terminal switch choice %q", choice)
	}
}

func terminalSwitchChoiceFromPrivate(choice privatev1.TerminalSwitchChoice) domain.TerminalSwitchChoice {
	switch choice {
	case privatev1.TerminalSwitchChoice_TERMINAL_SWITCH_CHOICE_PRESERVE:
		return domain.TerminalSwitchPreserve
	case privatev1.TerminalSwitchChoice_TERMINAL_SWITCH_CHOICE_DISCARD:
		return domain.TerminalSwitchDiscard
	case privatev1.TerminalSwitchChoice_TERMINAL_SWITCH_CHOICE_CANCEL:
		return domain.TerminalSwitchCancel
	default:
		return ""
	}
}
