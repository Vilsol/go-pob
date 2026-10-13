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
    const fetchMock = vi.fn((url: string) => Promise.resolve(new Response(` code-of-${url.split(/[/=]/).at(-1)} \n`)));
    vi.stubGlobal('fetch', fetchMock);
    currentBuildPath.current = '/Old';
    unsavedName.imported = false;
    await openImportLink('https://pobb.in/abc');
    expect(fetchMock).toHaveBeenCalledWith('/api/build?site=POBBin&id=abc');
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

  it('reads pob.codes directly (it allows pob.gg), every other site through /api/build', async () => {
    const fetchMock = vi.fn((url: string) => Promise.resolve(new Response(`code-of-${url}`)));
    vi.stubGlobal('fetch', fetchMock);
    await openImportLink('https://pob.codes/b/xyz');
    expect(fetchMock).toHaveBeenLastCalledWith('https://api.pob.codes/xyz/raw');
    await openImportLink('https://maxroll.gg/poe/pob/m5ui4t0x');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=Maxroll&id=m5ui4t0x');
    await openImportLink('https://pastebin.com/DsS6Qsxa');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=pastebin&id=DsS6Qsxa');
  });

  it("says what the proxy said, and that a host without it can't read sites", async () => {
    vi.stubGlobal('fetch', () => Promise.resolve(new Response('Pastebin.com answered 404', { status: 404 })));
    await expect(openImportLink('https://pastebin.com/gone')).rejects.toThrow(
      'could not read the build from Pastebin.com: Pastebin.com answered 404. Paste the code instead.'
    );
    vi.stubGlobal('fetch', () => Promise.resolve(new Response('<!doctype html>', { headers: { 'Content-Type': 'text/html' } })));
    await expect(openImportLink('https://pobb.in/abc')).rejects.toThrow("pob.gg's import proxy is not available here");
    expect(worker.ImportCode).not.toHaveBeenCalled();
  });

  it("opens PoB's pob://<site id>/<id> links as Main.lua does, poe.ninja's character link included", async () => {
    const fetchMock = vi.fn((url: string) => Promise.resolve(new Response(`code-of-${url}`)));
    vi.stubGlobal('fetch', fetchMock);
    await openImportLink('pob://pobbin/h-M3H3It2fDw');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=POBBin&id=h-M3H3It2fDw');
    await openImportLink('pob://pobcodes/xyz');
    expect(fetchMock).toHaveBeenLastCalledWith('https://api.pob.codes/xyz/raw');
    await openImportLink('pob://poeninja/overview/code?account=Vilsol-3184&name=Some_Char&overview=mercenaries&type=exp');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=PoeNinja&id=overview%2Fcode&account=Vilsol-3184&name=Some_Char&overview=mercenaries&type=exp');
    expect(worker.ImportCode).toHaveBeenCalledTimes(3);
  });

  it("does not take poe.ninja's builds page for a site link, as the pin's matchURL does not", async () => {
    const fetchMock = vi.fn(() => Promise.resolve(new Response('x')));
    vi.stubGlobal('fetch', fetchMock);
    const page = 'https://poe.ninja/poe1/builds/mercenaries/character/Vilsol-3184/Some_Char?i=9';
    await openImportLink(page);
    expect(fetchMock).not.toHaveBeenCalled();
    expect(worker.ImportCode).toHaveBeenCalledWith(page);
  });

  it('names a comparison by the popup, else by the site it came from', () => {
    expect(compareName('https://pobb.in/abc', '')).toBe('Imported from pobb.in');
    expect(compareName('eNrtfQ', '')).toBe('Imported build');
    expect(compareName('https://pobb.in/abc', 'Mine')).toBe('Mine');
  });
});
