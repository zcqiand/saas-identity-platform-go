# 更新日志

格式照 `v<MAJOR>.<MINOR>.<PATCH>-<YYYYMMDD>`（tag 即放行：全量回归绿后打 tag）。

## v0.1.1-20261026 (2026-10-26)

REQ-2026-002 进 prod deploy 链（ktor/laravel 先例五件套）：

- deploy 五件套：Dockerfile（golang:1.27-alpine builder + CGO_ENABLED=0 静态二进制，alpine:3 runtime）+ .dockerignore + ci.yml（fmt/vet/build/test + 镜像装配冒烟 build+run+/health+405 守卫；deploy job tag 触发）+ deploy/ 脚本 + nginx vhost 模板 + setup-vps.sh
- 绑定地址 `127.0.0.1` → `:port`（容器可达；ktor Netty 默认 0.0.0.0 先例）
- `https://{domain}` 上线（X13={port}，host=container 单层映射，ADR-0018）


## v0.1.0-20261026 (2026-10-26)

REQ-2026-001 健康检查 + 端口装配最小切片——骨架可跑首航：

- `internal/config`：`SERVER_PORT` fail-fast 装配（缺键/非数字/超范围报错，禁默认值兜底）
- `internal/server`：`GET /health` → `{"ok":true,"mode":"go"}`（家族探活契约形状）+ 405 守卫
- `cmd/server`：组合根 `New(addr)`（实机指纹：http.Server 零值 Addr 落默认 :80，必须显式传地址）
- 删 `internal/harness` 空仓锚点包（首个装配包落地）
- live 验收：GET 200 / POST 405 / 缺 env exit 1；门禁 L0-L5 全绿（16 条目 0 软告警）
