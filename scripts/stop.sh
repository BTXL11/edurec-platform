#!/usr/bin/env bash
# =============================================================================
# stop.sh — 停止本地开发/演示服务
#
# 停止 start.sh 启动的后端 / 前端进程（按 PID 文件 + 进程名兜底）。
# 加 --with-db 参数时连 MySQL/Redis 容器一起停（默认不动容器）。
#
# 用法：
#   bash scripts/stop.sh            # 只停后端 + 前端
#   bash scripts/stop.sh --with-db  # 连容器一起停
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RUN_DIR="$SCRIPT_DIR/.run"

WITH_DB=0
for arg in "$@"; do
  case "$arg" in
    --with-db) WITH_DB=1 ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
    *) echo "未知参数: $arg" >&2; exit 2 ;;
  esac
done

log() { printf '\n\033[1;36m[stop]\033[0m %s\n' "$*"; }

stop_by_pid() {
  local pidfile="$1" name="$2"
  if [ -f "$pidfile" ]; then
    local pid; pid="$(cat "$pidfile")"
    if kill "$pid" 2>/dev/null; then
      echo "已停止 $name (pid $pid)"
    fi
    rm -f "$pidfile"
  fi
}

log "停止后端 / 前端"
stop_by_pid "$RUN_DIR/backend.pid" "后端"
stop_by_pid "$RUN_DIR/frontend.pid" "前端"
# 兜底：清理可能的残留进程（go run 会派生实际 server 子进程）
pkill -f "go run ./cmd/server" 2>/dev/null && echo "已清理残留后端进程" || true
pkill -f "edurec-platform/frontend" 2>/dev/null && echo "已清理残留前端进程" || true

if [ "$WITH_DB" = 1 ]; then
  log "停止 MySQL / Redis 容器"
  docker stop edurec-mysql edurec-redis 2>/dev/null || true
  echo "容器已停止（数据保留，下次运行 start.sh 会自动启动容器）"
fi

log "完成"
