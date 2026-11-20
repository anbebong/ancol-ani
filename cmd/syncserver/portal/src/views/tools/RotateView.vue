<script setup>
import { ref } from 'vue';
import PageHeader from '@ui/components/PageHeader.vue';
import ToolPage from '../../components/ToolPage.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

const angle = ref(90);
const flip = ref('none');
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
        const r = await postTool(`/api/tools/rotate?angle=${angle.value}&flip=${flip.value}`, fd);
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
    <PageHeader icon="rotate-ccw" title="Rotate GIF" subtitle="Xoay hoặc lật GIF."/>
    <ToolPage :result="result" :loading="loading" :error="error" run-label="Rotate" @run="onRun">
      <template #params>
        <div class="flex gap-3">
          <label class="form-control flex-1">
            <span class="label-text">Góc xoay</span>
            <select v-model.number="angle" class="select select-bordered">
              <option :value="90">90°</option>
              <option :value="180">180°</option>
              <option :value="270">270°</option>
            </select>
          </label>
          <label class="form-control flex-1">
            <span class="label-text">Lật</span>
            <select v-model="flip" class="select select-bordered">
              <option value="none">Không lật</option>
              <option value="h">Lật ngang</option>
              <option value="v">Lật dọc</option>
            </select>
          </label>
        </div>
      </template>
    </ToolPage>
  </div>
</template>
