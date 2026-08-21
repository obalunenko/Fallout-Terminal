#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

fail() {
  printf 'tool module check: %s\n' "$1" >&2
  return 1
}

check_tool_module() {
  local scan_root="$1"
  local directory="$2"
  local command_package="$3"
  local parent_module="$4"
  local version="$5"
  local module_file="$scan_root/tools/$directory/go.mod"
  local sum_file="$scan_root/tools/$directory/go.sum"
  local tool_count

  [[ -f "$module_file" ]] || { fail "missing tools/$directory/go.mod"; return 1; }
  [[ -s "$sum_file" ]] || { fail "missing or empty tools/$directory/go.sum"; return 1; }
  grep -Eq '^go[[:space:]]+[0-9]+\.[0-9]+([.][0-9]+)?$' "$module_file" || {
    fail "tools/$directory/go.mod has no explicit Go version"
    return 1
  }
  tool_count="$(grep -Ec '^tool[[:space:]]+' "$module_file" || true)"
  [[ "$tool_count" == 1 ]] || {
    fail "tools/$directory/go.mod must declare exactly one tool"
    return 1
  }
  grep -Eq "^tool[[:space:]]+${command_package//\//\\/}$" "$module_file" || {
    fail "tools/$directory/go.mod does not own $command_package"
    return 1
  }
  grep -Eq "^[[:space:]]*(require[[:space:]]+)?${parent_module//\//\\/}[[:space:]]+${version//./\\.}([[:space:]]+// indirect)?$" "$module_file" || {
    fail "tools/$directory/go.mod does not pin $parent_module $version"
    return 1
  }
}

check_root_module() {
  local scan_root="$1"
  local root_module="$scan_root/go.mod"

  [[ -f "$root_module" ]] || { fail 'root go.mod is missing'; return 1; }
  if grep -En '^tool[[:space:]]*(\(|[^[:space:]])|github\.com/bufbuild/buf|github\.com/wailsapp/wails/v3/cmd/wails3|google\.golang\.org/protobuf/cmd/protoc-gen-go|connectrpc\.com/connect/cmd/protoc-gen-connect-go' "$root_module"; then
    fail 'root application go.mod contains a tool declaration or tool-only dependency'
    return 1
  fi
}

list_scan_files() {
  local scan_root="$1"

  if git -C "$scan_root" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git -C "$scan_root" ls-files -co --exclude-standard -z
  else
    find "$scan_root" -type f -print0
  fi
}

check_active_commands() {
  local scan_root="$1"
  local file
  local relative_file
  local file_matches
  local matches=''
  local forbidden_pattern='(go[[:space:]]+install[[:space:]]+(github\.com/wailsapp/wails|github\.com/bufbuild/buf/cmd/buf|google\.golang\.org/protobuf/cmd/protoc-gen-go|connectrpc\.com/connect/cmd/protoc-gen-connect-go)|go[[:space:]]+tool[[:space:]]+(wails3|buf|protoc-gen-go|protoc-gen-connect-go)([[:space:]]|$)|(^|[[:space:]`;&|])(wails3|buf)[[:space:]]+(dev|build|package|generate|format|lint|breaking)([[:space:]]|$))'
  local allowed_pattern='go tool -modfile=tools/(wails|buf|protoc-gen-go|protoc-gen-connect-go)/go\.mod (wails3|buf|protoc-gen-go|protoc-gen-connect-go)([[:space:]]|$)'

  while IFS= read -r -d '' file; do
    if [[ "$file" = /* ]]; then
      relative_file="${file#"$scan_root"/}"
    else
      relative_file="$file"
      file="$scan_root/$file"
    fi

    case "$relative_file" in
      scripts/tool-modules-check.sh|specs/*|docs/wails-migration-rollback.md|node_modules/*|frontend/node_modules/*|frontend/client/node_modules/*|frontend/overseer/node_modules/*|tests/browser/node_modules/*)
        continue
        ;;
    esac

    file_matches="$(LC_ALL=C grep -IEn "$forbidden_pattern" "$file" 2>/dev/null || true)"
    file_matches="$(printf '%s\n' "$file_matches" | grep -Ev "$allowed_pattern" || true)"
    if [[ -n "$file_matches" ]]; then
      matches+="${relative_file}:${file_matches}"$'\n'
    fi
  done < <(list_scan_files "$scan_root")

  [[ -z "$matches" ]] || {
    printf '%s' "$matches" >&2
    fail 'active files contain a global, bare, or root-module Go tool invocation'
    return 1
  }
}

check_tree() {
  local scan_root="$1"

  check_tool_module "$scan_root" wails github.com/wailsapp/wails/v3/cmd/wails3 github.com/wailsapp/wails/v3 v3.0.0-beta.10 || return
  check_tool_module "$scan_root" buf github.com/bufbuild/buf/cmd/buf github.com/bufbuild/buf v1.72.0 || return
  check_tool_module "$scan_root" protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/protobuf v1.36.11 || return
  check_tool_module "$scan_root" protoc-gen-connect-go connectrpc.com/connect/cmd/protoc-gen-connect-go connectrpc.com/connect v1.20.0 || return
  check_root_module "$scan_root" || return
  check_active_commands "$scan_root" || return
}

write_fixture_module() {
  local scan_root="$1"
  local directory="$2"
  local command_package="$3"
  local parent_module="$4"
  local version="$5"

  mkdir -p "$scan_root/tools/$directory"
  printf 'module example.test/tools/%s\n\ngo 1.27.0\n\ntool %s\n\nrequire %s %s\n' \
    "$directory" "$command_package" "$parent_module" "$version" >"$scan_root/tools/$directory/go.mod"
  printf '%s %s/go.mod h1:fixture\n' "$parent_module" "$version" >"$scan_root/tools/$directory/go.sum"
}

self_test() {
  local fixture_root
  fixture_root="$(mktemp -d)"
  trap 'rm -rf "$fixture_root"' RETURN

  printf 'module example.test/app\n\ngo 1.27.0\n' >"$fixture_root/go.mod"
  mkdir -p "$fixture_root/docs"
  printf 'go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...\n' >"$fixture_root/docs/commands.md"
  write_fixture_module "$fixture_root" wails github.com/wailsapp/wails/v3/cmd/wails3 github.com/wailsapp/wails/v3 v3.0.0-beta.10
  write_fixture_module "$fixture_root" buf github.com/bufbuild/buf/cmd/buf github.com/bufbuild/buf v1.72.0
  write_fixture_module "$fixture_root" protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go google.golang.org/protobuf v1.36.11
  write_fixture_module "$fixture_root" protoc-gen-connect-go connectrpc.com/connect/cmd/protoc-gen-connect-go connectrpc.com/connect v1.20.0
  check_tree "$fixture_root"

  printf '\ntool github.com/bufbuild/buf/cmd/buf\n' >>"$fixture_root/go.mod"
  if check_tree "$fixture_root" >/dev/null 2>&1; then
    fail 'self-test accepted a root tool declaration'
  fi
  printf 'module example.test/app\n\ngo 1.27.0\n' >"$fixture_root/go.mod"

  printf 'go install github.com/wailsapp/wails/v3/cmd/wails3@latest\n' >"$fixture_root/docs/commands.md"
  if check_tree "$fixture_root" >/dev/null 2>&1; then
    fail 'self-test accepted a global tool installation'
  fi

  printf 'go tool -modfile=tools/wails/go.mod wails3 generate bindings -clean ./...\n' >"$fixture_root/docs/commands.md"
  printf '\ntool example.test/second-tool\n' >>"$fixture_root/tools/wails/go.mod"
  if check_tree "$fixture_root" >/dev/null 2>&1; then
    fail 'self-test accepted multiple tools in one module'
  fi

  printf 'tool module check self-test passed\n'
}

case "${1:-}" in
  --self-test)
    self_test
    ;;
  '')
    check_tree "$repository_root"
    printf 'Go development tools are isolated, exactly pinned, and invoked through their owning modules.\n'
    ;;
  *)
    fail "unknown argument: $1"
    ;;
esac
