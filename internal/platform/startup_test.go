package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWailsOwnsTheWholeDevelopmentStartup(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "wails.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatalf("decode wails.json: %v", err)
	}

	required := map[string]string{
		"frontend:dir":           "frontend",
		"frontend:install":       "npm ci",
		"frontend:dev:install":   "npm ci",
		"frontend:build":         "npm run build",
		"frontend:dev:watcher":   "npm run dev",
		"frontend:dev:serverUrl": "auto",
	}
	for key, want := range required {
		value, exists := config[key]
		if !exists {
			t.Errorf("wails.json is missing %q", key)
			continue
		}
		var got string
		if err := json.Unmarshal(value, &got); err != nil {
			t.Errorf("wails.json %q is not a string: %v", key, err)
			continue
		}
		if got != want {
			t.Errorf("wails.json %q = %q, want %q", key, got, want)
		}
	}
}

func TestQuickstartHasOneRepositoryRootStartupCommand(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "specs", "001-wails-v2-migration", "quickstart.md"))
	if err != nil {
		t.Fatal(err)
	}
	section := markdownSection(string(raw), "## 3. Develop the Wails candidate")
	if section == "" {
		t.Fatal("quickstart is missing the Wails development section")
	}
	if count := strings.Count(section, "\nwails dev\n"); count != 1 {
		t.Fatalf("development section contains %d exact root startup commands, want one", count)
	}
	for _, forbidden := range []string{"cd frontend", "npm run dev", "go run", "node server", "npm start"} {
		if strings.Contains(section, forbidden) {
			t.Errorf("development startup requires a separate command %q", forbidden)
		}
	}
	if !strings.Contains(section, "repository root") {
		t.Error("development startup does not explicitly require the repository root")
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve startup test location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func markdownSection(document, heading string) string {
	start := strings.Index(document, heading)
	if start < 0 {
		return ""
	}
	rest := document[start+len(heading):]
	if end := strings.Index(rest, "\n## "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}
