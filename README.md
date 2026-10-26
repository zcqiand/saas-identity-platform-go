# SaaS 身份平台 Go 端

SaaS 身份平台 Go 端（`saas-identity-platform-go`）——一句话系统定位（待补）。

本仓为《（书稿信息待补）》案例（待补）的可运行配套工程，是书稿代码块的 **source of truth**。

## 快速开始

```bash
# 前置：go 1.25+ 在 PATH（本机 scoop 装 1.27.3，GOPROXY=goproxy.cn）
go test ./...                          # 全量测试（零三方依赖，无 Key / 无 Docker / 无网可跑）
SERVER_PORT=5111 go run ./cmd/server   # 本地起服务（缺 SERVER_PORT 会 fail-fast 退出）
curl http://127.0.0.1:5111/health      # 探活：{"mode":"go","ok":true}
go build -o server.exe ./cmd/server    # 生产构建
```

门禁在 suite 根目录跑：`python scripts/gate.py -p saas-identity-platform-go`（L0-L5）。

## 功能特性

按特性分点描述核心功能（见 `docs/functions/function-tree.md`）。

已交付基础设施：`GET /health` 家族统一探活端点 + `SERVER_PORT` fail-fast 端口装配（REQ-2026-001，基础设施端点不进功能树）。

## 技术栈

| 技术 | 版本 |
| :--- | :--- |
| Go | 1.25+（go.mod 指令版；工具链 1.27.3） |
| 依赖 | 零三方依赖——四门（gofmt/vet/build/test）全走工具链自带命令 |

> 依赖版本与 `version-lock.json` 的 `version_lock` 一致，不引入 lock 外的库。

## 配套书籍及章节映射

| 章 | 主题 | 对应源文件 |
| :--- | :--- | :--- |
| （待补） | | |

## 快速链接

- [CLAUDE.md](CLAUDE.md) — 开发约定与编码规范
- [系统架构.md](docs/ARCHITECTURE.md) — 结构 / 边界 / 数据流 / 决策
- [功能规格.md](docs/functions/function-tree.md) — 功能名称、描述与验收标准
- [未来开发计划](PLAN.md) — 待办与迭代方向
- [更新日志](CHANGELOG.md) — 版本变更记录
