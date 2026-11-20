# BA — GIF Toolkit (bộ công cụ GIF trên sync server, kiểu ezgif)

## 1. Mục đích

Thêm vào `cmd/syncserver` (+ `portal`) một bộ công cụ xử lý GIF chạy trên server, dùng ngay trong trình duyệt — không cần tài khoản, không cần cài gì — tương tự các công cụ rời rạc trên **ezgif.com**. Đây là tính năng độc lập với phần đồng bộ/chia sẻ GIF (`BA-sync-share.md`); không đụng tới bảng `users`/`gifs`/`sessions` đã có.

Tham khảo cốt lõi của ezgif.com, thu hẹp về các công cụ **thường dùng nhất** trước, chưa làm bản đầy đủ:

1. **Convert to GIF**: Video → GIF (mp4/webm/mov), Ảnh → GIF (1 ảnh tĩnh), nhiều Ảnh → GIF (ghép thành animation).
2. **GIF Resize** — đổi kích thước (theo % hoặc width/height, giữ hoặc không giữ tỉ lệ).
3. **GIF Crop** — cắt theo khung chữ nhật (x, y, width, height).
4. **GIF Optimize** (giảm dung lượng) — giảm số màu / lossy compression, có thể chọn mức.
5. **GIF Rotate** — xoay 90/180/270 độ, lật ngang/dọc.
6. **GIF Reverse** — đảo thứ tự frame.
7. **GIF Speed** — tăng/giảm tốc độ phát (đổi delay từng frame theo hệ số `factor`).
8. **GIF to frames (split)** — xuất từng frame thành ảnh PNG, trả về file `.zip`.
9. **GIF Add text/caption** — chèn 1 dòng chữ (nội dung, vị trí, màu, cỡ chữ) lên tất cả frame.

## 2. Vì sao xử lý ở server (không ở client/app desktop)

- Đây là tính năng web, dùng được từ máy không cài Ancol-Ani — không gắn với app widget (`app.go`, `frontend/`, `web/`).
- Xử lý ảnh/video tốn CPU/RAM — nên xếp hàng (giống `gifDecodeSlots` đã có ở `gifguard.go`) để không làm sập server đang chạy trên điện thoại Termux (xem `CLAUDE.md` mục Production).
- Video → GIF cần `ffmpeg`; các việc còn lại thao tác trực tiếp trên GIF (decode/encode qua `image/gif` của Go, không cần ffmpeg).

## 3. Kiến trúc

- **Trang mới trong portal** (`cmd/syncserver/portal`, Vue 3 + Vue Router, theme `ui/` dùng chung với `web/`): `/tools` — danh sách công cụ; mỗi công cụ 1 trang con, dùng chung layout "upload → chỉnh tham số → xem trước → tải kết quả".
- **Không cần đăng nhập** — các trang `/tools/*` công khai như `/gallery`, không đọc token.
- **API mới trên server**, tiền tố `/api/tools/...`, không dùng chung middleware `requireAuth`:

| Method | Path | Input | Output |
|---|---|---|---|
| POST | `/api/tools/convert` | multipart: `file` (video hoặc ảnh, có thể nhiều `file` cho multi-image), `fps` (tuỳ chọn, ảnh→gif) | `image/gif` |
| POST | `/api/tools/resize` | multipart: `file` (gif), `width`, `height` (giá trị tuyệt đối theo pixel; không có tham số `percent`) | `image/gif` |
| POST | `/api/tools/crop` | multipart: `file`, `x`, `y`, `width`, `height` | `image/gif` |
| POST | `/api/tools/optimize` | multipart: `file`, `level` (1-3; không có tham số `colors` riêng) | `image/gif` |
| POST | `/api/tools/rotate` | multipart: `file`, `angle` (90/180/270), `flip` (`none`/`h`/`v`) | `image/gif` |
| POST | `/api/tools/reverse` | multipart: `file` | `image/gif` |
| POST | `/api/tools/speed` | multipart: `file`, `factor` (vd 0.5 = chậm 2 lần, 2 = nhanh 2 lần) | `image/gif` |
| POST | `/api/tools/split` | multipart: `file` | `application/zip` (các `frame_001.png`, ...) |
| POST | `/api/tools/caption` | multipart: `file`, `text`, `position` (`top`/`bottom`), `color`, `size` | `image/gif` |

Mỗi endpoint xử lý xong **trả file ngay trong response** (không lưu DB, không lưu ổ đĩa lâu dài) — người dùng tải xuống hoặc dùng làm input cho công cụ tiếp theo (tải lại lên). Giữ đơn giản ở vòng đầu; chưa làm "chuỗi xử lý" nhiều bước trên server.

## 4. Giới hạn & an toàn (bắt buộc, học từ `gifguard.go` đã có)

Số liệu đã chốt (hằng số mới, đặt cùng khối với `gifguard.go`, ví dụ file `limits.go`):

| Hằng số | Giá trị | Áp dụng |
|---|---|---|
| `maxVideoBytes` | 20MB | upload video cho `/api/tools/convert` (tab Video) |
| `maxVideoDurationSec` | 30 giây | đọc qua `ffprobe` trước khi chạy `ffmpeg`, vượt thì từ chối ngay |
| `maxImageBytes` | 20MB | mỗi ảnh (tab Ảnh, tab Nhiều ảnh) |
| `maxImagesCount` | 100 | số ảnh tối đa khi ghép nhiều ảnh → 1 GIF |
| `maxConcurrentToolJobs` | 2 | số job xử lý đồng thời cho **toàn bộ** nhóm `/api/tools/*` (1 pool riêng, không dùng chung `gifDecodeSlots` của luồng sync/share — tránh 2 tính năng tranh nhau) |
| `toolRequestTimeout` | 60 giây | timeout tổng cho mỗi request (tính cả thời gian chờ slot) |

- **Input GIF**: mọi endpoint nhận GIF phải chạy qua `inspectGif`/`checkGifSize` (đã có ở `gifguard.go`) trước khi `DecodeAll` — chặn GIF bomb giống cơ chế hiện tại (`maxGifSide`=1024, `maxGifFrames`=500, `maxGifTotalPixels`=64MB).
- **Input video**: chặn theo `maxVideoBytes` (kiểm tra `Content-Length`/đọc giới hạn trước khi ghi file tạm) và `maxVideoDurationSec` (qua `ffprobe`, chạy trước `ffmpeg`, không convert thử rồi mới biết dài).
- **Input ảnh** (convert nhiều ảnh → GIF): `maxImageBytes` mỗi ảnh, `maxImagesCount` tổng số ảnh; decode qua `image.DecodeConfig` để biết kích thước trước khi decode toàn bộ.
- **Giới hạn kích thước/số frame đầu ra**: mọi output GIF cũng phải tuân `maxGifSide`/`maxGifFrames`/`maxGifTotalPixels` — nếu resize/convert tạo ra GIF vượt ngưỡng thì trả lỗi rõ ràng (không cố gắng xử lý rồi crash).
- **Giới hạn đồng thời (hành vi thực tế đã triển khai)**: pool riêng theo `maxConcurrentToolJobs`, và slot chỉ được giữ trong bước xử lý CPU/RAM (encode/decode GIF, chạy `ffmpeg`) — KHÔNG giữ trong lúc đọc/parse body multipart, để một upload chậm/cố ý trì hoãn không chiếm slot của người khác. Khi pool đầy, request chờ tối đa `poolFullWait` (hiện ~50ms) rồi nhận lỗi 429 ngay — không xếp hàng tới hết `toolRequestTimeout` rồi mới báo lỗi.
- **Timeout**: `toolRequestTimeout` (60 giây) là trần thời gian toàn bộ cho một request (middleware `withToolLimits`) — quá hạn thì huỷ bằng `context.WithTimeout`, trả lỗi 504, không treo worker. Đây là trần tổng, độc lập với cơ chế chờ-slot-rồi-429 ở trên.
- **Dọn file tạm**: nếu endpoint nào cần ghi file tạm ra đĩa (ví dụ gọi `ffmpeg` cần input là file, không phải stream), phải xoá file tạm ngay sau khi xử lý xong, dùng thư mục riêng (vd `data/tmp/`), kể cả khi lỗi giữa chừng (đảm bảo bằng `defer`).
- **Không log nội dung file** người dùng tải lên (chỉ log kích thước, loại, kết quả thành công/lỗi) — tránh phình log trên máy Termux.

## 5. Phụ thuộc mới

- `ffmpeg` phải có sẵn trên máy chạy server (kiểm tra lúc khởi động `syncserver`, log cảnh báo rõ nếu thiếu — route `/api/tools/convert` cho video trả lỗi 503 "server thiếu ffmpeg" thay vì crash).
- Không cần thêm Go module cho riêng GIF (dùng `image/gif` chuẩn); convert ảnh tĩnh→GIF dùng `image/png`, `image/jpeg` chuẩn.

## 6. Goals / Non-goals / Anti-goals

**Goals**
- G1: 9 công cụ ở mục 1, mỗi công cụ 1 trang độc lập trong portal, không cần tài khoản.
- G2: Mọi input/output đều có giới hạn rõ ràng (kích thước, số frame, thời lượng, số lượng), không có đường nào decode không giới hạn.
- G3: Xử lý xong trả file ngay, không lưu trữ lâu dài trên server (không phình `data/`).
- G4: Lỗi (file sai định dạng, vượt giới hạn, quá tải, thiếu ffmpeg) trả thông báo rõ, không crash server.

**Non-goals (chưa làm ở vòng này)**
- Xử lý theo lô (batch nhiều file 1 lúc) ngoài convert nhiều ảnh → 1 GIF.
- Lưu lịch sử/kết quả đã xử lý theo tài khoản (không tích hợp với bảng `gifs` đã có).
- Chuỗi xử lý nhiều công cụ liên tiếp trên server (resize rồi crop rồi optimize trong 1 request) — người dùng tự tải lên lại giữa các bước.
- Các công cụ khác của ezgif ngoài 9 công cụ ở mục 1 (ví dụ: thêm overlay ảnh, GIF maker từ webcam, meme generator, video editor...).
- Giới hạn theo IP/rate-limit riêng cho nhóm API này (có thể tái dùng `ratelimit.go` đã có sau, ngoài scope BA này).

**Anti-goals**
- Decode file người dùng tải lên mà không kiểm tra kích thước/giới hạn trước — PHẢI chặn kiểu GIF/ảnh bomb như đã làm với `gifguard.go`.
- Giữ file người dùng tải lên trên đĩa lâu hơn thời gian xử lý 1 request.
- Chạy `ffmpeg`/xử lý ảnh không có timeout hoặc không có giới hạn số job đồng thời.

## 7. Việc cần làm (chưa code — để BA chốt trước)

1. `cmd/syncserver/gif_ops.go` (tên thực tế đã triển khai, không phải `tools.go`): các hàm xử lý GIF thuần Go — resize, crop, rotate, reverse, speed, split, caption (dùng `image`, `image/gif`, `image/draw`; chữ caption dùng `golang.org/x/image/font/basicfont` hoặc font nhúng sẵn, không cần thêm dependency nặng).
2. `cmd/syncserver/convert.go`: convert video→GIF (gọi `ffmpeg` qua `os/exec`, input/output qua file tạm trong `data/tmp/`), ảnh/nhiều ảnh→GIF (decode PNG/JPEG, ghép frame bằng `image/gif`).
3. `cmd/syncserver/tools_handlers.go` (tên thực tế đã triển khai, không phải `handlers.go` — file `handlers.go` đã có từ trước và chứa route của tính năng sync/share, không liên quan tới GIF Toolkit): 9 route `/api/tools/...`, không qua `requireAuth`.
4. `cmd/syncserver/main.go`: kiểm tra `ffmpeg` có trong `PATH` lúc khởi động, tạo `data/tmp/` nếu chưa có.
5. Thêm file mới `cmd/syncserver/limits.go` với 6 hằng số ở mục 4 (`maxVideoBytes`, `maxVideoDurationSec`, `maxImageBytes`, `maxImagesCount`, `maxConcurrentToolJobs`, `toolRequestTimeout`) + pool slot riêng (`toolJobSlots := make(chan struct{}, maxConcurrentToolJobs)`) — tái dùng `inspectGif`/`checkGifSize` của `gifguard.go` cho mọi output GIF.
6. `cmd/syncserver/portal`: trang `/tools` (danh sách, mỗi card 1 công cụ) + 9 trang con `/tools/{ten-cong-cu}`, dùng theme `ui/` sẵn có, GIF preview bằng canvas (giống `GifCanvas.vue` đã có, không dùng `<img>`).
7. Thêm link "Công cụ GIF" vào thanh điều hướng của portal (file điều hướng chung, cạnh `/gallery`).
