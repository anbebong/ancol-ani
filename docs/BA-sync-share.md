# BA — Đồng bộ & Chia sẻ GIF (bản sửa)

## 1. Mục đích
- Người dùng có tài khoản thật (user/pass), đăng nhập trên nhiều máy → GIF đồng bộ theo tài khoản đó.
- Người dùng có thể chia sẻ GIF công khai cho người khác (không cần tài khoản để *xem*).

## 2. Kiến trúc tổng thể
- **Web portal / API server**: 1 service chạy độc lập (không phải chạy trên máy client), có database riêng. Lúc dev thì chạy ở `localhost` để test, nhưng kiến trúc không gắn với máy client nào — triển khai lên VPS/cloud là đổi domain, không đổi code.
- **Client (app `ancol-ani`)**: gọi tới portal này qua API, không tự lưu "danh tính" nào ngoài token đăng nhập.

## 3. Mô hình dữ liệu (database, dùng SQLite — nhẹ, không cần cài DB server riêng, vẫn chạy được khi deploy thật)

```
users
  id            INTEGER PK
  username      TEXT UNIQUE NOT NULL
  password_hash TEXT NOT NULL      -- bcrypt
  created_at    DATETIME

sessions
  token         TEXT PK            -- random, trả về sau khi login
  user_id       INTEGER FK -> users.id
  created_at    DATETIME
  expires_at    DATETIME

gifs
  id            INTEGER PK
  user_id       INTEGER FK -> users.id   -- người sở hữu
  data          BLOB hoặc đường dẫn file trên đĩa
  is_current    BOOLEAN            -- GIF đang active của user này (dùng để đồng bộ)
  is_public     BOOLEAN
  slug          TEXT UNIQUE NULL   -- có giá trị khi is_public = true
  created_at    DATETIME
```

Vì sao SQLite: không cần người dùng tự cài Postgres/MySQL để chạy thử, 1 file `.db` là xong, nhưng vẫn là DB thật (không phải lưu file thủ công như bản trước) — nếu sau này cần scale thì đổi sang Postgres cũng chỉ sửa tầng truy cập dữ liệu.

## 4. Luồng xác thực
1. **Đăng ký**: `POST /auth/register {username, password}` → tạo user, trả về lỗi nếu username đã tồn tại.
2. **Đăng nhập**: `POST /auth/login {username, password}` → trả `{token}`. Token lưu trong `sessions`, có hạn (vd 30 ngày).
3. Mọi API sau đó gửi kèm `Authorization: Bearer {token}`.
4. App lưu token đã đăng nhập vào `%APPDATA%\ancol-ani\session.json` — máy nào đăng nhập cùng 1 tài khoản thì tự nhiên "đồng bộ" theo `user_id`, không cần device key nào nữa (bỏ hẳn cơ chế device-key ở bản trước).

## 5. API (thiết kế lại, thay toàn bộ bản cũ)
| Method | Path | Auth | Mô tả |
|---|---|---|---|
| POST | `/auth/register` | không | Tạo tài khoản |
| POST | `/auth/login` | không | Đăng nhập, trả token |
| GET | `/me/gif` | có | Lấy GIF `is_current=true` của user |
| POST | `/me/gif` | có | Upload GIF mới, set làm `is_current` |
| POST | `/me/gif/share` | có | Đánh dấu GIF hiện tại `is_public=true`, sinh `slug`, trả về |
| POST | `/me/gif/unshare` | có | Gỡ public (`is_public=false`) |
| GET | `/public` | không | Danh sách GIF public (phân trang, mới nhất trước) |
| GET | `/public/{slug}` | không | Lấy 1 GIF public theo slug |

## 6. Luồng người dùng
### Đồng bộ
1. Máy A: mở settings → chưa đăng nhập → form đăng ký/đăng nhập (username + password).
2. Máy A đổi GIF → app tự `POST /me/gif` (hoặc bấm nút "Đồng bộ lên").
3. Máy B: đăng nhập **cùng tài khoản** → bấm "Lấy GIF từ tài khoản" → `GET /me/gif` → set làm GIF local.

### Chia sẻ công khai
1. User đã đăng nhập, có GIF hiện tại → bấm "Chia sẻ công khai" → `POST /me/gif/share` → nhận link `/public/{slug}`.
2. Người khác (không cần tài khoản) mở `GET /public` trong app của họ → xem danh sách → chọn 1 cái → set làm GIF local.

## 7. Goals / Non-goals / Anti-goals
**Goals**
- G1: Tài khoản thật (username/password), nhiều máy đăng nhập cùng tài khoản thì thấy cùng GIF.
- G2: Chia sẻ công khai qua slug, ai cũng xem được không cần tài khoản.
- G3: Server là 1 service độc lập, deploy ở đâu cũng được, không phụ thuộc máy client nào.
- G4: Mật khẩu không lưu plaintext (bcrypt hash).

**Non-goals (chưa làm)**
- Quên mật khẩu / đổi mật khẩu / xoá tài khoản.
- Real-time sync (vẫn là bấm nút, không tự động đẩy khi đổi GIF).
- Kiểm duyệt nội dung public.
- Giới hạn số GIF public/giới hạn dung lượng tài khoản.
- Deploy thật lên domain/VPS (chỉ chạy localhost để test).

**Anti-goals**
- Lưu password dạng plaintext hoặc hash yếu (MD5/SHA1 trần) — PHẢI bcrypt.
- Token không có hạn / không thể thu hồi.
- Một request không có token vẫn sửa được GIF của người khác.

## 8. Việc cần làm (chưa code gì theo hướng này, code cũ ở `cmd/syncserver` sẽ viết lại từ đầu)
1. Thêm dependency SQLite driver (pure Go, không cần cgo) + bcrypt (`golang.org/x/crypto/bcrypt`).
2. Viết lại `cmd/syncserver`: schema migration, các endpoint ở mục 5.
3. Sửa app phía client: bỏ device-key, thêm form đăng nhập/đăng ký trong settings page, lưu token, gọi API mới.

## 9. Quyết định (đã chốt với user)

**9.1 DB linh hoạt theo config, không sửa logic**
- Dùng `sqlx` (`github.com/jmoiron/sqlx`) cho tầng truy cập dữ liệu — hỗ trợ named query (`:param`), tự rebind theo dialect, nên business code viết 1 lần dùng được cho nhiều driver.
- Driver chọn qua config (`-driver` / env `DB_DRIVER`): `sqlite` (mặc định, dùng `modernc.org/sqlite` — pure Go, không cần cgo) hoặc `postgres` (`-dsn` khác). Thêm driver mới = thêm 1 case trong hàm tạo schema + import driver, không đụng tới handler/business logic.
- Schema (CREATE TABLE) có khác biệt nhỏ giữa dialect (ví dụ AUTOINCREMENT vs SERIAL) — việc này nằm ở bước khởi tạo DB (migration), không phải logic nghiệp vụ, nên vẫn tính là "chỉ cấu hình".
- Ở bước này: cài đầy đủ & test cho **SQLite**. Case cho `postgres` viết sẵn khung (switch-case) nhưng không bắt buộc test ngay (ngoài scope hiện tại), chỉ cần bật lên là chạy được.

**9.2 JWT theo chuẩn hiện đại, có thu hồi**
- Dùng JWT (HS256, `golang-jwt/jwt/v5`), claims gồm `sub` (user id), `jti` (id riêng mỗi token), `exp`.
- Hạn token: **1 giờ** (theo yêu cầu — token sống ngắn, giảm rủi ro nếu bị lộ; nhược điểm là app phải đăng nhập lại mỗi giờ vì hiện chưa làm refresh token, chấp nhận đánh đổi này ở bước này).
- Thu hồi: mỗi token phát ra có 1 dòng trong bảng `sessions` (jti, user_id, expires_at, revoked_at). Mỗi request xác thực: verify chữ ký + hạn JWT **và** kiểm tra `sessions` chưa bị revoke. Logout = set `revoked_at`. Đây là lý do có JWT (tự chứa thông tin, verify nhanh bằng chữ ký) **và** vẫn thu hồi được (không phải JWT thuần stateless không thu hồi được).

**9.3 Giới hạn 1 GIF/user**
- Bảng `gifs` có unique constraint theo `user_id` (1 user — 1 row). Đổi GIF = update đè, không tạo row mới. Mở rộng nhiều GIF sau này là việc khác, ngoài scope.

## 10. Bổ sung — Web portal (user + admin)

Server (`cmd/syncserver`) hiện chỉ là API thuần (không có trang nào, gọi `/` sẽ 404). Cần thêm giao diện web thật, do chính server này phục vụ.

### 10.1 Phân quyền admin
- Dùng chung bảng `users`, thêm cột `is_admin BOOLEAN DEFAULT false`.
- Không có ai tự nâng quyền qua API — tài khoản admin đầu tiên set bằng tay (sửa trực tiếp DB hoặc 1 lệnh CLI nhỏ `syncserver -make-admin <username>`), tránh lỗ hổng "ai đăng ký cũng thành admin được".

### 10.2 Trang người dùng (ai cũng vào được)
- `/` — trang chủ: nếu chưa đăng nhập → form đăng ký/đăng nhập; nếu đã đăng nhập → thấy GIF hiện tại của mình, nút đổi GIF, nút chia sẻ công khai/gỡ chia sẻ, nút đăng xuất.
- `/gallery` — thư viện GIF public (ai cũng xem được, không cần đăng nhập), mỗi item preview bằng canvas (tái dùng cơ chế tự vẽ frame đã làm ở app, tránh lỗi gif treo).

### 10.3 Trang admin (chỉ `is_admin = true` vào được)
- `/admin` — yêu cầu đăng nhập + `is_admin`, nếu không đủ quyền thì chặn (403), không chỉ ẩn nút trên UI.
- Danh sách user: username, ngày tạo, có đang public gif hay không. Nút xoá user (xoá luôn gif của họ).
- Danh sách gif public (toàn hệ thống, không chỉ của admin): nút "Gỡ public" (admin gỡ được gif của người khác, không cần là chủ).

### 10.4 API bổ sung cần thêm (ngoài bảng ở mục 5)
| Method | Path | Auth | Mô tả |
|---|---|---|---|
| GET | `/admin/users` | admin | Danh sách user |
| DELETE | `/admin/users/{id}` | admin | Xoá user + gif của họ |
| DELETE | `/admin/public/{slug}` | admin | Gỡ 1 gif public bất kỳ |

### 10.5 Công nghệ UI
- Serve tĩnh từ chính `cmd/syncserver` qua `go:embed`, build bằng Vite (giống `web/` của app desktop, tái dùng style pixel/Tailwind cho nhất quán, nhưng là 1 project Vite riêng vì mục đích khác — không dùng chung code với `web/` của desktop app).
- Không dùng framework nặng (React/Vue) — vẫn vanilla JS như các phần trước, giữ app nhẹ.

## 11. Việc cần làm (bắt đầu code)
1. `cmd/syncserver/store.go`: schema (sqlite trước, khung sẵn cho postgres), các hàm `CreateUser`, `GetUserByUsername`, `UpsertCurrentGif`, `GetCurrentGif`, `SetPublic`/`Unpublish`, `GetPublicBySlug`, `ListPublic`, `CreateSession`, `IsSessionValid`, `RevokeSession`.
2. `cmd/syncserver/auth.go`: bcrypt hash/verify, sign/parse JWT, middleware xác thực.
3. `cmd/syncserver/handlers.go`: `/auth/register`, `/auth/login`, `/auth/logout`, `/me/gif` (GET/POST), `/me/gif/share`, `/me/gif/unshare`, `/public`, `/public/{slug}`.
4. `go.mod`: thêm `sqlx`, `modernc.org/sqlite`, `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`.
5. Xoá code cũ của `cmd/syncserver` (bản device-key, không DB) — thay hoàn toàn.
6. Sau khi server xong & test: sửa UI web (form đăng ký/đăng nhập, nút đồng bộ/chia sẻ) — làm ở bước sau, báo lại trước khi code UI.

## 12. Việc cần làm — Portal (bước tiếp theo, đang code)
1. `store.go`: thêm cột `is_admin`, hàm `ListUsers`, `DeleteUser`, `AdminUnpublish`, cờ `-make-admin <username>` trong `main.go`.
2. `handlers.go`: thêm `/admin/users` (GET), `/admin/users/{id}` (DELETE), `/admin/public/{slug}` (DELETE) — middleware `requireAdmin` (check `is_admin` sau khi `requireAuth`).
3. Project Vite mới `cmd/syncserver/portal/` (tách biệt `web/` của desktop app): trang `/` (đăng nhập/đăng ký + quản lý gif của mình), `/gallery` (public, không cần đăng nhập), `/admin` (danh sách user + gif public, nút xoá/gỡ).
4. `cmd/syncserver/main.go`: `go:embed portal/dist`, serve static tại `/`, `/gallery`, `/admin` (SPA đơn giản, không cần router phức tạp — 3 trang HTML riêng, không SPA).
5. Build & test thủ công toàn bộ luồng qua trình duyệt trước khi báo xong.

## 13. Bổ sung — Nhiều GIF/user (quota) [đã code]

Thay cho giới hạn 1 GIF/user ở mục 9.3 (đã bỏ):
- `users.gif_quota` — số slot GIF tối đa của user, mặc định `3`, admin chỉnh được theo từng user (`PATCH /admin/users/{id}/quota`) — quota là 1 thông tin của user, không phải hằng số toàn hệ thống.
- `gifs` là nhiều dòng/user (không còn unique theo `user_id`). **Không có khái niệm "active"** — không có GIF nào mặc định là "đang đồng bộ" ở phía server. Mỗi dòng có `is_public`/`slug` độc lập, bật/tắt chia sẻ không ảnh hưởng gì tới việc đồng bộ.
- "Đồng bộ" = client tự chọn 1 GIF cụ thể trong thư viện (`GET /me/gifs` để xem danh sách, `GET /me/gifs/{id}/image` để lấy đúng ảnh đó) rồi dùng theo ý mình — không có "ảnh hiện tại" nào server tự quyết.
- API: `GET/POST /me/gifs`, `DELETE /me/gifs/{id}`, `POST /me/gifs/{id}/share|unshare`, `GET /me/gifs/{id}/image`, `GET /me/gif/history` (lịch sử share, có `gif_id`), `GET /me` (role + quota của chính mình).

## 14. Bổ sung — Client lấy ảnh từ server (đang làm)

### 14.1 Vấn đề
`cmd/syncserver` + `portal` đã xong, nhưng app desktop (`app.go`, trang settings local `web/` ở `localhost:8788`) **chưa hề gọi tới syncserver**. Cần nối client vào, nhưng theo đúng ràng buộc đã chốt:
- Giao diện đăng nhập/token **chỉ sống ở web** (trang settings local, JS chạy trong WebView) — **không sửa `app.go`**, không lưu token ra đĩa phía Go.
- Token do portal phát ra là để **web local client** tự gọi syncserver lấy ảnh, rồi mới đẩy ảnh đó vào app qua endpoint sẵn có.

### 14.2 Luồng (browser-redirect, giống `gh auth login`)
1. Trang settings local (`web/`, `localhost:8788`) có khối mới **"Đồng bộ tài khoản"**.
2. Chưa liên kết: nhập địa chỉ server, bấm "Liên kết tài khoản" → JS điều hướng tới `{syncServer}/?callback=http://localhost:8788/` (trang đăng nhập của portal nằm ở `/`, không phải `/login`).
3. Portal: trang login hiện có nhận thêm query `callback`. Login/đăng ký thành công → **redirect trình duyệt** về `callback` kèm `?token=...` (thay vì chỉ lưu `localStorage` như hiện tại).
4. Trang settings local nhận lại `?token=...` trên URL của chính nó → JS đọc token, lưu vào `localStorage` của trang này (chỉ tồn tại trong WebView, không phải trên đĩa qua Go).
5. Đã liên kết: JS gọi `GET {syncServer}/me/gifs` (có CORS `*`) lấy **danh sách** GIF trong thư viện của tài khoản đó, hiện từng cái kèm nút "Dùng ảnh này" — **không có khái niệm ảnh active nào để tự lấy**, user phải tự chọn đúng 1 ảnh.
6. Bấm "Dùng ảnh này" trên 1 item → JS gọi `GET {syncServer}/me/gifs/{id}/image` lấy blob, rồi `POST` blob đó vào `http://localhost:8788/gif` (endpoint **đã có sẵn** trong `app.go`, multipart field `gif`) — giống hệt như khi user tự chọn file tải lên. App lưu `customGif`, bắn `gif-changed` như luồng cũ, không cần biết gì về syncserver.

### 14.3 Việc cần sửa
- `cmd/syncserver/portal`: trang login đọc query `callback`, sau khi `POST /auth/login` (hoặc `/auth/register`) thành công thì `location.href = callback + '?token=' + token` thay vì chỉ `setToken()` cục bộ.
- `web/` (local client, Vite project riêng của app desktop): thêm khối "Đồng bộ tài khoản" (`src/sync.js`):
  - Nếu URL hiện tại có `?token=`, lưu vào `localStorage`, xoá token khỏi URL (`history.replaceState`), gọi `GET /me` lấy username để hiển thị, báo đã liên kết tài khoản.
  - Đã liên kết: tải danh sách `GET /me/gifs`, render từng ảnh (thumbnail qua `GET /me/gifs/{id}/image`) kèm nút chọn riêng — không có nút "lấy ảnh" chung nào tự suy ra ảnh nào.
  - Cần biết địa chỉ `{syncServer}` — thêm 1 input cấu hình lưu trong `localStorage` của trang này (mặc định `http://localhost:9090`, không hardcode cứng vì server có thể deploy chỗ khác).
- **Không sửa** `app.go` — toàn bộ việc này nằm ở 2 project Vite (`web/`, `cmd/syncserver/portal`).
- Ngoài scope đợt này: đẩy ảnh local lên server (2 chiều), tự động polling nền — chỉ làm chọn thủ công 1 chiều (server → client) trước.
