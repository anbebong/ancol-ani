# GIF Toolkit UI (Resize + Convert) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add portal pages for 2 of the 9 GIF Toolkit backend tools (Resize, Convert) on `cmd/syncserver/portal`, establishing a shared `ToolPage.vue`/`ResultPreview.vue` pattern the remaining 7 tools will reuse later.

**Architecture:** A small `toolsApi.js` posts `multipart/form-data` to the existing `/api/tools/*` endpoints (no auth header — these routes are public) and returns a blob + suggested filename. A shared `ResultPreview.vue` renders the blob (`<img>` for GIF, a plain download link for `.zip`) with a download button. `ToolPage.vue` is a generic "pick file → set params → run → preview" shell that `ResizeView.vue` plugs straight into; `ConvertView.vue` has its own 3-tab (video/image/images) layout since its inputs differ per tab, but reuses `ResultPreview.vue` for the output. A new `/tools` index page lists all 9 tools, 2 of them live.

**Tech Stack:** Vue 3 (`<script setup>`), Vue Router (hash history), daisyUI/Tailwind (existing `ui/` theme), native `fetch`. No test runner exists in `cmd/syncserver/portal` (checked `package.json` — no `test` script, no vitest/jest) — verification is `npm run build` (catches syntax/type errors) plus manual browser testing per this repo's CLAUDE.md ("For UI or frontend changes, start the dev server and use the feature in a browser before reporting the task as complete").

**Spec:** `docs/BA-gif-toolkit.md` §3 (portal), backend already implemented and reviewed on this same branch (`cmd/syncserver/tools_handlers.go`, routes `/api/tools/resize`, `/api/tools/convert`).

## Global Constraints

- Backend routes are public (no `Authorization` header) — do NOT use `api.js`'s `api()` helper (it attaches a Bearer token and treats 401 specially, neither of which applies here). Use a separate small helper.
- Router stays hash-based (`createWebHashHistory`), no `meta.auth` on the new routes.
- Reuse `ui/` components (`Icon`, `PageHeader`) and existing Tailwind/daisyUI classes — no new CSS framework, no new icon unless genuinely missing (checked `ui/icons.js`: `upload`/`download`/`image`/`images`/`sparkles` already exist; one new icon, `wand-2`, is needed for the nav link and must be added to `ui/icons.js` per this repo's rule that new icons go there, not duplicated locally).
- `/api/tools/resize` takes `width`, `height` as query params + multipart field `file` (a GIF). `/api/tools/convert` takes multipart field `file` (video OR image(s) — check the actual handler in `tools_handlers.go` for the exact field/kind-discriminator names before wiring the frontend call, since the plan that built it may have named things slightly differently than the BA table).
- No backend changes in this plan — if the frontend's expected request shape doesn't match what `tools_handlers.go` actually accepts, fix the frontend to match the backend (already-reviewed, already-shipped), not the other way around.

## Review Focus

- **A file the server rejects (wrong type, over size limit, GIF bomb, oversized video) must show the server's actual error text to the user**, not a generic "something went wrong" — the backend returns a plain-text body on 400/413/503/504 (per `tools_handlers.go`'s `http.Error` usage); the frontend must read and display `await res.text()`.
- **Switching tabs/files mid-preview must not show a stale result** — if a user runs Resize, then changes the width and clicks "Resize" again (or picks a new file), the old preview/blob URL must be cleared/revoked before the new request starts, not left showing while a new one loads silently in the background.
- **Convert's 3 tabs must not let a stale selection from one tab leak into another** — e.g. switching from "Nhiều ảnh" (multiple files selected) to "Video" tab must clear the multi-file selection, not silently submit it if the user forgets to pick a new file.
- **A request that's still in flight when the user navigates away (Vue Router) must not update a now-unmounted component's state** — guard the async response handler, mirroring the existing `GalleryView.vue` pattern (`mySeq`/sequence check) rather than assuming the component is still alive.
- **Object URLs must be revoked** when a new preview replaces an old one or the component unmounts, or repeated use leaks memory in a long browser session (same concern `api.js`'s `forgetImage` already handles for the gallery/library views — the new preview code needs its own equivalent, since it's not going through `cachedImageURL`).

---

## File Structure

- Create `cmd/syncserver/portal/src/toolsApi.js` — `postTool(path, formData) => Promise<{ blob: Blob, filename: string }>`, throws with the server's response text on non-2xx.
- Create `cmd/syncserver/portal/src/components/ResultPreview.vue` — takes a `result` prop (`{ blob, filename } | null`), an `error` prop, a `loading` prop; renders spinner / error alert / `<img>`-or-download-link + a "Tải xuống" button; owns its own object-URL lifecycle (create on new blob, revoke on replace/unmount).
- Create `cmd/syncserver/portal/src/components/ToolPage.vue` — generic single-GIF-in, single-GIF-out tool shell: file input (accepts `.gif`), a `<slot name="params">` for the tool-specific param inputs, a "Chạy" button, and `<ResultPreview>`. Emits `run` with `(file, formData)` for the parent to call `toolsApi.postTool`; the parent owns loading/error/result state and passes them back down (keeps `ToolPage.vue` free of knowledge about which endpoint it's calling).
- Create `cmd/syncserver/portal/src/views/tools/ResizeView.vue` — uses `ToolPage.vue`, params slot has `width`/`height` number inputs, calls `/api/tools/resize?width=..&height=..`.
- Create `cmd/syncserver/portal/src/views/tools/ConvertView.vue` — own layout (3 tabs: Video / Ảnh / Nhiều ảnh), each tab's file input(s) + tab-specific params (fps for video/images), one shared `<ResultPreview>` for the output. Switching tabs clears the other tabs' selected file(s).
- Create `cmd/syncserver/portal/src/views/ToolsIndexView.vue` — grid of 9 cards (title, icon, one-line description from BA §1); the 2 built ones (`Resize`, `Convert`) are links, the other 7 render disabled/"Sắp có".
- Modify `cmd/syncserver/portal/src/router.js` — add 3 routes: `/tools` (ToolsIndexView), `/tools/resize` (ResizeView), `/tools/convert` (ConvertView). No `meta.auth`.
- Modify `cmd/syncserver/portal/src/App.vue` — add `{ to: '/tools', label: 'Công cụ GIF', icon: 'wand-2' }` to the `links` computed array (goes in the base/public section, not the admin-only push).
- Modify `ui/icons.js` — add `wand-2` import + map entry.

---

### Task 1: `toolsApi.js` + `wand-2` icon

**Files:**
- Create: `cmd/syncserver/portal/src/toolsApi.js`
- Modify: `ui/icons.js`

**Interfaces:**
- Produces: `async function postTool(path, formData) => Promise<{ blob: Blob, filename: string }>` — on non-2xx response, throws `new Error(await res.text() || `Lỗi ${res.status}`)`. `filename` is derived from the `Content-Disposition` response header if present, else a sensible default based on the response `Content-Type` (`image/gif` → `result.gif`, `application/zip` → `result.zip`).

- [ ] **Step 1: Read `cmd/syncserver/tools_handlers.go`** to confirm: (a) whether `handleToolResize`/`handleToolConvert` set a `Content-Disposition` header on success, (b) the exact multipart field name for the uploaded file (BA says `file`, confirm it matches what Task 11 actually shipped), (c) `handleToolConvert`'s exact mechanism for choosing video vs. image(s) (a `kind` field, or `looksLikeVideo` sniffing the uploaded filename — confirm which, since the frontend must match it).

- [ ] **Step 2: Implement `postTool` in `toolsApi.js`**

```js
export async function postTool(path, formData) {
    const res = await fetch(path, { method: 'POST', body: formData });
    if (!res.ok) throw new Error((await res.text()) || `Lỗi ${res.status}`);
    const blob = await res.blob();
    const cd = res.headers.get('Content-Disposition') || '';
    const match = cd.match(/filename="?([^"]+)"?/);
    const filename = match ? match[1] : (blob.type === 'application/zip' ? 'result.zip' : 'result.gif');
    return { blob, filename };
}
```

(Adjust the field name / disposition-parsing against what Step 1 found — this is the exact shape unless Step 1 reveals the backend does something different, in which case match the backend.)

- [ ] **Step 3: Add `wand-2` to `ui/icons.js`**

```js
import wand2 from 'lucide-static/icons/wand-2.svg?raw';
```
Add `'wand-2': wand2,` to the `ICONS` map.

- [ ] **Step 4: Verify**

Run `cd cmd/syncserver/portal && npm run build` — must succeed with no errors (this file isn't imported by anything yet, so build success here just confirms no syntax errors).

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/portal/src/toolsApi.js ui/icons.js
git commit -m "feat(portal): add public tools API helper and wand-2 icon"
```

---

### Task 2: `ResultPreview.vue`

**Files:**
- Create: `cmd/syncserver/portal/src/components/ResultPreview.vue`

**Interfaces:**
- Consumes: nothing from Task 1 directly (works with plain `{ blob, filename }` shape `postTool` returns).
- Produces: a component with props `result: { blob: Blob, filename: string } | null`, `loading: Boolean`, `error: String` — used by `ToolPage.vue` (Task 3) and `ConvertView.vue` (Task 4).

- [ ] **Step 1: Implement**

```vue
<script setup>
import { ref, watch, onUnmounted } from 'vue';
import Icon from '@ui/components/Icon.vue';

const props = defineProps({
    result: { type: Object, default: null },
    loading: { type: Boolean, default: false },
    error: { type: String, default: '' },
});

const url = ref('');
function revoke() { if (url.value) URL.revokeObjectURL(url.value); url.value = ''; }

watch(() => props.result, (r) => {
    revoke();
    if (r) url.value = URL.createObjectURL(r.blob);
}, { immediate: true });

onUnmounted(revoke);
</script>

<template>
  <div class="flex flex-col items-center gap-3 py-4">
    <span v-if="loading" class="loading loading-spinner loading-lg text-primary"/>
    <div v-else-if="error" class="alert alert-error text-sm py-2 w-full">{{ error }}</div>
    <template v-else-if="result">
      <img v-if="result.blob.type === 'image/gif'" :src="url" alt="" class="max-w-full max-h-96 object-contain rounded"/>
      <a :href="url" :download="result.filename" class="btn btn-primary btn-sm gap-2">
        <Icon name="download"/> Tải xuống {{ result.filename }}
      </a>
    </template>
  </div>
</template>
```

- [ ] **Step 2: Verify**

`npm run build` succeeds (still unreferenced by any route, so this just checks syntax).

- [ ] **Step 3: Commit**

```bash
git add cmd/syncserver/portal/src/components/ResultPreview.vue
git commit -m "feat(portal): add ResultPreview component"
```

---

### Task 3: `ToolPage.vue` + `ResizeView.vue` + routing

**Files:**
- Create: `cmd/syncserver/portal/src/components/ToolPage.vue`
- Create: `cmd/syncserver/portal/src/views/tools/ResizeView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`
- Modify: `cmd/syncserver/portal/src/App.vue`

**Interfaces:**
- Consumes: `postTool` (Task 1), `ResultPreview` (Task 2).
- Produces: `ToolPage.vue` emits `run(file: File)` when the user clicks its run button — the parent (`ResizeView.vue`) listens, builds its own `FormData` (file + its own params), calls `postTool`, and passes `result`/`loading`/`error` back down as props.

- [ ] **Step 1: Implement `ToolPage.vue`**

```vue
<script setup>
import { ref } from 'vue';
import Icon from '@ui/components/Icon.vue';
import ResultPreview from './ResultPreview.vue';

defineProps({ result: Object, loading: Boolean, error: String, runLabel: { type: String, default: 'Chạy' } });
const emit = defineEmits(['run']);

const file = ref(null);
function onFile(e) { file.value = e.target.files[0] || null; }
function run() { if (file.value) emit('run', file.value); }
</script>

<template>
  <div class="flex flex-col gap-6 max-w-xl">
    <input type="file" accept=".gif,image/gif" class="file-input file-input-bordered w-full" @change="onFile"/>
    <slot name="params"/>
    <button type="button" class="btn btn-primary w-full" :disabled="!file || loading" @click="run">
      <Icon name="sparkles"/> {{ runLabel }}
    </button>
    <ResultPreview :result="result" :loading="loading" :error="error"/>
  </div>
</template>
```

- [ ] **Step 2: Implement `ResizeView.vue`**

```vue
<script setup>
import { ref } from 'vue';
import PageHeader from '@ui/components/PageHeader.vue';
import ToolPage from '../../components/ToolPage.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

const width = ref(320);
const height = ref(320);
const result = ref(null);
const loading = ref(false);
const error = ref('');

// Sequence guard (same pattern as GalleryView.vue): if the user runs again
// before this request resolves, a stale response must not overwrite state
// a newer request already set.
let seq = 0;
async function onRun(file) {
    const mySeq = ++seq;
    result.value = null;
    error.value = '';
    loading.value = true;
    try {
        const fd = new FormData();
        fd.append('file', file);
        const r = await postTool(`/api/tools/resize?width=${width.value}&height=${height.value}`, fd);
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
    <PageHeader icon="image" title="Resize GIF" subtitle="Đổi kích thước GIF theo pixel."/>
    <ToolPage :result="result" :loading="loading" :error="error" run-label="Resize" @run="onRun">
      <template #params>
        <div class="flex gap-3">
          <label class="form-control flex-1">
            <span class="label-text">Width</span>
            <input v-model.number="width" type="number" min="1" max="1024" class="input input-bordered"/>
          </label>
          <label class="form-control flex-1">
            <span class="label-text">Height</span>
            <input v-model.number="height" type="number" min="1" max="1024" class="input input-bordered"/>
          </label>
        </div>
      </template>
    </ToolPage>
  </div>
</template>
```

(Field name `file` and query params `width`/`height` per BA — adjust against Task 1 Step 1's findings if the shipped handler differs.)

- [ ] **Step 3: Add routes to `router.js`**

Add inside the `routes` array (before the catch-all):
```js
{ path: '/tools', name: 'tools', component: () => import('./views/ToolsIndexView.vue'), meta: { title: 'Công cụ GIF' } },
{ path: '/tools/resize', name: 'tools-resize', component: () => import('./views/tools/ResizeView.vue'), meta: { title: 'Resize GIF' } },
```
(`ToolsIndexView.vue` is created in Task 5 — this route will 404-import until then; either create a placeholder file now or defer this line to Task 5. Defer: add only the `/tools/resize` line in this task, add `/tools` and the Convert route in their own tasks, to keep each task's own files complete and buildable on their own.)

- [ ] **Step 4: Add nav link to `App.vue`**

In the `links` computed's `base` array, add `{ to: '/tools', label: 'Công cụ GIF', icon: 'wand-2' }` — this will 404 until Task 5 adds `/tools`; acceptable for one task's intermediate state as long as the final state (after all tasks) is correct. Alternatively point it at `/tools/resize` for now and fix to `/tools` in Task 5 — pick whichever is less churn once you're looking at the actual file.

- [ ] **Step 5: Verify**

`npm run build` succeeds. Then manually: `go run ./cmd/syncserver` from repo root (needs `data/` dir and a rebuilt `portal/dist` — run `npm run build` in `portal/` first since the Go server embeds `portal/dist`), open the local URL, navigate to `/#/tools/resize`, upload a real small GIF, set width/height, click Resize, confirm a resized GIF preview appears and downloads correctly. Also test an invalid input (width=0) and confirm the server's error text displays.

- [ ] **Step 6: Commit**

```bash
git add cmd/syncserver/portal/src/components/ToolPage.vue cmd/syncserver/portal/src/views/tools/ResizeView.vue cmd/syncserver/portal/src/router.js cmd/syncserver/portal/src/App.vue
git commit -m "feat(portal): add Resize tool page"
```

---

### Task 4: `ConvertView.vue` + routing

**Files:**
- Create: `cmd/syncserver/portal/src/views/tools/ConvertView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`

**Interfaces:**
- Consumes: `postTool` (Task 1), `ResultPreview` (Task 2). Does NOT use `ToolPage.vue` (different input shape per tab).

- [ ] **Step 1: Confirm the backend's convert request shape** (if not already settled in Task 1 Step 1): read `handleToolConvert`/`handleConvertImages`/`handleConvertVideo` in `tools_handlers.go` for: the field name(s) for video vs. single-image vs. multi-image uploads, how the handler tells them apart (kind param vs. extension-sniffing vs. multiple files present), the `fps` param name and which tabs accept it.

- [ ] **Step 2: Implement `ConvertView.vue`** — 3 tabs (daisyUI `tabs`/`tab` classes, matching the general look of other multi-section views in this portal):
  - **Video tab**: single file input (`accept="video/*"`), an `fps` number input, "Chuyển đổi" button → `FormData` with the video file + `fps` → `postTool('/api/tools/convert', fd)` (adjust field names per Step 1).
  - **Ảnh tab**: single file input (`accept="image/*"`) → same endpoint, no `fps`.
  - **Nhiều ảnh tab**: multi-file input (`multiple accept="image/*"`), a `delay`-in-ms number input → same endpoint with all files appended under the field name Step 1 found.
  - Switching tabs clears the other tabs' selected file(s) (reset the relevant `ref`s in the tab-switch handler) and clears any stale `result`/`error`.
  - Shared `loading`/`error`/`result` state and a single `<ResultPreview>` below the tabs.
  - Same sequence-guard pattern as `ResizeView.vue` (Task 3) around the `postTool` call — a response from a superseded request (user clicked Convert again, or switched tabs and ran again, before the first finished) must not overwrite newer state.

- [ ] **Step 3: Add the `/tools/convert` route** in `router.js`.

- [ ] **Step 4: Verify**

`npm run build` succeeds. Manually: rebuild `portal/dist`, run the server, test all 3 tabs with real small files (a short video, a single image, 2-3 images), confirm each produces a correct preview; test switching tabs mid-selection clears the previous tab's file; test an oversized/invalid input shows the server's error text.

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/portal/src/views/tools/ConvertView.vue cmd/syncserver/portal/src/router.js
git commit -m "feat(portal): add Convert tool page"
```

---

### Task 5: `ToolsIndexView.vue` + finalize routing/nav

**Files:**
- Create: `cmd/syncserver/portal/src/views/ToolsIndexView.vue`
- Modify: `cmd/syncserver/portal/src/router.js`
- Modify: `cmd/syncserver/portal/src/App.vue` (confirm nav link points at `/tools`, fix if Task 3 pointed it elsewhere)

**Interfaces:**
- Produces: the `/tools` landing page, completing the route set this plan adds.

- [ ] **Step 1: Implement `ToolsIndexView.vue`** — a grid of 9 cards (daisyUI `card`), one per tool from BA §1 (Convert, Resize, Crop, Optimize, Rotate, Reverse, Speed, Split, Caption), each with an icon + 1-line Vietnamese description. `Resize` and `Convert` are `<RouterLink>`s to their routes; the other 7 render as a disabled-looking card with a "Sắp có" (coming soon) badge, matching daisyUI's disabled-card convention (reduced opacity, no pointer, no link).

- [ ] **Step 2: Add the `/tools` route** in `router.js` (if not already present from Task 3's deferred step).

- [ ] **Step 3: Confirm nav link** in `App.vue` points at `/tools` (not `/tools/resize`).

- [ ] **Step 4: Verify**

`npm run build` succeeds. Manually: rebuild + run server, click "Công cụ GIF" in the nav, confirm the index page shows 9 cards, 2 clickable leading to working pages, 7 visibly disabled with no broken links.

- [ ] **Step 5: Commit**

```bash
git add cmd/syncserver/portal/src/views/ToolsIndexView.vue cmd/syncserver/portal/src/router.js cmd/syncserver/portal/src/App.vue
git commit -m "feat(portal): add tools index page, complete Resize+Convert UI"
```
