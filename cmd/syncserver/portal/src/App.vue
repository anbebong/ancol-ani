<script setup>
import { computed, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { session, loadMe, signOut } from './session.js';
import Icon from '@ui/components/Icon.vue';
import AppLogo from '@ui/components/AppLogo.vue';

const route = useRoute();
const router = useRouter();

const showShell = computed(() => route.name && route.name !== 'login' && route.name !== 'sso');

// Fetch the profile (username / admin flag / quota) whenever a session
// starts, so the navbar can show it and the admin links appear.
watch(() => session.token, (t) => { if (t) loadMe().catch(() => {}); }, { immediate: true });

const links = computed(() => {
    const base = [
        { to: '/library', label: 'Thư viện', icon: 'images' },
        { to: '/history', label: 'Lịch sử', icon: 'history' },
        { to: '/gallery', label: 'Khám phá', icon: 'compass' },
    ];
    if (session.me?.isAdmin) {
        base.push(
            { to: '/admin/users', label: 'User', icon: 'users', admin: true },
            { to: '/admin/public', label: 'GIF public', icon: 'shield', admin: true },
        );
    }
    return base;
});

async function logout() {
    await signOut();
    router.push({ name: 'login' });
}
</script>

<template>
  <div v-if="showShell" class="min-h-screen flex flex-col">
    <header class="app-header">
      <div class="max-w-5xl mx-auto px-4 h-16 flex items-center gap-4">
        <RouterLink to="/library" class="flex items-center gap-2 shrink-0">
          <AppLogo :size="36"/>
          <span class="font-extrabold text-lg hidden sm:inline md:hidden lg:inline whitespace-nowrap"><span class="gradient-text">Ancol-ani</span></span>
        </RouterLink>

        <nav class="hidden md:flex items-center gap-1 flex-1">
          <template v-for="(l, i) in links" :key="l.to">
            <span v-if="l.admin && !links[i - 1]?.admin" class="nav-sep">Quản trị</span>
            <RouterLink :to="l.to" class="nav-link" active-class="is-active">
              <Icon :name="l.icon"/> {{ l.label }}
            </RouterLink>
          </template>
        </nav>
        <div class="flex-1 md:hidden"></div>

        <div class="flex items-center gap-2 shrink-0">
          <RouterLink v-if="session.me" to="/account" class="user-chip" active-class="is-active" title="Tài khoản · đổi mật khẩu">
            <Icon :name="session.me.isAdmin ? 'crown' : 'user'" :size="14"/>
            <span class="max-w-[8rem] truncate">{{ session.me.username }}</span>
          </RouterLink>
          <button type="button" class="btn btn-ghost btn-sm gap-1" title="Đăng xuất" @click="logout">
            <Icon name="log-out"/> <span class="hidden xl:inline">Đăng xuất</span>
          </button>
        </div>
      </div>

      <!-- Mobile: nav as a horizontally scrollable strip under the bar -->
      <nav class="md:hidden flex gap-1 overflow-x-auto px-4 pb-2">
        <RouterLink v-for="l in links" :key="l.to" :to="l.to" class="nav-link shrink-0" active-class="is-active">
          <Icon :name="l.icon"/> {{ l.label }}
        </RouterLink>
      </nav>
    </header>

    <main class="max-w-5xl w-full mx-auto px-4 py-8 flex-1">
      <RouterView/>
    </main>
  </div>

  <RouterView v-else/>
</template>
