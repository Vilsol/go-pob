// @vitest-environment jsdom
// Main.lua:1032-1035

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within, waitFor } from '@testing-library/svelte';
import { get } from 'svelte/store';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
});
afterEach(() => {
  const seen = pageErrors.splice(0, pageErrors.length);
  expect(seen, 'the mounted dialog threw').toEqual([]);
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

const scheme = vi.hoisted(() => {
  const listeners = new Set<() => void>();
  const s = { light: false, listeners };
  globalThis.matchMedia = ((q: string) => ({
    get matches() {
      return q.includes('light') ? s.light : false;
    },
    media: q,
    addEventListener: (_: string, f: () => void) => listeners.add(f),
    removeEventListener: (_: string, f: () => void) => listeners.delete(f)
  })) as unknown as typeof matchMedia;
  return s;
});

const fake = {
  SetShowFlavourText: vi.fn(() => Promise.resolve()),
  SetItemModDebug: vi.fn(() => Promise.resolve()),
  SetDefaultItemAffixQuality: vi.fn(() => Promise.resolve()),
  GameDataVersion: vi.fn(() => Promise.resolve('3.29')),
  BuildInfo: vi.fn(() => Promise.resolve({ Deps: { 0: { Path: 'github.com/Vilsol/go-pob-data', Version: 'v0.5.4' }, __crystallineHandle: 7 } })),
  ClearDataCache: vi.fn(() => Promise.resolve(3)),
  ExportBuilds: vi.fn(() =>
    Promise.resolve([
      { path: 'Top build.xml', data: new TextEncoder().encode('<PathOfBuilding/>') },
      { path: 'Leagues/Spark.xml', data: new TextEncoder().encode('<PathOfBuilding2/>') }
    ])
  )
};
vi.mock('$lib/go/worker', () => ({ syncWrap: fake, syncWorker: undefined }));
const pwa = vi.hoisted(() => ({ checkForUpdate: vi.fn(() => Promise.resolve()) }));
vi.mock('$lib/pwa', () => pwa);

const { fontScaling, zoomSensitivity, markBackendAsLoaded } = await import('$lib/global');
const { theme, showAnimations, showStatBars } = await import('$lib/display/appearance');
const { allAffixSliders } = await import('$lib/design/items');
const { showAllItemAffixes, defaultItemAffixQuality } = await import('$lib/display/item_options');
const AppSettings = (await import('./AppSettings.svelte')).default;

const open = () => {
  const view = render(AppSettings, { open: true });
  return within(view.getByRole('dialog', { name: 'Settings' }));
};
const root = document.documentElement;

describe('AppSettings', () => {
  it('Font scaling sizes the root font on release and is kept; Reset puts back 12 pt', async () => {
    const s = open();
    const slider = s.getByRole('slider', { name: 'Font scaling' });
    const shown = () => slider.closest('[data-setting]')!.textContent;
    expect(root.style.fontSize).toBe('16px');
    await fireEvent.input(slider, { target: { value: '18' } });
    expect(shown()).toContain('18pt');
    expect(get(fontScaling)).toBe(12);
    expect(root.style.fontSize).toBe('16px');
    await fireEvent.input(slider, { target: { value: '17' } });
    expect(root.style.fontSize).toBe('16px');
    await fireEvent.change(slider, { target: { value: '18' } });
    expect(get(fontScaling)).toBe(18);
    expect(root.style.fontSize).toBe('24px');
    expect(localStorage.getItem('options:fontScaling')).toBe('18');
    await fireEvent.input(slider, { target: { value: '6' } });
    await fireEvent.change(slider, { target: { value: '6' } });
    expect(root.style.fontSize).toBe('8px');
    expect(shown()).toContain('6pt');
    await fireEvent.click(s.getByRole('button', { name: 'Reset this section' }));
    expect(root.style.fontSize).toBe('16px');
  });

  it('Theme switches the token set on <html>; Follow system tracks prefers-color-scheme', async () => {
    const s = open();
    const pick = s.getByRole('combobox', { name: 'Theme' });
    expect(root.dataset.theme).toBe('dark');
    await fireEvent.change(pick, { target: { value: 'Light' } });
    expect(root.dataset.theme).toBe('light');
    expect(root.style.colorScheme).toBe('light');
    expect(localStorage.getItem('options:theme')).toBe('Light');
    await fireEvent.change(pick, { target: { value: 'High contrast' } });
    expect(root.dataset.theme).toBe('contrast');
    await fireEvent.change(pick, { target: { value: 'Follow system' } });
    expect(root.dataset.theme).toBe('dark');
    scheme.light = true;
    for (const f of scheme.listeners) f();
    expect(root.dataset.theme).toBe('light');
    scheme.light = false;
    for (const f of scheme.listeners) f();
    theme.set('Dark');
  });

  it('Show animations off marks <html> data-motion="off"', async () => {
    const s = open();
    const box = s.getByRole('checkbox', { name: 'Show animations' });
    expect((box as HTMLInputElement).checked).toBe(true);
    expect(root.dataset.motion).toBeUndefined();
    await fireEvent.click(box);
    expect(get(showAnimations)).toBe(false);
    expect(root.dataset.motion).toBe('off');
    expect(localStorage.getItem('options:showAnimations')).toBe('false');
    showAnimations.set(true);
    expect(root.dataset.motion).toBeUndefined();
  });

  it('Show stat bars is on by default, and off is kept; Reset turns it back on', async () => {
    const s = open();
    const box = s.getByRole('checkbox', { name: 'Show stat bars' });
    expect((box as HTMLInputElement).checked).toBe(true);
    await fireEvent.click(box);
    expect(get(showStatBars)).toBe(false);
    expect(localStorage.getItem('options:showStatBars')).toBe('false');
    await fireEvent.click(s.getByRole('button', { name: 'Reset this section' }));
    expect(get(showStatBars)).toBe(true);
  });

  it('Show all item affix sliders (Main.lua:121, on by default) sets the item editor’s option', async () => {
    const s = open();
    await fireEvent.click(s.getByRole('tab', { name: /^Items/ }));
    const box = s.getByRole('checkbox', { name: 'Show all item affix sliders' });
    expect((box as HTMLInputElement).checked).toBe(true);
    expect(get(allAffixSliders)).toBe(true);
    await fireEvent.click(box);
    expect(get(allAffixSliders)).toBe(false);
    expect(localStorage.getItem('options:showAllItemAffixes')).toBe('false');
    showAllItemAffixes.set(true);
  });

  it('Default items view (Board / List) is kept as the Items page’s opening layout', async () => {
    const { itemsLayout } = await import('$lib/display/appearance');
    const s = open();
    await fireEvent.click(s.getByRole('tab', { name: /^Items/ }));
    const pick = s.getByRole<HTMLSelectElement>('combobox', { name: 'Default items view' });
    expect(pick.value).toBe('Board');
    await fireEvent.change(pick, { target: { value: 'List' } });
    expect(get(itemsLayout)).toBe('list');
    expect(localStorage.getItem('options:itemsLayout')).toBe('list');
    await fireEvent.click(s.getByRole('button', { name: 'Reset this section' }));
    expect(get(itemsLayout)).toBe('board');
  });

  it('Wide lists on a phone (Cards / Table) is kept for the build list and the Items list', async () => {
    const { phoneTables } = await import('$lib/display/appearance');
    const s = open();
    await fireEvent.click(s.getByRole('tab', { name: /^General/ }));
    const pick = s.getByRole<HTMLSelectElement>('combobox', { name: 'Wide lists on a phone' });
    expect(pick.value).toBe('Cards');
    await fireEvent.change(pick, { target: { value: 'Table' } });
    expect(get(phoneTables)).toBe('table');
    expect(localStorage.getItem('options:phoneTables')).toBe('table');
    await fireEvent.click(s.getByRole('button', { name: 'Reset this section' }));
    expect(get(phoneTables)).toBe('cards');
  });

  it('Crafting card layout (Stepped sections / Tabs) is kept for the paste panel’s crafting card', async () => {
    const { craftCardLayout } = await import('$lib/display/appearance');
    const s = open();
    await fireEvent.click(s.getByRole('tab', { name: /^Items/ }));
    const pick = s.getByRole<HTMLSelectElement>('combobox', { name: 'Crafting card layout' });
    expect(pick.value).toBe('Stepped sections');
    await fireEvent.change(pick, { target: { value: 'Tabs' } });
    expect(get(craftCardLayout)).toBe('tabs');
    expect(localStorage.getItem('options:craftCardLayout')).toBe('tabs');
    await fireEvent.click(s.getByRole('button', { name: 'Reset this section' }));
    expect(get(craftCardLayout)).toBe('stepped');
  });

  it('Default item affix quality (Main.lua:1099-1106) is a 0-1 slider in 0.05 steps, kept, and given to Go', async () => {
    const s = open();
    const slider = s.getByRole<HTMLInputElement>('slider', { name: 'Default item affix quality' });
    expect([slider.min, slider.max, slider.step, slider.value]).toEqual(['0', '1', '0.05', '0.5']);
    await new Promise((r) => setTimeout(r, 50));
    expect(fake.SetDefaultItemAffixQuality).not.toHaveBeenCalled();
    get(markBackendAsLoaded)();
    await waitFor(() => expect(fake.SetDefaultItemAffixQuality).toHaveBeenLastCalledWith(0.5));
    await fireEvent.input(slider, { target: { value: '0.85' } });
    expect(get(defaultItemAffixQuality)).toBe(0.85);
    expect(localStorage.getItem('options:defaultItemAffixQuality')).toBe('0.85');
    await waitFor(() => expect(fake.SetDefaultItemAffixQuality).toHaveBeenLastCalledWith(0.85));
    await fireEvent.click(s.getAllByRole('button', { name: 'Reset this section' })[0]);
    expect(get(defaultItemAffixQuality)).toBe(0.5);
  });

  it('Data and storage: the game data, a check for newer data, the cache size, an in-page confirmed clear, a .zip of every build', async () => {
    Object.defineProperty(navigator, 'storage', { configurable: true, value: { estimate: () => Promise.resolve({ usage: 38.4 * 1024 * 1024 }) } });
    const saved: { name: string; blob: Blob }[] = [];
    URL.createObjectURL = vi.fn(() => 'blob:x');
    URL.revokeObjectURL = vi.fn();
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
      saved.push({ name: this.download, blob: (URL.createObjectURL as ReturnType<typeof vi.fn>).mock.calls.at(-1)![0] as Blob });
    });
    const confirm = vi.spyOn(window, 'confirm');

    const s = open();
    await fireEvent.click(s.getByRole('tab', { name: /^Data and storage/ }));
    await waitFor(() => expect(s.getByText('3.29 · go-pob-data v0.5.4')).toBeTruthy());
    await waitFor(() => expect(s.getByText('38.4 MB')).toBeTruthy());

    await fireEvent.click(s.getByRole('button', { name: 'Check for newer game data' }));
    expect(pwa.checkForUpdate).toHaveBeenCalledTimes(1);

    await fireEvent.click(s.getByRole('button', { name: 'Clear cached data' }));
    expect(fake.ClearDataCache).not.toHaveBeenCalled();
    expect(s.getByRole('alert').textContent).toMatch(/builds are kept/i);
    await fireEvent.click(s.getByRole('button', { name: 'Keep' }));
    expect(s.queryByRole('button', { name: 'Keep' })).toBeNull();
    await fireEvent.click(s.getByRole('button', { name: 'Clear cached data' }));
    await fireEvent.click(s.getByRole('button', { name: 'Clear cached data' }));
    expect(fake.ClearDataCache).toHaveBeenCalledTimes(1);

    await fireEvent.click(s.getByRole('button', { name: 'Export all builds' }));
    await waitFor(() => expect(saved.length).toBe(1));
    expect(fake.ExportBuilds).toHaveBeenCalledTimes(1);
    expect(saved[0].name).toMatch(/^go-pob-builds-\d{4}-\d{2}-\d{2}\.zip$/);
    const bytes = await new Promise<Uint8Array>((done) => {
      const r = new FileReader();
      r.onload = () => done(new Uint8Array(r.result as ArrayBuffer));
      r.readAsArrayBuffer(saved[0].blob);
    });
    expect([...bytes.slice(0, 4)]).toEqual([0x50, 0x4b, 0x03, 0x04]);
    expect(new TextDecoder().decode(bytes)).toContain('Leagues/Spark.xml');
    expect(confirm).not.toHaveBeenCalled();
    click.mockRestore();
  });

  it('Reset all settings asks in the sheet, then puts every section back to its default', async () => {
    fontScaling.set(15);
    theme.set('Light');
    showAnimations.set(false);
    zoomSensitivity.set(8);
    const s = open();
    await fireEvent.click(s.getByRole('tab', { name: /^Data and storage/ }));
    await fireEvent.click(s.getByRole('button', { name: 'Reset all settings' }));
    expect(s.getByRole('alert').textContent).toMatch(/every setting/i);
    expect(get(fontScaling)).toBe(15);
    await fireEvent.click(s.getByRole('button', { name: 'Reset all settings' }));
    expect([get(fontScaling), get(theme), get(showAnimations), get(zoomSensitivity)]).toEqual([12, 'Dark', true, 5]);
    expect(root.dataset.theme).toBe('dark');
  });
});
