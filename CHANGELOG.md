# Changelog

Thay đổi của app desktop Ancol-ani (widget + trang settings `http://<ip-máy>:8788`).
Phiên bản đang chạy hiện ở chân trang settings và ở trang **Có gì mới** trong app.

Mỗi phiên bản là một mục `## <version> — <ngày>`, chia thành `### Mới`, `### Thay đổi`, `### Sửa lỗi`.
Trang **Có gì mới** của app đọc trực tiếp file này, nên giữ đúng định dạng đó.

## 1.1.3 — 2026-11-19

### Mới
- Bấm phải chuột vào widget để hiện "Cài đặt / Hủy" đè lên GIF; bấm "Cài đặt" mở trang settings trong một cửa sổ riêng (vẫn là WebView2 thật, không phải trình duyệt), widget vẫn chạy hoạt ảnh bình thường bên cạnh.
- Chỉ chạy được một widget cùng lúc: mở app lần hai tự đưa widget đang chạy lên trước thay vì chạy thêm một bản nữa.

### Thay đổi
- Widget không còn "click-through" hoàn toàn (trước đây mọi chuột xuyên qua widget) để nhận được click chuột phải.

## 1.1.2 — 2026-11-19

### Thay đổi
- Làm lại khối chọn vị trí widget trên trang **Widget**: mô phỏng màn hình rõ hơn, gộp cùng hàng với ô nhập chữ hiển thị thay vì tách rời.
- Đổi nhãn "Widget" (menu, tiêu đề trang) tiếng Việt thành "Hoạt ảnh".

### Sửa lỗi / nội bộ
- Gộp 2 file dữ liệu `custom-text.txt` và `position.txt` thành một file `settings.data` duy nhất, mã hóa AES-GCM (trước đây lưu dạng chữ thường, đọc được trực tiếp).

## 1.1.0 — 2026-11-19

### Mới
- Liên kết app với tài khoản trên sync server (trang **Tài khoản**): đăng nhập qua portal rồi tự quay về, hủy liên kết bất cứ lúc nào. Token chỉ lưu trong trình duyệt, app không giữ.
- Trang **Thư viện**: xem GIF trong tài khoản, bấm **Dùng cho widget** để đổi ảnh widget, tải file GIF lên server, chia sẻ công khai (đặt tên + tag), gỡ chia sẻ, xóa.
- Trang **Khám phá**: xem GIF mọi người chia sẻ công khai, tìm theo tên/tag hoặc người chia sẻ, dùng cho widget chỉ với một nút.
- Tải ảnh widget đang hiển thị lên thư viện server ngay từ trang **Widget**.
- Phân trang cho thư viện và trang khám phá; ảnh đã tải được giữ lại trong phiên, chuyển trang không phải tải lại.
- Trang **Có gì mới**: xem phiên bản đang chạy và changelog; có chấm báo khi app vừa lên phiên bản mới mà bạn chưa xem.

### Thay đổi
- Thiết kế lại toàn bộ trang settings theo cùng giao diện với web portal của server: thanh điều hướng, chia thành các trang Widget / Thư viện / Khám phá / Tài khoản, kéo thả file GIF, thông báo dạng toast.
- Chữ trên widget hiện trong bong bóng thoại (nền tím than, chữ trắng) thay cho chữ đỏ viền vàng khó đọc; đọc rõ trên cả hình nền sáng lẫn tối. Tối đa 2 dòng, dài hơn tự cắt bằng dấu "…".
- Đổi ngôn ngữ VI/EN có hiệu lực ngay, không cần tải lại trang.
- Hộp thoại xác nhận theo giao diện app thay cho hộp thoại mặc định của trình duyệt.
- Dùng icon chính của app (chữ "A") cho file `.exe`, bộ cài, favicon và logo; thay hết icon mặc định của Wails.

### Sửa lỗi
- Widget bị lệch khỏi mép phải và lấn xuống dưới taskbar (nên bị che, trông như không còn "luôn nằm trên") khi Windows để scale khác 100%. Giờ luôn nằm đúng góc dưới phải, phía trên taskbar, và tự giữ "luôn nằm trên".
- Phiên đăng nhập hết hạn hoặc bị thu hồi (đăng xuất nơi khác, đổi mật khẩu, admin reset) thì trang settings tự về trạng thái chưa liên kết thay vì hiện thư viện không tải được.

## 1.0.0

- Widget hoạt ảnh góc màn hình (frameless, click-through, always-on-top, tự định vị lại khi đổi màn hình).
- Hiển thị gif + chữ, chữ nhập qua trang web local `http://<ip-máy>:8788`.
- Trang web cho phép tải lên / đặt lại ảnh GIF, xem trước bằng canvas tự vẽ frame (tránh lỗi GIF bị đứng hình do decoder gốc của Chromium).
- Tự "dọn" GIF lỗi encode khi tải lên (decode + re-encode qua `image/gif`).
- Trang web hỗ trợ đa ngôn ngữ (VI/EN), responsive cho điện thoại (Tailwind CSS).
- Server local lắng nghe `0.0.0.0:8788` để truy cập được từ điện thoại cùng mạng LAN.
