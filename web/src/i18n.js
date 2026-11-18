export const translations = {
    vi: {
        subtitle: 'Điều khiển hoạt ảnh và chữ hiển thị trên màn hình',
        textLabel: 'Chữ hiển thị',
        textPlaceholder: 'Gõ gì vào đây...',
        textSubmit: 'Hiển thị',
        gifLabel: 'Ảnh GIF',
        dropzoneDefault: 'Chọn hoặc kéo thả file GIF',
        dropzoneMax: 'Tối đa 3MB',
        uploadSubmit: 'Áp dụng',
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
        syncQuotaSuffix: 'GIF tối đa',
        pushCurrentGifBtn: 'Tải lên',
        syncShareBtn: 'Chia sẻ',
        syncUnshareBtn: 'Gỡ chia sẻ',
        syncDeleteBtn: 'Xóa',
        syncSharePromptName: 'Đặt tên cho GIF này (bắt buộc để chia sẻ):',
        syncSharePromptTags: 'Tag (phân tách bằng dấu phẩy, có thể để trống):',
        toastSyncUploaded: 'Đã tải lên thư viện',
        toastSyncUploadQuotaFull: 'Đã đạt số lượng GIF tối đa, xóa 1 cái trước khi thêm',
        toastSyncShared: 'Đã chia sẻ',
        toastSyncUnshared: 'Đã gỡ chia sẻ',
        toastSyncDeleted: 'Đã xóa khỏi thư viện',
        confirmSyncDelete: 'Xóa GIF này khỏi thư viện trên server?',
        confirmApply: 'Áp dụng ảnh GIF này cho widget?',
        confirmReset: 'Đặt lại ảnh mặc định? Ảnh hiện tại sẽ bị thay thế.',
        confirmPull: 'Đồng bộ ảnh này vào widget? Ảnh hiện tại sẽ bị thay thế.',
        confirmUnshare: 'Gỡ chia sẻ GIF này? Link public sẽ không còn hoạt động.',
        confirmPushUpload: 'Tải ảnh hiện tại lên thư viện server?',
        confirmUnlink: 'Hủy liên kết tài khoản khỏi thiết bị này?',
        modalCancel: 'Hủy',
        syncUnlinkBtn: 'Hủy liên kết',
        syncLinkedAs: 'Đã liên kết:',
        syncNotLinked: 'Chưa liên kết tài khoản nào',
        toastSyncLinked: 'Đã liên kết tài khoản',
        toastSyncPulled: 'Đã đồng bộ ảnh đã chọn',
        toastSyncUnlinked: 'Đã hủy liên kết',
        toastSyncExpired: 'Phiên đăng nhập hết hạn hoặc đã bị thu hồi, vui lòng liên kết lại',
    },
    en: {
        subtitle: 'Control the on-screen animation and text',
        textLabel: 'Display text',
        textPlaceholder: 'Type something...',
        textSubmit: 'Show',
        gifLabel: 'GIF image',
        dropzoneDefault: 'Choose or drop a GIF file',
        dropzoneMax: 'Up to 3MB',
        uploadSubmit: 'Apply',
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
        syncQuotaSuffix: 'GIF max',
        pushCurrentGifBtn: 'Upload',
        syncShareBtn: 'Share',
        syncUnshareBtn: 'Unshare',
        syncDeleteBtn: 'Delete',
        syncSharePromptName: 'Name this GIF (required to share):',
        syncSharePromptTags: 'Tags (comma-separated, optional):',
        toastSyncUploaded: 'Uploaded to library',
        toastSyncUploadQuotaFull: 'GIF quota reached, delete one first',
        toastSyncShared: 'Shared',
        toastSyncUnshared: 'Unshared',
        toastSyncDeleted: 'Removed from library',
        confirmSyncDelete: 'Remove this GIF from the server library?',
        confirmApply: 'Apply this GIF to the widget?',
        confirmReset: 'Reset to the default image? The current one will be replaced.',
        confirmPull: 'Sync this image to the widget? The current one will be replaced.',
        confirmUnshare: 'Unshare this GIF? Its public link will stop working.',
        confirmPushUpload: 'Upload the current image to the server library?',
        confirmUnlink: 'Unlink this device from the account?',
        modalCancel: 'Cancel',
        syncUnlinkBtn: 'Unlink',
        syncLinkedAs: 'Linked as:',
        syncNotLinked: 'No account linked',
        toastSyncLinked: 'Account linked',
        toastSyncPulled: 'Synced the selected GIF',
        toastSyncUnlinked: 'Unlinked',
        toastSyncExpired: 'Session expired or revoked, please link again',
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
