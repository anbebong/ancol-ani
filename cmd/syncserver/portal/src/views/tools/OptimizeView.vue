<script setup>
import { ref } from 'vue';
import PageHeader from '@ui/components/PageHeader.vue';
import ToolBackLink from '../../components/ToolBackLink.vue';
import ToolPage from '../../components/ToolPage.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

const level = ref(2);
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
        const r = await postTool(`/api/tools/optimize?level=${level.value}`, fd);
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
    <PageHeader icon="sparkles" title="Optimize GIF" subtitle="Giảm dung lượng GIF."/>
    <ToolPage :result="result" :loading="loading" :error="error" run-label="Optimize" @run="onRun">
      <template #params>
        <label class="form-control">
          <span class="label-text">Mức giảm</span>
          <select v-model.number="level" class="select select-bordered">
            <option :value="1">1 - Nhẹ</option>
            <option :value="2">2 - Vừa</option>
            <option :value="3">3 - Mạnh</option>
          </select>
        </label>
      </template>
    </ToolPage>
  </div>
</template>
