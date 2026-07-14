"""
MemPlace 统一网关 · FastAPI Server
=================================

对应 §5.7.7 控制台实时视图 + §5.10.3 P0-3 ADR 双写同步。
Ide 客户端零改造接入：只需改 MCP server URL 指向本网关即可。

端点概览：
  GET  /health                         健康检查
  GET  /api/v1/decisions               列表（分页）
  POST /api/v1/decisions/sync         核心同步端点（IDE MCP → 本网关 → 双侧写回）
  GET  /api/v1/decisions/{id}         单条详情
  GET  /api/v1/decisions/status       全量系统状态（控制台轮询）
  GET  /api/v1/decisions/reconcile    对账报告
  WS   /ws                             实时推送（控制台订阅）

启动方式：
  uvicorn gateway.server:app --reload --port 8888

环境变量：
  MP_PORT               : 网关端口（默认 8888）
  MP_HOST               : 监听地址（默认 0.0.0.0）
  MP_MEMPALACE_URL      : MemPalace 后端 URL
  MP_CODEBASE_MEM_BIN   : codebase-memory-mcp 二进制路径
  MP_INDEX_PATH         : 索引文件路径
  MP_CORS_ORIGINS       : 允许的 CORS 源（逗号分隔）
  MP_DEBUG              : 调试模式
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Annotated, Any

from fastapi import (
    BackgroundTasks,
    Depends,
    FastAPI,
    HTTPException,
    Path as FP,
    Query,
    Request,
    WebSocket,
    WebSocketDisconnect,
)
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import (
    FileResponse,
    HTMLResponse,
    JSONResponse,
)
from fastapi.staticfiles import StaticFiles

# 本地模块
try:
    from models import (
        ADRStatusEnum,
        HealthResponse,
        ListDecisionsResponse,
        ReconcileResponse,
        SyncDirectionEnum,
        SyncRequest,
        SyncResponse,
        SyncSideResult,
        SyncSideStatusEnum,
        StatusResponse,
        WSMessage,
        WSEventType,
    )
    from sync_engine import (
        SyncDirection,
        SyncSideStatus,
        SyncStatus,
        fetch_adr,
        fetch_drawer,
        find_adr_by_decision_id,
        find_drawer_by_decision_id,
        full_status,
        load_index,
        make_decision_id,
        merge_contexts,
        next_adr_id,
        reconcile as do_reconcile,
        sync_adr_to_drawer,
        sync_bidirectional,
        sync_drawer_to_adr,
        upsert_adr,
        upsert_drawer,
    )
except ImportError:
    # 尝试从父目录导入（独立运行）
    sys.path.insert(0, str(Path(__file__).parent))
    from models import (
        ADRStatusEnum,
        HealthResponse,
        ListDecisionsResponse,
        ReconcileResponse,
        SyncDirectionEnum,
        SyncRequest,
        SyncResponse,
        SyncSideResult,
        SyncSideStatusEnum,
        StatusResponse,
        WSMessage,
        WSEventType,
    )
    from sync_engine import (
        SyncDirection,
        SyncSideStatus,
        make_decision_id,
        fetch_adr,
        fetch_drawer,
        find_adr_by_decision_id,
        find_drawer_by_decision_id,
        full_status,
        load_index,
        merge_contexts,
        next_adr_id,
        reconcile as do_reconcile,
        sync_adr_to_drawer,
        sync_bidirectional,
        sync_drawer_to_adr,
        upsert_adr,
        upsert_drawer,
    )


# ============================================================
# App 配置
# ============================================================

APP_START = time.monotonic()
VERSION = "0.1.0"

app = FastAPI(
    title="MemPlace Gateway",
    description=(
        "MemPalace × codebase-memory-mcp 双轨统一网关。"
        "IDE 客户端只需把 MCP server URL 指向本网关即可零改造接入。"
        "对应 §5.7.7 控制台实时视图 + §5.10.3 P0-3。"
    ),
    version=VERSION,
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
)

# CORS
_allowed = os.environ.get("MP_CORS_ORIGINS", "*")
_allowed_origins = [o.strip() for o in _allowed.split(",")]
app.add_middleware(
    CORSMiddleware,
    allow_origins=_allowed_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)


# ============================================================
# WebSocket 连接管理器
# ============================================================

class ConnectionManager:
    """支持多个控制台客户端并发订阅"""

    def __init__(self):
        self.active: dict[str, WebSocket] = {}
        self._lock = asyncio.Lock()

    async def connect(self, client_id: str, ws: WebSocket) -> None:
        await ws.accept()
        async with self._lock:
            self.active[client_id] = ws

    async def disconnect(self, client_id: str) -> None:
        async with self._lock:
            self.active.pop(client_id, None)

    async def broadcast(self, message: dict) -> None:
        payload = json.dumps(message, ensure_ascii=False)
        async with self._lock:
            dead = []
            for cid, ws in self.active.items():
                try:
                    await ws.send_text(payload)
                except Exception:
                    dead.append(cid)
            for cid in dead:
                self.active.pop(cid, None)


manager = ConnectionManager()


# ============================================================
# 辅助函数
# ============================================================

def _side_from_status(s: SyncSideStatus) -> SyncSideStatusEnum:
    mapping = {
        SyncSideStatus.CREATED: SyncSideStatusEnum.CREATED,
        SyncSideStatus.UPDATED: SyncSideStatusEnum.UPDATED,
        SyncSideStatus.UNCHANGED: SyncSideStatusEnum.UNCHANGED,
        SyncSideStatus.FAILED: SyncSideStatusEnum.FAILED,
    }
    return mapping.get(s, SyncSideStatusEnum.FAILED)


def _direction_from_str(d: SyncDirection | str) -> SyncDirectionEnum:
    if isinstance(d, SyncDirection):
        return SyncDirectionEnum(d.value)
    return SyncDirectionEnum(d)


async def _notify_status_snapshot() -> None:
    """推送一次全量状态快照到所有 WebSocket 客户端"""
    status = full_status()
    msg = WSMessage(
        type=WSEventType.STATUS_SNAPSHOT,
        payload=status,
    )
    await manager.broadcast({
        "type": msg.type.value,
        "payload": msg.payload,
        "ts": datetime.now(timezone.utc).isoformat(),
    })


# ============================================================
# 端点
# ============================================================

@app.get("/health", response_model=HealthResponse, tags=["系统"])
async def health():
    """
    健康检查。
    返回网关、MemPalace、codebase-memory-mcp 的连接状态。
    """
    import httpx, subprocess

    uptime = time.monotonic() - APP_START
    mempalace_status: Literal["connected", "disconnected", "unknown"] = "unknown"
    cbm_status: Literal["connected", "disconnected", "unknown"] = "unknown"
    sync_ready = True

    try:
        async with httpx.AsyncClient(timeout=3) as client:
            r = await client.post(
                os.environ.get("MP_MEMPALACE_URL", "http://192.168.110.60:8080/mcp"),
                json={"tool": "mempalace_status", "args": {}}
            )
            mempalace_status = "connected" if r.status_code == 200 else "disconnected"
    except Exception:
        mempalace_status = "disconnected"

    try:
        subprocess.run(
            [os.environ.get("MP_CODEBASE_MEM_BIN", "codebase-memory-mcp"), "health"],
            capture_output=True, timeout=5
        )
        cbm_status = "connected"
    except Exception:
        cbm_status = "disconnected"

    overall = "healthy"
    if mempalace_status == "disconnected" or cbm_status == "disconnected":
        overall = "degraded"
    if mempalace_status == "disconnected" and cbm_status == "disconnected":
        overall = "unhealthy"

    return HealthResponse(
        status=overall,
        mempalace=mempalace_status,
        codebase_mem=cbm_status,
        sync_engine="ready" if sync_ready else "error",
        version=VERSION,
        uptime_seconds=round(uptime, 2),
    )


@app.post(
    "/api/v1/decisions/sync",
    response_model=SyncResponse,
    tags=["核心同步"],
    summary="ADR 双写同步（IDE 客户端零改造接入点）",
)
async def sync_decision(
    req: SyncRequest,
    background: BackgroundTasks,
):
    """
    核心同步端点。

    IDE 客户端通过 MCP 协议调用本端点，本网关将请求路由到 MemPalace +
    codebase-memory-mcp 任意一侧或两侧，返回同步结果。

    **调用方式**：
    - MemPalace MCP → IDE：MCP 协议调用本端点
    - cURL：`curl -X POST http://localhost:8888/api/v1/decisions/sync -H "Content-Type: application/json" -d '{...}'`
    - IDE 零改造：只需把 IDE 的 mempalace MCP server URL 改为 `http://localhost:8888/mcp`
    """
    direction = SyncDirection(req.direction.value)
    errors: list[str] = []

    if direction == SyncDirection.DRAWER_TO_ADR:
        result = sync_drawer_to_adr(req.drawer_id)
    elif direction == SyncDirection.ADR_TO_DRAWER:
        result = sync_adr_to_drawer(req.adr_id)
    else:
        result = sync_bidirectional(req.adr_id, req.drawer_id)

    # 背景推送状态快照
    background.add_task(_notify_status_snapshot)

    ok = (
        result.a_status != SyncSideStatus.FAILED
        and result.b_status != SyncSideStatus.FAILED
    )

    return SyncResponse(
        ok=ok,
        decision_id=result.decision_id,
        direction=_direction_from_str(result.direction),
        mempalace=SyncSideResult(
            side="A",
            status=_side_from_status(result.a_status),
            ref=result.a_ref,
        ),
        codebase_mem=SyncSideResult(
            side="B",
            status=_side_from_status(result.b_status),
            ref=result.b_ref,
        ),
        warnings=result.warnings,
        errors=result.errors,
        rolled_back=result.rolled_back,
        elapsed_ms=result.elapsed_ms,
    )


@app.get(
    "/api/v1/decisions",
    response_model=ListDecisionsResponse,
    tags=["查询"],
)
async def list_decisions(
    page: Annotated[int, Query(ge=1, description="页码，从 1 开始")] = 1,
    page_size: Annotated[int, Query(ge=1, le=100, description="每页条数")] = 20,
    status: ADRStatusEnum | None = Query(None, description="按状态过滤"),
    wing: str | None = Query(None, description="按 wing 过滤"),
    direction: SyncDirectionEnum | None = Query(None, description="按最近同步方向过滤"),
    search: str | None = Query(None, description="按标题 / decision_id 模糊搜索"),
):
    """
    分页列出所有决策记录。
    支持按 status / wing / direction / search 过滤。
    """
    index = load_index()
    decisions = list(index.get("decisions", {}).values())

    # 过滤
    if status:
        decisions = [d for d in decisions if d.get("status") == status.value]
    if wing:
        decisions = [d for d in decisions if wing in d.get("wing", "")]
    if direction:
        decisions = [d for d in decisions if d.get("last_sync_direction") == direction.value]
    if search:
        q = search.lower()
        decisions = [
            d for d in decisions
            if q in d.get("title", "").lower() or q in d.get("decision_id", "").lower()
        ]

    # 排序（最近同步优先）
    decisions.sort(key=lambda d: d.get("last_synced_at", ""), reverse=True)

    total = len(decisions)
    pages = (total + page_size - 1) // page_size
    start = (page - 1) * page_size
    page_items = decisions[start:start + page_size]

    records = [
        {
            "decision_id": d["decision_id"],
            "adr_id": d.get("adr_id"),
            "drawer_id": d.get("drawer_id"),
            "title": d.get("title", ""),
            "status": d.get("status", "unknown"),
            "wing": d.get("wing", ""),
            "room": d.get("room", ""),
            "linked_symbols": d.get("linked_symbols", []),
            "supersedes": d.get("supersedes", []),
            "superseded_by": d.get("superseded_by"),
            "last_sync_direction": d.get("last_sync_direction", ""),
            "last_synced_at": d.get("last_synced_at", ""),
            "warnings": d.get("warnings", []),
            "errors": d.get("errors", []),
        }
        for d in page_items
    ]

    return ListDecisionsResponse(
        decisions=records,
        total=total,
        page=page,
        page_size=page_size,
        pages=pages,
    )


@app.get(
    "/api/v1/decisions/{decision_id}",
    tags=["查询"],
)
async def get_decision(decision_id: str = FP(description="decision_id 或 adr_id 或 drawer_id")):
    """
    获取单条决策详情。

    查找顺序：decision_id > adr_id > drawer_id
    """
    index = load_index()
    record = index.get("decisions", {}).get(decision_id)

    if not record:
        # 尝试 adr_id
        adr_id = find_adr_by_decision_id(decision_id)
        if adr_id:
            ctx = fetch_adr(adr_id)
            if ctx:
                record = {
                    "decision_id": ctx.decision_id,
                    "adr_id": ctx.adr_id,
                    "drawer_id": ctx.drawer_id,
                    "title": ctx.title,
                    "status": ctx.status.value,
                    "wing": ctx.wing,
                    "room": ctx.room,
                    "linked_symbols": ctx.linked_symbols,
                    "last_synced_at": ctx.last_synced_at,
                    "source": "codebase-memory-mcp",
                }
        else:
            drawer_id = find_drawer_by_decision_id(decision_id)
            if drawer_id:
                ctx = fetch_drawer(drawer_id)
                if ctx:
                    record = {
                        "decision_id": ctx.decision_id,
                        "adr_id": ctx.adr_id,
                        "drawer_id": ctx.drawer_id,
                        "title": ctx.title,
                        "status": ctx.status.value,
                        "wing": ctx.wing,
                        "room": ctx.room,
                        "linked_symbols": ctx.linked_symbols,
                        "last_synced_at": ctx.last_synced_at,
                        "source": "mempalace",
                    }

    if not record:
        raise HTTPException(status_code=404, detail=f"Decision not found: {decision_id}")

    return JSONResponse(record)


@app.get(
    "/api/v1/decisions/status",
    response_model=StatusResponse,
    tags=["系统"],
    summary="全量系统状态（控制台轮询）",
)
async def get_status():
    """
    返回双轨同步系统的全量状态，用于控制台实时视图轮询（每 5-10 秒一次）。

    包含：决策总数 / 按状态分布 / 按同步方向分布 / 最近失败 / 演进链
    """
    return full_status()


@app.get(
    "/api/v1/decisions/reconcile",
    response_model=ReconcileResponse,
    tags=["系统"],
    summary="ADR 双写对账报告",
)
async def reconcile():
    """
    扫描所有决策，报告字段不一致和同步失败。
    由 §5.11 self-check M4 排障路径调用。
    """
    report = do_reconcile()
    return ReconcileResponse(
        stats=report.get("stats", {}),
        inconsistencies=report.get("inconsistencies", []),
        total_inconsistent=report.get("total_inconsistent", 0),
        checked_at=report.get("checked_at", ""),
    )


# ============================================================
# WebSocket：实时推送
# ============================================================

@app.websocket("/ws")
async def websocket_endpoint(ws: WebSocket):
    """
    WebSocket 端点，控制台客户端订阅实时事件。

    推送事件类型：
      - sync_completed  : 每次同步完成后推送
      - sync_failed     : 同步失败时推送
      - reconcile_alert : 对账发现新不一致时推送
      - status_snapshot : 控制台主动请求推送当前快照
    """
    client_id = str(uuid.uuid4())[:8]
    await manager.connect(client_id, ws)

    # 首次连接：推送一次全量快照
    await _notify_status_snapshot()

    try:
        while True:
            # 接收客户端消息（心跳 / 请求快照）
            data = await ws.receive_text()
            try:
                msg = json.loads(data)
                if msg.get("type") == "request_snapshot":
                    await _notify_status_snapshot()
                elif msg.get("type") == "ping":
                    await ws.send_json({"type": "pong", "ts": datetime.now(timezone.utc).isoformat()})
            except json.JSONDecodeError:
                pass
    except WebSocketDisconnect:
        await manager.disconnect(client_id)


# ============================================================
# 控制台静态页面（开发/调试用）
# ============================================================

console_path = Path(__file__).parent / "console.html"
if console_path.exists():
    @app.get("/", include_in_schema=False)
    async def root():
        return FileResponse(str(console_path))

    @app.get("/console", include_in_schema=False)
    async def console():
        return FileResponse(str(console_path))


# ============================================================
# MCP 代理端点（IDE 零改造核心）
# ============================================================

@app.post("/mcp", tags=["MCP 代理"], summary="MCP 协议代理（IDE 零改造接入）")
async def mcp_proxy(req: Request):
    """
    MCP 协议代理。

    IDE 客户端把 MemPalace MCP server URL 改为本网关的 /mcp 端点，
    本网关自动将请求路由到 MemPalace 或 codebase-memory-mcp。

    IDE 零改造：只需在 IDE MCP 配置中把 server URL 从
    `http://192.168.110.60:8080/mcp`
    改为 `http://localhost:8888/mcp`（或部署后的网关 URL）。

    路由规则：
      - tool 包含 "mempalace" 或 "drawer"  →  MemPalace 后端
      - tool 包含 "codebase" 或 "manage_adr" / "trace_path" → codebase-memory-mcp
      - tool = "mcp_proxy_health" → 本网关自身
      - 其余 → 两侧并发请求，结果 round-robin merge
    """
    body = await req.json()
    tool: str = body.get("tool", "")
    args: dict = body.get("args", {})

    # 路由
    if "mempalace" in tool or "drawer" in tool:
        # → MemPalace
        import httpx
        mp_url = os.environ.get("MP_MEMPALACE_URL", "http://192.168.110.60:8080/mcp")
        try:
            async with httpx.AsyncClient(timeout=30) as client:
                resp = await client.post(mp_url, json=body)
                resp.raise_for_status()
                return resp.json()
        except httpx.HTTPStatusError as e:
            return JSONResponse(status_code=e.response.status_code, content=e.response.json())
        except Exception as e:
            return JSONResponse(status_code=502, content={"error": str(e)})

    elif "codebase" in tool or "manage_adr" in tool or tool.startswith("trace_") or tool.startswith("query_") or tool.startswith("search_"):
        # → codebase-memory-mcp
        import subprocess
        payload = json.dumps(body)
        try:
            result = subprocess.run(
                [os.environ.get("MP_CODEBASE_MEM_BIN", "codebase-memory-mcp"), "call", "--json", payload],
                capture_output=True, text=True, timeout=30
            )
            if result.returncode != 0:
                return JSONResponse(status_code=502, content={"error": result.stderr})
            return json.loads(result.stdout)
        except Exception as e:
            return JSONResponse(status_code=502, content={"error": str(e)})

    elif tool == "mcp_proxy_health":
        return {"ok": True, "version": VERSION, "ts": datetime.now(timezone.utc).isoformat()}

    else:
        # 两侧并发
        import asyncio, httpx, subprocess

        async def call_mp():
            mp_url = os.environ.get("MP_MEMPALACE_URL", "http://192.168.110.60:8080/mcp")
            async with httpx.AsyncClient(timeout=30) as client:
                resp = await client.post(mp_url, json=body)
                resp.raise_for_status()
                return resp.json()

        def call_cbm():
            payload = json.dumps(body)
            result = subprocess.run(
                [os.environ.get("MP_CODEBASE_MEM_BIN", "codebase-memory-mcp"), "call", "--json", payload],
                capture_output=True, text=True, timeout=30
            )
            if result.returncode == 0:
                return json.loads(result.stdout)
            return {"error": result.stderr}

        try:
            results = await asyncio.gather(call_mp(), asyncio.to_thread(call_cbm), return_exceptions=True)
            # round-robin merge
            merged: dict = {"tool": tool, "results": [str(r) for r in results]}
            return merged
        except Exception as e:
            return JSONResponse(status_code=500, content={"error": str(e)})


# ============================================================
# 入口
# ============================================================

if __name__ == "__main__":
    import uvicorn
    port = int(os.environ.get("MP_PORT", "8888"))
    host = os.environ.get("MP_HOST", "0.0.0.0")
    uvicorn.run(app, host=host, port=port, reload=bool(os.environ.get("MP_DEBUG")))

from typing import Literal
