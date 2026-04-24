#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TOOLS_DIR="${TOOLS_DIR:-${ROOT_DIR}/tmp/tools}"
mkdir -p "${TOOLS_DIR}"

GOBIN="${TOOLS_DIR}" go install golang.org/x/vuln/cmd/govulncheck@latest

(cd "${ROOT_DIR}/services/api" && "${TOOLS_DIR}/govulncheck" ./...)
(cd "${ROOT_DIR}/services/judge" && "${TOOLS_DIR}/govulncheck" ./...)

printf 'go vulnerability verification passed\n'
