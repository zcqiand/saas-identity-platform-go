# 更新日志

## v0.2.1-20261027 (2026-10-27)

CORS 白名单中间件补交付（lab-go final review Critical#1 对称面；rails cors.rb 镜像）：

- `internal/impl/harness/cors.go`：SAAS_CORS_ALLOWED_ORIGINS 白名单 fail-fast、预检 204、ACAO 回显 + expose Authorization、非白名单不出 CORS 头
- env 契约 7 key（+.env.test 同键）；deploy go.env 自举 + append_if_missing 补 prod origin；family-env SAAS_CORS scope 纳 go
- 全 suite 8/8 ok，gate exit 0（L0.5 deploy parity 含新键）


格式照 `v<MAJOR>.<MINOR>.<PATCH>-<YYYYMMDD>`（tag 即放行：全量回归绿后打 tag）。

## v0.2.0-20261027 (2026-10-27)

REQ-2026-003 全量业务切片（shared 契约 47 op / 11 tag 全实现）：

- codegen 双链：oapi-codegen v2.8.1 std-http-server 生成区（internal/gen/api）+ GORM DB-First entity 12 表（internal/gen/entity），双幂等门（route-parity + codegen strict）接管
- 基础件：render.Camelize/WriteJSON、errors.{code,message} 映射、auth（HS256 签验/密码校验/守卫+租户缝）、testsupport（单事务 Feature 隔离）
- 11 tag 全落地：auth（登录/登出）、me（whoami/tenants/menus/switch）、tenant-members（CRUD+邀请+状态双写+授权）、tenant-roles、tenant-role-menus（行锁差量）、client-menus、tenant-applications（订阅校验链）、admin-clients、admin-tenants、oauth（authorize 码流 + authorization_code/refresh_token 双 grant、code 一次性消费）、clients 匿名元数据
- env 六键契约（SERVER_PORT/DATABASE_URL/JWT_* 四件）三件套严格相等 + deploy 自举同步
- contract-test 仓 TARGETS 注册 go:5111，live 2-way（springboot 参照）收敛

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
