# Changelog

## 1.0.0

- Widget hoạt ảnh góc màn hình (frameless, click-through, always-on-top, tự định vị lại khi đổi màn hình).
- Hiển thị gif + chữ, chữ nhập qua trang web local `http://<ip-máy>:8788`.
- Trang web cho phép tải lên / đặt lại ảnh GIF, xem trước bằng canvas tự vẽ frame (tránh lỗi GIF bị đứng hình do decoder gốc của Chromium).
- Tự "dọn" GIF lỗi encode khi tải lên (decode + re-encode qua `image/gif`).
- Trang web hỗ trợ đa ngôn ngữ (VI/EN), responsive cho điện thoại (Tailwind CSS).
- Server local lắng nghe `0.0.0.0:8788` để truy cập được từ điện thoại cùng mạng LAN.
