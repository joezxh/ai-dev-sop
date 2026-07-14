#pylint: disable=duplicate-code,import-error
"""
ADR 双写同步引擎 · §5.10.3 P0-3 · §5.7.7 控制台实时视图
====================================================

从 adr-sync.py 重构而来，核心逻辑独立为可导入模块。
被 gateway/server.py 调用，也可独立使用。

幂等设计：
  - 幂等键 = decision_id
  - drawer ↔ ADR 双向互为幂等
  - 冲突时 last-write-wins + 字段级 union

ADR 双写同步状态机：
  ┌──────────┐  sync_drawer_to_adr()  ┌──────────┐
  │  drawer  │ ──────────────────────► │    ADR   │
  │  已存在  │                         │  已存在  │
  └──────────┘                         └──────────┘
       ▲                                    │
       │           sync_adr_to_drawer()      │
       └────────────────────────────────────┘
                    bidirectional() 时
                    触发冲突检测 + 合并
"""

from __future__ import annotations

import hashlib
import json
import os
import time
from dataclasses import dataclass, field
from datetime import datetime, timezone
from enum import Enum
from pathlib import Path
from typing import Any

import httpx

try:
    import yaml
    from tenacity import retry, stop_after_attempt, wait_exponential
except ImportError:  # pragma: no cover
    yaml = None  # type: ignore
    retry = lambda *a, **kw: lambda f: f  # type: ignore
    stop_after_attempt = wait_exponential = None  # type: ignore


# ============================================================
# 配置
# ============================================================

MEMPALACE_URL: str = os.environ.get(
    "MEMPALACE_URL", "http://192.168.110.60:8080/mcp"
)
CODEBASE_MEM_BIN: str = os.environ.get(
    "CODEBASE_MEM_BIN", "codebase-memory-mcp"
)
INDEX_PATH: Path = Path(
    os.environ.get("SYNC_INDEX_PATH", str(Path(__file__).parent / "index" / "adr-drawer-index.json"))
)
INDEX_PATH.parent.mkdir(parents=True, exist_ok=True)


# ============================================================
# 枚举
# ============================================================

class SyncDirection(str, Enum):
    DRAWER_TO_ADR = "drawer-to-adr"
    ADR_TO_DRAWER = "adr-to-drawer"
    BIDIRECTIONAL = "bidirectional"


class SyncSideStatus(str, Enum):
    CREATED = "created"
    UPDATED = "updated"
    UNCHANGED = "unchanged"
    FAILED = "failed"


class ADRStatus(str, Enum):
    PROPOSED = "proposed"
    ACCEPTED = "accepted"
    SUPERSEDED = "superseded"
    DEPRECATED = "deprecated"


# ============================================================
# 数据结构
# ============================================================

@dataclass
class DecisionContext:
    """
    跨 MemPalace drawer 与 codebase-mem ADR 共享的决策上下文。
    所有字段均已规范化，双侧字段一一对应（见字段映射表）。
    """
    decision_id: str
    # --- 共享字段 ---
    title: str = ""
    status: ADRStatus = ADRStatus.PROPOSED
    context: str = ""          # 背景与问题陈述
    decision: str = ""         # 决策正文
    consequences: str = ""     # 后果与影响
    supersedes: list[str] = field(default_factory=list)
    superseded_by: str | None = None
    linked_symbols: list[str] = field(default_factory=list)
    created_at: str = ""
    created_by: str = ""
    # --- MemPalace 侧 ---
    drawer_id: str | None = None
    wing: str = "wing-org-decisions"
    room: str = "general"
    hall: str = "facts"
    # --- codebase-mem 侧 ---
    adr_id: str | None = None
    linked_wing: str = ""      # 同 wing
    linked_room: str = ""      # 同 room
    linked_hall: str = ""      # 同 hall
    # --- 同步状态 ---
    last_synced_at: str = ""
    last_sync_direction: SyncDirection = SyncDirection.DRAWER_TO_ADR

    def to_drawer_payload(self) -> dict[str, Any]:
        """→ MemPalace drawer 写入 payload"""
        return {
            "tool": "mempalace_add_drawer" if not self.drawer_id else "mempalace_update_drawer",
            "args": {
                "drawer_id": self.drawer_id,
                "wing": self.wing,
                "room": self.room,
                "hall": self.hall,
                "title": self.title,
                "content": self.context,
                "metadata": {
                    "decision_id": self.decision_id,
                    "linked_adr": self.adr_id,
                    "linked_symbols": self.linked_symbols,
                    "status": self.status.value,
                    "supersedes": self.supersedes,
                    "superseded_by": self.superseded_by,
                    "decision_text": self.decision,
                    "consequences": self.consequences,
                    "last_synced_at": self.last_synced_at,
                    "last_sync_direction": self.last_sync_direction.value,
                },
            },
        }

    def to_adr_payload(self, action: str = "create") -> dict[str, Any]:
        """→ codebase-mem ADR 写入 payload"""
        return {
            "tool": "manage_adr",
            "args": {
                "action": action,
                "adr_id": self.adr_id,
                "decision_id": self.decision_id,
                "title": self.title,
                "context": self.context,
                "decision": self.decision,
                "consequences": self.consequences,
                "status": self.status.value,
                "supersedes": self.supersedes,
                "superseded_by": self.superseded_by,
                "linked_wing": self.linked_wing or self.wing,
                "linked_room": self.linked_room or self.room,
                "linked_hall": self.linked_hall or self.hall,
                "linked_drawer": self.drawer_id,
                "linked_symbols": self.linked_symbols,
                "last_synced_at": self.last_synced_at,
                "last_sync_direction": self.last_sync_direction.value,
            },
        }


@dataclass
class SyncResult:
    """一次同步操作的结果"""
    decision_id: str
    direction: SyncDirection
    a_status: SyncSideStatus  # MemPalace
    b_status: SyncSideStatus  # codebase-mem
    a_ref: str | None = None   # drawer_id
    b_ref: str | None = None   # adr_id
    warnings: list[str] = field(default_factory=list)
    rolled_back: bool = False
    elapsed_ms: float = 0.0
    errors: list[str] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        return {
            "decision_id": self.decision_id,
            "direction": self.direction.value,
            "a_status": self.a_status.value,
            "b_status": self.b_status.value,
            "a_ref": self.a_ref,
            "b_ref": self.b_ref,
            "warnings": self.warnings,
            "rolled_back": self.rolled_back,
            "elapsed_ms": round(self.elapsed_ms, 2),
            "errors": self.errors,
        }


# ============================================================
# 幂等键生成
# ============================================================

def make_decision_id(
    adr_id: str | None = None,
    drawer_id: str | None = None,
) -> str:
    """
    生成稳定的 decision_id 作为幂等键。

    规则：
      1. 同时有 adr_id + drawer_id → SHA-256(|ADR_ID|drawer_id)[:12]
      2. 仅 adr_id → 直接用
      3. 仅 drawer_id → 直接用
      4. 两者都没有 → ValueError
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
# MemPalace 拉取
# ============================================================

def fetch_drawer(drawer_id: str) -> DecisionContext | None:
    """从 MemPalace 拉取 drawer，转为 DecisionContext"""
    try:
        with httpx.Client(timeout=15) as client:
            resp = client.post(MEMPALACE_URL, json={
                "tool": "mempalace_get_drawer",
                "args": {"drawer_id": drawer_id}
            })
            if resp.status_code == 404:
                return None
            resp.raise_for_status()
            drawer = resp.json()
    except Exception:
        return None

    meta = drawer.get("metadata", {})
    return DecisionContext(
        decision_id=meta.get("decision_id", drawer_id),
        drawer_id=drawer_id,
        adr_id=meta.get("linked_adr"),
        title=drawer.get("title", ""),
        status=ADRStatus(meta.get("status", "accepted")),
        wing=drawer.get("wing", "wing-org-decisions"),
        room=drawer.get("room", "general"),
        hall=drawer.get("hall", "facts"),
        context=drawer.get("content", ""),
        decision=meta.get("decision_text", ""),
        consequences=meta.get("consequences", ""),
        supersedes=meta.get("supersedes", []),
        superseded_by=meta.get("superseded_by"),
        linked_symbols=meta.get("linked_symbols", []),
        created_at=drawer.get("created_at", ""),
        created_by=drawer.get("created_by", ""),
        last_synced_at=meta.get("last_synced_at", ""),
        last_sync_direction=SyncDirection(
            meta.get("last_sync_direction", "drawer-to-adr")
        ),
    )


def find_drawer_by_decision_id(decision_id: str, wing: str = "") -> str | None:
    """用 decision_id 查 MemPalace drawer_id（幂等查找）"""
    try:
        with httpx.Client(timeout=10) as client:
            args: dict[str, Any] = {"query": decision_id, "limit": 5}
            if wing:
                args["wing_scope"] = [wing]
            resp = client.post(MEMPALACE_URL, json={
                "tool": "mempalace_search",
                "args": args
            }).json()
        for d in resp.get("drawers", []):
            if d.get("metadata", {}).get("decision_id") == decision_id:
                return d["drawer_id"]
    except Exception:
        pass
    return None


# ============================================================
# codebase-memory-mcp 拉取
# ============================================================

import subprocess

def fetch_adr(adr_id: str) -> DecisionContext | None:
    """从 codebase-memory-mcp 拉取 ADR，转为 DecisionContext"""
    payload = json.dumps({
        "tool": "manage_adr",
        "args": {"action": "get", "adr_id": adr_id}
    })
    try:
        result = subprocess.run(
            [CODEBASE_MEM_BIN, "call", "--json", payload],
            capture_output=True, text=True, timeout=15
        )
        if result.returncode != 0:
            return None
        adr = json.loads(result.stdout)
    except Exception:
        return None

    meta = adr  # ADR 字段直接暴露
    return DecisionContext(
        decision_id=meta.get("decision_id", adr_id),
        adr_id=adr_id,
        drawer_id=meta.get("linked_drawer"),
        title=meta.get("title", ""),
        status=ADRStatus(meta.get("status", "proposed")),
        wing=meta.get("linked_wing", "wing-org-decisions"),
        room=meta.get("linked_room", "general"),
        hall=meta.get("linked_hall", "facts"),
        context=meta.get("context", ""),
        decision=meta.get("decision", ""),
        consequences=meta.get("consequences", ""),
        supersedes=meta.get("supersedes", []),
        superseded_by=meta.get("superseded_by"),
        linked_symbols=meta.get("linked_symbols", []),
        created_at=meta.get("created_at", ""),
        created_by=meta.get("created_by", ""),
        last_synced_at=meta.get("last_synced_at", ""),
        last_sync_direction=SyncDirection(
            meta.get("last_sync_direction", "adr-to-drawer")
        ),
    )


def find_adr_by_decision_id(decision_id: str) -> str | None:
    """用 decision_id 查 codebase-mem ADR（幂等查找）"""
    payload = json.dumps({
        "tool": "manage_adr",
        "args": {"action": "list", "filter": f"decision_id={decision_id}"}
    })
    try:
        result = subprocess.run(
            [CODEBASE_MEM_BIN, "call", "--json", payload],
            capture_output=True, text=True, timeout=10
        )
        if result.returncode != 0:
            return None
        for a in json.loads(result.stdout).get("adrs", []):
            if a.get("decision_id") == decision_id:
                return a["adr_id"]
    except Exception:
        pass
    return None


# ============================================================
# 写入侧
# ============================================================

def upsert_drawer(ctx: DecisionContext) -> tuple[SyncSideStatus, str | None]:
    """
    写入 MemPalace drawer（幂等创建或更新）。
    返回 (status, drawer_id)
    """
    # 幂等查找：是否已存在
    existing_id = find_drawer_by_decision_id(ctx.decision_id, ctx.wing)
    ctx.drawer_id = existing_id
    is_update = bool(existing_id)

    payload = ctx.to_drawer_payload()
    if is_update:
        payload["args"]["drawer_id"] = existing_id

    try:
        with httpx.Client(timeout=15) as client:
            resp = client.post(MEMPALACE_URL, json=payload)
            resp.raise_for_status()
            result = resp.json()
        return (
            SyncSideStatus.UPDATED if is_update else SyncSideStatus.CREATED,
            result.get("drawer_id", existing_id),
        )
    except Exception as e:
        return SyncSideStatus.FAILED, str(e)


def upsert_adr(ctx: DecisionContext) -> tuple[SyncSideStatus, str | None]:
    """
    写入 codebase-memory-mcp ADR（幂等创建或更新）。
    返回 (status, adr_id)
    """
    existing_id = find_adr_by_decision_id(ctx.decision_id)
    ctx.adr_id = existing_id
    is_update = bool(existing_id)

    payload = ctx.to_adr_payload(action="update" if is_update else "create")

    try:
        result = subprocess.run(
            [CODEBASE_MEM_BIN, "call", "--json", json.dumps(payload)],
            capture_output=True, text=True, timeout=15
        )
        if result.returncode != 0:
            return SyncSideStatus.FAILED, result.stderr or "non-zero exit"
        adr_resp = json.loads(result.stdout)
        return (
            SyncSideStatus.UPDATED if is_update else SyncSideStatus.CREATED,
            adr_resp.get("adr_id", existing_id),
        )
    except Exception as e:
        return SyncSideStatus.FAILED, str(e)


# ============================================================
# 冲突检测
# ============================================================

def detect_conflicts(a: DecisionContext, b: DecisionContext) -> list[str]:
    """检测两侧冲突，返回警告列表"""
    warnings: list[str] = []
    if a.title and b.title and a.title != b.title:
        warnings.append(
            f"title_diff: drawer='{a.title[:40]}' vs ADR='{b.title[:40]}'"
        )
    if a.status and b.status and a.status != b.status:
        warnings.append(f"status_diff: drawer={a.status.value} vs ADR={b.status.value}")
    if set(a.supersedes) != set(b.supersedes):
        warnings.append(
            f"supersedes_diff: drawer={a.supersedes} vs ADR={b.supersedes}"
        )
    return warnings


# ============================================================
# 上下文合并（last-write-wins + union）
# ============================================================

def merge_contexts(a: DecisionContext, b: DecisionContext) -> DecisionContext:
    """
    合并两个 DecisionContext。
    - 字符串字段：last_synced_at 更新者胜出
    - 数组字段：union 并集
    - 单值引用：非空优先
    """
    def newer(x: DecisionContext, y: DecisionContext) -> DecisionContext:
        if not x.last_synced_at:
            return y
        if not y.last_synced_at:
            return x
        return x if x.last_synced_at >= y.last_synced_at else y

    winner = newer(a, b)

    return DecisionContext(
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
        last_synced_at=datetime.now(timezone.utc).isoformat(),
    )


# ============================================================
# 主同步逻辑
# ============================================================

def sync_drawer_to_adr(drawer_id: str) -> SyncResult:
    """drawer → ADR 单向幂等同步"""
    start = time.perf_counter()
    ts = datetime.now(timezone.utc).isoformat()
    errors: list[str] = []

    ctx = fetch_drawer(drawer_id)
    if not ctx:
        return SyncResult(
            decision_id=drawer_id,
            direction=SyncDirection.DRAWER_TO_ADR,
            a_status=SyncSideStatus.FAILED,
            b_status=SyncSideStatus.FAILED,
            errors=[f"drawer not found: {drawer_id}"],
            elapsed_ms=(time.perf_counter() - start) * 1000,
        )

    # 幂等键：仅 drawer_id → 用 SHA 哈希生成 decision_id
    if not ctx.decision_id or ctx.decision_id == drawer_id:
        ctx.decision_id = make_decision_id(drawer_id=drawer_id)

    ctx.last_sync_direction = SyncDirection.DRAWER_TO_ADR
    ctx.last_synced_at = ts

    # ADR 是否已存在
    existing_adr = find_adr_by_decision_id(ctx.decision_id)
    ctx.adr_id = existing_adr
    b_action = "update" if existing_adr else "create"

    # 写入 ADR
    b_status, b_ref = upsert_adr(ctx)

    result = SyncResult(
        decision_id=ctx.decision_id,
        direction=SyncDirection.DRAWER_TO_ADR,
        a_status=SyncSideStatus.UNCHANGED,  # 源侧不变
        b_status=b_status,
        a_ref=drawer_id,
        b_ref=ctx.adr_id or b_ref,
        elapsed_ms=(time.perf_counter() - start) * 1000,
    )
    if b_status == SyncSideStatus.FAILED:
        result.errors.append(f"ADR upsert failed: {b_ref}")

    update_index_record(result, ctx)
    return result


def sync_adr_to_drawer(adr_id: str) -> SyncResult:
    """ADR → drawer 单向幂等同步"""
    start = time.perf_counter()
    ts = datetime.now(timezone.utc).isoformat()

    ctx = fetch_adr(adr_id)
    if not ctx:
        return SyncResult(
            decision_id=adr_id,
            direction=SyncDirection.ADR_TO_DRAWER,
            a_status=SyncSideStatus.FAILED,
            b_status=SyncSideStatus.FAILED,
            errors=[f"ADR not found: {adr_id}"],
            elapsed_ms=(time.perf_counter() - start) * 1000,
        )

    ctx.last_sync_direction = SyncDirection.ADR_TO_DRAWER
    ctx.last_synced_at = ts

    a_status, a_ref = upsert_drawer(ctx)

    result = SyncResult(
        decision_id=ctx.decision_id,
        direction=SyncDirection.ADR_TO_DRAWER,
        a_status=a_status,
        b_status=SyncSideStatus.UNCHANGED,
        a_ref=a_ref,
        b_ref=adr_id,
        elapsed_ms=(time.perf_counter() - start) * 1000,
    )
    if a_status == SyncSideStatus.FAILED:
        result.errors.append(f"Drawer upsert failed: {a_ref}")

    update_index_record(result, ctx)
    return result


def sync_bidirectional(adr_id: str, drawer_id: str) -> SyncResult:
    """双向同步（冲突检测 + 合并 + 幂等写入）"""
    start = time.perf_counter()
    ts = datetime.now(timezone.utc).isoformat()

    ctx_a = fetch_drawer(drawer_id)
    ctx_b = fetch_adr(adr_id)

    if not ctx_a and not ctx_b:
        return SyncResult(
            decision_id=make_decision_id(adr_id=adr_id, drawer_id=drawer_id),
            direction=SyncDirection.BIDIRECTIONAL,
            a_status=SyncSideStatus.FAILED,
            b_status=SyncSideStatus.FAILED,
            errors=["both drawer and ADR not found"],
            elapsed_ms=(time.perf_counter() - start) * 1000,
        )

    # 合并
    merged = merge_contexts(
        ctx_a or _empty_context(drawer_id),
        ctx_b or _empty_context(adr_id=adr_id),
    )
    merged.decision_id = make_decision_id(adr_id=adr_id, drawer_id=drawer_id)
    merged.last_sync_direction = SyncDirection.BIDIRECTIONAL
    merged.last_synced_at = ts

    # 冲突检测
    warnings: list[str] = []
    if ctx_a and ctx_b:
        warnings = detect_conflicts(ctx_a, ctx_b)

    # 幂等写入
    try:
        a_status, a_ref = upsert_drawer(merged)
    except Exception as e:
        a_status, a_ref = SyncSideStatus.FAILED, str(e)

    try:
        b_status, b_ref = upsert_adr(merged)
    except Exception as e:
        b_status, b_ref = SyncSideStatus.FAILED, str(e)

    rolled_back = a_status == SyncSideStatus.FAILED or b_status == SyncSideStatus.FAILED

    result = SyncResult(
        decision_id=merged.decision_id,
        direction=SyncDirection.BIDIRECTIONAL,
        a_status=a_status,
        b_status=b_status,
        a_ref=a_ref,
        b_ref=b_ref,
        warnings=warnings,
        rolled_back=rolled_back,
        elapsed_ms=(time.perf_counter() - start) * 1000,
    )
    if a_status == SyncSideStatus.FAILED:
        result.errors.append(f"Drawer upsert failed: {a_ref}")
    if b_status == SyncSideStatus.FAILED:
        result.errors.append(f"ADR upsert failed: {b_ref}")

    update_index_record(result, merged)
    return result


def _empty_context(ref: str | None = None, adr_id: str | None = None) -> DecisionContext:
    return DecisionContext(
        decision_id=make_decision_id(drawer_id=ref, adr_id=adr_id),
        drawer_id=ref,
        adr_id=adr_id,
    )


# ============================================================
# ID 生成（ADR 新编号）
# ============================================================

def next_adr_id() -> str:
    """生成下一个 ADR 编号（YYYY-NNNN）"""
    year = datetime.now(timezone.utc).year
    payload = json.dumps({
        "tool": "manage_adr",
        "args": {"action": "list", "year_only": True}
    })
    try:
        result = subprocess.run(
            [CODEBASE_MEM_BIN, "call", "--json", payload],
            capture_output=True, text=True, timeout=10
        )
        existing = json.loads(result.stdout).get("adrs", [])
        next_n = max(
            [int(a["adr_id"].rsplit("-", 1)[-1])
             for a in existing
             if a.get("adr_id", "").startswith(f"ADR-{year}-")] + [0]
        ) + 1
    except Exception:
        next_n = 1
    return f"ADR-{year}-{next_n:04d}"


# ============================================================
# 索引管理
# ============================================================

def load_index() -> dict[str, Any]:
    if INDEX_PATH.exists():
        return json.loads(INDEX_PATH.read_text(encoding="utf-8"))
    return {"decisions": {}, "last_reconcile": None, "version": "1.0"}


def save_index(index: dict[str, Any]) -> None:
    INDEX_PATH.write_text(
        json.dumps(index, ensure_ascii=False, indent=2),
        encoding="utf-8"
    )


def update_index_record(result: SyncResult, ctx: DecisionContext) -> None:
    """更新索引中的单条记录"""
    index = load_index()
    entry = index["decisions"].get(ctx.decision_id, {})
    entry.update({
        "decision_id": ctx.decision_id,
        "adr_id": result.b_ref or ctx.adr_id,
        "drawer_id": result.a_ref or ctx.drawer_id,
        "title": ctx.title,
        "status": ctx.status.value,
        "wing": ctx.wing,
        "room": ctx.room,
        "linked_symbols": ctx.linked_symbols,
        "supersedes": ctx.supersedes,
        "superseded_by": ctx.superseded_by,
        "last_sync_direction": ctx.last_sync_direction.value,
        "last_synced_at": ctx.last_synced_at,
        "last_sync_result": result.to_dict(),
        "warnings": result.warnings,
        "errors": result.errors,
    })
    index["decisions"][ctx.decision_id] = entry
    index["last_reconcile"] = datetime.now(timezone.utc).isoformat()
    save_index(index)


def reconcile() -> dict[str, Any]:
    """扫描所有索引记录，返回不一致报告"""
    index = load_index()
    inconsistencies: list[dict[str, Any]] = []
    stats = {"total": len(index["decisions"]), "by_status": {}, "by_direction": {}}

    for decision_id, entry in index["decisions"].items():
        s = entry.get("status", "unknown")
        d = entry.get("last_sync_direction", "unknown")
        stats["by_status"][s] = stats["by_status"].get(s, 0) + 1
        stats["by_direction"][d] = stats["by_direction"].get(d, 0) + 1

        if entry.get("warnings") or entry.get("errors"):
            inconsistencies.append(entry)
        if entry.get("status") == "superseded" and entry.get("superseded_by") is None:
            inconsistencies.append({**entry, "warning_extra": "superseded but superseded_by is null"})

    report = {
        "stats": stats,
        "inconsistencies": inconsistencies,
        "total_inconsistent": len(inconsistencies),
        "checked_at": datetime.now(timezone.utc).isoformat(),
    }
    return report


# ============================================================
# 全量状态（供控制台实时视图）
# ============================================================

def full_status() -> dict[str, Any]:
    """返回当前双轨同步系统的全量状态（供控制台轮询）"""
    index = load_index()
    decisions = list(index.get("decisions", {}).values())

    by_status: dict[str, int] = {}
    by_direction: dict[str, int] = {}
    recent_failures: list[dict[str, Any]] = []
    evolution_chains: dict[str, list[str]] = {}

    for d in decisions:
        s = d.get("status", "unknown")
        by_status[s] = by_status.get(s, 0) + 1
        direction = d.get("last_sync_direction", "unknown")
        by_direction[direction] = by_direction.get(direction, 0) + 1

        last_result = d.get("last_sync_result", {})
        if last_result.get("errors"):
            recent_failures.append({
                "decision_id": d["decision_id"],
                "title": d.get("title", "")[:60],
                "errors": last_result["errors"],
                "at": d.get("last_synced_at", ""),
            })

        if d.get("superseded_by"):
            topic = d.get("wing", "unknown")
            evolution_chains.setdefault(d["superseded_by"], []).append(d["decision_id"])

    # 最近同步历史（最后 20 条）
    recent_syncs = sorted(
        [d for d in decisions if d.get("last_synced_at")],
        key=lambda x: x["last_synced_at"],
        reverse=True
    )[:20]

    return {
        "total_decisions": len(decisions),
        "by_status": by_status,
        "by_direction": by_direction,
        "recent_failures": recent_failures[:10],
        "recent_syncs": recent_syncs,
        "evolution_chains": evolution_chains,
        "checked_at": datetime.now(timezone.utc).isoformat(),
    }


# ============================================================
# CLI 兼容入口（直接运行 adr-sync.py 风格的入口）
# ============================================================

def main() -> None:
    import argparse, sys as _sys
    parser = argparse.ArgumentParser(description="ADR 双写同步引擎")
    parser.add_argument("--direction", choices=["drawer-to-adr","adr-to-drawer","bidirectional"])
    parser.add_argument("--drawer-id")
    parser.add_argument("--adr-id")
    parser.add_argument("--reconcile", action="store_true")
    parser.add_argument("--status", action="store_true")
    args = parser.parse_args()

    if args.status:
        print(json.dumps(full_status(), ensure_ascii=False, indent=2))
        return

    if args.reconcile:
        print(json.dumps(reconcile(), ensure_ascii=False, indent=2))
        return

    if not args.direction:
        parser.print_help()
        _sys.exit(1)

    direction = SyncDirection(args.direction)
    if direction == SyncDirection.DRAWER_TO_ADR:
        if not args.drawer_id:
            parser.error("--drawer-id required")
        result = sync_drawer_to_adr(args.drawer_id)
    elif direction == SyncDirection.ADR_TO_DRAWER:
        if not args.adr_id:
            parser.error("--adr-id required")
        result = sync_adr_to_drawer(args.adr_id)
    else:
        if not (args.adr_id and args.drawer_id):
            parser.error("--adr-id and --drawer-id both required for bidirectional")
        result = sync_bidirectional(args.adr_id, args.drawer_id)

    print(json.dumps(result.to_dict(), ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
