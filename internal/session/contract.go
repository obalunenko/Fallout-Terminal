package session

import (
	"fmt"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	persistencev1 "github.com/obalunenko/Fallout-Terminal/internal/gen/fallout/terminal/persistence/v1"
)

// SessionToProto maps only known version-1 semantics. JSON extras remain on
// the compatibility domain values and are never discarded or serialized by protobuf.
func SessionToProto(value domain.Session) (*persistencev1.Session, error) {
	if err := domain.ValidateSession(value); err != nil {
		return nil, err
	}
	result := &persistencev1.Session{Version: int32(value.Version), Name: value.Name}
	if value.PlayerConfig != "" {
		reference := value.PlayerConfig
		result.PlayerConfig = &reference
	}
	result.Terminals = make([]*persistencev1.Terminal, 0, len(value.Terminals))
	for _, terminal := range value.Terminals {
		root, err := contentNodeToProto(terminal.Root)
		if err != nil {
			return nil, err
		}
		mapped := &persistencev1.Terminal{
			Id: terminal.ID, Name: terminal.Name, HackLevel: int32(terminal.HackLevel), IntroText: terminal.IntroText, Root: root,
		}
		if len(terminal.CommandStates) != 0 {
			mapped.CommandStates = make(map[string]*persistencev1.CommandExecutionState, len(terminal.CommandStates))
			for commandID, state := range terminal.CommandStates {
				mapped.CommandStates[commandID] = &persistencev1.CommandExecutionState{
					CompletedName: state.CompletedName,
					ResultText:    state.ResultText,
				}
			}
		}
		result.Terminals = append(result.Terminals, mapped)
	}
	return result, nil
}

// SessionFromProto restores known fields and merges the caller-owned extras
// from template at the session, terminal, and recursive node levels.
func SessionFromProto(value *persistencev1.Session, template domain.Session) (domain.Session, error) {
	if value == nil {
		return domain.Session{}, fmt.Errorf("session contract is required")
	}
	result := domain.Session{Version: int(value.GetVersion()), Name: value.GetName(), PlayerConfig: value.GetPlayerConfig(), Extra: template.Extra}
	result.Terminals = make([]domain.Terminal, 0, len(value.GetTerminals()))
	for index, terminal := range value.GetTerminals() {
		var terminalTemplate domain.Terminal
		if index < len(template.Terminals) {
			terminalTemplate = template.Terminals[index]
		}
		root, err := contentNodeFromProto(terminal.GetRoot(), terminalTemplate.Root)
		if err != nil {
			return domain.Session{}, err
		}
		mapped := domain.Terminal{
			ID: terminal.GetId(), Name: terminal.GetName(), HackLevel: int(terminal.GetHackLevel()), IntroText: terminal.GetIntroText(), Root: root, Extra: terminalTemplate.Extra,
		}
		if len(terminal.GetCommandStates()) != 0 {
			mapped.CommandStates = make(map[string]domain.CommandExecutionState, len(terminal.GetCommandStates()))
			for commandID, state := range terminal.GetCommandStates() {
				mapped.CommandStates[commandID] = domain.CommandExecutionState{
					CompletedName: state.GetCompletedName(),
					ResultText:    state.GetResultText(),
				}
			}
		}
		result.Terminals = append(result.Terminals, mapped)
	}
	if err := domain.ValidateSession(result); err != nil {
		return domain.Session{}, err
	}
	return result, nil
}

func verifySessionContract(value domain.Session) error {
	semantic, err := SessionToProto(value)
	if err != nil {
		return err
	}
	_, err = SessionFromProto(semantic, value)
	return err
}

// ContentNodeToProto maps one already-validated authored tree without
// inventing a partial session around it. Cross-terminal links are resolved by
// the complete session validator at the application boundary, not by this
// shape-preserving private-contract adapter.
func ContentNodeToProto(node domain.ContentNode) (*persistencev1.ContentNode, error) {
	return contentNodeToProto(node)
}

// ContentNodeFromProto maps one authored tree while preserving JSON-only
// extension fields from its native template. It deliberately performs no
// session-wide reference validation because the surrounding terminal catalog
// is not part of this private bridge message.
func ContentNodeFromProto(node *persistencev1.ContentNode, template domain.ContentNode) (domain.ContentNode, error) {
	return contentNodeFromProto(node, template)
}

func contentNodeToProto(node domain.ContentNode) (*persistencev1.ContentNode, error) {
	result := &persistencev1.ContentNode{Id: node.ID, Name: node.Name}
	switch node.Type {
	case domain.NodeFolder:
		folder := &persistencev1.FolderContent{Children: make([]*persistencev1.ContentNode, 0, len(node.Children))}
		for _, child := range node.Children {
			mapped, err := contentNodeToProto(child)
			if err != nil {
				return nil, err
			}
			folder.Children = append(folder.Children, mapped)
		}
		result.Content = &persistencev1.ContentNode_Folder{Folder: folder}
	case domain.NodeCommand:
		command := &persistencev1.CommandContent{Text: node.Text}
		if node.StateChange != nil {
			command.StateChange = &persistencev1.StateChangeConfig{
				CompletedName:    node.StateChange.CompletedName,
				ConfirmationText: node.StateChange.ConfirmationText,
			}
		}
		if node.TerminalTransition != nil {
			command.TerminalTransition = &persistencev1.TerminalTransitionConfig{
				TargetTerminalId: node.TerminalTransition.TargetTerminalID,
			}
		}
		result.Content = &persistencev1.ContentNode_Command{Command: command}
	case domain.NodeEntry:
		result.Content = &persistencev1.ContentNode_Entry{Entry: &persistencev1.EntryContent{Description: node.Description}}
	default:
		return nil, fmt.Errorf("unsupported content node type %q", node.Type)
	}
	return result, nil
}

func contentNodeFromProto(node *persistencev1.ContentNode, template domain.ContentNode) (domain.ContentNode, error) {
	if node == nil {
		return domain.ContentNode{}, fmt.Errorf("content node is required")
	}
	result := domain.ContentNode{ID: node.GetId(), Name: node.GetName(), Extra: template.Extra}
	switch content := node.Content.(type) {
	case *persistencev1.ContentNode_Folder:
		result.Type = domain.NodeFolder
		result.Children = make([]domain.ContentNode, 0, len(content.Folder.GetChildren()))
		for index, child := range content.Folder.GetChildren() {
			var childTemplate domain.ContentNode
			if index < len(template.Children) {
				childTemplate = template.Children[index]
			}
			mapped, err := contentNodeFromProto(child, childTemplate)
			if err != nil {
				return domain.ContentNode{}, err
			}
			result.Children = append(result.Children, mapped)
		}
	case *persistencev1.ContentNode_Command:
		result.Type, result.Text = domain.NodeCommand, content.Command.GetText()
		if content.Command.GetStateChange() != nil {
			result.StateChange = &domain.StateChangeConfig{
				CompletedName:    content.Command.GetStateChange().GetCompletedName(),
				ConfirmationText: content.Command.GetStateChange().GetConfirmationText(),
			}
		}
		if content.Command.GetTerminalTransition() != nil {
			result.TerminalTransition = &domain.TerminalTransitionConfig{
				TargetTerminalID: content.Command.GetTerminalTransition().GetTargetTerminalId(),
			}
		}
	case *persistencev1.ContentNode_Entry:
		result.Type, result.Description = domain.NodeEntry, content.Entry.GetDescription()
	default:
		return domain.ContentNode{}, fmt.Errorf("content node variant is required")
	}
	return result, nil
}
