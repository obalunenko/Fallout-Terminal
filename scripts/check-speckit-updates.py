#!/usr/bin/env python3
"""Check pinned and installed Spec Kit versions without changing them."""

from __future__ import annotations

import json
import os
import re
import shutil
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path


SEMVER_RE = re.compile(r"^v?(\d+)\.(\d+)\.(\d+)$")
REQUIRED_ENV = (
    "SPECKIT_VERSION",
    "SPECKIT_COMPANION_VERSION",
    "SPECKIT_BROWNFIELD_VERSION",
    "SPECKIT_BUGFIX_VERSION",
    "SPECKIT_FEATURE_NUMBERING_VERSION",
)
UPSTREAMS = {
    "companion": (
        "alfredoperez/speckit-companion",
        "releases",
        re.compile(r"^speckit-ext-v(\d+\.\d+\.\d+)$"),
    ),
    "brownfield": (
        "Quratulain-bilal/spec-kit-brownfield",
        "tags",
        re.compile(r"^v(\d+\.\d+\.\d+)$"),
    ),
    "bugfix": (
        "Quratulain-bilal/spec-kit-bugfix",
        "tags",
        re.compile(r"^v(\d+\.\d+\.\d+)$"),
    ),
}


def semantic_version(value: str) -> tuple[int, int, int]:
    match = SEMVER_RE.fullmatch(value.strip())
    if match is None:
        raise ValueError(f"unsupported semantic version: {value}")
    return tuple(int(part, 10) for part in match.groups())


def manifest_version(path: Path) -> str:
    in_extension = False
    for line in path.read_text(encoding="utf-8").splitlines():
        if line == "extension:":
            in_extension = True
            continue
        if in_extension and line and not line.startswith(" "):
            break
        if in_extension and line.startswith("  version:"):
            return line.split(":", 1)[1].strip().strip('"\'')
    raise ValueError(f"extension version is missing in {path}")


def latest_github_version(
    repository: str,
    endpoint: str,
    tag_pattern: re.Pattern[str],
) -> str:
    request = urllib.request.Request(
        f"https://api.github.com/repos/{repository}/{endpoint}?per_page=100",
        headers={
            "Accept": "application/vnd.github+json",
            "User-Agent": "fallout-terminal-speckit-update-check",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")
    if token:
        request.add_header("Authorization", f"Bearer {token}")
    with urllib.request.urlopen(request, timeout=20) as response:
        tags = json.load(response)

    versions = []
    name_field = "tag_name" if endpoint == "releases" else "name"
    for tag in tags:
        name = str(tag.get(name_field, ""))
        match = tag_pattern.fullmatch(name)
        if match is not None:
            version = match.group(1)
            versions.append((semantic_version(version), version))
    if not versions:
        raise ValueError(f"no matching stable releases found for {repository}")
    return max(versions)[1]


def pin_state(installed: str, pinned: str) -> str:
    if semantic_version(installed) == semantic_version(pinned):
        return "installed matches pin"
    return "PIN DRIFT"


def update_state(pinned: str, latest: str) -> str:
    if semantic_version(latest) > semantic_version(pinned):
        return "UPDATE AVAILABLE"
    if semantic_version(latest) < semantic_version(pinned):
        return "pin is ahead of discovered tags"
    return "pin is current"


def main() -> int:
    missing = [name for name in REQUIRED_ENV if not os.environ.get(name)]
    if missing:
        print(f"Missing required environment variables: {', '.join(missing)}", file=sys.stderr)
        return 1

    repo_root = Path(__file__).resolve().parent.parent
    registry_path = repo_root / ".specify/extensions/.registry"
    if not registry_path.is_file():
        print("Missing Spec Kit extension registry; run make speckit-install first.", file=sys.stderr)
        return 1

    specify_bin = shutil.which("specify")
    if specify_bin is None:
        print("Specify CLI is not on PATH; run make speckit-install first.", file=sys.stderr)
        return 1

    pinned_cli = os.environ["SPECKIT_VERSION"].removeprefix("v")
    version_result = subprocess.run(
        [specify_bin, "--version"],
        check=True,
        capture_output=True,
        text=True,
    )
    installed_match = re.search(r"(\d+\.\d+\.\d+)", version_result.stdout)
    if installed_match is None:
        print("Could not parse installed Specify CLI version.", file=sys.stderr)
        return 1
    installed_cli = installed_match.group(1)

    print("Specify CLI")
    print(f"  installed: {installed_cli}")
    print(f"  pinned:    {pinned_cli} ({pin_state(installed_cli, pinned_cli)})")
    self_check = subprocess.run(
        [specify_bin, "self", "check"],
        check=False,
        capture_output=True,
        text=True,
    )
    self_check_output = (self_check.stdout or self_check.stderr).strip()
    for line in self_check_output.splitlines():
        print(f"  {line}")

    registry = json.loads(registry_path.read_text(encoding="utf-8"))
    installed_extensions = registry.get("extensions", {})
    pins = {
        "companion": os.environ["SPECKIT_COMPANION_VERSION"],
        "brownfield": os.environ["SPECKIT_BROWNFIELD_VERSION"],
        "bugfix": os.environ["SPECKIT_BUGFIX_VERSION"],
        "feature-numbering": os.environ["SPECKIT_FEATURE_NUMBERING_VERSION"],
    }

    errors: list[str] = []
    print("\nSpec Kit extensions/plugins")
    for extension_id, metadata in sorted(installed_extensions.items()):
        installed = str(metadata.get("version", "unknown"))
        pinned = pins.get(extension_id)
        print(f"  {extension_id}")
        print(f"    installed: {installed}")
        if pinned is None:
            print("    pinned:    unmanaged (add a Makefile pin to track it)")
        else:
            try:
                state = pin_state(installed, pinned)
            except ValueError as exc:
                state = str(exc)
                errors.append(f"{extension_id}: {exc}")
            print(f"    pinned:    {pinned} ({state})")

        if extension_id == "feature-numbering":
            source_manifest = repo_root / ".speckit/feature-numbering/extension.yml"
            try:
                source = manifest_version(source_manifest)
                source_state = "source matches pin" if source == pinned else "SOURCE/PIN DRIFT"
                print(f"    source:    {source} ({source_state}; local extension)")
                if source != pinned:
                    errors.append(f"feature-numbering source {source} != pin {pinned}")
            except (OSError, ValueError) as exc:
                print(f"    source:    check failed: {exc}")
                errors.append(f"feature-numbering: {exc}")
            continue

        upstream = UPSTREAMS.get(extension_id)
        if upstream is None:
            print("    upstream:  not configured")
            continue
        try:
            repository, endpoint, tag_pattern = upstream
            latest = latest_github_version(repository, endpoint, tag_pattern)
            comparison_pin = pinned or installed
            print(f"    upstream:  {latest} ({update_state(comparison_pin, latest)})")
        except (OSError, ValueError, urllib.error.URLError) as exc:
            print(f"    upstream:  check failed: {exc}")
            errors.append(f"{extension_id}: {exc}")

    uninstalled_pins = sorted(set(pins) - set(installed_extensions))
    for extension_id in uninstalled_pins:
        print(f"  {extension_id}: NOT INSTALLED (pinned {pins[extension_id]})")

    if self_check.returncode != 0:
        errors.append("specify self check failed")
    if errors:
        print("\nUpdate check incomplete:", file=sys.stderr)
        for error in errors:
            print(f"  - {error}", file=sys.stderr)
        return 1

    print("\nRead-only check complete; no packages were changed.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
