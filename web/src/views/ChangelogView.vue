<script setup>
import { ref, onMounted } from 'vue';
import Icon from '@ui/components/Icon.vue';
import PageHeader from '@ui/components/PageHeader.vue';
import { t } from '../i18n.js';
import { appInfo, loadChangelog, markVersionSeen } from '../appInfo.js';

const releases = ref([]);
const loading = ref(true);
const failed = ref(false);

onMounted(async () => {
    try {
        releases.value = await loadChangelog();
    } catch (_) {
        failed.value = true;
    } finally {
        loading.value = false;
        markVersionSeen();
    }
});

// Color per group title (both languages / common wordings).
function groupClass(title) {
    const s = title.toLowerCase();
    if (/mới|new|added/.test(s)) return 'badge-success';
    if (/sửa|fix/.test(s)) return 'badge-error';
    if (/thay đổi|change|improv/.test(s)) return 'badge-warning';
    return 'badge-neutral';
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="sparkles" :title="t('changelogTitle')"
      :subtitle="appInfo.version ? t('changelogSubtitle', { version: appInfo.version }) : ''"/>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else-if="failed || !releases.length" class="empty-state">
      <Icon name="sparkles" :size="40" class="opacity-40"/>
      <p>{{ t('changelogError') }}</p>
    </div>

    <div v-else class="flex flex-col gap-5 max-w-3xl">
      <div v-for="r in releases" :key="r.version" class="card bg-base-100" :class="{ 'release-current': r.version === appInfo.version }">
        <div class="card-body gap-4">
          <div class="flex items-center gap-3 flex-wrap">
            <h2 class="text-xl font-extrabold flex items-center gap-2">
              <Icon name="tag"/> v{{ r.version }}
            </h2>
            <span v-if="r.version === appInfo.version" class="badge badge-primary">{{ t('changelogCurrent') }}</span>
            <span v-if="r.date" class="text-sm text-base-content/50 ml-auto">{{ r.date }}</span>
          </div>

          <div v-for="(g, gi) in r.groups" :key="gi" class="flex flex-col gap-2">
            <span v-if="g.title" class="badge badge-sm self-start" :class="groupClass(g.title)">{{ g.title }}</span>
            <ul class="changelog-list">
              <li v-for="(item, ii) in g.items" :key="ii">
                <template v-for="(tok, ti) in item" :key="ti">
                  <code v-if="tok.type === 'code'">{{ tok.value }}</code>
                  <b v-else-if="tok.type === 'bold'">{{ tok.value }}</b>
                  <template v-else>{{ tok.value }}</template>
                </template>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.release-current { border-color: #5c3eeb; }
.changelog-list {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  padding-left: 1.1rem;
  list-style: square;
  font-size: 0.9rem;
  line-height: 1.5;
}
.changelog-list li::marker { color: #5c3eeb; }
.changelog-list code {
  padding: 0.05rem 0.35rem;
  font-size: 0.8rem;
  background: #0e0e22;
  border: 1px solid #35335a;
  border-radius: 0.25rem;
}
</style>
