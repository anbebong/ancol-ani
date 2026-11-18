import { createRouter, createWebHashHistory } from 'vue-router';
import { account } from './account.js';
import { t } from './i18n.js';
import { toast } from './toast.js';

// Hash history (/#/library) like the server portal, so the Go app's plain
// static file server needs no SPA fallback and /gif, /text, /version stay
// free for the app's own endpoints.
const routes = [
    { path: '/', name: 'widget', component: () => import('./views/WidgetView.vue'), meta: { title: 'navWidget' } },
    { path: '/library', name: 'library', component: () => import('./views/LibraryView.vue'), meta: { needsLink: true, title: 'navLibrary' } },
    { path: '/gallery', name: 'gallery', component: () => import('./views/GalleryView.vue'), meta: { needsLink: true, title: 'navGallery' } },
    { path: '/account', name: 'account', component: () => import('./views/AccountView.vue'), meta: { title: 'accountTitle' } },
    { path: '/:pathMatch(.*)*', redirect: '/' },
];

// A factory, not a module-level router: createWebHashHistory() captures
// the current URL (incl. ?query) as its base when called, so it must run
// only after main.js has consumed and stripped the portal's ?token=...
// callback - otherwise the router starts on the stale route and the
// token stays in every URL.
export function createAppRouter() {
    const router = createRouter({
        history: createWebHashHistory(),
        routes,
    });

    router.beforeEach((to) => {
        if (to.meta.needsLink && !account.token) {
            toast(t('needLink'));
            return { name: 'account' };
        }
        return true;
    });

    router.afterEach((to) => {
        document.title = to.meta.title ? `${t(to.meta.title)} · ANCOL-ANI` : 'ANCOL-ANI';
    });

    return router;
}
