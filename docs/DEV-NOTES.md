# Dev notes — sync/share feature

Ghi chú vận hành, tránh quên lại các lệnh/luồng đã làm. Chi tiết thiết kế đầy đủ xem `docs/BA-sync-share.md`.

## 1. Kiến trúc tổng quan

Có 3 phần tách biệt:

| Phần | Vị trí | Vai trò |
|---|---|---|
| **App desktop** (`ancol-ani.exe`) | repo root (`app.go`, `main.go`) | Widget hiển thị GIF/chữ trên màn hình. Build bằng **Wails**, không phải `go build` thường. |
| **Web local** (trang settings `localhost:8788`) | `settings-ui/` (Vite + **Vue 3 + Vue Router**, cùng thiết kế với portal) | Trang cấu hình của app, nhiều trang: `#/` Widget (chữ + GIF local), `#/library` (thư viện GIF trên server, "Dùng cho widget"), `#/gallery` (GIF public), `#/account` (liên kết/hủy liên kết với syncserver). 2 trang library/gallery chỉ vào được khi đã liên kết. Song ngữ VI/EN (`src/i18n.js`, reactive). Trạng thái liên kết ở `src/account.js`. GIF luôn vẽ bằng canvas (`src/gif-player.js` qua `GifCanvas.vue`), không dùng `<img>` vì animation GIF gốc của Chromium có thể bị đứng. Nhúng vào `ancol-ani.exe` qua `go:embed settings-ui/dist`. |
| **Thiết kế dùng chung** | `theme/` (không phải project Vite, không có `package.json`) | Mọi thứ về giao diện mà portal và trang settings dùng chung: theme daisyUI "pixel" + Tailwind (`tailwind.shared.cjs`), CSS (`pixel.css`), component (`components/`: Icon, AppLogo, PageHeader, Pager), hộp thoại xác nhận (`modal.js`), bộ icon lucide (`icons.js`), logo (`assets/logo.png`). Hai project import qua alias **`@theme/...`**, cấu hình bởi `theme/vite.shared.js`. **Sửa giao diện chung thì sửa ở `theme/` rồi build lại cả hai project.** |
| **Syncserver** (`syncserver.exe`) | `cmd/syncserver/` | Server độc lập: tài khoản, thư viện GIF, chia sẻ công khai. Có DB riêng (SQLite file `data/ancol-ani.db`). |
| **Portal** (web UI của syncserver) | `cmd/syncserver/portal/` (Vite + **Vue 3 + Vue Router**) | SPA nhiều trang: `#/login`, `#/library`, `#/history`, `#/gallery`, `#/admin/users`, `#/admin/public` (2 trang admin có guard theo `/me.isAdmin`). Dùng hash router nên server Go không cần SPA fallback. Mỗi trang là 1 file trong `src/views/`, component dùng chung trong `src/components/`, trạng thái đăng nhập ở `src/session.js`. Nhúng vào `syncserver.exe` qua `go:embed portal/dist`. |

**Icon / logo / mascot:** từng file dùng ở đâu, đã tạo thế nào và cách đổi → xem [ICONS.md](ICONS.md). Sinh lại toàn bộ icon từ 1 ảnh gốc bằng `scripts/gen-icons.ps1`.

**3 project Vite, mỗi cái build và nhúng riêng:** `widget/` (UI chính của widget), `settings-ui/` (settings local), `cmd/syncserver/portal/` (portal server). `settings-ui/` và `portal/` **dùng chung code giao diện trong `theme/`**:
- `theme/` không có `node_modules` riêng. `theme/vite.shared.js` trỏ các import `vue` và `lucide-static` trong file của `theme/` về `node_modules` của project đang build, nên không cần cài gì cho `theme/`. Vì vậy **cả hai project phải có cùng các dependency đó** (`vue`, `lucide-static`, `tailwindcss` 3, `daisyui` 4).
- Thêm icon mới: thêm vào `theme/icons.js` (import từ `lucide-static/icons/<tên>.svg?raw`), cả hai bên dùng được ngay.
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

### Sửa `settings-ui/**` (settings local) hoặc `app.go`
```bash
cd web && npm run build

# luôn kill app cũ trước khi build lại:
taskkill //F //PID <pid của ancol-ani.exe>   # lấy PID qua: tasklist | grep -i ancol-ani.exe

# PHẢI dùng wails build, không phải go build thường — wails build mới tự
# chạy npm run build cho widget/, generate binding, gắn icon/manifest.
# wails.exe không nằm trong PATH mặc định của shell này, đường dẫn thật:
"/e/go-cache/gopath/bin/wails.exe" build
```
Nếu không chắc `wails.exe` ở đâu, tìm qua: `GOPATH` (PowerShell `go env GOPATH`) rồi `<gopath>\bin\wails.exe`.

**Lỗi `'vite' is not recognized` khi `wails build`:** máy này đặt `NODE_ENV=production` (npm config `omit = ["dev"]`), nên `npm install` mà Wails tự chạy bỏ qua devDependencies, trong đó có `vite`. Đặt lại `NODE_ENV` cho riêng một lệnh vẫn không đủ. Cách chắc ăn: tự build widget +trước rồi bảo Wails bỏ qua bước đó:
```bash
cd widget && npm ci --include=dev && npm run build && cd ..
wails build -s        # -s = skip frontend build
```
(Các project Vite khác (`settings-ui/`, `cmd/syncserver/portal/`) cũng vậy: cài bằng `npm ci --include=dev`.)

`wails build` còn tự sửa `go.mod`, `widget/wailsjs/runtime/*` (chỉ đổi kiểu xuống dòng LF/CRLF) và `widget/package.json.md5` (hash cache). Không có thay đổi thật → `git checkout -- go.mod widget/wailsjs widget/package.json.md5`.

**Tắt app để build:** người dùng đã đồng ý (2026-11-21) cho tự tắt `ancol-ani.exe` (cả widget lẫn cửa sổ settings) khi cần build lại, không cần hỏi. **Không tự mở lại** app GUI sau khi build — để họ tự mở; khởi động GUI app từ shell/agent không đáng tin (có thể không hiện cửa sổ, không có session desktop thật).

## 3. Mô hình dữ liệu (syncserver)

- `users`: username/password (bcrypt), `is_admin`, `gif_quota` (mặc định 3, admin chỉnh được per-user qua `PATCH /admin/users/{id}/quota`).
- `gifs`: **nhiều dòng/user** (không giới hạn 1 GIF/user nữa, không có khái niệm "active"). Mỗi dòng có `is_public`, `slug`, `name` (bắt buộc khi share), `tags` (comma-separated, optional).
- `share_history`: log mỗi lần share/unshare theo `gif_id`.
- `sessions`: JWT (1h) + revoke theo `jti` — logout = set `revoked_at`, không phải JWT thuần.

**Migration theo version** — file SQL ở `cmd/syncserver/migrations/<driver>/NNN_tên.sql` (`sqlite/`, `postgres/`), nhúng vào binary bằng `go:embed`; code chạy ở `migrations.go`.

- DB ghi các version đã chạy vào bảng `schema_migrations`. Khi khởi động, server chạy các file còn thiếu theo số thứ tự, mỗi file trong 1 transaction cùng dòng ghi version: lỗi là rollback sạch, DB giữ version cũ và server không chạy (lỗi ghi rõ tên file).
- Số file phải liên tục 001, 002, ... (thiếu/trùng số hoặc sai tên là server báo lỗi lúc khởi động). `sqlite/` và `postgres/` phải có cùng danh sách số + tên (có test).
- DB có version cao hơn code (chạy binary cũ trên DB đã migrate bởi bản mới) thì server từ chối khởi động.
- **Đổi schema = thêm file mới** `002_<tên>.sql` cho cả hai dialect. **Không sửa/đổi tên/đánh lại số file đã lên production** (DB đã ghi là chạy rồi nên sửa sẽ không bao giờ tới DB cũ).
- `001_baseline.sql` là schema trước khi có hệ thống này, viết idempotent (`IF NOT EXISTS`) để nhận DB production cũ mà không đụng dữ liệu. Thứ SQL thuần không làm được thì đặt trong `afterSQL` (`migrations.go`), chạy cùng transaction ngay sau file đó; hiện chỉ có `addColumnIfMissing` cho `gifs.name`/`tags` trên SQLite (SQLite không có `ADD COLUMN IF NOT EXISTS`).
- Test ở `migrate_test.go`.

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
| GET | `/public?q=&user=&sort=` | có (cần đăng nhập, không anonymous) | tìm theo tên/tag (`q`) và/hoặc đúng username (`user`); `sort=stars` = nhiều star nhất trước (mặc định mới nhất). Mỗi mục có `stars` và `starred` (người gọi đã star chưa) |
| GET | `/public/{slug}` | có | blob ảnh public |
| POST/DELETE | `/public/{slug}/star` | có | star / bỏ star kiểu GitHub (1 người 1 star / GIF, gọi lại không sao), trả `{stars, starred}`. Lưu ở bảng `gif_stars (user_id, gif_id)`; gắn với GIF chứ không với slug nên gỡ chia sẻ rồi chia sẻ lại vẫn giữ star; xóa GIF/user thì star liên quan bị xóa. `GET /me/gifs` có thêm `stars` (số star đã nhận) |
| GET/DELETE/PATCH | `/admin/users...` | admin | list, xoá user, sửa quota |
| DELETE | `/admin/public/{slug}` | admin | gỡ public bất kỳ |

Admin đầu tiên set bằng CLI: `syncserver.exe -make-admin <username>` (không có API tự nâng quyền). Sửa quota không cần admin: `syncserver.exe -set-quota <username>=<N>` (0–50). Cả hai sửa thẳng DB rồi thoát, chạy được khi server đang chạy.

## 5. Luồng liên kết client ↔ server (web local ↔ syncserver)

Không dùng OAuth/OIDC thật, dùng pattern "browser-redirect" giống `gh auth login`:

1. `settings-ui/` (local, `localhost:8788`) bấm "Liên kết tài khoản" → lưu địa chỉ server vào `localStorage`, redirect tới `{server}/?callback=http://localhost:8788/`.
2. Portal: nếu **tab portal đã có session sẵn** (đăng nhập từ trước) và có `?callback=`, **redirect ngay** không chờ login mới (tránh bug "kẹt ở portal"). Nếu session đó đã hết hạn thì hiện trang đăng nhập như bước 3.
3. Nếu chưa login: login/register xong thì portal **giữ phiên của chính nó** rồi redirect về `callback?token=...`.
   - Ở cả bước 2 và 3, token gửi cho client là token **mới** xin từ `POST /auth/token` (`sendTokenToClient()` trong `portal/src/session.js`), **không phải** token của portal. Hai bên luôn có phiên riêng: đăng xuất portal không làm client mất liên kết, và ngược lại. Đừng quay lại kiểu đưa thẳng token của portal cho client.
4. `settings-ui/` nhận lại `?token=` trên URL của chính nó (`consumeCallbackToken()` trong `settings-ui/src/account.js`), lưu vào `localStorage` riêng của **trang này** (khác hẳn `localStorage` của portal — 2 origin khác nhau). Go app (`app.go`) **không bao giờ biết** gì về token này.
   - **Router của `settings-ui/` phải được tạo SAU bước này** (`createAppRouter()` gọi trong `main.js` sau khi xử lý token). `createWebHashHistory()` đọc URL lúc khởi tạo, tạo sớm thì kẹt ở trang cũ và `?token=` dính vào mọi URL.
5. Token hết hạn/bị revoke → mọi lời gọi `authedFetch()`/`serverApi()` trong `settings-ui/src/account.js` tự phát hiện `401`, xoá token, báo toast và đưa về trang Tài khoản (không cần reload tay).
6. **"Mở portal" từ client (đăng nhập sẵn):** nút ở trang Tài khoản của client mở `{server}/#/sso?t=<token của client>&to=/account`. Trang `portal/src/views/SsoView.vue` đổi token đó lấy một phiên portal riêng qua `POST /auth/token`, rồi `router.replace(to)` để token không còn trong URL/lịch sử. Token sai/hết hạn → về trang đăng nhập. Token đặt sau `#` nên trình duyệt không gửi lên server/log.

Giới hạn còn lại: mọi token sống **1 giờ** (`tokenTTL` trong `auth.go`), chưa có refresh token. Hết giờ thì client phải liên kết lại và portal phải đăng nhập lại (xem mục 8).

## 6. GIF hiển thị — luôn qua canvas, không dùng `<img>` (trang settings)

Chromium's native `<img>` GIF animation có bug treo vĩnh viễn sau một lúc loop (tái hiện được cả trên Chrome thường). Toàn bộ nơi hiển thị GIF trong `settings-ui/` (preview widget, thư viện, khám phá) đều phải dùng component `settings-ui/src/components/GifCanvas.vue` (bọc `settings-ui/src/gif-player.js::playGif`) — **không** gắn `blob` vào `<img src>`. (Portal của server hiện vẫn dùng `<img>` qua `GifImage.vue`.)

## 7. UI dialog — dùng modal pixel-style, không dùng `confirm()`/`prompt()` của browser

`theme/modal.js` (dùng chung) cung cấp `confirmDialog(message, { confirmText, cancelText })`, style khớp theme pixel. Dùng cho mọi xác nhận xoá / áp dụng / hủy liên kết ở cả portal lẫn trang settings — không gọi `window.confirm`/`window.prompt` trần. Nhập tên + tag khi chia sẻ thì làm inline ngay trên thẻ GIF (giống portal), không dùng hộp thoại.

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

`CHANGELOG.md` cũng được `app.go` nhúng (`//go:embed CHANGELOG.md`) và trả qua `GET /changelog`. Trang **Có gì mới** (`settings-ui/src/views/ChangelogView.vue`, parser trong `settings-ui/src/appInfo.js`) đọc file đó, nên **giữ đúng định dạng**:

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

Thứ tự: sửa 3 file trên → `cd web && npm run build` → build widget ++ `wails build -s` (mục 2) → commit `chore(release): <version>`.

## 10. Deploy sync server lên điện thoại (production)

Sync server chạy công khai tại **https://game.anbebong.io.vn**, trên điện thoại Android tự host (Termux, SSH `u0_a249@192.168.0.105 -p 8022`, đăng nhập bằng key). Hạ tầng điện thoại (Termux, proot, tmux, Cloudflare Tunnel, Termux:Boot) do repo `code-game/dino-race` dựng sẵn; xem `dino-race/deploy/README.md` để hiểu chi tiết. Ancol-Ani thay chỗ server dino-race trên cổng 7879.

**Deploy (1 lệnh, từ gốc repo trên Windows, Git Bash):**
```bash
bash deploy/phone/deploy.sh
```
Script build portal + syncserver `linux/arm64` (CGO tắt), bật maintenance mode, **sao lưu DB**, upload, khởi động lại rồi chờ `https://game.anbebong.io.vn` trả 200. Site gián đoạn khoảng 10–20 giây.

| Thứ | Ở đâu |
|---|---|
| Tunnel ingress (cấu hình trên Cloudflare Dashboard) | `game.anbebong.io.vn` → `localhost:7879` (**Ancol-Ani**), `mothe.anbebong.io.vn` → `localhost:8080` (Ancha). Domain gốc `anbebong.io.vn` **không** đi qua tunnel này. |
| Binary | `~/ancolani/ancolani-sync-arm64` (ghi đè mỗi lần deploy) |
| Dữ liệu | `~/ancolani/data/ancol-ani.db`: **dữ liệu thật**, deploy không đụng tới. Không chép DB dev từ máy lên (có tài khoản test `demo/demo123`). |
| Backup DB | **Trên máy dev**, không trên điện thoại: mỗi lần deploy, lúc server đã dừng, script kéo DB về `deploy/backup/ancol-ani-<YYYYmmdd-HHMMSS>.db` trong repo (git-ignored, đổi chỗ bằng biến `LOCAL_BACKUP_DIR`) và so sha256 hai đầu. Không tự xoá bản cũ. Backup lỗi thì script bật lại server cũ và dừng. Khôi phục: `~/maintenance-on.sh`, `scp` bản backup đè lên `~/ancolani/data/ancol-ani.db`, rồi `~/maintenance-off.sh`. |
| JWT secret | `~/ancolani/.jwt-secret` (chmod 600), sinh ngẫu nhiên ở lần deploy đầu, giữ nguyên về sau. Đổi file này là mọi người bị đăng xuất. |
| Vòng lặp chạy | `~/run.sh` (nguồn: `deploy/phone/run.sh`): Ancol-Ani :7879 + Ancha :8080 + cloudflared, trong proot, tmux session `dinorace`. |
| Log | `~/dinorace-tunnel.log` (gộp cả 3) |

- Cấp quyền admin trên production: đăng ký tài khoản trên portal, rồi chạy `ssh -p 8022 u0_a249@192.168.0.105 "cd ~/ancolani && ./ancolani-sync-arm64 -make-admin <username>"`.
- Sửa quota trên production khi chưa có admin: lệnh trên với `-set-quota <username>=<N>` thay cho `-make-admin`. Nhớ đặt `DB_DSN="file:data/ancol-ani.db?_pragma=busy_timeout(5000)"` giống `run.sh`. Sao lưu DB trước (`cp data/ancol-ani.db data/ancol-ani.db.bak-<ngày>`).
- **Quay lại dino-race:** chạy `bash deploy/deploy-arm.sh` trong repo dino-race. Script đó ghi đè lại `~/run.sh` bản dino-race; dữ liệu dino-race (`~/server`, `~/users.json`, `~/admin.json`) vẫn còn nguyên. Bản `run.sh` dino-race cũ cũng được lưu ở `~/run.sh.dinorace-backup`.
- Ngược lại: **đừng chạy `deploy-arm.sh` của dino-race** nếu muốn giữ Ancol-Ani, vì nó sẽ đưa dino-race trở lại cổng 7879.
- `deploy/phone/maintenance-on.sh` là bản của repo này: giống bản dino-race nhưng kill thêm `ancolani-sync-arm64`.

## 11. Giới hạn bảo mật của sync server (đang public)

| Bảo vệ | Ở đâu | Giới hạn |
|---|---|---|
| Bom GIF (file nhỏ giải nén ra hàng GB RAM, làm sập điện thoại) | `gifguard.go`: quét cấu trúc GIF **trước** `gif.DecodeAll`, không giải nén | mỗi cạnh ≤ 1024, ≤ 500 frame, tổng w×h các frame ≤ 64M pixel; tối đa 2 lần giải mã cùng lúc. Test: `go test ./cmd/syncserver -run TestCheckGif` |
| Đoán mật khẩu | `ratelimit.go` + `handleLogin` | 20 lần đăng nhập / 10 phút / IP; sai 5 lần / 15 phút / username thì khóa username đó (cả khi đúng mật khẩu) |
| Đoán mật khẩu cũ qua phiên bị lộ | `handleChangePassword` | sai mật khẩu hiện tại 5 lần / 15 phút → khóa |
| Đăng ký hàng loạt | `handleRegister` | 5 tài khoản / giờ / IP. Đóng hẳn đăng ký: env `ALLOW_REGISTER=false` (thêm vào dòng chạy server trong `deploy/phone/run.sh`) |
| Kéo dài token bị lộ qua `/auth/token` | `handleIssueToken`, `issueTokenUntil` | token mới **cùng hạn** với phiên gốc, không bao giờ lâu hơn |
| Body JSON lớn | `maxJSONBody` | 64KB cho login/register/đổi mật khẩu |

- **IP thật:** request đi qua cloudflared trên chính điện thoại (kết nối loopback), nên IP người dùng lấy từ header `CF-Connecting-IP`. Header này **chỉ được tin khi kết nối đến từ loopback** (`clientIP()` trong `ratelimit.go`); ai gọi thẳng cổng 7879 từ LAN không giả được. Đã kiểm chứng trên điện thoại: mỗi IP có bộ đếm riêng.
- Bộ đếm nằm trong RAM: khởi động lại server là reset. Đủ để làm chậm dò mật khẩu/spam trên 1 server nhỏ.
- Còn chưa làm (mức trung bình/thấp): timeout cho HTTP server, header bảo mật (CSP, `X-Frame-Options`, `nosniff`), bind `127.0.0.1` thay vì `0.0.0.0`, `jwt.WithValidMethods`.
