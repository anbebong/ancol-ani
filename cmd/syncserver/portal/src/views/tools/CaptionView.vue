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
const text = ref('');
const color = ref('#ffffff');
const size = ref(16);

// position is "top" | "bottom" (server's built-in margin placement) or
// "custom" (free drag on the preview, in image-pixel coords posX/posY).
const position = ref('bottom');
const posX = ref(0);
const posY = ref(0);

const previewUrl = ref('');
const naturalW = ref(0);
const naturalH = ref(0);
const boxEl = ref(null);

function revokePreview() { if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); previewUrl.value = ''; }
onUnmounted(revokePreview);

function loadFile(f) {
    file.value = f;
    revokePreview();
    naturalW.value = 0; naturalH.value = 0;
    if (f) previewUrl.value = URL.createObjectURL(f);
}

function onFile(e) { fromChain.value = false; loadFile(e.target.files[0] || null); }

if (chainedFile.value) {
    fromChain.value = true;
    loadFile(takeChainedFile());
}

const MARGIN = 4;
function onImgLoad(e) {
    naturalW.value = e.target.naturalWidth;
    naturalH.value = e.target.naturalHeight;
    posX.value = MARGIN;
    posY.value = position.value === 'top' ? MARGIN : Math.max(0, naturalH.value - MARGIN - size.value);
}

function presetPosition(p) {
    position.value = p;
    if (!naturalH.value) return;
    posX.value = MARGIN;
    posY.value = p === 'top' ? MARGIN : Math.max(0, naturalH.value - MARGIN - size.value);
}

// Drag-to-position marker, mirroring ezgif's "drag the text to change its
// position" caption UX.
const dragging = ref(false);
const dragOffset = ref({ x: 0, y: 0 });

function clientToImagePx(clientX, clientY) {
    const rect = boxEl.value.getBoundingClientRect();
    const scaleX = naturalW.value / rect.width;
    const scaleY = naturalH.value / rect.height;
    return { px: (clientX - rect.left) * scaleX, py: (clientY - rect.top) * scaleY };
}

function onMarkerDragStart(e) {
    e.stopPropagation();
    const { px, py } = clientToImagePx(e.clientX, e.clientY);
    dragOffset.value = { x: px - posX.value, y: py - posY.value };
    dragging.value = true;
}

function onDragMove(e) {
    if (!dragging.value || !naturalW.value) return;
    const { px, py } = clientToImagePx(e.clientX, e.clientY);
    position.value = 'custom';
    posX.value = Math.round(Math.min(Math.max(px - dragOffset.value.x, 0), naturalW.value));
    posY.value = Math.round(Math.min(Math.max(py - dragOffset.value.y, 0), naturalH.value));
}

function onDragEnd() { dragging.value = false; }

const markerStyle = computed(() => {
    if (!naturalW.value || !boxEl.value) return { display: 'none' };
    const rect = boxEl.value.getBoundingClientRect();
    const scaleX = rect.width / naturalW.value;
    const scaleY = rect.height / naturalH.value;
    return {
        left: `${posX.value * scaleX}px`,
        top: `${posY.value * scaleY}px`,
        fontSize: `${Math.max(8, size.value * scaleY)}px`,
        color: color.value,
    };
});

const result = ref(null);
const loading = ref(false);
const error = ref('');

// captionGIF rejects an empty/whitespace-only text server-side (see
// gif_ops.go), so the button is disabled on empty text rather than letting
// the user hit that error round-trip.
function canRun() { return !!file.value && text.value.trim() !== ''; }

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
        let url = `/api/tools/caption?text=${encodeURIComponent(text.value)}&position=${position.value}&color=${encodeURIComponent(color.value)}&size=${size.value}`;
        if (position.value === 'custom') url += `&x=${posX.value}&y=${posY.value}`;
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
    <PageHeader icon="type" title="Caption GIF" subtitle="Kéo chữ trên ảnh để đặt vị trí, hoặc chọn nhanh Trên/Dưới."/>
    <div class="flex flex-col gap-6 max-w-xl">
      <div v-if="fromChain" class="alert alert-info text-sm py-2">Đã nạp sẵn GIF vừa tạo từ công cụ trước.</div>
      <input type="file" accept=".gif,image/gif" class="file-input file-input-bordered w-full" @change="onFile"/>

      <div
        v-if="previewUrl"
        ref="boxEl"
        class="relative inline-block select-none max-w-full"
        @mousemove="onDragMove"
        @mouseup="onDragEnd"
        @mouseleave="dragging && onDragEnd()"
      >
        <img :src="previewUrl" alt="" class="max-w-full max-h-96 object-contain block" @load="onImgLoad" draggable="false"/>
        <div
          v-if="naturalW && text.trim()"
          class="absolute font-bold whitespace-nowrap px-1 cursor-move"
          style="text-shadow: 1px 1px 2px rgba(0,0,0,0.8)"
          :style="markerStyle"
          @mousedown="onMarkerDragStart"
        >{{ text }}</div>
      </div>

      <label class="form-control">
        <span class="label-text">Nội dung</span>
        <input v-model="text" type="text" class="input input-bordered" placeholder="Nhập chữ cần thêm..."/>
      </label>

      <div class="flex gap-3">
        <div class="form-control flex-1">
          <span class="label-text">Vị trí</span>
          <div class="join">
            <button type="button" class="btn join-item" :class="{ 'btn-active': position === 'top' }" @click="presetPosition('top')">Trên</button>
            <button type="button" class="btn join-item" :class="{ 'btn-active': position === 'bottom' }" @click="presetPosition('bottom')">Dưới</button>
            <button type="button" class="btn join-item" :class="{ 'btn-active': position === 'custom' }" disabled title="Kéo chữ trên ảnh để đặt vị trí tự do">Tự do</button>
          </div>
        </div>
        <label class="form-control">
          <span class="label-text">Màu chữ</span>
          <input v-model="color" type="color" class="input input-bordered h-10 w-16 p-1"/>
        </label>
        <label class="form-control flex-1">
          <span class="label-text">Cỡ chữ</span>
          <input v-model.number="size" type="number" min="8" max="72" class="input input-bordered"/>
        </label>
      </div>

      <button type="button" class="btn btn-primary w-full" :disabled="!canRun() || loading" @click="run">
        <Icon name="sparkles"/> Thêm chữ
      </button>

      <ResultPreview :result="result" :loading="loading" :error="error"/>
    </div>
  </div>
</template>
