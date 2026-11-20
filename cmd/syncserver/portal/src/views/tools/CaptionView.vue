<script setup>
import { ref } from 'vue';
import PageHeader from '@ui/components/PageHeader.vue';
import Icon from '@ui/components/Icon.vue';
import ResultPreview from '../../components/ResultPreview.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

const file = ref(null);
const text = ref('');
const position = ref('bottom');
const color = ref('#ffffff');
const size = ref(16);

const result = ref(null);
const loading = ref(false);
const error = ref('');

function onFile(e) { file.value = e.target.files[0] || null; }

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
        const url = `/api/tools/caption?text=${encodeURIComponent(text.value)}&position=${position.value}&color=${encodeURIComponent(color.value)}&size=${size.value}`;
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
    <PageHeader icon="type" title="Caption GIF" subtitle="Thêm chữ vào GIF."/>
    <div class="flex flex-col gap-6 max-w-xl">
      <input type="file" accept=".gif,image/gif" class="file-input file-input-bordered w-full" @change="onFile"/>

      <label class="form-control">
        <span class="label-text">Nội dung</span>
        <input v-model="text" type="text" class="input input-bordered" placeholder="Nhập chữ cần thêm..."/>
      </label>

      <div class="flex gap-3">
        <label class="form-control flex-1">
          <span class="label-text">Vị trí</span>
          <select v-model="position" class="select select-bordered">
            <option value="top">Trên</option>
            <option value="bottom">Dưới</option>
          </select>
        </label>
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
