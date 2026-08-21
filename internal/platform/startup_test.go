package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWailsV3GoToolsAreIsolatedFromApplicationModule(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	tests := []struct {
		name          string
		directory     string
		tool          string
		parentRequire string
	}{
		{
			name:          "Wails CLI",
			directory:     "wails",
			tool:          "github.com/wailsapp/wails/v3/cmd/wails3",
			parentRequire: "github.com/wailsapp/wails/v3 v3.0.0-beta.10",
		},
		{
			name:          "Buf CLI",
			directory:     "buf",
			tool:          "github.com/bufbuild/buf/cmd/buf",
			parentRequire: "github.com/bufbuild/buf v1.72.0",
		},
		{
			name:          "golangci-lint",
			directory:     "golangci-lint",
			tool:          "github.com/golangci/golangci-lint/v2/cmd/golangci-lint",
			parentRequire: "github.com/golangci/golangci-lint/v2 v2.13.1",
		},
		{
			name:          "protoc-gen-go",
			directory:     "protoc-gen-go",
			tool:          "google.golang.org/protobuf/cmd/protoc-gen-go",
			parentRequire: "google.golang.org/protobuf v1.36.11",
		},
		{
			name:          "protoc-gen-connect-go",
			directory:     "protoc-gen-connect-go",
			tool:          "connectrpc.com/connect/cmd/protoc-gen-connect-go",
			parentRequire: "connectrpc.com/connect v1.20.0",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			module := readAcceptanceDocument(t, filepath.Join(root, "tools", test.directory, "go.mod"))
			assert.Equal(t, 1, strings.Count(module, "\ntool "))
			assert.Contains(t, module, "\ntool "+test.tool+"\n")
			parentPattern := "(?m)^[[:space:]]*(require[[:space:]]+)?" +
				regexp.QuoteMeta(test.parentRequire) + "([[:space:]]+// indirect)?$"
			assert.Regexp(t, parentPattern, module)
			assert.Contains(t, module, "\ngo 1.27.0\n")

			sum, err := os.ReadFile(filepath.Join(root, "tools", test.directory, "go.sum"))
			require.NoError(t, err)
			require.NotEmpty(t, sum)
		})
	}

	applicationModule := readAcceptanceDocument(t, filepath.Join(root, "go.mod"))
	assert.NotContains(t, applicationModule, "\ntool ")
	assert.NotContains(t, applicationModule, "github.com/bufbuild/buf")
	assert.NotContains(t, applicationModule, "github.com/golangci/golangci-lint")
	assert.NotContains(t, applicationModule, "/cmd/protoc-gen-go")
	assert.NotContains(t, applicationModule, "/cmd/protoc-gen-connect-go")
	assert.NotContains(t, applicationModule, "/v3/cmd/wails3")
}

func TestWailsV3PinsAndGoBuildToolAreOwnedAndExact(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	applicationModule := readAcceptanceDocument(t, filepath.Join(root, "go.mod"))
	assert.Equal(t, 1, strings.Count(applicationModule, "github.com/wailsapp/wails/v3 v3.0.0-beta.10"))

	packageRaw, err := os.ReadFile(filepath.Join(root, "frontend", "overseer", "package.json"))
	require.NoError(t, err)
	var packageConfig struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	require.NoError(t, json.Unmarshal(packageRaw, &packageConfig))
	assert.Equal(t, "3.0.0-beta.10", packageConfig.Dependencies["@wailsio/runtime"])

	lock := readAcceptanceDocument(t, filepath.Join(root, "frontend", "package-lock.json"))
	assert.Contains(t, lock, `"@wailsio/runtime": "3.0.0-beta.10"`)
	assert.Contains(t, lock, `runtime-3.0.0-beta.10.tgz`)

	files := []struct {
		path   string
		tokens []string
	}{
		{"cmd/build/main.go", []string{"buildtool.Run", "dev|build|package|run|prepare"}},
		{"internal/buildtool/buildtool.go", []string{"scripts", "proto-check.sh", "tools/wails/go.mod", "frontend/overseer/bindings", "GOARCH", "arm64", "13.0", `applicationName+".app"`}},
		{"build/darwin/Info.plist", []string{"com.vaulttec.fallout-terminal", "13.0", "icon.icns"}},
		{"build/darwin/Info.dev.plist", []string{"com.vaulttec.fallout-terminal", "13.0", "icon.icns"}},
		{"build/darwin/entitlements.plist", []string{"com.apple.security.network.server"}},
	}

	icon, err := os.Stat(filepath.Join(root, "build", "appicon.png"))
	require.NoError(t, err)
	assert.True(t, icon.Mode().IsRegular())
	assert.Positive(t, icon.Size(), "the development application icon source must not be empty")

	buildSource := readAcceptanceDocument(t, filepath.Join(root, "internal", "buildtool", "buildtool.go"))
	for _, required := range []string{
		`filepath.Join("build", "dev", applicationName+".app")`,
		`filepath.Join("build", "darwin", "Info.dev.plist")`,
		`Name: "install development application metadata"`,
		`commandStep("install development application icon"`,
		`commandStep("run development application"`,
	} {
		assert.Contains(t, buildSource, required)
	}
	for _, file := range files {
		t.Run(file.path, func(t *testing.T) {
			t.Parallel()
			contents := readAcceptanceDocument(t, filepath.Join(root, filepath.FromSlash(file.path)))
			for _, token := range file.tokens {
				assert.Contains(t, contents, token)
			}
		})
	}

	for _, path := range []string{
		"Taskfile.yml",
		"Taskfile.yaml",
		"build/Taskfile.yml",
		"build/common/Taskfile.yml",
		"build/darwin/Taskfile.yml",
	} {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
		assert.ErrorIs(t, err, os.ErrNotExist, "%s must not exist", path)
	}
}

func TestWailsV3QuickstartHasOneRootDevelopmentCommand(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	quickstart := readAcceptanceDocument(t, filepath.Join(root, "specs", "006-wails-v3-migration", "quickstart.md"))
	section := markdownSection(quickstart, "## Local Development")
	require.NotEmpty(t, section)
	command := "go run ./cmd/build dev"
	assert.Equal(t, 1, strings.Count(section, command))
	for _, forbidden := range []string{"cd frontend", "npm run dev", "wails3 dev", "wails3 task", "node server", "npm start"} {
		assert.NotContains(t, section, forbidden)
	}
	assert.Contains(t, section, "repository root")
}

func TestGoPackageOutputDeploymentTargetAndFinalSignOrderAreExplicit(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	buildSource := readAcceptanceDocument(t, filepath.Join(root, "internal", "buildtool", "buildtool.go"))
	for _, required := range []string{
		`minimumMacOS    = "13.0"`,
		`filepath.Join("build", "bin", applicationName+".app")`,
		`"GOARCH":                   "arm64"`,
		`"GOOS":                     "darwin"`,
		`"MACOSX_DEPLOYMENT_TARGET": minimumMacOS`,
		`commandStep("sign completed application bundle"`,
	} {
		assert.Contains(t, buildSource, required)
	}

	metadata := readAcceptanceDocument(t, filepath.Join(root, "build", "darwin", "Info.plist"))
	assert.Contains(t, metadata, "<key>LSMinimumSystemVersion</key>")
	assert.Contains(t, metadata, "<string>13.0</string>")
	assert.Contains(t, metadata, "<string>icon.icns</string>")

	packageStart := strings.Index(buildSource, "func packageSteps() []Step {")
	require.NotEqual(t, -1, packageStart)
	packageEnd := strings.Index(buildSource[packageStart:], "\nfunc compileStep(")
	require.NotEqual(t, -1, packageEnd)
	packageSource := buildSource[packageStart : packageStart+packageEnd]
	installDemo := strings.Index(packageSource, `Name: "install bundled demo"`)
	compile := strings.Index(packageSource, `compileStep(executable)`)
	sign := strings.Index(packageSource, `commandStep("sign completed application bundle"`)
	require.NotEqual(t, -1, installDemo)
	require.NotEqual(t, -1, compile)
	require.NotEqual(t, -1, sign)
	assert.Less(t, installDemo, compile)
	assert.Less(t, compile, sign)
}

func TestReproducibleBuildHashesPackagedExecutableAndUsesQuietToolEnvironments(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	reproducible := readAcceptanceDocument(t, filepath.Join(root, "scripts", "reproducible-build-check.sh"))
	assert.Contains(t, reproducible, `application_executable="${application_bundle}/Contents/MacOS/Fallout Terminal"`)
	assert.NotContains(t, reproducible, `shasum -a 256 "build/bin/Fallout Terminal"`)

	protoCheck := readAcceptanceDocument(t, filepath.Join(root, "scripts", "proto-check.sh"))
	assert.Contains(t, protoCheck, `MACOSX_DEPLOYMENT_TARGET="${MACOSX_DEPLOYMENT_TARGET:-13.0}"`)
	assert.Contains(t, protoCheck, `CGO_CFLAGS="${CGO_CFLAGS:--mmacosx-version-min=${MACOSX_DEPLOYMENT_TARGET}}"`)
	assert.Contains(t, protoCheck, `CGO_LDFLAGS="${CGO_LDFLAGS:--mmacosx-version-min=${MACOSX_DEPLOYMENT_TARGET}}"`)

	protoGenerate := readAcceptanceDocument(t, filepath.Join(root, "scripts", "proto-generate.sh"))
	assert.Contains(t, protoGenerate, `node_major >= 22`)
	assert.Contains(t, protoGenerate, `--no-experimental-webstorage`)
}

func TestCIRunsOnlyMinimalLintTestProtobufAndApplicationBuild(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	workflow := readAcceptanceDocument(t, filepath.Join(root, ".github", "workflows", "wails-macos.yml"))
	assert.Equal(t, 1, strings.Count(workflow, "\n    runs-on:"), "CI must use one job")
	for _, required := range []string{
		"- name: Lint",
		"gofmt -l .",
		"go vet ./...",
		"- name: Test",
		"go test ./...",
		"- name: Build protobuf",
		"buf format proto --diff --exit-code",
		"buf lint proto",
		"buf build proto",
		"scripts/proto-generate.sh",
		"git diff --exit-code -- internal/gen frontend/client/gen",
		"- name: Build application",
		"go run ./cmd/build package",
	} {
		assert.Contains(t, workflow, required)
	}
	for _, forbidden := range []string{
		"go test -race",
		"tests/browser",
		"reproducible-build-check.sh",
		"secret-leak-check.sh",
		"legacy-public-access-check.sh",
		"proto-breaking.sh",
		"actions/upload-artifact",
	} {
		assert.NotContains(t, workflow, forbidden)
	}
}

func TestWailsV3RollbackRecordHasIdentitySafetyTriggersAndHonestEvidenceFields(t *testing.T) {
	t.Parallel()

	root := repositoryRoot(t)
	rollback := readAcceptanceDocument(t, filepath.Join(root, "docs", "wails-v3-migration-rollback.md"))
	for _, required := range []string{
		"f1084b3df8b5630862bdf7a0f347b599156653ef",
		"Source verification | `PASS`",
		"Artifact status | `BUILT FOR DRILL — ACCEPTED FOR THIS DRILL`",
		"Executable SHA-256 | `c1faf7fe4f2ed0abc5c4814b8e71805f5b57a65b817fd3a45bbcc90bdaf29530`",
		"invent or prefill an artifact digest",
		"bcb207704657a92f9902f4ac04ef11765b18f031",
		"provenance only—not the build candidate",
		"## Rollback Triggers",
		"| Trigger | Required action | Decision owner |",
		"## Data-Safe Rollback Procedure",
		"safety copies",
		"Record SHA-256 values for the originals and safety copies",
		"separate maintenance worktree or clone",
		"Open only the safety-copy version-1 data without migration or conversion",
		"## Rollback Drill Evidence",
		"Overall drill result | `PASS`",
	} {
		assert.Contains(t, rollback, required)
	}
	assert.NotContains(t, rollback, "Artifact status | `PASS`")
	assert.Contains(t, rollback, "immutable source commit remains canonical rollback authority")
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
	_, after, ok := strings.Cut(document, heading)
	if !ok {
		return ""
	}
	rest := after
	if end := strings.Index(rest, "\n## "); end >= 0 {
		rest = rest[:end]
	}
	return rest
}
