#!/usr/bin/env node
/**
 * mem0-tool-capture-hook — PostToolUse hook：把 IDE 原样传出的工具事件
 * （tool_input / tool_response，零 LLM）append 到 <cwd>/.mem0/tool-events.jsonl。
 *
 * 背景：IDE 会话转录已落盘本地（CodeBuddyExtension\Data\...\history\...\messages\*.json），
 * 而本 hook 把 PostToolUse 事件另作原样落盘（<cwd>/.mem0/tool-events.jsonl），与转录通道
 * 互补，覆盖工具层的逐字留痕。本 hook 从 stdin 读取 hook
 * 事件 JSON，原样提取关键字段落盘；由 mem0-setup 自动接线（PostToolUse, matcher .*）。
 *
 * 兼容多种 payload 命名（Claude Code 风格 tool_name/tool_input/tool_response 与
 * 常见变体），解析失败时原样保存 raw 以保证不丢数据。
 */

import fs from "node:fs";
import path from "node:path";

const chunks = [];
for await (const chunk of process.stdin) chunks.push(chunk);
const raw = Buffer.concat(chunks).toString("utf8");
if (!raw.trim()) process.exit(0);

let evt = null;
try { evt = JSON.parse(raw); } catch { /* 非 JSON：原样保存 raw */ }

const cwd = evt?.cwd ?? evt?.workspace ?? evt?.project_dir ?? process.cwd();
const out = path.join(cwd, ".mem0", "tool-events.jsonl");
try { fs.mkdirSync(path.dirname(out), { recursive: true }); } catch { process.exit(0); }

const pick = (...keys) => {
    for (const k of keys) {
        if (evt && evt[k] !== undefined && evt[k] !== null) return evt[k];
    }
    return null;
};

const rec = {
    ts: new Date().toISOString(),
    session_id: pick("session_id", "sessionId", "conversation_id", "conversationId"),
    tool_name: pick("tool_name", "toolName", "tool"),
    tool_input: pick("tool_input", "toolInput", "input", "arguments"),
    tool_response: pick("tool_response", "toolResponse", "output", "response", "result"),
};
// 绝对不丢数据：无论解析成败都带上原始包（截断到 256KB 防御异常膨胀）
rec.raw = raw.length > 256 * 1024 ? raw.slice(0, 256 * 1024) + "…(truncated)" : raw;

try {
    fs.appendFileSync(out, JSON.stringify(rec) + "\n");
} catch { /* 落盘失败不影响 IDE 工具执行 */ }
process.exit(0);
