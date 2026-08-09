#!/bin/sh
set -eu

binary_path=${1:?"usage: postbuild.sh path/to/App.app/Contents/MacOS/binary"}
case "$binary_path" in
  /*) ;;
  *) binary_path="$PWD/$binary_path" ;;
esac

macos_directory=$(dirname "$binary_path")
contents_directory=$(dirname "$macos_directory")
app_bundle=$(dirname "$contents_directory")
if [ "$(basename "$macos_directory")" != "MacOS" ] || [ "$(basename "$contents_directory")" != "Contents" ]; then
  echo "unexpected Wails bundle binary path: $binary_path" >&2
  exit 1
fi

script_directory=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repository_root=$(CDPATH= cd -- "$script_directory/../.." && pwd)
demo_source="$repository_root/sessions/demo.json"
demo_directory="$contents_directory/Resources/sessions"

test -s "$demo_source"
mkdir -p "$demo_directory"
install -m 0444 "$demo_source" "$demo_directory/demo.json"

# Wails' hook runs after packaging. Re-sign the completed bundle so adding the
# read-only demo resource does not invalidate the unsigned validation build.
/usr/bin/codesign --force --deep --options runtime \
  --entitlements "$script_directory/entitlements.plist" --sign - "$app_bundle"
