# REQ-2026-003 saas-go 全量业务切片

| 项 | 值 |
|---|---|
| 提出人 | zcqiand |
| 提出日期 | 2026-10-27 |
| 优先级 | P1 |
| 状态 | 已交付（2026-10-27，v0.2.0-20261027） |
| 关联 ADR | — |

## 1. 需求描述

> 用户原话：「saas+lab go版本，继续所有需求」

理解：saas-go 仓从骨架态推到全量业务面，与 rails/laravel/ktor 同门对齐——shared 契约 47 op / 11 tag 全实现、全量 route-parity、DB-First entity 生成链、双幂等门接管、gate 全绿。codegen 按人裁（2026-10-26）接 oapi-codegen；server 模式与实体层方案在本 REQ 接线时定（std-http-server + GORM，已回填 profiles/codegen.md go 行）。lab-go 复刻后置独立批。

### 澄清记录

| 疑问 | 澄清结论 | 澄清人 | 日期 |
|---|---|---|---|
| 「所有需求」边界 | = laravel 批同款全量面（47 op parity），不含 lab-go（独立批）与 contract-test go 注册（本批 Task 11 含） | Claude（inline 执行） | 2026-10-27 |

## 2. 验收标准

| 编号 | 场景（给定） | 操作（当） | 面预期（则） |
|---|---|---|---|
| AC-1 | shared openapi 47 op | 跑 go 仓 route-parity 契约门 | ✅ 47 条路由全挂接、无契约外路由（allowlist 仅 GET /health） |
| AC-2 | 本机 PG saas_dev | 跑 strict 幂等门 | ✅ 双脚本复放字节幂等，三生成区与 HEAD 一致（check_codegen_idempotent.py --strict exit 0） |
| AC-3 | saas_test 种子在位 | 跑 go test ./... | ✅ Feature 链全绿且事务回滚不污染（suite gate L4 exit 0） |
| AC-4 | 本机起 saas-go :5111 连 saas_dev | login→me→menus 冒烟 | ✅ live 探针全 200；CT live 2-way（springboot,go）201/201 收敛 |

## 3. 任务拆解

见 suite 仓 `docs/superpowers/plans/2026-10-27-go-saas-full-slice.md`（Task 1-11）。

## 4. 功能影响

| 功能 ID | 功能名称 | 影响类型 | 说明 | 关联任务 |
|---|---|---|---|---|
| M00/M01/M04 全部活跃 I 项（58 项） | 租户/用户/应用管理 | 新增 | saas-go 侧首交付，I 级随实现逐段 mirror 登记 | Task 2-10 |
