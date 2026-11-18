export const translations = {
    vi: {
        subtitle: 'Điều khiển hoạt ảnh và chữ hiển thị trên màn hình',
        textLabel: 'Chữ hiển thị',
        textPlaceholder: 'Gõ gì vào đây...',
        textSubmit: 'Hiển thị',
        gifLabel: 'Ảnh GIF',
        dropzoneDefault: 'Chọn hoặc kéo thả file GIF',
        dropzoneMax: 'Tối đa 3MB',
        uploadSubmit: 'Tải lên',
        resetSubmit: 'Đặt lại ảnh mặc định',
        toastTextSent: 'Đã hiển thị',
        toastGifChanged: 'Đã đổi ảnh GIF',
        toastGifReset: 'Đã đặt lại ảnh mặc định',
        syncLabel: 'Đồng bộ tài khoản',
        syncServerPlaceholder: 'Địa chỉ server (vd: http://localhost:9090)',
        syncLinkBtn: 'Liên kết tài khoản',
        syncUseThis: 'Dùng ảnh này',
        syncPersonalLabel: 'Thư viện cá nhân',
        syncPublicLabel: 'Thư viện public',
        syncLibraryEmpty: 'Tài khoản chưa có GIF nào trong thư viện',
        syncPublicEmpty: 'Chưa có GIF public nào',
        syncUnlinkBtn: 'Hủy liên kết',
        syncLinkedAs: 'Đã liên kết:',
        syncNotLinked: 'Chưa liên kết tài khoản nào',
        toastSyncLinked: 'Đã liên kết tài khoản',
        toastSyncPulled: 'Đã đồng bộ ảnh đã chọn',
        toastSyncUnlinked: 'Đã hủy liên kết',
    },
    en: {
        subtitle: 'Control the on-screen animation and text',
        textLabel: 'Display text',
        textPlaceholder: 'Type something...',
        textSubmit: 'Show',
        gifLabel: 'GIF image',
        dropzoneDefault: 'Choose or drop a GIF file',
        dropzoneMax: 'Up to 3MB',
        uploadSubmit: 'Upload',
        resetSubmit: 'Reset to default image',
        toastTextSent: 'Text shown',
        toastGifChanged: 'GIF updated',
        toastGifReset: 'Reset to default image',
        syncLabel: 'Account sync',
        syncServerPlaceholder: 'Server address (e.g. http://localhost:9090)',
        syncLinkBtn: 'Link account',
        syncUseThis: 'Use this',
        syncPersonalLabel: 'Personal library',
        syncPublicLabel: 'Public library',
        syncLibraryEmpty: 'This account has no GIFs in its library yet',
        syncPublicEmpty: 'No public GIFs yet',
        syncUnlinkBtn: 'Unlink',
        syncLinkedAs: 'Linked as:',
        syncNotLinked: 'No account linked',
        toastSyncLinked: 'Account linked',
        toastSyncPulled: 'Synced the selected GIF',
        toastSyncUnlinked: 'Unlinked',
    },
};

const STORAGE_KEY = 'desktop-pet-lang';

export function getLang() {
    return localStorage.getItem(STORAGE_KEY) || 'vi';
}

export function setLang(lang) {
    localStorage.setItem(STORAGE_KEY, lang);
}

export function t(key) {
    const lang = getLang();
    return translations[lang][key] ?? translations.vi[key];
}

export function applyTranslations() {
    document.querySelectorAll('[data-i18n]').forEach((el) => {
        el.textContent = t(el.dataset.i18n);
    });
    document.querySelectorAll('[data-i18n-placeholder]').forEach((el) => {
        el.placeholder = t(el.dataset.i18nPlaceholder);
    });
    document.getElementById('lang-toggle').textContent = getLang() === 'vi' ? 'EN' : 'VI';
}
