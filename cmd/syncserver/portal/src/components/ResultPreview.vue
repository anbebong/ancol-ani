<script setup>
import { ref, watch, onUnmounted } from 'vue';
import Icon from '@ui/components/Icon.vue';
import { setChainedResult } from '../toolChain.js';

const props = defineProps({
    result: { type: Object, default: null },
    loading: { type: Boolean, default: false },
    error: { type: String, default: '' },
});

const url = ref('');
function revoke() { if (url.value) URL.revokeObjectURL(url.value); url.value = ''; }

watch(() => props.result, (r) => {
    revoke();
    if (r) url.value = URL.createObjectURL(r.blob);
}, { immediate: true });

onUnmounted(revoke);

// Only a GIF result can feed back into another tool as its input file; a
// zip (Split's output) has nowhere to go but download.
function continueWithResult() {
    if (props.result && props.result.blob.type === 'image/gif') {
        setChainedResult(props.result.blob, props.result.filename);
    }
}
</script>

<template>
  <div class="flex flex-col items-center gap-3 py-4">
    <span v-if="loading" class="loading loading-spinner loading-lg text-primary"/>
    <div v-else-if="error" class="alert alert-error text-sm py-2 w-full">{{ error }}</div>
    <template v-else-if="result">
      <img v-if="result.blob.type === 'image/gif'" :src="url" alt="" class="max-w-full max-h-96 object-contain rounded"/>
      <a :href="url" :download="result.filename" class="btn btn-primary btn-sm gap-2">
        <Icon name="download"/> Tải xuống {{ result.filename }}
      </a>
      <RouterLink v-if="result.blob.type === 'image/gif'" to="/tools" class="btn btn-ghost border-2 border-neutral btn-sm gap-2" @click="continueWithResult">
        <Icon name="wand-2" :size="16"/> Tiếp tục xử lý GIF này
      </RouterLink>
      <RouterLink v-else to="/tools" class="btn btn-ghost border-2 border-neutral btn-sm gap-2">
        <Icon name="wand-2" :size="16"/> Dùng công cụ khác
      </RouterLink>
    </template>
  </div>
</template>
