package nav

import (
	"testing"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestDefault(t *testing.T) {
	t.Parallel()

	want := domain.NavState{Path: []string{"root"}, Mode: "list"}
	assert.Equal(t, want, Default())
}

func TestApplyAction(t *testing.T) {
	t.Parallel()

	tree := navigationTree()
	tests := []struct {
		name   string
		state  domain.NavState
		action string
		nodeID string
		want   domain.NavState
	}{
		{
			name:   "enter direct folder child",
			state:  Default(),
			action: "enter",
			nodeID: "docs",
			want:   domain.NavState{Path: []string{"root", "docs"}, Mode: "list"},
		},
		{
			name:   "reject descendant folder that is not a direct child",
			state:  Default(),
			action: "enter",
			nodeID: "nested",
			want:   Default(),
		},
		{
			name:   "reject non-folder enter target",
			state:  Default(),
			action: "enter",
			nodeID: "root-entry",
			want:   Default(),
		},
		{
			name:   "open direct entry child and clear command",
			state:  navState([]string{"root"}, "list", "", "root-command"),
			action: "entry",
			nodeID: "root-entry",
			want:   navState([]string{"root"}, "entry", "root-entry", ""),
		},
		{
			name:   "reject entry outside current folder",
			state:  Default(),
			action: "entry",
			nodeID: "report",
			want:   Default(),
		},
		{
			name:   "run direct command child",
			state:  Default(),
			action: "command",
			nodeID: "root-command",
			want:   navState([]string{"root"}, "list", "", "root-command"),
		},
		{
			name:   "reject command outside current folder",
			state:  Default(),
			action: "command",
			nodeID: "read",
			want:   Default(),
		},
		{
			name:   "back closes entry before leaving folder",
			state:  navState([]string{"root", "docs"}, "entry", "report", ""),
			action: "back",
			want:   domain.NavState{Path: []string{"root", "docs"}, Mode: "list"},
		},
		{
			name:   "back leaves folder and clears command",
			state:  navState([]string{"root", "docs"}, "list", "", "read"),
			action: "back",
			want:   Default(),
		},
		{
			name:   "back never escapes root",
			state:  Default(),
			action: "back",
			want:   Default(),
		},
		{
			name:   "back at root closes command result",
			state:  navState([]string{"root"}, "list", "", "root-command"),
			action: "back",
			want:   Default(),
		},
		{
			name:   "unknown action is a no-op",
			state:  Default(),
			action: "launch",
			nodeID: "root-command",
			want:   Default(),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, ApplyAction(test.state, tree, test.action, test.nodeID))
		})
	}
}

func TestRevalidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state domain.NavState
		tree  domain.ContentNode
		want  domain.NavState
	}{
		{
			name:  "keep valid direct-child entry",
			state: navState([]string{"root", "docs"}, "entry", "report", ""),
			tree:  navigationTree(),
			want:  navState([]string{"root", "docs"}, "entry", "report", ""),
		},
		{
			name:  "truncate path at first missing folder",
			state: navState([]string{"root", "docs", "missing"}, "list", "", "read"),
			tree:  navigationTree(),
			want:  domain.NavState{Path: []string{"root", "docs"}, Mode: "list", CommandNodeID: stringPointer("read")},
		},
		{
			name:  "reject a non-folder path component",
			state: navState([]string{"root", "root-entry"}, "entry", "root-entry", ""),
			tree:  navigationTree(),
			want:  navState([]string{"root"}, "entry", "root-entry", ""),
		},
		{
			name:  "drop deleted entry",
			state: navState([]string{"root", "docs"}, "entry", "report", ""),
			tree:  treeWithout("report"),
			want:  domain.NavState{Path: []string{"root", "docs"}, Mode: "list"},
		},
		{
			name:  "drop entry moved outside current folder",
			state: navState([]string{"root", "docs"}, "entry", "report", ""),
			tree:  treeWithReportMovedToArchive(),
			want:  domain.NavState{Path: []string{"root", "docs"}, Mode: "list"},
		},
		{
			name:  "drop command moved outside current folder",
			state: navState([]string{"root", "docs"}, "list", "", "read"),
			tree:  treeWithReadMovedToArchive(),
			want:  domain.NavState{Path: []string{"root", "docs"}, Mode: "list"},
		},
		{
			name:  "entry mode requires an entry id",
			state: domain.NavState{Path: []string{"root", "docs"}, Mode: "entry"},
			tree:  navigationTree(),
			want:  domain.NavState{Path: []string{"root", "docs"}, Mode: "list"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, Revalidate(test.state, test.tree))
		})
	}
}

func navigationTree() domain.ContentNode {
	return domain.ContentNode{
		ID: "root", Type: domain.NodeFolder, Name: "ROOT",
		Children: []domain.ContentNode{
			{
				ID: "docs", Type: domain.NodeFolder, Name: "DOCS",
				Children: []domain.ContentNode{
					{ID: "report", Type: domain.NodeEntry, Name: "REPORT", Description: "Report"},
					{ID: "read", Type: domain.NodeCommand, Name: "READ", Text: "Reading"},
					{
						ID: "nested", Type: domain.NodeFolder, Name: "NESTED",
						Children: []domain.ContentNode{
							{ID: "deep-entry", Type: domain.NodeEntry, Name: "DEEP", Description: "Deep"},
						},
					},
				},
			},
			{
				ID: "archive", Type: domain.NodeFolder, Name: "ARCHIVE",
				Children: []domain.ContentNode{
					{ID: "old-entry", Type: domain.NodeEntry, Name: "OLD", Description: "Old"},
				},
			},
			{ID: "root-entry", Type: domain.NodeEntry, Name: "WELCOME", Description: "Welcome"},
			{ID: "root-command", Type: domain.NodeCommand, Name: "STATUS", Text: "Online"},
		},
	}
}

func treeWithout(nodeID string) domain.ContentNode {
	tree := navigationTree()
	removeNode(&tree, nodeID)
	return tree
}

func treeWithReportMovedToArchive() domain.ContentNode {
	tree := treeWithout("report")
	tree.Children[1].Children = append(tree.Children[1].Children, domain.ContentNode{
		ID: "report", Type: domain.NodeEntry, Name: "REPORT", Description: "Moved report",
	})
	return tree
}

func treeWithReadMovedToArchive() domain.ContentNode {
	tree := treeWithout("read")
	tree.Children[1].Children = append(tree.Children[1].Children, domain.ContentNode{
		ID: "read", Type: domain.NodeCommand, Name: "READ", Text: "Moved command",
	})
	return tree
}

func removeNode(parent *domain.ContentNode, nodeID string) bool {
	for index := range parent.Children {
		if parent.Children[index].ID == nodeID {
			parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
			return true
		}
		if removeNode(&parent.Children[index], nodeID) {
			return true
		}
	}
	return false
}

func navState(path []string, mode, viewEntryID, commandNodeID string) domain.NavState {
	state := domain.NavState{Path: path, Mode: mode}
	if viewEntryID != "" {
		state.ViewEntryID = stringPointer(viewEntryID)
	}
	if commandNodeID != "" {
		state.CommandNodeID = stringPointer(commandNodeID)
	}
	return state
}

func stringPointer(value string) *string {
	return &value
}
