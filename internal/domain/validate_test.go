package domain

import (
	"fmt"
	"strings"
	"testing"
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
			if err := ValidateSession(candidate); err == nil {
				t.Fatal("ValidateSession() error = nil")
			}
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
	if err := ValidateSession(Session{Version: 1, Name: "Too many", Terminals: terminals}); err == nil {
		t.Fatal("ValidateSession() accepted too many terminals")
	}

	root := ContentNode{ID: "root", Type: NodeFolder, Name: "ROOT", Children: []ContentNode{}}
	cursor := &root
	for depth := 1; depth <= MaxNodeDepth; depth++ {
		cursor.Children = []ContentNode{{ID: fmt.Sprintf("f%d", depth), Type: NodeFolder, Name: "Folder", Children: []ContentNode{}}}
		cursor = &cursor.Children[0]
	}
	tooDeep := Session{Version: 1, Name: "Deep", Terminals: []Terminal{{ID: "t1", Name: "T", Root: root}}}
	if err := ValidateSession(tooDeep); err == nil {
		t.Fatal("ValidateSession() accepted excessive nesting")
	}
}

func cloneSession(session Session) Session {
	clone := session
	clone.Terminals = append([]Terminal(nil), session.Terminals...)
	for i := range clone.Terminals {
		clone.Terminals[i].Root = cloneNode(session.Terminals[i].Root)
	}
	return clone
}

func cloneNode(node ContentNode) ContentNode {
	clone := node
	clone.Children = make([]ContentNode, len(node.Children))
	for i := range node.Children {
		clone.Children[i] = cloneNode(node.Children[i])
	}
	return clone
}
