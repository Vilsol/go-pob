// @vitest-environment jsdom

import { describe, expect, it, beforeAll, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, waitFor, within } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { createRawSnippet } from 'svelte';

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

// the PoE1 worker answers nothing: no PoE1 build is open
const poe1 = vi.hoisted(() => ({}) as Record<string, ReturnType<typeof vi.fn>>);
const syncWrap = new Proxy(poe1, { get: (t, k: string) => t[k] ?? (() => Promise.resolve(undefined)) });
vi.mock('$lib/go/worker', () => ({ syncWrap, syncWorker: undefined }));
vi.mock('$app/navigation', () => ({ goto: vi.fn(() => Promise.resolve()), pushState: vi.fn() }));
vi.mock('$app/state', () => ({ page: { state: {}, url: new URL('http://localhost/tree') } }));

// a PoE2 view as the worker hands it back; the values are the frame's inputs, not PoB-PoE2 output under test
const calc = (over: Record<string, unknown> = {}) => ({
  className: 'Sorceress',
  ascendClassName: 'None',
  level: 40,
  output: { Life: 683, Mana: 231 },
  skillFlags: {},
  skillData: {},
  socketGroups: [{ label: 'Spark, Fireball', skills: ['Spark', 'Fireball'] }],
  mainSocketGroup: 1,
  mainActiveSkill: 1,
  configInputs: {},
  tree: {
    alloc: [],
    allocMode: {},
    text: {},
    className: 'Sorceress',
    ascendClassBaseName: '',
    used: 12,
    ascUsed: 2,
    jewelRadiusIndex: {},
    progress: {
      level: 13,
      act: 'Act 1',
      extra: 0,
      normalPassives: 12,
      usedMax: 24,
      weaponSet1Used: 0,
      weaponSet2Used: 0,
      weaponSetMax: 0,
      ascUsed: 2,
      ascMax: 0,
      warnings: []
    }
  },
  treeSpecs: { labels: ['Default'], active: 1 },
  ...over
});
const w2 = vi.hoisted(() => ({
  SetMainActiveSkill: vi.fn(),
  TreeData: vi.fn(() => new Promise(() => {})),
  TreeClassList: vi.fn(() => Promise.resolve([{ label: 'Sorceress', classId: 7, ascendancies: ['None', 'Stormweaver'] }]))
}));
vi.mock('$lib/go/worker2', () => ({ poe2Worker: () => Promise.resolve(w2) }));

const { currentBuild, currentBuildPath, poe2Build } = await import('$lib/global');
const { game } = await import('$lib/go/engine');
const { poe2Engine, poe2Error } = await import('$lib/go/engine_poe2.svelte');
const { opening } = await import('./build_name.svelte');
const AppTopBar = (await import('./AppTopBar.svelte')).default;
const AppSidebar = (await import('./AppSidebar.svelte')).default;
const AppTree2 = (await import('./AppTree2.svelte')).default;

afterEach(() => {
  poe2Build.set(undefined);
  poe2Error.set(undefined);
  currentBuildPath.current = undefined;
  opening.modFlag = false;
});

describe('a PoE2 build in the app frame', () => {
  it('is the open game, and only a PoE1 build opening replaces it', () => {
    poe2Build.set(calc());
    expect(get(game)).toBe('poe2');
    currentBuild.set({} as never);
    expect(get(game)).toBe('poe1');
    expect(get(poe2Build)).toBeUndefined();
    currentBuild.set(undefined);
    expect(get(game)).toBeUndefined();
  });

  it('the top bar links only the pages PoE2 has', () => {
    poe2Build.set(calc());
    const bar = render(AppTopBar);
    const tabs = [...bar.container.querySelectorAll<HTMLAnchorElement>('nav a.tab')];
    expect(tabs.map((t) => [t.textContent?.replace(/\d/g, '').trim(), new URL(t.href).pathname])).toEqual([
      ['Tree', '/tree'],
      ['Skills', '/skills'],
      ['Items', '/items'],
      ['Calcs', '/calcs'],
      ['Config', '/configuration'],
      ['Notes', '/notes'],
      ['Party', '/party'],
      ['Compare', '/compare']
    ]);
  });

  // the Notes, Party and Compare pages and the Import / Export dialog serve a PoE2 build
  it("the phone frame's More lists the pages PoE2 has", async () => {
    poe2Build.set(calc());
    const bar = render(AppTopBar, { props: { phone: true } });
    await fireEvent.click(bar.getByRole('button', { name: 'More' }));
    const more = await waitFor(() => within(document.body).getByRole('navigation', { name: 'More pages' }));
    expect(
      within(more)
        .getAllByRole('link')
        .map((l) => l.textContent?.trim())
    ).toEqual(['Notes', 'Party', 'Compare', 'Import/Export']);
  });

  it('a route PoE2 has no page for says so', async () => {
    poe2Build.set(calc());
    const GamePage = (await import('./GamePage.svelte')).default;
    const page = render(GamePage, { props: { route: 'timeless', children: createRawSnippet(() => ({ render: () => '<span>PoE1 page</span>' })) } });
    expect(page.getByRole('status').textContent).toBe('The timeless page is not available for Path of Exile 2 builds yet.');
  });

  it('opens saved, an edit leaves it unsaved, and leaving without saving closes it', async () => {
    poe2Build.set(calc());
    const bar = render(AppTopBar);
    await bar.findByRole('button', { name: 'Save (no unsaved changes)' });
    w2.SetMainActiveSkill.mockResolvedValueOnce(calc({ mainActiveSkill: 2 }));
    poe2Engine.setMainSkill('MainActiveSkill', 2);
    await bar.findByRole('button', { name: 'Save (unsaved changes)' });
    expect(w2.SetMainActiveSkill).toHaveBeenCalledWith(2);
    expect(get(game)).toBe('poe2');

    await fireEvent.click(bar.getByRole('link', { name: 'Back to builds' }));
    const prompt = within(await bar.findByRole('dialog', { name: 'Save Changes' }));
    await fireEvent.click(prompt.getByRole('button', { name: "Don't Save" }));
    await waitFor(() => expect(get(poe2Build)).toBeUndefined());
    expect(get(game)).toBeUndefined();
  });

  it('the sidebar shows the minion over the player and the main skill pickers', () => {
    poe2Build.set(calc({ minion: { Life: 120 } }));
    expect(get(poe2Engine.sidebar)?.sections.map((s) => s.header)).toEqual(['Minion', 'Player']);
    const side = render(AppSidebar);
    const aside = within(side.getByRole('complementary', { name: 'Main skill and stats' }));
    const stats = within(aside.getByRole('region', { name: 'Build stats' }));
    expect(stats.getByText('Minion')).toBeTruthy();
    expect(stats.getByText('Player')).toBeTruthy();
    expect(aside.getByRole<HTMLSelectElement>('combobox', { name: 'Main skill' }).value).toBe('0');
    expect([...aside.getByRole<HTMLSelectElement>('combobox', { name: 'Active skill' }).options].map((o) => o.text)).toEqual(['Spark', 'Fireball']);
  });

  it('the tree body shows the level, class and points, and an edit error reaches the page', async () => {
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
    poe2Build.set(calc());
    const page = render(AppTree2);
    await vi.waitFor(() => expect((page.getByRole('combobox', { name: 'Class' }) as HTMLSelectElement).value).toBe('Sorceress'));
    expect([...(page.getByRole('combobox', { name: 'Ascendancy' }) as HTMLSelectElement).options].map((o) => o.text)).toEqual(['None', 'Stormweaver']);
    expect((page.getByRole('spinbutton', { name: 'Character level' }) as HTMLInputElement).value).toBe('40');
    expect(page.getByRole('status', { name: 'Points used' }).textContent?.replace(/\s+/g, ' ').trim()).toBe(
      'Points 12 / 24 · Weapon set 1 0 / 0 · Weapon set 2 0 / 0 · Ascendancy 2 / 0'
    );
    w2.SetMainActiveSkill.mockRejectedValueOnce(new Error('no PoE2 build is open'));
    poe2Engine.setMainSkill('MainActiveSkill', 2);
    expect((await page.findByRole('alert')).textContent).toBe('no PoE2 build is open');
  });
});
