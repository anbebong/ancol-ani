// Helper for the public, unauthenticated /api/tools/* endpoints (GIF
// toolkit). Unlike api.js's api() helper, these requests carry no auth
// header and the server never sets Content-Disposition, so the filename
// always falls back to a sensible default based on the response's
// Content-Type.
export async function postTool(path, formData) {
    const res = await fetch(path, { method: 'POST', body: formData });
    if (!res.ok) throw new Error((await res.text()) || `Lỗi ${res.status}`);
    const blob = await res.blob();
    const cd = res.headers.get('Content-Disposition') || '';
    const match = cd.match(/filename="?([^"]+)"?/);
    const filename = match ? match[1] : (blob.type === 'application/zip' ? 'result.zip' : 'result.gif');
    return { blob, filename };
}
