import { createRouter, createWebHashHistory } from 'vue-router';
import { session, loadMe } from './session.js';

// Hash history (/#/library) so the Go server's plain static file server
// needs no SPA fallback, and so /public/... and /me/... stay free for the
// JSON API.
const routes = [
    { path: '/', redirect: '/library' },
    { path: '/login', name: 'login', component: () => import('./views/LoginView.vue'), meta: { guestOnly: true, title: 'Đăng nhập' } },
    { path: '/sso', name: 'sso', component: () => import('./views/SsoView.vue'), meta: { title: 'Đăng nhập' } },
    { path: '/library', name: 'library', component: () => import('./views/LibraryView.vue'), meta: { auth: true, title: 'Thư viện của tôi' } },
    { path: '/history', name: 'history', component: () => import('./views/HistoryView.vue'), meta: { auth: true, title: 'Lịch sử chia sẻ' } },
    { path: '/account', name: 'account', component: () => import('./views/AccountView.vue'), meta: { auth: true, title: 'Tài khoản' } },
    { path: '/gallery', name: 'gallery', component: () => import('./views/GalleryView.vue'), meta: { auth: true, title: 'Khám phá' } },
    { path: '/tools/resize', name: 'tools-resize', component: () => import('./views/tools/ResizeView.vue'), meta: { title: 'Resize GIF' } },
    { path: '/tools/convert', name: 'tools-convert', component: () => import('./views/tools/ConvertView.vue'), meta: { title: 'Chuyển đổi thành GIF' } },
    { path: '/admin/users', name: 'admin-users', component: () => import('./views/AdminUsersView.vue'), meta: { auth: true, admin: true, title: 'Quản trị user' } },
    { path: '/admin/public', name: 'admin-public', component: () => import('./views/AdminPublicView.vue'), meta: { auth: true, admin: true, title: 'Quản trị GIF public' } },
    { path: '/:pathMatch(.*)*', redirect: '/library' },
];

export const router = createRouter({
    history: createWebHashHistory(),
    routes,
});

router.beforeEach(async (to) => {
    if (to.meta.guestOnly && session.token) return { name: 'library' };
    if (to.meta.auth && !session.token) {
        return { name: 'login', query: to.fullPath !== '/library' ? { redirect: to.fullPath } : {} };
    }
    if (to.meta.admin) {
        try {
            const me = await loadMe();
            if (!me?.isAdmin) return { name: 'library' };
        } catch (_) {
            return session.token ? { name: 'library' } : { name: 'login' };
        }
    }
    return true;
});

router.afterEach((to) => {
    document.title = to.meta.title ? `${to.meta.title} · Ancol-Ani` : 'Ancol-Ani';
});
