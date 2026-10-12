// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi, type MockInstance } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { Blob as NodeBlob } from 'node:buffer';

afterEach(cleanup);

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

/* ImportTab.lua:648-653 */
const confirmImport = async () => {
  const dialog = await vi.waitFor(() => within(document.body).getByRole('dialog', { name: 'Build Import' }));
  expect(dialog.textContent).toContain('will erase ALL existing data for this build');
  await fireEvent.click(within(dialog).getByRole('button', { name: 'Import' }));
};

vi.mock('$lib/go/worker', async () => {
  const { remoteWorker } = await import('../../../harness/component/go_runtime');
  return { syncWrap: remoteWorker(), syncWorker: undefined };
});

const { bootWorker, loadCorpusBuild } = await import('../../../harness/component/go_runtime');
const { currentBuild, outputs } = await import('$lib/global');
const { workerInstance } = await import('$lib/go/sync_worker');
const types = await import('$lib/types');

const BUILD = '0008-necromancer-vaal-summon-skeletons.xml';
const OTHER = '0005-occultist-vaal-double-strike.xml';

const serialize = (b: unknown) => new TextDecoder().decode(types.builds.SerializeBuild(b as never).unwrap());
let other: { code: string; xml: string };
let tick: MockInstance;
const writeText = vi.fn((text: string) => Promise.resolve(void text));

const mount = async () => {
  const Page = (await import('./+page.svelte')).default;
  const view = render(Page);
  const code = view.getByRole('textbox', { name: 'Build code' }) as HTMLTextAreaElement;
  await vi.waitFor(() => expect(code.value).not.toBe(''));
  return { view, code, box: within(view.getByRole('region', { name: 'Build import' })) };
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

describe('import/+page.svelte', () => {
  it('mounting alone writes nothing and does not Tick', async () => {
    const before = serialize(workerInstance.currentBuild);
    tick.mockClear();
    await mount();
    await new Promise((r) => setTimeout(r, 50));
    expect(serialize(workerInstance.currentBuild)).toBe(before);
    expect(tick).not.toHaveBeenCalled();
  }, 20_000);

  it("shows Go's code for the build it holds, and re-reads it after a recalculation", async () => {
    const { code } = await mount();
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
    const { view } = await mount();
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
    const { view, code } = await mount();
    writeText.mockClear();
    await fireEvent.click(view.getByRole('button', { name: 'Copy code' }));
    expect(writeText).toHaveBeenCalledWith(code.value);
    await vi.waitFor(() => expect(view.getByRole('status').textContent).toContain('Copied'));
  });

  it('an imported code replaces the build Go holds, then Ticks', async () => {
    const { box } = await mount();
    const field = box.getByRole('textbox', { name: /^To import a build/ });
    const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    expect(go.disabled).toBe(true);
    await fireEvent.input(field, { target: { value: other.code } });
    await vi.waitFor(() =>
      expect(field.getAttribute('aria-describedby') && document.getElementById(field.getAttribute('aria-describedby')!)!.textContent).toBe('Code is valid')
    );
    expect(go.disabled).toBe(false);
    expect(box.getByRole<HTMLInputElement>('radio', { name: /Import to this build/ }).checked).toBe(true);
    expect(serialize(workerInstance.currentBuild), 'typing alone must not write').not.toBe(other.xml);
    tick.mockClear();
    goto.mockClear();
    await fireEvent.click(go);
    expect(serialize(workerInstance.currentBuild), 'nothing before the confirm').not.toBe(other.xml);
    await confirmImport();
    await vi.waitFor(() => expect(serialize(workerInstance.currentBuild)).toBe(other.xml));
    expect(tick).toHaveBeenCalledWith('ImportCode');
    await vi.waitFor(() => expect(goto).toHaveBeenCalledWith(expect.stringMatching(/\/tree$/)));
  });

  it('Cancel in the confirm imports nothing; Enter in the field imports a valid code (ImportTab.lua:671-675)', async () => {
    const { box } = await mount();
    const before = serialize(workerInstance.currentBuild);
    const field = box.getByRole('textbox', { name: /^To import a build/ });
    await fireEvent.input(field, { target: { value: other.code } });
    await vi.waitFor(() => expect(box.getByRole<HTMLButtonElement>('button', { name: 'Import' }).disabled).toBe(false));
    await fireEvent.keyDown(field, { key: 'Enter' });
    const dialog = await vi.waitFor(() => within(document.body).getByRole('dialog', { name: 'Build Import' }));
    await fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));
    await new Promise((r) => setTimeout(r, 200));
    expect(serialize(workerInstance.currentBuild)).toBe(before);
  });

  // ImportTab.lua:663-666
  it('Import to a new build opens the code without the current save path, then Ticks', async () => {
    const { currentBuildPath } = await import('$lib/global');
    currentBuildPath.current = 'Builds/kept.xml';
    const { box } = await mount();
    const field = box.getByRole('textbox', { name: /^To import a build/ });
    await fireEvent.input(field, { target: { value: other.code } });
    const asNew = box.getByRole<HTMLInputElement>('radio', { name: /Import as a new build/ });
    expect(asNew.disabled, 'the new-build mode is live').toBe(false);
    await fireEvent.click(asNew);
    const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    await vi.waitFor(() => expect(go.disabled).toBe(false));
    tick.mockClear();
    await fireEvent.click(go);
    await vi.waitFor(() => expect(serialize(workerInstance.currentBuild)).toBe(other.xml));
    expect(tick).toHaveBeenCalledWith('ImportCode');
    await vi.waitFor(() => expect(currentBuildPath.current, 'the new build still saves over the old one').toBe(''));
    const { unsavedName } = await import('$lib/shell/build_name.svelte');
    expect(unsavedName.imported, 'named "Imported build" (ImportTab.lua:665)').toBe(true);
  });

  it('a code that inflates but is no build says so, and leaves the build alone', async () => {
    const { box } = await mount();
    const before = serialize(workerInstance.currentBuild);
    const field = box.getByRole('textbox', { name: /^To import a build/ });
    await fireEvent.input(field, { target: { value: types.pob.CompressEncode('not a build').unwrap() } });
    const go = box.getByRole<HTMLButtonElement>('button', { name: 'Import' });
    await vi.waitFor(() => expect(go.disabled).toBe(false));
    await fireEvent.click(go);
    await confirmImport();
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

    it("downloads the site's raw code, then imports it and Ticks", async () => {
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
      await vi.waitFor(() => expect(fetch).toHaveBeenCalledWith('https://pobb.in/pob/aBcD1234'));
      const busy = box.getByRole<HTMLButtonElement>('button', { name: 'Retrieving paste..' });
      expect(busy.disabled).toBe(true);
      answer(new Response(`${other.code}\n`, { status: 200 }));
      await confirmImport();
      await vi.waitFor(() => expect(serialize(workerInstance.currentBuild)).toBe(other.xml));
      expect(tick).toHaveBeenCalledWith('ImportCode');
      await vi.waitFor(() => expect(box.getByRole('button', { name: 'Import' })).toBeTruthy());
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

    it("reports the site's error status, and leaves the build alone", async () => {
      vi.stubGlobal(
        'fetch',
        vi.fn(() => Promise.resolve(new Response('gone', { status: 404 })))
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
    const before = serialize(workerInstance.currentBuild);
    goto.mockClear();
    const { box } = await mount();
    const field = box.getByRole('textbox', { name: /To import a build/ });
    await fireEvent.input(field, { target: { value: json } });
    expect(document.getElementById(field.getAttribute('aria-describedby')!)!.textContent).toBe('JSON is valid');
    await fireEvent.click(box.getByRole('button', { name: 'Import' }));
    await vi.waitFor(() => expect(box.getByRole('status').textContent).toBe('Items and skills successfully imported.'));
    expect(within(document.body).queryByRole('dialog', { name: 'Build Import' })).toBeNull();
    expect(call).toHaveBeenCalledTimes(1);
    expect(JSON.parse(call.mock.calls[0][0])).toEqual(JSON.parse(json));
    expect(serialize(workerInstance.currentBuild)).toBe(before);
    expect(goto).not.toHaveBeenCalled();
    call.mockRestore();
  });
});
