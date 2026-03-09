#!/usr/bin/env bash
# 로그 스트리밍 데모 실행 스크립트
# 사용: bash demo/run-log-streaming.sh
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

GOMODCACHE=/tmp/gomodcache go run "$ROOT_DIR/demo/log_streaming_demo.go"
