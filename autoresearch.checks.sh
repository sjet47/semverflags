#!/bin/bash
set -euo pipefail

go test ./... >/tmp/semverflags-autoresearch-checks.log 2>&1 || {
  tail -80 /tmp/semverflags-autoresearch-checks.log
  exit 1
}
