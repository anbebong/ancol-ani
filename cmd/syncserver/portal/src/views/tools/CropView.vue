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
