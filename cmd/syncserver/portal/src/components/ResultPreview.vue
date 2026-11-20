<script setup>
import { ref, watch, onUnmounted } from 'vue';
import Icon from '@ui/components/Icon.vue';

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
    </template>
  </div>
</template>
