// @vitest-environment jsdom

import { describe, expect, it, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent, within } from '@testing-library/svelte';
import type { Tree } from '$lib/skill_tree/types';

afterEach(cleanup);

if (!HTMLDialogElement.prototype.showModal) {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
    this.dispatchEvent(new Event('close'));
  };
}

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver;

const row = (seed: number) => ({
  Label: `# Ghost Dance: 10 | ${seed}:`,
  Seed: seed,
  Total: 2,
  SocketID: 10,
  SocketLabel: '# Ghost Dance: 10',
  Nodes: [{ ID: 'karui_notable_add_life', TargetNodeNames: ['Herbalism'], TotalWeight: 2 }]
});

const w = vi.hoisted(() => {
  const ok = <A extends unknown[], T>(f: (...a: A) => T) => vi.fn((...a: A) => Promise.resolve(f(...a)));
  return {
    GetTimelessOptions: ok(() => ({
      types: [
        { ID: 1, Label: 'Glorious Vanity', Name: 'vaal' },
        { ID: 2, Label: 'Lethal Pride', Name: 'karui' }
      ],
      conquerors: {
        1: ['Any', 'Doryani (Corrupted Soul)', 'Xibaqua (Divine Flesh)', 'Ahuana (Immortal Ambition)'],
        2: ['Any', 'Kaom (Strength of Blood)', 'Rakiata (Tempered by War)', 'Akoya (Chainbreaker)']
      } as Record<number, string[]>,
      devotion: ['Any', 'Totem Damage'],
      tradeTypes: ['Instant buyout', 'Any (includes offline)'],
      realms: ['PC', 'Sony']
    })),
    GetTimelessFallbackModes: ok(() => [{ label: 'Sort by Life', stat: 'Life' }]),
    GetTimelessData: ok(() => ({})),
    SetTimelessData: vi.fn(),
    GetTimelessLists: ok(() => ({
      sockets: [
        { ID: -1, Label: 'All Sockets', Keystone: 'Multi-Socket Search' },
        { ID: 10, Label: '# Ghost Dance: 10', Keystone: 'Ghost Dance' }
      ],
      mods: [
        { Label: '...', ID: '', Descriptions: [], TotalMod: false },
        { Label: 'Total Strength', ID: 'total_strength', Descriptions: ['This is a hybrid node containing all additions to Strength'], TotalMod: true },
        {
          Label: 'Flesh of the Kaom                    5% increased maximum Life +10 to Strength',
          ID: 'karui_notable_add_life',
          Descriptions: ['5% increased maximum Life', '+10 to Strength'],
          TotalMod: false
        }
      ]
    })),
    GetTimelessProtectable: ok(() => []),
    GetTreeVersion: ok(() => '3_29'),
    GetPassiveNodes: ok(() => [10, 11]),
    GetBuildInfo: ok(() => ({ className: 'Witch' })),
    CalculateAllocationPaths: ok(() => ({})),
    GenerateTimelessFallback: ok(() => 'karui_notable_add_life, 3, 0, 0'),
    TimelessSearch: ok(() => ({ count: 2, desired: [{ ID: 'karui_notable_add_life', DisplayName: 'Flesh of the Kaom' }] })),
    TimelessSearchProgress: ok(() => 0.5),
    CancelTimelessSearch: vi.fn(),
    TimelessRows: ok((start: number, n: number) => [row(10001), row(10002)].slice(start, start + n)),
    AddTimelessResult: ok(() => 'jewel'),
    TimelessTradeURL: ok(() => ({
      URL: 'https://www.pathofexile.com/trade/search/Standard',
      SelIndex: 1,
      HighlightIndex: 2,
      LastSearchStart: 1,
      LastSearchCount: 2,
      LastSearchPresent: true
    }))
  };
});
vi.mock('$lib/go/worker', () => ({ syncWrap: w, syncWorker: undefined }));
vi.mock('$lib/components/skill-tree/renderer', () => ({
  createRenderer: () => ({ kind: 'canvas', setView: () => {}, setState: () => {}, complete: () => true, drawListArt: () => undefined, destroy: () => {} })
}));
Object.defineProperty(HTMLElement.prototype, 'offsetWidth', { configurable: true, get: () => 800 });
Object.defineProperty(HTMLElement.prototype, 'offsetHeight', { configurable: true, get: () => 560 });

const { skillTree, skillTreeVersion } = await import('$lib/skill_tree');
const TimelessDialog = (await import('./TimelessDialog.svelte')).default;

const tree = {
  classes: [],
  constants: { orbitRadii: [0, 82], skillsPerOrbit: [1, 6], classes: {}, characterAttributes: {}, PSSCentreInnerRadius: 0 },
  groups: { 1: { x: 0, y: 0, orbits: [0, 1], nodes: ['10', '11', '12'] } },
  nodes: {
    10: { skill: 10, name: 'Jewel Socket', group: 1, orbit: 0, orbitIndex: 0, isJewelSocket: true, out: ['11'] },
    11: { skill: 11, name: 'Strength', group: 1, orbit: 1, orbitIndex: 0, out: ['12'] },
    12: { skill: 12, name: 'Herbalism', group: 1, orbit: 1, orbitIndex: 3, isNotable: true }
  },
  min_x: -100,
  min_y: -100,
  max_x: 100,
  max_y: 100
} as unknown as Tree;

const saved = () => w.SetTimelessData.mock.lastCall?.[0] as Record<string, string>;

describe('the live timeless wizard (TreeTab.lua:1161-2863)', () => {
  it('drives every step through the worker: jewel, socket, requirements, fallback, search, results', async () => {
    skillTree.set(tree);
    skillTreeVersion.set('3_29');
    const open = vi.spyOn(window, 'open').mockReturnValue(null);
    const v = render(TimelessDialog, { props: { open: true } });
    const next = () => fireEvent.click(v.getByRole('button', { name: 'Next' }));

    const jewels = within(await v.findByRole('group', { name: 'Jewels' }));
    await vi.waitFor(() => expect(w.GetTimelessLists).toHaveBeenCalledWith(1));
    await vi.waitFor(() => expect(w.GetPassiveNodes).toHaveBeenCalled());
    await fireEvent.click(jewels.getByRole('button', { name: /^Lethal Pride/ }));
    await vi.waitFor(() => expect(w.GetTimelessLists).toHaveBeenCalledWith(2));
    expect(saved().JewelTypeID).toBe('2');
    await fireEvent.click(within(v.getByRole('region', { name: 'Conqueror' })).getByRole('button', { name: /^Rakiata/ }));
    expect(saved().ConquerorTypeID).toBe('3');

    await next();
    const view = await v.findByRole('region', { name: 'Passive tree around the build' });
    expect(view.querySelector('canvas')).not.toBeNull();
    await fireEvent.click(within(v.getByRole('region', { name: 'Sockets' })).getByRole('button', { name: /^# Ghost Dance: 10/ }));
    expect(saved().JewelSocketID).toBe('10');
    const ring = await vi.waitFor(() => view.querySelector<HTMLElement>('.radius-ring[data-ring="10"]')!);
    expect(ring.style.getPropertyValue('--x')).toMatch(/px$/);
    expect(view.querySelector('[data-picked="10"]')).not.toBeNull();
    expect(view.querySelector('[data-conq="12"]')).not.toBeNull();
    await fireEvent.click(v.getByRole('checkbox', { name: /^Filter nodes/ }));
    expect(saved().SocketFilter).toBe('true');
    await vi.waitFor(() => expect(w.GetTimelessProtectable).toHaveBeenCalledWith(10));

    await next();
    const pick = v.getByRole('combobox', { name: 'Add a required stat' });
    await fireEvent.focus(pick);
    await fireEvent.input(pick, { target: { value: 'maximum life' } });
    const options = within(v.getByRole('listbox', { name: 'Add a required stat options' })).getAllByRole('option');
    expect(options.map((o) => o.textContent)).toEqual([expect.stringContaining('Flesh of the Kaom')]);
    await fireEvent.mouseDown(options[0]);
    expect(saved().SearchList).toBe('karui_notable_add_life, 1, 1, 0');
    await fireEvent.click(v.getByRole('checkbox', { name: 'Flesh of the Kaom required' }));
    expect(saved().SearchList).toBe('karui_notable_add_life, 1, 1, required');

    await next();
    await fireEvent.click(v.getByRole('button', { name: 'Generate' }));
    await vi.waitFor(() => expect(w.GenerateTimelessFallback).toHaveBeenCalledWith(2, 'Life'));
    await vi.waitFor(() => expect(saved().SearchListFallback).toBe('karui_notable_add_life, 3, 0, 0'));
    await fireEvent.click(v.getByRole('button', { name: 'Search' }));
    await vi.waitFor(() =>
      expect(w.TimelessSearch).toHaveBeenCalledWith(
        expect.objectContaining({
          JewelType: 2,
          ConquerorType: 3,
          SocketID: 10,
          SocketFilter: true,
          SearchList: 'karui_notable_add_life, 1, 1, required',
          SearchListFallback: 'karui_notable_add_life, 3, 0, 0',
          HasTotalMinimumWeight: false
        })
      )
    );

    const results = within(await v.findByRole('listbox', { name: 'Search results' }));
    await vi.waitFor(() =>
      expect(results.getAllByRole('option').map((o) => o.getAttribute('aria-label'))).toEqual(['# Ghost Dance: 10 | 10001:', '# Ghost Dance: 10 | 10002:'])
    );
    expect(v.getByText('2 results')).toBeTruthy();
    expect(within(v.getByRole('group', { name: 'Result preview' })).getByText('Flesh of the Kaom: Herbalism')).toBeTruthy();
    await fireEvent.dblClick(results.getAllByRole('option')[1]);
    await vi.waitFor(() => expect(w.AddTimelessResult).toHaveBeenCalledWith(1, 1, 1, 'Ghost Dance'));
    await fireEvent.click(v.getByRole('button', { name: 'Open Trade URL' }));
    await vi.waitFor(() => expect(w.TimelessTradeURL).toHaveBeenCalledWith(expect.objectContaining({ JewelType: 2, ConquerorType: 3, SelIndex: 1 })));
    await vi.waitFor(() => expect(open).toHaveBeenCalledWith('https://www.pathofexile.com/trade/search/Standard', '_blank', 'noopener'));
    expect(v.getByRole('button', { name: 'Open Next Trade URL' })).toBeTruthy();
    open.mockRestore();
  });
});
