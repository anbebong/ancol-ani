# Ancol-Ani

Widget GIF/chữ ở góc màn hình cho Windows (Go + Wails), kèm trang settings local và một sync server có web portal để đồng bộ/chia sẻ GIF qua tài khoản.

## Tính năng chính

- Widget hoạt ảnh (GIF) + chữ, luôn nổi trên cùng, không chặn click, tự định vị lại khi đổi màn hình/độ phân giải.
- Chọn vị trí hiển thị (4 góc màn hình), đổi chữ, đổi GIF — tất cả qua trang settings local mở bằng cách phải chuột vào widget.
- Bật/tắt khởi động cùng Windows.
- Liên kết tài khoản (tùy chọn) để đồng bộ/chia sẻ GIF giữa các thiết bị qua sync server riêng.
- Chỉ chạy một widget cùng lúc; cửa sổ settings mở trong WebView2 riêng, không phải trình duyệt.
- Dữ liệu cá nhân (chữ, vị trí, port server) lưu mã hóa (AES-GCM) trên máy, server local chỉ nghe `127.0.0.1`.

Chi tiết kiến trúc, build, API, luồng liên kết tài khoản: xem [CLAUDE.md](CLAUDE.md) và [docs/DEV-NOTES.md](docs/DEV-NOTES.md).

## Cấu trúc thư mục

| Thư mục | Là gì |
|---|---|
| repo root (`app.go`, `main.go`) | App desktop Wails: widget + server local |
| `widget/` | UI widget (GIF + chữ) |
| `settings-ui/` | Trang settings local — Vue 3 |
| `cmd/syncserver/` | Sync server Go (đồng bộ/chia sẻ GIF qua tài khoản) |
| `cmd/syncserver/portal/` | Web portal của sync server — Vue 3 |
| `theme/` | Thiết kế dùng chung giữa `settings-ui/` và `portal/` |

## Phát triển

Máy dev đặt `NODE_ENV=production` thì cần cài bằng `npm ci --include=dev` (hoặc `npm install --include=dev`) cho cả `settings-ui/`, `widget/` và `cmd/syncserver/portal/`.

```sh
# App desktop
cd web && npm run build
cd ../widget && npm run build
wails build -s   # -s vì Wails tự cài npm sẽ thiếu vite

# Sync server
cd cmd/syncserver/portal && npm run build
go run ./cmd/syncserver
```

Build xong, khôi phục file Wails tự sửa: `git checkout -- go.mod widget/package.json.md5 widget/wailsjs`.

## Build

Build bản production: `wails build`. Exe nằm ở `build/bin/ancol-ani.exe`, nhúng sẵn toàn bộ UI (`go:embed`) — chỉ cần gửi đúng 1 file exe, máy nhận cần có Microsoft Edge WebView2 Runtime (có sẵn trên hầu hết Windows 10/11).

## Phiên bản & changelog

Xem [CHANGELOG.md](CHANGELOG.md), hoặc trong app: phải chuột vào widget → Cài đặt → trang **Về Ancol-Ani**.
