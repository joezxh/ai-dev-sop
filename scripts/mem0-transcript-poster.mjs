#!/usr/bin/env node
/**
 * mem0-transcript-poster — 纯脚本（零 LLM）把 CodeBuddy 会话转录 JSONL
 * 逐字组装为 Q/A Markdown 并直投 mem0 REST（infer=false 原文留痕）。
 *
 * 数据源：~/.codebuddy/projects/<munged-cwd>/<session-uuid>.jsonl
 *   记录类型：message(user/assistant) / reasoning / function_call / function_call_result
 *
 * 断点续传：状态存 <workspace>/.mem0/.poster-state.json（按 session 记字节偏移与
 * turn_seq），只提交已闭合的轮次（遇到下一条 user 消息才算闭合）；mem0 按内容
 * 哈希去重，重复提交安全。
 *
 * 用法（通常由 SessionStart hook 调用，flush 上一会话）：
 *   node mem0-transcript-poster.mjs [--workspace <dir>] [--dry-run]
 * 凭证读 <workspace>/.mem0/mem0.config.json（api_key / git_remote / admin_user_id）。
 */

import fs from "node:fs";
import path from "node:path";
import os from "node:os";

// ---------- args ----------
const argv = process.argv.slice(2);
const arg = (name, def) => {
  const i = argv.indexOf(name);
  return i >= 0 && argv[i + 1] ? argv[i + 1] : def;
};
const DRY = argv.includes("--dry-run");
const VERIFY = argv.includes("--verify"); // 端到端原样校验：POST→GET 回读→字节级比对→清理
const workspace = path.resolve(arg("--workspace", process.cwd()));
const restUrl = arg("--rest-url", "http://localhost:8002");
const transcriptDirOverride = arg("--transcript-dir", null); // for testing

// ---------- config ----------
const mem0Dir = path.join(workspace, ".mem0");
const cfgPath = path.join(mem0Dir, "mem0.config.json");
if (!fs.existsSync(cfgPath)) {
  console.error(`[mem0-poster] config missing: ${cfgPath}`);
  process.exit(1);
}
const cfg = JSON.parse(fs.readFileSync(cfgPath, "utf8"));
const apiKey = cfg.api_key;
if (!apiKey) { console.error("[mem0-poster] api_key missing"); process.exit(1); }

// session_id（.mem0/.session_id-cb 优先，与 Agent 留痕同池）
let sessionId = null;
const sidFile = path.join(mem0Dir, ".session_id-cb");
if (fs.existsSync(sidFile)) {
  sessionId = fs.readFileSync(sidFile, "utf8").trim() || null;
}

// ---------- transcript files ----------
// 与 CodeBuddy 转录目录命名保持一致：小写盘符 + 把 :\ / 连续分隔符压成单个 "-"
// （例如 d:\projects\github -> d-projects-github，而非 D--projects-...）
const munged = workspace.toLowerCase().replace(/[:\\/]+/g, "-");
const projDir = transcriptDirOverride ?? path.join(os.homedir(), ".codebuddy", "projects", munged);
if (!fs.existsSync(projDir)) {
  console.log(`[mem0-poster] no transcript dir: ${projDir} (session not flushed yet)`);
  process.exit(0);
}
const jsonlFiles = fs.readdirSync(projDir).filter((f) => f.endsWith(".jsonl"))
  .map((f) => ({ file: path.join(projDir, f), uuid: f.replace(/\.jsonl$/, "") }));
if (jsonlFiles.length === 0) { console.log("[mem0-poster] no transcripts"); process.exit(0); }

const statePath = path.join(mem0Dir, ".poster-state.json");
let state = {};
try { state = JSON.parse(fs.readFileSync(statePath, "utf8")); } catch {}

// ---------- helpers ----------
const iso = (ts) => new Date(ts).toISOString();
const linesOf = (buf) => buf.toString("utf8").split("\n").map((l) => l.trim()).filter(Boolean);
const parse = (line) => { try { return JSON.parse(line); } catch { return null; } };
const texts = (o) => (o?.content ?? []).map((c) => c.text ?? "").filter(Boolean);

function renderTurn(turn) {
  const parts = [`Q: ${turn.q}`];
  parts.push("");
  let a = ["A: "];
  let n = 0;
  for (const th of turn.thinkings) {
    n += 1;
    a.push(`### Thinking ${n}（逐字全文）`, th);
  }
  let t = 0;
  for (const call of turn.calls) {
    t += 1;
    const argsStr = call.arguments ? String(call.arguments) : (call.display ?? "");
    a.push(`### Tool ${t}：\`${call.name ?? "unknown"}\``);
    a.push("```json", argsStr, "```");
    a.push("**输出**：", "```text", call.result ?? "（无输出记录）", "```");
  }
  if (turn.finalReply) a.push("### 最终回复原文（逐字）", turn.finalReply);
  parts.push(a.join("\n\n"));
  return parts.join("\n");
}

async function postMemory(text, meta) {
  // REST 契约（mem0/server/main.py）：POST /memories 需要 messages 数组；
  // MCP 层的 text 参数是在 mcp_server.py 里包装成 messages 的，直连 REST 必须用 messages。
  const body = {
    messages: [{ role: "user", content: text }],
    infer: false,
    user_id: cfg.admin_user_id || "admin@mem0.dev",
    metadata: meta,
  };
  if (cfg.git_remote) body.git_remote = cfg.git_remote;
  if (cfg.project_id) body.project_id = cfg.project_id;
  const res = await fetch(`${restUrl}/memories`, {
    method: "POST",
    headers: { "X-API-Key": apiKey, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  return res.json();
}

// ---------- main ----------
let totalPosted = 0;
for (const { file, uuid } of jsonlFiles) {
  const st = state[uuid] ?? { offset: 0, turnSeq: 0 };
  const size = fs.statSync(file).size;
  if (st.offset >= size) continue; // nothing new

  const fd = fs.openSync(file, "r");
  const buf = Buffer.alloc(size - st.offset);
  fs.readSync(fd, buf, 0, buf.length, st.offset);
  fs.closeSync(fd);

  const recs = linesOf(buf).map(parse).filter(Boolean);
  // Build closed turns: a turn ends when the NEXT user message appears.
  const turns = [];
  let cur = null;
  for (const r of recs) {
    if (r.type === "message" && r.role === "user") {
      if (cur) {
        // Skip noise turns: system-injected caveat messages with no content.
        const isEmpty = !cur.thinkings.length && !cur.callOrder.length && !cur.finalReply;
        if (!isEmpty || !/^Caveat:/.test(cur.q)) turns.push(cur);
      }
      cur = {
        q: texts(r).join("\n"), userTs: r.timestamp,
        thinkings: [], calls: [], finalReply: "",
        callOrder: [], resultMap: new Map(),
      };
    } else if (!cur) {
      continue; // skip records before first user message
    } else if (r.type === "reasoning") {
      const th = (r.rawContent ?? []).filter((c) => c.type === "reasoning_text")
        .map((c) => c.text).join("\n").trim();
      if (th) cur.thinkings.push(th);
    } else if (r.type === "function_call") {
      const args = r.arguments ?? r.argumentsJson ?? r.providerData?.argumentsDisplayText ?? "";
      cur.callOrder.push({ name: r.name ?? r.toolName ?? null, arguments: args, callId: r.callId ?? null });
    } else if (r.type === "function_call_result") {
      cur.resultMap.set(r.callId, { name: r.name, out: r.output?.text ?? "" });
    } else if (r.type === "message" && r.role === "assistant") {
      const txt = texts(r).join("\n").trim();
      if (txt && txt !== "\n") cur.finalReply = txt; // keep the latest non-empty reply
    }
  }
  // attach results to calls in order
  for (const turn of turns) {
    turn.calls = turn.callOrder.map((c, i) => {
      const res = c.callId ? turn.resultMap.get(c.callId) : turn.resultMap.get(`#${i}`);
      return { name: c.name ?? res?.name, arguments: c.arguments, result: res?.out };
    });
  }
  if (cur) {
    // last (possibly still-open) turn: flush it too when session file is idle —
    // safest at next session start; mark as closed turn as well.
    const isEmpty = !cur.thinkings.length && !cur.callOrder.length && !cur.finalReply;
    if (!isEmpty || !/^Caveat:/.test(cur.q)) {
      cur.calls = cur.callOrder.map((c) => {
        const res = c.callId ? cur.resultMap.get(c.callId) : null;
        return { name: c.name ?? res?.name, arguments: c.arguments, result: res?.out };
      });
      turns.push(cur);
    }
  }

  // Post
  let posted = 0;
  let newOffset = st.offset + buf.length;
  for (const turn of turns) {
    if (!turn.q) continue;
    const text = renderTurn(turn);

    // --verify：取第一个有内容的轮次做端到端原样校验（POST→回读→字节比对→清理）
    if (VERIFY) {
      if (!turn.thinkings.length && !turn.calls.length && !turn.finalReply) continue;
      const ts = Date.now();
      const vmeta = {
        type: "conversation", session_id: `poster-verify-${ts}`,
        agent: "CodeBuddy", role: "turn", turn_seq: "1",
        created_at: new Date().toISOString(), source: "transcript-poster-verify",
      };
      const postedJson = await postMemory(text, vmeta);
      const id = postedJson?.results?.[0]?.id ?? postedJson?.id;
      if (!id) { console.error("[verify] FAIL: 服务端未返回 memory id"); process.exit(1); }
      const got = await fetch(`${restUrl}/memories/${id}`, {
        headers: { "X-API-Key": apiKey },
      });
      const gotJson = await got.json();
      const serverText = gotJson?.memory ?? gotJson?.text ?? "";
      const byteEqual = serverText === text;
      // JSONL 溯源：Q 与 Thinking 必须能在服务端文本中逐字找到
      const qIn = serverText.includes(turn.q.slice(0, Math.min(80, turn.q.length)));
      const thIn = !turn.thinkings.length ||
        serverText.includes(turn.thinkings[0].slice(0, Math.min(60, turn.thinkings[0].length)));
      const outIn = !turn.calls.length ||
        turn.calls.every((c) => !c.result || serverText.includes(c.result.slice(0, Math.min(60, c.result.length))));
      console.log(`[verify] 本地组装: ${text.length} chars / 服务端回读: ${serverText.length} chars`);
      console.log(`[verify] 字节级一致(infer=false, 零 LLM 链路): ${byteEqual ? "PASS" : "FAIL"}`);
      console.log(`[verify] JSONL 溯源 Q=${qIn ? "OK" : "FAIL"} Thinking=${thIn ? "OK" : "FAIL"} ToolOutput=${outIn ? "OK" : "FAIL"}`);
      try {
        await fetch(`${restUrl}/memories/${id}`, {
          method: "DELETE", headers: { "X-API-Key": apiKey },
        });
        console.log("[verify] 校验记忆已清理");
      } catch {}
      process.exit(byteEqual && qIn && thIn && outIn ? 0 : 1);
    }

    const sid = sessionId ?? `cb-${new Date(turn.userTs).toISOString().slice(0, 10).replace(/-/g, "")}-${uuid.slice(0, 6)}`;
    const meta = {
      type: "conversation", session_id: sid, agent: "CodeBuddy", role: "turn",
      people: ["joezxh"], turn_seq: String(st.turnSeq + 1),
      created_at: iso(turn.userTs), source: "transcript-poster",
    };
    if (DRY) {
      st.turnSeq += 1;
      console.log(`--- DRY turn#${st.turnSeq} (${text.length} chars) ---\n${text.slice(0, 600)}\n`);
    } else {
      try {
        await postMemory(text, meta);
        posted += 1; totalPosted += 1;
        st.turnSeq += 1; // 仅提交成功后才前进，失败轮次下次重试（避免瞬断导致永久丢轮/序号错乱）
      } catch (e) {
        console.error(`[mem0-poster] POST failed turn#${st.turnSeq + 1}: ${e.message}`);
        newOffset = st.offset; // do not advance on failure; retry next run
        break;
      }
    }
  }
  st.offset = DRY ? st.offset : newOffset;
  state[uuid] = st;
  console.log(`[mem0-poster] ${uuid}: ${posted} turns posted (offset -> ${st.offset})`);
}

if (!DRY && !VERIFY) fs.writeFileSync(statePath, JSON.stringify(state, null, 2));
console.log(`[mem0-poster] done, ${totalPosted} turns posted${DRY ? " (dry-run)" : ""}`);
