// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { Blob as NodeBlob } from 'node:buffer';

afterEach(async () => {
  cleanup();
  (await import('./import_modal.svelte')).importDialog.open = false;
});

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted page threw').toEqual([]);
});

// ImportTab.lua:652, 666
const goto = vi.fn((url: string) => Promise.resolve(void url));
vi.mock('$app/navigation', () => ({ goto }));

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, outputs, poe2Enabled } = await import('$lib/global');
const { NOT_A_CODE, POE2_CHARACTER, POE2_CHARACTER_DISABLED, unreadableBuild } = await import('$lib/design/share');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const BUILD = '0008-necromancer-vaal-summon-skeletons.xml';
const OTHER = '0005-occultist-vaal-double-strike.xml';

const serialize = (b: unknown) => new TextDecoder().decode(types.builds.SerializeBuild(b as never).unwrap());
let other: { code: string; xml: string };
let tick: MockInstance;
const writeText = vi.fn((text: string) => Promise.resolve(void text));

/* Build.lua:89-94 */
const topBar = async () => {
  const { opening } = await import('$lib/shell/build_name.svelte');
  opening.modFlag = false;
  return render((await import('$lib/shell/AppTopBar.svelte')).default);
};

const open = async (tab: 'import' | 'export') => {
  (await import('./import_modal.svelte')).openImport(tab);
  const view = render((await import('./AppImportModal.svelte')).default);
  return { view, dialog: view.getByRole('dialog', { name: 'Import / Export' }) };
};
const mount = async () => {
  const { view, dialog } = await open('import');
  return { view, dialog, box: within(within(dialog).getByRole('region', { name: 'Build import' })) };
};
const mountExport = async () => {
  const { view, dialog } = await open('export');
  const code = within(dialog).getByRole<HTMLTextAreaElement>('textbox', { name: 'Build code' });
  await vi.waitFor(() => expect(code.value).not.toBe(''));
  return { view, dialog, code };
};

beforeAll(async () => {
  await bootWorker(currentBuild);
  await loadCorpusBuild(BUILD);
  tick = vi.spyOn(workerInstance, 'Tick').mockResolvedValue(undefined);
  outputs.set(undefined);
  const parsed = types.builds.ParseBuildStr(await readFile(resolve(process.cwd(), '../testdata/corpus-3.29', OTHER), 'utf8')).unwrap();
  const xml = serialize(parsed);
  other = { xml, code: types.pob.CompressEncode(xml).unwrap() };
  expect(typeof Blob.prototype.stream, 'jsdom grew Blob.stream; drop this swap').toBe('undefined');
  vi.stubGlobal('Blob', NodeBlob);
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true });
}, 120_000);

describe('the Import / Export dialog', () => {
  it('mounting alone writes nothing and does not Tick', async () => {
    const before = serialize(workerInstance.currentBuild);
    tick.mockClear();
    await mountExport();
    await new Promise((r) => setTimeout(r, 50));
    expect(serialize(workerInstance.currentBuild)).toBe(before);
    expect(tick).not.toHaveBeenCalled();
  }, 20_000);

  it("shows Go's code for the build it holds, and re-reads it after a recalculation", async () => {
    const { code } = await mountExport();
    expect(code.value).toBe(workerInstance.ExportCode());
    expect(types.pob.DecodeDecompress(code.value).unwrap()).toBe(serialize(workerInstance.currentBuild));
    const notes = workerInstance.currentBuild!.Notes;
    workerInstance.currentBuild!.Notes = 'changed while the page was open';
    outputs.set({} as never);
    await vi.waitFor(() => expect(code.value).toBe(workerInstance.ExportCode()));
    expect(types.pob.DecodeDecompress(code.value).unwrap()).toContain('changed while the page was open');
    workerInstance.currentBuild!.Notes = notes;
    outputs.set(undefined);
  });

  it('Export support (ImportTab.lua:506-509) shows what Go holds, writes the change and Ticks', async () => {
    const { view } = await mountExport();
    const box = view.getByRole('checkbox', { name: /^Export support/ }) as HTMLInputElement;
    expect(box.disabled).toBe(false);
    expect(box.checked).toBe(false);
    tick.mockClear();
    await fireEvent.click(box);
    await vi.waitFor(() => expect(workerInstance.currentBuild!.ExportParty()).toBe(true));
    expect(tick).toHaveBeenCalled();
    await fireEvent.click(box);
    await vi.waitFor(() => expect(workerInstance.currentBuild!.ExportParty()).toBe(false));
  });

  it('Copy code puts that code on the clipboard', async () => {
    const { view, code } = await mountExport();
    writeText.mockClear();
    await fireEvent.click(view.getByRole('button', { name: 'Copy code' }));
    expect(writeText).toHaveBeenCalledWith(code.value);
    const sharing = within(view.getByRole('region', { name: 'Build sharing' }));
    await vi.waitFor(() => expect(sharing.getByRole('status').textContent).toContain('Copied'));
  });

  // ImportTab.lua:663-666: an import never replaces the open build; it opens as a new one
  it('an imported code opens as a new build without the current save path, closes the dialog, then Ticks', async () => {
    const { currentBuildPath } = await import('$lib/global');
    currentBuildPath.current = 'Builds/kept.xml';
    const { box, dialog } = await mount();
    const field = box.getByRole('textbox', { name: /^To import a build/ });
    const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    expect(go.disabled).toBe(true);
    await fireEvent.input(field, { target: { value: other.code } });
    await vi.waitFor(() =>
      expect(field.getAttribute('aria-describedby') && document.getElementById(field.getAttribute('aria-describedby')!)!.textContent).toBe('Code is valid')
    );
    expect(go.disabled).toBe(false);
    expect(box.getByRole<HTMLInputElement>('radio', { name: /Import as a new build/ }).checked).toBe(true);
    expect(box.queryByRole('radio', { name: /Import to this build/ }), 'no mode replaces the open build').toBeNull();
    expect(serialize(workerInstance.currentBuild), 'typing alone must not write').not.toBe(other.xml);
    tick.mockClear();
    goto.mockClear();
    const bar = await topBar();
    await fireEvent.click(go);
    await vi.waitFor(() => expect(serialize(workerInstance.currentBuild)).toBe(other.xml));
    expect(within(document.body).queryByRole('dialog', { name: 'Build Import' }), 'no overwrite confirm').toBeNull();
    expect(tick).toHaveBeenCalledWith('ImportCode');
    await bar.findByRole('button', { name: 'Save (unsaved changes)' });
    await vi.waitFor(() => expect(currentBuildPath.current, 'the new build still saves over the old one').toBe(''));
    const { unsavedName } = await import('$lib/shell/build_name.svelte');
    expect(unsavedName.imported, 'named "Imported build" (ImportTab.lua:665)').toBe(true);
    await vi.waitFor(() => expect(goto).toHaveBeenCalledWith(expect.stringMatching(/\/tree$/)));
    expect((dialog as HTMLDialogElement).open, 'the dialog closes once the build opens').toBe(false);
  });

  it('Enter in the field imports a valid code (ImportTab.lua:671-675)', async () => {
    await loadCorpusBuild(BUILD);
    const { box } = await mount();
    await topBar();
    const field = box.getByRole('textbox', { name: /^To import a build/ });
    await fireEvent.input(field, { target: { value: other.code } });
    await vi.waitFor(() => expect(box.getByRole<HTMLButtonElement>('button', { name: 'Import' }).disabled).toBe(false));
    await fireEvent.keyDown(field, { key: 'Enter' });
    await vi.waitFor(() => expect(serialize(workerInstance.currentBuild)).toBe(other.xml));
  });

  // Build.lua:1421-1448: the same unsaved-changes prompt leaving the build asks
  it("over a build with unsaved changes it asks first: Cancel imports nothing, Don't Save imports and leaves the saved file alone", async () => {
    await loadCorpusBuild(BUILD);
    const before = serialize(workerInstance.currentBuild);
    const { currentBuildPath } = await import('$lib/global');
    const { opening } = await import('$lib/shell/build_name.svelte');
    currentBuildPath.current = 'Builds/kept.xml';
    opening.modFlag = true;
    const bar = render((await import('$lib/shell/AppTopBar.svelte')).default);
    await bar.findByRole('button', { name: 'Save (unsaved changes)' });
    const save = vi.spyOn(workerInstance, 'SaveBuildAs');
    const { box } = await mount();
    const field = box.getByRole('textbox', { name: /^To import a build/ });
    await fireEvent.input(field, { target: { value: other.code } });
    const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    await vi.waitFor(() => expect(go.disabled).toBe(false));

    await fireEvent.click(go);
    let prompt = await vi.waitFor(() => within(document.body).getByRole('dialog', { name: 'Save Changes' }));
    await fireEvent.click(within(prompt).getByRole('button', { name: 'Cancel' }));
    await new Promise((r) => setTimeout(r, 200));
    expect(serialize(workerInstance.currentBuild), 'Cancel keeps the open build').toBe(before);
    expect(currentBuildPath.current).toBe('Builds/kept.xml');
    await bar.findByRole('button', { name: 'Save (unsaved changes)' });

    await fireEvent.click(go);
    prompt = await vi.waitFor(() => within(document.body).getByRole('dialog', { name: 'Save Changes' }));
    await fireEvent.click(within(prompt).getByRole('button', { name: "Don't Save" }));
    await vi.waitFor(() => expect(serialize(workerInstance.currentBuild)).toBe(other.xml));
    await vi.waitFor(() => expect(currentBuildPath.current, 'the import does not take over the saved build').toBe(''));
    expect(save, 'the old build is not written').not.toHaveBeenCalled();
    save.mockRestore();
  });

  it('a saved build with no unsaved changes is not asked about, and keeps its file', async () => {
    await loadCorpusBuild(BUILD);
    const { currentBuildPath } = await import('$lib/global');
    currentBuildPath.current = 'Builds/kept.xml';
    await topBar();
    const save = vi.spyOn(workerInstance, 'SaveBuildAs');
    const { box } = await mount();
    await fireEvent.input(box.getByRole('textbox', { name: /^To import a build/ }), { target: { value: other.code } });
    const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    await vi.waitFor(() => expect(go.disabled).toBe(false));
    await fireEvent.click(go);
    await vi.waitFor(() => expect(serialize(workerInstance.currentBuild)).toBe(other.xml));
    expect(within(document.body).queryByRole('dialog', { name: 'Save Changes' })?.hasAttribute('open') ?? false).toBe(false);
    await vi.waitFor(() => expect(currentBuildPath.current).toBe(''));
    expect(save).not.toHaveBeenCalled();
    save.mockRestore();
  });

  it('a code that inflates but is no build says so, and leaves the build alone', async () => {
    const { box } = await mount();
    const before = serialize(workerInstance.currentBuild);
    const field = box.getByRole('textbox', { name: /^To import a build/ });
    await fireEvent.input(field, { target: { value: types.pob.CompressEncode('not a build').unwrap() } });
    const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    await vi.waitFor(() => expect(go.disabled).toBe(false));
    await fireEvent.click(go);
    await vi.waitFor(() => expect(box.getByRole('alert').textContent).toMatch(/^Import failed: ./));
    expect(serialize(workerInstance.currentBuild)).toBe(before);
  });

  // ImportTab.lua:684-699, BuildSiteTools.lua:88-113
  describe('a site link', () => {
    const LINK = 'https://pobb.in/aBcD1234';
    afterEach(() => vi.unstubAllGlobals());

    const typeLink = async () => {
      const { box } = await mount();
      const field = box.getByRole('textbox', { name: /^To import a build/ });
      await fireEvent.input(field, { target: { value: LINK } });
      const verdict = document.getElementById(field.getAttribute('aria-describedby')!)!;
      await vi.waitFor(() => expect(verdict.textContent).toBe('URL is valid (pobb.in)'));
      return box;
    };

    it("downloads the site's raw code through pob.gg's proxy (/api/build), then imports it and Ticks", async () => {
      let answer!: (r: Response) => void;
      const fetch = vi.fn((url: string) => {
        void url;
        return new Promise<Response>((r) => (answer = r));
      });
      vi.stubGlobal('fetch', fetch);
      await loadCorpusBuild(BUILD);
      expect(serialize(workerInstance.currentBuild), 'the build is already the one the link holds').not.toBe(other.xml);
      const box = await typeLink();
      const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
      expect(go.disabled).toBe(false);
      tick.mockClear();
      await fireEvent.click(go);
      await vi.waitFor(() => expect(fetch).toHaveBeenCalledWith('/api/build?site=POBBin&id=aBcD1234'));
      const busy = box.getByRole<HTMLButtonElement>('button', { name: 'Retrieving paste..' });
      expect(busy.disabled).toBe(true);
      answer(new Response(`${other.code}\n`, { status: 200 }));
      await vi.waitFor(() => expect(serialize(workerInstance.currentBuild)).toBe(other.xml));
      expect(tick).toHaveBeenCalledWith('ImportCode');
    });

    it('says the site could not be read (CORS or offline), and leaves the build alone', async () => {
      vi.stubGlobal(
        'fetch',
        vi.fn(() => Promise.reject(new TypeError('Failed to fetch')))
      );
      const before = serialize(workerInstance.currentBuild);
      const box = await typeLink();
      await fireEvent.click(box.getByRole('button', { name: 'Import' }));
      await vi.waitFor(() => expect(box.getByRole('alert').textContent).toMatch(/^Import failed: .*pobb\.in.*CORS/));
      expect(serialize(workerInstance.currentBuild)).toBe(before);
    });

    it('says a downloaded page that is not a build code is not one, and leaves the build alone', async () => {
      vi.stubGlobal(
        'fetch',
        vi.fn(() => Promise.resolve(new Response('<html>not found</html>', { status: 200, headers: { 'Content-Type': 'text/plain' } })))
      );
      const before = serialize(workerInstance.currentBuild);
      const box = await typeLink();
      await fireEvent.click(box.getByRole('button', { name: 'Import' }));
      await vi.waitFor(() => expect(box.getByRole('alert').textContent).toBe(`Import failed: ${NOT_A_CODE}`));
      expect(serialize(workerInstance.currentBuild)).toBe(before);
    });

    it("reports the site's error status the proxy passed through, and leaves the build alone", async () => {
      vi.stubGlobal(
        'fetch',
        vi.fn(() => Promise.resolve(new Response('pobb.in answered 404', { status: 404 })))
      );
      const before = serialize(workerInstance.currentBuild);
      const box = await typeLink();
      await fireEvent.click(box.getByRole('button', { name: 'Import' }));
      await vi.waitFor(() => expect(box.getByRole('alert').textContent).toMatch(/^Import failed: .*pobb\.in.*404/));
      expect(serialize(workerInstance.currentBuild)).toBe(before);
    });
  });

  // ImportTab.lua:609-628, 701-705
  it('a character JSON is imported through the worker with no confirm, and its message shown', async () => {
    const json = await readFile(resolve(process.cwd(), '../testdata/import/sample_character.json'), 'utf8');
    const call = vi.spyOn(workerInstance, 'ImportItemsAndSkills').mockResolvedValue({ ok: true, text: 'Items and skills successfully imported.' });
    const tree = vi.spyOn(workerInstance, 'ImportPassiveTreeAndJewels').mockResolvedValue({ ok: true, text: 'Passive tree and jewels successfully imported.' });
    const before = serialize(workerInstance.currentBuild);
    goto.mockClear();
    const { box } = await mount();
    const field = box.getByRole('textbox', { name: /To import a build/ });
    await fireEvent.input(field, { target: { value: json } });
    expect(document.getElementById(field.getAttribute('aria-describedby')!)!.textContent).toBe('JSON is valid');
    await fireEvent.click(box.getByRole('button', { name: 'Import' }));
    await vi.waitFor(() =>
      expect(box.getByRole('status').textContent).toBe('Items and skills successfully imported. Passive tree and jewels successfully imported.')
    );
    expect(within(document.body).queryByRole('dialog', { name: 'Build Import' })).toBeNull();
    expect(call).toHaveBeenCalledTimes(1);
    expect(JSON.parse(call.mock.calls[0][0])).toEqual(JSON.parse(json));
    // ImportTab.lua:703
    expect(tree).toHaveBeenCalledTimes(1);
    expect(tree.mock.calls[0][0]).toBe(call.mock.calls[0][0]);
    expect(call.mock.invocationCallOrder[0]).toBeLessThan(tree.mock.invocationCallOrder[0]);
    expect(serialize(workerInstance.currentBuild)).toBe(before);
    expect(goto).not.toHaveBeenCalled();
    call.mockRestore();
    tree.mockRestore();
  });

  // ImportTab.lua:946-953 (PoB-PoE2): a PoE2 character's passives carry its weapon-set specialisations
  it.each([false, true])('a PoE2 character JSON is refused with the flag %s, and the build left alone', async (on) => {
    const json = JSON.stringify({
      character: { name: 'Two', class: 'Sorceress', level: 1, equipment: [], jewels: [], passives: { hashes: [], specialisations: { set1: [], set2: [] } } }
    });
    poe2Enabled.set(on);
    const call = vi.spyOn(workerInstance, 'ImportItemsAndSkills');
    const tree = vi.spyOn(workerInstance, 'ImportPassiveTreeAndJewels');
    const before = serialize(workerInstance.currentBuild);
    const { box } = await mount();
    await fireEvent.input(box.getByRole('textbox', { name: /To import a build/ }), { target: { value: json } });
    await fireEvent.click(box.getByRole('button', { name: 'Import' }));
    await vi.waitFor(() => expect(box.getByRole('alert').textContent).toBe(`Import failed: ${on ? POE2_CHARACTER : POE2_CHARACTER_DISABLED}`));
    expect(call).not.toHaveBeenCalled();
    expect(tree).not.toHaveBeenCalled();
    expect(serialize(workerInstance.currentBuild)).toBe(before);
    poe2Enabled.set(false);
    call.mockRestore();
    tree.mockRestore();
  });

  // Build.lua:2222-2231
  it.each([
    ['hello', 'EOF'],
    ['<PathOfBuilding', 'XML syntax error on line 1: unexpected EOF']
  ])('a code that inflates to something other than a build (%s) says its XML is unreadable', async (xml, why) => {
    vi.stubGlobal('Blob', NodeBlob);
    const before = serialize(workerInstance.currentBuild);
    const { box } = await mount();
    await fireEvent.input(box.getByRole('textbox', { name: /To import a build/ }), { target: { value: types.pob.CompressEncode(xml).unwrap() } });
    await fireEvent.click(box.getByRole('radio', { name: /Import as a new build/ }));
    const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    await vi.waitFor(() => expect(go.disabled).toBe(false));
    await fireEvent.click(go);
    await vi.waitFor(() => expect(box.getByRole('alert').textContent).toBe(`Import failed: ${unreadableBuild(why)}`));
    expect(serialize(workerInstance.currentBuild)).toBe(before);
  });

  // ImportTab.lua:321-472, 813-1020
  describe('the account import', () => {
    const list = [
      { name: 'Zed', realm: 'pc', class: 'Necromancer', league: 'Standard', level: 90 },
      { name: 'Ann', realm: 'pc', class: 'Deadeye', league: 'Mercenaries', level: 95 }
    ];
    const serveApi = (answers: Record<string, () => Response>) => {
      const f = vi.fn((url: string) => Promise.resolve(answers[/\/api\/poe\/(\w+)\?/.exec(url)?.[1] ?? '']?.() ?? new Response('', { status: 500 })));
      vi.stubGlobal('fetch', f);
      return f;
    };
    const section = async () => {
      const { view } = await mount();
      return within(view.getByRole('region', { name: 'Character import' }));
    };
    const start = async (chars: Awaited<ReturnType<typeof section>>, account: string) => {
      await fireEvent.input(chars.getByRole('textbox', { name: /Account name/ }), { target: { value: account } });
      await fireEvent.click(chars.getByRole('button', { name: 'Start' }));
    };
    afterEach(async () => {
      vi.unstubAllGlobals();
      localStorage.clear();
      const { accountSession, resetAccountSession } = await import('$lib/shell/account_session.svelte');
      resetAccountSession();
      accountSession.lastLeague = undefined;
    });

    it('an account another tab adds to the history shows in this one', async () => {
      const chars = await section();
      const others = ['Other#1111', ...(JSON.parse(localStorage.getItem('import:accounts') ?? '[]') as string[])];
      localStorage.setItem('import:accounts', JSON.stringify(others));
      window.dispatchEvent(new StorageEvent('storage', { key: 'import:accounts' }));
      const pick = await vi.waitFor(() => chars.getByRole<HTMLSelectElement>('combobox', { name: 'Account history' }));
      expect([...pick.options].map((o) => o.textContent)).toContain('Other#1111');
      localStorage.setItem('import:accounts', JSON.stringify(others.slice(1)));
      window.dispatchEvent(new StorageEvent('storage', { key: 'import:accounts' }));
    });

    it('needs the discriminator, says the private-profile message, and writes nothing', async () => {
      const f = serveApi({ characters: () => new Response('', { status: 403 }) });
      const before = serialize(workerInstance.currentBuild);
      const chars = await section();
      await fireEvent.input(chars.getByRole('textbox', { name: /Account name/ }), { target: { value: 'Vilsol' } });
      expect(chars.getByRole<HTMLButtonElement>('button', { name: 'Start' }).disabled).toBe(true);
      expect(chars.getByText('Missing discriminator e.g. #1234')).toBeTruthy();
      await start(chars, 'Vilsol#3184');
      await vi.waitFor(() => expect(chars.getByRole('status').textContent).toBe('Character import status: Account profile is private or does not exist.'));
      expect(f).toHaveBeenCalledTimes(1);
      expect(f.mock.calls[0][0]).toBe('/api/poe/characters?realm=pc&account=Vilsol%233184');
      expect(serialize(workerInstance.currentBuild)).toBe(before);
    });

    it("lists the account's characters by league, remembers the account, and imports the tree after the overwrite confirm", async () => {
      const f = serveApi({
        characters: () => Response.json(list),
        profile: () => Response.json({ accountName: 'Vilsol-3184' }),
        passives: () => new Response('{"hashes":[1,2]}')
      });
      const tree = vi.spyOn(workerInstance, 'ImportAccountPassiveTree').mockResolvedValue({ ok: true, text: 'Passive tree and jewels successfully imported.' });
      outputs.set({ PointsUsed: [12, 0, 0, 0] } as never);
      const chars = await section();
      await start(chars, 'vilsol#3184');
      await vi.waitFor(() => expect(chars.getByRole('status').textContent).toBe('Character import status: Character list successfully retrieved.'));
      expect(JSON.parse(localStorage.getItem('import:accounts') ?? '[]')).toEqual(['Vilsol#3184']);
      const pick = chars.getByRole<HTMLSelectElement>('combobox', { name: 'Character' });
      expect([...pick.options].map((o) => o.textContent)).toEqual(['Ann — Deadeye lvl 95']);
      await fireEvent.change(chars.getByRole('combobox', { name: 'League' }), { target: { value: '' } });
      expect([...pick.options].map((o) => o.textContent)).toEqual(['Ann — Deadeye lvl 95 in Mercenaries', 'Zed — Necromancer lvl 90 in Standard']);
      await fireEvent.change(pick, { target: { value: 'Zed' } });
      await fireEvent.click(chars.getByRole('button', { name: 'Passive Tree and Jewels' }));
      const dialog = await vi.waitFor(() => within(document.body).getByRole('dialog', { name: 'Character Import' }));
      expect(tree).not.toHaveBeenCalled();
      await fireEvent.click(within(dialog).getByRole('button', { name: 'Import' }));
      await vi.waitFor(() => expect(chars.getByRole('status').textContent).toBe('Character import status: Passive tree and jewels successfully imported.'));
      expect(f.mock.calls.at(-1)?.[0]).toBe('/api/poe/passives?realm=pc&account=Vilsol%233184&character=Zed');
      expect(tree).toHaveBeenCalledWith(JSON.stringify(list[0]), '{"hashes":[1,2]}', true);
      // ImportTab.lua:441, 1925-1931
      const { unsavedName } = await import('$lib/shell/build_name.svelte');
      expect(unsavedName.predefined).toBe('Vilsol - Zed');
      tree.mockRestore();
      outputs.set(undefined);
    });

    it('returning to the page selects the character chosen in a league other than the default one', async () => {
      serveApi({ characters: () => Response.json(list), profile: () => Response.json({ accountName: 'Vilsol-3184' }) });
      let chars = await section();
      await start(chars, 'vilsol#3184');
      await vi.waitFor(() => expect(chars.getByRole('status').textContent).toBe('Character import status: Character list successfully retrieved.'));
      expect(chars.getByRole<HTMLSelectElement>('combobox', { name: 'League' }).value).toBe('Mercenaries');
      await fireEvent.change(chars.getByRole('combobox', { name: 'League' }), { target: { value: 'Standard' } });
      expect(chars.getByRole<HTMLSelectElement>('combobox', { name: 'Character' }).value).toBe('Zed');
      const { accountSession } = await import('$lib/shell/account_session.svelte');
      await vi.waitFor(() => expect(accountSession.selected?.name).toBe('Zed'));
      cleanup();
      chars = await section();
      expect(chars.getByRole<HTMLSelectElement>('combobox', { name: 'League' }).value).toBe('Standard');
      expect(chars.getByRole<HTMLSelectElement>('combobox', { name: 'Character' }).value).toBe('Zed');
      expect(accountSession.selected?.name).toBe('Zed');
    });

    it("imports the items and skills with PoB's three boxes, and says GGG's error", async () => {
      serveApi({
        characters: () => Response.json(list),
        profile: () => new Response('pathofexile.com answered 403 (challenge)', { status: 403 }),
        items: () => new Response('{"items":[]}')
      });
      const items = vi.spyOn(workerInstance, 'ImportAccountItems').mockResolvedValue({ ok: true, text: 'Items and skills successfully imported.' });
      const chars = await section();
      await start(chars, 'vilsol#3184');
      await vi.waitFor(() => expect(chars.getByRole('button', { name: 'Items and Skills' })).toBeTruthy());
      await fireEvent.click(chars.getByRole('checkbox', { name: 'Ignore weapon swap' }));
      await fireEvent.click(chars.getByRole('button', { name: 'Items and Skills' }));
      await vi.waitFor(() => expect(chars.getByRole('status').textContent).toBe('Character import status: Items and skills successfully imported.'));
      expect(items).toHaveBeenCalledWith(JSON.stringify(list[1]), '{"items":[]}', true, true, true);
      serveApi({ items: () => new Response('', { status: 429, headers: { 'Retry-After': '30' } }) });
      await fireEvent.click(chars.getByRole('button', { name: 'Items and Skills' }));
      await vi.waitFor(() =>
        expect(chars.getByRole('status').textContent).toBe('Character import status: Error importing character data, try again (Response code: 429)')
      );
      expect(items).toHaveBeenCalledTimes(1);
      items.mockRestore();
    });
  });
});

describe('opening the Import / Export dialog', () => {
  const modal = async () => render((await import('./AppImportModal.svelte')).default);
  const dialogOf = () => within(document.body).getByRole<HTMLDialogElement>('dialog', { name: 'Import / Export' });

  it("the top bar's Import / Export opens it over the page on Import, with the code field focused; Esc closes it and returns focus", async () => {
    await loadCorpusBuild(BUILD);
    const bar = await topBar();
    await modal();
    goto.mockClear();
    const link = bar.getByRole('link', { name: 'Import / Export' });
    link.focus();
    await fireEvent.click(link);
    const dlg = await vi.waitFor(() => dialogOf());
    expect(dlg.open).toBe(true);
    expect(goto, 'no page change').not.toHaveBeenCalled();
    expect(within(dlg).getByRole('tab', { name: 'Import' }).getAttribute('aria-selected')).toBe('true');
    expect(within(dlg).queryByRole('region', { name: 'Build sharing' })).toBeNull();
    await vi.waitFor(() => expect(document.activeElement).toBe(within(dlg).getByRole('textbox', { name: /^To import a build/ })));
    await fireEvent.keyDown(dlg, { key: 'Escape' });
    expect(dlg.open).toBe(false);
    expect(document.activeElement).toBe(link);
  });

  it('Ctrl+I opens it, and its Export tab shows the code and the share controls', async () => {
    await loadCorpusBuild(BUILD);
    await topBar();
    await modal();
    await fireEvent.keyDown(window, { key: 'i', ctrlKey: true });
    const dlg = await vi.waitFor(() => dialogOf());
    expect(dlg.open).toBe(true);
    await fireEvent.click(within(dlg).getByRole('tab', { name: 'Export' }));
    const sharing = within(within(dlg).getByRole('region', { name: 'Build sharing' }));
    await vi.waitFor(() => expect(sharing.getByRole<HTMLTextAreaElement>('textbox', { name: 'Build code' }).value).toBe(workerInstance.ExportCode()));
    expect(sharing.getByRole('button', { name: 'Share' })).toBeTruthy();
    expect(within(dlg).queryByRole('region', { name: 'Build import' })).toBeNull();
    expect(within(dlg).getByRole('checkbox', { name: /^Export support/ })).toBeTruthy();
  });

  it("the phone frame's Import/Export opens it instead of a page", async () => {
    await loadCorpusBuild(BUILD);
    const { opening } = await import('$lib/shell/build_name.svelte');
    opening.modFlag = false;
    const bar = render((await import('$lib/shell/AppTopBar.svelte')).default, { props: { phone: true } });
    await modal();
    goto.mockClear();
    await fireEvent.click(bar.getByRole('button', { name: 'More' }));
    const more = within(await vi.waitFor(() => within(document.body).getByRole('navigation', { name: 'More pages' })));
    await fireEvent.click(more.getByRole('link', { name: 'Import/Export' }));
    expect((await vi.waitFor(() => dialogOf())).open).toBe(true);
    expect(goto).not.toHaveBeenCalled();
  });
});
