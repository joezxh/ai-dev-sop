#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
codebuddy-session-export.py — 从 CodeBuddy IDE 本地消息缓存导出当前会话全文。

CodeBuddy IDE 不在 ~/.codebuddy/projects/*.jsonl 写转录（那是 CLI 专用），
而是把逐条消息存到（按账号/工作区分目录）：
  %LOCALAPPDATA%/CodeBuddyExtension/Data/<userId>/CodeBuddyIDE/<userId>/history/<workspaceHash>/<sessionId>/messages/<msgId>.json
每条消息: {"role","message"(内嵌 JSON: {"role","content":[{type:"reasoning"|"text"|"tool_use"|...}]}),"id","extra","createdAt"}

本脚本自动发现所有账号根目录，导出"最近修改且消息数>=5"的会话（或指定 ws/sid）：
  - session-<id>.md   : 人类可读全文（用户/助手/思考/工具调用/工具结果）
  - session-<id>.json : 原始结构（便于二次处理/入库）

用法:
  python codebuddy-session-export.py                 # 导出当前(最近实质)会话
  python codebuddy-session-export.py <ws> <sid>      # 导出指定会话
  python codebuddy-session-export.py list            # 列出所有账号/工作区/会话
"""
import os, json, sys, glob, datetime

DATA_ROOT = os.path.join(os.environ.get('LOCALAPPDATA', ''), 'CodeBuddyExtension', 'Data')


def history_roots():
    roots = []
    if not os.path.isdir(DATA_ROOT):
        return roots
    for u in os.listdir(DATA_ROOT):
        h = os.path.join(DATA_ROOT, u, 'CodeBuddyIDE', u, 'history')
        if os.path.isdir(h):
            roots.append(h)
    return roots


def resolve_root(ws, sid):
    for r in history_roots():
        if os.path.isdir(os.path.join(r, ws, sid, 'messages')):
            return r
    return None


def list_workspaces():
    ws = set()
    for r in history_roots():
        for d in os.listdir(r):
            if os.path.isdir(os.path.join(r, d)):
                ws.add(d)
    return list(ws)


def list_sessions(ws):
    out = []
    for r in history_roots():
        hp = os.path.join(r, ws)
        if not os.path.isdir(hp):
            continue
        for sid in os.listdir(hp):
            mp = os.path.join(hp, sid, 'messages')
            if os.path.isdir(mp):
                n = len([f for f in os.listdir(mp) if f.endswith('.json')])
                out.append((sid, n, os.path.getmtime(os.path.join(hp, sid))))
    return sorted(out, key=lambda x: x[2])


def find_current():
    # 在所有账号/工作区中选取"最近修改且消息数>=5"的会话（排除游离/失败的单条会话）。
    all_s = []
    for r in history_roots():
        for d in os.listdir(r):
            hp = os.path.join(r, d)
            if not os.path.isdir(hp):
                continue
            for sid in os.listdir(hp):
                mp = os.path.join(hp, sid, 'messages')
                if os.path.isdir(mp):
                    n = len([f for f in os.listdir(mp) if f.endswith('.json')])
                    all_s.append((r, d, sid, n, os.path.getmtime(os.path.join(hp, sid))))
    if not all_s:
        return None
    all_s.sort(key=lambda x: x[4], reverse=True)
    substantial = [s for s in all_s if s[3] >= 5]
    pick = substantial[0] if substantial else all_s[0]
    return (pick[1], pick[2], pick[0])  # ws, sid, root


def render_content(content, out):
    if content is None:
        return
    if isinstance(content, str):
        out.append(content)
        return
    if not isinstance(content, list):
        out.append(json.dumps(content, ensure_ascii=False))
        return
    for blk in content:
        if not isinstance(blk, dict):
            out.append(str(blk))
            continue
        t = blk.get('type')
        txt = blk.get('text') or blk.get('content') or ''
        if isinstance(txt, list):
            render_content(txt, out)
            continue
        txt = '' if txt is None else str(txt)
        if t in ('reasoning', 'thinking'):
            out.append('\n#### 💭 思考 (Thinking)\n\n' + txt + '\n')
        elif t == 'text':
            out.append(txt + '\n')
        elif t in ('tool_use', 'tool_call', 'tool_calls'):
            name = blk.get('name') or blk.get('tool_name') or blk.get('function', {}).get('name', '?')
            inp = blk.get('input') or blk.get('parameters') or blk.get('args') or ''
            out.append('\n**🔧 工具调用**: `' + str(name) + '`\n')
            if inp:
                try:
                    s = json.dumps(inp, ensure_ascii=False)
                except Exception:
                    s = str(inp)
                out.append('```json\n' + s[:3000] + '\n```\n')
        elif t in ('tool_result', 'tool', 'tool_response'):
            out.append('\n**🔧 工具结果**\n\n' + txt[:6000] + '\n')
        elif t in ('image', 'image_url'):
            out.append('[图片]\n')
        else:
            out.append(txt + '\n')


def export_session(ws, sid, outdir):
    root = resolve_root(ws, sid)
    if not root:
        print('未找到会话:', ws, sid)
        sys.exit(1)
    mp = os.path.join(root, ws, sid, 'messages')
    files = glob.glob(os.path.join(mp, '*.json'))
    msgs = []
    for f in files:
        try:
            d = json.load(open(f, encoding='utf-8'))
        except Exception:
            continue
        role = d.get('role')
        created = d.get('createdAt', 0)
        raw = d.get('message')
        inner = None
        if isinstance(raw, str):
            try:
                inner = json.loads(raw)
            except Exception:
                inner = {'content': [{'type': 'text', 'text': raw}]}
        elif isinstance(raw, dict):
            inner = raw
        content = inner.get('content') if inner else None
        msgs.append((created, role, content))
    msgs.sort(key=lambda x: (x[0] if isinstance(x[0], (int, float)) else str(x[0])))

    md = []
    md.append('# CodeBuddy IDE 会话导出\n\n'
              f'- workspace: `{ws}`\n- session: `{sid}`\n'
              f'- messages: {len(msgs)}\n'
              f'- exported: {datetime.datetime.now().isoformat(timespec="seconds")}\n')
    for created, role, content in msgs:
        if role == 'user':
            md.append('\n---\n\n### 👤 用户\n')
            if isinstance(content, str):
                md.append(content + '\n')
            else:
                render_content(content, md)
        elif role == 'assistant':
            md.append('\n### 🤖 助手\n')
            render_content(content, md)
        else:
            md.append(f'\n### [{role}]\n')
            render_content(content, md)

    os.makedirs(outdir, exist_ok=True)
    md_path = os.path.join(outdir, f'session-{sid}.md')
    json_path = os.path.join(outdir, f'session-{sid}.json')
    with open(md_path, 'w', encoding='utf-8') as fh:
        fh.write('\n'.join(md))
    with open(json_path, 'w', encoding='utf-8') as fh:
        json.dump([{'role': m[1], 'content': m[2], 'createdAt': m[0]} for m in msgs],
                  fh, ensure_ascii=False, indent=1)
    print(f'✅ 已导出 {len(msgs)} 条消息:\n   {md_path}\n   {json_path}')


if __name__ == '__main__':
    if not history_roots():
        print('未找到 CodeBuddy IDE 本地数据目录:', DATA_ROOT)
        sys.exit(1)
    args = sys.argv[1:]
    if args and args[0] == 'list':
        for r in history_roots():
            print('ROOT', r)
            for ws in os.listdir(r):
                if os.path.isdir(os.path.join(r, ws)):
                    for sid, n, t in list_sessions(ws):
                        print('   ws=', ws, ' sid=', sid, ' msgs=', n,
                              datetime.datetime.fromtimestamp(t))
        sys.exit(0)
    outdir = r'D:\projects\ai-dev-sop\.mem0\sessions'
    if len(args) >= 2:
        ws, sid = args[0], args[1]
    else:
        cur = find_current()
        if not cur:
            print('未发现任何会话')
            sys.exit(1)
        ws, sid, _ = cur
        print(f'自动选择当前会话: ws={ws} sid={sid}')
    export_session(ws, sid, outdir)
