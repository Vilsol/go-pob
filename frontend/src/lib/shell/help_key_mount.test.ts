// @vitest-environment jsdom
// Build.lua:311-315, Main.lua:1415-1423

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import type { TreeFixture, TreeNodeFx } from '$lib/design/fixtures';
import type { CanvasApi } from '$lib/design/tree';

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

const worker = vi.hoisted(() => ({
  GetUndoHistoryWhenRecorded: () => Promise.resolve({ steps: [], undone: 0, max: 50 }),
  BuildInfo: () => Promise.resolve({ Main: { Version: 'test' } }),
  GetLoadouts: () => Promise.resolve({ loadouts: [], selected: -1 }),
  GameDataVersion: () => Promise.resolve('3.29'),
  GetTreeVersion: () => Promise.resolve('3_29')
}));
const syncWrap = new Proxy(worker, { get: (t, k: string) => (k in t ? t[k as keyof typeof t] : () => Promise.resolve(undefined)) });
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto: vi.fn(() => Promise.resolve()), pushState: vi.fn() }));
const page = vi.hoisted(() => ({ state: {}, url: new URL('http://localhost/') }));
vi.mock('$app/state', () => ({ page }));

const help = readFileSync(resolve(__dirname, '../../../static/help.txt'), 'utf8');
vi.stubGlobal(
  'fetch',
  vi.fn(() => Promise.resolve(new Response(help)))
);

const AppTopBar = (await import('./AppTopBar.svelte')).default;
const TreePage = (await import('$lib/design/TreePage.svelte')).default;
const { currentBuild } = await import('$lib/global');
currentBuild.set({} as never);

const helpNav = async () => {
  const dlg = await within(document.body).findByRole('dialog', { name: 'About go-pob' });
  return within(dlg).findByRole('navigation', { name: 'Help sections' });
};
const current = async () => {
  const nav = await helpNav();
  await vi.waitFor(() => expect(nav.querySelector('[aria-current="true"]')).not.toBeNull());
  return nav.querySelector('[aria-current="true"]')!.textContent!.trim();
};

describe('F1 opens the help at the current page', () => {
  const PAGES: [string, string][] = [
    ['tree', 'Tree tab'],
    ['skills', 'Skills tab'],
    ['items', 'Items tab'],
    ['calcs', 'Calcs tab'],
    ['configuration', 'Config tab'],
    ['notes', 'Notes tab'],
    ['party', 'Party tab'],
    ['import', 'Import tab'],
    ['compare', 'Compare tab']
  ];
  it.each(PAGES)('/%s opens "%s"', async (route, title) => {
    // Import / Export is a dialog over the page its address opens on
    const { importDialog } = await import('./import_modal.svelte');
    importDialog.open = route === 'import';
    page.url = new URL(`http://localhost/${route === 'import' ? 'tree' : route}`);
    render(AppTopBar);
    await fireEvent.keyDown(document.body, { key: 'F1' });
    expect(await current()).toMatch(new RegExp(`^\\d+\\. ${title}$`));
    importDialog.open = false;
  });
});

describe('F1 over a hovered passive node', () => {
  const node: TreeNodeFx = { id: 7, k: 'n', x: 0.5, y: 0.5, name: 'Heart of Oak', pts: 1 };
  const tree: TreeFixture = {
    version: '3_29',
    classes: ['Witch'],
    ascendancies: ['None'],
    specs: ['Main'],
    activeSpec: 0,
    aspect: 1,
    radius: 0.1,
    nodes: [node],
    edges: [],
    tips: {},
    sockets: []
  };
  const canvas = createRawSnippet((api: () => CanvasApi) => ({
    render: () => '<button data-testid="node">node</button>',
    setup: (el: Element) => el.addEventListener('mouseenter', () => api().enter(node))
  }));

  // Main.lua:476-485
  it('opens the node on the wiki and not the help, with the top bar listening first', async () => {
    page.url = new URL('http://localhost/tree');
    const open = vi.spyOn(window, 'open').mockReturnValue(null);
    try {
      render(AppTopBar);
      const view = render(TreePage, { tree, className: 'Witch', ascendancy: 'None', alloc: {}, canvas });
      await fireEvent.mouseEnter(view.getByTestId('node'));
      await fireEvent.keyDown(document.body, { key: 'F1' });
      await new Promise((r) => setTimeout(r, 0));
      expect(open.mock.calls.map((c) => c[0])).toEqual(['https://www.poewiki.net/wiki/Heart_of_Oak']);
      expect(document.querySelector('dialog[open]'), 'the help opened as well').toBeNull();
    } finally {
      open.mockRestore();
    }
  });
});
