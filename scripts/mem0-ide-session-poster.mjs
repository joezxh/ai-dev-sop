#!/usr/bin/env node
/**
 * mem0-ide-session-poster — 纯脚本（零 LLM）把 CodeBuddy IDE 本地落盘的会话转录
 * 逐字组装为 Q/A Markdown 并直投 mem0 REST（infer=false，原文留痕）。
 *
 * 数据源（落盘事实）：
 *   %LOCALAPPDATA%/CodeBuddyExtension/Data/<userId>/CodeBuddyIDE/<userId>/
 *     history/<workspaceHash>/<sessionId>/messages/<msgId>.json
 *   每条消息 JSON：{ role, message(JSON字符串), id, extra?(JSON字符串), createdAt? }
 *     - message 解析后 content 块：text / reasoning / tool-call(toolCallId,toolName,args)
 *     - role=tool 的消息 content 块：tool-result(toolCallId, result)
 *   顺序由 <sessionId>/index.json 的 messages[]（id/role/isComplete）决定
 *
 * 断点续传：状态存 <workspace>/.mem0/.ide-poster-state.json（按 sessionId 记已提交轮次数）。
 * mem0 按内容哈希去重，重复提交安全。
 *
 * 用法（由 hook 在本地会话产生时调用，flush 最近产生的会话）：
 *   node mem0-ide-session-poster.mjs [--workspace <dir>] [--session <id>] \
 *                                    [--rest-url <url>] [--dry-run] [--verify]
 */

import fs from "node:fs";
import path from "node:path";
import os from "node:os";

const argv = process.argv.slice(2);
const arg = (n, d) => {
  const i = argv.indexOf(n);
  return i >= 0 && argv[i + 1] ? argv[i + 1] : d;
};
const DRY = argv.includes("--dry-run");
const VERIFY = argv.includes("--verify");
const LIMIT = parseInt(arg("--limit", "0"), 10) || 0;
const workspace = path.resolve(arg("--workspace", process.cwd()));
const restUrl = arg("--rest-url", "http://localhost:8002").replace(/\/$/, "");
const sessionArg = arg("--session", null);

// --- 配置 ---
const mem0Dir = path.join(workspace, ".mem0");
const cfgPath = path.join(mem0Dir, "mem0.config.json");
if (!fs.existsSync(cfgPath)) {
  console.error(`[ide-poster] config 缺失: ${cfgPath}（找不到则静默退出）`);
  process.exit(1);
}
const cfg = JSON.parse(fs.readFileSync(cfgPath, "utf8"));
const apiKey = cfg.api_key;
const gitRemote = cfg.git_remote;
const projectId = cfg.project_id || "ai-dev-sop";
const adminUserId = cfg.admin_user_id || "admin@mem0.dev";
if (!apiKey) {
  console.error("[ide-poster] api_key 缺失");
  process.exit(1);
}

// --- 落盘路径解析 ---
const localApp = process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local");
// 注意：这里用纯字符串拼接（不要用 path.join，否则 '*' 会被转义成 '*\*' 导致无法展开）
const histRoot = `${localApp}\\CodeBuddyExtension\\Data\\*\\CodeBuddyIDE\\*\\history`;

// 展开路径中的 '*'（userId 两段）
function expandStars(p) {
  if (!p.includes("*")) return [p];
  const segs = p.split(path.sep);
  for (let i = 0; i < segs.length; i++) {
    if (segs[i] === "*") {
      const prefix = segs.slice(0, i).join(path.sep);
      const rest = segs.slice(i + 1).join(path.sep);
      let entries = [];
      try {
        entries = fs.readdirSync(prefix, { withFileTypes: true });
      } catch {
        return [];
      }
      const res = [];
      for (const e of entries) {
        if (e.isDirectory()) res.push(...expandStars(path.join(prefix, e.name, rest)));
      }
      return res;
    }
  }
  return [p];
}

function listDirs(p) {
  try {
    return fs
      .readdirSync(p, { withFileTypes: true })
      .filter((d) => d.isDirectory())
      .map((d) => path.join(p, d.name));
  } catch {
    return [];
  }
}

function dirMtime(p) {
  let mt = 0;
  try {
    for (const f of fs.readdirSync(p)) {
      const s = fs.statSync(path.join(p, f));
      if (s.mtimeMs > mt) mt = s.mtimeMs;
    }
  } catch {}
  return mt;
}

// 收集所有会话目录 history/<ws>/<sid>
const historyDirs = expandStars(histRoot);
let sessionDirs = [];
for (const hd of historyDirs) {
  for (const ws of listDirs(hd)) {
    // ws = <workspaceHash>，其下为 <sessionId>
    for (const sd of listDirs(ws)) {
      const md = path.join(sd, "messages");
      if (fs.existsSync(md)) sessionDirs.push(sd);
    }
  }
}
if (sessionArg) {
  sessionDirs = sessionDirs.filter((sd) => path.basename(sd) === sessionArg);
}

// --- 状态（断点续传）---
const statePath = path.join(mem0Dir, ".ide-poster-state.json");
let state = {};
try {
  state = JSON.parse(fs.readFileSync(statePath, "utf8"));
} catch {}
const nowMs = Date.now();
// 首次运行：只处理最近 7 天内、且最多 10 个最新会话（避免历史回填风暴）
const firstRun = !state.lastRun;
const lastRun = state.lastRun ? new Date(state.lastRun).getTime() : nowMs - 7 * 864e5;

// 候选会话：自上次运行以来有修改（或显式 --session）
let candidates = sessionDirs;
if (!sessionArg) {
  candidates = candidates
    .map((sd) => ({ sd, mtime: dirMtime(path.join(sd, "messages")) }))
    .filter((x) => x.mtime > lastRun)
    .sort((a, b) => b.mtime - a.mtime)
    .map((x) => x.sd);
  if (firstRun) candidates = candidates.slice(0, 10);
}
if (argv.includes("--debug")) {
  console.error(
    `DBG histDirs=${historyDirs.length} sessionDirs=${sessionDirs.length} candidates=${candidates.length} sample=${JSON.stringify(
      sessionDirs.slice(0, 3).map((p) => path.basename(p))
    )}`
  );
}
if (!candidates.length) {
  console.log("[ide-poster] 无新会话需要提交");
  process.exit(0);
}

// --- 消息解析 ---
function safeParse(s) {
  if (typeof s === "object") return s;
  try {
    return JSON.parse(s);
  } catch {
    return null;
  }
}

function stringifyResult(r) {
  if (r == null) return "";
  if (typeof r === "string") return r;
  if (typeof r === "object") {
    if (typeof r.result === "string") return r.result;
    if (typeof r.content === "string") return r.content;
    if (typeof r.output === "string") return r.output;
    if (r.result && typeof r.result === "object") {
      if (typeof r.result.content === "string") return r.result.content;
      if (typeof r.result.output === "string") return r.result.output;
    }
    try {
      return JSON.stringify(r, null, 2);
    } catch {
      return String(r);
    }
  }
  return String(r);
}

function foldIfLong(text) {
  const lines = text.split("\n");
  const M = lines.length;
  if (M <= 800) return { body: text, folded: false, total: M };
  const head = lines.slice(0, 50).join("\n");
  const tail = lines.slice(-50).join("\n");
  return {
    body: `${head}\n…（中间省略 ${M - 100} 行，共 ${M} 行）…\n${tail}`,
    folded: true,
    total: M,
  };
}

// 去除 IDE 注入的巨型系统上下文包裹块（规则/记忆/工程指南等每条重复），
// 仅保留用户真实输入（<user_query> 内容 + @文件引用），避免 mem0 存储爆炸。
function cleanQ(full) {
  let inner = null;
  const um = full.match(/<user_query>([\s\S]*?)<\/user_query>/);
  if (um) inner = um[1].trim();
  let q = inner !== null ? inner : full;
  const wrappers = [
    "user_info",
    "always_applied_user_rules",
    "rules",
    "additional_data",
    "memories",
    "project_guidance",
    "git_status",
    "project_layout",
    "system_reminder",
    "cb_summary",
    "previous_user_message",
  ];
  for (const w of wrappers) {
    try {
      q = q.replace(new RegExp(`<${w}>[\\s\\S]*?<\\/${w}>`, "g"), "");
    } catch {}
  }
  q = q.replace(/\r\n/g, "\n").replace(/\n{3,}/g, "\n\n").trim();
  return q;
}

// 归一化 message.content 为内容块数组（兼容 string / 单块对象 / 块数组 三种形态）
function asContentBlocks(parsed) {
  const c = parsed && parsed.content;
  if (Array.isArray(c)) return c;
  if (typeof c === "string") return [{ type: "text", text: c }];
  if (c && typeof c === "object" && c.type) return [c];
  return [];
}

function extractUserText(raw, parsed) {
  const full = asContentBlocks(parsed)
    .filter((c) => c.type === "text")
    .map((c) => c.text || "")
    .join("\n")
    .trim();
  let text = cleanQ(full);
  if (!text) text = full; // 回退，避免丢轮
  // 附加 @文件引用（来自顶部 references）
  const refs = raw && raw.references;
  if (Array.isArray(refs)) {
    const files = [];
    for (const r of refs) {
      if (r.type === "file" && Array.isArray(r.filePaths)) {
        for (const fp of r.filePaths) files.push(fp);
      }
    }
    if (files.length) {
      text += `\n\n> 系统附加上下文：@${files.join(" @")}`;
    }
  }
  return text;
}

function buildTurns(sd, msgList) {
  const resultMap = new Map(); // toolCallId -> result text
  const msgs = [];
  for (const m of msgList) {
    const fp = path.join(sd, "messages", m.id + ".json");
    if (!fs.existsSync(fp)) continue;
    let raw;
    try {
      raw = JSON.parse(fs.readFileSync(fp, "utf8"));
    } catch {
      continue;
    }
    const role = raw.role;
    const parsed = safeParse(raw.message);
    const extra = safeParse(raw.extra);
    msgs.push({ role, raw, parsed, extra, createdAt: raw.createdAt, isComplete: m.isComplete });
  }
  // 先收集 tool 结果（role=tool 消息的 tool-result），以及 assistant extra.toolStatus
  for (const msg of msgs) {
    if (msg.role === "tool" && msg.parsed) {
      for (const b of asContentBlocks(msg.parsed)) {
        if (b.type === "tool-result" && b.toolCallId) {
          resultMap.set(b.toolCallId, stringifyResult(b.result));
        }
      }
    }
    if (msg.role === "assistant" && msg.extra && msg.extra.toolStatus) {
      for (const [cid, v] of Object.entries(msg.extra.toolStatus)) {
        if (v && v.result) resultMap.set(cid, stringifyResult(v.result));
      }
    }
  }
  // 组装 turns（user → assistant/tool … 直到下一个 user）
  const turns = [];
  let cur = null;
  for (const msg of msgs) {
    if (msg.role === "user") {
      if (cur) turns.push(cur);
      cur = {
        q: extractUserText(msg.raw, msg.parsed),
        thinkings: [],
        calls: [],
        finalReply: "",
        createdAt: msg.createdAt,
      };
    } else if (!cur) {
      continue;
    } else if (msg.role === "assistant" && msg.parsed) {
      for (const b of asContentBlocks(msg.parsed)) {
        if (b.type === "reasoning") {
          if (b.text) cur.thinkings.push(b.text);
        } else if (b.type === "tool-call") {
          cur.calls.push({ callId: b.toolCallId, name: b.toolName, args: b.args, result: null });
        } else if (b.type === "text") {
          const t = (b.text || "").trim();
          if (t) cur.finalReply = t;
        }
      }
    }
  }
  if (cur) turns.push(cur);
  // 回填调用结果
  for (const t of turns) {
    for (const c of t.calls) {
      const r = resultMap.get(c.callId);
      if (r) c.result = r;
    }
  }
  return turns;
}

function iso(s) {
  if (s) {
    const d = new Date(s);
    if (!isNaN(d.getTime())) return d.toISOString();
  }
  return new Date().toISOString();
}

function renderTurn(sid, seq, t, createdAt) {
  const parts = [];
  parts.push(`Q: ${t.q}`);
  parts.push("");
  parts.push("A: ");
  let ti = 0;
  for (const th of t.thinkings) {
    ti++;
    parts.push(`### Thinking ${ti}（逐字全文）`);
    parts.push(th);
    parts.push("");
  }
  let ci = 0;
  for (const c of t.calls) {
    ci++;
    parts.push(`### Tool ${ci}：${c.name}`);
    parts.push("```json");
    parts.push(JSON.stringify(c.args || {}, null, 2));
    parts.push("```");
    const out = c.result || "(无输出/未捕获)";
    const f = foldIfLong(out);
    parts.push(`**输出**：`);
    parts.push("```text");
    parts.push(f.body);
    parts.push("```");
    parts.push("");
  }
  if (t.finalReply) {
    parts.push("### 最终回复原文（逐字）");
    parts.push(t.finalReply);
    parts.push("");
  }
  parts.push("### 完成结果");
  parts.push(
    `会话 ${sid} 第 ${seq} 轮：用户提问 + ${
      t.thinkings.length
    } 段思考 + ${t.calls.length} 次工具调用${
      t.finalReply ? "，含最终回复" : "（进行中，暂未提交）"
    }。`
  );
  return parts.join("\n");
}

// --- 直投 mem0 ---
// 所有 HTTP 请求统一加超时，避免 embedding 慢/卡死时进程无限挂起
const REQ_TIMEOUT = 30000;
function fetchWithTimeout(url, options = {}) {
  const ac = new AbortController();
  const timer = setTimeout(() => ac.abort(), REQ_TIMEOUT);
  return fetch(url, { ...options, signal: ac.signal }).finally(() => clearTimeout(timer));
}

async function postMemory(text, meta) {
  // REST 契约（mem0/server/main.py）：POST /memories 用 messages 数组；
  // git_remote/project_id 走顶层字段（服务端据此写入 metadata）。
  const body = {
    messages: [{ role: "user", content: text }],
    infer: false,
    user_id: adminUserId,
    metadata: meta,
  };
  if (gitRemote) body.git_remote = gitRemote;
  if (projectId) body.project_id = projectId;
  const res = await fetchWithTimeout(`${restUrl}/memories`, {
    method: "POST",
    headers: { "X-API-Key": apiKey, "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  return res.json();
}

// 全量上送：零 LLM 直投，不做任何字符截断，完整原文入库（mem0 服务端按内容哈希去重）。被 502 拒绝时重试一次，确保会话不丢。
async function postMemorySafe(text, meta) {
  try {
    return await postMemory(text, meta);
  } catch (e) {
    if (/502|provider_bad_request|malformed/i.test(e.message)) {
      console.error(`[ide-poster] 提交被拒(${e.message})，重试一次`);
      return await postMemory(text, meta);
    }
    throw e;
  }
}

async function run() {
  let totalPosted = 0;
  state.sessions = state.sessions || {};
  for (const sd of candidates) {
    const sid = path.basename(sd);
    try {
      const idxPath = path.join(sd, "index.json");
      if (!fs.existsSync(idxPath)) continue;
      let idx;
      try {
        idx = JSON.parse(fs.readFileSync(idxPath, "utf8"));
      } catch {
        continue;
      }
      const msgList = idx.messages || [];
      if (!msgList.length) continue;
      const turns = buildTurns(sd, msgList);
      const st = state.sessions[sid] || { turnSeq: 0 };
      let seq = st.turnSeq;
      let posted = 0;
      for (let i = 0; i < turns.length; i++) {
        if (i < st.turnSeq) continue; // 已提交
        const t = turns[i];
        if (!t.q) continue;
        const isLast = i === turns.length - 1;
        if (isLast && !t.finalReply && turns.length > 1) break; // 进行中，暂不提交，也不前进
        const text = renderTurn(sid, i + 1, t, t.createdAt);
        if (LIMIT && posted >= LIMIT) break;
        const meta = {
          type: "conversation",
          session_id: sid,
          agent: "CodeBuddy",
          role: "turn",
          people: ["joezxh"],
          turn_seq: String(i + 1),
          created_at: iso(t.createdAt),
          source: "ide-session-poster",
        };
        if (DRY || VERIFY) {
          console.log(`\n===== [${sid}] turn ${i + 1} (dry/verify) len=${text.length} =====`);
          console.log(text.slice(0, 2000) + (text.length > 2000 ? "\n…(截断预览)" : ""));
        }
        if (DRY) {
          posted++;
          seq = i + 1;
          continue;
        }
        if (VERIFY) {
          const ok = await verifyOne(text, meta, sid, i + 1);
          process.exitCode = ok ? 0 : 1;
          return; // 正常退出，避免异步上下文内 process.exit 触发 uv 断言
        }
        try {
          await postMemorySafe(text, meta);
          posted++;
          totalPosted++;
          seq = i + 1; // 仅提交成功后才前进；失败轮次不前进，下次运行重试（避免瞬断导致永久丢轮）
        } catch (e) {
          console.error(`[ide-poster] ${sid} turn ${i + 1} 提交失败: ${e.message}`);
          break;
        }
      }
      state.sessions[sid] = { turnSeq: seq };
      if (posted) console.log(`[ide-poster] ${sid}: ${posted} 轮已提交（累计 turnSeq=${seq}）`);
    } catch (e) {
      console.error(`[ide-poster] 会话 ${sid} 处理异常，已跳过: ${e.message}`);
    }
  }
  // 更新状态
  if (!DRY) {
    // 仅在确有提交时推进 lastRun：避免“全部提交失败却推进 lastRun”导致该会话被 mtime 过滤永久跳过
    if (totalPosted > 0) {
      state.lastRun = new Date().toISOString();
    }
    fs.writeFileSync(statePath, JSON.stringify(state, null, 2), "utf8");
  }
  console.log(`[ide-poster] 完成，本次共提交 ${totalPosted} 轮。`);
  process.exit(0); // 显式退出，避免 fetch keep-alive 连接使事件循环无法排空而挂起
}

// 校验（零 LLM 链路）：提交一条 → 回读 → 字节比对 → 清理
async function verifyOne(text, meta, sid, seq) {
  const postedJson = await postMemory(text, meta);
  const id = postedJson?.results?.[0]?.id ?? postedJson?.id;
  if (!id) {
    console.error("[verify] FAIL: 服务端未返回 memory id");
    process.exit(1);
  }
  const got = await fetchWithTimeout(`${restUrl}/memories/${id}`, { headers: { "X-API-Key": apiKey } });
  const gotJson = await got.json();
  const serverText = gotJson?.memory ?? gotJson?.text ?? "";
  const byteEqual = serverText === text;
  const qIn = serverText.includes(text.slice(0, Math.min(80, text.length)));
  console.log(`[verify] 本地组装: ${text.length} chars / 服务端回读: ${serverText.length} chars`);
  console.log(`[verify] 字节级一致(infer=false, 零 LLM 链路): ${byteEqual ? "PASS" : "FAIL"}`);
  console.log(`[verify] Q 溯源: ${qIn ? "OK" : "FAIL"}`);
  try {
    await fetchWithTimeout(`${restUrl}/memories/${id}`, { method: "DELETE", headers: { "X-API-Key": apiKey } });
    console.log("[verify] 校验记忆已清理");
  } catch {}
  return byteEqual && qIn;
}

run().catch((e) => {
  console.error("[ide-poster] 致命错误:", e);
  process.exit(1);
});
