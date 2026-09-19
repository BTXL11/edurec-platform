#!/usr/bin/env bash
# =============================================================================
# clean.sh — 清空数据库所有表数据（保留表结构）
#
# 清空 users/resources/categories/behaviors/ratings/recommendations 等全部表，
# 保留表结构。清空后可重新 seed.sh（sim 演示）或 bilibili.sh（真实数据）。
#
# 用法：
#   bash scripts/clean.sh
#
# 可用环境变量覆盖：
#   MYSQL_PASSWORD MySQL root 密码（默认 root）
#   MYSQL_DATABASE 数据库名（默认 edurec）
# =============================================================================
set -euo pipefail

MYSQL_PASSWORD="${MYSQL_PASSWORD:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-edurec}"

log() { echo "[clean] $*"; }
die() { echo "[clean] 错误: $*" >&2; exit 1; }

command -v docker >/dev/null || die "未找到 docker"

log "清空数据库 (db=$MYSQL_DATABASE)"
TABLES="$(docker exec edurec-mysql mysql -uroot -p"$MYSQL_PASSWORD" -N -e \
  "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA='$MYSQL_DATABASE'" 2>/dev/null || true)"
if [ -z "$TABLES" ]; then
  log "数据库为空，无需清理"
  exit 0
fi
for t in $TABLES; do
  docker exec edurec-mysql mysql -uroot -p"$MYSQL_PASSWORD" -e \
    "SET FOREIGN_KEY_CHECKS=0; TRUNCATE TABLE \`$MYSQL_DATABASE\`.\`$t\`; SET FOREIGN_KEY_CHECKS=1;" >/dev/null 2>&1
  echo "  $t: 已清空"
done
log "完成"
