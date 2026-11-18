<script setup>
import { ref, onMounted } from 'vue';
import { api, errMsg, fmtDate } from '../api.js';
import Icon from '../components/Icon.vue';
import PageHeader from '../components/PageHeader.vue';

const entries = ref([]);
const loading = ref(true);
const error = ref('');

onMounted(async () => {
    try {
        const res = await api('/me/gif/history');
        entries.value = await res.json();
    } catch (err) {
        error.value = errMsg(err);
    } finally {
        loading.value = false;
    }
});
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="history" title="Lịch sử chia sẻ" subtitle="Mọi lần bạn bật/tắt chia sẻ công khai GIF."/>

    <div v-if="error" class="alert alert-error text-sm py-2">{{ error }}</div>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else-if="!entries.length" class="empty-state">
      <Icon name="history" :size="40" class="opacity-40"/>
      <p>Chưa có lượt chia sẻ nào.</p>
    </div>

    <div v-else class="card bg-base-100">
      <div class="overflow-x-auto">
        <table class="table">
          <thead><tr><th>Link</th><th>Bắt đầu</th><th>Kết thúc</th><th>Trạng thái</th></tr></thead>
          <tbody>
            <tr v-for="h in entries" :key="h.slug + h.sharedAt">
              <td class="font-mono text-xs break-all">/public/{{ h.slug }}</td>
              <td class="text-xs whitespace-nowrap">{{ fmtDate(h.sharedAt) }}</td>
              <td class="text-xs whitespace-nowrap">{{ h.unsharedAt ? fmtDate(h.unsharedAt) : '—' }}</td>
              <td>
                <span v-if="!h.unsharedAt" class="badge badge-success badge-sm whitespace-nowrap">đang chia sẻ</span>
                <span v-else class="badge badge-ghost badge-sm whitespace-nowrap">đã gỡ</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
