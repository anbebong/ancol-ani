<script setup>
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import Icon from '@ui/components/Icon.vue';
import PageHeader from '@ui/components/PageHeader.vue';
import { confirmDialog } from '@ui/modal.js';
import { t } from '../i18n.js';
import { account, loadMe, startLink, unlink } from '../account.js';
import { toast } from '../toast.js';

const router = useRouter();
const server = ref(account.server);

onMounted(() => { if (account.token) loadMe().catch(() => {}); });

function link() {
    startLink(server.value);
}

async function doUnlink() {
    if (!(await confirmDialog(t('confirmUnlink'), { confirmText: t('unlinkBtn'), cancelText: t('modalCancel') }))) return;
    unlink();
    toast(t('toastUnlinked'), 'success');
    router.push('/account');
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="user" :title="t('accountTitle')"
      :subtitle="account.token ? t('accountSubtitleLinked', { username: account.username }) : t('accountSubtitleUnlinked')"/>

    <!-- Not linked: server address + link -->
    <div v-if="!account.token" class="card bg-base-100 max-w-lg">
      <form class="card-body gap-4" @submit.prevent="link">
        <h2 class="font-bold flex items-center gap-2"><Icon name="link"/> {{ t('linkCardTitle') }}</h2>
        <p class="text-sm text-base-content/60 -mt-1">{{ t('linkCardHint') }}</p>
        <label class="flex flex-col gap-1.5">
          <span class="field-label">{{ t('serverLabel') }}</span>
          <span class="input input-bordered auth-input flex items-center gap-2">
            <Icon name="server" class="text-base-content/50"/>
            <input v-model="server" type="text" :placeholder="t('serverPlaceholder')" class="grow bg-transparent min-w-0"/>
          </span>
        </label>
        <button type="submit" class="btn btn-primary auth-submit gap-2"><Icon name="link"/> {{ t('linkBtn') }}</button>
      </form>
    </div>

    <!-- Linked: who/where + unlink -->
    <div v-else class="card bg-base-100 max-w-lg">
      <div class="card-body gap-4">
        <div class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-3 items-center">
          <span class="field-label">{{ t('linkedAs') }}</span>
          <span class="font-bold flex items-center gap-1.5"><Icon name="user"/> {{ account.username }}</span>
          <span class="field-label">{{ t('serverLabel') }}</span>
          <span class="font-mono text-sm break-all">{{ account.server }}</span>
          <span class="field-label">{{ t('quotaLabel') }}</span>
          <span>{{ account.me?.gifQuota ?? '–' }} GIF</span>
        </div>

        <p class="text-xs text-base-content/60">{{ t('changePasswordHint') }}</p>

        <div class="flex gap-2 flex-wrap">
          <a :href="account.server + '/#/account'" target="_blank" rel="noopener" class="btn btn-primary btn-sm gap-1">
            <Icon name="external-link"/> {{ t('openPortal') }}
          </a>
          <button type="button" class="btn btn-error btn-sm gap-1" @click="doUnlink"><Icon name="unlink"/> {{ t('unlinkBtn') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>
