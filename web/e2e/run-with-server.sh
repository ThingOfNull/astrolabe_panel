#!/usr/bin/env bash
# Phase 4 E2E 启动器：
#
#   - 在 18080 端口启动一个独立的 ./astrolabe（隔离 db 到临时目录）
#   - 等待 /healthz 通；
#   - 运行 playwright test
#   - 无论成功失败都 kill astrolabe + 清理临时目录
#
# 使用：在 web/ 下 `bash e2e/run-with-server.sh`，需要先在仓库根 make build。
set -euo pipefail

E2E_PORT="${E2E_PORT:-18080}"
ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
BIN="$ROOT_DIR/astrolabe"
TMP_HOME="$(mktemp -d -t astro-e2e.XXXXXX)"

if [[ ! -x "$BIN" ]]; then
  echo "binary $BIN not found; run 'make build' from repo root first" >&2
  exit 1
fi

cleanup() {
  if [[ -n "${ASTRO_PID:-}" ]] && kill -0 "$ASTRO_PID" 2>/dev/null; then
    kill "$ASTRO_PID" 2>/dev/null || true
    sleep 1
    kill -9 "$ASTRO_PID" 2>/dev/null || true
  fi
  rm -rf "$TMP_HOME"
}
trap cleanup EXIT

# 用临时 HOME 隔离配置 / db / uploads
HOME_OVERRIDE="$TMP_HOME"
mkdir -p "$HOME_OVERRIDE/.astrolabe_panel"
cat > "$HOME_OVERRIDE/.astrolabe_panel/config.json" <<JSON
{
  "server": { "listen_host": "127.0.0.1", "listen_port": $E2E_PORT, "base_url": "" },
  "paths": {
    "data_dir":   "$HOME_OVERRIDE/.astrolabe_panel",
    "db_path":    "$HOME_OVERRIDE/.astrolabe_panel/astrolabe.db",
    "upload_dir": "$HOME_OVERRIDE/.astrolabe_panel/uploads",
    "log_dir":    "$HOME_OVERRIDE/.astrolabe_panel/logs"
  },
  "log": { "level": "warn", "format": "console" },
  "i18n": { "default_locale": "zh-CN" },
  "iconify": { "mirror": "" },
  "metric": { "retention_minutes": 30, "cleanup_interval_minutes": 5 },
  "probe": { "default_interval_sec": 30, "default_timeout_sec": 4 }
}
JSON

HOME="$HOME_OVERRIDE" "$BIN" --config "$HOME_OVERRIDE/.astrolabe_panel/config.json" \
  > "$TMP_HOME/server.log" 2>&1 &
ASTRO_PID=$!

echo "astrolabe e2e pid=$ASTRO_PID port=$E2E_PORT home=$HOME_OVERRIDE"
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:$E2E_PORT/healthz" >/dev/null; then
    break
  fi
  sleep 0.5
done
if ! curl -sf "http://127.0.0.1:$E2E_PORT/healthz" >/dev/null; then
  echo "server did not become healthy:" >&2
  tail -50 "$TMP_HOME/server.log" >&2 || true
  exit 1
fi

E2E_PORT="$E2E_PORT" pnpm exec playwright test "$@"
