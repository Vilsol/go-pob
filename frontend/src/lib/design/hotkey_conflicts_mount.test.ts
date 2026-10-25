// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';

afterEach(cleanup);

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
globalThis.matchMedia = ((q: string) => ({
  matches: false,
  media: q,
  addEventListener: () => {},
  removeEventListener: () => {}
})) as unknown as typeof matchMedia;
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

const checkForUpdate = vi.hoisted(() => vi.fn(() => Promise.resolve()));
vi.mock('$lib/pwa', async (orig) => ({ ...(await orig<typeof import('$lib/pwa')>()), checkForUpdate, startPwa: () => Promise.resolve() }));
vi.mock('$lib/go/worker', () => ({ syncWrap: null, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto: vi.fn(() => Promise.resolve()), pushState: vi.fn() }));
vi.mock('$app/state', () => ({ page: { state: {}, url: new URL('http://localhost/notes') } }));

const TopBar = (await import('./TopBar.svelte')).default;
const NotesPage = (await import('./NotesPage.svelte')).default;
const ItemEditor = (await import('./ItemEditor.svelte')).default;
const Layout = (await import('../../routes/+layout.svelte')).default;

const bar = { name: 'Build', tabs: [], current: '', loadouts: [], issues: [], logs: [], dirty: true };
const history = [{ action: 'Allocate node', page: 'Tree', when: 'just now' }];
const doc = { type: 'doc', content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Hello notes' }] }] };

const notes = async () => {
  const view = render(NotesPage, { doc });
  const editor = await vi.waitFor(() => {
    const el = view.container.querySelector<HTMLElement>('[contenteditable="true"]');
    expect(el).not.toBeNull();
    return el!;
  });
  editor.focus();
  let taken: boolean | undefined;
  editor.addEventListener('keydown', (e) => (taken = e.defaultPrevented));
  return { view, editor, taken: () => taken };
};
const ctrl = (el: Element, key: string, shiftKey = false) => fireEvent.keyDown(el, { key, keyCode: key.toUpperCase().charCodeAt(0), ctrlKey: true, shiftKey });

describe('a key the focused editor handles does nothing else', () => {
  // Launch.lua:158-161
  it('Ctrl+U in the notes underlines without checking for an update', async () => {
    render(Layout);
    const { editor, taken } = await notes();
    await ctrl(editor, 'u');
    expect(taken(), 'the editor did not take Ctrl+U').toBe(true);
    expect(checkForUpdate).not.toHaveBeenCalled();
    await ctrl(document.body, 'u');
    expect(checkForUpdate, 'Ctrl+U outside the notes').toHaveBeenCalledTimes(1);
  });

  // Build.lua:1171-1185
  it('Ctrl+Shift+S in the notes strikes through without saving; Ctrl+S still saves', async () => {
    const onsave = vi.fn();
    render(TopBar, { ...bar, history, onsave });
    const { editor, taken } = await notes();
    await ctrl(editor, 'S', true);
    expect(taken(), 'the editor did not take Ctrl+Shift+S').toBe(true);
    expect(onsave).not.toHaveBeenCalled();
    await ctrl(editor, 's');
    expect(onsave).toHaveBeenCalledTimes(1);
  });

  // ItemsTab.lua:988-993
  it('Ctrl+Z in the item editor undoes the edit, not the build', async () => {
    const onundo = vi.fn();
    render(TopBar, { ...bar, history, onundo });
    const item = {
      name: 'Doom Ring',
      base: 'Gold Ring',
      rarity: 'rare' as const,
      sockets: '',
      implicits: [],
      explicits: [{ text: '+20 to Strength', kind: 'magic' as const }],
      flask: false,
      fits: ['Ring 1']
    };
    const view = render(ItemEditor, { open: true, item, affixes: { tiers: [], sources: [], sorts: [], rows: [] } });
    const dlg = await view.findByRole('dialog');
    await fireEvent.click(within(dlg).getByRole('button', { name: 'Remove +20 to Strength' }));
    expect(within(dlg).queryByText('+20 to Strength')).toBeNull();
    const button = within(dlg).getByRole('region', { name: 'Item' }).parentElement!.parentElement!.querySelector('button')!;
    button.focus();
    await ctrl(button, 'z');
    expect(within(dlg).getByText('+20 to Strength'), 'the edit was not undone').toBeTruthy();
    expect(onundo).not.toHaveBeenCalled();
  });
});
