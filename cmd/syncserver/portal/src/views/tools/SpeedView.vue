<script setup>
import { ref } from 'vue';
import PageHeader from '@ui/components/PageHeader.vue';
import ToolBackLink from '../../components/ToolBackLink.vue';
import ToolPage from '../../components/ToolPage.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

const factor = ref(1);
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
        const r = await postTool(`/api/tools/speed?factor=${factor.value}`, fd);
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
    <PageHeader icon="wand-2" title="Speed GIF" subtitle="Tăng/giảm tốc độ phát GIF."/>
    <ToolPage :result="result" :loading="loading" :error="error" run-label="Áp dụng" @run="onRun">
      <template #params>
        <label class="form-control">
          <span class="label-text">Hệ số tốc độ (0.5 = chậm 2 lần, 2 = nhanh 2 lần)</span>
          <input v-model.number="factor" type="number" step="0.1" min="0.1" max="10" class="input input-bordered"/>
        </label>
      </template>
    </ToolPage>
  </div>
</template>
