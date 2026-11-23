<script setup>
import { computed, watch, onMounted } from 'vue';
import Icon from '@ui/components/Icon.vue';
import AppLogo from '@ui/components/AppLogo.vue';
import { t, lang, toggleLang } from './i18n.js';
import { account, loadMe } from './account.js';
import { appInfo, hasUnseenUpdate, loadVersion } from './appInfo.js';
import { toastState } from './toast.js';

// Plain `location.reload()` in a template gets compiled as `_ctx.location.reload()`
// (Vue prefixes free identifiers that aren't local bindings), and the component
// has no `location` property - so it silently throws on `undefined`. Go through
// `window` explicitly instead.
function reloadPage() { window.location.reload(); }

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
</script>

<template>
  <div class="min-h-screen flex flex-col">
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
          <button type="button" class="btn btn-sm btn-square btn-gradient" :title="t('reloadBtn')" @click="reloadPage">
            <Icon name="refresh-cw"/>
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

    <main class="max-w-5xl w-full mx-auto px-4 py-8 flex-1">
      <RouterView/>
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

    <div class="app-toast" :class="{ 'is-visible': toastState.visible, 'is-success': toastState.kind === 'success', 'is-error': toastState.kind === 'error' }" role="status">
      {{ toastState.message }}
    </div>
  </div>
</template>
