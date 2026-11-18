#!/bin/bash
# Phone (Termux) main loop: Ancol-ani sync server + Ancha + Cloudflare Tunnel,
# all inside termux-chroot (proot), restarted together if cloudflared exits.
# Uploaded to ~/run.sh by deploy/phone/deploy.sh in the ancol-ani repo, and
# started (inside tmux, detached) by ~/run-in-tmux.sh.
#
# Replaces dino-race's run.sh (repo code-game/dino-race, deploy/run.sh):
# same structure, but the dino-race game server on :7879 is swapped for the
# Ancol-ani sync server on the same port - so the existing tunnel hostname
# game.anbebong.io.vn -> http://localhost:7879 serves the Ancol-ani portal
# with no Cloudflare change. Ancha (:8080, mothe.anbebong.io.vn) is
# unchanged. The root domain anbebong.io.vn is NOT on this tunnel. The old
# file is kept on the phone as ~/run.sh.dinorace-backup.
#
# Tunnel ingress (configured on the Cloudflare dashboard, read from the
# cloudflared log): game.anbebong.io.vn -> :7879, mothe.anbebong.io.vn -> :8080.
#
# Why proot / exec < /dev/null: see dino-race deploy/README.md - Termux's
# sandbox has no usable /etc/resolv.conf (cloudflared needs DNS), and proot
# hangs inside a detached tmux pane unless stdin is /dev/null.
#
# MAINTENANCE MODE: while ~/.dinorace-maintenance exists the loop stops and
# does not restart anything (maintenance-on.sh / maintenance-off.sh).

CURRENT_DIR="$HOME"
MAINT_FLAG="$HOME/.dinorace-maintenance"

echo "[+] Đang kích hoạt termux-chroot, chạy Ancol-ani, Ancha và Cloudflare Tunnel..."

exec < /dev/null
termux-chroot /bin/bash -c "
    mkdir -p /etc
    echo 'nameserver 1.1.1.1' > /etc/resolv.conf
    echo 'nameserver 8.8.8.8' >> /etc/resolv.conf
    export GODEBUG=netdns=go
    export SSL_CERT_FILE=\$PREFIX/etc/tls/cert.pem
    cd '$CURRENT_DIR'
    mkdir -p ./ancha ./ancolani/data
    chmod +x ./ancolani/ancolani-sync-arm64 2>/dev/null
    chmod +x ./ancha/ancha-arm64 2>/dev/null
    chmod +x ./cloudflared 2>/dev/null

    while true; do
        if [ -f '$MAINT_FLAG' ]; then
            echo \"[\$(date)] Maintenance mode bật (\$MAINT_FLAG tồn tại) — dừng vòng lặp, không tự khởi động lại.\"
            exit 0
        fi

        echo \"[\$(date)] Đang khởi chạy Ancol-ani sync server (:7879)...\"
        ( cd ./ancolani && \
          ADDR=0.0.0.0:7879 \
          DB_DSN='file:data/ancol-ani.db?_pragma=busy_timeout(5000)' \
          JWT_SECRET=\"\$(cat ./.jwt-secret)\" \
          ./ancolani-sync-arm64 ) &
        SERVER_PID=\$!

        echo \"[\$(date)] Đang khởi chạy Ancha (:8080)...\"
        ( cd ./ancha && ./ancha-arm64 ) &
        ANCHA_PID=\$!

        sleep 2

        echo \"[\$(date)] Đang khởi chạy Cloudflare Tunnel (named tunnel, token: game./mothe.anbebong.io.vn)...\"
        ./cloudflared tunnel run --token \"\$(cat '$CURRENT_DIR/.cf-tunnel-token')\"
        echo \"[\$(date)] cloudflared thoát, dừng server cũ (PID \$SERVER_PID \$ANCHA_PID) và khởi động lại toàn bộ sau 5s\"
        kill \$SERVER_PID \$ANCHA_PID 2>/dev/null
        pkill -f ancolani-sync-arm64 2>/dev/null
        wait \$SERVER_PID 2>/dev/null
        wait \$ANCHA_PID 2>/dev/null
        sleep 5
    done
"
