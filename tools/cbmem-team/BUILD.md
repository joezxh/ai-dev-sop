# cbmem-team Build & Deploy

## One-time build

```bash
cd tools/cbmem-team
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" \
    -o dist/cbmem-team      ./cmd/cbmem-team
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w" \
    -o dist/cbmem-mint-token ./cmd/cbmem-mint-token
ls -lh dist/
```

## Server deploy (192.168.100.83)

```bash
# Upload
scp dist/cbmem-team      tianque@192.168.100.83:/tmp/
scp dist/cbmem-mint-token tianque@192.168.100.83:/tmp/
scp deploy/cbmem-team.service tianque@192.168.100.83:/tmp/
scp deploy/cbmem-team.env    tianque@192.168.100.83:/tmp/

# Install on server (via SSH)
ssh tianque@192.168.100.83 <<'REMOTE'
set -e
sudo install -m 0755 /tmp/cbmem-team      /usr/local/bin/
sudo install -m 0755 /tmp/cbmem-mint-token /usr/local/bin/
sudo useradd --system --no-create-home --shell /usr/sbin/nologin cbmem 2>/dev/null || true
sudo mkdir -p /etc/cbmem-team /var/lib/cbmem-team /var/log/cbmem-team
sudo cp /tmp/cbmem-team.env /etc/cbmem-team/
sudo chmod 0600 /etc/cbmem-team/cbmem-team.env
sudo cp /tmp/cbmem-team.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now cbmem-team
sudo systemctl status cbmem-team --no-pager
REMOTE
```

## Smoke test

```bash
export ADMIN_TOKEN=...
export BASE=http://192.168.100.83:8787
bash examples/smoke.sh
```