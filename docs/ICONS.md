# Icon, logo và ảnh thương hiệu

Ghi lại **từng file ảnh dùng làm icon/logo**: nó nằm ở đâu, được dùng vào việc gì, được tạo ra thế nào, để sau này muốn đổi thì biết phải sửa những gì.

**Đổi icon nhanh:** chuẩn bị 1 ảnh PNG vuông, nền trong suốt, **≥ 1024px**, rồi chạy:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/gen-icons.ps1 -Source duong\dan\logo.png
```

Script tự sinh lại mọi file trong bảng ở mục 1 (trừ mascot). Các bước đầy đủ ở [mục 4](#4-cách-đổi-icon).

---

## 1. Icon chính: chữ "A" kèm các nút mạng

Hình: chữ **A** màu tím than (`#1f1635`) kèm các nút mạng ở góc trên bên phải, nền trong suốt.

**Nguồn gốc:** `ancol.ico` ở thư mục gốc repo là bản gốc duy nhất hiện có. File này **chỉ chứa 1 ảnh 64×64**, nên mọi bản lớn hơn 64px đều được phóng to từ đó và bị mờ, răng cưa ở viền. Có file gốc độ phân giải cao thì chạy lại script để thay hết.

### App desktop (Wails)

| File | Kích thước | Nền | Dùng ở đâu |
|---|---|---|---|
| `ancol.ico` | 64 (hiện tại) | trong suốt | Bản gốc lưu ở thư mục gốc repo. Không có code nào đọc trực tiếp; giữ giống hệt `build/windows/icon.ico`. |
| `build/windows/icon.ico` | 64 (hiện tại) | trong suốt | **Icon file `ancol-ani.exe`** (Wails nhúng vào lúc `wails build`) và **icon bộ cài / gỡ cài NSIS** (`build/windows/installer/project.nsi`: `MUI_ICON`, `MUI_UNICON`). |
| `build/appicon.png` | 1024 | trong suốt | Ảnh nguồn của Wails: dùng để tạo icon `.icns` khi build macOS, và để **tự tạo `icon.ico` nếu file đó bị thiếu**. Đã có `icon.ico` thì Windows không đọc file này. Trước đây là logo "W" mặc định của Wails. |

### Hai giao diện web: portal của server và trang settings của client

Hai trang dùng **chung một bộ icon** (xem thư mục `ui/` trong `DEV-NOTES.md`).

| File | Kích thước | Nền | Dùng ở đâu |
|---|---|---|---|
| `ui/assets/logo.png` | 64 (hiện tại) | trong suốt | **Logo trên thanh điều hướng** của cả hai trang, và **cạnh tên Ancol-ani ở trang đăng nhập** của portal. Hiển thị qua component dùng chung `ui/components/AppLogo.vue` (ô nền sáng do CSS vẽ). Vite nhúng thẳng vào JS dạng base64 vì file nhỏ hơn 4KB. |
| `cmd/syncserver/portal/public/favicon.ico`<br>`web/public/favicon.ico` | 16, 32, 48, 64 | **ô nền sáng bo góc** | **Favicon trên tab trình duyệt**: portal (`:9090`) và trang settings (`localhost:8788`). |
| `cmd/syncserver/portal/public/favicon-32.png`<br>`web/public/favicon-32.png` | 32 | **ô nền sáng bo góc** | Favicon dạng PNG (trình duyệt hiện đại ưu tiên file này). |
| `cmd/syncserver/portal/public/apple-touch-icon.png`<br>`web/public/apple-touch-icon.png` | 180 | **nền sáng kín** | Icon khi "Thêm vào màn hình chính" trên iPhone/iPad. |

Hai thư mục `public/` có file giống hệt nhau. Mỗi project build và nhúng `dist/` riêng (portal vào `syncserver.exe`, trang settings vào `ancol-ani.exe`), mà Vite chỉ chép `public/` của chính project đó, nên mỗi bên phải có một bản. Script ghi cả hai.

### Vì sao có chỗ dùng nền sáng, có chỗ không

Icon là **hình màu tối trên nền trong suốt**. Đặt lên nền tối thì gần như không nhìn thấy:

- **Tab trình duyệt ở chế độ tối** → favicon có ô nền sáng `#f1eefc` bo góc.
- **Giao diện web nền tối** → `AppLogo.vue` vẽ ô nền sáng `#f1eefc`, viền `#35335a`, bóng đổ cứng kiểu pixel bằng CSS. Vì vậy `logo.png` vẫn để nền trong suốt.
- **iOS tô đen phần trong suốt** của apple-touch-icon → file này dùng nền sáng kín.
- **Icon .exe, bộ cài, icon Windows**: hiện trên nền sáng hoặc do hệ điều hành tự xử lý nên để nền trong suốt.

Đổi màu ô nền thì sửa **cả hai chỗ** cho đồng bộ: biến `$TileColor` trong `scripts/gen-icons.ps1` và `background` trong `ui/components/AppLogo.vue`.

---

## 2. Mascot GIF (chú chó nhảy)

Không phải icon nhưng cũng là ảnh nhận diện. Có **3 bản giống hệt nhau** (cùng mã md5):

| File | Dùng ở đâu |
|---|---|
| `frontend/src/mascot.gif` | **GIF mặc định của widget desktop**. `app.go` nhúng file này bằng `//go:embed` (biến `defaultGif`) và dùng khi người dùng chưa chọn GIF nào. |
| `mascot.gif` (thư mục gốc) | Bản lưu ở thư mục gốc repo, không có code nào đọc. |
| `cmd/syncserver/portal/src/assets/mascot.gif` | Mascot nhún nhảy phía trên **card đăng nhập** của portal (`LoginView.vue`). |

Script **không** xử lý mascot. Muốn đổi thì copy file GIF mới đè lên cả 3 chỗ. Kích thước nên vuông; GIF của widget bị giới hạn 3MB (`maxGifBytes` trong `app.go`).

---

## 3. Đã làm thế nào (11/2026)

1. **Lấy ảnh 64px từ `ancol.ico`** bằng `System.Drawing.Icon` (PowerShell). File `.ico` chỉ có đúng 1 ảnh 64×64 32-bit; kết quả lưu thành `logo.png` (nay ở `ui/assets/logo.png`).
2. **`build/appicon.png`**: phóng 64 → 1024 bằng nội suy HighQualityBicubic, thay logo "W" của Wails.
3. **Favicon**: vẽ ô nền sáng bo góc (bán kính 22%) rồi đặt icon vào giữa, cách viền 10%. Đóng gói nhiều cỡ (16/32/48/64) vào một file `.ico` có các ảnh nén PNG bên trong; trình duyệt và Windows Vista trở lên đều đọc được. Ban đầu chỉ làm cho portal, sau đó dùng chung cho cả trang settings khi hai bên đồng bộ thiết kế.
4. **apple-touch-icon**: ảnh 180px, nền sáng kín, cách viền 22px.
5. **Logo trong trang** (`AppLogo.vue`): component nhận prop `size`. Padding tính bằng **px** (12% của `size`), **không dùng `%` trong CSS**. Lý do: `padding: 12%` tính theo bề rộng của *phần tử cha*, có lúc padding to hơn cả ô logo và ảnh bị co về 0×0. Lỗi này đã xảy ra một lần: logo chỉ hiện ô trắng trống.
6. **Chống cache favicon**: các thẻ `<link rel="icon">` trong `index.html` của cả hai trang có đuôi `?v=2`. Trình duyệt cache favicon rất lâu; lúc trước `/favicon.ico` từng trả về 404 nên tab vẫn hiện icon cũ dù server đã có file mới.
7. Xóa code mẫu "Greet" của Wails (`frontend/src/main.js`, `app.css`, `style.css`, `logo-universal.png`, font Nunito), vì widget chỉ load `gifPlayer.js`.

Toàn bộ quy trình trên (trừ bước 5 và 7) giờ nằm trong `scripts/gen-icons.ps1`.

---

## 4. Cách đổi icon

1. Chuẩn bị ảnh mới: **PNG vuông, nền trong suốt, ≥ 1024px**. Nếu ảnh có màu tối, các ô nền sáng ở trên vẫn giữ cho nó dễ nhìn.
2. (Tùy chọn) Chạy thử vào một thư mục tạm để xem kết quả trước:
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts/gen-icons.ps1 -Source logo.png -OutRoot $env:TEMP\icons-test
   ```
3. Chạy thật (ghi đè các file trong repo):
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts/gen-icons.ps1 -Source logo.png
   ```
4. **Tăng `?v=`** (vd `?v=2` → `?v=3`) trên các thẻ `<link>` icon trong **cả** `cmd/syncserver/portal/index.html` **và** `web/index.html`.
5. Build lại và khởi động lại (xem `DEV-NOTES.md` mục 2, nhớ tắt process cũ trước):
   - Portal: `cd cmd/syncserver/portal && npm run build`, rồi build và chạy lại syncserver (server nhúng `portal/dist` lúc build).
   - Trang settings: `cd web && npm run build`, rồi `wails build -s` (app nhúng `web/dist`).
   - Icon file .exe: `wails build`.
6. Kiểm tra: tab trình duyệt của cả hai trang (thử cả giao diện sáng và tối), logo trên thanh điều hướng và trang đăng nhập, icon file `.exe` trong Explorer.

### Lưu ý khi dùng script

- **Nên dùng file PNG làm nguồn, không dùng `.ico`.** Script đọc được `.ico`, nhưng `System.Drawing` của .NET Framework bỏ qua mục 256px trong `.ico`, nên sẽ lấy ảnh 128px. Riêng `ancol.ico` hiện tại chỉ có 64px nên không bị ảnh hưởng.
- Nguồn nhỏ hơn 256px thì script vẫn chạy nhưng in cảnh báo mờ ảnh.
- Script ghi đè cả `ancol.ico` và `build/windows/icon.ico` để hai file luôn giống nhau.
