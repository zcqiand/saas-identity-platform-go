#!/usr/bin/env bash
# scripts/gen-shared.sh — 从家族 shared 契约生成 API 面（REQ-2026-003，oapi-codegen std-http）
#
# 生成区双件，regen 先删后写——手改会被机械抹掉：
#   internal/gen/api/gen_api.go   oapi-codegen std-http-server + models（禁手改）
#   generated/api_manifest.json   genmanifest 路由面 SSOT（route-parity 契约门读它）
#
# 漂移护栏：跑完与 git HEAD 不一致（含 untracked）→ FATAL。
# 首跑 untracked FATAL 属预期——git add 生成区并 commit 后重跑核绿。
#
# 用法：bash scripts/gen-shared.sh   （无 DB 依赖；L4.codegen.idempotent strict 复放本脚本）
set -euo pipefail

# 不要用 `git rev-parse --show-toplevel` —— 本仓是 submodule，该命令返回外层
# xr-code-suite 根而不是本仓根（laravel/ktor 同款坑）。锚定脚本自身位置。
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/.."
SHARED_DIR="$(cd ../saas-identity-platform-shared && pwd)"

echo "[gen-shared] step 1/4 — shared emit:openapi"
( cd "$SHARED_DIR" && npm run emit:openapi )

echo "[gen-shared] step 2/4 — oapi-codegen std-http-server + models"
rm -f internal/gen/api/gen_api.go
go tool oapi-codegen -config oapi-codegen.cfg.yaml "$SHARED_DIR/generated/openapi/openapi.yaml"

echo "[gen-shared] step 3/4 — genmanifest（路由面 manifest）"
rm -f generated/api_manifest.json
go run ./scripts/genmanifest "$SHARED_DIR/generated/openapi/openapi.yaml"

echo "[gen-shared] step 4/4 — 漂移护栏（tracked diff ∪ untracked）"
if ! git diff --exit-code --quiet -- internal/gen/api generated 2>/dev/null \
  || [ -n "$(git ls-files --others --exclude-standard -- internal/gen/api generated)" ]; then
  echo "[gen-shared] FATAL: 生成区与 git HEAD 不一致（含 untracked 新文件）" >&2
  echo "[gen-shared]        处理：确认契约变更合法（shared emit 已收口）后" >&2
  echo "[gen-shared]        git add internal/gen/api generated && git commit，再重跑本脚本核绿" >&2
  exit 1
fi
echo "[gen-shared] OK — 生成区与 HEAD 一致"
