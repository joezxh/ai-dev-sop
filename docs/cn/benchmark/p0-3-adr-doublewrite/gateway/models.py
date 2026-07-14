"""
MemPlace 网关 · Pydantic 请求 / 响应模型
========================================

对应 §5.7.7 控制台实时视图 + §5.10.3 P0-3 ADR 双写同步。

命名约定（OpenAPI 兼容）：
  - 请求：XxxRequest
  - 响应：XxxResponse / XxxStatus
  - 枚举：XxxEnum
"""

from __future__ import annotations

from datetime import datetime
from enum import Enum
from typing import Annotated

from pydantic import BaseModel, Field, HttpUrl, field_validator


# ============================================================
# 枚举
# ============================================================

class SyncDirectionEnum(str, Enum):
    DRAWER_TO_ADR = "drawer-to-adr"
    ADR_TO_DRAWER = "adr-to-drawer"
    BIDIRECTIONAL = "bidirectional"


class ADRStatusEnum(str, Enum):
    PROPOSED = "proposed"
    ACCEPTED = "accepted"
    SUPERSEDED = "superseded"
    DEPRECATED = "deprecated"


class SyncSideStatusEnum(str, Enum):
    CREATED = "created"
    UPDATED = "updated"
    UNCHANGED = "unchanged"
    FAILED = "failed"


# ============================================================
# 请求模型
# ============================================================

class SyncRequest(BaseModel):
    """
    POST /api/v1/decisions/sync
    核心同步端点。
    """
    direction: SyncDirectionEnum = Field(
        description="同步方向",
        examples=["drawer-to-adr", "adr-to-drawer", "bidirectional"],
    )
    drawer_id: str | None = Field(
        default=None,
        description="MemPalace drawer_id（direction=drawer-to-adr 或 bidirectional 时必填）",
        examples=["drawer_8f2c1a"],
        min_length=1,
    )
    adr_id: str | None = Field(
        default=None,
        description="codebase-memory-mcp ADR id（direction=adr-to-drawer 或 bidirectional 时必填）",
        examples=["ADR-2026-0007"],
        min_length=1,
    )
    wing: str | None = Field(
        default="wing-org-decisions",
        description="MemPalace Wing（新建 drawer 时使用）",
    )
    room: str = Field(default="general", description="MemPalace Room")
    hall: str = Field(default="facts", description="MemPalace Hall")
    title: str | None = Field(
        default=None,
        description="决策标题（新建时可选）",
        examples=["向量库后端选型：PGVector"],
    )
    context: str | None = Field(
        default=None,
        description="背景与问题陈述",
        max_length=8000,
    )
    decision: str | None = Field(
        default=None,
        description="决策正文",
        max_length=4000,
    )
    consequences: str | None = Field(
        default=None,
        description="后果与影响",
        max_length=4000,
    )
    status: ADRStatusEnum = Field(
        default=ADRStatusEnum.ACCEPTED,
        description="ADR 状态",
    )
    supersedes: list[str] = Field(
        default_factory=list,
        description="被本决策替代的前置 ADR 列表",
        examples=[["ADR-2025-0001"]],
    )
    linked_symbols: list[str] = Field(
        default_factory=list,
        description="关联的代码符号（qualified_name 列表）",
        examples=[["adsop.module.kms.service.VectorStoreService"]],
    )
    idempotency_key: str | None = Field(
        default=None,
        description="可选幂等键（覆盖默认 decision_id 策略）",
    )

    @field_validator("drawer_id", "adr_id", mode="after")
    @classmethod
    def strip_whitespace(cls, v: str | None) -> str | None:
        if isinstance(v, str):
            v = v.strip()
        return v if v else None

    def model_post_init(self, __context) -> None:
        if self.direction == SyncDirectionEnum.DRAWER_TO_ADR and not self.drawer_id:
            raise ValueError("drawer_id required for drawer-to-adr")
        if self.direction == SyncDirectionEnum.ADR_TO_DRAWER and not self.adr_id:
            raise ValueError("adr_id required for adr-to-drawer")
        if self.direction == SyncDirectionEnum.BIDIRECTIONAL:
            if not self.drawer_id and not self.adr_id:
                raise ValueError("at least one of drawer_id or adr_id required for bidirectional")


class HealthCheckRequest(BaseModel):
    """GET /health（无请求体）"""
    pass


# ============================================================
# 响应模型
# ============================================================

class SyncSideResult(BaseModel):
    """同步结果中一侧的状态"""
    side: str = Field(description="A = MemPalace, B = codebase-memory-mcp")
    status: SyncSideStatusEnum
    ref: str | None = Field(description="drawer_id 或 adr_id")


class SyncResponse(BaseModel):
    """POST /api/v1/decisions/sync 响应"""
    ok: bool = Field(description="是否成功（两侧都未 failed）")
    decision_id: str
    direction: SyncDirectionEnum
    mempalace: SyncSideResult = Field(
        description="MemPalace 侧结果",
    )
    codebase_mem: SyncSideResult = Field(
        description="codebase-memory-mcp 侧结果",
    )
    warnings: list[str] = Field(
        default_factory=list,
        description="冲突警告（字段差异等）",
    )
    errors: list[str] = Field(
        default_factory=list,
        description="错误列表",
    )
    rolled_back: bool = Field(
        default=False,
        description="是否触发了回滚（bidirectional 模式下一侧失败时）",
    )
    elapsed_ms: float = Field(
        description="处理耗时（毫秒）",
    )
    synced_at: datetime = Field(
        default_factory=datetime.utcnow,
        description="同步完成时间",
    )

    @field_validator("elapsed_ms")
    @classmethod
    def round_elapsed(cls, v: float) -> float:
        return round(v, 2)


class DecisionRecord(BaseModel):
    """单个决策的完整记录"""
    decision_id: str
    adr_id: str | None
    drawer_id: str | None
    title: str
    status: ADRStatusEnum
    wing: str
    room: str
    linked_symbols: list[str] = Field(default_factory=list)
    supersedes: list[str] = Field(default_factory=list)
    superseded_by: str | None
    last_sync_direction: SyncDirectionEnum
    last_synced_at: str
    warnings: list[str] = Field(default_factory=list)
    errors: list[str] = Field(default_factory=list)


class StatusResponse(BaseModel):
    """
    GET /api/v1/decisions/status
    全量系统状态（供控制台实时视图轮询）
    """
    total_decisions: int
    by_status: dict[str, int]
    by_direction: dict[str, int]
    recent_failures: list[dict] = Field(
        default_factory=list,
        description="最近 10 条失败记录",
    )
    evolution_chains: dict[str, list[str]] = Field(
        default_factory=dict,
        description="演进链（adr_id → 被其替代的 ADR 列表）",
    )
    checked_at: str


class ListDecisionsResponse(BaseModel):
    """GET /api/v1/decisions"""
    decisions: list[DecisionRecord]
    total: int
    page: int
    page_size: int
    pages: int


class ReconcileResponse(BaseModel):
    """GET /api/v1/decisions/reconcile"""
    stats: dict[str, dict[str, int]]
    inconsistencies: list[dict]
    total_inconsistent: int
    checked_at: str


class HealthResponse(BaseModel):
    """GET /health"""
    status: Literal["healthy", "degraded", "unhealthy"]
    mempalace: Literal["connected", "disconnected", "unknown"]
    codebase_mem: Literal["connected", "disconnected", "unknown"]
    sync_engine: Literal["ready", "error"]
    version: str
    uptime_seconds: float


from typing import Literal


# ============================================================
# WebSocket 消息（实时推送）
# ============================================================

class WSEventType(str, Enum):
    SYNC_COMPLETED = "sync_completed"
    SYNC_FAILED = "sync_failed"
    RECONCILE_ALERT = "reconcile_alert"
    STATUS_SNAPSHOT = "status_snapshot"


class WSMessage(BaseModel):
    """WebSocket 推送消息"""
    type: WSEventType
    payload: dict
    ts: datetime = Field(default_factory=datetime.utcnow)
