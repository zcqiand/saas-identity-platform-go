# CLAUDE.md — SaaS 身份平台 Go 端

> 书稿配套仓 + harness 门禁仓双身份。入口，不是手册。L0 门强制上限 60 行。
> 本仓为《（书稿信息待补）》案例（待补）（SaaS 身份平台 Go 端）的可运行配套工程，是书稿代码块的 **source of truth**。

## 1. 项目定位

SaaS 身份平台 Go 端（`saas-identity-platform-go`，技术栈 `go`）。一句话定位见 README.md。

## 2. 铁律

- **TDD**：每个模块先写失败测试 → 跑确认失败 → 实现 → 跑确认绿 → commit
- **版本钉死**：依赖与 `version-lock.json` 的 `version_lock` 一致；不引入 lock 外的库
- **tag 即放行**：全量回归绿后打 `v<MAJOR>.<MINOR>.<PATCH>-<YYYYMMDD>`（如 `v0.3.54-20260826`）
- **mock-friendly**：安装 + 测试必须在无 Key、无 Docker、无网下全绿
- **功能清单是锚点**：改 `docs/functions/function-tree.md` 走 `/tree-change` 提案，由人批准；
  改功能与改功能清单必须同一个 commit；废弃只改状态，编号永不复用；禁止给 skip 的测试挂功能 ID

- 禁止在 HTTP handler 内写业务逻辑（handler 只做装配与状态码映射，业务进 service）
- 禁止吞错：err 必须处理或带上下文 wrap 后返回，禁止 _ = err 丢错
- 禁止 fmt.Println 调试残留进 main 源码（用 log/slog）
- 禁止 panic 代替 error 返回（main 装配期 fail-fast panic 除外）
- 禁止 env 键写默认值兜底（fail-fast，suite-hard-rules §1）

## 3. 技术栈与版本（钉死于 version-lock.json）

Go 1.25+ / gofmt / go vet / go test（四门全走工具链自带命令，零第三方依赖，无网全绿）。L1 经 bash scripts/gofmt-check.sh——gofmt 不在 EXEC_ALLOWLIST 而 go 在，且 Go 无 check-only 的 fmt 子命令；go 在 PATH 时 gofmt（GOROOT/bin 同目录）必然在。trace_cmd 走 python scripts/harness_trace.py：`go test -json` 事件流 + 测试函数 `// fn:` 注解 → .state/trace.json，skip 测试在源头抹空 fns；仓内无 *_test.go 时产空 trace 正常退出（新建仓 L5 不假红）。明细见 `version-lock.json` 与 README.md 技术栈表。

门禁命令见 `.harness/stack.json`。**不要改它来让门变松。**

## 4. 验收

- 在 **suite 根目录** 跑 `python scripts/gate.py -p saas-identity-platform-go`；exit 0 才算完成
- 本地命令见 README.md「快速开始」

## 5. 指向别处

- 功能清单（唯一锚点） → `docs/functions/function-tree.md`
- 需求 → 任务 → 功能影响 → `docs/requirements/`
- 流程/设计 与功能对齐 → `docs/design/`（人评审，机器只查引用）
- 决策背景 → `docs/adr/`；编码细则 → `docs/conventions/`（不进主上下文）
- 待办与迭代方向 → `PLAN.md`；版本变更 → `CHANGELOG.md`

## 6. 工作循环

0. **开工前分诊**：先过 `using-skills`，把激活 skill 的清单落成 todo。
   顺序：规格(brainstorming)→计划(writing-plans)→测试先红(red-first)→实现(executing-plans)
1. 读 `.state/session.json` 恢复上下文
2. 最小改动
3. 跑 `python scripts/gate.py -p saas-identity-platform-go`；exit 1 回到第 2 步；exit 2 停下问人
4. `/handoff` 更新 `.state/session.json`
