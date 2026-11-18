<script setup>
import { computed, ref, watch, onMounted } from 'vue';
import Icon from '@ui/components/Icon.vue';
import AppLogo from '@ui/components/AppLogo.vue';
import { t, lang, toggleLang } from './i18n.js';
import { account, loadMe } from './account.js';
import { toastState } from './toast.js';

// Same shell as the server portal: sticky header with logo + nav, page
// content below. Library/Explore only show once an account is linked.
const links = computed(() => {
    const base = [{ to: '/', label: t('navWidget'), icon: 'monitor' }];
    if (account.token) {
        base.push(
            { to: '/library', label: t('navLibrary'), icon: 'images' },
            { to: '/gallery', label: t('navGallery'), icon: 'compass' },
        );
    }
    return base;
});

watch(() => account.token, (tok) => { if (tok && !account.me) loadMe().catch(() => {}); }, { immediate: true });

const version = ref('');
onMounted(() => {
    fetch('/version').then((r) => (r.ok ? r.text() : '')).then((v) => { version.value = v.trim(); }).catch(() => {});
});
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <header class="app-header">
      <div class="max-w-5xl mx-auto px-4 h-16 flex items-center gap-4">
        <RouterLink to="/" class="flex items-center gap-2 shrink-0">
          <AppLogo :size="36"/>
          <span class="font-extrabold text-lg hidden sm:inline whitespace-nowrap"><span class="gradient-text">ANCOL-ANI</span></span>
        </RouterLink>

        <nav class="hidden md:flex items-center gap-1 flex-1">
          <RouterLink v-for="l in links" :key="l.to" :to="l.to" class="nav-link" active-class="" exact-active-class="is-active">
            <Icon :name="l.icon"/> {{ l.label }}
          </RouterLink>
        </nav>
        <div class="flex-1 md:hidden"></div>

        <div class="flex items-center gap-2 shrink-0">
          <button type="button" class="btn btn-ghost btn-sm gap-1" :title="t('langToggleTitle')" @click="toggleLang">
            <Icon name="languages"/> {{ lang === 'vi' ? 'EN' : 'VI' }}
          </button>
          <RouterLink v-if="account.token" to="/account" class="user-chip" active-class="is-active">
            <Icon name="user" :size="14"/>
            <span class="max-w-[8rem] truncate">{{ account.username || '?' }}</span>
          </RouterLink>
          <RouterLink v-else to="/account" class="btn btn-primary btn-sm gap-1">
            <Icon name="link"/> {{ t('navLink') }}
          </RouterLink>
        </div>
      </div>

      <nav class="md:hidden flex gap-1 overflow-x-auto px-4 pb-2">
        <RouterLink v-for="l in links" :key="l.to" :to="l.to" class="nav-link shrink-0" active-class="" exact-active-class="is-active">
          <Icon :name="l.icon"/> {{ l.label }}
        </RouterLink>
      </nav>
    </header>

    <main class="max-w-5xl w-full mx-auto px-4 py-8 flex-1">
      <RouterView/>
    </main>

    <footer class="flex items-center justify-center gap-1.5 text-xs text-base-content/50 pb-6">
      <span>anbebong</span>
      <span v-if="version">· v{{ version }}</span>
    </footer>

    <div class="app-toast" :class="{ 'is-visible': toastState.visible, 'is-success': toastState.kind === 'success', 'is-error': toastState.kind === 'error' }" role="status">
      {{ toastState.message }}
    </div>
  </div>
</template>
