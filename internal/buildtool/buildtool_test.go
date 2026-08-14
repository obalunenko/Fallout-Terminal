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
