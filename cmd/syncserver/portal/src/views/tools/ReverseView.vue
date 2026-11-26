<script setup>
import { ref } from 'vue';
import PageHeader from '@theme/components/PageHeader.vue';
import ToolBackLink from '../../components/ToolBackLink.vue';
import ToolPage from '../../components/ToolPage.vue';
import { postTool } from '../../toolsApi.js';
import { errMsg } from '../../api.js';

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
        const r = await postTool('/api/tools/reverse', fd);
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
    <PageHeader icon="history" title="Reverse GIF" subtitle="Đảo ngược thứ tự khung hình."/>
    <ToolPage :result="result" :loading="loading" :error="error" run-label="Reverse" @run="onRun"/>
  </div>
</template>
