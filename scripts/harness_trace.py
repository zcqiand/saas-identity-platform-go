#!/usr/bin/env python3
"""Go trace 适配器：`go test -json` 事件流 + `// fn:` 注解 → suite 契约的 .state/trace.json。

约定：
- 测试函数上方注释块或行尾写 `// fn: M01.F01.I01 M01.F01.I02` 挂功能 ID。
  ID 语法与 suite FUNCTION_ID_RE 同源（M\\d\\d[.F\\d\\d[.I\\d\\d]]）。
- skip 的测试在源头抹空 fns（inert=true），suite 会复核。「禁止给 skip 测试挂
  功能 ID」是仓规，这里兜底防假绿。
- 仓内没有任何 *_test.go → 产空 trace 正常退出，**不调 go**（suite 契约：L0/L5
  初始化当场绿，不依赖语言工具链）；有测试文件却解析不出任何事件 = 适配器坏了
  或编译失败，exit 2——安静失败会得到一张假的对齐表，这是 suite 真实踩过的坑
  （vitest reporter 变更）。
"""

from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / ".state" / "trace.json"
TIMEOUT = 3600

FN_ID_RE = re.compile(r"\bM\d{2}(?:\.F\d{2}(?:\.I\d{2})?)?\b")
FUNC_RE = re.compile(r"^func\s+(Test\w+)\s*\(")


def collect_fn_map() -> dict[str, set[str]]:
    """扫 *_test.go，取每个 Test 函数的 `// fn:` 注解（上方注释块或行尾）。"""
    fn_map: dict[str, set[str]] = {}
    for path in sorted(ROOT.rglob("*_test.go")):
        parts = set(path.parts)
        if parts & {"_work", "vendor", "node_modules"}:
            continue
        pending: list[str] = []
        for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
            stripped = line.strip()
            if stripped.startswith("//"):
                comment = stripped.lstrip("/").strip()
                if comment.startswith("fn:"):
                    pending.extend(FN_ID_RE.findall(comment.removeprefix("fn:")))
                continue
            if not stripped:
                continue  # 空行不打断注解块
            m = FUNC_RE.match(line)
            if m:
                ids = set(pending) | set(FN_ID_RE.findall(line))
                if ids:
                    fn_map.setdefault(m.group(1), set()).update(ids)
            pending = []
    return fn_map


def run_go_test_json() -> tuple[list[dict], str]:
    """跑 go test -json，返回（事件列表，未解析行+stderr 拼的诊断串）。"""
    proc = subprocess.run(
        ["go", "test", "-json", "./..."],
        cwd=ROOT,
        capture_output=True,
        timeout=TIMEOUT,
    )
    events: list[dict] = []
    noise: list[str] = []
    for raw in proc.stdout.decode("utf-8", errors="replace").splitlines():
        line = raw.strip()
        if not line:
            continue
        try:
            events.append(json.loads(line))
        except json.JSONDecodeError:
            noise.append(line)
    noise.extend(proc.stderr.decode("utf-8", errors="replace").splitlines())
    return events, "\n".join(noise)


def main() -> int:
    fn_map = collect_fn_map()
    has_test_files = any(
        not set(p.parts) & {"_work", "vendor", "node_modules"}
        for p in ROOT.rglob("*_test.go")
    )

    final: dict[str, str] = {}
    diag = ""
    if has_test_files:  # 零测试的仓不调 go：L5 初始化必须当场绿，不依赖工具链
        try:
            events, diag = run_go_test_json()
        except FileNotFoundError:
            print(
                "go 不在 PATH，但仓内已有 *_test.go：trace 适配器无法运行。先装 Go 工具链。",
                file=sys.stderr,
            )
            return 2
        except subprocess.TimeoutExpired:
            print(f"go test -json 超时（>{TIMEOUT}s）", file=sys.stderr)
            return 2

        # 每个测试取最后一个终态 action（pass/fail/skip 会成对重复：父测试/子测试）。
        for ev in events:
            name, action = ev.get("Test"), ev.get("Action")
            if name and action in ("pass", "fail", "skip"):
                final[name] = action

        if not final:
            print(
                "go test -json 没解析出任何测试事件（适配器坏了或编译失败），"
                f"诊断输出：\n{diag[:2000]}",
                file=sys.stderr,
            )
            return 2

    entries: list[dict] = []
    for name in sorted(final):
        inert = final[name] == "skip"
        top = name.split("/")[0]  # 子测试 TestX/sub 沿用父测试的挂 ID
        ids = sorted(fn_map.get(name) or fn_map.get(top) or set())
        entries.append({"test": name, "fns": [] if inert else ids, "inert": inert})

    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(
        json.dumps({"schema": 1, "tests": entries}, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    n_ids = sum(len(e["fns"]) for e in entries)
    n_inert = sum(1 for e in entries if e["inert"])
    print(f"harness_trace: {len(entries)} tests（挂 ID {n_ids} 个，inert {n_inert}）-> {OUT}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
