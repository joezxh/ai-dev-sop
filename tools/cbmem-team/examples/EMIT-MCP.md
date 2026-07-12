# cbmem-team: emitting client mcp.json

Once `cbmem-team` is running, each developer still needs a `~/.cursor/mcp.json`
entry that points Cursor at the server. This entry contains a **JWT bearer
token** that expires — so re-emitting the file is a recurring task.

The two scripts under `examples/` automate that flow end-to-end:

| Script | OS | Purpose |
|---|---|---|
| `cbmem-emit-mcp.ps1` | Windows / pwsh 7 | mint + write `~/.cursor/mcp.json` |
| `cbmem-emit-mcp.sh` | macOS / Linux | mint + write `~/.cursor/mcp.json` |

## What they do

1. Call `POST /admin/users/<user>/token?ttl=<ttl>` with `X-Admin-Token` to mint
   a JWT. The default TTL is `4320h` (180 days).
2. Load `~/.cursor/mcp.json` (creating it if missing).
3. Upsert the `cbmem-team` entry with the fresh JWT + URL.
4. Run a `tools/list` smoke test to confirm the server accepts the token.
5. Print a one-line summary suitable for shell pipelines.

## Quick start

```bash
# Linux/macOS
BASE=http://192.168.100.83:8787 \
ADMIN_TOKEN=$YOUR_ADMIN_TOKEN \
USER=tianque \
PROJECT_SERVER_PATH=/var/lib/cbmem-team/users/tianque/repos/github.com__you_ai-dev-sop \
bash examples/cbmem-emit-mcp.sh
```

```powershell
# Windows
pwsh examples/cbmem-emit-mcp.ps1 `
  -Base http://192.168.100.83:8787 `
  -AdminToken $YOUR_ADMIN_TOKEN `
  -User tianque `
  -ProjectServerPath /var/lib/cbmem-team/users/tianque/repos/github.com__you_ai-dev-sop `
  -Ttl 4320h
```

## End-user refresh (no admin required)

The end user can swap a still-valid JWT for a fresh one without bothering the
admin, using `POST /refresh`:

```bash
curl -sS -X POST "http://192.168.100.83:8787/refresh?ttl=4320h" \
  -H "Authorization: Bearer $OLD_JWT"
```

Note: `/refresh` does **not** require the admin token — it only verifies the
caller's existing JWT. This is what lets a `cbmem-emit-mcp.*` script run from
a developer's laptop using only their own (still valid) token.

## Server endpoints used

| Endpoint | Method | Auth | Purpose |
|---|---|---|---|
| `/admin/users/:id/token?ttl=` | POST | `X-Admin-Token` | mint a brand-new JWT |
| `/refresh?ttl=` | POST | Bearer JWT | swap a still-valid JWT for a fresh one |
| `/mcp?as=...&project=...` | POST | Bearer JWT | MCP request itself |
| `/admin/repos/clone` | POST | `X-Admin-Token` | (new) clone a remote URL into the user's repos dir |
| `/admin/users/:id` | PATCH | `X-Admin-Token` | (new) partial user update (allow-list, disabled, ...) |

## Pitfalls

- The server-side `project` path is the **server filesystem** path. After
  admins register a clone via `/admin/repos/clone`, the returned `local_path`
  is what you should pass to the script's `project-server-path`.
- Always run the script from the **same shell user** that owns `~/.cursor/`
  — running as Administrator on Windows will write to a different profile.
- If `tools/list` returns 403 in step 5, the user's `ProjectPaths` allow-list
  hasn't been updated yet; admins can do that with PATCH
  `/admin/users/:id` (`{"project_paths": ["/new/path"]}`).