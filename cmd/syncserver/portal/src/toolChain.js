import { ref } from 'vue';

// Holds a just-produced GIF result so the next tool page can pick it up as
// its input file, instead of forcing the user to download it and re-upload
// (matches ezgif's "chain tools on the current output" workflow).
export const chainedFile = ref(null);

export function setChainedResult(blob, filename) {
    chainedFile.value = new File([blob], filename, { type: blob.type });
}

// Consumes the chained file so it's only applied once, even if the user
// visits several tool pages afterward without producing a new result.
export function takeChainedFile() {
    const f = chainedFile.value;
    chainedFile.value = null;
    return f;
}
