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
