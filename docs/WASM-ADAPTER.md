# Text adapter (.wasm) - chuẩn ABI

Một **adapter** là 1 file `.wasm` cung cấp chữ hiển thị trên widget tự động,
thay cho gõ tay. Khách import file này vào Settings, không cần build lại
client. App chạy nó qua [wazero](https://github.com/tetratelabs/wazero)
(thuần Go, sandbox hoàn toàn: không filesystem, không socket - xem
`wasmsource.go`).

Hợp đồng chỉ có 1 ý duy nhất: **app đưa cấu hình vào, adapter trả chữ ra**.
Mọi logic nghiệp vụ (gọi API nào, parse ra sao, bao lâu gọi lại) nằm hết
trong adapter, app không biết và không cần biết.

## Export bắt buộc

Mọi adapter phải export đúng các hàm sau (tên và kiểu phải khớp chính xác):

| Hàm | Kiểu | Mô tả |
|---|---|---|
| `alloc` | `(i32) -> i32` | Nhận `size`, trả về con trỏ tới vùng nhớ trống `size` byte trong chính adapter. App dùng hàm này để ghi dữ liệu (config, response HTTP) vào bộ nhớ adapter mà không đè lên code/data của nó. |
| `config_schema` | `() -> i64` | Trả về **packed(ptr, len)** của 1 chuỗi UTF-8 chứa JSON mô tả field cấu hình cần, ví dụ `[{"key":"api_key","label":"API Key"}]`. Trả `[]` nếu không cần cấu hình gì. Settings tự vẽ form từ đây - app không biết trước field nào. |
| `configure` | `(i32, i32) -> ()` | App gọi 1 lần mỗi khi cấu hình được lưu, tham số là `(ptr, len)` của chuỗi JSON giá trị khách đã điền (key khớp `config_schema`). Adapter tự lưu lại nội bộ để `poll` dùng sau. |
| `poll` | `() -> i64` | App gọi theo chu kỳ. Trả về **packed(ptr, len)** của chuỗi text cần hiển thị (rỗng = "chưa có gì mới, giữ nguyên"). Nhiều dòng (`\n`) = nhiều câu xoay vòng trên widget, giống hệt gõ tay nhiều dòng. |

### `info` - bắt buộc

| Hàm | Kiểu | Mô tả |
|---|---|---|
| `info` | `() -> i64` | packed(ptr, len) của 1 chuỗi JSON object mô tả chính adapter. |

4 trường **bắt buộc** trong JSON đó (thiếu/rỗng là import bị từ chối):

| Key | Ý nghĩa |
|---|---|
| `name` | Tên module |
| `description` | Chức năng (adapter làm gì) |
| `author` | Thông tin người tạo |
| `version` | Phiên bản adapter |

Có thể thêm **bất kỳ key tùy chọn khác** (ví dụ `homepage`, `contact`...).
**Mọi trường, bắt buộc hay tùy chọn, đều tối đa 255 ký tự** - dài hơn là
import bị từ chối. Ví dụ:

```json
{
  "name": "Thong bao CRM noi bo",
  "description": "Lay thong bao chua doc tu CRM qua REST API",
  "author": "Pham Thanh An <angokodau@gmail.com>",
  "version": "1.0.0"
}
```

## Quy ước packed(ptr, len)

Một giá trị `i64` duy nhất gói cả con trỏ và độ dài:

```
packed = (ptr << 32) | len
```

`ptr`/`len` đều là `uint32`. App tách ra bằng cách shift phải 32 bit lấy
`ptr`, ép kiểu `uint32` lấy `len`. Chuỗi luôn là UTF-8, nằm trong bộ nhớ
tuyến tính (linear memory) của chính adapter.

## Host import - cách duy nhất gọi mạng

wasm không có socket. App cấp sẵn 1 import duy nhất:

```
env.http_fetch: (urlPtr i32, urlLen i32) -> i64   // packed(respPtr, respLen)
```

Adapter tự xây `url` (vd gắn `api_key` đã nhận ở `configure`), ghi vào
bộ nhớ của mình, gọi `http_fetch(urlPtr, urlLen)`. App tự làm `GET` HTTP
thật, ghi response thô vào bộ nhớ adapter (qua `alloc` của chính adapter),
trả về `packed(ptr, len)` của response đó. Thất bại (lỗi mạng, response
rỗng) trả về `(0, 0)` - **không có mã lỗi riêng**, adapter tự quyết định
response rỗng nghĩa là gì.

Chỉ hỗ trợ `GET`. Cần method/header khác thì chưa có - mở rộng sau nếu cần.

## Field cấu hình đặc biệt: `_interval_sec`

Muốn cho khách tự chỉnh chu kỳ gọi `poll` (giây), adapter khai báo field
`_interval_sec` ngay trong `config_schema` như field bình thường - Settings
sẽ tự vẽ ô số riêng cho nó (không hiện kiểu text như field khác). **Không
khai báo thì ô này không hiện ra** - adapter không khai nghĩa là tự quyết
chu kỳ của mình, không cho khách chỉnh. App đọc giá trị khách lưu (nếu có)
làm chu kỳ gọi; không có thì mặc định 10s (giới hạn 2s–600s). Không cần
export riêng cho việc này.

## Vòng đời & lỗi

- Nhiều adapter có thể import cùng lúc, mỗi cái lưu riêng (file + config
  riêng). Chỉ 1 được chọn chạy (`ActiveAdapterID` + `AdapterEnabled`).
- `poll` trả lỗi Go thật (trap, ví dụ `unreachable`, out-of-bounds...) -
  không phải chuỗi rỗng - khiến app **tự động chuyển về chế độ Thủ công**,
  không tự thử lại. Phải người dùng tự chọn lại/enable lại sau khi sửa.
- Text thủ công (ô nhập tay) và text adapter **hoàn toàn độc lập**: adapter
  không bao giờ ghi đè lên text thủ công đã lưu, chỉ đẩy trực tiếp ra
  widget khi đang ở chế độ Tự động. Tắt Tự động là quay lại đúng text thủ
  công, không mất gì.

## Build thử nhanh

Nên viết adapter thật bằng **TinyGo** hoặc **Rust (`wasm32-unknown-unknown`)**
- cả hai đều export được hàm trực tiếp sang WASM, dễ đọc hơn hẳn so với tự
ghép bytes WASM tay. `alloc` cần một bump allocator đơn giản (vài dòng),
`info`/`config_schema`/`poll` chỉ là trả về 1 chuỗi UTF-8 qua
packed(ptr,len) như mô tả ở trên.

Có 3 file `.wasm` ví dụ (không cần toolchain gì, không parse JSON thật -
chỉ để test đường ống): `build/bin/test-adapter.wasm` (trả cố định 2 câu,
có field cấu hình giả), `build/bin/http-adapter.wasm` (gọi `env.http_fetch`
thật tới 1 URL cứng), `build/bin/error-adapter.wasm` (luôn lỗi, để test
tự fallback về Thủ công). Dùng `POST /text-adapter` (multipart field
`wasm`) để import thử, `GET /text-adapter` (không body) để xem danh sách
và lỗi gần nhất.
