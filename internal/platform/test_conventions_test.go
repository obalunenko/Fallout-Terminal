package platform

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConnectRPCMigrationTestConventions keeps the feature's Go tests aligned
// with the constitution instead of relying on review-only style guidance.
func TestConnectRPCMigrationTestConventions(t *testing.T) {
	t.Parallel()
	root := assetRepositoryRoot(t)
	files := []string{
		"app_contract_test.go",
		"app_test.go",
		"internal/control/service_test.go",
		"internal/live/service_test.go",
		"internal/platform/assets_test.go",
		"internal/platform/startup_test.go",
		"internal/platform/test_conventions_test.go",
		"internal/player/adapter_test.go",
		"internal/player/handler_test.go",
		"internal/player/http_test.go",
		"internal/player/public_stream_test.go",
		"internal/player/stream_test.go",
		"internal/playerconfig/contract_test.go",
		"internal/playerconfig/service_test.go",
		"internal/session/contract_test.go",
		"internal/tunnel/service_test.go",
		"internal/tunnel/manager_test.go",
		"internal/tunnel/model_test.go",
		"internal/tunnel/ngrok_test.go",
		"internal/tunnel/ngrok_integration_test.go",
		"internal/tunnel/redaction_test.go",
		"internal/tunnel/secret_test.go",
		"internal/tunnel/settings_test.go",
		"internal/testutil/public_access_fakes_test.go",
	}
	protobufAware := map[string]string{
		"app_contract_test.go":                   "google.golang.org/protobuf/testing/prototest",
		"internal/platform/assets_test.go":       "google.golang.org/protobuf/testing/prototest",
		"internal/player/handler_test.go":        "google.golang.org/protobuf/testing/protocmp",
		"internal/player/public_stream_test.go":  "google.golang.org/protobuf/testing/protocmp",
		"internal/playerconfig/contract_test.go": "google.golang.org/protobuf/testing/protocmp",
		"internal/session/contract_test.go":      "google.golang.org/protobuf/testing/protocmp",
	}

	for _, relative := range files {
		t.Run(relative, func(t *testing.T) {
			parsed, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, relative), nil, 0)
			require.NoError(t, err)
			imports := make(map[string]string, len(parsed.Imports))
			for _, spec := range parsed.Imports {
				path, unquoteErr := strconv.Unquote(spec.Path.Value)
				require.NoError(t, unquoteErr)
				name := filepath.Base(path)
				if spec.Name != nil {
					name = spec.Name.Name
				}
				imports[path] = name
			}
			_, usesRequire := imports["github.com/stretchr/testify/require"]
			_, usesAssert := imports["github.com/stretchr/testify/assert"]
			usesTestify := usesRequire || usesAssert
			require.True(t, usesTestify, "%s must use Testify assertions", relative)
			if helper := protobufAware[relative]; helper != "" {
				_, usesHelper := imports[helper]
				require.True(t, usesHelper, "%s must use %s for protobuf values or descriptors", relative, helper)
			}

			testingNames := testParameterNames(parsed, imports["testing"])
			contextName := imports["context"]
			reflectName := imports["reflect"]
			requireName := imports["github.com/stretchr/testify/require"]
			ast.Inspect(parsed, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				owner, ok := selector.X.(*ast.Ident)
				if !ok {
					return true
				}
				if testingNames[owner.Name] {
					switch selector.Sel.Name {
					case "Fatal", "Fatalf", "Error", "Errorf", "Fail", "FailNow":
						assert.Fail(t, "%s uses direct testing.T failure %s; use Testify", relative, selector.Sel.Name)
					}
				}
				if owner.Name == contextName && (selector.Sel.Name == "Background" || selector.Sel.Name == "TODO") {
					assert.Fail(t, "%s creates a root context with context.%s; use t.Context()", relative, selector.Sel.Name)
				}
				if owner.Name == reflectName && selector.Sel.Name == "DeepEqual" {
					assert.Fail(t, "%s uses reflect.DeepEqual; use cmp.Equal or protobuf-aware comparison", relative)
				}
				if owner.Name == requireName {
					switch selector.Sel.Name {
					case "Fail", "Failf", "FailNow", "FailNowf":
						assert.Fail(t, "%s uses require.%s as a handwritten conditional assertion; use a semantic Testify helper", relative, selector.Sel.Name)
					}
				}
				return true
			})
		})
	}
}

func testParameterNames(file *ast.File, testingImport string) map[string]bool {
	names := make(map[string]bool)
	if testingImport == "" {
		return names
	}
	ast.Inspect(file, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok {
			return true
		}
		pointer, ok := field.Type.(*ast.StarExpr)
		if !ok {
			return true
		}
		selector, ok := pointer.X.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "T" {
			return true
		}
		owner, ok := selector.X.(*ast.Ident)
		if !ok || owner.Name != testingImport {
			return true
		}
		for _, name := range field.Names {
			names[name.Name] = true
		}
		return true
	})
	return names
}
