// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, fireEvent, cleanup, within } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { tick } from 'svelte';

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

// PoE2 is unlocked before the settings read it
vi.hoisted(() => {
  localStorage.setItem('options:poe2', 'true');
  globalThis.matchMedia = ((q: string) => ({
    matches: false,
    media: q,
    addEventListener: () => {},
    removeEventListener: () => {}
  })) as unknown as typeof matchMedia;
});

const fake = {
  SetShowFlavourText: vi.fn(() => Promise.resolve()),
  SetItemModDebug: vi.fn(() => Promise.resolve()),
  SetDefaultItemAffixQuality: vi.fn(() => Promise.resolve()),
  GameDataVersion: vi.fn(() => Promise.resolve('3.29')),
  BuildInfo: vi.fn(() => Promise.resolve({ Deps: {} })),
  ClearDataCache: vi.fn(() => Promise.resolve(0))
};
vi.mock('$lib/go/worker', () => ({ syncWrap: fake, syncWorker: undefined }));
vi.mock('$lib/pwa', () => ({ checkForUpdate: vi.fn(() => Promise.resolve()), clearArtCache: vi.fn(() => Promise.resolve(0)) }));
vi.mock('$lib/go/worker2', () => ({ clearPoe2DataCache: vi.fn(() => Promise.resolve(0)) }));

const { engineBoot } = await import('$lib/global');
const AppSettings = (await import('./AppSettings.svelte')).default;

const open = () => {
  const view = render(AppSettings, { open: true });
  return within(view.getByRole('dialog', { name: 'Settings' }));
};

describe('Which game engine to load first', () => {
  it('is Last used by default, Both is kept, follows another tab, and Reset puts back Last used', async () => {
    const s = open();
    await fireEvent.click(s.getByRole('tab', { name: /^General/ }));
    const pick = s.getByRole<HTMLSelectElement>('combobox', { name: 'Which game engine to load first' });
    expect([...pick.options].map((o) => o.value)).toEqual(['Last used', 'Both']);
    expect(pick.value).toBe('Last used');
    expect(get(engineBoot)).toBe('last');

    await fireEvent.change(pick, { target: { value: 'Both' } });
    expect(get(engineBoot)).toBe('both');
    expect(localStorage.getItem('options:engineBoot')).toBe('both');

    // another tab picks Last used
    localStorage.setItem('options:engineBoot', 'last');
    window.dispatchEvent(new StorageEvent('storage', { key: 'options:engineBoot' }));
    expect(get(engineBoot)).toBe('last');
    await tick();
    expect(pick.value).toBe('Last used');

    await fireEvent.change(pick, { target: { value: 'Both' } });
    await fireEvent.click(s.getByRole('button', { name: 'Reset this section' }));
    expect(get(engineBoot)).toBe('last');
  });
});
