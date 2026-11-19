# Ancol-Ani

Widget GIF/chữ góc màn hình cho Windows (Go + Wails), kèm trang settings local và một sync server có web portal để đồng bộ/chia sẻ GIF qua tài khoản.

**Đọc trước khi sửa:** [docs/DEV-NOTES.md](docs/DEV-NOTES.md) (kiến trúc, build, API, luồng liên kết, phát hành version) và [docs/ICONS.md](docs/ICONS.md) (icon/logo/mascot). Khi thay đổi làm một trong hai file đó sai đi thì cập nhật luôn trong cùng thay đổi.

## Cấu trúc

| Thư mục | Là gì | Nhúng vào |
|---|---|---|
| repo root (`app.go`, `main.go`) | App desktop Wails: widget + HTTP server local, port tự dò từ `preferredPort` (48821) nếu bị chiếm (`/gif`, `/text`, `/version`, `/changelog`, `/position`, `/autostart`) | `build/bin/ancol-ani.exe` |
| `frontend/` | UI widget (chỉ `index.html` + `src/gifPlayer.js`) | exe (`frontend/dist`) |
| `web/` | Trang settings `localhost:<port>` — Vue 3 + Vue Router (hash) | exe (`web/dist`) |
| `cmd/syncserver/` | Sync server Go (`:9090`, SQLite `data/`) | `syncserver.exe` |
| `cmd/syncserver/portal/` | Web portal của server — Vue 3 + Vue Router (hash) | `syncserver.exe` (`portal/dist`) |
| `ui/` | **Thiết kế dùng chung** của `web/` và `portal/`: theme, `pixel.css`, component, `modal.js`, `icons.js`, logo. Import qua `@ui/...`. Không có `package.json`. | — |

## Quy tắc

- Tên app hiển thị cho người dùng viết là **"Ancol-Ani"** (cả hai chữ A đầu đều viết hoa), không viết `ANCOL-ANI` hay `Ancol-ani`. **Không đổi** các chuỗi kỹ thuật viết thường `ancol-ani`: tiêu đề cửa sổ widget (`Title` trong `main.go`, `<title>` của `frontend/index.html`, vì `app.go`/`findWindowHandle` tìm cửa sổ widget theo đúng tiêu đề `"ancol-ani"` này), khóa localStorage (`ancol-ani-*`, đổi là người dùng mất đăng nhập/liên kết), tên file exe, tên gói `wails.json` (`name`/`outputfilename`) và tên sản phẩm của bộ cài.

- **Giao diện chung sửa ở `ui/`**, rồi build lại cả `web/` và `portal/`. Không chép component/CSS sang từng project. Icon mới: thêm vào `ui/icons.js` (lucide-static).
- Hai trang web phải giữ cùng thiết kế (theme daisyUI "pixel", thanh điều hướng, card, toast/alert). Trang settings song ngữ: mọi chữ hiển thị đi qua `t()` trong `web/src/i18n.js`, thêm key cho **cả** `vi` và `en`.
- GIF trong `web/` luôn vẽ bằng canvas (`GifCanvas.vue`), không dùng `<img>` (Chromium làm đứng animation).
- Xác nhận dùng `confirmDialog` trong `ui/modal.js`, không dùng `window.confirm/prompt`.
- Router của `web/` được tạo sau khi xử lý `?token=` (xem DEV-NOTES mục 5); đừng chuyển về router tạo sẵn ở module level.
- Lỗi server trả về 401 nghĩa là "phiên đã mất" (client tự đăng xuất). Lỗi nghiệp vụ như sai mật khẩu hiện tại thì dùng 403/400.
- Portal và client **không bao giờ dùng chung một token**: khi liên kết hay "Mở portal" thì luôn xin token mới qua `POST /auth/token` (DEV-NOTES mục 5). Đây không phải OIDC, mà là luồng chuyển hướng tự viết; token sống 1 giờ, chưa có refresh token.
- Vị trí widget tính bằng **pixel thật** và đặt qua Win32 `SetWindowPos(HWND_TOPMOST)` (`app.go`); đừng quay lại `runtime.WindowSetPosition` với kích thước logic (lệch khi scale ≠ 100%).

- Server local **không có port cố định**: `resolveListener` (`app.go`) dò từ `preferredPort` (48821) tới khi tìm được port trống, ghi vào `%AppData%\ancol-ani\port.txt`. `LocalPort()` (bind cho widget) và cửa sổ settings riêng (`runSettingsApp` trong `main.go`, process khác nên không đọc được biến Go của widget) đều lấy port qua đó, không hardcode. Đừng quay lại hardcode `8788` ở đâu cả — chỉ có `preferredPort` là nơi cố định một con số.

## Build (Windows, chi tiết ở DEV-NOTES mục 2)

- Máy dev đặt `NODE_ENV=production` → `npm install` bỏ devDependencies. Luôn cài bằng `npm ci --include=dev` (hoặc `npm install --include=dev`).
- App desktop: `cd web && npm run build`, `cd frontend && npm run build`, rồi `wails build -s` (`-s` vì Wails tự cài npm sẽ thiếu `vite`). Wails CLI ở `C:\Users\an\go\bin\wails.exe`.
- Sau `wails build`, khôi phục file nó tự sửa: `git checkout -- go.mod frontend/package.json.md5 frontend/wailsjs`.
- Server: `cd cmd/syncserver/portal && npm run build`, rồi `go run ./cmd/syncserver` (cần thư mục `data/`). Server nhúng `portal/dist` lúc build, sửa portal phải khởi động lại server.
- **Không tự tắt/mở `ancol-ani.exe`** khi chưa hỏi người dùng (exe đang chạy thì build đè sẽ lỗi). Build xong để họ tự mở.

## Production

Sync server chạy công khai ở **https://game.anbebong.io.vn**, trên điện thoại Termux tự host (thay chỗ dino-race trên cổng 7879). Deploy bằng `bash deploy/phone/deploy.sh`, chi tiết ở DEV-NOTES mục 10. Domain gốc `anbebong.io.vn` là site khác, không phải server này. Deploy là thao tác công khai: hỏi người dùng trước khi chạy. Không đụng `~/ancolani/data/` và `~/ancolani/.jwt-secret` trên điện thoại.

## Version & changelog

Phát hành app desktop: sửa cùng lúc `VERSION`, `wails.json` (`info.productVersion`) và thêm mục đầu `CHANGELOG.md` dạng `## x.y.z — YYYY-MM-DD` với `### Mới / Thay đổi / Sửa lỗi`. App nhúng cả hai file; trang **Về Ancol-Ani** trong settings parse `CHANGELOG.md`, nên giữ đúng định dạng (DEV-NOTES mục 9). Changelog chỉ ghi thay đổi người dùng app thấy được.

## Cập nhật app (tự cập nhật)

Trang **Về Ancol-Ani** có nút "Kiểm tra cập nhật" — chỉ chạy khi người dùng tự bấm, không tự động/định kỳ. App gọi `/check-update` local (`app.go`), cái đó gọi `GET {updateServerBase}/latest-version` trên sync server công khai; có bản mới thì hỏi xác nhận rồi `POST /apply-update` tải exe mới, kiểm `sha256`, tự thay file + khởi động lại (qua file `.bat` tạm chờ process cũ thoát).

**Publish 1 bản mới là thao tác tay** (không có upload API, không CI): sau khi build xong `ancol-ani.exe`, copy đè vào `data/releases/ancol-ani.exe` trên server (`cmd/syncserver/update.go` serve tĩnh từ đó), rồi ghi/`sửa data/releases/latest.json` dạng `{"version":"x.y.z","sha256":"<sha256 của file exe đó>","notes":"..."}`. Thiếu `latest.json` thì nút "Kiểm tra cập nhật" chỉ báo lỗi, không crash app.

## Icon

Sinh lại mọi icon từ một ảnh gốc: `scripts/gen-icons.ps1 -Source <png>`, rồi tăng `?v=` ở thẻ icon trong `web/index.html` và `cmd/syncserver/portal/index.html`. Xem docs/ICONS.md.

## Git

- Commit theo Conventional Commits, type/scope tiếng Anh, mô tả tiếng Việt (vd `feat(web): ...`, `fix(desktop): ...`). Scope thường dùng: `desktop`, `web`, `portal`, `syncserver`, `ui`, `release`.
- Làm trên nhánh tính năng, không commit thẳng vào `main`. Chỉ push khi được yêu cầu.
- Không commit `data/`, các thư mục `dist/`, `node_modules/`.

## Dữ liệu test local

DB dev có tài khoản `demo` / `demo123` (admin, dùng để test) và `anpt` (tài khoản thật của người dùng; không đổi mật khẩu/quota/xóa khi test).
