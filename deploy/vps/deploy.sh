#!/bin/bash
# Build Ancol-ani sync server cho linux/amd64 va deploy len VPS, chay qua
# systemd service "ancolani" (chay bang user thuong "an", xem ancolani.service).
# Script nay SSH bang root (ancolani-vps-root) chi de lam cac viec can quyen
# he thong (systemctl, ghi file vao /etc/systemd/system/, chown ve an) -
# khong can sua sudoers tren VPS.
#
# Server CHI nghe localhost:9090 (xem .env tren VPS), khong public port nay
# ra Internet. Truy cap tu ngoai di qua Cloudflare Tunnel (service he thong
# rieng ten "cloudflared", cai + cau hinh thu cong 1 lan tren VPS - xem
# docs/DEV-NOTES.md muc 10b), route cong khai cau hinh tren Cloudflare
# Dashboard (Zero Trust > Networks > Tunnels), KHONG o script nay.
#
# Chay tu goc repo ancol-ani tren Windows (Git Bash):
#   bash deploy/vps/deploy.sh
#
# Lam gi:
#   1. Build portal (npm) + syncserver linux/amd64 (CGO tat, SQLite thuan Go).
#   2. maintenance-on (dung service cu an toan), roi keo data/ancol-ani.db*
#      ve may nay: deploy/backup/ trong repo (git-ignored; doi bang bien
#      LOCAL_BACKUP_DIR), so sha256. Backup loi thi khoi dong lai service
#      cu va dung, khong deploy.
#   3. Upload binary vao ~/ancolani/, cai systemd unit (lan dau), tao
#      ~/ancolani/.env + .jwt-secret ngau nhien (lan dau) voi chmod 600.
#   4. systemctl start, kiem tra http://<host>:<port>/version tra 200.
#
# KHONG dung toi du lieu: ~/ancolani/data/ (SQLite), .env, .jwt-secret deu
# giu nguyen qua cac lan deploy sau.
#
# Thong tin server (IP, user, port) KHONG nam trong script nay (se len git)
# ma doc tu deploy/vps/vps.env (git-ignored, chi co tren may dev). Lan dau
# chay, copy deploy/vps/vps.env.example thanh vps.env va dien vao.
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

VPS_ENV="$REPO_ROOT/deploy/vps/vps.env"
if [ ! -f "$VPS_ENV" ]; then
  echo "[ERROR] Chua co $VPS_ENV - copy deploy/vps/vps.env.example thanh vps.env va dien IP/user/port thuc te." >&2
  exit 1
fi
# shellcheck disable=SC1090
source "$VPS_ENV"
: "${VPS_HOST:?thieu VPS_HOST trong vps.env}"
: "${VPS_PORT:?thieu VPS_PORT trong vps.env}"
: "${VPS_IDFILE:?thieu VPS_IDFILE trong vps.env}"

HOST="root@$VPS_HOST"
PORT="$VPS_PORT"
IDFILE="$VPS_IDFILE"
REMOTE_HOME=/home/an
REMOTE_DIR="$REMOTE_HOME/ancolani"
RUNUSER=an
BIN=ancolani-sync-amd64
LOCAL_PORT=9090
SSH="ssh -p $PORT -i $IDFILE -o BatchMode=yes -o ConnectTimeout=10 $HOST"
SCP="scp -P $PORT -i $IDFILE -o BatchMode=yes"

echo "== [1/6] Build portal =="
( cd cmd/syncserver/portal && { [ -d node_modules ] || npm ci --include=dev; } && npm run build )

echo "== [2/6] Build syncserver for linux/amd64 =="
mkdir -p build/bin
( export GOOS=linux GOARCH=amd64 CGO_ENABLED=0; go build -o "build/bin/$BIN" ./cmd/syncserver )

echo "== [3/6] Cai systemd unit (idempotent) =="
$SCP deploy/vps/ancolani.service "$HOST:/etc/systemd/system/ancolani.service"
$SSH "systemctl daemon-reload && systemctl enable ancolani >/dev/null"

echo "== [4/6] Maintenance mode on (dung service cu, neu da chay) =="
$SSH "mkdir -p '$REMOTE_DIR/data' && chown -R $RUNUSER:$RUNUSER '$REMOTE_DIR'"
if $SSH "systemctl is-active --quiet ancolani"; then
  $SSH "systemctl stop ancolani"
  echo "   da dung service."
else
  echo "   service chua chay (lan dau deploy), bo qua."
fi

echo "== [4b/6] Backup DB ve may nay =="
LOCAL_BACKUP_DIR="${LOCAL_BACKUP_DIR:-$REPO_ROOT/deploy/backup}"
backup_db() {
  local db="$REMOTE_DIR/data/ancol-ani.db"
  if ! $SSH "[ -f '$db' ]"; then echo "   chua co DB tren VPS, bo qua"; return 0; fi
  mkdir -p "$LOCAL_BACKUP_DIR"
  local dst="$LOCAL_BACKUP_DIR/ancol-ani-$(date +%Y%m%d-%H%M%S).db"
  local ext
  for ext in "" -wal -shm; do
    if [ -n "$ext" ] && ! $SSH "[ -f '$db$ext' ]"; then continue; fi
    $SCP "$HOST:$db$ext" "$dst$ext" || return 1
    local remote local_sum
    remote=$($SSH "sha256sum '$db$ext'" | cut -d' ' -f1) || return 1
    local_sum=$(sha256sum "$dst$ext" | cut -d' ' -f1)
    if [ -z "$remote" ] || [ "$remote" != "$local_sum" ]; then
      echo "   sha256 khong khop: ${ext:-db}"; return 1
    fi
  done
  echo "   -> $dst ($(du -h "$dst" | cut -f1), sha256 khop)"
}
if ! backup_db; then
  echo "   [ERROR] backup DB that bai - khoi dong lai service cu, KHONG deploy"
  $SSH "systemctl start ancolani" || true
  exit 1
fi

echo "== [5/6] Upload binary + config (lan dau tao .env/.jwt-secret) =="
$SSH "{ [ -s '$REMOTE_DIR/.jwt-secret' ] || { head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' > '$REMOTE_DIR/.jwt-secret'; chmod 600 '$REMOTE_DIR/.jwt-secret'; echo '   created new JWT secret'; }; } \
  && [ -s '$REMOTE_DIR/.env' ] || { printf 'ADDR=127.0.0.1:%s\nDB_DRIVER=sqlite\nDB_DSN=file:data/ancol-ani.db?_pragma=busy_timeout(5000)\nJWT_SECRET=%s\n' '$LOCAL_PORT' \"\$(cat '$REMOTE_DIR/.jwt-secret')\" > '$REMOTE_DIR/.env'; chmod 600 '$REMOTE_DIR/.env'; echo '   created .env'; }"
$SCP "build/bin/$BIN" "$HOST:$REMOTE_DIR/$BIN"
$SSH "chmod +x '$REMOTE_DIR/$BIN' && chown -R $RUNUSER:$RUNUSER '$REMOTE_DIR'"

echo "== [6/6] Restart service =="
$SSH "systemctl restart ancolani"

echo "== Cho server len (http://<vps>:$LOCAL_PORT/) =="
for i in $(seq 1 30); do
  code=$($SSH "curl -s -o /dev/null -w '%{http_code}' --max-time 5 http://localhost:$LOCAL_PORT/" || true)
  if [ "$code" = "200" ]; then echo "   OK (HTTP 200)"; break; fi
  sleep 2
done
[ "${code:-}" = "200" ] || {
  echo "   [WARN] chua thay HTTP 200 (lan cuoi: ${code:-none}). Xem log:"
  echo "     ssh ancolani-vps-root journalctl -u ancolani -n 50 --no-pager"
}

echo "== Done. Ancol-ani syncserver chay tren VPS, port local $LOCAL_PORT =="
echo "   Cap quyen admin: ssh ancolani-vps \"cd ancolani && ./$BIN -make-admin <username>\""
echo "   Xem log: ssh ancolani-vps-root journalctl -u ancolani -f"
