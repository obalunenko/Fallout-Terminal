package domain

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	// MaxTerminals bounds terminals in one version-1 session.
	MaxTerminals = 1000
	// MaxNodeDepth counts the root as depth one.
	MaxNodeDepth = 64
	// MaxNodes bounds recursive content nodes within one terminal.
	MaxNodes = 100000
	// MaxRosterEntries bounds authored characters in one player config.
	MaxRosterEntries = 1000
	// MaxCharacterNameRunes is the shared player-facing character-name limit.
	MaxCharacterNameRunes = 80
	maxNameBytes          = 256
	maxIntroBytes         = 64 * 1024
	maxBodyBytes          = 1024 * 1024
)

// ValidateSession validates every known version-1 field without mutating data.
func ValidateSession(session Session) error {
	if session.Version != 1 {
		return fmt.Errorf("version must be 1")
	}
	if err := validateRequiredString("name", session.Name, maxNameBytes); err != nil {
		return err
	}
	if session.Terminals == nil {
		return fmt.Errorf("terminals must be an array")
	}
	if session.PlayerConfig != "" {
		if filepath.IsAbs(session.PlayerConfig) || filepath.Clean(session.PlayerConfig) == "." {
			return fmt.Errorf("playerConfig must be a relative file path")
		}
		if strings.ContainsRune(session.PlayerConfig, '\x00') {
			return fmt.Errorf("playerConfig contains an invalid path character")
		}
	}
	if len(session.Terminals) > MaxTerminals {
		return fmt.Errorf("terminals exceeds limit %d", MaxTerminals)
	}
	if err := validateExtras("session", session.Extra, sessionFields); err != nil {
		return err
	}

	terminalIDs := make(map[string]struct{}, len(session.Terminals))
	for index := range session.Terminals {
		terminal := session.Terminals[index]
		path := fmt.Sprintf("terminals[%d]", index)
		if err := validateRequiredString(path+".id", terminal.ID, maxNameBytes); err != nil {
			return err
		}
		if _, exists := terminalIDs[terminal.ID]; exists {
			return fmt.Errorf("%s.id duplicates %q", path, terminal.ID)
		}
		terminalIDs[terminal.ID] = struct{}{}
		if err := validateRequiredString(path+".name", terminal.Name, maxNameBytes); err != nil {
			return err
		}
		if terminal.HackLevel < 0 || terminal.HackLevel > 5 {
			return fmt.Errorf("%s.hackLevel must be between 0 and 5", path)
		}
		if len([]byte(terminal.IntroText)) > maxIntroBytes {
			return fmt.Errorf("%s.introText exceeds %d bytes", path, maxIntroBytes)
		}
		if terminal.Root.ID != "root" || terminal.Root.Type != NodeFolder {
			return fmt.Errorf("%s.root must be folder root", path)
		}
		if err := validateExtras(path, terminal.Extra, terminalFields); err != nil {
			return err
		}
		if err := validateTree(path+".root", terminal.Root); err != nil {
			return err
		}
	}
	return nil
}

// ValidatePlayerConfig validates a complete standalone authored roster.
func ValidatePlayerConfig(config PlayerConfig) error {
	if config.Version != 1 {
		return fmt.Errorf("version must be 1")
	}
	if err := validateRequiredString("name", config.Name, maxNameBytes); err != nil {
		return err
	}
	if config.Roster == nil {
		return fmt.Errorf("roster must be an array")
	}
	if len(config.Roster) > MaxRosterEntries {
		return fmt.Errorf("roster exceeds limit %d", MaxRosterEntries)
	}
	ids := make(map[CharacterID]struct{}, len(config.Roster))
	for index, entry := range config.Roster {
		path := fmt.Sprintf("roster[%d]", index)
		if err := validateRequiredString(path+".id", string(entry.ID), maxNameBytes); err != nil {
			return err
		}
		if _, exists := ids[entry.ID]; exists {
			return fmt.Errorf("%s.id duplicates %q", path, entry.ID)
		}
		ids[entry.ID] = struct{}{}
		if strings.TrimSpace(entry.Name) == "" {
			return fmt.Errorf("%s.name must not be blank", path)
		}
		if utf8.RuneCountInString(entry.Name) > MaxCharacterNameRunes {
			return fmt.Errorf("%s.name exceeds %d characters", path, MaxCharacterNameRunes)
		}
	}
	return nil
}

func validateTree(path string, root ContentNode) error {
	ids := make(map[string]struct{})
	count := 0
	var visit func(string, ContentNode, int) error
	visit = func(nodePath string, node ContentNode, depth int) error {
		count++
		if count > MaxNodes {
			return fmt.Errorf("%s exceeds node limit %d", path, MaxNodes)
		}
		if depth > MaxNodeDepth {
			return fmt.Errorf("%s exceeds depth limit %d", nodePath, MaxNodeDepth)
		}
		if err := validateRequiredString(nodePath+".id", node.ID, maxNameBytes); err != nil {
			return err
		}
		if _, exists := ids[node.ID]; exists {
			return fmt.Errorf("%s.id duplicates %q", nodePath, node.ID)
		}
		ids[node.ID] = struct{}{}
		if err := validateRequiredString(nodePath+".name", node.Name, maxNameBytes); err != nil {
			return err
		}
		if err := validateExtras(nodePath, node.Extra, nodeFields); err != nil {
			return err
		}

		switch node.Type {
		case NodeFolder:
			if node.Children == nil {
				return fmt.Errorf("%s.children must be an array", nodePath)
			}
			if node.Text != "" || node.Description != "" {
				return fmt.Errorf("%s folder cannot contain leaf body fields", nodePath)
			}
			for index := range node.Children {
				if err := visit(fmt.Sprintf("%s.children[%d]", nodePath, index), node.Children[index], depth+1); err != nil {
					return err
				}
			}
		case NodeCommand:
			if len(node.Children) != 0 || node.Description != "" {
				return fmt.Errorf("%s command must be a leaf", nodePath)
			}
			if len([]byte(node.Text)) > maxBodyBytes {
				return fmt.Errorf("%s.text exceeds %d bytes", nodePath, maxBodyBytes)
			}
		case NodeEntry:
			if len(node.Children) != 0 || node.Text != "" {
				return fmt.Errorf("%s entry must be a leaf", nodePath)
			}
			if len([]byte(node.Description)) > maxBodyBytes {
				return fmt.Errorf("%s.description exceeds %d bytes", nodePath, maxBodyBytes)
			}
		default:
			return fmt.Errorf("%s.type %q is unsupported", nodePath, node.Type)
		}
		return nil
	}
	return visit(path, root, 1)
}

func validateRequiredString(path, value string, maxBytes int) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s must not be blank", path)
	}
	if len([]byte(value)) > maxBytes {
		return fmt.Errorf("%s exceeds %d bytes", path, maxBytes)
	}
	return nil
}

func validateExtras(path string, extras map[string]json.RawMessage, known map[string]struct{}) error {
	for field, value := range extras {
		if _, exists := known[field]; exists {
			return fmt.Errorf("%s extra field %q shadows a known field", path, field)
		}
		if !json.Valid(value) {
			return fmt.Errorf("%s extra field %q contains invalid JSON", path, field)
		}
	}
	return nil
}
