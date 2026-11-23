<script setup>
import { ref, reactive, onMounted, watch, nextTick } from 'vue';
import { useRoute } from 'vue-router';
import Icon from '@ui/components/Icon.vue';
import PageHeader from '@ui/components/PageHeader.vue';
import { confirmDialog } from '@ui/modal.js';
import { t } from '../i18n.js';
import { appInfo, loadChangelog, markVersionSeen } from '../appInfo.js';
import { toast, toastError } from '../toast.js';

const route = useRoute();
const releases = ref([]);
const loading = ref(true);
const failed = ref(false);
const checkingUpdate = ref(false);

// Manual only, per request - no background polling. Hits the widget's
// own local /check-update (app.go), which in turn calls the public sync
// server's /latest-version (cmd/syncserver/update.go) - this page never
// talks to that server directly.
async function checkForUpdate() {
    checkingUpdate.value = true;
    try {
        const res = await fetch('/check-update');
        const data = await res.json();
        if (data.error) {
            toast('Không kiểm tra được cập nhật: ' + data.error, 'error');
            return;
        }
        if (!data.hasUpdate) {
            toast('Đang dùng bản mới nhất.', 'success');
            return;
        }

        const ok = await confirmDialog(
            `Có bản mới v${data.version}. Cập nhật ngay? App sẽ tự khởi động lại.`,
            { confirmText: 'Cập nhật', cancelText: 'Để sau' },
        );
        if (!ok) return;

        const applyRes = await fetch('/apply-update', { method: 'POST' });
        if (!applyRes.ok) {
            toast('Cập nhật lỗi: ' + (await applyRes.text()), 'error');
            return;
        }
        toast('Đang cập nhật, app sẽ tự khởi động lại...', 'success');
    } catch (err) {
        toastError(err);
    } finally {
        checkingUpdate.value = false;
    }
}

// Footer "Chính sách" / version links (App.vue) both route here with
// ?section=policy|version, since hash-history routing already uses '#'
// for the route itself - a plain '#anchor' would collide with it.
async function scrollToSection(section) {
    if (!section) return;
    await nextTick();
    document.getElementById(`section-${section}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

// Watches the query too, not just onMounted: clicking the other footer
// link while already on this page reuses the same component instance
// (same route, only the query changes), so onMounted alone would miss it.
watch(() => route.query.section, (section) => scrollToSection(section));

onMounted(async () => {
    try {
        releases.value = await loadChangelog();
    } catch (_) {
        failed.value = true;
    } finally {
        loading.value = false;
        markVersionSeen();
    }

    scrollToSection(route.query.section);
});

// Version history is collapsed by default - each release shows only
// "vX.Y.Z · date"; clicking a row toggles its details.
const expanded = reactive({});
function toggleRelease(version) { expanded[version] = !expanded[version]; }
function itemCount(r) { return r.groups.reduce((n, g) => n + g.items.length, 0); }

// Color per group title (both languages / common wordings).
function groupClass(title) {
    const s = title.toLowerCase();
    if (/mới|new|added/.test(s)) return 'badge-success';
    if (/sửa|fix/.test(s)) return 'badge-error';
    if (/thay đổi|change|improv/.test(s)) return 'badge-warning';
    return 'badge-neutral';
}

// Privacy notice kept in Vietnamese only (legal text for a Vietnam-based
// personal app, per Nghị định 13/2023/NĐ-CP) regardless of the UI
// language toggle - same as how a site's Điều khoản/Privacy page usually
// isn't swapped per visitor language.
const policySections = [
    {
        title: 'Phạm vi dữ liệu thực sự xử lý / không thu thập',
        items: [
            'Không thu thập hồ sơ cá nhân, lịch sử duyệt web/hoạt động máy, danh bạ hay bất kỳ tệp tin nào khác ngoài ảnh GIF bạn tự chọn.',
            'Chỉ dùng tên đăng nhập, mật khẩu ở mức tối thiểu để xác thực khi bạn chủ động liên kết tài khoản. Không liên kết thì mọi thứ chỉ nằm trên máy bạn, không gửi đi đâu.',
        ],
    },
    {
        title: 'Mục đích sử dụng & không chia sẻ/mua bán dữ liệu',
        items: [
            'Thông tin đăng nhập chỉ dùng để đồng bộ/chia sẻ GIF theo ý bạn, không dùng cho mục đích khác.',
            'Không chuyển giao, không tiết lộ, không mua bán dữ liệu cho bên thứ ba nào.',
        ],
    },
    {
        title: 'Lưu trữ, bảo mật & xóa/hủy dữ liệu',
        items: [
            'Mật khẩu được mã hóa an toàn, không lưu dạng chữ thường. Phiên đăng nhập chỉ có hiệu lực ngắn, không giữ lâu dài.',
            'Hủy liên kết tài khoản bất cứ lúc nào ở trang Tài khoản. Muốn xóa toàn bộ dữ liệu, liên hệ địa chỉ bên dưới.',
        ],
    },
    {
        title: 'Trách nhiệm & giới hạn sử dụng hợp pháp',
        items: [
            'Không dùng ứng dụng/tài khoản vào mục đích vi phạm pháp luật, không can thiệp trái phép vào máy chủ.',
            'Người dùng tự bảo quản thông tin đăng nhập của chính mình.',
        ],
    },
    {
        title: 'Thông tin liên hệ',
        items: [
            'Có sự cố bảo mật, lộ lọt dữ liệu, hoặc cần hỗ trợ/yêu cầu xóa dữ liệu, liên hệ địa chỉ bên dưới.',
        ],
    },
];
const policyContact = 'angokodau@gmail.com';
</script>

<template>
  <div class="flex flex-col gap-6">
    <PageHeader icon="sparkles" :title="t('changelogTitle')"
      :subtitle="appInfo.version ? t('changelogSubtitle', { version: appInfo.version }) : ''">
      <button type="button" class="btn btn-primary gap-2" :disabled="checkingUpdate" @click="checkForUpdate">
        <span v-if="checkingUpdate" class="loading loading-spinner loading-sm"></span>
        <Icon v-else name="download"/>
        Kiểm tra cập nhật
      </button>
    </PageHeader>

    <div id="section-policy" class="card bg-base-100 max-w-3xl">
      <div class="card-body gap-4">
        <h2 class="text-xl font-extrabold flex items-center gap-2">
          <Icon name="shield"/> Chính sách bảo mật & quyền riêng tư
        </h2>
        <p class="text-xs text-base-content/60 -mt-1">
          Ancol-Ani là app cá nhân, tuân thủ Nghị định 13/2023/NĐ-CP về bảo vệ dữ liệu cá nhân.
        </p>

        <div v-for="(section, si) in policySections" :key="si" class="flex flex-col gap-2">
          <span class="badge badge-sm badge-neutral self-start">{{ section.title }}</span>
          <ul class="changelog-list">
            <li v-for="(item, ii) in section.items" :key="ii">{{ item }}</li>
          </ul>
        </div>

        <p class="text-xs text-base-content/60">
          Liên hệ về dữ liệu cá nhân: <a :href="'mailto:' + policyContact" class="link link-primary">{{ policyContact }}</a>
        </p>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-12"><span class="loading loading-spinner loading-lg text-primary"></span></div>

    <div v-else-if="failed || !releases.length" class="empty-state">
      <Icon name="sparkles" :size="40" class="opacity-40"/>
      <p>{{ t('changelogError') }}</p>
    </div>

    <div v-else id="section-version" class="flex flex-col gap-3 max-w-3xl">
      <div v-for="r in releases" :key="r.version" class="card bg-base-100" :class="{ 'release-current': r.version === appInfo.version }">
        <button type="button" class="release-row" :aria-expanded="!!expanded[r.version]"
          :aria-controls="'release-' + r.version" :title="t('changelogToggle')" @click="toggleRelease(r.version)">
          <span class="text-lg font-extrabold flex items-center gap-2">
            <Icon name="tag"/> v{{ r.version }}
          </span>
          <span v-if="r.version === appInfo.version" class="badge badge-primary badge-sm">{{ t('changelogCurrent') }}</span>
          <span class="text-xs text-base-content/40 hidden sm:inline">{{ t('changelogItems', { n: itemCount(r) }) }}</span>
          <span class="ml-auto flex items-center gap-3">
            <span v-if="r.date" class="text-sm text-base-content/50">{{ r.date }}</span>
            <Icon name="chevron-down" class="release-chevron" :class="{ 'is-open': expanded[r.version] }"/>
          </span>
        </button>

        <div v-if="expanded[r.version]" :id="'release-' + r.version" class="card-body gap-4 pt-1">
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
.release-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
  width: 100%;
  padding: 0.9rem 1.25rem;
  text-align: left;
  border-radius: inherit;
  transition: background .15s ease;
}
.release-row:hover { background: rgba(92, 62, 235, 0.08); }
.release-row:focus-visible { outline: 2px solid #5c3eeb; outline-offset: -2px; }
.release-chevron { color: #8d89a8; transition: transform .2s ease; }
.release-chevron.is-open { transform: rotate(180deg); }
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
