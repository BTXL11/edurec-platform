#!/usr/bin/env bash
# =============================================================================
# start.sh — 本地开发/演示一键启动
#
# 依次完成：起 MySQL/Redis 容器 → 等 MySQL 就绪 → 准备模拟数据 →
# 幂等播种 → 后台启动后端 → 后台启动前端 → 打印访问信息。
#
# 与 scripts/handoff.sh 互补：本脚本负责"把服务跑起来"，
# 个性化推荐刷新仍用 handoff.sh（训练/推理/导入是重操作，不放进启动脚本）。
#
# 用法：
#   bash scripts/start.sh           # 一键启动
#   bash scripts/stop.sh         # 停止服务
#   bash scripts/stop.sh --with-db   # 连容器一起停
#
# 可用环境变量覆盖：
#   ENGINE_ROOT    engine 仓库根目录（默认 ../edurec-engine）
#   CONFIG_PATH    后端配置（默认 configs/config.yaml，相对 backend/）
#   MYSQL_PASSWORD MySQL root 密码（默认 root，与 config.yaml 对齐）
#   MYSQL_DATABASE 数据库名（默认 edurec）
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLATFORM_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKEND_DIR="$PLATFORM_ROOT/backend"
FRONTEND_DIR="$PLATFORM_ROOT/frontend"
RUN_DIR="$SCRIPT_DIR/.run"

ENGINE_ROOT="${ENGINE_ROOT:-$PLATFORM_ROOT/../edurec-engine}"
CONFIG_PATH="${CONFIG_PATH:-configs/config.yaml}"
MYSQL_PASSWORD="${MYSQL_PASSWORD:-root}"
MYSQL_DATABASE="${MYSQL_DATABASE:-edurec}"
BACKEND_PORT="${BACKEND_PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"

log() { printf '\n\033[1;36m[start]\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m[start][错误]\033[0m %s\n' "$*" >&2; exit 1; }
port_listening() { ss -tln 2>/dev/null | grep -qE ":${1}([[:space:]]|\$)"; }

# ---- 前置检查 ---------------------------------------------------------------
command -v docker >/dev/null || die "未找到 docker"
command -v go >/dev/null || die "未找到 go"
# 前端需要 node/pnpm（nvm 通常挂在 .zshrc，bash 里手动加载一次）
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh" || true
command -v pnpm >/dev/null || die "未找到 pnpm（前端需要）"

# ---- ① 容器 -----------------------------------------------------------------
log "① 确保 MySQL / Redis 容器运行"
if docker ps -a --format '{{.Names}}' | grep -qx edurec-mysql; then
  docker start edurec-mysql >/dev/null 2>&1 || true
  echo "MySQL 容器已启动（复用已有）"
else
  docker run -d --name edurec-mysql -p 3306:3306 \
    -e MYSQL_ROOT_PASSWORD="$MYSQL_PASSWORD" -e MYSQL_DATABASE="$MYSQL_DATABASE" mysql:8 >/dev/null
  echo "MySQL 容器已创建"
fi
if docker ps -a --format '{{.Names}}' | grep -qx edurec-redis; then
  docker start edurec-redis >/dev/null 2>&1 || true
  echo "Redis 容器已启动（复用已有）"
else
  docker run -d --name edurec-redis -p 6379:6379 redis:7 >/dev/null
  echo "Redis 容器已创建"
fi

# ---- ② 等 MySQL --------------------------------------------------------------
log "② 等待 MySQL 就绪"
for i in $(seq 1 30); do
  if docker exec edurec-mysql mysqladmin ping -uroot -p"$MYSQL_PASSWORD" --silent >/dev/null 2>&1; then
    echo "MySQL ready（第 ${i} 次探测）"
    break
  fi
  [ "$i" = 30 ] && die "MySQL 迟迟未就绪，查看日志：docker logs edurec-mysql"
  sleep 2
done

# ---- ③ 模拟数据 + 幂等播种 ---------------------------------------------------
log "③ 准备模拟数据并幂等播种"
[ -d "$ENGINE_ROOT" ] || die "engine 目录不存在: $ENGINE_ROOT"
if [ ! -d "$BACKEND_DIR/data/sim" ]; then
  [ -d "$ENGINE_ROOT/dataset/sim" ] || die "engine 无模拟数据: $ENGINE_ROOT/dataset/sim（先跑 gen_sim_data）"
  cp -r "$ENGINE_ROOT/dataset/sim" "$BACKEND_DIR/data/sim"
  echo "已从 engine 拷贝模拟数据 -> backend/data/sim"
fi
USER_COUNT="$(docker exec edurec-mysql mysql -uroot -p"$MYSQL_PASSWORD" -N -e \
  "SELECT COUNT(*) FROM $MYSQL_DATABASE.users" 2>/dev/null || echo 0)"
if [ "$USER_COUNT" = "0" ] || [ -z "$USER_COUNT" ]; then
  ( cd "$BACKEND_DIR" && CONFIG_PATH="$CONFIG_PATH" go run ./cmd/demo_seed -with-behaviors )
else
  echo "已有 ${USER_COUNT} 个用户，跳过播种"
fi

# ---- ④ 后端 ------------------------------------------------------------------
log "④ 启动后端 (:${BACKEND_PORT})"
mkdir -p "$RUN_DIR"
if port_listening "$BACKEND_PORT"; then
  echo "端口 ${BACKEND_PORT} 已在监听，跳过"
else
  ( cd "$BACKEND_DIR" && CONFIG_PATH="$CONFIG_PATH" \
    nohup go run ./cmd/server > "$RUN_DIR/backend.log" 2>&1 & \
    echo $! > "$RUN_DIR/backend.pid" )
  echo "已后台启动，日志 scripts/.run/backend.log"
fi

# ---- ⑤ 前端 ------------------------------------------------------------------
log "⑤ 启动前端 (:${FRONTEND_PORT})"
if port_listening "$FRONTEND_PORT"; then
  echo "端口 ${FRONTEND_PORT} 已在监听，跳过"
else
  ( cd "$FRONTEND_DIR" && \
    nohup pnpm dev > "$RUN_DIR/frontend.log" 2>&1 & \
    echo $! > "$RUN_DIR/frontend.pid" )
  echo "已后台启动，日志 scripts/.run/frontend.log"
fi

# ---- ⑥ 总结 ------------------------------------------------------------------
log "⑥ 完成"
MOCK_NOTE=""
if grep -q "VITE_MOCK=1" "$FRONTEND_DIR/.env.development" 2>/dev/null; then
  MOCK_NOTE="  ⚠ 前端当前是 mock 模式（VITE_MOCK=1），连真实后端请改 VITE_MOCK=0 并重启前端"
fi
cat <<EOF

  后端   http://localhost:${BACKEND_PORT}   日志 $RUN_DIR/backend.log
  前端   http://localhost:${FRONTEND_PORT}   日志 $RUN_DIR/frontend.log
  数据库 MySQL root/${MYSQL_PASSWORD} @ localhost:3306 / ${MYSQL_DATABASE}

  演示账号：demo1 / demo123456（普通用户）；demo_admin / demo123456（管理员）
${MOCK_NOTE}
  个性化推荐刷新：source $ENGINE_ROOT/.venv/bin/activate && bash scripts/handoff.sh
  停止服务：bash scripts/stop.sh
EOF
