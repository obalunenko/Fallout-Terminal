package main

import (
	"fmt"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	playerv1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/player/v1"
	privatev1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/private/v1"
	playerconfigservice "github.com/obalunenko/Fallout-Terminal/internal/playerconfig"
	sessionservice "github.com/obalunenko/Fallout-Terminal/internal/session"
)

// The private protobuf graph governs trusted desktop semantics only. These
// adapters are invoked inside App while Wails continues carrying the existing
// native DTOs; no protobuf bytes, ProtoJSON, or generic envelope crosses Wails.

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

func commandResultToPrivate(result CommandResult) *privatev1.CommandResult {
	semantic := &privatev1.CommandResult{Ok: result.OK}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	return semantic
}

func routeCommandResult(result CommandResult) CommandResult {
	_ = commandResultToPrivate(result)
	return result
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
	return result
}

func routeSaveSessionResult(result sessionservice.SaveResult) sessionservice.SaveResult {
	semantic := &privatev1.SaveSessionResult{Ok: result.OK, RequestedRevision: result.RequestedRevision}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	if result.SavedRevision != 0 {
		semantic.SavedRevision = &result.SavedRevision
	}
	return result
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
		semantic.PlayerConfig, _ = playerconfigservice.PlayerConfigToProto(domain.PlayerConfig{Version: result.Config.Version, Name: result.Config.Name, Roster: []domain.CharacterRosterEntry{}})
	}
	return result
}

func coordinationResultToPrivate(result CoordinationCommandResult) *privatev1.CoordinationResult {
	semantic := &privatev1.CoordinationResult{Ok: result.OK, State: coordinationStateToPrivate(result.State)}
	if result.Error != "" {
		semantic.Error = &result.Error
	}
	return semantic
}

func routeCoordinationResult(result CoordinationCommandResult) CoordinationCommandResult {
	_ = coordinationResultToPrivate(result)
	return result
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
	_ = terminalSwitchResultToPrivate(result)
	return result
}

func routeServerInfoEvent(info domain.ServerInfo) domain.ServerInfo {
	_ = &privatev1.ServerInformationEvent{ServerInfo: serverInfoToPrivate(info)}
	return info
}

func routeClientCountEvent(count int) int {
	_ = &privatev1.ClientCountEvent{ClientCount: uint32(max(count, 0))}
	return count
}

func routeHackStateEvent(state *domain.PublicHackState) *domain.PublicHackState {
	_ = &privatev1.HackStateEvent{HackState: publicHackToPrivate(state)}
	return state
}

func routeCoordinationEvent(state *domain.MasterCoordinationState) *domain.MasterCoordinationState {
	_ = &privatev1.CoordinationStateEvent{CoordinationState: coordinationStateToPrivate(state)}
	return state
}

func coordinationStateToPrivate(state *domain.MasterCoordinationState) *privatev1.CoordinationState {
	if state == nil {
		return nil
	}
	result := &privatev1.CoordinationState{
		Roster:          make([]*privatev1.CharacterState, 0, len(state.Roster)),
		LogicalSessions: make([]*privatev1.LogicalSessionState, 0, len(state.Sessions)),
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
		mapped := &privatev1.LogicalSessionState{LogicalSessionId: string(session.ID), FallbackName: session.FallbackName, Connected: session.Connected}
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
		result.PendingTerminalSwitch = &privatev1.PendingTerminalSwitch{SwitchId: string(state.PendingSwitch.SwitchID), TerminalId: targetID}
	}
	return result
}

func serverInfoToPrivate(info domain.ServerInfo) *privatev1.ServerInformation {
	localURL := info.LocalURL
	if localURL == "" && !info.Tunnel {
		localURL = info.URL
	}
	result := &privatev1.ServerInformation{LocalUrl: localURL, TunnelEnabled: info.Tunnel}
	if info.Tunnel && info.URL != "" {
		result.PublicUrl = &info.URL
	}
	return result
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
	default:
		return privatev1.TerminalSwitchStatus_TERMINAL_SWITCH_STATUS_UNSPECIFIED
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
