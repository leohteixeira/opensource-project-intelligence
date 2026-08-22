#!/usr/bin/env bash
# Fails when a reviewed source or committed generated output has drifted.
set -Eeuo pipefail

readonly REPOSITORY_ROOT="${REPOSITORY_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "${REPOSITORY_ROOT}"

sha256sum --check --strict generated.sha256
