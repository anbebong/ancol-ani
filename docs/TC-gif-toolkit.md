# TC — GIF Toolkit

Tham chiếu: `docs/UC-gif-toolkit.md`. Cột "UC" trỏ tới use case gốc.

## Convert (UC-01, UC-02, UC-03)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-01 | UC-01 | Convert video mp4 ngắn (5s) hợp lệ | Upload mp4 5s, bấm Chuyển đổi | Trả `image/gif`, preview phát được, nhiều frame |
| TC-02 | UC-01 | Convert webm, mov | Upload lần lượt webm, mov hợp lệ | Cả hai convert thành công như TC-01 |
| TC-03 | UC-01 | Video vượt giới hạn thời lượng | Upload video 60s (giới hạn 30s) | 400, thông báo "video quá dài", không chạy ffmpeg |
| TC-04 | UC-01 | File không phải video | Upload file `.txt` đổi tên `.mp4` | 400 "file không hợp lệ" |
| TC-05 | UC-01 | Server thiếu ffmpeg | Giả lập biến môi trường thiếu ffmpeg (hoặc sửa PATH khi test) | 503 "server thiếu ffmpeg", server không crash |
| TC-06 | UC-01 | Chọn FPS đầu ra | Convert với fps=10 và fps=24 | GIF kết quả có số frame/giây tương ứng khác nhau |
| TC-07 | UC-02 | Convert 1 ảnh PNG | Upload PNG hợp lệ | Trả GIF 1 frame, kích thước giống ảnh gốc |
| TC-08 | UC-02 | Convert 1 ảnh JPEG | Upload JPEG hợp lệ | Trả GIF 1 frame |
| TC-09 | UC-02 | Ảnh vượt 20MB | Upload ảnh >20MB | 400 "ảnh quá lớn" |
| TC-10 | UC-02 | File ảnh sai định dạng | Upload file `.pdf` đổi tên `.png` | 400 "file không hợp lệ" |
| TC-11 | UC-03 | Ghép 5 ảnh cùng kích thước | Chọn 5 ảnh PNG cùng size, delay=200ms, Chuyển đổi | Trả GIF 5 frame, đúng thứ tự đã chọn, đúng delay |
| TC-12 | UC-03 | Vượt số lượng ảnh | Chọn 150 ảnh (giới hạn 100) | 400 "vượt số lượng ảnh cho phép" |
| TC-13 | UC-03 | 1 ảnh trong danh sách lỗi | 4 ảnh hợp lệ + 1 file sai định dạng | 400, thông báo nêu rõ ảnh thứ mấy lỗi |
| TC-14 | UC-03 | Ảnh kích thước khác nhau | Chọn ảnh có size khác nhau | Convert vẫn thành công, kết quả co giãn/cắt theo ảnh đầu (đúng như hành vi đã định nghĩa, không crash) |

## Resize (UC-04)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-15 | UC-04 | Resize theo width/height hợp lệ | GIF 300x300, resize về 150x150 | Trả GIF 150x150, giữ số frame gốc |
| TC-16 | UC-04 | Resize theo % | GIF 300x300, resize 50% | Trả GIF 150x150 |
| TC-17 | UC-04 | Width/height = 0 | Nhập width=0 | 400 lỗi tham số |
| TC-18 | UC-04 | Width/height âm | Nhập height=-10 | 400 lỗi tham số |
| TC-19 | UC-04 | Resize vượt `maxGifSide` | Nhập width=2000 (giới hạn 1024) | 400 "kích thước vượt giới hạn" |
| TC-20 | UC-04 | GIF input là GIF bomb | Upload GIF frame nhỏ nhưng khai 10000 frame | 400 (`errGifTooBig`), không treo server, không OOM |

## Crop (UC-05)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-21 | UC-05 | Crop khung hợp lệ trong biên | GIF 300x300, crop (50,50,100,100) | Trả GIF 100x100 đúng vùng chọn |
| TC-22 | UC-05 | Khung cắt vượt biên phải/dưới | GIF 300x300, crop (250,250,100,100) | 400 "khung cắt vượt ra ngoài ảnh" |
| TC-23 | UC-05 | Width/height crop = 0 | crop (10,10,0,50) | 400 lỗi tham số |
| TC-24 | UC-05 | GIF input lỗi/quá lớn | Upload GIF vượt `maxGifSide` | 400, như TC-20 |

## Optimize (UC-06)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-25 | UC-06 | Optimize mức 1/2/3 | GIF dung lượng lớn, chọn từng mức | Dung lượng sau < trước ở cả 3 mức, mức 3 nhỏ nhất, vẫn phát được |
| TC-26 | UC-06 | Optimize GIF đã nhỏ | GIF rất nhẹ, optimize mức 3 | Không lỗi, trả GIF hợp lệ (có thể gần như không đổi dung lượng) |
| TC-27 | UC-06 | GIF input lỗi/quá lớn | Upload GIF vượt giới hạn | 400, như TC-20 |

## Rotate / Flip (UC-07)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-28 | UC-07 | Xoay 90/180/270 | GIF chữ nhật, xoay từng góc | Kích thước đổi đúng theo góc (90/270 đổi w/h), ảnh xoay đúng chiều |
| TC-29 | UC-07 | Lật ngang, lật dọc | Áp dụng flip=h, flip=v riêng | Ảnh lật đúng chiều, kích thước không đổi |
| TC-30 | UC-07 | Không chọn gì (angle=0, flip=none) | Submit không đổi tham số | Trả GIF giống bản gốc, không lỗi |
| TC-31 | UC-07 | GIF input lỗi/quá lớn | Upload GIF vượt giới hạn | 400, như TC-20 |

## Reverse (UC-08)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-32 | UC-08 | GIF nhiều frame | GIF 10 frame, đảo ngược | Thứ tự frame đảo đúng, delay từng frame đi theo đúng frame của nó sau khi đảo |
| TC-33 | UC-08 | GIF 1 frame | GIF chỉ 1 frame, đảo ngược | Trả nguyên bản, không lỗi |
| TC-34 | UC-08 | GIF input lỗi/quá lớn | Upload GIF vượt giới hạn | 400, như TC-20 |

## Speed (UC-09)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-35 | UC-09 | Factor 2x (nhanh gấp đôi) | GIF delay 10cs/frame, factor=2 | Delay mới ~5cs/frame |
| TC-36 | UC-09 | Factor 0.5x (chậm gấp đôi) | GIF delay 10cs/frame, factor=0.5 | Delay mới ~20cs/frame |
| TC-37 | UC-09 | Factor = 0 hoặc âm | Nhập factor=0 | 400 lỗi tham số |
| TC-38 | UC-09 | Factor quá lớn làm delay dưới mức tối thiểu | factor=100 trên delay nhỏ | Delay ghim về mức tối thiểu hợp lệ của GIF, không lỗi, không delay=0 |

## Split (UC-10)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-39 | UC-10 | GIF nhiều frame | GIF 10 frame, tách frame | Trả `.zip` chứa đúng 10 file PNG, đặt tên tuần tự, mở được từng ảnh |
| TC-40 | UC-10 | GIF 1 frame | GIF 1 frame, tách frame | Trả `.zip` chứa 1 file PNG |
| TC-41 | UC-10 | GIF input lỗi/quá lớn | Upload GIF vượt giới hạn | 400, như TC-20 |

## Caption (UC-11)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-42 | UC-11 | Chèn chữ hợp lệ, vị trí trên | text="Hello", position=top | Chữ hiện trên mọi frame, đúng vị trí trên |
| TC-43 | UC-11 | Chèn chữ vị trí dưới, đổi màu/cỡ | position=bottom, color khác mặc định, size lớn | Chữ hiện đúng vị trí, đúng màu, đúng cỡ |
| TC-44 | UC-11 | Nội dung chữ rỗng | text="" | 400 lỗi tham số, không submit được / server chặn |
| TC-45 | UC-11 | Chữ dài hơn chiều rộng GIF | text rất dài trên GIF nhỏ | Không crash — tự xuống dòng hoặc thu nhỏ cỡ chữ theo hành vi đã chốt, kiểm tra thực tế khi code |
| TC-46 | UC-11 | GIF input lỗi/quá lớn | Upload GIF vượt giới hạn | 400, như TC-20 |

## Giới hạn hệ thống / quá tải (UC-12, áp dụng chung)

| TC | UC | Mô tả | Bước | Kỳ vọng |
|---|---|---|---|---|
| TC-47 | UC-12 | Vượt số slot xử lý đồng thời | Gửi đồng thời nhiều request hơn số slot cho phép | Request vượt slot chờ hàng đợi hoặc nhận 429, server không crash, các request trong slot vẫn xử lý đúng |
| TC-48 | UC-12 | Request quá timeout | Giả lập xử lý kéo dài hơn 60s (file lớn/chậm) | Server huỷ, trả 504, file tạm (nếu có) bị xoá |
| TC-49 | mọi UC | Không đăng nhập vẫn dùng được | Gọi bất kỳ `/api/tools/*` không kèm token | Không bị 401 — các API này công khai theo thiết kế |
| TC-50 | mọi UC | File tạm được dọn sau xử lý | Convert video (sinh file tạm), kiểm tra `data/tmp/` sau request | Thư mục `data/tmp/` không còn file rác sau khi request (thành công hoặc lỗi) kết thúc |
