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
		"verify protobuf and generated clients",
		"install locked player dependencies",
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

func TestUnknownActionIsRejected(t *testing.T) {
	_, err := Plan("task", nil)
	require.Error(t, err)
}
