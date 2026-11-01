// Main.lua:56-85

import { afterEach, describe, expect, it, vi } from 'vitest';

const worker = vi.hoisted(() => ({ ImportCode: vi.fn((code: string) => Promise.resolve(void code)) }));
vi.mock('$lib/go/worker', () => ({ syncWrap: worker, syncWorker: undefined }));

const { compareName, importLanding, importParam, openImportLink } = await import('./link_import');
const { currentBuildPath, markBackendAsLoaded } = await import('$lib/global');
const { get } = await import('svelte/store');
// a PoE1 build waits for the game data
get(markBackendAsLoaded)();
const { unsavedName } = await import('./build_name.svelte');
const { EXPORT_SITES, NOT_A_CODE } = await import('$lib/design/share');

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

  it('lands on the requested page, or the tree from the root', () => {
    expect(importLanding(new URL('https://pob.gg/?import=eNrt'), '')).toBe('/tree');
    expect(importLanding(new URL('https://pob.gg/items?import=eNrt&sw=off'), '')).toBe('/items?sw=off');
    expect(importLanding(new URL('https://x.io/pob/?import=eNrt'), '/pob')).toBe('/pob/tree');
    expect(importLanding(new URL('https://x.io/pob/calcs/?import=eNrt'), '/pob')).toBe('/pob/calcs/');
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

  it("reads PoB-PoE2's site links through /api/build, and leaves pob:// to PoE1's sites", async () => {
    const fetchMock = vi.fn((url: string) => Promise.resolve(new Response(`code-of-${url}`)));
    vi.stubGlobal('fetch', fetchMock);
    await openImportLink('https://poe.ninja/poe2/pob/Abc1');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=PoeNinja2&id=Abc1');
    await openImportLink('https://poe2.ninja/pob/Abc2');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=PoeNinja2&id=Abc2');
    await openImportLink('https://poe.ninja/poe1/pob/Abc3');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=PoeNinja&id=Abc3');
    // poe.ninja answers /pob/<id> with a 301 to /poe1/pob/<id>, and one id names a different build in each game
    await openImportLink('https://poe.ninja/pob/Abc4');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=PoeNinja&id=Abc4');
    await openImportLink('https://maxroll.gg/poe2/pob/m5ui4t0x');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=Maxroll2&id=m5ui4t0x');
    await openImportLink('https://poe2db.tw/pob/abc');
    expect(fetchMock).toHaveBeenLastCalledWith('/api/build?site=PoE2DB&id=abc');
    expect(fetchMock).toHaveBeenCalledTimes(6);
    await openImportLink('pob://poeninja2/Abc1');
    expect(fetchMock).toHaveBeenCalledTimes(6);
    expect(worker.ImportCode).toHaveBeenLastCalledWith('pob://poeninja2/Abc1');
    expect(EXPORT_SITES.filter((s) => s.game)).toEqual([]);
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

  it('with the PoE2 flag off refuses a PoE2 build from a code, a site link, a pob:// link or an opened file, and keeps the open build', async () => {
    const { POE2_DISABLED } = await import('$lib/go/game_route');
    const { poe2Enabled } = await import('$lib/global');
    poe2Enabled.set(false);
    vi.stubGlobal(
      'fetch',
      vi.fn(() => Promise.resolve(new Response('eNrt', { headers: { 'Content-Type': 'text/plain' } })))
    );
    worker.ImportCode.mockImplementation(() => Promise.resolve({ game: 'poe2', xml: '<PathOfBuilding2/>' } as never));
    currentBuildPath.current = '/Open';
    unsavedName.imported = false;
    for (const text of ['eNrt', 'https://poe.ninja/poe2/pob/Abc1', 'pob://pobbin/abc', 'gopob-file:/home/a.xml'])
      await expect(openImportLink(text), text).rejects.toThrow(POE2_DISABLED);
    expect(worker.ImportCode).toHaveBeenCalledTimes(4);
    expect(currentBuildPath.current).toBe('/Open');
    expect(unsavedName.imported).toBe(false);
    worker.ImportCode.mockImplementation((code: string) => Promise.resolve(void code));
  });

  it('names a comparison by the popup, else by the site it came from', () => {
    expect(compareName('https://pobb.in/abc', '')).toBe('Imported from pobb.in');
    expect(compareName('eNrtfQ', '')).toBe('Imported build');
    expect(compareName('https://pobb.in/abc', 'Mine')).toBe('Mine');
  });
});

it('a link whose download is not a build code says so', async () => {
  worker.ImportCode.mockImplementation(() => Promise.reject(new Error('failed to create a zlib reader: illegal base64 data at input byte 0')));
  await expect(openImportLink('not-a-code')).rejects.toThrow(NOT_A_CODE);
  worker.ImportCode.mockImplementation((code: string) => Promise.resolve(void code));
});
