# UC — GIF Toolkit

Tham chiếu: `docs/BA-gif-toolkit.md`. Mọi UC không cần đăng nhập (trang `/tools/*` công khai).

## UC-01 — Convert video sang GIF
**Actor**: khách (ai cũng dùng được)
**Trigger**: vào `/tools/convert`, chọn tab "Video", chọn file video.
**Luồng chính**:
1. Chọn file video (mp4/webm/mov), tuỳ chọn FPS đầu ra.
2. Bấm "Chuyển đổi" → `POST /api/tools/convert`.
3. Server kiểm tra định dạng + thời lượng (`ffprobe`), chạy `ffmpeg` chuyển sang GIF.
4. Trả về GIF, portal hiện preview (canvas) + nút "Tải xuống".
**Luồng phụ**:
- 3a. Video dài hơn giới hạn (đề xuất 30s) → trả lỗi 400 "video quá dài", không chạy ffmpeg.
- 3b. File không phải video hợp lệ / ffprobe lỗi → 400 "file không hợp lệ".
- 3c. Server thiếu `ffmpeg` → 503 "server thiếu ffmpeg".
- 3d. GIF kết quả vượt giới hạn frame/kích thước (mục 4 BA) → 400 báo rõ lý do, không trả file lỗi.

## UC-02 — Convert 1 ảnh tĩnh sang GIF
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/convert`, tab "Ảnh", chọn 1 ảnh (PNG/JPEG).
2. Bấm "Chuyển đổi" → server bọc ảnh thành GIF 1 frame.
3. Trả GIF, preview + tải xuống.
**Luồng phụ**: ảnh sai định dạng/quá lớn (>20MB) → 400.

## UC-03 — Convert nhiều ảnh thành GIF animation
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/convert`, tab "Nhiều ảnh", chọn nhiều ảnh, nhập delay giữa frame (ms).
2. Bấm "Chuyển đổi" → server ghép các ảnh theo thứ tự chọn thành 1 GIF nhiều frame.
3. Trả GIF, preview + tải xuống.
**Luồng phụ**:
- Số ảnh vượt giới hạn (đề xuất >100) → 400.
- Có ảnh quá lớn hoặc sai định dạng → 400, nêu rõ ảnh nào lỗi.
- Các ảnh kích thước khác nhau → server co giãn/cắt theo ảnh đầu tiên (quyết định kỹ thuật, nêu rõ trong UI).

## UC-04 — Resize GIF
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/resize`, chọn GIF, nhập width/height mới hoặc %.
2. Bấm "Resize" → `POST /api/tools/resize`.
3. Server kiểm tra GIF input qua `checkGifSize`, resize từng frame, trả GIF mới.
**Luồng phụ**:
- GIF input vượt giới hạn (`errGifTooBig`) → 400.
- Width/height = 0 hoặc âm → 400 lỗi tham số.
- Kích thước mới vượt `maxGifSide` → 400 "kích thước vượt giới hạn".

## UC-05 — Crop GIF
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/crop`, chọn GIF, xem preview, kéo chọn khung cắt (x, y, w, h).
2. Bấm "Crop" → server crop từng frame theo đúng toạ độ, trả GIF mới.
**Luồng phụ**:
- Khung cắt vượt ra ngoài kích thước GIF gốc → 400.
- GIF input không hợp lệ/quá lớn → 400 (như UC-04).

## UC-06 — Optimize (giảm dung lượng) GIF
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/optimize`, chọn GIF, chọn mức tối ưu (1=nhẹ, 2=vừa, 3=mạnh) hoặc số màu cụ thể.
2. Bấm "Optimize" → server giảm số màu/palette theo mức chọn, trả GIF mới + hiện dung lượng trước/sau.
**Luồng phụ**: GIF input lỗi/quá lớn → 400.

## UC-07 — Rotate / Flip GIF
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/rotate`, chọn GIF, chọn góc xoay (90/180/270) và/hoặc lật ngang/dọc.
2. Bấm "Áp dụng" → server xoay/lật từng frame, trả GIF mới.
**Luồng phụ**: GIF input lỗi/quá lớn → 400; không chọn gì (angle=0, flip=none) → trả nguyên bản, không lỗi.

## UC-08 — Reverse GIF
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/reverse`, chọn GIF.
2. Bấm "Đảo ngược" → server đảo thứ tự frame (giữ delay tương ứng), trả GIF mới.
**Luồng phụ**: GIF chỉ có 1 frame → trả nguyên bản (không lỗi, không có gì để đảo).

## UC-09 — Đổi tốc độ GIF
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/speed`, chọn GIF, nhập hệ số tốc độ (vd 0.5x, 2x) hoặc chọn preset.
2. Bấm "Áp dụng" → server nhân delay mỗi frame theo `1/factor`, trả GIF mới.
**Luồng phụ**:
- Factor <= 0 → 400 lỗi tham số.
- Delay kết quả quá nhỏ (ví dụ <2 centisecond, dưới chuẩn GIF) → server ghim về mức tối thiểu hợp lệ, không lỗi.

## UC-10 — Split GIF thành frame (ảnh)
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/split`, chọn GIF.
2. Bấm "Tách frame" → server decode từng frame, đóng thành `frame_NNN.png`, nén `.zip`.
3. Trả `.zip`, trình duyệt tải xuống.
**Luồng phụ**: GIF chỉ 1 frame → vẫn tách ra 1 file, không lỗi; GIF lỗi/quá lớn → 400.

## UC-11 — Thêm chữ (caption) vào GIF
**Actor**: khách
**Luồng chính**:
1. Vào `/tools/caption`, chọn GIF, nhập nội dung chữ, chọn vị trí (trên/dưới), màu, cỡ chữ.
2. Bấm "Thêm chữ" → server vẽ chữ lên từng frame ở cùng vị trí, trả GIF mới.
**Luồng phụ**:
- Nội dung chữ rỗng → 400 lỗi tham số (không cho submit rỗng).
- Chữ dài hơn chiều rộng GIF → tự xuống dòng hoặc thu nhỏ cỡ chữ (quyết định kỹ thuật, nêu rõ trong UI khi test).
- GIF input lỗi/quá lớn → 400.

## UC-12 — Quá tải hệ thống (áp dụng chung mọi công cụ)
**Actor**: khách
**Luồng chính**:
1. Nhiều người cùng xử lý → vượt số slot xử lý đồng thời cho phép.
2. Request mới chờ trong hàng đợi hoặc nhận lỗi 429 "server đang quá tải, thử lại sau" nếu chờ quá lâu.
**Luồng phụ**: request xử lý quá thời gian timeout (60s) → server huỷ, trả lỗi 504, dọn file tạm (nếu có) qua `defer`.
