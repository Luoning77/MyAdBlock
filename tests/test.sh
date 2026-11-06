#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
export GOMODCACHE="$PWD/.cache/gomod" GOCACHE="$PWD/.cache/gobuild"
export GOPROXY=${GOPROXY:-https://goproxy.cn,https://proxy.golang.org,direct}
GO=${GO:-$PWD/.tools/go/bin/go}
"$GO" test -race -count=1 ./...
"$GO" vet ./...
for s in module/*.sh; do sh -n "$s"; done
python3 tools/audit.py
