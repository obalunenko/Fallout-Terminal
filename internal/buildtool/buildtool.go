// Package buildtool owns the repository's dependency-free development, build,
// and macOS packaging graph. It deliberately uses only the Go standard library;
// versioned development tools continue to run through their isolated Go modules.
package buildtool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	applicationName = "Fallout Terminal"
	minimumMacOS    = "13.0"
)

type operation uint8

const (
	runCommand operation = iota
	removeTree
	makeDirectory
	copyFile
	changeMode
)

// Step is one deterministic node in the repository build graph.
type Step struct {
	Name        string
	Operation   operation
	Program     string
	Arguments   []string
	Environment map[string]string
	Path        string
	Source      string
	Destination string
	Mode        os.FileMode
}

// Plan returns the ordered, nonrecursive graph for an action.
func Plan(action string, applicationArguments []string) ([]Step, error) {
	switch action {
	case "prepare":
		return preparePlan(), nil
	case "build":
		return append(preparePlan(), buildSteps()...), nil
	case "dev", "run":
		steps := append(preparePlan(), developmentSteps()...)
		return append(steps, commandStep("run development application", developmentExecutable(), applicationArguments...)), nil
	case "package":
		return append(preparePlan(), packageSteps()...), nil
	default:
		return nil, fmt.Errorf("unknown action %q (want dev, build, package, run, or prepare)", action)
	}
}

func preparePlan() []Step {
	return []Step{
		commandStep("verify protobuf and generated clients", filepath.Join("scripts", "proto-check.sh")),
		commandStep("install locked player dependencies", "npm", "ci", "--prefix", "client"),
		commandStep("build player frontend", "npm", "run", "build", "--prefix", "client"),
		commandStep("generate Wails bindings", "go", "tool", "-modfile=tools/wails/go.mod", "wails3", "generate", "bindings", "-clean", "-d", "frontend/bindings", "./..."),
		commandStep("install locked master dependencies", "npm", "ci", "--prefix", "frontend"),
		commandStep("build master frontend", "npm", "run", "build", "--prefix", "frontend"),
	}
}

func buildSteps() []Step {
	return []Step{
		{Name: "create binary output directory", Operation: makeDirectory, Path: filepath.Join("build", "bin"), Mode: 0o755},
		compileStep(filepath.Join("build", "bin", applicationName)),
	}
}

func developmentSteps() []Step {
	app := developmentBundle()
	contents := filepath.Join(app, "Contents")
	macOS := filepath.Join(contents, "MacOS")
	resources := filepath.Join(contents, "Resources")
	executable := filepath.Join(macOS, applicationName)

	return []Step{
		{Name: "remove previous development application bundle", Operation: removeTree, Path: app},
		{Name: "create development application executable directory", Operation: makeDirectory, Path: macOS, Mode: 0o755},
		{Name: "create development bundled session directory", Operation: makeDirectory, Path: filepath.Join(resources, "sessions"), Mode: 0o755},
		{Name: "install development application metadata", Operation: copyFile, Source: filepath.Join("build", "darwin", "Info.dev.plist"), Destination: filepath.Join(contents, "Info.plist"), Mode: 0o644},
		commandStep("install development application icon", "go", "tool", "-modfile=tools/wails/go.mod", "wails3", "generate", "icons", "-input", filepath.Join("build", "appicon.png"), "-macfilename", filepath.Join(resources, "icon.icns"), "-windowsfilename", filepath.Join(resources, "icon.ico")),
		{Name: "install development bundled demo", Operation: copyFile, Source: filepath.Join("sessions", "demo.json"), Destination: filepath.Join(resources, "sessions", "demo.json"), Mode: 0o444},
		compileStep(executable),
		{Name: "make development application executable", Operation: changeMode, Path: executable, Mode: 0o755},
	}
}

func developmentBundle() string {
	return filepath.Join("build", "dev", applicationName+".app")
}

func developmentExecutable() string {
	return filepath.Join(developmentBundle(), "Contents", "MacOS", applicationName)
}

func packageSteps() []Step {
	app := filepath.Join("build", "bin", applicationName+".app")
	contents := filepath.Join(app, "Contents")
	macOS := filepath.Join(contents, "MacOS")
	resources := filepath.Join(contents, "Resources")
	executable := filepath.Join(macOS, applicationName)

	return []Step{
		{Name: "remove previous application bundle", Operation: removeTree, Path: app},
		{Name: "create application executable directory", Operation: makeDirectory, Path: macOS, Mode: 0o755},
		{Name: "create bundled session directory", Operation: makeDirectory, Path: filepath.Join(resources, "sessions"), Mode: 0o755},
		{Name: "install application metadata", Operation: copyFile, Source: filepath.Join("build", "darwin", "Info.plist"), Destination: filepath.Join(contents, "Info.plist"), Mode: 0o644},
		commandStep("install application icon", "go", "tool", "-modfile=tools/wails/go.mod", "wails3", "generate", "icons", "-input", filepath.Join("build", "appicon.png"), "-macfilename", filepath.Join(resources, "icon.icns"), "-windowsfilename", filepath.Join(resources, "icon.ico")),
		{Name: "install bundled demo", Operation: copyFile, Source: filepath.Join("sessions", "demo.json"), Destination: filepath.Join(resources, "sessions", "demo.json"), Mode: 0o444},
		compileStep(executable),
		{Name: "make application executable", Operation: changeMode, Path: executable, Mode: 0o755},
		commandStep("sign completed application bundle", "/usr/bin/codesign", "--force", "--deep", "--options", "runtime", "--entitlements", filepath.Join("build", "darwin", "entitlements.plist"), "--sign", "-", app),
	}
}

func compileStep(output string) Step {
	step := commandStep("compile macOS arm64 application", "go", "build", "-tags", "production", "-trimpath", "-buildvcs=false", "-ldflags=-w -s", "-o", output, ".")
	step.Environment = map[string]string{
		"CGO_ENABLED":              "1",
		"CGO_CFLAGS":               "-mmacosx-version-min=" + minimumMacOS,
		"CGO_LDFLAGS":              "-mmacosx-version-min=" + minimumMacOS,
		"GOARCH":                   "arm64",
		"GOOS":                     "darwin",
		"MACOSX_DEPLOYMENT_TARGET": minimumMacOS,
	}
	return step
}

func commandStep(name, program string, arguments ...string) Step {
	return Step{Name: name, Operation: runCommand, Program: program, Arguments: append([]string(nil), arguments...)}
}

// Run executes one action from the repository root.
func Run(ctx context.Context, root, action string, applicationArguments []string) error {
	if err := validateRoot(root); err != nil {
		return err
	}
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return fmt.Errorf("the supported build host is macOS arm64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	steps, err := Plan(action, applicationArguments)
	if err != nil {
		return err
	}
	for _, step := range steps {
		fmt.Printf("==> %s\n", step.Name)
		if err := execute(ctx, root, step); err != nil {
			return fmt.Errorf("%s: %w", step.Name, err)
		}
	}
	return nil
}

func validateRoot(root string) error {
	module, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return fmt.Errorf("run from the repository root: %w", err)
	}
	if !strings.Contains(string(module), "module github.com/obalunenko/Fallout-Terminal") {
		return errors.New("run from the Fallout-Terminal repository root")
	}
	return nil
}

func execute(ctx context.Context, root string, step Step) error {
	switch step.Operation {
	case runCommand:
		cmd := exec.CommandContext(ctx, step.Program, step.Arguments...)
		cmd.Dir = root
		cmd.Env = mergeEnvironment(os.Environ(), step.Environment)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case removeTree:
		target, err := resolvePath(root, step.Path)
		if err != nil {
			return err
		}
		allowed := map[string]struct{}{
			filepath.Join(filepath.Clean(root), "build", "bin", applicationName+".app"): {},
			filepath.Join(filepath.Clean(root), developmentBundle()):                    {},
		}
		if _, ok := allowed[target]; !ok {
			return fmt.Errorf("refusing to remove unexpected path %q", target)
		}
		return os.RemoveAll(target)
	case makeDirectory:
		target, err := resolvePath(root, step.Path)
		if err != nil {
			return err
		}
		return os.MkdirAll(target, step.Mode)
	case copyFile:
		return copyRegularFile(root, step.Source, step.Destination, step.Mode)
	case changeMode:
		target, err := resolvePath(root, step.Path)
		if err != nil {
			return err
		}
		return os.Chmod(target, step.Mode)
	default:
		return fmt.Errorf("unsupported build operation %d", step.Operation)
	}
}

func resolvePath(root, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("build path must be repository-relative: %q", path)
	}
	cleanRoot := filepath.Clean(root)
	resolved := filepath.Join(cleanRoot, filepath.Clean(path))
	if resolved != cleanRoot && !strings.HasPrefix(resolved, cleanRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("build path escapes repository root: %q", path)
	}
	return resolved, nil
}

func copyRegularFile(root, source, destination string, mode os.FileMode) error {
	sourcePath, err := resolvePath(root, source)
	if err != nil {
		return err
	}
	destinationPath, err := resolvePath(root, destination)
	if err != nil {
		return err
	}
	input, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source is not a regular file: %s", source)
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chmod(destinationPath, mode)
}

func mergeEnvironment(base []string, overrides map[string]string) []string {
	if len(overrides) == 0 {
		return base
	}
	values := make(map[string]string, len(base)+len(overrides))
	for _, item := range base {
		key, value, found := strings.Cut(item, "=")
		if found {
			values[key] = value
		}
	}
	for key, value := range overrides {
		values[key] = value
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}
