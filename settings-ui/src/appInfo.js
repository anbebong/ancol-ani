// App version + changelog, both served by the desktop app itself
// (GET /version from the VERSION file, GET /changelog from CHANGELOG.md,
// embedded at build time), so what the page shows always matches the
// running build.
import { reactive, computed } from 'vue';

const SEEN_KEY = 'ancol-ani-seen-version';

function read(key) { try { return localStorage.getItem(key); } catch (_) { return null; } }
function write(key, v) { try { localStorage.setItem(key, v); } catch (_) {} }

export const appInfo = reactive({
    version: '',
    seenVersion: read(SEEN_KEY) || '',
});

// True when the app was updated since the person last opened "What's new".
export const hasUnseenUpdate = computed(() => !!appInfo.version && appInfo.version !== appInfo.seenVersion);

export function markVersionSeen() {
    if (!appInfo.version) return;
    appInfo.seenVersion = appInfo.version;
    write(SEEN_KEY, appInfo.version);
}

export function loadVersion() {
    return fetch('/version')
        .then((r) => (r.ok ? r.text() : ''))
        .then((v) => {
            appInfo.version = v.trim();
            // First run ever on this browser: nothing to announce yet.
            if (appInfo.version && !appInfo.seenVersion) markVersionSeen();
        })
        .catch(() => {});
}

export async function loadChangelog() {
    const res = await fetch('/changelog', { cache: 'no-store' });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return parseChangelog(await res.text());
}

// Parses CHANGELOG.md into structured data rendered by ChangelogView (no
// v-html, so nothing in the file can inject markup). Understands only the
// format the file documents:
//   ## <version> — <date>      a release
//   ### <title>                a group inside a release (Mới / Sửa lỗi ...)
//   - item                     a bullet, with `code` and **bold** inline
// Anything else (intro paragraphs) is ignored.
export function parseChangelog(md) {
    const releases = [];
    let release = null;
    let group = null;

    for (const raw of md.split(/\r?\n/)) {
        const line = raw.trimEnd();
        let m;
        if ((m = line.match(/^##\s+(\S+)(?:\s+[—–-]\s+(.+))?$/))) {
            release = { version: m[1], date: (m[2] || '').trim(), groups: [] };
            group = null;
            releases.push(release);
        } else if (release && (m = line.match(/^###\s+(.+)$/))) {
            group = { title: m[1].trim(), items: [] };
            release.groups.push(group);
        } else if (release && (m = line.match(/^\s*[-*]\s+(.+)$/))) {
            if (!group) {
                group = { title: '', items: [] };
                release.groups.push(group);
            }
            group.items.push(inlineTokens(m[1]));
        }
    }
    return releases;
}

function inlineTokens(text) {
    const tokens = [];
    const re = /`([^`]+)`|\*\*([^*]+)\*\*/g;
    let last = 0;
    let m;
    while ((m = re.exec(text))) {
        if (m.index > last) tokens.push({ type: 'text', value: text.slice(last, m.index) });
        tokens.push(m[1] !== undefined ? { type: 'code', value: m[1] } : { type: 'bold', value: m[2] });
        last = re.lastIndex;
    }
    if (last < text.length) tokens.push({ type: 'text', value: text.slice(last) });
    return tokens;
}
