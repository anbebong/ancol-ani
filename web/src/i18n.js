import { ref } from 'vue';

export const translations = {
    vi: {
        subtitle: 'Điều khiển hoạt ảnh và chữ hiển thị trên màn hình',

        navWidget: 'Hoạt ảnh',
        navLibrary: 'Thư viện',
        navGallery: 'Cộng đồng',
        navLink: 'Liên kết',
        langToggleTitle: 'Switch to English',

        // Widget page
        widgetTitle: 'Hoạt ảnh',
        textLabel: 'Chữ hiển thị',
        textHint: 'Hiện trong bong bóng ngay phía trên hoạt ảnh. Nên ngắn gọn: tối đa khoảng 2 dòng, dài hơn sẽ bị cắt.',
        textPlaceholder: 'Gõ gì vào đây...',
        textSubmit: 'Hiển thị',
        positionLabel: 'Vị trí trên màn hình',
        positionHint: 'Chọn góc màn hình để đặt widget.',
        positionTL: 'Trên trái',
        positionTR: 'Trên phải',
        positionBL: 'Dưới trái',
        positionBR: 'Dưới phải',
        autoStartLabel: 'Khởi động cùng Windows',
        autoStartHint: 'Tự mở widget mỗi khi bạn đăng nhập vào máy.',
        gifLabel: 'Ảnh GIF',
        gifCurrent: 'Đang hiển thị',
        dropzoneDefault: 'Kéo thả file GIF vào đây',
        dropzoneOr: 'hoặc bấm để chọn file · tối đa 3MB',
        uploadSubmit: 'Áp dụng',
        resetSubmit: 'Đặt lại mặc định',
        pushCurrentGifBtn: 'Tải ảnh này lên thư viện',
        toastTextSent: 'Đã hiển thị',
        toastGifChanged: 'Đã đổi ảnh GIF',
        toastGifReset: 'Đã đặt lại ảnh mặc định',
        errPickFile: 'Chọn file GIF trước',
        errGifOnly: 'Chỉ nhận file .gif',
        confirmApply: 'Áp dụng ảnh GIF này cho widget?',
        confirmReset: 'Đặt lại ảnh mặc định? Ảnh hiện tại sẽ bị thay thế.',

        // Library page
        libraryTitle: 'Thư viện của tôi',
        librarySubtitle: 'GIF trong tài khoản trên server. Chọn 1 cái để dùng cho widget trên máy này.',
        libraryDropTitle: 'Kéo thả file GIF để tải lên thư viện',
        libraryEmpty: 'Tài khoản chưa có GIF nào. Tải lên ở khung phía trên, hoặc tải ảnh widget hiện tại lên.',
        quotaFull: 'Đã đạt số lượng GIF tối đa ({quota}). Xóa bớt 1 GIF để thêm cái mới.',
        badgePublic: 'công khai',
        badgePrivate: 'riêng tư',
        useThis: 'Sử dụng GIF',
        shareNamePlaceholder: 'Tên GIF (bắt buộc để chia sẻ)',
        shareTagsPlaceholder: 'Tag, phân tách bằng dấu phẩy',
        shareBtn: 'Chia sẻ',
        unshareBtn: 'Gỡ chia sẻ',
        deleteBtn: 'Xóa',
        errShareName: 'Phải đặt tên cho GIF trước khi chia sẻ',
        toastUploaded: 'Đã tải lên thư viện',
        toastQuotaFull: 'Đã đạt số lượng GIF tối đa, xóa 1 cái trước khi thêm',
        toastShared: 'Đã chia sẻ công khai',
        toastUnshared: 'Đã gỡ chia sẻ',
        toastDeleted: 'Đã xóa khỏi thư viện',
        toastPulled: 'Đã đổi ảnh widget sang GIF đã chọn',
        confirmDelete: 'Xóa GIF này khỏi thư viện trên server?',
        confirmUnshare: 'Gỡ chia sẻ GIF này? Link public sẽ không còn hoạt động.',
        confirmPull: 'Dùng GIF này cho widget? Ảnh hiện tại sẽ bị thay thế.',
        confirmPushUpload: 'Tải ảnh widget hiện tại lên thư viện server?',

        // Gallery page
        galleryTitle: 'Cộng đồng',
        gallerySubtitle: 'GIF mọi người đã chia sẻ công khai.',
        searchPlaceholder: 'Tìm theo tên hoặc tag...',
        userFilterPlaceholder: 'Lọc theo người chia sẻ...',
        galleryEmpty: 'Chưa có GIF nào được chia sẻ công khai.',
        galleryNoMatch: 'Không tìm thấy GIF nào khớp.',
        by: 'bởi',

        // Account page
        accountTitle: 'Tài khoản',
        accountSubtitleLinked: 'Máy này đang liên kết với tài khoản {username}.',
        accountSubtitleUnlinked: 'Liên kết với tài khoản trên server để đồng bộ và chia sẻ GIF.',
        linkCardTitle: 'Liên kết tài khoản',
        linkCardHint: 'Bấm Liên kết để mở trang đăng nhập của server. Đăng nhập xong sẽ tự quay về đây.',
        serverLabel: 'Địa chỉ server',
        serverPlaceholder: 'vd: http://localhost:9090',
        linkBtn: 'Liên kết tài khoản',
        linkedAs: 'Tài khoản',
        quotaLabel: 'Quota GIF',
        openPortal: 'Mở portal',
        changePasswordHint: 'Đổi mật khẩu ở trang Tài khoản trên portal.',
        unlinkBtn: 'Hủy liên kết',
        confirmUnlink: 'Hủy liên kết tài khoản khỏi thiết bị này?',
        toastLinked: 'Đã liên kết tài khoản',
        toastUnlinked: 'Đã hủy liên kết',
        toastExpired: 'Phiên đăng nhập hết hạn hoặc đã bị thu hồi, vui lòng liên kết lại',
        needLink: 'Cần liên kết tài khoản trước',

        // What's new
        navChangelog: 'Về Ancol-Ani',
        footerPolicy: 'Chính sách',
        changelogTitle: 'Về Ancol-Ani',
        changelogSubtitle: 'Phiên bản đang chạy: v{version}',
        changelogCurrent: 'đang dùng',
        changelogError: 'Không tải được changelog.',
        changelogNewDot: 'Có phiên bản mới, xem thay đổi',

        modalCancel: 'Hủy',
        pagerPrev: '‹ Trước',
        pagerNext: 'Sau ›',
        pagerLabel: 'Trang {page}/{total}',
    },
    en: {
        subtitle: 'Control the on-screen animation and text',

        navWidget: 'Widget',
        navLibrary: 'Library',
        navGallery: 'Explore',
        navLink: 'Link',
        langToggleTitle: 'Chuyển sang tiếng Việt',

        widgetTitle: 'Widget',
        textLabel: 'Display text',
        textHint: 'Shown in a bubble right above the animation. Keep it short: about 2 lines max, longer text gets cut off.',
        textPlaceholder: 'Type something...',
        textSubmit: 'Show',
        positionLabel: 'Screen position',
        positionHint: 'Pick a corner of the screen for the widget.',
        positionTL: 'Top left',
        positionTR: 'Top right',
        positionBL: 'Bottom left',
        positionBR: 'Bottom right',
        autoStartLabel: 'Start with Windows',
        autoStartHint: 'Open the widget automatically whenever you log in.',
        gifLabel: 'GIF image',
        gifCurrent: 'Now showing',
        dropzoneDefault: 'Drop a GIF file here',
        dropzoneOr: 'or click to choose · up to 3MB',
        uploadSubmit: 'Apply',
        resetSubmit: 'Reset to default',
        pushCurrentGifBtn: 'Upload this to library',
        toastTextSent: 'Text shown',
        toastGifChanged: 'GIF updated',
        toastGifReset: 'Reset to default image',
        errPickFile: 'Choose a GIF file first',
        errGifOnly: 'Only .gif files are accepted',
        confirmApply: 'Apply this GIF to the widget?',
        confirmReset: 'Reset to the default image? The current one will be replaced.',

        libraryTitle: 'My library',
        librarySubtitle: 'GIFs in your server account. Pick one to use on this device\'s widget.',
        libraryDropTitle: 'Drop a GIF file to upload it to your library',
        libraryEmpty: 'No GIFs in this account yet. Upload one above, or upload the current widget image.',
        quotaFull: 'GIF limit reached ({quota}). Delete one to add a new one.',
        badgePublic: 'public',
        badgePrivate: 'private',
        useThis: 'Use on widget',
        shareNamePlaceholder: 'GIF name (required to share)',
        shareTagsPlaceholder: 'Tags, comma-separated',
        shareBtn: 'Share',
        unshareBtn: 'Unshare',
        deleteBtn: 'Delete',
        errShareName: 'Name the GIF before sharing it',
        toastUploaded: 'Uploaded to library',
        toastQuotaFull: 'GIF quota reached, delete one first',
        toastShared: 'Shared publicly',
        toastUnshared: 'Unshared',
        toastDeleted: 'Removed from library',
        toastPulled: 'Widget now shows the selected GIF',
        confirmDelete: 'Remove this GIF from the server library?',
        confirmUnshare: 'Unshare this GIF? Its public link will stop working.',
        confirmPull: 'Use this GIF on the widget? The current one will be replaced.',
        confirmPushUpload: 'Upload the current widget image to the server library?',

        galleryTitle: 'Explore',
        gallerySubtitle: 'GIFs people have shared publicly.',
        searchPlaceholder: 'Search by name or tag...',
        userFilterPlaceholder: 'Filter by who shared it...',
        galleryEmpty: 'No public GIFs yet.',
        galleryNoMatch: 'No GIFs match.',
        by: 'by',

        accountTitle: 'Account',
        accountSubtitleLinked: 'This device is linked to the account {username}.',
        accountSubtitleUnlinked: 'Link a server account to sync and share GIFs.',
        linkCardTitle: 'Link account',
        linkCardHint: 'Link opens the server\'s sign-in page and brings you back here afterwards.',
        serverLabel: 'Server address',
        serverPlaceholder: 'e.g. http://localhost:9090',
        linkBtn: 'Link account',
        linkedAs: 'Account',
        quotaLabel: 'GIF quota',
        openPortal: 'Open portal',
        changePasswordHint: 'Change your password on the portal\'s Account page.',
        unlinkBtn: 'Unlink',
        confirmUnlink: 'Unlink this device from the account?',
        toastLinked: 'Account linked',
        toastUnlinked: 'Unlinked',
        toastExpired: 'Session expired or revoked, please link again',
        needLink: 'Link an account first',

        navChangelog: 'About Ancol-Ani',
        footerPolicy: 'Policy',
        changelogTitle: 'About Ancol-Ani',
        changelogSubtitle: 'Running version: v{version}',
        changelogCurrent: 'current',
        changelogError: 'Could not load the changelog.',
        changelogNewDot: 'New version available, see what changed',

        modalCancel: 'Cancel',
        pagerPrev: '‹ Prev',
        pagerNext: 'Next ›',
        pagerLabel: 'Page {page}/{total}',
    },
};

const STORAGE_KEY = 'desktop-pet-lang';

function readLang() {
    try { return localStorage.getItem(STORAGE_KEY) === 'en' ? 'en' : 'vi'; } catch (_) { return 'vi'; }
}

// Reactive, so every template that calls t() re-renders on a switch.
export const lang = ref(readLang());

export function toggleLang() {
    lang.value = lang.value === 'vi' ? 'en' : 'vi';
    try { localStorage.setItem(STORAGE_KEY, lang.value); } catch (_) {}
    document.documentElement.lang = lang.value;
}

export function t(key, vars) {
    let s = translations[lang.value][key] ?? translations.vi[key] ?? key;
    if (vars) s = s.replace(/\{(\w+)\}/g, (_, k) => vars[k] ?? '');
    return s;
}
