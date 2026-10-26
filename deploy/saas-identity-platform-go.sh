#!/bin/sh
# Usage: saas-identity-platform-go.sh <DOCKER_USERNAME> <DOCKER_PASSWORD> [VERSION]
#
# 由 .github/workflows/ci.yml 的 deploy job 远程调用:
#   ssh deploy@vps -- cd /home/deploy/saas-identity-platform-go
#                    && sh saas-identity-platform-go.sh $DOCKER_USERNAME $DOCKER_PASSWORD $VERSION
#
# VERSION 默认是 latest。tag-based deploy 时显式传 tag 名（v0.x.y-YYYYMMDD）。
# CI 同时 push :latest + :<tag> 两份镜像,回滚只要手动指定旧 tag 再跑一次本脚本。
#
# 与姊妹仓 saas-identity-platform-kotlin.sh 的差异:
#   - 运行时 Go 1.27 静态二进制（golang:1.27-alpine builder, CGO_ENABLED=0,
#     alpine:3 runtime;镜像内零 JVM/解释器）
#   - env 契约 6 key = .env.example 全集（REQ-2026-003 全量业务切片起接 PG/JWT）——
#     PG_PASSWORD 走 SSH envs 转发（缺则 fail fast）；JWT_SIGNING_KEY 转发缺省随机
#     生成持久化；DATABASE_URL 由 PG_HOST/PG_PASSWORD 合成（远程 saas_prod）
#   - 绑全接口 ":port"（容器可达;kotor 先例 Netty 默认 0.0.0.0 同姿态）
#
# 前置: deploy 用户需在 docker 组中(sudo usermod -aG docker deploy)。
#        go.env 由本脚本首启自举（PG_PASSWORD 必须由 CI secrets 转发）。

set -eu

USERNAME="${1:-}"
PASSWORD="${2:-}"
VERSION="${3:-latest}"
IMAGE="${USERNAME}/saas-identity-platform-go:${VERSION}"
BASE="/home/deploy/saas-identity-platform-go"
CONTAINER_NAME="saas-identity-platform-go"

NGINX_DOMAIN="${NGINX_DOMAIN:-saas-go.xiangru.uk}"
NGINX_CERT_BASENAME="${NGINX_CERT_BASENAME:-xiangru-uk}"

if [ -z "$USERNAME" ] || [ -z "$PASSWORD" ]; then
  echo "Usage: $0 <DOCKER_USERNAME> <DOCKER_PASSWORD> [VERSION]" >&2
  exit 2
fi

# go.env 自举: env 契约 = .env.example 全集 6 key（L0.5 deploy parity 比对基线）。
# PG_PASSWORD 缺转发 = fail fast；JWT_SIGNING_KEY 转发缺省随机生成（持久化落 env-file）。
if [ ! -f "$BASE/go.env" ]; then
  if [ -n "${PG_PASSWORD:-}" ]; then
    echo "→ bootstrapping $BASE/go.env from env PG_PASSWORD"
    umask 077
    {
      printf 'SERVER_PORT=5111\n'
      printf 'DATABASE_URL=postgresql://postgres:%s@%s:5432/saas_prod\n' "$PG_PASSWORD" "${PG_HOST:-100.79.128.25}"
      printf 'JWT_SIGNING_KEY=%s\n' "${JWT_SIGNING_KEY:-$(head -c 48 /dev/urandom | base64 | tr -d '\n')}"
      printf 'JWT_TTL_SECONDS=3600\n'
      printf 'JWT_ISSUER=saas-identity-platform\n'
      printf 'JWT_AUDIENCE=saas-identity-platform-clients\n'
    } > "$BASE/go.env"
    chown deploy:deploy "$BASE/go.env" 2>/dev/null || true
    chmod 600 "$BASE/go.env"
  else
    echo "ERROR: $BASE/go.env missing AND PG_PASSWORD secret not forwarded (CI envs: PG_HOST,PG_PASSWORD,JWT_SIGNING_KEY)" >&2
    exit 1
  fi
fi
if ! grep -q '^SERVER_PORT=..*' "$BASE/go.env"; then
  echo "ERROR: $BASE/go.env has no SERVER_PORT line" >&2
  exit 1
fi
if ! grep -q '^DATABASE_URL=..*' "$BASE/go.env"; then
  echo "ERROR: $BASE/go.env has no DATABASE_URL line (REQ-2026-003 起业务面必需；存量老 env 请删除后由本脚本重新自举)" >&2
  exit 1
fi

# nginx vhost 重渲染(每次 deploy 都跑,ADR-0018:容器端口变了 vhost 必须跟)。
# 模板每次都从 master 拉最新 —— VPS 本地老模板会渲染出老端口全家族 502(2026-09-03 事故)。
# ?v=$(date +%s) cache-buster：CDN/raw 缓存防在案（flutter 先例）。
NGINX_SITES_AVAILABLE="/etc/nginx/sites-available"
NGINX_SITES_ENABLED="/etc/nginx/sites-enabled"
NGINX_VHOST_FILE="${NGINX_SITES_AVAILABLE}/${NGINX_DOMAIN}"
NGINX_VHOST_LINK="${NGINX_SITES_ENABLED}/${NGINX_DOMAIN}"
NGINX_TEMPLATE="${BASE}/nginx-vps.conf.example"

echo "→ fetching nginx-vps.conf.example template (always fresh from master)"
curl -fsSL "https://raw.githubusercontent.com/zcqiand/saas-identity-platform-go/refs/heads/master/deploy/nginx-vps.conf.example?v=$(date +%s)" -o "${NGINX_TEMPLATE}"

# 渲染到临时文件 —— sed 顺序:cert 归一化规则必须排在 <domain>/YOUR_DOMAIN 通配之前
# (先替换 <domain> 会把 cert 路径占位符一并吃掉,2026-09-03 事故根因)。
TMP_VHOST="$(mktemp -t vpstpl.XXXXXX)"
sed \
  -e "s|/etc/nginx/ssl/<domain>\.crt|/etc/nginx/ssl/${NGINX_CERT_BASENAME}.cert|g" \
  -e "s|/etc/nginx/ssl/<domain>\.cert|/etc/nginx/ssl/${NGINX_CERT_BASENAME}.cert|g" \
  -e "s|/etc/nginx/ssl/<domain>\.key|/etc/nginx/ssl/${NGINX_CERT_BASENAME}.key|g" \
  -e "s|/etc/nginx/ssl/your-cert\.crt|/etc/nginx/ssl/${NGINX_CERT_BASENAME}.cert|g" \
  -e "s|/etc/nginx/ssl/your-cert\.cert|/etc/nginx/ssl/${NGINX_CERT_BASENAME}.cert|g" \
  -e "s|/etc/nginx/ssl/your-cert\.key|/etc/nginx/ssl/${NGINX_CERT_BASENAME}.key|g" \
  -e "s|<domain>|${NGINX_DOMAIN}|g" \
  -e "s|lab\.YOUR_DOMAIN|${NGINX_DOMAIN}|g" \
  -e "s|saas\.YOUR_DOMAIN|${NGINX_DOMAIN}|g" \
  "${NGINX_TEMPLATE}" > "${TMP_VHOST}"

if [ -e "${NGINX_VHOST_FILE}" ] && diff -q "${TMP_VHOST}" "${NGINX_VHOST_FILE}" >/dev/null 2>&1; then
  echo "→ nginx vhost ${NGINX_VHOST_FILE} unchanged, skip"
  rm -f "${TMP_VHOST}"
else
  echo "→ rendering nginx vhost ${NGINX_VHOST_FILE} (domain=${NGINX_DOMAIN} cert=${NGINX_CERT_BASENAME})"
  if [ -w "${NGINX_SITES_AVAILABLE}" ]; then
    cp "${TMP_VHOST}" "${NGINX_VHOST_FILE}"
  else
    sudo cp "${TMP_VHOST}" "${NGINX_VHOST_FILE}" \
      || { echo "ERROR: sudo cp ${NGINX_VHOST_FILE} failed"; rm -f "${TMP_VHOST}"; exit 1; }
  fi
  if [ -w "${NGINX_SITES_ENABLED}" ]; then
    ln -sf "${NGINX_VHOST_FILE}" "${NGINX_VHOST_LINK}"
  else
    sudo ln -sf "${NGINX_VHOST_FILE}" "${NGINX_VHOST_LINK}" \
      || { echo "ERROR: sudo ln ${NGINX_VHOST_LINK} failed"; rm -f "${TMP_VHOST}"; exit 1; }
  fi
  rm -f "${TMP_VHOST}"
  echo "→ nginx -t"
  sudo nginx -t
  echo "→ systemctl reload nginx"
  sudo systemctl reload nginx
  echo "✓ nginx reloaded"
fi

echo "→ image: $IMAGE"
echo "→ docker login"
printf '%s' "$PASSWORD" | docker login -u "$USERNAME" --password-stdin

echo "→ docker pull"
docker pull "$IMAGE"

echo "→ docker stop & rm $CONTAINER_NAME"
docker stop "$CONTAINER_NAME" 2>/dev/null || true
docker rm "$CONTAINER_NAME" 2>/dev/null || true

echo "→ docker run"
docker run -d \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  -p "127.0.0.1:5111:5111" \
  --env-file "$BASE/go.env" \
  "$IMAGE"

echo "→ docker image prune"
docker image prune -f

echo "→ docker ps"
docker ps --filter name="$CONTAINER_NAME"

# 健康检查: 直接 wget /health 探 200。容器死亡提前终止循环, 立刻报失败。
i=0
while [ $i -lt 120 ]; do
  if wget --tries=1 --timeout=3 -q "http://127.0.0.1:5111/health" -O /dev/null 2>/dev/null; then
    echo "→ /health 200 (host 127.0.0.1:5111) after ${i}s"
    break
  fi
  if ! docker inspect --format='{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null | grep -q true; then
    echo "→ container not running, logs:"
    docker logs --tail 30 "$CONTAINER_NAME"
    exit 1
  fi
  i=$((i+1))
  sleep 1
done

if [ $i -ge 120 ]; then
  echo "→ /health 仍未 200(120s 上限), logs:"
  docker logs --tail 30 "$CONTAINER_NAME"
  exit 1
fi

echo "→ deploy done at $(date -u)"
