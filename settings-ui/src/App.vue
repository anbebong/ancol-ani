<script setup>
import { computed, ref, watch, onMounted } from 'vue';
import Icon from '@theme/components/Icon.vue';
import AppLogo from '@theme/components/AppLogo.vue';
import { t, lang, toggleLang } from './i18n.js';
import { account, loadMe } from './account.js';
import { appInfo, hasUnseenUpdate, loadVersion } from './appInfo.js';
import { toastState, toast } from './toast.js';

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
    base.push({ to: '/changelog', label: t('navChangelog'), icon: 'sparkles', dot: true });
    return base;
});

watch(() => account.token, (tok) => { if (tok && !account.me) loadMe().catch(() => {}); }, { immediate: true });

onMounted(loadVersion);

// "Tải lại" in the header, on every page: remounts the current page (so it
// refetches its own data - GIF lists, changelog, widget preview...) and
// refreshes the shared version/account info. Not a full location.reload():
// keeps login/language state, and also works in the standalone settings
// window, which has no browser refresh button.
const reloadKey = ref(0);
const reloading = ref(false);
async function reloadPage() {
    if (reloading.value) return;
    reloading.value = true;
    reloadKey.value++;
    try {
        await Promise.all([
            loadVersion(),
            account.token ? loadMe().catch(() => {}) : Promise.resolve(),
        ]);
        toast(t('reloadDone'), 'success');
    } finally {
        // keep the spin visible briefly so the click is clearly acknowledged
        setTimeout(() => { reloading.value = false; }, 400);
    }
}
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="max-w-5xl mx-auto px-4 h-16 flex items-center gap-4">
        <RouterLink to="/" class="flex items-end gap-0 shrink-0">
          <AppLogo :size="36"/>
          <span class="font-extrabold text-lg hidden sm:inline whitespace-nowrap leading-none"><span class="gradient-text">ncol-Ani</span></span>
        </RouterLink>

        <nav class="hidden md:flex items-center gap-1 flex-1">
          <RouterLink v-for="l in links" :key="l.to" :to="l.to" class="nav-link" active-class="" exact-active-class="is-active">
            <Icon :name="l.icon"/> {{ l.label }}
            <span v-if="l.dot && hasUnseenUpdate" class="nav-dot" :title="t('changelogNewDot')"></span>
          </RouterLink>
        </nav>
        <div class="flex-1 md:hidden"></div>

        <div class="flex items-center gap-2 shrink-0">
          <button type="button" class="btn btn-gradient btn-sm gap-1" :title="t('reloadTitle')" :disabled="reloading" @click="reloadPage">
            <Icon name="refresh-cw" :class="{ 'animate-spin': reloading }"/>
            <span class="hidden sm:inline">{{ t('reloadLabel') }}</span>
          </button>
          <button type="button" class="btn btn-sm gap-1 btn-gradient" :title="t('langToggleTitle')" @click="toggleLang">
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
          <span v-if="l.dot && hasUnseenUpdate" class="nav-dot"></span>
        </RouterLink>
      </nav>
    </header>

    <div class="app-scroll">
      <main class="max-w-5xl w-full mx-auto px-4 py-8 flex-1">
        <RouterView v-slot="{ Component, route }">
          <component :is="Component" :key="route.path + ':' + reloadKey"/>
        </RouterView>
      </main>

      <footer class="flex items-center justify-center gap-1.5 text-xs text-base-content/50 pb-6">
        <span class="inline-flex items-center gap-1">
          <Icon name="github" :size="14"/> anbebong
        </span>
        <RouterLink v-if="appInfo.version" :to="{ path: '/changelog', query: { section: 'version' } }" class="inline-flex items-center gap-1 hover:text-base-content" :title="t('navChangelog')">
          · v{{ appInfo.version }}
          <span v-if="hasUnseenUpdate" class="nav-dot"></span>
        </RouterLink>
        <RouterLink :to="{ path: '/changelog', query: { section: 'policy' } }" class="inline-flex items-center gap-1 hover:text-base-content" :title="t('footerPolicy')">
          · {{ t('footerPolicy') }}
        </RouterLink>
      </footer>
    </div>

    <div class="app-toast" :class="{ 'is-visible': toastState.visible, 'is-success': toastState.kind === 'success', 'is-error': toastState.kind === 'error' }" role="status">
      {{ toastState.message }}
    </div>
  </div>
</template>
