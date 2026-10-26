# REQ-2026-001 Go 端健康检查 + 端口装配最小切片（骨架可跑）

| 项 | 值 |
|---|---|
| 提出人 | zcqiand |
| 提出日期 | 2026-10-26 |
| 优先级 | P1 |
| 状态 | 已验收（2026-10-26：live 探针 GET /health → `{"mode":"go","ok":true}`、POST 405、缺 env exit 1、L0-L5 全绿；自动化验收，人裁复核随检视） |
| 关联 ADR | —（基础设施端点不进契约面，ktor REQ-2026-001/003 先例） |

## 1. 需求描述

用户原话：「go 双仓：等第一个需求 → /req 立档」，随后在澄清问答中选定**健康检查 + 端口装配最小切片**、**saas-go 先行，lab-go 复刻**。

理解：go 双仓目前只有锚点包骨架，没有任何可运行进程。本需求让 saas-go 骨架真正可跑——HTTP 服务从 `SERVER_PORT` env 装配监听地址（fail-fast，无默认值兜底），暴露家族统一探活端点 `GET /health`，并补齐最小工具测试。这是 Go 栈全流程（red-first → 门禁 → trace → 台账）的首跑验证切片。

### 澄清记录

| 疑问 | 澄清结论 | 澄清人 | 日期 |
|---|---|---|---|
| 第一个需求做什么 | 健康检查 + 端口装配最小切片（选项确认） | zcqiand | 2026-10-26 |
| 双仓落地顺序 | saas-go 先行，lab-go 原样复刻（ktor/laravel/miniprogram 先例同款） | zcqiand | 2026-10-26 |
| /health 是否挂功能 ID / 动功能树 | 不挂不动——基础设施端点影响功能数 0（ktor REQ-2026-001 deploy 链、REQ-2026-003 Swagger 台账先例，trace 中 fns 留空） | 依 ktor 台账先例（如不认可，退回本表重议） | 2026-10-26 |

## 2. 验收标准

| 编号 | 场景（给定） | 操作（当） | 预期（则） |
|---|---|---|---|
| AC-1 | .env 含 `SERVER_PORT=5111` | 启动服务 | 进程监听 `127.0.0.1:5111`；端口值只来自 env，代码无默认值兜底（suite-hard-rules §1） |
| AC-2 | 服务已启动 | `GET /health` | 200 且 body 为 `{"ok":true,"mode":"go"}`（家族探活契约形状，ktor `{"ok":true,"mode":"ktor"}` 同款） |
| AC-3 | .env 缺 `SERVER_PORT` | 启动服务 | fail-fast 退出非零，错误信息指明缺失键名 |
| AC-4 | 仓内实现与测试就位 | `python scripts/gate.py -p saas-identity-platform-go` | L0-L5 全绿；trace.json 由 harness_trace.py 产出，/health 相关测试 fns 为空数组 |
| AC-5 | `internal/server` 装配包落地 | `go vet ./...` | `./...` 非空通过，`internal/harness` 锚点包删除 |

## 3. 任务拆解

| 任务 ID | 任务描述 | 类型 | 负责人 | 预估 | 状态 |
|---|---|---|---|---|---|
| T-1 | `internal/config`：读 `SERVER_PORT`，缺键/非数字 fail-fast（禁默认值兜底） | 开发 | Claude | 0.5h | 已完成 |
| T-2 | `internal/server`：HTTP 装配 + `GET /health` → `{"ok":true,"mode":"go"}`；`cmd/server/main.go` 组合根 | 开发 | Claude | 0.5h | 已完成（实机抓真 bug 一枚：New() 原未接 Addr，零值落 :80 bind 拒——已修为 New(addr)） |
| T-3 | 工具测试：httptest 探 /health 形状（AC-2）+ config 缺键 fail-fast（AC-3）；不挂 fn ID | 测试 | Claude | 0.5h | 已完成（red-first：编译红证据后实现） |
| T-4 | 删 `internal/harness` 锚点包（AC-5） | 开发 | Claude | 0.1h | 已完成 |
| T-5 | 门禁 L0-L5 全绿收口 + 提交推送 + gitlink bump | 收口 | Claude | 0.5h | 已完成 |
| T-6 | lab-management-system-go 原样复刻（端口 5211、mode 值 `go` 同款）+ 独立 REQ 立档 | 复刻 | Claude | 0.5h | 进行中 |

## 4. 功能影响（需求与功能对齐的唯一位置）

| 功能 ID | 功能名称 | 影响类型 | 说明 | 关联任务 |
|---|---|---|---|---|
| （无） | — | — | 基础设施端点不进契约面/功能树，影响功能数 0（ktor REQ-2026-001/003 台账先例） | — |

## 5. 流程影响

无（docs/design/flow-function-map.md 尚未建立，本需求为纯工程设施）。

## 6. 风险与回滚

| 风险 | 影响面 | 缓解 | 回滚方式 |
|---|---|---|---|
| 5111 端口被占（本机其他服务/后续家族成员） | 本地启动失败 | 绑定失败即退出非零并报错，不重试不换端口 | 停占进程或改 .env 本地值（家族槽位不变） |
| harness_trace.py 首次面对真测试事件流（此前空仓路径） | L5 红 | 适配器已处理 `go test -json` 事件解析；红了按 ContractError 流程停下排查 | revert 提交 |
