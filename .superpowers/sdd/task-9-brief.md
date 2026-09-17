# Task 9 (extracted from implementation plan)

**Files:**
- Create: `mem0/server/dashboard/src/app/(root)/dashboard/playground/page.tsx`

- [ ] **Step 1: Create the page** — Add/Search modes, role-toggling message rows, `infer` switch, instructions input (injected as `prompt`), Output/Code views, sandbox banner:

```tsx
"use client";

import React, { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/use-toast";
import { getErrorMessage } from "@/lib/error-message";
import { api } from "@/utils/api";
import { MEMORY_ENDPOINTS } from "@/utils/api-endpoints";

type Msg = { role: "user" | "assistant"; content: string };

const newId = () => `sandbox-${Math.random().toString(36).slice(2, 10)}`;

export default function PlaygroundPage() {
  const [mode, setMode] = useState<"add" | "search">("add");
  const [messages, setMessages] = useState<Msg[]>([{ role: "user", content: "" }]);
  const [query, setQuery] = useState("");
  const [infer, setInfer] = useState(true);
  const [instructions, setInstructions] = useState("");
  const [output, setOutput] = useState("");
  const [running, setRunning] = useState(false);

  const sandboxUserId = newId();

  const codeSnippet =
    mode === "add"
      ? `curl -X POST $API/memories -H "X-API-Key: $KEY" -d '{"messages": ${JSON.stringify(messages)}, "infer": ${infer}, "user_id": "${sandboxUserId}"${instructions ? `, "prompt": ${JSON.stringify(instructions)}` : ""}}'`
      : `curl -X POST $API/search -H "X-API-Key: $KEY" -d '{"query": ${JSON.stringify(query)}, "user_id": "${sandboxUserId}"}'`;

  async function run() {
    setRunning(true);
    try {
      if (mode === "add") {
        const res = await api.post(MEMORY_ENDPOINTS.BASE, {
          messages: messages.filter((m) => m.content.trim()),
          infer,
          user_id: sandboxUserId,
          ...(instructions.trim() ? { prompt: instructions.trim() } : {}),
        });
        setOutput(JSON.stringify(res.data, null, 2));
      } else {
        const res = await api.post("/search", { query, user_id: sandboxUserId });
        setOutput(JSON.stringify(res.data, null, 2));
      }
    } catch (error) {
      toast({ title: "Playground call failed", description: getErrorMessage(error), variant: "destructive" });
    } finally {
      setRunning(false);
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-xl font-semibold font-fustat">Playground</h1>
      <div className="rounded border border-amber-300 bg-amber-50 dark:bg-amber-950/30 px-3 py-2 text-xs">
        沙盒数据使用隔离的 user_id（<code>{sandboxUserId}</code> 前缀 sandbox-），不会进入你的项目记忆。
      </div>

      <div className="flex gap-2">
        {(["add", "search"] as const).map((m) => (
          <Button key={m} size="sm" variant={m === mode ? "default" : "outline"} onClick={() => setMode(m)}>
            {m === "add" ? "Add Memories" : "Search Memories"}
          </Button>
        ))}
      </div>

      {mode === "add" ? (
        <Card className="border-memBorder-primary">
          <CardHeader><CardTitle className="text-sm">会话消息</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            {messages.map((m, i) => (
              <div key={i} className="flex gap-2 items-center">
                <button
                  className="px-2 py-1 text-xs border rounded border-memBorder-primary w-20"
                  onClick={() =>
                    setMessages((prev) => prev.map((x, j) => (j === i ? { ...x, role: x.role === "user" ? "assistant" : "user" } : x)))
                  }
                >
                  {m.role}
                </button>
                <Input
                  value={m.content}
                  onChange={(e) => setMessages((prev) => prev.map((x, j) => (j === i ? { ...x, content: e.target.value } : x)))}
                  placeholder={m.role === "user" ? "用户输入…" : "助手回复…"}
                />
                <Button variant="ghost" size="sm" onClick={() => setMessages((prev) => prev.filter((_, j) => j !== i))}>
                  ✕
                </Button>
              </div>
            ))}
            <Button variant="outline" size="sm" onClick={() => setMessages((prev) => [...prev, { role: "user", content: "" }])}>
              + 添加消息
            </Button>
            <div className="flex items-center gap-2 pt-2">
              <input id="infer" type="checkbox" checked={infer} onChange={(e) => setInfer(e.target.checked)} />
              <Label htmlFor="infer" className="text-xs">Extraction（开=抽取事实，关=逐字存储）</Label>
            </div>
            <div className="space-y-1">
              <Label className="text-xs">Custom Instructions（试运行）</Label>
              <Textarea value={instructions} onChange={(e) => setInstructions(e.target.value)} rows={2}
                placeholder="例如：只存储技术偏好，忽略闲聊" />
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-memBorder-primary">
          <CardHeader><CardTitle className="text-sm">检索</CardTitle></CardHeader>
          <CardContent className="space-y-3">
            <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="搜索查询…" />
          </CardContent>
        </Card>
      )}

      <div className="flex items-center gap-3">
        <Button onClick={run} disabled={running}>{running ? "Running…" : "Run"}</Button>
      </div>

      {output && (
        <Card className="border-memBorder-primary">
          <CardHeader><CardTitle className="text-sm">Output</CardTitle></CardHeader>
          <CardContent>
            <pre className="text-xs bg-surface-default-secondary p-3 rounded font-mono overflow-x-auto max-h-80">{output}</pre>
            <p className="text-xs text-onSurface-default-tertiary mt-2">Code: <code>{codeSnippet}</code></p>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Rebuild + verify** — `GET /dashboard/playground` → 200；在页面 Run 一条 add（AUTH_DISABLED 环境 admin 权限）→ Output 返回 `results`。

- [ ] **Step 3: Commit** — `cd d:/projects/ai-dev-sop/mem0 && git add "server/dashboard/src/app/(root)/dashboard/playground" && git commit --no-verify -m "feat(dashboard): playground sandbox (add/search, infer switch, instructions)"`

---

