# Dev notes — sync/share feature

Ghi chú vận hành, tránh quên lại các lệnh/luồng đã làm. Chi tiết thiết kế đầy đủ xem `docs/BA-sync-share.md`.

## 1. Kiến trúc tổng quan

Có 3 phần tách biệt:

| Phần | Vị trí | Vai trò |
|---|---|---|
| **App desktop** (`ancol-ani.exe`) | repo root (`app.go`, `main.go`) | Widget hiển thị GIF/chữ trên màn hình. Build bằng **Wails**, không phải `go build` thường. |
| **Web local** (trang settings `localhost:8788`) | `web/` (Vite + **Vue 3 + Vue Router**, cùng thiết kế với portal) | Trang cấu hình của app, nhiều trang: `#/` Widget (chữ + GIF local), `#/library` (thư viện GIF trên server, "Dùng cho widget"), `#/gallery` (GIF public), `#/account` (liên kết/hủy liên kết với syncserver). 2 trang library/gallery chỉ vào được khi đã liên kết. Song ngữ VI/EN (`src/i18n.js`, reactive). Trạng thái liên kết ở `src/account.js`. GIF luôn vẽ bằng canvas (`src/gif-player.js` qua `GifCanvas.vue`), không dùng `<img>` vì animation GIF gốc của Chromium có thể bị đứng. Nhúng vào `ancol-ani.exe` qua `go:embed web/dist`. |
| **Thiết kế dùng chung** | `ui/` (không phải project Vite, không có `package.json`) | Mọi thứ về giao diện mà portal và trang settings dùng chung: theme daisyUI "pixel" + Tailwind (`tailwind.shared.cjs`), CSS (`pixel.css`), component (`components/`: Icon, AppLogo, PageHeader, Pager), hộp thoại xác nhận (`modal.js`), bộ icon lucide (`icons.js`), logo (`assets/logo.png`). Hai project import qua alias **`@ui/...`**, cấu hình bởi `ui/vite.shared.js`. **Sửa giao diện chung thì sửa ở `ui/` rồi build lại cả hai project.** |
| **Syncserver** (`syncserver.exe`) | `cmd/syncserver/` | Server độc lập: tài khoản, thư viện GIF, chia sẻ công khai. Có DB riêng (SQLite file `data/ancol-ani.db`). |
| **Portal** (web UI của syncserver) | `cmd/syncserver/portal/` (Vite + **Vue 3 + Vue Router**) | SPA nhiều trang: `#/login`, `#/library`, `#/history`, `#/gallery`, `#/admin/users`, `#/admin/public` (2 trang admin có guard theo `/me.isAdmin`). Dùng hash router nên server Go không cần SPA fallback. Mỗi trang là 1 file trong `src/views/`, component dùng chung trong `src/components/`, trạng thái đăng nhập ở `src/session.js`. Nhúng vào `syncserver.exe` qua `go:embed portal/dist`. |

**Icon / logo / mascot:** từng file dùng ở đâu, đã tạo thế nào và cách đổi → xem [ICONS.md](ICONS.md). Sinh lại toàn bộ icon từ 1 ảnh gốc bằng `scripts/gen-icons.ps1`.

**3 project Vite, mỗi cái build và nhúng riêng:** `frontend/` (UI chính của widget), `web/` (settings local), `cmd/syncserver/portal/` (portal server). `web/` và `portal/` **dùng chung code giao diện trong `ui/`**:
- `ui/` không có `node_modules` riêng. `ui/vite.shared.js` trỏ các import `vue` và `lucide-static` trong file của `ui/` về `node_modules` của project đang build, nên không cần cài gì cho `ui/`. Vì vậy **cả hai project phải có cùng các dependency đó** (`vue`, `lucide-static`, `tailwindcss` 3, `daisyui` 4).
- Thêm icon mới: thêm vào `ui/icons.js` (import từ `lucide-static/icons/<tên>.svg?raw`), cả hai bên dùng được ngay.
- Dev có hot reload: portal `npm run dev` proxy API sang syncserver `:9090`; web `npm run dev` proxy `/gif /text /version` sang app đang chạy ở `:8788`.

## 2. Build — LUÔN kill process đang chạy trước khi build lại

Build đè lên file `.exe` đang chạy có thể fail hoặc để lại bản cũ không nhận ra. Thứ tự chuẩn mỗi lần sửa code:

### Sửa `cmd/syncserver/**` (Go) hoặc `cmd/syncserver/portal/**` (web)
```bash
# nếu sửa portal (web):
cd cmd/syncserver/portal && npm run build
# (dev có hot reload: chạy server Go ở :9090 rồi `npm run dev` - Vite tự proxy /auth /me /public /admin sang :9090)

# luôn kill server cũ trước khi build lại binary:
taskkill //F //PID <pid của syncserver.exe>   # lấy PID qua: tasklist | grep -i syncserver.exe
go build -o build/bin/syncserver.exe ./cmd/syncserver

# chạy lại (giữ DB cũ, đừng xóa data/ trừ khi muốn reset thật):
ADDR=0.0.0.0:9090 DB_DSN="file:data/ancol-ani.db" ./build/bin/syncserver.exe &
```

### Sửa `web/**` (settings local) hoặc `app.go`
```bash
cd web && npm run build

# luôn kill app cũ trước khi build lại:
taskkill //F //PID <pid của ancol-ani.exe>   # lấy PID qua: tasklist | grep -i ancol-ani.exe

# PHẢI dùng wails build, không phải go build thường — wails build mới tự
# chạy npm run build cho frontend/, generate binding, gắn icon/manifest.
# wails.exe không nằm trong PATH mặc định của shell này, đường dẫn thật:
"/e/go-cache/gopath/bin/wails.exe" build
```
Nếu không chắc `wails.exe` ở đâu, tìm qua: `GOPATH` (PowerShell `go env GOPATH`) rồi `<gopath>\bin\wails.exe`.

**Lỗi `'vite' is not recognized` khi `wails build`:** máy này đặt `NODE_ENV=production` (npm config `omit = ["dev"]`), nên `npm install` mà Wails tự chạy bỏ qua devDependencies, trong đó có `vite`. Đặt lại `NODE_ENV` cho riêng một lệnh vẫn không đủ. Cách chắc ăn: tự build frontend trước rồi bảo Wails bỏ qua bước đó:
```bash
cd frontend && npm ci --include=dev && npm run build && cd ..
wails build -s        # -s = skip frontend build
```
(Các project Vite khác (`web/`, `cmd/syncserver/portal/`) cũng vậy: cài bằng `npm ci --include=dev`.)

`wails build` còn tự sửa `go.mod`, `frontend/wailsjs/runtime/*` (chỉ đổi kiểu xuống dòng LF/CRLF) và `frontend/package.json.md5` (hash cache). Không có thay đổi thật → `git checkout -- go.mod frontend/wailsjs frontend/package.json.md5`.

**Không bao giờ** tự ý tắt/mở app GUI (`ancol-ani.exe`) hộ user trừ khi họ đồng ý rõ ràng — build xong để họ tự mở; khởi động GUI app từ shell/agent không đáng tin (có thể không hiện cửa sổ, không có session desktop thật).

## 3. Mô hình dữ liệu (syncserver)

- `users`: username/password (bcrypt), `is_admin`, `gif_quota` (mặc định 3, admin chỉnh được per-user qua `PATCH /admin/users/{id}/quota`).
- `gifs`: **nhiều dòng/user** (không giới hạn 1 GIF/user nữa, không có khái niệm "active"). Mỗi dòng có `is_public`, `slug`, `name` (bắt buộc khi share), `tags` (comma-separated, optional).
- `share_history`: log mỗi lần share/unshare theo `gif_id`.
- `sessions`: JWT (1h) + revoke theo `jti` — logout = set `revoked_at`, không phải JWT thuần.

**Lưu ý migration**: `CREATE TABLE IF NOT EXISTS` không tự thêm cột mới vào bảng đã tồn tại trên DB cũ. Cột `name`/`tags` thêm sau phải có bước `addColumnIfMissing` riêng trong `store.go::migrate()` (dùng `PRAGMA table_info` + `s.db.Unsafe()` vì PRAGMA trả nhiều cột hơn struct Go map).

## 4. API chính (syncserver)

| Method | Path | Auth | Ghi chú |
|---|---|---|---|
| POST | `/auth/register`, `/auth/login` | không | trả `{token}` |
| POST | `/auth/logout` | có | revoke session |
| POST | `/auth/token` | có | cấp thêm 1 token **mới, phiên riêng** cho cùng tài khoản (dùng để portal và client không bao giờ dùng chung 1 phiên, xem mục 5) |
| POST | `/me/password` | có | đổi mật khẩu `{currentPassword, newPassword}`; sai mật khẩu cũ → 403; thu hồi mọi phiên khác |
| POST | `/admin/users/{id}/reset-password` | admin | trả `{tempPassword}` một lần, thu hồi mọi phiên của user đó |
| GET | `/me` | có | `{username, isAdmin, gifQuota}` |
| GET/POST | `/me/gifs` | có | list / upload mới (409 nếu đầy quota) |
| GET | `/me/gifs/{id}/image` | có | blob ảnh, scoped theo owner |
| DELETE | `/me/gifs/{id}` | có | |
| POST | `/me/gifs/{id}/share` | có | body `{name (bắt buộc), tags}` |
| POST | `/me/gifs/{id}/unshare` | có | |
| GET | `/me/gif/history` | có | lịch sử share, có `gif_id` |
| GET | `/public?q=&user=` | có (cần đăng nhập, không anonymous) | tìm theo tên/tag (`q`) và/hoặc đúng username (`user`) |
| GET | `/public/{slug}` | có | blob ảnh public |
| GET/DELETE/PATCH | `/admin/users...` | admin | list, xoá user, sửa quota |
| DELETE | `/admin/public/{slug}` | admin | gỡ public bất kỳ |

Admin đầu tiên set bằng CLI: `syncserver.exe -make-admin <username>` (không có API tự nâng quyền).

## 5. Luồng liên kết client ↔ server (web local ↔ syncserver)

Không dùng OAuth/OIDC thật, dùng pattern "browser-redirect" giống `gh auth login`:

1. `web/` (local, `localhost:8788`) bấm "Liên kết tài khoản" → lưu địa chỉ server vào `localStorage`, redirect tới `{server}/?callback=http://localhost:8788/`.
2. Portal: nếu **tab portal đã có session sẵn** (đăng nhập từ trước) và có `?callback=`, **redirect ngay** không chờ login mới (tránh bug "kẹt ở portal"). Nếu session đó đã hết hạn thì hiện trang đăng nhập như bước 3.
3. Nếu chưa login: login/register xong thì portal **giữ phiên của chính nó** rồi redirect về `callback?token=...`.
   - Ở cả bước 2 và 3, token gửi cho client là token **mới** xin từ `POST /auth/token` (`sendTokenToClient()` trong `portal/src/session.js`), **không phải** token của portal. Hai bên luôn có phiên riêng: đăng xuất portal không làm client mất liên kết, và ngược lại. Đừng quay lại kiểu đưa thẳng token của portal cho client.
4. `web/` nhận lại `?token=` trên URL của chính nó (`consumeCallbackToken()` trong `web/src/account.js`), lưu vào `localStorage` riêng của **trang này** (khác hẳn `localStorage` của portal — 2 origin khác nhau). Go app (`app.go`) **không bao giờ biết** gì về token này.
   - **Router của `web/` phải được tạo SAU bước này** (`createAppRouter()` gọi trong `main.js` sau khi xử lý token). `createWebHashHistory()` đọc URL lúc khởi tạo, tạo sớm thì kẹt ở trang cũ và `?token=` dính vào mọi URL.
5. Token hết hạn/bị revoke → mọi lời gọi `authedFetch()`/`serverApi()` trong `web/src/account.js` tự phát hiện `401`, xoá token, báo toast và đưa về trang Tài khoản (không cần reload tay).
6. **"Mở portal" từ client (đăng nhập sẵn):** nút ở trang Tài khoản của client mở `{server}/#/sso?t=<token của client>&to=/account`. Trang `portal/src/views/SsoView.vue` đổi token đó lấy một phiên portal riêng qua `POST /auth/token`, rồi `router.replace(to)` để token không còn trong URL/lịch sử. Token sai/hết hạn → về trang đăng nhập. Token đặt sau `#` nên trình duyệt không gửi lên server/log.

Giới hạn còn lại: mọi token sống **1 giờ** (`tokenTTL` trong `auth.go`), chưa có refresh token. Hết giờ thì client phải liên kết lại và portal phải đăng nhập lại (xem mục 8).

## 6. GIF hiển thị — luôn qua canvas, không dùng `<img>` (trang settings)

Chromium's native `<img>` GIF animation có bug treo vĩnh viễn sau một lúc loop (tái hiện được cả trên Chrome thường). Toàn bộ nơi hiển thị GIF trong `web/` (preview widget, thư viện, khám phá) đều phải dùng component `web/src/components/GifCanvas.vue` (bọc `web/src/gif-player.js::playGif`) — **không** gắn `blob` vào `<img src>`. (Portal của server hiện vẫn dùng `<img>` qua `GifImage.vue`.)

## 7. UI dialog — dùng modal pixel-style, không dùng `confirm()`/`prompt()` của browser

`ui/modal.js` (dùng chung) cung cấp `confirmDialog(message, { confirmText, cancelText })`, style khớp theme pixel. Dùng cho mọi xác nhận xoá / áp dụng / hủy liên kết ở cả portal lẫn trang settings — không gọi `window.confirm`/`window.prompt` trần. Nhập tên + tag khi chia sẻ thì làm inline ngay trên thẻ GIF (giống portal), không dùng hộp thoại.

## 8. Việc còn thiếu / ngoài scope hiện tại

- Chưa test Postgres (`DB_DRIVER=postgres`) thật — chỉ có khung schema, SQLite là driver duy nhất đã verify.
- Chưa có refresh token — token JWT 1h hết là phải liên kết lại (chấp nhận đánh đổi theo yêu cầu ban đầu).
- Chưa đẩy 2 chiều tự động (client không tự poll nền để kéo GIF mới) — mọi hành động đều là bấm tay.
- `JWT_SECRET` chưa set qua env khi deploy thật — đang dùng fallback dev không an toàn, phải set trước khi deploy ngoài localhost.

## 9. Phát hành version mới của app desktop

Version của app nằm ở **3 chỗ, phải sửa cùng lúc**:

| File | Dùng làm gì |
|---|---|
| `VERSION` | Nguồn chính. `app.go` nhúng (`//go:embed VERSION`) và trả qua `GET /version`; trang settings hiện ở chân trang và trang **Có gì mới**. |
| `wails.json` → `info.productVersion` | Version ghi vào file `ancol-ani.exe` (Properties → Details) và bộ cài NSIS. |
| `CHANGELOG.md` | Mục mới `## <version> — <YYYY-MM-DD>` ở **trên cùng**. |

`CHANGELOG.md` cũng được `app.go` nhúng (`//go:embed CHANGELOG.md`) và trả qua `GET /changelog`. Trang **Có gì mới** (`web/src/views/ChangelogView.vue`, parser trong `web/src/appInfo.js`) đọc file đó, nên **giữ đúng định dạng**:

```markdown
## 1.2.0 — 2026-12-01

### Mới
- Mô tả theo góc nhìn người dùng, có thể dùng `code` và **đậm**.

### Thay đổi
- ...

### Sửa lỗi
- ...
```

- Parser chỉ hiểu `## version — ngày`, `### nhóm`, và dòng `- ...`. Đoạn văn khác bị bỏ qua (nên phần giới thiệu đầu file không hiện).
- Nhóm tô màu theo tên: "Mới/New" xanh, "Thay đổi/Changed" vàng, "Sửa lỗi/Fix" đỏ.
- Changelog chỉ ghi thay đổi **của app desktop** (widget + trang settings), viết cho người dùng chứ không phải dev. Thay đổi riêng của server/portal không ghi ở đây.
- Khi app chạy version khác version người dùng xem lần cuối (`localStorage['ancol-ani-seen-version']`), trang settings hiện chấm hồng cạnh mục **Có gì mới** và số version; mở trang đó là chấm tắt. Lần đầu mở trên một trình duyệt thì không báo.

Thứ tự: sửa 3 file trên → `cd web && npm run build` → build frontend + `wails build -s` (mục 2) → commit `chore(release): <version>`.
