#!/usr/bin/env python3
"""
双轨记忆框架对比评测运行脚本
============================

对应 benchmark/plan.md §3 评测运行协议。

支持的模式：
  - A-only  : 仅调用 mempalace MCP
  - B-only  : 仅调用 codebase-memory-mcp
  - A+B     : 双轨联合 + 控制台合并

用法：
  python run_bench.py --dataset DS-Decision --mode A-only
  python run_bench.py --dataset DS-Cross --mode A+B --limit 20
  python run_bench.py --all --report reports/report-v0.1-2026-07-06.md

依赖：
  pip install mcp-client httpx tenacity pydantic

环境变量：
  MEMPALACE_URL      : 默认 http://192.168.110.60:8080/mcp
  CODEBASE_MEM_BIN   : 默认 codebase-memory-mcp
  LLM_MODEL          : 默认 claude-sonnet-4
"""

import argparse
import json
import os
import statistics
import sys
import time
from dataclasses import dataclass, field, asdict
from pathlib import Path
from typing import Any

try:
    from mcp import ClientSession  # type: ignore
    from mcp.client.stdio import stdio_client  # type: ignore
    from mcp.client.http import http_client  # type: ignore
except ImportError:
    print("ERROR: mcp-client not installed. Run: pip install mcp-client")
    sys.exit(1)

try:
    import httpx
    from tenacity import retry, stop_after_attempt, wait_exponential
except ImportError:
    print("ERROR: missing dependencies. Run: pip install httpx tenacity")
    sys.exit(1)


# ============================================================
# 配置
# ============================================================

ROOT = Path(__file__).resolve().parents[1]
DATASETS_DIR = ROOT / "datasets"
REPORTS_DIR = ROOT / "reports"
REPORTS_DIR.mkdir(exist_ok=True)

MEMPALACE_URL = os.environ.get("MEMPALACE_URL", "http://192.168.110.60:8080/mcp")
CODEBASE_MEM_BIN = os.environ.get("CODEBASE_MEM_BIN", "codebase-memory-mcp")
LLM_MODEL = os.environ.get("LLM_MODEL", "claude-sonnet-4")


# ============================================================
# 数据结构
# ============================================================

@dataclass
class SampleResult:
    sample_id: str
    query: str
    latency_ms: float
    top_k_ids: list[str]
    token_in: int = 0
    token_out: int = 0
    tool_calls: list[dict] = field(default_factory=list)
    error: str | None = None


@dataclass
class DatasetMetrics:
    dataset: str
    mode: str
    n_samples: int
    n_failed: int
    r_at_5: float
    r_at_10: float
    mrr: float
    p_at_5: float
    p_at_10: float
    latency_p50_ms: float
    latency_p95_ms: float
    latency_p99_ms: float
    token_total: int
    complementarity_gain: float | None = None
    fidelity_match_rate: float | None = None
    extra_metrics: dict = field(default_factory=dict)


# ============================================================
# MCP 调用层
# ============================================================

@retry(stop=stop_after_attempt(2), wait=wait_exponential(min=1, max=10))
def call_mempalace(tool: str, args: dict) -> dict:
    """调用 MemPalace MCP"""
    # 实际实现：根据 IDE 客户端 SDK 调整
    # 这里用 HTTP 客户端示例
    with httpx.Client(timeout=30) as client:
        resp = client.post(
            MEMPALACE_URL,
            json={"tool": tool, "args": args}
        )
        resp.raise_for_status()
        return resp.json()


@retry(stop=stop_after_attempt(2), wait=wait_exponential(min=1, max=10))
def call_codebase_mem(tool: str, args: dict) -> dict:
    """调用 codebase-memory-mcp"""
    # 实际实现：stdio 启动二进制
    import subprocess
    payload = json.dumps({"tool": tool, "args": args})
    result = subprocess.run(
        [CODEBASE_MEM_BIN, "call", "--json", payload],
        capture_output=True, text=True, timeout=30
    )
    result.check_returncode()
    return json.loads(result.stdout)


# ============================================================
# 指标计算
# ============================================================

def compute_ranking_metrics(predicted_ids: list[str], ground_truth: list[dict],
                              top_k: int = 10) -> dict:
    """R@K / P@K / MRR"""
    gt_ids = {gt.get("ref") or gt.get("drawer_id") or gt.get("qualified_name")
              for gt in ground_truth}

    # R@K
    r_at_k = float(any(pid in gt_ids for pid in predicted_ids[:top_k]))

    # P@K
    hits = sum(1 for pid in predicted_ids[:top_k] if pid in gt_ids)
    p_at_k = hits / top_k

    # MRR
    mrr = 0.0
    for i, pid in enumerate(predicted_ids, start=1):
        if pid in gt_ids:
            mrr = 1.0 / i
            break

    return {"r_at_k": r_at_k, "p_at_k": p_at_k, "mrr": mrr, "top_k": top_k}


def compute_fidelity(raw_quote: str, response_content: str) -> dict:
    """verbatim 字符级保真度"""
    from difflib import SequenceMatcher
    matcher = SequenceMatcher(None, raw_quote, response_content)
    match_rate = matcher.ratio()
    edit_distance = len(raw_quote) + len(response_content) - \
                    2 * sum(b.size for b in matcher.get_matching_blocks())
    return {"match_rate": match_rate, "edit_distance": edit_distance}


# ============================================================
# 各数据集评测器
# ============================================================

def eval_decision(sample: dict, mode: str) -> SampleResult:
    """DS-Decision: A 语义检索"""
    start = time.time()
    try:
        if mode in ("A-only", "A+B"):
            resp = call_mempalace("mempalace_search", {
                "query": sample["query"],
                "wing_scope": sample.get("expected_wing_scope", [])
            })
            top_ids = [r["drawer_id"] for r in resp.get("drawers", [])]
        else:
            top_ids = []
        latency = (time.time() - start) * 1000
        return SampleResult(sample["id"], sample["query"], latency, top_ids)
    except Exception as e:
        return SampleResult(sample["id"], sample["query"],
                            (time.time() - start) * 1000, [], error=str(e))


def eval_callpath(sample: dict, mode: str) -> SampleResult:
    """DS-CallPath: B 结构调用链"""
    start = time.time()
    try:
        if mode in ("B-only", "A+B"):
            resp = call_codebase_mem("trace_path", {
                "start": sample["entry_point"]["qualified_name"],
                "direction": sample["query_type"],
                "max_depth": sample["max_depth"]
            })
            top_ids = [n["qualified_name"] for n in resp.get("nodes", [])]
        else:
            top_ids = []
        latency = (time.time() - start) * 1000
        return SampleResult(sample["id"], sample["query"], latency, top_ids)
    except Exception as e:
        return SampleResult(sample["id"], sample["query"],
                            (time.time() - start) * 1000, [], error=str(e))


def eval_cross(sample: dict, mode: str) -> SampleResult:
    """DS-Cross: A+B 联合检索"""
    start = time.time()
    try:
        top_ids = []
        # A 侧
        if mode in ("A-only", "A+B"):
            resp_a = call_mempalace("mempalace_search", {"query": sample["query"]})
            top_ids += [r["drawer_id"] for r in resp_a.get("drawers", [])]
        # B 侧
        if mode in ("B-only", "A+B"):
            resp_b = call_codebase_mem("semantic_query", {
                "query": sample["query"], "limit": 10
            })
            top_ids += [n["qualified_name"] for n in resp_b.get("nodes", [])]
        # 去重但保持顺序
        seen = set()
        top_ids = [x for x in top_ids if not (x in seen or seen.add(x))]
        latency = (time.time() - start) * 1000
        return SampleResult(sample["id"], sample["query"], latency, top_ids)
    except Exception as e:
        return SampleResult(sample["id"], sample["query"],
                            (time.time() - start) * 1000, [], error=str(e))


def eval_adr(sample: dict, mode: str) -> SampleResult:
    """DS-ADR: B ADR 检索"""
    start = time.time()
    try:
        resp = call_codebase_mem("manage_adr", {
            "action": "list",
            "filter": "topic=" + sample["metadata"]["topic"]
        })
        top_ids = [a["adr_id"] for a in resp.get("adrs", [])]
        latency = (time.time() - start) * 1000
        return SampleResult(sample["id"], sample["query"], latency, top_ids)
    except Exception as e:
        return SampleResult(sample["id"], sample["query"],
                            (time.time() - start) * 1000, [], error=str(e))


def eval_customer(sample: dict, mode: str) -> SampleResult:
    """DS-Customer: verbatim 保真度"""
    start = time.time()
    try:
        raw = sample["source"]["raw_quote"]
        resp = call_mempalace("mempalace_search", {
            "query": raw, "wing_scope": [sample["expected_drawer"]["wing"]]
        })
        top_ids = [r["drawer_id"] for r in resp.get("drawers", [])]
        # 额外采集响应原文用于 fidelity 计算
        resp_content = resp.get("drawers", [{}])[0].get("content", "")
        sample["__fidelity__"] = compute_fidelity(raw, resp_content)
        latency = (time.time() - start) * 1000
        return SampleResult(sample["id"], sample["query"], latency, top_ids)
    except Exception as e:
        return SampleResult(sample["id"], sample["query"],
                            (time.time() - start) * 1000, [], error=str(e))


def eval_deadcode(sample: dict, mode: str) -> SampleResult:
    """DS-DeadCode: B 静态分析"""
    start = time.time()
    try:
        resp = call_codebase_mem("query_graph", {
            "query": f"MATCH (n {{qualified_name: '{sample['candidate']['qualified_name']}'}})<-[r:CALLS|HTTP_CALLS|LISTENS_ON]-() RETURN count(r)"
        })
        inbound = resp.get("count", 0)
        is_dead_predicted = (inbound == 0)
        # 这里 top_ids 用 0/1 编码 dead/active
        top_ids = ["dead" if is_dead_predicted else "alive"]
        latency = (time.time() - start) * 1000
        return SampleResult(sample["id"], sample["query"], latency, top_ids)
    except Exception as e:
        return SampleResult(sample["id"], sample["query"],
                            (time.time() - start) * 1000, [], error=str(e))


# ============================================================
# 入口
# ============================================================

EVALUATORS = {
    "DS-Decision": eval_decision,
    "DS-CallPath": eval_callpath,
    "DS-Cross": eval_cross,
    "DS-ADR": eval_adr,
    "DS-Customer": eval_customer,
    "DS-DeadCode": eval_deadcode,
}


def run_dataset(dataset_name: str, mode: str, limit: int | None = None) -> DatasetMetrics:
    """跑一个数据集的评测"""
    samples_path = DATASETS_DIR / f"{dataset_name}.jsonl"
    samples = [json.loads(line) for line in samples_path.read_text(encoding="utf-8").splitlines() if line.strip()]
    if limit:
        samples = samples[:limit]

    evaluator = EVALUATORS[dataset_name]
    results: list[SampleResult] = []
    for sample in samples:
        result = evaluator(sample, mode)
        results.append(result)
        # 即时打印
        print(f"[{dataset_name}][{mode}] {sample['id']}: {len(result.top_k_ids)} hits, {result.latency_ms:.1f}ms")

    # 计算指标
    metrics = aggregate_metrics(dataset_name, mode, samples, results)
    return metrics


def aggregate_metrics(dataset: str, mode: str, samples: list[dict],
                     results: list[SampleResult]) -> DatasetMetrics:
    """聚合指标"""
    n_samples = len(results)
    n_failed = sum(1 for r in results if r.error)

    # 召回率 / 精度 / MRR
    rks, pks, mrrs = [], [], []
    for sample, result in zip(samples, results):
        m = compute_ranking_metrics(result.top_k_ids, sample.get("ground_truth", []))
        rks.append(m["r_at_k"] if m["top_k"] == 5 else None)
        pks.append(m["p_at_k"] if m["top_k"] == 5 else None)
        mrrs.append(m["mrr"])
    r_at_5 = sum(x for x in rks if x is not None) / max(1, sum(1 for x in rks if x is not None))
    p_at_5 = sum(x for x in pks if x is not None) / max(1, sum(1 for x in pks if x is not None))
    mrr = sum(mrrs) / max(1, len(mrrs))

    # 延迟
    latencies = sorted(r.latency_ms for r in results if not r.error)
    p50 = latencies[int(len(latencies) * 0.5)] if latencies else 0
    p95 = latencies[int(len(latencies) * 0.95)] if latencies else 0
    p99 = latencies[int(len(latencies) * 0.99)] if latencies else 0

    return DatasetMetrics(
        dataset=dataset, mode=mode,
        n_samples=n_samples, n_failed=n_failed,
        r_at_5=r_at_5, r_at_10=r_at_5,  # 简化：实际跑两份 top5 / top10
        mrr=mrr, p_at_5=p_at_5, p_at_10=p_at_5,
        latency_p50_ms=p50, latency_p95_ms=p95, latency_p99_ms=p99,
        token_total=sum(r.token_in + r.token_out for r in results)
    )


def main():
    parser = argparse.ArgumentParser(description="双轨记忆框架评测")
    parser.add_argument("--dataset", choices=list(EVALUATORS.keys()))
    parser.add_argument("--mode", choices=["A-only", "B-only", "A+B"], required=True)
    parser.add_argument("--limit", type=int, help="限制样本数")
    parser.add_argument("--all", action="store_true", help="跑全部数据集")
    parser.add_argument("--report", type=str, help="报告输出路径")
    args = parser.parse_args()

    if args.all:
        metrics_list = []
        for ds in EVALUATORS:
            for mode in ["A-only", "B-only", "A+B"]:
                try:
                    m = run_dataset(ds, mode, args.limit)
                    metrics_list.append(m)
                except Exception as e:
                    print(f"ERROR {ds}/{mode}: {e}")
    else:
        if not args.dataset:
            parser.error("either --dataset or --all is required")
        metrics_list = [run_dataset(args.dataset, args.mode, args.limit)]

    # 输出
    out = "\n".join(json.dumps(asdict(m), ensure_ascii=False, indent=2) for m in metrics_list)
    print(out)

    if args.report:
        Path(args.report).write_text(out, encoding="utf-8")
        print(f"\nReport saved to {args.report}")


if __name__ == "__main__":
    main()