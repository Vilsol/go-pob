// @vitest-environment jsdom
// Main.lua:1426-1428

import { afterEach, beforeAll, describe, expect, it } from 'vitest';
import { cleanup, fireEvent, render, within } from '@testing-library/svelte';
import LiveHelp from './stories/LiveHelp.svelte';

afterEach(cleanup);

const pageErrors: string[] = [];
beforeAll(() => {
  window.addEventListener('error', (e) => pageErrors.push(e.error ? String(e.error) : e.message));
  window.addEventListener('unhandledrejection', (e) => pageErrors.push(String(e.reason)));
  process.on('unhandledRejection', (r) => pageErrors.push(String(r)));
  if (!HTMLDialogElement.prototype.showModal) {
    HTMLDialogElement.prototype.showModal = function () {
      this.open = true;
    };
    HTMLDialogElement.prototype.close = function () {
      this.open = false;
      this.dispatchEvent(new Event('close'));
    };
  }
});
afterEach(async () => {
  await new Promise((r) => setTimeout(r, 0));
  expect(pageErrors.splice(0, pageErrors.length), 'the mounted page threw').toEqual([]);
});

const openHelp = async (section?: string) => {
  const view = render(LiveHelp, { section });
  const opener = view.getByRole('button', { name: 'Help' });
  opener.focus();
  await fireEvent.click(opener);
  const dlg = await view.findByRole('dialog', { name: 'About go-pob' });
  expect(within(dlg).getByRole('tab', { name: 'Help', selected: true })).toBeTruthy();
  return { view, opener, dlg: dlg as HTMLDialogElement };
};

describe('Help', () => {
  it('closes by ×', async () => {
    const { opener, dlg } = await openHelp('Loadouts');
    await fireEvent.click(within(dlg).getByRole('button', { name: 'Close' }));
    expect(dlg.open).toBe(false);
    expect(document.activeElement).toBe(opener);
  });

  it('closes by Escape', async () => {
    const { opener, dlg } = await openHelp();
    await fireEvent.keyDown(dlg, { key: 'Escape' });
    expect(dlg.open).toBe(false);
    expect(document.activeElement).toBe(opener);
  });

  it('closes by a click on the backdrop, not inside', async () => {
    const { opener, dlg } = await openHelp();
    await fireEvent.click(within(dlg).getByRole('heading', { name: 'Hotkeys (4)' }));
    expect(dlg.open).toBe(true);
    await fireEvent.click(dlg, { clientX: 5000, clientY: 5000 });
    expect(dlg.open).toBe(false);
    expect(document.activeElement).toBe(opener);
  });

  it('lists the sections and jumps to one', async () => {
    const { dlg } = await openHelp();
    const nav = within(dlg).getByRole('navigation', { name: 'Help sections' });
    const links = within(nav).getAllByRole('button');
    expect(links.map((b) => b.textContent?.trim())).toContain('7. Loadouts');
    await fireEvent.click(within(nav).getByRole('button', { name: '7. Loadouts' }));
    await new Promise((r) => setTimeout(r, 0));
    expect(within(nav).getByRole('button', { name: '7. Loadouts' }).getAttribute('aria-current')).toBe('true');
  });
});
