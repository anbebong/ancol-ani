<script setup>
import { ref, onMounted } from 'vue';
import { api, errMsg } from '../api.js';
import { session } from '../session.js';
import { confirmDialog } from '@ui/modal.js';
import Icon from '@ui/components/Icon.vue';
import PageHeader from '@ui/components/PageHeader.vue';

const users = ref([]);
const loading = ref(true);
const error = ref('');
const notice = ref('');

async function load() {
    error.value = '';
    try {
        const res = await api('/admin/users');
        users.value = await res.json();
    } catch (err) {
        error.value = errMsg(err);
    } finally {
        loading.value = false;
    }
}
onMounted(load);

async function setQuota(u, value) {
    error.value = '';
    notice.value = '';
    // A cleared field would otherwise be sent as Number('') === 0 and
    // silently drop the user's quota to zero.
    if (String(value).trim() === '' || !Number.isInteger(Number(value))) {
        await load();
        return;
    }
    try {
        await api(`/admin/users/${u.id}/quota`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ quota: Number(value) }),
        });
        notice.value = `Đã đổi quota của ${u.username} thành ${value}`;
        await load();
    } catch (err) {
        error.value = errMsg(err);
        await load();
    }
}

// Shown once after a reset; never stored anywhere else.
const reset = ref(null); // { username, password }
const copied = ref(false);

async function resetPassword(u) {
    const ok = await confirmDialog(
        `Reset mật khẩu của "${u.username}"? Họ sẽ bị đăng xuất khỏi mọi thiết bị và phải đăng nhập bằng mật khẩu tạm.`,
        { confirmText: 'Reset' },
    );
    if (!ok) return;
    error.value = '';
    notice.value = '';
    try {
        const res = await api(`/admin/users/${u.id}/reset-password`, { method: 'POST' });
        const { tempPassword } = await res.json();
        copied.value = false;
        reset.value = { username: u.username, password: tempPassword };
    } catch (err) {
        error.value = errMsg(err);
    }
}

async function copyTemp() {
    try {
        await navigator.clipboard.writeText(reset.value.password);
        copied.value = true;
    } catch (_) {
        copied.value = false;
    }
}

async function remove(u) {
    if (!(await confirmDialog(`Xóa user "${u.username}" và toàn bộ GIF của họ?`, { confirmText: 'Xóa' }))) return;
    error.value = '';
    notice.value = '';
    try {
        await api(`/admin/users/${u.id}`, { method: 'DELETE' });
        notice.value = `Đã xóa user ${u.username}`;
        await load();
    } catch (err) {
        error.value = errMsg(err);
    }
}
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="users" title="Quản trị user" :subtitle="`${users.length} tài khoản trên server`"/>

    <div v-if="error" class="alert alert-error text-sm py-2">{{ error }}</div>
    <div v-else-if="notice" class="alert alert-success text-sm py-2">{{ notice }}</div>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else class="card bg-base-100">
      <div class="overflow-x-auto">
        <table class="table">
          <thead>
            <tr><th>ID</th><th>Username</th><th>Vai trò</th><th>Quota GIF</th><th>Đang có</th><th>Public</th><th></th></tr>
          </thead>
          <tbody>
            <tr v-for="u in users" :key="u.id" class="hover">
              <td class="text-base-content/50">{{ u.id }}</td>
              <td class="font-bold">
                {{ u.username }}
                <span v-if="u.username === session.me?.username" class="badge badge-ghost badge-xs ml-1">bạn</span>
              </td>
              <td>
                <span v-if="u.isAdmin" class="badge badge-warning badge-sm gap-1"><Icon name="crown" :size="12"/> admin</span>
                <span v-else class="text-base-content/50 text-xs">user</span>
              </td>
              <td>
                <input type="number" min="0" max="50" :value="u.gifQuota" class="input input-bordered input-xs w-16"
                  @change="setQuota(u, $event.target.value)"/>
              </td>
              <td>{{ u.gifCount }}</td>
              <td>
                <Icon v-if="u.hasPublic" name="globe" class="text-success"/>
                <Icon v-else name="minus" class="opacity-30"/>
              </td>
              <td class="text-right whitespace-nowrap">
                <button v-if="u.username !== session.me?.username" class="btn btn-warning btn-xs gap-1 mr-2" @click="resetPassword(u)">
                  <Icon name="key" :size="12"/> Reset MK
                </button>
                <button class="btn btn-error btn-xs gap-1" @click="remove(u)"><Icon name="trash-2" :size="12"/> Xóa</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- One-time display of the temporary password after a reset -->
    <div v-if="reset" class="fixed inset-0 bg-black/60 flex items-center justify-center p-4 z-50" @keydown.esc="reset = null">
      <div class="card bg-base-100 w-full max-w-sm">
        <div class="card-body gap-4">
          <h2 class="font-bold flex items-center gap-2"><Icon name="key"/> Mật khẩu tạm cho {{ reset.username }}</h2>
          <div class="flex items-center gap-2">
            <code class="temp-password flex-1">{{ reset.password }}</code>
            <button type="button" class="btn btn-primary btn-sm" @click="copyTemp">{{ copied ? 'Đã copy' : 'Copy' }}</button>
          </div>
          <p class="text-xs text-base-content/60">
            Gửi mật khẩu này cho user. Họ đã bị đăng xuất khỏi mọi thiết bị; sau khi đăng nhập nên tự đổi lại ở trang <b>Tài khoản</b>.
          </p>
          <p class="text-xs text-warning">Mật khẩu chỉ hiện một lần, đóng hộp này là không xem lại được.</p>
          <button type="button" class="btn btn-ghost btn-sm" @click="reset = null">Đóng</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.temp-password {
  padding: 0.5rem 0.75rem;
  font-size: 1.1rem;
  letter-spacing: 0.08em;
  background: #0e0e22;
  border: 2px solid #35335a;
  border-radius: 0.375rem;
  user-select: all;
}
</style>
