#!/usr/bin/env bash
# =============================================================================
# mock-users.sh — 生成模拟用户，对现有真实资源产生行为
#
# 复用库里现有的资源（如 B 站采集的视频），生成模拟用户并让每个资源至少
# 被 -min-item 个用户交互（默认 5，对齐 engine 阈值），跳过冷启动又不引入
# sim 资源。与 seed.sh 的区别：seed 会额外播 500 个 sim 资源。
#
# 用法：
#   bash scripts/mock-users.sh                    # 默认 50 用户
#   bash scripts/mock-users.sh -users 100 -min-item 5
#
# 可用环境变量覆盖：
#   CONFIG_PATH  后端配置（默认 configs/config.yaml，相对 backend/）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/backend"

CONFIG_PATH="${CONFIG_PATH:-configs/config.yaml}"

log() { echo "[mock-users] $*"; }
die() { echo "[mock-users] 错误: $*" >&2; exit 1; }

command -v go >/dev/null || die "未找到 go"

log "生成模拟用户（复用现有资源）"
( cd "$BACKEND_DIR" && CONFIG_PATH="$CONFIG_PATH" go run ./cmd/mock_users "$@" )
