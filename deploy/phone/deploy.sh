#!/bin/bash
# Build Ancol-ani sync server cho ARM64 va deploy len dien thoai Termux
# (192.168.0.105:8022) - thay cho dino-race tren cong 7879, de domain
# https://game.anbebong.io.vn (Cloudflare Tunnel -> localhost:7879) phuc vu portal
# Ancol-ani. Ancha (:8080) giu nguyen.
#
# Chay tu goc repo ancol-ani tren Windows (Git Bash):
#   bash deploy/phone/deploy.sh
#
# Lam gi:
#   1. Build portal (npm) + syncserver linux/arm64 (CGO tat, SQLite thuan Go).
#   2. maintenance-on (dung server/tunnel cu an toan), roi keo
#      ~/ancolani/data/ancol-ani.db ve may nay: deploy/backup/ trong repo
#      (git-ignored; doi bang bien LOCAL_BACKUP_DIR), so sha256. Backup loi
#      thi bat lai server cu va dung, khong deploy.
#   3. Upload binary vao ~/ancolani/, run.sh + maintenance-on.sh ban ancol-ani.
#      Lan dau: sao luu ~/run.sh cu thanh ~/run.sh.dinorace-backup, tao
#      ~/ancolani/.jwt-secret ngau nhien (chmod 600).
#   4. maintenance-off (khoi dong lai), kiem tra https://game.anbebong.io.vn.
#
# KHONG dung toi du lieu: ~/ancolani/data/ (SQLite Ancol-ani), ~/ancha/data/,
# va du lieu dino-race (~/server, ~/users.json, ~/admin.json) deu giu nguyen.
# Quay lai dino-race: bash deploy/deploy-arm.sh trong repo dino-race (no ghi
# de lai run.sh ban dino-race).
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

HOST=u0_a249@192.168.0.105
PORT=8022
REMOTE_HOME=/data/data/com.termux/files/home
REMOTE_DIR="$REMOTE_HOME/ancolani"
BIN=ancolani-sync-arm64
SSH="ssh -p $PORT -o BatchMode=yes -o ConnectTimeout=10 $HOST"

echo "== [1/5] Build portal =="
( cd cmd/syncserver/portal && { [ -d node_modules ] || npm ci --include=dev; } && npm run build )

echo "== [2/5] Build syncserver for linux/arm64 =="
mkdir -p build/bin
( export GOOS=linux GOARCH=arm64 CGO_ENABLED=0; go build -o "build/bin/$BIN" ./cmd/syncserver )

echo "== [3/5] Maintenance mode on (stops current server/tunnel) =="
scp -P "$PORT" -o BatchMode=yes deploy/phone/maintenance-on.sh "$HOST:$REMOTE_HOME/maintenance-on.sh"
$SSH "chmod +x '$REMOTE_HOME/maintenance-on.sh' && '$REMOTE_HOME/maintenance-on.sh'"

echo "== [3b/5] Backup DB ve may nay =="
# Server da dung (maintenance-on) nen DB tren dien thoai dang nhat quan.
# Keo ve $LOCAL_BACKUP_DIR (git-ignored), so sha256 hai dau. Khong luu ban
# backup nao tren dien thoai. Loi thi bat lai server cu va dung deploy.
LOCAL_BACKUP_DIR="${LOCAL_BACKUP_DIR:-$REPO_ROOT/deploy/backup}"
backup_db() {
  local db="$REMOTE_DIR/data/ancol-ani.db"
  if ! $SSH "[ -f '$db' ]"; then echo "   chua co DB tren dien thoai, bo qua"; return 0; fi
  mkdir -p "$LOCAL_BACKUP_DIR"
  local dst="$LOCAL_BACKUP_DIR/ancol-ani-$(date +%Y%m%d-%H%M%S).db"
  local ext
  for ext in "" -wal -shm; do
    if [ -n "$ext" ] && ! $SSH "[ -f '$db$ext' ]"; then continue; fi
    scp -P "$PORT" -o BatchMode=yes "$HOST:$db$ext" "$dst$ext" || return 1
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
  echo "   [ERROR] backup DB that bai - khoi dong lai server cu, KHONG deploy"
  $SSH "'$REMOTE_HOME/maintenance-off.sh'" || true
  exit 1
fi

echo "== [4/5] Upload =="
$SSH "mkdir -p '$REMOTE_DIR/data' \
  && { [ -f '$REMOTE_HOME/run.sh.dinorace-backup' ] || ! grep -q dinorace-server-arm64 '$REMOTE_HOME/run.sh' || cp '$REMOTE_HOME/run.sh' '$REMOTE_HOME/run.sh.dinorace-backup'; } \
  && { [ -s '$REMOTE_DIR/.jwt-secret' ] || { head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n' > '$REMOTE_DIR/.jwt-secret'; chmod 600 '$REMOTE_DIR/.jwt-secret'; echo '   created new JWT secret'; }; }"
scp -P "$PORT" -o BatchMode=yes "build/bin/$BIN" "$HOST:$REMOTE_DIR/$BIN"
scp -P "$PORT" -o BatchMode=yes deploy/phone/run.sh "$HOST:$REMOTE_HOME/run.sh"
$SSH "chmod +x '$REMOTE_DIR/$BIN' '$REMOTE_HOME/run.sh'"

echo "== [5/5] Maintenance mode off (starts Ancol-ani + Ancha + tunnel) =="
$SSH "'$REMOTE_HOME/maintenance-off.sh'"

echo "== Waiting for https://game.anbebong.io.vn =="
for i in $(seq 1 30); do
  code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 https://game.anbebong.io.vn/ || true)
  if [ "$code" = "200" ]; then echo "   OK (HTTP 200)"; break; fi
  sleep 2
done
[ "${code:-}" = "200" ] || echo "   [WARN] chua thay HTTP 200 (lan cuoi: ${code:-none}). Xem log: ssh -p $PORT $HOST tail -50 ~/dinorace-tunnel.log"

echo "== Done. Ancol-ani :7879 (https://game.anbebong.io.vn) + Ancha :8080 tren dien thoai =="
echo "   Cap quyen admin: ssh -p $PORT $HOST \"cd ~/ancolani && ./$BIN -make-admin <username>\""
