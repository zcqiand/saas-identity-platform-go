# REQ-2026-002 进 prod deploy 链（ktor/laravel 先例五件套）

| 项 | 值 |
|---|---|
| 提出人 | zcqiand（人裁 2026-10-26「go 双仓进 prod deploy 链」） |
| 提出日期 | 2026-10-26 |
| 优先级 | P1 |
| 状态 | 已验收 |
| 关联 ADR | ADR-0018（host=container 单层端口）；suite-hard-rules §1（禁 env 兜底） |

## 1. 需求描述

用户原话：「进 prod deploy 链 (Recommended)」（AskUserQuestion 三选一，go 双仓下一步）。

理解：照 ktor/laravel REQ-2026-001 同款节奏，把 saas-go（X13=5111）送进生产部署链——
deploy 五件套（Dockerfile + ci.yml + deploy 脚本 + nginx vhost 模板 + setup-vps.sh），
tag push → CI 镜像 build&push → VPS 拉镜像起容器 → `/health` 探活 →
`https://saas-go.xiangru.uk` 上线；lab-go（X13=5211）复刻。

### 澄清记录

| 疑问 | 澄清结论 | 澄清人 | 日期 |
|---|---|---|---|
| 容器内绑定地址：main.go 现绑 `127.0.0.1`，host 端口映射打不进容器回环 | 改绑 `:port`（全接口）——ktor 先例：`embeddedServer(Netty, port=…)` 无 host 参数即 Netty 默认 0.0.0.0，dev/prod 同姿态，不加新 env key | Claude（实证后照先例） | 2026-10-26 |
| 本仓无 PG/JWT 接线，env 契约只有 SERVER_PORT | deploy env-file 只有 `SERVER_PORT=5111` 字面量，无需 secret 转发（ktor 的 DATABASE_URL/JWT 一套不适用）；后续 PG/JWT 需求进来时再扩 | Claude（照 .env.example 全集） | 2026-10-26 |
| GitHub secrets 双 go 仓为空 | 首航前补 DOCKER_USERNAME/DOCKER_PASSWORD/VPS_HOST/VPS_USER/VPS_SSH_KEY 5 secrets + NGINX_DOMAIN/NGINX_CERT_BASENAME 2 vars（值源与 kotlin 同源） | Claude | 2026-10-26 |

## 2. 验收标准

| 编号 | 场景（给定） | 操作（当） | 预期（则） |
|---|---|---|---|
| AC-1 | 仓内 deploy 五件套齐全 | `ls deploy/ Dockerfile .github/workflows/ci.yml` | 五件在盘，`sh -n` 语法过 |
| AC-2 | push tag `v*` | CI test job | gofmt/vet/build/test + 镜像装配冒烟（build+run+/health 200）全绿 |
| AC-3 | CI test 绿 | deploy job | 镜像 push Docker Hub（latest+tag 双份）→ VPS 起容器 → `/health` 120s 内 200 |
| AC-4 | 部署完成 | `curl https://saas-go.xiangru.uk/health` | `{"ok":true,"mode":"go"}` |
| AC-5 | 本地门禁 | `python scripts/gate.py -p saas-identity-platform-go` | L0-L5 全绿 |

## 3. 任务拆解

| 任务 ID | 任务描述 | 类型 | 负责人 | 预估 | 状态 |
|---|---|---|---|---|---|
| T-1 | 绑定地址 `127.0.0.1:%d` → `:%d`（容器可达，ktor 先例）+ 注释 | 代码 | Claude | 5min | 已完成 |
| T-2 | Dockerfile（golang builder 两段式，CGO_ENABLED=0 静态二进制，alpine runtime + HEALTHCHECK GET 探针）+ .dockerignore | 基础设施 | Claude | 30min | 已完成 |
| T-3 | ci.yml（test：fmt/vet/build/test + 镜像冒烟；deploy：login→push→ssh 拉脚本起容器） | 基础设施 | Claude | 30min | 已完成 |
| T-4 | deploy/saas-identity-platform-go.sh + nginx-vps.conf.example + setup-vps.sh | 基础设施 | Claude | 30min | 已完成 |
| T-5 | GitHub secrets/vars 补配 + tag `v0.1.1-20261026` 首航 + live 探针收账 | 运维 | Claude | 20min | 已完成 |

## 4. 功能影响（需求与功能对齐的唯一位置）

基础设施需求：影响功能数 **0**（ktor REQ-2026-001/003 台账先例——不挂 fn ID、不动树）。

| 功能 ID | 功能名称 | 影响类型 | 说明 | 关联任务 |
|---|---|---|---|---|
| — | — | — | 基础设施端点 `/health` 不挂 fn ID；绑定地址为装配参数非功能语义 | — |

## 5. 流程影响

无。

## 6. 风险与回滚

| 风险 | 影响面 | 缓解 | 回滚方式 |
|---|---|---|---|
| dev 侧改绑全接口（:5111 对局域网监听） | 本机开发 | ktor 家族同姿态；Windows 防火墙默认拦外网入站 | 回退 commit 重打 tag |
| 镜像首航指纹（路径形状/探针方法/busybox 解析） | 首航 CI | ktor 五指纹预检清单逐项过（127.0.0.1 字面量、-O /dev/null 非 --spider） | `gh run rerun --failed` / 旧 tag 回滚重跑脚本 |
| 双仓并发 deploy 撞 VPS prune | 两仓 deploy job | tag 串行推（saas 绿后再 lab，60s 间隔先例） | rerun failed job |
