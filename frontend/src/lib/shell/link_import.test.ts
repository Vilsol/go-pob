// Main.lua:56-85

import { afterEach, describe, expect, it, vi } from 'vitest';

const worker = vi.hoisted(() => ({ ImportCode: vi.fn((code: string) => Promise.resolve(void code)) }));
vi.mock('$lib/go/worker', () => ({ syncWrap: worker, syncWorker: undefined }));

const { compareName, importParam, openImportLink } = await import('./link_import');
const { currentBuildPath } = await import('$lib/global');
const { unsavedName } = await import('./build_name.svelte');

afterEach(() => {
  vi.unstubAllGlobals();
  worker.ImportCode.mockClear();
});

describe('opening a build from a link', () => {
  it('reads ?import=, and nothing without it', () => {
    expect(importParam(new URL('https://pob.gg/?import=eNrtXVtz'))).toBe('eNrtXVtz');
    expect(importParam(new URL('https://pob.gg/?import=https%3A%2F%2Fpobb.in%2Fabc'))).toBe('https://pobb.in/abc');
    expect(importParam(new URL('https://pob.gg/tree'))).toBeUndefined();
  });

  it('downloads a site link and opens it as an imported build', async () => {
    const fetchMock = vi.fn((url: string) => Promise.resolve(new Response(` code-of-${url.split('/').at(-1)} \n`)));
    vi.stubGlobal('fetch', fetchMock);
    currentBuildPath.current = '/Old';
    unsavedName.imported = false;
    await openImportLink('https://pobb.in/abc');
    expect(fetchMock).toHaveBeenCalledWith('https://pobb.in/pob/abc');
    expect(worker.ImportCode).toHaveBeenCalledWith('code-of-abc');
    expect(currentBuildPath.current).toBe('');
    expect(unsavedName.imported).toBe(true);
  });

  it('opens a code as it is, and says when a site cannot be read', async () => {
    const fetchMock = vi.fn(() => Promise.reject(new Error('blocked')));
    vi.stubGlobal('fetch', fetchMock);
    await openImportLink('  eNrtXVtz  ');
    expect(fetchMock).not.toHaveBeenCalled();
    expect(worker.ImportCode).toHaveBeenCalledWith('eNrtXVtz');
    await expect(openImportLink('https://pastebin.com/qwErTy12')).rejects.toThrow(/could not read the build from Pastebin\.com/);
  });

  it('names a comparison by the popup, else by the site it came from', () => {
    expect(compareName('https://pobb.in/abc', '')).toBe('Imported from pobb.in');
    expect(compareName('eNrtfQ', '')).toBe('Imported build');
    expect(compareName('https://pobb.in/abc', 'Mine')).toBe('Mine');
  });
});
