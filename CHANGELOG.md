# Changelog

Thay đổi của app desktop Ancol-Ani (widget + trang settings `http://localhost:<port>`).
Phiên bản đang chạy hiện ở chân trang settings và ở trang **Về Ancol-Ani** trong app.

Mỗi phiên bản là một mục `## <version> — <ngày>`, chia thành `### Mới`, `### Thay đổi`, `### Sửa lỗi`.
Trang **Về Ancol-Ani** của app đọc trực tiếp file này, nên giữ đúng định dạng đó.

## 1.1.7 — 2026-11-19

### Sửa lỗi
- Cài cập nhật có lúc không thay được file thật (bị kẹt do antivirus quét file mới tải xong), mở lại app cũ mà không báo lỗi. Giờ tự thử lại vài lần, có log để kiểm tra khi cần.

## 1.1.6 — 2026-11-19

### Thay đổi
- Nguồn kiểm tra cập nhật đổi từ server riêng sang GitHub Releases của repo (đơn giản hơn, không cần tự host).

## 1.1.5 — 2026-11-19

### Mới
- Nút "Kiểm tra cập nhật" ở trang Về Ancol-Ani: có bản mới thì hỏi xác nhận, đồng ý mới tải và tự cài, tự khởi động lại app.

## 1.1.4 — 2026-11-19

### Mới
- Bật/tắt khởi động cùng Windows ngay trong trang Widget.
- Trang **Về Ancol-Ani**: thêm chính sách bảo mật & quyền riêng tư, điều hướng riêng ở chân trang.

### Thay đổi
- Đổi tên hiển thị "Ancol-ani" thành "Ancol-Ani"; đổi tab "Có gì mới" thành "Về Ancol-Ani".
- Server local chỉ nghe `127.0.0.1` (không còn mở cho máy khác trong mạng LAN), tự dò port trống nếu port mặc định bị chiếm.

### Sửa lỗi / nội bộ
- Gộp port server vào `settings.data` chung, bỏ file riêng không cần thiết.

## 1.1.3 — 2026-11-19

### Mới
- Phải chuột vào widget để mở trang settings trong cửa sổ riêng, widget vẫn chạy bên cạnh.
- Chỉ chạy được một widget cùng lúc.

### Thay đổi
- Widget nhận chuột phải được (bỏ click-through hoàn toàn).

## 1.1.2 — 2026-11-19

### Thay đổi
- Làm lại khối chọn vị trí widget, gộp cùng hàng với ô nhập chữ.
- Đổi nhãn "Widget" tiếng Việt thành "Hoạt ảnh".

### Sửa lỗi / nội bộ
- Gộp dữ liệu chữ + vị trí vào một file, mã hóa thay vì lưu chữ thường.

## 1.1.0 — 2026-11-19

### Mới
- Liên kết tài khoản để đồng bộ GIF giữa các thiết bị.
- Trang **Thư viện**: quản lý, chia sẻ, dùng GIF cho widget.
- Trang **Khám phá**: dùng GIF người khác chia sẻ công khai.
- Trang **Về Ancol-Ani**: xem phiên bản và changelog.

### Thay đổi
- Thiết kế lại trang settings theo giao diện mới, nhiều trang, thông báo toast.
- Chữ trên widget hiện trong bong bóng thoại dễ đọc hơn.
- Đổi ngôn ngữ VI/EN không cần tải lại trang.

### Sửa lỗi
- Widget lệch vị trí khi Windows để scale khác 100%.
- Phiên đăng nhập hết hạn thì tự về trạng thái chưa liên kết.

## 1.0.0

- Widget hoạt ảnh góc màn hình, luôn nổi trên cùng, tự định vị khi đổi màn hình.
- Hiển thị GIF + chữ, chỉnh qua trang web local.
- Tải lên / đặt lại ảnh GIF, hỗ trợ đa ngôn ngữ (VI/EN), dùng được từ điện thoại cùng mạng LAN.
