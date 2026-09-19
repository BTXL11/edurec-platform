#!/usr/bin/env bash
# =============================================================================
# seed.sh — 播种 sim 演示数据
#
# 把 engine 的模拟数据集（dataset/sim）拷到 backend/data/sim，再用 demo_seed
# 播种进 MySQL（用户/资源/类目/行为/评分 + 管理员 demo_admin）。
# 保证平台库 ID 与 engine 推荐产物 ID 一致，使个性化推荐可完整演示。
#
# 幂等：已播种（users 表非空）则跳过；如需重置请手动清库。
#
# 用法：
#   bash scripts/seed.sh
#
# 可用环境变量覆盖：
#   ENGINE_ROOT    engine 仓库根目录（默认 ../edurec-engine）
#   CONFIG_PATH    后端配置（默认 configs/config.yaml，相对 backend/）
#   MYSQL_PASSWORD MySQL root 密码（默认 root）
#   MYSQL_DATABASE 数据库名（默认 edurec）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PLATFORM_ROOT/backend"

ENGINE_ROOT="${ENGINE_ROOT:-$PLATFORM_ROOT/../edurec-engine}"
CONFIG_PATH="${CONFIG_PATH:-configs/config.yaml}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-edurec}"

log() { printf '\n\033[1;36m[seed]\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m[seed][错误]\033[0m %s\n' "$*" >&2; exit 1; }

# ---- 前置检查 ---------------------------------------------------------------
command -v docker >/dev/null || die "未找到 docker（需 MySQL 容器在运行）"
command -v go >/dev/null || die "未找到 go"
[ -d "$ENGINE_ROOT" ] || die "engine 目录不存在: $ENGINE_ROOT"

# ---- ① 拷贝 sim 数据 ---------------------------------------------------------
log "① 准备模拟数据"
if [ ! -d "$BACKEND_DIR/data/sim" ]; then
  [ -d "$ENGINE_ROOT/dataset/sim" ] || die "engine 无模拟数据: $ENGINE_ROOT/dataset/sim（先跑 gen_sim_data）"
  cp -r "$ENGINE_ROOT/dataset/sim" "$BACKEND_DIR/data/sim"
  echo "已从 engine 拷贝 -> backend/data/sim"
else
  echo "backend/data/sim 已存在，跳过拷贝"
fi

# ---- ② 幂等播种 --------------------------------------------------------------
log "② 播种演示数据"
USER_COUNT="$(docker exec edurec-mysql mysql -uroot -p"$MYSQL_PASSWORD" -N -e \
  "SELECT COUNT(*) FROM $MYSQL_DATABASE.users" 2>/dev/null || echo 0)"
if [ "$USER_COUNT" = "0" ] || [ -z "$USER_COUNT" ]; then
  ( cd "$BACKEND_DIR" && CONFIG_PATH="$CONFIG_PATH" go run ./cmd/demo_seed -with-behaviors )
else
  echo "已有 ${USER_COUNT} 个用户，跳过播种（如需重置请手动清库）"
fi

log "完成。演示账号：demo1 / demo123456；管理员：demo_admin / demo123456"
