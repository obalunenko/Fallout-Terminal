package buildtool

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPlanHasOneOrderedOwnerAndNoTaskfileTool(t *testing.T) {
	steps, err := Plan("build", nil)
	require.NoError(t, err)

	wantNames := []string{
		"install locked player dependencies",
		"verify protobuf and generated clients",
		"build player frontend",
		"generate Wails bindings",
		"install locked master dependencies",
		"build master frontend",
		"create binary output directory",
		"compile macOS arm64 application",
	}
	require.Len(t, steps, len(wantNames))
	for index, want := range wantNames {
		assert.Equal(t, want, steps[index].Name)
		command := strings.Join(append([]string{steps[index].Program}, steps[index].Arguments...), " ")
		assert.Falsef(t, strings.Contains(strings.ToLower(command), "taskfile") || strings.HasPrefix(command, "task ") || strings.Contains(command, " wails3 task"),
			"step %q invokes a Taskfile tool: %q", want, command)
	}
	got := steps[len(steps)-1].Arguments
	assert.Equal(t, filepath.Join("build", "bin", applicationName), got[len(got)-2])
}

func TestPreparePlanInstallsLockedPlayerToolBeforeProtobufVerificationForEveryAction(t *testing.T) {
	for _, action := range []string{"prepare", "build", "dev", "run", "package"} {
		t.Run(action, func(t *testing.T) {
			steps, err := Plan(action, nil)
			require.NoError(t, err)

			positions := make(map[string]int, len(steps))
			for index, step := range steps {
				positions[step.Name] = index
			}
			installIndex, hasInstall := positions["install locked player dependencies"]
			verifyIndex, hasVerify := positions["verify protobuf and generated clients"]
			require.True(t, hasInstall)
			require.True(t, hasVerify)
			assert.Less(t, installIndex, verifyIndex)
			assert.Equal(t, "npm", steps[installIndex].Program)
			assert.Equal(t, []string{"ci", "--prefix", "client"}, steps[installIndex].Arguments)
			assert.Equal(t, filepath.Join("scripts", "proto-check.sh"), steps[verifyIndex].Program)
			assert.Empty(t, steps[verifyIndex].Arguments)
		})
	}
}

func TestDevelopmentPlansAssembleAndLaunchOwnedApplicationIdentity(t *testing.T) {
	for _, action := range []string{"dev", "run"} {
		t.Run(action, func(t *testing.T) {
			steps, err := Plan(action, []string{"--fixture"})
			require.NoError(t, err)

			positions := make(map[string]int, len(steps))
			for index, step := range steps {
				positions[step.Name] = index
			}

			for _, name := range []string{
				"remove previous development application bundle",
				"install development application metadata",
				"install development application icon",
				"compile macOS arm64 application",
				"run development application",
			} {
				assert.Contains(t, positions, name)
			}
			assert.Less(t, positions["install development application metadata"], positions["compile macOS arm64 application"])
			assert.Less(t, positions["install development application icon"], positions["compile macOS arm64 application"])
			assert.Less(t, positions["compile macOS arm64 application"], positions["run development application"])

			metadata := steps[positions["install development application metadata"]]
			assert.Equal(t, filepath.Join("build", "darwin", "Info.dev.plist"), metadata.Source)
			assert.Equal(t, filepath.Join("build", "dev", applicationName+".app", "Contents", "Info.plist"), metadata.Destination)

			icon := steps[positions["install development application icon"]]
			assert.Contains(t, icon.Arguments, filepath.Join("build", "appicon.png"))
			assert.Contains(t, icon.Arguments, filepath.Join("build", "dev", applicationName+".app", "Contents", "Resources", "icon.icns"))

			launch := steps[positions["run development application"]]
			assert.Equal(t, filepath.Join("build", "dev", applicationName+".app", "Contents", "MacOS", applicationName), launch.Program)
			assert.Equal(t, []string{"--fixture"}, launch.Arguments)

			productionBundle := filepath.Join("build", "bin", applicationName+".app")
			for _, step := range steps {
				for _, value := range append([]string{step.Program, step.Path, step.Source, step.Destination}, step.Arguments...) {
					assert.NotEqual(t, productionBundle, value, "development action must not mutate or launch the production package")
				}
			}
		})
	}
}

func TestPackagePlanCompletesResourcesBeforeFinalSignature(t *testing.T) {
	steps, err := Plan("package", nil)
	require.NoError(t, err)

	positions := make(map[string]int, len(steps))
	for index, step := range steps {
		positions[step.Name] = index
	}
	for _, resource := range []string{"install application metadata", "install application icon", "install bundled demo"} {
		assert.Lessf(t, positions[resource], positions["sign completed application bundle"], "%s must precede final signature", resource)
	}
	assert.Less(t, positions["compile macOS arm64 application"], positions["sign completed application bundle"], "application compilation must precede final signature")
}

func TestPackagePlanOwnsEmbeddedDependencyNoticesAndNoProviderExecutable(t *testing.T) {
	t.Parallel()

	steps, err := Plan("package", nil)
	require.NoError(t, err)

	positions := make(map[string]int, len(steps))
	for index, step := range steps {
		positions[step.Name] = index
		values := append([]string{step.Program, step.Path, step.Source, step.Destination}, step.Arguments...)
		joined := strings.ToLower(strings.Join(values, " "))
		assert.NotContains(t, joined, "ngrok", "package plan must not copy or execute a provider binary")
		assert.NotContains(t, joined, "curl", "package plan must not download a provider runtime")
	}

	dependencyIndex, exists := positions["verify embedded dependency and license inventory"]
	require.True(t, exists, "package must run the exact SDK/license inventory gate")
	assert.Equal(t, filepath.Join("scripts", "dependency-license-check.sh"), steps[dependencyIndex].Program)

	noticeIndex, exists := positions["install third-party notices"]
	require.True(t, exists, "package must include reviewed third-party notices")
	notice := steps[noticeIndex]
	assert.Equal(t, "THIRD_PARTY_NOTICES.md", notice.Source)
	assert.Equal(t, filepath.Join("build", "bin", applicationName+".app", "Contents", "Resources", "THIRD_PARTY_NOTICES.md"), notice.Destination)
	assert.Equal(t, 0o444, int(notice.Mode.Perm()))
	assert.Less(t, noticeIndex, positions["compile macOS arm64 application"])
	assert.Less(t, noticeIndex, positions["sign completed application bundle"])

	compile := steps[positions["compile macOS arm64 application"]]
	assert.Equal(t, "darwin", compile.Environment["GOOS"])
	assert.Equal(t, "arm64", compile.Environment["GOARCH"])
	assert.Equal(t, "1", compile.Environment["CGO_ENABLED"])
	assert.Equal(t, minimumMacOS, compile.Environment["MACOSX_DEPLOYMENT_TARGET"])
	assert.Equal(t, "-mmacosx-version-min="+minimumMacOS, compile.Environment["CGO_CFLAGS"])
	assert.Equal(t, "-mmacosx-version-min="+minimumMacOS, compile.Environment["CGO_LDFLAGS"])
}

func TestPackagePlanPreservesCanonicalFrontendAndOfflineResourceOwnership(t *testing.T) {
	t.Parallel()

	steps, err := Plan("package", nil)
	require.NoError(t, err)

	positions := make(map[string]int, len(steps))
	for index, step := range steps {
		positions[step.Name] = index
	}
	ordered := []string{
		"install locked player dependencies",
		"verify protobuf and generated clients",
		"build player frontend",
		"generate Wails bindings",
		"install locked master dependencies",
		"build master frontend",
		"install application metadata",
		"install application icon",
		"install bundled demo player config",
		"install bundled demo",
		"compile macOS arm64 application",
		"sign completed application bundle",
	}
	for index := 1; index < len(ordered); index++ {
		assert.Lessf(t, positions[ordered[index-1]], positions[ordered[index]], "%s must precede %s", ordered[index-1], ordered[index])
	}

	demo := steps[positions["install bundled demo"]]
	assert.Equal(t, filepath.Join("sessions", "demo.json"), demo.Source)
	assert.Equal(t, filepath.Join("build", "bin", applicationName+".app", "Contents", "Resources", "sessions", "demo.json"), demo.Destination)
	assert.Equal(t, 0o444, int(demo.Mode.Perm()))

	players := steps[positions["install bundled demo player config"]]
	assert.Equal(t, filepath.Join("sessions", "demo-players.json"), players.Source)
	assert.Equal(t, filepath.Join("build", "bin", applicationName+".app", "Contents", "Resources", "sessions", "demo-players.json"), players.Destination)
	assert.Equal(t, 0o444, int(players.Mode.Perm()))
}

func TestUnknownActionIsRejected(t *testing.T) {
	_, err := Plan("task", nil)
	require.Error(t, err)
}
