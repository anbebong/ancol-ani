<script setup>
import { ref, computed, onUnmounted } from 'vue';
import PageHeader from '@ui/components/PageHeader.vue';
import ToolBackLink from '../../components/ToolBackLink.vue';
import Icon from '@ui/components/Icon.vue';
import ResultPreview from '../../components/ResultPreview.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';
import { chainedFile, takeChainedFile } from '../../toolChain.js';

const file = ref(null);
const fromChain = ref(false);

const previewUrl = ref('');
const naturalW = ref(0);
const naturalH = ref(0);
const imgEl = ref(null);
const boxEl = ref(null);

// Selection rectangle, kept in the IMAGE's own pixel coordinates (what the
// crop API takes) so the number inputs and the drag overlay always agree.
const x = ref(0);
const y = ref(0);
const width = ref(0);
const height = ref(0);

function revokePreview() { if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); previewUrl.value = ''; }
onUnmounted(revokePreview);

function loadFile(f) {
    file.value = f;
    revokePreview();
    x.value = 0; y.value = 0; width.value = 0; height.value = 0;
    naturalW.value = 0; naturalH.value = 0;
    if (f) previewUrl.value = URL.createObjectURL(f);
}

function onFile(e) { fromChain.value = false; loadFile(e.target.files[0] || null); }

if (chainedFile.value) {
    fromChain.value = true;
    loadFile(takeChainedFile());
}

function onImgLoad() {
    naturalW.value = imgEl.value.naturalWidth;
    naturalH.value = imgEl.value.naturalHeight;
    // Default selection: the whole image, so "Crop" with no drag is a no-op.
    x.value = 0; y.value = 0; width.value = naturalW.value; height.value = naturalH.value;
}

// Drag-to-select, mirroring ezgif's crop UX: drag on the image preview
// instead of only typing x/y/width/height.
const dragging = ref(false);
const dragStart = ref({ x: 0, y: 0 });

function clientToImagePx(clientX, clientY) {
    const rect = boxEl.value.getBoundingClientRect();
    const scaleX = naturalW.value / rect.width;
    const scaleY = naturalH.value / rect.height;
    const px = Math.min(Math.max(clientX - rect.left, 0), rect.width) * scaleX;
    const py = Math.min(Math.max(clientY - rect.top, 0), rect.height) * scaleY;
    return { px: Math.round(px), py: Math.round(py) };
}

function onDragStart(e) {
    if (!naturalW.value) return;
    dragging.value = true;
    const { px, py } = clientToImagePx(e.clientX, e.clientY);
    dragStart.value = { x: px, y: py };
    x.value = px; y.value = py; width.value = 0; height.value = 0;
}

function onDragMove(e) {
    if (!dragging.value) return;
    const { px, py } = clientToImagePx(e.clientX, e.clientY);
    x.value = Math.min(dragStart.value.x, px);
    y.value = Math.min(dragStart.value.y, py);
    width.value = Math.abs(px - dragStart.value.x);
    height.value = Math.abs(py - dragStart.value.y);
}

function onDragEnd() {
    dragging.value = false;
    if (width.value < 1 || height.value < 1) {
        // A click with no drag: fall back to the whole image rather than a
        // zero-size crop the API would reject.
        x.value = 0; y.value = 0; width.value = naturalW.value; height.value = naturalH.value;
    }
}

// Selection overlay position in the *displayed* image's CSS pixels.
const selectionStyle = computed(() => {
    if (!naturalW.value || !boxEl.value) return { display: 'none' };
    const rect = boxEl.value.getBoundingClientRect();
    const scaleX = rect.width / naturalW.value;
    const scaleY = rect.height / naturalH.value;
    return {
        left: `${x.value * scaleX}px`,
        top: `${y.value * scaleY}px`,
        width: `${width.value * scaleX}px`,
        height: `${height.value * scaleY}px`,
    };
});

const result = ref(null);
const loading = ref(false);
const error = ref('');

function canRun() { return !!file.value && width.value >= 1 && height.value >= 1; }

let seq = 0;
async function run() {
    if (!canRun()) return;
    const mySeq = ++seq;
    result.value = null;
    error.value = '';
    loading.value = true;
    try {
        const fd = new FormData();
        fd.append('file', file.value);
        const url = `/api/tools/crop?x=${x.value}&y=${y.value}&width=${width.value}&height=${height.value}`;
        const r = await postTool(url, fd);
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
    <ToolBackLink/>
    <PageHeader icon="square" title="Crop GIF" subtitle="Kéo chuột trên ảnh để chọn vùng cắt."/>
    <div class="flex flex-col gap-4 max-w-xl">
      <div v-if="fromChain" class="alert alert-info text-sm py-2">Đã nạp sẵn GIF vừa tạo từ công cụ trước.</div>
      <input type="file" accept=".gif,image/gif" class="file-input file-input-bordered w-full" @change="onFile"/>

      <div
        v-if="previewUrl"
        ref="boxEl"
        class="relative inline-block select-none max-w-full cursor-crosshair"
        @mousedown="onDragStart"
        @mousemove="onDragMove"
        @mouseup="onDragEnd"
        @mouseleave="dragging && onDragEnd()"
      >
        <img ref="imgEl" :src="previewUrl" alt="" class="max-w-full max-h-96 object-contain block" @load="onImgLoad" draggable="false"/>
        <div v-if="naturalW" class="absolute border-2 border-primary bg-primary/20" :style="selectionStyle"/>
      </div>

      <div class="grid grid-cols-2 gap-3">
        <label class="form-control"><span class="label-text">X</span><input v-model.number="x" type="number" min="0" class="input input-bordered"/></label>
        <label class="form-control"><span class="label-text">Y</span><input v-model.number="y" type="number" min="0" class="input input-bordered"/></label>
        <label class="form-control"><span class="label-text">Width</span><input v-model.number="width" type="number" min="1" class="input input-bordered"/></label>
        <label class="form-control"><span class="label-text">Height</span><input v-model.number="height" type="number" min="1" class="input input-bordered"/></label>
      </div>

      <button type="button" class="btn btn-primary w-full" :disabled="!canRun() || loading" @click="run">
        <Icon name="sparkles"/> Crop
      </button>

      <ResultPreview :result="result" :loading="loading" :error="error"/>
    </div>
  </div>
</template>
