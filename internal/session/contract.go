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
		result.Terminals = append(result.Terminals, &persistencev1.Terminal{
			Id: terminal.ID, Name: terminal.Name, HackLevel: int32(terminal.HackLevel), IntroText: terminal.IntroText, Root: root,
		})
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
		result.Terminals = append(result.Terminals, domain.Terminal{
			ID: terminal.GetId(), Name: terminal.GetName(), HackLevel: int(terminal.GetHackLevel()), IntroText: terminal.GetIntroText(), Root: root, Extra: terminalTemplate.Extra,
		})
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
		result.Content = &persistencev1.ContentNode_Command{Command: &persistencev1.CommandContent{Text: node.Text}}
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
	case *persistencev1.ContentNode_Entry:
		result.Type, result.Description = domain.NodeEntry, content.Entry.GetDescription()
	default:
		return domain.ContentNode{}, fmt.Errorf("content node variant is required")
	}
	return result, nil
}
