// @vitest-environment jsdom
import { expect, describe, it, vi, afterEach, beforeAll } from 'vitest';
import { render, cleanup, fireEvent, within, waitFor } from '@testing-library/svelte';

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

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

const poe1Export = vi.fn(() => Promise.resolve('poe1-code'));
vi.mock('$lib/go/worker', () => ({ syncWrap: { ExportCode: poe1Export } }));
vi.mock('$app/navigation', () => ({ goto: () => Promise.resolve() }));

const CODE = 'eJyzCSjKz0ksyS-yq8jNsbPRB1MANiQGFg==';
const exportCode = vi.fn((stats: unknown) => Promise.resolve(stats ? CODE : ''));
vi.mock('$lib/go/worker2', () => ({ poe2Worker: () => Promise.resolve({ ExportCode: exportCode, GetPartyExport: () => ({ on: true, parts: [] }) }) }));

const VIEW = { className: 'Sorceress', ascendClassName: 'None', level: 40, output: { Life: 683, Mana: 231 }, skillFlags: {}, socketGroups: [] };

const mount = async () => {
  const { poe2Build } = await import('$lib/global');
  poe2Build.set(VIEW as never);
  (await import('./import_modal.svelte')).openImport('export');
  const view = render((await import('./AppImportModal.svelte')).default);
  const dialog = within(view.getByRole('dialog', { name: 'Import / Export' }));
  return { dialog, panel: within(dialog.getByRole('tabpanel')) };
};

describe('the Export tab for a PoE2 build', () => {
  // ImportTab.lua:135-137 (PoB-PoE2): the code is the PoE2 worker's, made from the build as Save writes it
  it("shows the PoE2 worker's code, made with the stat rows Save lists", async () => {
    const { panel } = await mount();
    const box = panel.getByRole<HTMLTextAreaElement>('textbox', { name: 'Build code' });
    await vi.waitFor(() => expect(box.value).toBe(CODE));
    expect(exportCode).toHaveBeenCalledWith({
      player: [
        { Stat: 'Life', Value: 683 },
        { Stat: 'Mana', Value: 231 }
      ],
      fullDPS: [],
      minion: []
    });
    expect(poe1Export).not.toHaveBeenCalled();
  });

  // ImportTab.lua:154-186, BuildSiteTools.lua:19-35 (PoB-PoE2)
  it("offers PoB-PoE2's share sites and uploads the code to the chosen one", async () => {
    const posted: [string, string][] = [];
    vi.stubGlobal(
      'fetch',
      (url: string, init: { body: string }) => (
        posted.push([url, init.body]),
        Promise.resolve(new Response('https://maxroll.gg/poe2/pob/abc', { headers: { 'Content-Type': 'text/plain; charset=utf-8' } }))
      )
    );
    const { panel } = await mount();
    await vi.waitFor(() => expect(panel.getByRole<HTMLTextAreaElement>('textbox', { name: 'Build code' }).value).toBe(CODE));
    const select = panel.getByRole<HTMLSelectElement>('combobox', { name: 'Share via' });
    expect([...select.options].map((o) => o.value)).toEqual(['Maxroll (PoE2)', 'pobb.in', 'poe.ninja (PoE2)', 'poe2db.tw']);
    await fireEvent.change(select, { target: { value: 'Maxroll (PoE2)' } });
    await fireEvent.click(panel.getByRole('button', { name: 'Share' }));
    await vi.waitFor(() => expect(panel.getByRole<HTMLInputElement>('textbox', { name: 'Share link' }).value).toBe('https://maxroll.gg/poe2/pob/abc'));
    expect(posted).toEqual([['/api/share?site=Maxroll2', CODE]]);
    vi.unstubAllGlobals();
  });

  // ImportTab.lua:138-141 (PoB-PoE2): the box shows the build's Export Support
  it("shows the PoE2 build's Export Support", async () => {
    const { panel } = await mount();
    const box = panel.getByRole<HTMLInputElement>('checkbox', { name: /^Export support/ });
    expect(box.disabled).toBe(false);
    await waitFor(() => expect(box.checked).toBe(true));
  });
});
