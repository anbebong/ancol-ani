# GIF Toolkit UI — Remaining 7 Tools Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add portal pages for the 7 GIF tools not yet covered (Crop, Optimize, Rotate, Reverse, Speed, Split, Caption), reusing the `ToolPage.vue`/`ResultPreview.vue` pattern already shipped for Resize/Convert, and flip their `ToolsIndexView.vue` cards from "Sắp có" to live links.

**Architecture:** Each tool gets one view file under `cmd/syncserver/portal/src/views/tools/`, following `ResizeView.vue`'s exact shape: `PageHeader` + `ToolPage` (file input + params slot + run button + `ResultPreview`) + a sequence-guarded `onRun` that builds `FormData`, calls `postTool`, and assigns `result`/`error`/`loading`. `ReverseView.vue` and `SplitView.vue` have no params (empty `#params` slot). `SplitView.vue`'s result is a `.zip`, not a GIF — `ResultPreview.vue` already handles this (its `<img>` only renders for `image/gif`, any other blob type just shows the download button), so no component change is needed.

**Tech Stack:** Vue 3 `<script setup>`, Vue Router (hash history), daisyUI/Tailwind, same portal already in place.

**Spec:** `docs/BA-gif-toolkit.md` (§3 API table, §4 limits) is the authority for every field name and param. `cmd/syncserver/tools_handlers.go` is the shipped ground truth — read it per-task before wiring a request, the same way Task 1 of the previous UI plan did, since the BA and the handler have drifted before (BA promised `percent`/`colors` params that were never implemented).

## Global Constraints

- Field name for the uploaded GIF is always multipart `file` (confirmed in `tools_handlers.go` for every one of these 9 endpoints).
- No test runner exists in the portal (`cmd/syncserver/portal/package.json` has no `test` script). Verification is `npm run build` plus manual testing against a real `go run ./cmd/syncserver` instance, exactly as done for the Resize/Convert plan.
- Routes get no `meta.auth` (these pages are public, like `/tools/resize`).
- Every view's `onRun` must use the sequence-guard pattern from `ResizeView.vue` (`let seq = 0; const mySeq = ++seq; ... if (mySeq === seq) ...`) so a stale response from a superseded run cannot overwrite newer state.
- `go run ./cmd/syncserver` needs `portal/dist` rebuilt first (`npm run build`) since the server embeds it via `go:embed`. Before starting a test server, check nothing is already holding port 9090 (`netstat -ano | grep 9090` or similar) — a leftover `go run` process from an earlier session caused exactly this during the previous plan's Task 3, and killing someone else's process needs the user's permission first.

## Review Focus

- **Rotate's `angle` param**: the backend (`rotateGIF`) only documents 90/180/270 as meaningful values (BA §3). An arbitrary number typed into a free-text field would produce a confusing error far from the point of entry. Use a `<select>` with exactly those three options, not a number input.
- **Optimize's `level` param**: BA says 1-3, no `colors` param exists despite what ezgif offers. A `<select>`/bounded number input, not a free numeric field that could send 0 or 99 and get a raw backend error.
- **Caption's `color` param**: the backend expects a hex string (`parseHexColor` in `gif_ops.go` from the backend plan). An `<input type="color">` always yields a valid `#rrggbb` value — don't use a free-text field that could send an invalid string.
- **Split's non-GIF result**: confirm in Task 6 that `ResultPreview.vue` genuinely renders a working download link (not a broken image attempt) for a `.zip` blob — this is an existing component, not new code, but it's never been exercised with a non-GIF response until this task.
- **Caption's `text` param empty-string edge case**: BA doesn't say what happens with empty text; trace what the backend does (likely a no-op caption or a harmless empty label) and make sure the frontend doesn't block submission on an empty text field if the backend itself allows it — check `captionGIF`'s behavior for `text=""` before deciding whether the Caption button should be disabled on empty text.

---

### Task 1: `CropView.vue`

**Files:**
- Create: `cmd/syncserver/portal/src/views/tools/CropView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`

**Interfaces:**
- Consumes: `postTool` (`../../toolsApi.js`), `errMsg` (`../../api.js`), `ToolPage` (`../../components/ToolPage.vue`), `PageHeader` (`@ui/components/PageHeader.vue`).

- [ ] **Step 1: Confirm `/api/tools/crop`'s exact param names** by reading `handleToolCrop` in `cmd/syncserver/tools_handlers.go` (expected: multipart `file`, query `x`/`y`/`width`/`height`, all ints — confirm against the live file, not this plan's memory of it).

- [ ] **Step 2: Implement `CropView.vue`**, following `ResizeView.vue`'s structure exactly:

```vue
<script setup>
import { ref } from 'vue';
import PageHeader from '@ui/components/PageHeader.vue';
import ToolPage from '../../components/ToolPage.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

const x = ref(0);
const y = ref(0);
const width = ref(100);
const height = ref(100);
const result = ref(null);
const loading = ref(false);
const error = ref('');

let seq = 0;
async function onRun(file) {
    const mySeq = ++seq;
    result.value = null;
    error.value = '';
    loading.value = true;
    try {
        const fd = new FormData();
        fd.append('file', file);
        const r = await postTool(`/api/tools/crop?x=${x.value}&y=${y.value}&width=${width.value}&height=${height.value}`, fd);
        if (mySeq === seq) result.value = r;
    } catch (e) {
        if (mySeq === seq) error.value = errMsg(e);
    } finally {
        if (mySeq === seq) loading.value = false;
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="square" title="Crop GIF" subtitle="Cắt một vùng chữ nhật của GIF."/>
    <ToolPage :result="result" :loading="loading" :error="error" run-label="Crop" @run="onRun">
      <template #params>
        <div class="grid grid-cols-2 gap-3">
          <label class="form-control"><span class="label-text">X</span><input v-model.number="x" type="number" min="0" class="input input-bordered"/></label>
          <label class="form-control"><span class="label-text">Y</span><input v-model.number="y" type="number" min="0" class="input input-bordered"/></label>
          <label class="form-control"><span class="label-text">Width</span><input v-model.number="width" type="number" min="1" class="input input-bordered"/></label>
          <label class="form-control"><span class="label-text">Height</span><input v-model.number="height" type="number" min="1" class="input input-bordered"/></label>
        </div>
      </template>
    </ToolPage>
  </div>
</template>
```

(Adjust field/query names per Step 1 if the live handler differs.)

- [ ] **Step 3: Add the route** to `router.js`, next to `tools-resize`:
```js
{ path: '/tools/crop', name: 'tools-crop', component: () => import('./views/tools/CropView.vue'), meta: { title: 'Crop GIF' } },
```

- [ ] **Step 4: Verify** — `npm run build` succeeds. Manually: rebuild `portal/dist`, run the server, upload a real GIF at `/#/tools/crop`, crop a sub-region, confirm the preview shows the cropped result; test an out-of-bounds crop and confirm the server's error text displays.

- [ ] **Step 5: Commit**
```bash
git add cmd/syncserver/portal/src/views/tools/CropView.vue cmd/syncserver/portal/src/router.js
git commit -m "feat(portal): add Crop tool page"
```

---

### Task 2: `OptimizeView.vue`

**Files:**
- Create: `cmd/syncserver/portal/src/views/tools/OptimizeView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`

- [ ] **Step 1: Confirm `/api/tools/optimize`'s param** by reading `handleToolOptimize` (expected: multipart `file`, query `level` int 1-3, default 2).

- [ ] **Step 2: Implement `OptimizeView.vue`**, same shape as Task 1's `CropView.vue` but with a single `level` param rendered as a `<select>` with options 1/2/3 (labels: "1 - Nhẹ", "2 - Vừa", "3 - Mạnh"), posting to `/api/tools/optimize?level=${level.value}`. Icon: `sparkles`. Title: "Optimize GIF". Subtitle: "Giảm dung lượng GIF."

- [ ] **Step 3: Add the route** `/tools/optimize` → `tools-optimize` in `router.js`.

- [ ] **Step 4: Verify** — `npm run build` succeeds. Manually: test all 3 levels against a real GIF, confirm each returns a smaller (or equal) file and the preview renders; confirm an invalid `level` (if the UI could somehow send one — it can't via the select, but confirm the component itself never sends out-of-range values).

- [ ] **Step 5: Commit**
```bash
git add cmd/syncserver/portal/src/views/tools/OptimizeView.vue cmd/syncserver/portal/src/router.js
git commit -m "feat(portal): add Optimize tool page"
```

---

### Task 3: `RotateView.vue`

**Files:**
- Create: `cmd/syncserver/portal/src/views/tools/RotateView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`

- [ ] **Step 1: Confirm `/api/tools/rotate`'s params** by reading `handleToolRotate` (expected: multipart `file`, query `angle` int, query `flip` string `none`/`h`/`v`, default `none`).

- [ ] **Step 2: Implement `RotateView.vue`**, same shape as Task 1, with two params:
  - `angle`: `<select>` with exactly 3 options — 90, 180, 270 (see Review Focus: no free-text angle input).
  - `flip`: `<select>` with options `none`/`h`/`v` (labels: "Không lật", "Lật ngang", "Lật dọc").
  Post to `/api/tools/rotate?angle=${angle.value}&flip=${flip.value}`. Icon: `rotate-ccw`. Title: "Rotate GIF". Subtitle: "Xoay hoặc lật GIF."

- [ ] **Step 3: Add the route** `/tools/rotate` → `tools-rotate` in `router.js`.

- [ ] **Step 4: Verify** — `npm run build` succeeds. Manually: test at least one angle + one flip combination against a real GIF, confirm the preview visibly reflects the rotation/flip.

- [ ] **Step 5: Commit**
```bash
git add cmd/syncserver/portal/src/views/tools/RotateView.vue cmd/syncserver/portal/src/router.js
git commit -m "feat(portal): add Rotate tool page"
```

---

### Task 4: `ReverseView.vue`

**Files:**
- Create: `cmd/syncserver/portal/src/views/tools/ReverseView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`

- [ ] **Step 1: Confirm `/api/tools/reverse`'s shape** by reading `handleToolReverse` (expected: multipart `file` only, no params).

- [ ] **Step 2: Implement `ReverseView.vue`**, same shape as `ResizeView.vue` but with an empty `#params` slot (no inputs at all — just the file picker and the run button, inherited from `ToolPage.vue`). Posts to `/api/tools/reverse` with no query string. Icon: `history`. Title: "Reverse GIF". Subtitle: "Đảo ngược thứ tự khung hình."

- [ ] **Step 3: Add the route** `/tools/reverse` → `tools-reverse` in `router.js`.

- [ ] **Step 4: Verify** — `npm run build` succeeds. Manually: upload a real multi-frame GIF, confirm the returned GIF plays in reverse order (visually distinguishable — use a GIF whose frames have an obvious order, e.g. frames of increasing size/color).

- [ ] **Step 5: Commit**
```bash
git add cmd/syncserver/portal/src/views/tools/ReverseView.vue cmd/syncserver/portal/src/router.js
git commit -m "feat(portal): add Reverse tool page"
```

---

### Task 5: `SpeedView.vue`

**Files:**
- Create: `cmd/syncserver/portal/src/views/tools/SpeedView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`

- [ ] **Step 1: Confirm `/api/tools/speed`'s param** by reading `handleToolSpeed` (expected: multipart `file`, query `factor` float, default 1 — e.g. 0.5 = half speed, 2 = double speed).

- [ ] **Step 2: Implement `SpeedView.vue`**, same shape as Task 1, with a single `factor` param as a number input (`step="0.1"`, `min="0.1"`, `max="10"`, default `1`), posting to `/api/tools/speed?factor=${factor.value}`. Icon: `wand-2`. Title: "Speed GIF". Subtitle: "Tăng/giảm tốc độ phát GIF."

- [ ] **Step 3: Add the route** `/tools/speed` → `tools-speed` in `router.js`.

- [ ] **Step 4: Verify** — `npm run build` succeeds. Manually: test factor=2 and factor=0.5 against a real GIF, confirm the result plays faster/slower (frame delays visibly changed, or at minimum the request succeeds and returns a valid GIF); test factor=0 and confirm the server's error text (if any) displays rather than a frontend crash.

- [ ] **Step 5: Commit**
```bash
git add cmd/syncserver/portal/src/views/tools/SpeedView.vue cmd/syncserver/portal/src/router.js
git commit -m "feat(portal): add Speed tool page"
```

---

### Task 6: `SplitView.vue`

**Files:**
- Create: `cmd/syncserver/portal/src/views/tools/SplitView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`

**Interfaces:**
- Consumes: `ResultPreview.vue` as-is — its `<img v-if="result.blob.type === 'image/gif'">` guard means a `.zip` result (`application/zip`, per `toolsApi.js`'s `postTool` fallback-filename logic) renders no image, only the download button. No change to `ResultPreview.vue` needed; this task just needs to confirm that in practice (Review Focus item).

- [ ] **Step 1: Confirm `/api/tools/split`'s shape** by reading `handleToolSplit` (expected: multipart `file` only, response `application/zip`).

- [ ] **Step 2: Implement `SplitView.vue`**, same shape as `ReverseView.vue` (empty `#params` slot, no query string), posting to `/api/tools/split`. Icon: `images`. Title: "Split GIF". Subtitle: "Tách GIF thành từng khung hình (.zip các ảnh PNG)."

- [ ] **Step 3: Add the route** `/tools/split` → `tools-split` in `router.js`.

- [ ] **Step 4: Verify** — `npm run build` succeeds. Manually: upload a real multi-frame GIF, confirm the response is a downloadable `.zip` (not a broken/attempted image render — per Review Focus, actually look at `ResultPreview`'s rendered output, not just the HTTP response), and that the downloaded zip actually contains one PNG per frame when extracted.

- [ ] **Step 5: Commit**
```bash
git add cmd/syncserver/portal/src/views/tools/SplitView.vue cmd/syncserver/portal/src/router.js
git commit -m "feat(portal): add Split tool page"
```

---

### Task 7: `CaptionView.vue`

**Files:**
- Create: `cmd/syncserver/portal/src/views/tools/CaptionView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`

- [ ] **Step 1: Confirm `/api/tools/caption`'s params and empty-text behavior** by reading `handleToolCaption` and `captionGIF` (in `gif_ops.go`, from the backend plan) — confirm: multipart `file`, query `text`/`position`(`top`/`bottom`)/`color`(hex)/`size`(int), and specifically what `captionGIF` does when `text=""` (per this plan's Review Focus — decide whether the Caption button should be disabled on empty text based on what you find, not by assumption).

- [ ] **Step 2: Implement `CaptionView.vue`**, same shape as Task 1, with 4 params:
  - `text`: `<input type="text">`.
  - `position`: `<select>` with `top`/`bottom` (labels "Trên", "Dưới").
  - `color`: `<input type="color">` (always yields a valid `#rrggbb` — see Review Focus), default `#ffffff`.
  - `size`: number input, `min="8"`, `max="72"`, default `16`.
  Post to `/api/tools/caption?text=${encodeURIComponent(text.value)}&position=${position.value}&color=${encodeURIComponent(color.value)}&size=${size.value}`. Icon: `type`. Title: "Caption GIF". Subtitle: "Thêm chữ vào GIF."
  If Step 1 found that empty text is rejected or is a meaningless no-op server-side, disable the run button when `text.value.trim() === ''` (in addition to `ToolPage.vue`'s own file-required disable — this requires either passing an extra prop to `ToolPage.vue` or, more simply, not using `ToolPage.vue`'s own button and instead composing this view like `ConvertView.vue` did, with its own button bound to `!file || !text || loading`). Pick whichever requires less rework once you're looking at `ToolPage.vue`'s actual prop surface; if Step 1 found empty text is harmless, skip this and keep using `ToolPage.vue` unmodified.

- [ ] **Step 3: Add the route** `/tools/caption` → `tools-caption` in `router.js`.

- [ ] **Step 4: Verify** — `npm run build` succeeds. Manually: caption a real GIF with real text/color/position/size, confirm the preview visibly shows the caption; test empty text per whatever Step 1 determined the correct behavior to be.

- [ ] **Step 5: Commit**
```bash
git add cmd/syncserver/portal/src/views/tools/CaptionView.vue cmd/syncserver/portal/src/router.js
git commit -m "feat(portal): add Caption tool page"
```

---

### Task 8: Finalize `ToolsIndexView.vue`

**Files:**
- Modify: `cmd/syncserver/portal/src/views/ToolsIndexView.vue`

**Interfaces:**
- Consumes: the 7 routes added in Tasks 1-7 (`tools-crop`, `tools-optimize`, `tools-rotate`, `tools-reverse`, `tools-speed`, `tools-split`, `tools-caption`).

- [ ] **Step 1: Flip all 7 remaining `tools` array entries to `ready: true`** in `ToolsIndexView.vue`, adding each one's `to` path (`/tools/crop`, `/tools/optimize`, `/tools/rotate`, `/tools/reverse`, `/tools/speed`, `/tools/split`, `/tools/caption`). After this change every one of the 9 cards is a live `<RouterLink>`; none render the disabled "Sắp có" state.

- [ ] **Step 2: Verify** — `npm run build` succeeds. Manually: visit `/#/tools`, confirm all 9 cards are clickable and lead to a working page (no 404, no blank screen).

- [ ] **Step 3: Commit**
```bash
git add cmd/syncserver/portal/src/views/ToolsIndexView.vue
git commit -m "feat(portal): complete full GIF toolkit UI (all 9 tools live)"
```

---
