package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSessionRejectsInvalidKnownShape(t *testing.T) {
	t.Parallel()

	validRoot := ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{}}
	valid := Session{
		Version: 1,
		Name:    "Campaign",
		Terminals: []Terminal{{
			ID: "t1", Name: "Terminal", HackLevel: 0, Root: validRoot,
		}},
	}

	tests := []struct {
		name   string
		mutate func(*Session)
	}{
		{name: "unsupported version", mutate: func(s *Session) { s.Version = 2 }},
		{name: "blank session name", mutate: func(s *Session) { s.Name = "  " }},
		{name: "session name bytes", mutate: func(s *Session) { s.Name = strings.Repeat("é", 129) }},
		{name: "blank terminal id", mutate: func(s *Session) { s.Terminals[0].ID = "" }},
		{name: "hack level", mutate: func(s *Session) { s.Terminals[0].HackLevel = 6 }},
		{name: "invalid root id", mutate: func(s *Session) { s.Terminals[0].Root.ID = "not-root" }},
		{name: "invalid root type", mutate: func(s *Session) { s.Terminals[0].Root.Type = NodeEntry }},
		{name: "unknown node type", mutate: func(s *Session) {
			s.Terminals[0].Root.Children = []ContentNode{{ID: "mystery", Type: "mystery", Name: "?"}}
		}},
		{name: "duplicate terminal id", mutate: func(s *Session) { s.Terminals = append(s.Terminals, s.Terminals[0]) }},
		{name: "duplicate node id", mutate: func(s *Session) {
			s.Terminals[0].Root.Children = []ContentNode{
				{ID: "n1", Type: NodeEntry, Name: "A", Description: "a"},
				{ID: "n1", Type: NodeCommand, Name: "B", Text: "b"},
			}
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := cloneSession(valid)
			test.mutate(&candidate)
			require.Error(t, ValidateSession(candidate))
		})
	}
}

func TestValidateSessionRejectsDocumentLimits(t *testing.T) {
	t.Parallel()

	terminals := make([]Terminal, MaxTerminals+1)
	for i := range terminals {
		terminals[i] = Terminal{
			ID:   fmt.Sprintf("t%d", i),
			Name: "Terminal",
			Root: ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{}},
		}
	}
	assert.Error(t, ValidateSession(Session{Version: 1, Name: "Too many", Terminals: terminals}))

	root := ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{}}
	cursor := &root
	for depth := 1; depth <= MaxNodeDepth; depth++ {
		cursor.Children = []ContentNode{{ID: fmt.Sprintf("f%d", depth), Type: NodeFolder, Name: "Folder", Children: []ContentNode{}}}
		cursor = &cursor.Children[0]
	}
	tooDeep := Session{Version: 1, Name: "Deep", Terminals: []Terminal{{ID: "t1", Name: "T", Root: root}}}
	assert.Error(t, ValidateSession(tooDeep))
}

func TestValidatePlayerConfigAcceptsIntelligenceBoundaries(t *testing.T) {
	t.Parallel()

	for _, intelligence := range []int{1, 10} {
		intelligence := intelligence
		t.Run(fmt.Sprintf("intelligence %d", intelligence), func(t *testing.T) {
			t.Parallel()
			config := PlayerConfig{
				Version: 1,
				Name:    "Players",
				Roster: []CharacterRosterEntry{{
					ID: "mara", Name: "Mara", Intelligence: intelligence, HackerPerkAvailable: true,
				}},
			}
			require.NoError(t, ValidatePlayerConfig(config))
		})
	}
}

func TestValidatePlayerConfigRejectsOutOfRangeIntelligence(t *testing.T) {
	t.Parallel()

	for _, intelligence := range []int{0, 11} {
		intelligence := intelligence
		t.Run(fmt.Sprintf("intelligence %d", intelligence), func(t *testing.T) {
			t.Parallel()
			config := PlayerConfig{
				Version: 1,
				Name:    "Players",
				Roster: []CharacterRosterEntry{{
					ID: "mara", Name: "Mara", Intelligence: intelligence, HackerPerkAvailable: false,
				}},
			}
			require.ErrorContains(t, ValidatePlayerConfig(config), "intelligence")
		})
	}
}

func TestValidateSessionAcceptsStateChangingCommandStateByStableID(t *testing.T) {
	t.Parallel()

	session := validStateChangingSessionForTest()
	command := session.Terminals[0].Root.Children[0].Children[0]
	command.Name = "Renamed authored command"
	command.StateChange.CompletedName = "Renamed authored completed title"
	session.Terminals[0].Root.Children = []ContentNode{
		{ID: "other", Type: NodeFolder, Name: "OTHER", Children: []ContentNode{}},
		{ID: "moved", Type: NodeFolder, Name: "MOVED", Children: []ContentNode{command}},
	}

	require.NoError(t, ValidateSession(session))
	assert.Contains(t, session.Terminals[0].CommandStates, command.ID)
}

func TestValidateSessionRejectsInvalidStateChangingCommandShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Session)
	}{
		{name: "state change on folder", mutate: func(s *Session) {
			s.Terminals[0].Root.StateChange = &StateChangeConfig{CompletedName: "Done", ConfirmationText: "Proceed?"}
		}},
		{name: "state change on entry", mutate: func(s *Session) {
			node := &s.Terminals[0].Root.Children[0].Children[0]
			node.Type, node.Text, node.Description = NodeEntry, "", "Description"
		}},
		{name: "blank authored result", mutate: func(s *Session) {
			s.Terminals[0].Root.Children[0].Children[0].Text = " \t"
		}},
		{name: "blank completed name", mutate: func(s *Session) {
			s.Terminals[0].Root.Children[0].Children[0].StateChange.CompletedName = " \t"
		}},
		{name: "blank confirmation text", mutate: func(s *Session) {
			s.Terminals[0].Root.Children[0].Children[0].StateChange.ConfirmationText = "\n"
		}},
		{name: "completed name exceeds name limit", mutate: func(s *Session) {
			s.Terminals[0].Root.Children[0].Children[0].StateChange.CompletedName = strings.Repeat("x", maxNameBytes+1)
		}},
		{name: "confirmation exceeds command body limit", mutate: func(s *Session) {
			s.Terminals[0].Root.Children[0].Children[0].StateChange.ConfirmationText = strings.Repeat("x", maxBodyBytes+1)
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := cloneSession(validStateChangingSessionForTest())
			test.mutate(&candidate)
			require.Error(t, ValidateSession(candidate))
		})
	}
}

func TestValidateSessionRejectsInvalidCommandExecutionStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Session)
	}{
		{name: "orphaned command ID", mutate: func(s *Session) {
			s.Terminals[0].CommandStates["missing-command"] = s.Terminals[0].CommandStates["doors"]
			delete(s.Terminals[0].CommandStates, "doors")
		}},
		{name: "new command ID does not inherit old state", mutate: func(s *Session) {
			s.Terminals[0].Root.Children[0].Children[0].ID = "replacement-doors"
		}},
		{name: "state points to folder", mutate: func(s *Session) {
			s.Terminals[0].CommandStates["section"] = s.Terminals[0].CommandStates["doors"]
			delete(s.Terminals[0].CommandStates, "doors")
		}},
		{name: "state points to ordinary command", mutate: func(s *Session) {
			node := &s.Terminals[0].Root.Children[0].Children[0]
			node.StateChange = nil
		}},
		{name: "state belongs to another terminal", mutate: func(s *Session) {
			command := s.Terminals[0].Root.Children[0].Children[0]
			s.Terminals[0].Root.Children[0].Children = nil
			s.Terminals = append(s.Terminals, Terminal{
				ID: "t2", Name: "Second terminal",
				Root: ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{command}},
			})
		}},
		{name: "blank snapshot completed name", mutate: func(s *Session) {
			state := s.Terminals[0].CommandStates["doors"]
			state.CompletedName = " "
			s.Terminals[0].CommandStates["doors"] = state
		}},
		{name: "blank snapshot result", mutate: func(s *Session) {
			state := s.Terminals[0].CommandStates["doors"]
			state.ResultText = "\n"
			s.Terminals[0].CommandStates["doors"] = state
		}},
		{name: "snapshot completed name exceeds name limit", mutate: func(s *Session) {
			state := s.Terminals[0].CommandStates["doors"]
			state.CompletedName = strings.Repeat("x", maxNameBytes+1)
			s.Terminals[0].CommandStates["doors"] = state
		}},
		{name: "snapshot result exceeds command body limit", mutate: func(s *Session) {
			state := s.Terminals[0].CommandStates["doors"]
			state.ResultText = strings.Repeat("x", maxBodyBytes+1)
			s.Terminals[0].CommandStates["doors"] = state
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := cloneSession(validStateChangingSessionForTest())
			test.mutate(&candidate)
			require.Error(t, ValidateSession(candidate))
		})
	}
}

func TestValidateSessionRejectsStateChangeKnownFieldsInExtras(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		mutate func(*Session)
	}{
		{name: "node stateChange", mutate: func(s *Session) {
			node := &s.Terminals[0].Root.Children[0].Children[0]
			node.Extra = map[string]json.RawMessage{"stateChange": json.RawMessage(`{"future":true}`)}
		}},
		{name: "terminal commandStates", mutate: func(s *Session) {
			s.Terminals[0].Extra = map[string]json.RawMessage{"commandStates": json.RawMessage(`{}`)}
		}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := cloneSession(validStateChangingSessionForTest())
			test.mutate(&candidate)
			require.Error(t, ValidateSession(candidate))
		})
	}
}

func TestValidateSessionResolvesTerminalTransitionsInTwoPasses(t *testing.T) {
	t.Parallel()

	linked := Session{Version: 1, Name: "Links", Terminals: []Terminal{
		{
			ID: "a", Name: "A",
			Root: ContentNode{
				ID: "root", Type: NodeFolder, Name: "ROOT",
				Children: []ContentNode{{
					ID: "go", Type: NodeCommand, Name: "GO",
					TerminalTransition: &TerminalTransitionConfig{TargetTerminalID: "b"},
				}},
			},
		},
		{
			ID: "b", Name: "B",
			Root: ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{}},
		},
	}}
	require.NoError(t, ValidateSession(linked), "a forward reference must not depend on terminal ordering")

	for _, test := range []struct {
		name   string
		mutate func(*Session)
	}{
		{name: "missing target", mutate: func(s *Session) { s.Terminals[0].Root.Children[0].TerminalTransition.TargetTerminalID = "missing" }},
		{name: "self target", mutate: func(s *Session) { s.Terminals[0].Root.Children[0].TerminalTransition.TargetTerminalID = "a" }},
		{name: "blank target", mutate: func(s *Session) { s.Terminals[0].Root.Children[0].TerminalTransition.TargetTerminalID = " " }},
		{name: "state change conflict", mutate: func(s *Session) {
			s.Terminals[0].Root.Children[0].StateChange = &StateChangeConfig{CompletedName: "Done", ConfirmationText: "Proceed?"}
			s.Terminals[0].Root.Children[0].Text = "Done"
		}},
		{name: "config on folder", mutate: func(s *Session) {
			s.Terminals[0].Root.TerminalTransition = &TerminalTransitionConfig{TargetTerminalID: "b"}
		}},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			candidate := CloneSession(linked)
			test.mutate(&candidate)
			require.Error(t, ValidateSession(candidate))
		})
	}
}

func validStateChangingSessionForTest() Session {
	return Session{
		Version: 1,
		Name:    "Stateful campaign",
		Terminals: []Terminal{{
			ID: "t1", Name: "Terminal", HackLevel: 0,
			Root: ContentNode{
				ID: "root", Type: NodeFolder, Name: "ROOT",
				Children: []ContentNode{{
					ID: "section", Type: NodeFolder, Name: "SECTION",
					Children: []ContentNode{{
						ID: "doors", Type: NodeCommand, Name: "Open doors", Text: "Doors opened.",
						StateChange: &StateChangeConfig{
							CompletedName:    "Doors open",
							ConfirmationText: "Open the doors?",
						},
					}},
				}},
			},
			CommandStates: map[string]CommandExecutionState{
				"doors": {CompletedName: "Doors were opened", ResultText: "Access granted."},
			},
		}},
	}
}

func cloneSession(session Session) Session {
	clone := session
	clone.Terminals = append([]Terminal(nil), session.Terminals...)
	for i := range clone.Terminals {
		clone.Terminals[i].Root = cloneNode(session.Terminals[i].Root)
		if session.Terminals[i].CommandStates != nil {
			clone.Terminals[i].CommandStates = make(map[string]CommandExecutionState, len(session.Terminals[i].CommandStates))
			for id, state := range session.Terminals[i].CommandStates {
				clone.Terminals[i].CommandStates[id] = state
			}
		}
	}
	return clone
}

func cloneNode(node ContentNode) ContentNode {
	clone := node
	if node.StateChange != nil {
		stateChange := *node.StateChange
		clone.StateChange = &stateChange
	}
	clone.Children = make([]ContentNode, len(node.Children))
	for i := range node.Children {
		clone.Children[i] = cloneNode(node.Children[i])
	}
	return clone
}
