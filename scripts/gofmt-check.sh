#!/usr/bin/env bash
# L1 格式门：gofmt check-only。
# 走 bash 脚本的原因：gofmt 不在 suite EXEC_ALLOWLIST（go 在），Go 又没有
# check-only 的 fmt 子命令（go fmt 是 -w 直改）。bash + scripts/<named>.sh
# 是 EXEC_ALLOWLIST 注释里写明的人审约定。
set -u
cd "$(dirname "$0")/.."

if ! command -v gofmt >/dev/null 2>&1; then
  echo "gofmt 不在 PATH：先确认 go 工具链已装且 GOROOT/bin 在 PATH（go version 自检）" >&2
  exit 2
fi

out="$(gofmt -l . 2>/dev/null)"
if [ -n "$out" ]; then
  echo "以下文件未过 gofmt（cd 仓根跑 gofmt -w . 修复）：" >&2
  echo "$out" >&2
  exit 1
fi
echo "gofmt: all formatted"
