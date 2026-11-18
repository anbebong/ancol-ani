#!/bin/bash
# Bật maintenance mode: tạo cờ khiến vòng lặp trong run.sh dừng hẳn (không tự
# restart nữa), rồi kill sạch server + cloudflared + proot hiện tại — dùng
# trước khi cập nhật binary, tránh vòng lặp khởi động lại bản cũ giữa chừng.
#
# Bản của repo ancol-ani: giống bản dino-race nhưng kill thêm Ancol-ani
# (ancolani-sync-arm64). Vẫn kill dinorace-server-arm64 để chuyển từ
# dino-race sang Ancol-ani (và ngược lại) đều sạch.
touch "$HOME/.dinorace-maintenance"
echo "[maintenance-on] Đã bật maintenance mode."

tmux kill-session -t dinorace 2>/dev/null
pkill -9 -f ancolani-sync-arm64 2>/dev/null
pkill -9 -f dinorace-server-arm64 2>/dev/null
pkill -9 -f ancha-arm64 2>/dev/null
pkill -9 -f 'cloudflared tunnel' 2>/dev/null
pkill -9 -f proot 2>/dev/null

sleep 1
echo "[maintenance-on] Đã dừng server/tunnel. Giờ có thể cập nhật binary an toàn."
echo "  Cập nhật xong, chạy: ~/maintenance-off.sh"
