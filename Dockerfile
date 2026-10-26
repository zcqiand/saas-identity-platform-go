# =============================================================================
# saas-identity-platform-go — 生产镜像
#
#   builder  → golang:1.27-alpine + CGO_ENABLED=0 静态编译（零三方依赖，
#              go.mod 无 require——go mod download 空转也保持缓存友好分层）
#   runtime  → alpine:3（静态二进制无 libc 依赖，musl 无碍——CGO 已关；
#              家族 jammy 姿态是给 JVM/native deps 的，go 静态二进制不需要）
#
# 数据库：本仓尚无 PG 接线（REQ-2026-001 健康切片形态）——env 契约只有
#         SERVER_PORT；PG/JWT 需求进来时同 commit 扩 env 契约 + deploy 脚本。
#
# 端口：容器内监听 SERVER_PORT=5111（host=container 单层映射，ADR-0018；
#       X13 槽）。SERVER_PORT 不在镜像内 ENV 兜底——fail-fast 键由 env-file
#       保证（suite 硬规则 §1：禁 env 默认值兜底）。
#
# 探针（ktor 首航指纹 3，2026-10-25 在案）：--spider 发 HEAD；本服务 mux
#       "GET /health" 按 net/http 语义 HEAD 也命中，但仍统一 -O /dev/null 真拉
#       GET——与家族 HEALTHCHECK 形态一致，不赌方法匹配细节。
#       地址用 127.0.0.1 字面量（busybox wget localhost→::1 指纹，2026-10-06 在案）。
# =============================================================================

# ---------- Stage 1: builder ----------
FROM golang:1.27-alpine AS builder
WORKDIR /app

# 缓存友好分层：先拷 go.mod（零依赖，仍走标准姿势），再拷源码
COPY go.mod ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/bin/server ./cmd/server

# ---------- Stage 2: runtime ----------
FROM alpine:3 AS runtime
WORKDIR /app

# busybox wget 自带（alpine 基础件），探针够用；ca-certificates 给后续出站 TLS 预留
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 appuser

COPY --from=builder /app/bin/server /app/server

USER appuser
ENV TZ=UTC

EXPOSE 5111

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:5111/health || exit 1

ENTRYPOINT ["/app/server"]
