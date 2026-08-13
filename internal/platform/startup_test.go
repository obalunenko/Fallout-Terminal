package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWailsOwnsTheWholeDevelopmentStartup(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "wails.json"))
	if err != nil {
		require.NoError(t, err)
	}
	var config map[string]json.RawMessage
	{
		err := json.Unmarshal(raw, &config)
		require.Falsef(t, err != nil,
			"decode wails.json: %v", err)
	}

	required := map[string]string{
		"frontend:dir":           "frontend",
		"frontend:install":       "npm run install:all",
		"frontend:dev:install":   "npm run install:all",
		"frontend:build":         "npm run build:all",
		"frontend:dev:watcher":   "npm run dev:all",
		"frontend:dev:serverUrl": "auto",
	}
	for key, want := range required {
		value, exists := config[key]
		if !exists {
			assert.Failf(t, "assertion failed", "wails.json is missing %q", key)
			continue
		}
		var got string
		if err := json.Unmarshal(value, &got); err != nil {
			assert.Failf(t, "assertion failed", "wails.json %q is not a string: %v", key, err)
			continue
		}
		assert.Falsef(t, got != want,
			"wails.json %q = %q, want %q", key, got, want)

	}

	packageRaw, err := os.ReadFile(filepath.Join(root, "frontend", "package.json"))
	if err != nil {
		require.NoError(t, err)
	}
	var packageConfig struct {
		Scripts map[string]string `json:"scripts"`
	}
	{
		err := json.Unmarshal(packageRaw, &packageConfig)
		require.Falsef(t, err != nil,
			"decode frontend/package.json: %v", err)
	}

	requiredScripts := map[string]string{
		"install:all":    "npm ci && npm ci --prefix ../client",
		"proto:generate": "../scripts/proto-generate.sh --sync-revision",
		"dev:all":        "npm run proto:generate && vite",
		"build:all":      "npm run proto:generate && npm run build --prefix ../client && vite build",
	}
	for name, want := range requiredScripts {
		{
			got := packageConfig.Scripts[name]
			assert.Falsef(t, got != want,
				"frontend/package.json script %q = %q, want %q", name, got, want)
		}

	}

	clientPackageRaw, err := os.ReadFile(filepath.Join(root, "client", "package.json"))
	if err != nil {
		require.NoError(t, err)
	}
	var clientPackageConfig struct {
		Scripts map[string]string `json:"scripts"`
	}
	{
		err := json.Unmarshal(clientPackageRaw, &clientPackageConfig)
		require.Falsef(t, err != nil,
			"decode client/package.json: %v", err)
	}
	{

		got, want := clientPackageConfig.Scripts["generate"], "../scripts/proto-generate.sh --sync-revision"
		assert.Falsef(t, got != want,
			"client/package.json script %q = %q, want %q", "generate", got, want)
	}

}

func TestQuickstartHasOneRepositoryRootStartupCommand(t *testing.T) {
	root := repositoryRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "specs", "001-wails-v2-migration", "quickstart.md"))
	if err != nil {
		require.NoError(t, err)
	}
	section := markdownSection(string(raw), "## 3. Develop the Wails candidate")
	require.False(t, section == "",
		"quickstart is missing the Wails development section")
	{

		count := strings.Count(section, "\nwails dev\n")
		require.Falsef(t, count != 1,
			"development section contains %d exact root startup commands, want one", count)
	}

	for _, forbidden := range []string{"cd frontend", "npm run dev", "go run", "node server", "npm start"} {
		assert.Falsef(t, strings.Contains(section, forbidden),
			"development startup requires a separate command %q", forbidden)

	}
	assert.False(t, !strings.Contains(section, "repository root"),
		"development startup does not explicitly require the repository root")

}

func TestAcceptanceEvidenceUsesOneCanonicalPostElectronCandidate(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	quickstart := readAcceptanceDocument(t, filepath.Join(root, "specs", "001-wails-v2-migration", "quickstart.md"))
	rollback := readAcceptanceDocument(t, filepath.Join(root, "docs", "wails-migration-rollback.md"))

	quickstartCommit, quickstartDigest := canonicalCandidate(t, "quickstart", quickstart)
	rollbackCommit, rollbackDigest := canonicalCandidate(t, "rollback guide", rollback)
	assert.Falsef(t, quickstartCommit != rollbackCommit,
		"canonical candidate commit conflicts: quickstart=%s rollback=%s", quickstartCommit, rollbackCommit)
	assert.Falsef(t, quickstartDigest != rollbackDigest,
		"canonical executable SHA-256 conflicts: quickstart=%s rollback=%s", quickstartDigest, rollbackDigest)

}

func readAcceptanceDocument(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		require.NoError(t, err)
	}
	return string(raw)
}

func canonicalCandidate(t *testing.T, name, document string) (string, string) {
	t.Helper()
	commit := canonicalValue(t, name, document, "Canonical candidate commit: `", 40)
	digest := canonicalValue(t, name, document, "Canonical executable SHA-256: `", 64)
	return commit, digest
}

func canonicalValue(t *testing.T, name, document, prefix string, length int) string {
	t.Helper()
	{
		count := strings.Count(document, prefix)
		require.Falsef(t, count != 1,
			"%s contains %d %q records, want exactly one", name, count, strings.TrimSuffix(prefix, "`"))
	}

	start := strings.Index(document, prefix) + len(prefix)
	rest := document[start:]
	end := strings.IndexByte(rest, '`')
	require.Falsef(t, end < 0,
		"%s canonical record %q has no closing backtick", name, strings.TrimSuffix(prefix, "`"))

	value := rest[:end]
	require.Falsef(t, len(value) != length,
		"%s canonical record %q has %d characters, want %d", name, strings.TrimSuffix(prefix, "`"), len(value), length)

	for _, character := range value {
		require.Falsef(t, !strings.ContainsRune("0123456789abcdef", character),
			"%s canonical record %q is not lowercase hexadecimal: %q", name, strings.TrimSuffix(prefix, "`"), value)

	}
	return value
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.False(t, !ok,
		"cannot resolve startup test location")

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
