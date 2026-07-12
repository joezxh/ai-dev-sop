# cbmem-team

**HTTP multi-user wrapper around [`codebase-memory-mcp`](https://github.com/DeusData/codebase-memory-mcp).**

`codebase-memory-mcp` is by design a single-developer, local-machine tool:
each developer runs their own binary against their own copy of the source.
That is great for privacy and zero-config, but it means a team cannot share
a central instance.

`cbmem-team` is the smallest possible piece of glue that fixes this:

```
┌──────────────┐    POST /mcp?as=alice&project=/code/foo
│  Cursor /    │ ─────────────────────────────────────────┐
│  Claude /    │   Authorization: Bearer <jwt>           │
│  Qoder       │                                          ▼
└──────────────┘                              ┌──────────────────────┐
                                              │   cbmem-team (HTTP)  │
                                              │  ┌────────────────┐  │
                                              │  │  per-user      │  │
                                              │  │  stdio pool    │  │
                                              │  └─────┬──────────┘  │
                                              └────────┼─────────────┘
                                                       ▼
                              ┌────────────────────────────────────┐
                              │  codebase-memory-mcp  (alice)       │
                              │  workdir: /var/lib/cbmem-team/      │
                              │           users/alice/projects/foo │
                              │  SQLite (writable only by alice)    │
                              └────────────────────────────────────┘
```

* One **stdio subprocess per user**. SQLite file-locks give natural isolation.
* Requests are routed by the JWT subject. The wrapper does **not** parse
  JSON-RPC - it forwards bytes - so upstream changes cannot break us.
* Admins mint tokens with `POST /admin/users/:id/token?ttl=720h`.
* Idle processes are reaped after `idle_ttl` (default 30 minutes).

## Build

```bash
cd tools/cbmem-team
go build -o cbmem-team ./cmd/cbmem-team
```

Produces a single static binary.

## Server-side setup

```bash
# 1. Install the wrapper
sudo install -m 0755 cbmem-team /usr/local/bin/cbmem-team

# 2. Create directories
sudo mkdir -p /etc/cbmem-team /var/lib/cbmem-team
sudo install -m 0600 deploy/cbmem-team.env /etc/cbmem-team/cbmem-team.env
sudo systemctl edit --full cbmem-team   # paste unit from deploy/
sudo systemctl enable --now cbmem-team
```

## Admin workflow

```bash
# Create a developer
curl -X POST http://localhost:8787/admin/users \
  -H "X-Admin-Token: $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"alice","display_name":"Alice","project_paths":["/code/foo","/code/bar"]}'

# Mint a 30-day token
TOKEN=$(curl -X POST 'http://localhost:8787/admin/users/alice/token?ttl=720h' \
        -H "X-Admin-Token: $ADMIN_TOKEN" | jq -r .token)

# Stats
curl http://localhost:8787/admin/stats -H "X-Admin-Token: $ADMIN_TOKEN"
```

## Client-side configuration (Cursor)

`~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "cbmem-team": {
      "url": "http://mcp.internal.example.com:8787/mcp?as=alice&project=${workspaceFolder}",
      "headers": {
        "Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...."
      }
    }
  }
}
```

Replace `${workspaceFolder}` with the literal path your team agrees on
(e.g. `/code/foo`). Each developer only needs their **own JWT**; the URL
parameter `as=...` is what isolates their indexes server-side.

## Security checklist

- [x] JWT signature verified on every `/mcp` request
- [x] Per-user `ProjectPaths` allow-list (optional, set per user)
- [ ] **TLS**: deploy behind nginx / caddy; do not expose `:8787` directly
- [ ] **Admin token rotation**: rotate quarterly, store in vault
- [ ] **Disk quota**: `/var/lib/cbmem-team` is one SQLite file per
       (user, project); monitor via `du -sh /var/lib/cbmem-team/users/*`

## Deployment gotchas (learned the hard way)

When we first deployed this we hit three issues that are worth documenting:

### 1. `codebase-memory-mcp` lives in a 750 home directory

On the test server the binary lives at `/home/tianque/codebase-memory-mcp`
and `/home/tianque` is `drwxr-x---` (mode 750). The wrapper runs as
user `cbmem`, which can neither traverse the directory nor exec the
binary until we either:

- `chmod o+rx /home/tianque` (acceptable on an internal dev box)
- or move the binary to `/usr/local/bin/` and tighten `ProtectHome=true`

The provided systemd unit uses `ProtectHome=read-only` so the user can at
least traverse and exec; if your binary lives outside `/home`, switch
to `ProtectHome=true` for stronger isolation.

### 2. `pscp` corrupts large binaries

PuTTY's `pscp.exe` is unreliable above ~5 MB on this network: progress
bars stall and bytes get dropped, leaving you with a binary that SEGVs
on startup. Use `scp` (OpenSSH) instead:

```bash
scp -P 22 dist/cbmem-team-linux-amd64 user@host:/tmp/cbmem-team
```

Verify before installing:

```bash
ssh user@host 'sha256sum /tmp/cbmem-team'
sha256sum dist/cbmem-team-linux-amd64      # local
```

The hashes **must** match.

### 3. PowerShell strips CR from `curl -d`

PowerShell's `curl` is `Invoke-WebRequest`, which silently mangles
JSON bodies when you escape them via PowerShell. Either:

- save the body to a file first and use `--data-binary @file.json`
- or run the test from WSL / Git Bash
- or use the scripts in `examples/` which already do that

## Limits and honest trade-offs

| Concern | Today | Plan |
|---------|-------|------|
| Transport | HTTP only | SSE in v0.3 once upstream exposes it |
| Auth model | Shared HS256 secret | Switch to per-user keys + Vault transit |
| Index sharing | None (intentional) | Add `team_share: true` to enable a shared read-only SQLite mount |
| High availability | Single instance | Sticky session load balancer (no shared state) |

## Related docs

- [`CONSOLE.md`](./CONSOLE.md) — admin console (CON-01..CON-07): users, projects,
  sessions, summarize, distill, **workflows (M3)**, **repo pipelines (M4)**.
- [`BUILD.md`](./BUILD.md) — build / cross-compile / release tarball.
- [`cbmem-team-prd.md`](./cbmem-team-prd.md) — full product requirements.