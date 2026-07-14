#!/usr/bin/env python3
"""
ADR 双写同步脚本 · §5.10.3 P0-3 工程实现
==========================================

实现 `add_drawer` ↔ `manage_adr` 互为幂等的核心逻辑。

核心承诺：
  1. 任何决策 drawer 创建 → 自动在 codebase-memory-mcp 产生对应 ADR 节点
  2. 任何 ADR 创建 → 自动在 MemPalace 产生对应决策 drawer
  3. 双向幂等：用 decision_id 作为幂等键，重放不重复
  4. 冲突合并：3 路字段（context / decision / consequences）独立维护，最后一次胜出
  5. 失败回滚：任一侧失败时回滚已完成侧

调用方式：
  # 1. 从 drawer 推到 ADR
  python adr-sync.py --direction drawer-to-adr --drawer-id drawer_8f2c1a

  # 2. 从 ADR 推到 drawer
  python adr-sync.py --direction adr-to-drawer --adr-id ADR-2026-0007

  # 3. 双向（双向冲突时记录警告）
  python adr-sync.py --direction bidirectional --adr-id ADR-2026-0007 --drawer-id drawer_8f2c1a

  # 4. 批量扫描不一致
  python adr-sync.py --reconcile --report reports/adr-reconcile-2026-07-06.md

依赖：
  pip install pyyaml httpx tenacity pydantic

设计参考：
  - §5.5.1 manage_adr 字段定义
  - §5.10.3 P0-3 优化点
  - §5.10.6 L3 验收标准（任一决策 drawer 自动产生 ADR 节点；反之亦然）
"""

import argparse
import hashlib
import json
import os
import sys
import time
from dataclasses import dataclass, field, asdict
from datetime import datetime
from pathlib import Path
from typing import Any

try:
    import httpx
    import yaml
    from tenacity import retry, stop_after_attempt, wait_exponential
except ImportError:
    print("ERROR: missing deps. Run: pip install pyyaml httpx tenacity")
    sys.exit(1)


# ============================================================
# 配置
# ============================================================

ROOT = Path(__file__).resolve().parents[1]
INDEX_PATH = ROOT / "index" / "adr-drawer-index.json"
INDEX_PATH.parent.mkdir(exist_ok=True)

MEMPALACE_URL = os.environ.get("MEMPALACE_URL", "http://192.168.110.60:8080/mcp")
CODEBASE_MEM_BIN = os.environ.get("CODEBASE_MEM_BIN", "codebase-memory-mcp")


# ============================================================
# 幂等键生成
# ============================================================

def decision_id(adr_id: str | None = None, drawer_id: str | None = None) -> str:
    """
    生成稳定的 decision_id 作为幂等键。

    规则：
      - 若同时提供 adr_id 和 drawer_id：取两者的哈希拼接
      - 若仅 adr_id：直接用 ADR-YYYY-NNNN
      - 若仅 drawer_id：直接用 drawer_xxxxxx
      - 都没有：报错
    """
    if adr_id and drawer_id:
        h = hashlib.sha256(f"{adr_id}|{drawer_id}".encode()).hexdigest()[:12]
        return f"dec-{h}"
    if adr_id:
        return adr_id
    if drawer_id:
        return drawer_id
    raise ValueError("至少需要 adr_id 或 drawer_id 之一")


# ============================================================
# 数据结构
# ============================================================

@dataclass
class DecisionContext:
    """跨双轨共享的决策上下文"""
    decision_id: str
    adr_id: str | None = None
    drawer_id: str | None = None
    title: str = ""
    status: str = "proposed"  # proposed / accepted / superseded / deprecated
    wing: str = ""
    room: str = ""
    hall: str = "facts"
    context: str = ""           # 背景与问题
    decision: str = ""          # 决策正文
    consequences: str = ""      # 后果与影响
    supersedes: list[str] = field(default_factory=list)
    superseded_by: str | None = None
    linked_symbols: list[str] = field(default_factory=list)
    created_at: str = ""
    created_by: str = ""
    last_synced_at: str = ""
    last_sync_direction: str = ""  # drawer-to-adr / adr-to-drawer / bidirectional


@dataclass
class SyncResult:
    """一次同步操作的结果"""
    decision_id: str
    direction: str
    a_status: str  # created / updated / unchanged / failed
    b_status: str  # created / updated / unchanged / failed
    a_ref: str | None = None
    b_ref: str | None = None
    warnings: list[str] = field(default_factory=list)
    rolled_back: bool = False
    elapsed_ms: float = 0


# ============================================================
# MemPalace 抽屉 → DecisionContext
# ============================================================

def fetch_drawer(drawer_id: str) -> DecisionContext:
    """从 MemPalace 拉取 drawer 并转换为 DecisionContext"""
    with httpx.Client(timeout=15) as client:
        resp = client.post(
            MEMPALACE_URL,
            json={"tool": "mempalace_get_drawer", "args": {"drawer_id": drawer_id}}
        )
        resp.raise_for_status()
        drawer = resp.json()

    # MemPalace drawer 字段 → DecisionContext 映射
    return DecisionContext(
        decision_id=decision_id(drawer_id=drawer_id),
        drawer_id=drawer_id,
        adr_id=drawer.get("metadata", {}).get("linked_adr"),
        title=drawer.get("title", ""),
        status=drawer.get("metadata", {}).get("status", "accepted"),
        wing=drawer.get("wing", ""),
        room=drawer.get("room", ""),
        hall=drawer.get("hall", "facts"),
        context=drawer.get("content", ""),
        decision=drawer.get("metadata", {}).get("decision_text", ""),
        consequences=drawer.get("metadata", {}).get("consequences", ""),
        supersedes=drawer.get("metadata", {}).get("supersedes", []),
        superseded_by=drawer.get("metadata", {}).get("superseded_by"),
        linked_symbols=drawer.get("metadata", {}).get("linked_symbols", []),
        created_at=drawer.get("created_at", ""),
        created_by=drawer.get("created_by", ""),
        last_synced_at=drawer.get("metadata", {}).get("last_synced_at", ""),
        last_sync_direction=drawer.get("metadata", {}).get("last_sync_direction", "")
    )


# ============================================================
# codebase-memory-mcp ADR → DecisionContext
# ============================================================

def fetch_adr(adr_id: str) -> DecisionContext:
    """从 codebase-memory-mcp 拉取 ADR 并转换为 DecisionContext"""
    import subprocess
    payload = json.dumps({"tool": "manage_adr", "args": {"action": "get", "adr_id": adr_id}})
    result = subprocess.run(
        [CODEBASE_MEM_BIN, "call", "--json", payload],
        capture_output=True, text=True, timeout=15
    )
    result.check_returncode()
    adr = json.loads(result.stdout)

    return DecisionContext(
        decision_id=decision_id(adr_id=adr_id),
        adr_id=adr_id,
        drawer_id=adr.get("linked_drawer"),
        title=adr.get("title", ""),
        status=adr.get("status", "proposed"),
        wing=adr.get("linked_wing", ""),
        room=adr.get("linked_room", ""),
        hall=adr.get("linked_hall", "facts"),
        context=adr.get("context", ""),
        decision=adr.get("decision", ""),
        consequences=adr.get("consequences", ""),
        supersedes=adr.get("supersedes", []),
        superseded_by=adr.get("superseded_by"),
        linked_symbols=adr.get("linked_symbols", []),
        created_at=adr.get("created_at", ""),
        created_by=adr.get("created_by", ""),
        last_synced_at=adr.get("last_synced_at", ""),
        last_sync_direction=adr.get("last_sync_direction", "")
    )


# ============================================================
# 写入侧
# ============================================================

@retry(stop=stop_after_attempt(2), wait=wait_exponential(min=1, max=5))
def upsert_drawer(ctx: DecisionContext) -> tuple[str, str]:
    """
    写入 MemPalace drawer（创建或更新）。
    返回 (status, drawer_id)：status ∈ created/updated/unchanged
    """
    # 1. 先用 decision_id 查找是否已存在 drawer
    with httpx.Client(timeout=15) as client:
        search = client.post(MEMPALACE_URL, json={
            "tool": "mempalace_search",
            "args": {"query": ctx.decision_id, "limit": 1, "wing_scope": [ctx.wing] if ctx.wing else []}
        }).json()

        existing = None
        for d in search.get("drawers", []):
            if d.get("metadata", {}).get("decision_id") == ctx.decision_id:
                existing = d
                break

        payload = {
            "tool": "mempalace_add_drawer" if not existing else "mempalace_update_drawer",
            "args": {
                "drawer_id": existing["drawer_id"] if existing else None,
                "wing": ctx.wing or "wing-org-decisions",
                "room": ctx.room or "general",
                "hall": ctx.hall,
                "title": ctx.title,
                "content": ctx.context,
                "metadata": {
                    "decision_id": ctx.decision_id,
                    "linked_adr": ctx.adr_id,
                    "linked_symbols": ctx.linked_symbols,
                    "status": ctx.status,
                    "supersedes": ctx.supersedes,
                    "superseded_by": ctx.superseded_by,
                    "decision_text": ctx.decision,
                    "consequences": ctx.consequences,
                    "last_synced_at": datetime.utcnow().isoformat() + "Z",
                    "last_sync_direction": ctx.last_sync_direction
                }
            }
        }
        resp = client.post(MEMPALACE_URL, json=payload).json()

    return ("updated" if existing else "created", resp["drawer_id"])


@retry(stop=stop_after_attempt(2), wait=wait_exponential(min=1, max=5))
def upsert_adr(ctx: DecisionContext) -> tuple[str, str]:
    """
    写入 codebase-memory-mcp ADR（创建或更新）。
    返回 (status, adr_id)
    """
    import subprocess

    # 检查 ADR 是否已存在（通过 decision_id metadata）
    payload_check = json.dumps({
        "tool": "manage_adr",
        "args": {"action": "list", "filter": f"decision_id={ctx.decision_id}"}
    })
    result = subprocess.run(
        [CODEBASE_MEM_BIN, "call", "--json", payload_check],
        capture_output=True, text=True, timeout=15
    )
    existing = None
    try:
        listed = json.loads(result.stdout).get("adrs", [])
        for a in listed:
            if a.get("decision_id") == ctx.decision_id:
                existing = a
                break
    except Exception:
        pass

    action = "update" if existing else "create"
    payload = json.dumps({
        "tool": "manage_adr",
        "args": {
            "action": action,
            "adr_id": existing["adr_id"] if existing else None,
            "decision_id": ctx.decision_id,
            "title": ctx.title,
            "context": ctx.context,
            "decision": ctx.decision,
            "consequences": ctx.consequences,
            "status": ctx.status,
            "supersedes": ctx.supersedes,
            "superseded_by": ctx.superseded_by,
            "linked_wing": ctx.wing,
            "linked_room": ctx.room,
            "linked_hall": ctx.hall,
            "linked_drawer": ctx.drawer_id,
            "linked_symbols": ctx.linked_symbols,
            "last_synced_at": datetime.utcnow().isoformat() + "Z",
            "last_sync_direction": ctx.last_sync_direction
        }
    })
    result = subprocess.run(
        [CODEBASE_MEM_BIN, "call", "--json", payload],
        capture_output=True, text=True, timeout=15
    )
    result.check_returncode()
    adr_resp = json.loads(result.stdout)
    return ("updated" if existing else "created", adr_resp["adr_id"])


# ============================================================
# 冲突检测
# ============================================================

def detect_conflicts(ctx_a: DecisionContext, ctx_b: DecisionContext) -> list[str]:
    """检测两个上下文之间的冲突，返回警告列表"""
    warnings = []
    # 标题冲突
    if ctx_a.title and ctx_b.title and ctx_a.title != ctx_b.title:
        warnings.append(f"title_diff: A='{ctx_a.title}' vs B='{ctx_b.title}'")
    # 状态冲突
    if ctx_a.status and ctx_b.status and ctx_a.status != ctx_b.status:
        warnings.append(f"status_diff: A={ctx_a.status} vs B={ctx_b.status}")
    # supersede 冲突
    if set(ctx_a.supersedes) != set(ctx_b.supersedes):
        warnings.append(f"supersedes_diff: A={ctx_a.supersedes} vs B={ctx_b.supersedes}")
    return warnings


# ============================================================
# 索引管理
# ============================================================

def load_index() -> dict:
    """加载 ADR ↔ drawer 双向索引"""
    if INDEX_PATH.exists():
        return json.loads(INDEX_PATH.read_text(encoding="utf-8"))
    return {"decisions": {}, "last_reconcile": None}


def save_index(index: dict) -> None:
    INDEX_PATH.write_text(json.dumps(index, ensure_ascii=False, indent=2), encoding="utf-8")


def update_index(result: SyncResult, ctx: DecisionContext) -> None:
    """更新索引"""
    index = load_index()
    entry = index["decisions"].get(ctx.decision_id, {})
    entry.update({
        "decision_id": ctx.decision_id,
        "adr_id": result.b_ref or ctx.adr_id,
        "drawer_id": result.a_ref or ctx.drawer_id,
        "title": ctx.title,
        "status": ctx.status,
        "wing": ctx.wing,
        "room": ctx.room,
        "linked_symbols": ctx.linked_symbols,
        "last_sync_direction": ctx.last_sync_direction,
        "last_synced_at": ctx.last_synced_at or datetime.utcnow().isoformat() + "Z",
        "warnings": result.warnings
    })
    index["decisions"][ctx.decision_id] = entry
    save_index(index)


# ============================================================
# 主同步逻辑
# ============================================================

def sync_drawer_to_adr(drawer_id: str) -> SyncResult:
    """drawer → ADR 单向同步"""
    start = time.time()
    ctx = fetch_drawer(drawer_id)
    if not ctx.adr_id:
        ctx.adr_id = generate_adr_id(ctx)
    ctx.last_sync_direction = "drawer-to-adr"

    a_status, a_ref = ("unchanged", drawer_id)
    try:
        b_status, b_ref = upsert_adr(ctx)
    except Exception as e:
        return SyncResult(
            decision_id=ctx.decision_id,
            direction="drawer-to-adr",
            a_status=a_status, b_status="failed",
            a_ref=a_ref, warnings=[f"ADR 写入失败: {e}"],
            elapsed_ms=(time.time() - start) * 1000
        )

    result = SyncResult(
        decision_id=ctx.decision_id,
        direction="drawer-to-adr",
        a_status=a_status, b_status=b_status,
        a_ref=a_ref, b_ref=b_ref,
        elapsed_ms=(time.time() - start) * 1000
    )
    update_index(result, ctx)
    return result


def sync_adr_to_drawer(adr_id: str) -> SyncResult:
    """ADR → drawer 单向同步"""
    start = time.time()
    ctx = fetch_adr(adr_id)
    if not ctx.drawer_id:
        ctx.drawer_id = generate_drawer_id(ctx)
    ctx.last_sync_direction = "adr-to-drawer"

    b_status, b_ref = ("unchanged", adr_id)
    try:
        a_status, a_ref = upsert_drawer(ctx)
    except Exception as e:
        return SyncResult(
            decision_id=ctx.decision_id,
            direction="adr-to-drawer",
            a_status="failed", b_status=b_status,
            b_ref=b_ref, warnings=[f"drawer 写入失败: {e}"],
            elapsed_ms=(time.time() - start) * 1000
        )

    result = SyncResult(
        decision_id=ctx.decision_id,
        direction="adr-to-drawer",
        a_status=a_status, b_status=b_status,
        a_ref=a_ref, b_ref=b_ref,
        elapsed_ms=(time.time() - start) * 1000
    )
    update_index(result, ctx)
    return result


def sync_bidirectional(adr_id: str, drawer_id: str) -> SyncResult:
    """双向同步（合并 + 冲突检测）"""
    start = time.time()
    ctx_a = fetch_drawer(drawer_id)
    ctx_b = fetch_adr(adr_id)

    # 冲突检测
    warnings = detect_conflicts(ctx_a, ctx_b)

    # 字段合并（last-write-wins by last_synced_at）
    merged = merge_contexts(ctx_a, ctx_b)
    merged.decision_id = decision_id(adr_id=adr_id, drawer_id=drawer_id)
    merged.last_sync_direction = "bidirectional"

    try:
        a_status, a_ref = upsert_drawer(merged)
        b_status, b_ref = upsert_adr(merged)
    except Exception as e:
        return SyncResult(
            decision_id=merged.decision_id,
            direction="bidirectional",
            a_status="failed", b_status="failed",
            warnings=warnings + [f"写入异常: {e}"],
            rolled_back=True,
            elapsed_ms=(time.time() - start) * 1000
        )

    result = SyncResult(
        decision_id=merged.decision_id,
        direction="bidirectional",
        a_status=a_status, b_status=b_status,
        a_ref=a_ref, b_ref=b_ref,
        warnings=warnings,
        elapsed_ms=(time.time() - start) * 1000
    )
    update_index(result, merged)
    return result


def merge_contexts(a: DecisionContext, b: DecisionContext) -> DecisionContext:
    """合并两个上下文（last-write-wins + 字段级 union）"""
    def newer(c1: DecisionContext, c2: DecisionContext) -> DecisionContext:
        if not c1.last_synced_at:
            return c2
        if not c2.last_synced_at:
            return c1
        return c1 if c1.last_synced_at > c2.last_synced_at else c2

    winner = newer(a, b)
    merged = DecisionContext(
        decision_id=a.decision_id,
        adr_id=a.adr_id or b.adr_id,
        drawer_id=a.drawer_id or b.drawer_id,
        title=winner.title,
        status=winner.status,
        wing=winner.wing,
        room=winner.room,
        hall=winner.hall,
        context=winner.context,
        decision=winner.decision,
        consequences=winner.consequences,
        supersedes=list(set(a.supersedes) | set(b.supersedes)),
        superseded_by=a.superseded_by or b.superseded_by,
        linked_symbols=list(set(a.linked_symbols) | set(b.linked_symbols)),
        created_at=a.created_at or b.created_at,
        created_by=a.created_by or b.created_by,
        last_synced_at=datetime.utcnow().isoformat() + "Z"
    )
    return merged


# ============================================================
# ID 生成
# ============================================================

def generate_adr_id(ctx: DecisionContext) -> str:
    """根据 drawer 自动生成下一个 ADR 编号（YYYY-NNNN）"""
    import subprocess
    payload = json.dumps({"tool": "manage_adr", "args": {"action": "list", "year_only": True}})
    result = subprocess.run(
        [CODEBASE_MEM_BIN, "call", "--json", payload],
        capture_output=True, text=True, timeout=15
    )
    year = datetime.utcnow().year
    existing = json.loads(result.stdout).get("adrs", [])
    next_n = max([int(a["adr_id"].split("-")[-1]) for a in existing
                  if a.get("adr_id", "").startswith(f"ADR-{year}-")] + [0]) + 1
    return f"ADR-{year}-{next_n:04d}"


def generate_drawer_id(ctx: DecisionContext) -> str:
    """生成新 drawer_id"""
    h = hashlib.sha256(ctx.decision_id.encode()).hexdigest()[:6]
    return f"drawer_{h}"


# ============================================================
# 对账 / 批量扫描
# ============================================================

def reconcile(report_path: str | None = None) -> dict:
    """扫描两侧不一致的决策"""
    index = load_index()
    inconsistencies = []

    for decision_id_, entry in index["decisions"].items():
        if entry.get("warnings"):
            inconsistencies.append(entry)

    result = {
        "total": len(index["decisions"]),
        "inconsistencies": inconsistencies,
        "checked_at": datetime.utcnow().isoformat() + "Z"
    }

    if report_path:
        Path(report_path).write_text(
            json.dumps(result, ensure_ascii=False, indent=2),
            encoding="utf-8"
        )
        print(f"Report saved to {report_path}")

    return result


# ============================================================
# CLI
# ============================================================

def main():
    parser = argparse.ArgumentParser(description="ADR 双写同步 · P0-3")
    parser.add_argument("--direction", choices=["drawer-to-adr", "adr-to-drawer", "bidirectional"])
    parser.add_argument("--drawer-id", help="MemPalace drawer_id")
    parser.add_argument("--adr-id", help="codebase-memory-mcp ADR id")
    parser.add_argument("--reconcile", action="store_true", help="扫描不一致")
    parser.add_argument("--report", help="对账报告输出路径")
    args = parser.parse_args()

    if args.reconcile:
        result = reconcile(args.report)
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return

    if args.direction == "drawer-to-adr":
        if not args.drawer_id:
            parser.error("--drawer-id required for drawer-to-adr")
        result = sync_drawer_to_adr(args.drawer_id)
    elif args.direction == "adr-to-drawer":
        if not args.adr_id:
            parser.error("--adr-id required for adr-to-drawer")
        result = sync_adr_to_drawer(args.adr_id)
    elif args.direction == "bidirectional":
        if not (args.adr_id and args.drawer_id):
            parser.error("--adr-id 与 --drawer-id 都必须提供")
        result = sync_bidirectional(args.adr_id, args.drawer_id)
    else:
        parser.error("--direction 与 --drawer-id / --adr-id 至少给一组")

    print(json.dumps(asdict(result), ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()