<script setup>
import { ref, onUnmounted } from 'vue';
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

const mode = ref('pixel'); // 'pixel' | 'percent'
const width = ref(320);
const height = ref(320);
const percent = ref(100);

function revokePreview() { if (previewUrl.value) URL.revokeObjectURL(previewUrl.value); previewUrl.value = ''; }
onUnmounted(revokePreview);

function loadFile(f) {
    file.value = f;
    revokePreview();
    naturalW.value = 0; naturalH.value = 0;
    if (f) previewUrl.value = URL.createObjectURL(f);
}

function onFile(e) { fromChain.value = false; loadFile(e.target.files[0] || null); }

function onImgLoad(e) {
    naturalW.value = e.target.naturalWidth;
    naturalH.value = e.target.naturalHeight;
    width.value = naturalW.value;
    height.value = naturalH.value;
}

if (chainedFile.value) {
    fromChain.value = true;
    loadFile(takeChainedFile());
}

// resizeGIF only takes absolute width/height, so "theo %" is computed here
// from the selected file's natural size before the request is sent.
function targetSize() {
    if (mode.value === 'pixel') return { w: width.value, h: height.value };
    const w = Math.max(1, Math.round((naturalW.value * percent.value) / 100));
    const h = Math.max(1, Math.round((naturalH.value * percent.value) / 100));
    return { w, h };
}

function canRun() {
    if (!file.value) return false;
    if (mode.value === 'percent') return !!naturalW.value && percent.value > 0;
    return width.value >= 1 && height.value >= 1;
}

const result = ref(null);
const loading = ref(false);
const error = ref('');

let seq = 0;
async function run() {
    if (!canRun()) return;
    const mySeq = ++seq;
    result.value = null;
    error.value = '';
    loading.value = true;
    try {
        const { w, h } = targetSize();
        const fd = new FormData();
        fd.append('file', file.value);
        const r = await postTool(`/api/tools/resize?width=${w}&height=${h}`, fd);
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
    <PageHeader icon="image" title="Resize GIF" subtitle="Đổi kích thước GIF theo pixel hoặc theo %."/>
    <div class="flex flex-col gap-4 max-w-xl">
      <div v-if="fromChain" class="alert alert-info text-sm py-2">Đã nạp sẵn GIF vừa tạo từ công cụ trước.</div>
      <input type="file" accept=".gif,image/gif" class="file-input file-input-bordered w-full" @change="onFile"/>
      <img v-if="previewUrl" :src="previewUrl" alt="" class="max-w-full max-h-64 object-contain rounded" @load="onImgLoad"/>
      <p v-if="naturalW" class="text-sm text-base-content/60">Kích thước gốc: {{ naturalW }}×{{ naturalH }}</p>

      <div role="tablist" class="tabs tabs-boxed w-fit">
        <a role="tab" class="tab" :class="{ 'tab-active': mode === 'pixel' }" @click="mode = 'pixel'">Theo pixel</a>
        <a role="tab" class="tab" :class="{ 'tab-active': mode === 'percent' }" @click="mode = 'percent'">Theo %</a>
      </div>

      <div v-if="mode === 'pixel'" class="flex gap-3">
        <label class="form-control flex-1">
          <span class="label-text">Width</span>
          <input v-model.number="width" type="number" min="1" max="1024" class="input input-bordered"/>
        </label>
        <label class="form-control flex-1">
          <span class="label-text">Height</span>
          <input v-model.number="height" type="number" min="1" max="1024" class="input input-bordered"/>
        </label>
      </div>
      <label v-else class="form-control">
        <span class="label-text">Phần trăm kích thước gốc</span>
        <input v-model.number="percent" type="number" min="1" max="500" class="input input-bordered"/>
      </label>

      <button type="button" class="btn btn-primary w-full" :disabled="!canRun() || loading" @click="run">
        <Icon name="sparkles"/> Resize
      </button>

      <ResultPreview :result="result" :loading="loading" :error="error"/>
    </div>
  </div>
</template>
