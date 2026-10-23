// @vitest-environment jsdom

import './pointer_event_polyfill';
import { describe, expect, it, afterEach, vi } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import ItemsBoard from './ItemsBoard.svelte';
import type { ItemSummary } from './fixtures';

afterEach(cleanup);

const wand: ItemSummary = {
  name: 'Doom Twig',
  base: 'Imbued Wand',
  rarity: 'rare',
  sockets: 'B-B-R',
  implicits: [],
  explicits: [],
  flask: false,
  fits: ['Weapon 1'],
  cells: [1, 3]
};

const setup = () => {
  const onedit = vi.fn();
  const r = render(ItemsBoard, {
    slots: { 'Weapon 1': { id: 1, item: wand } },
    flasks: {},
    calcFlasks: {},
    fitting: () => [],
    jewels: [],
    jewelOptions: [],
    onedit,
    onflask: () => {},
    onequip: () => {}
  });
  const tile = r.getByRole('button', { name: 'Weapon 1: Doom Twig' });
  const caption = () => tile.querySelector('[data-slot-caption]');
  return { ...r, onedit, tile, caption };
};

const tap = async (el: HTMLElement) => {
  await fireEvent(el, new PointerEvent('pointerdown', { pointerType: 'touch', bubbles: true }));
  await fireEvent.mouseEnter(el);
  await fireEvent.focus(el);
  await fireEvent.click(el);
};

describe('ItemsBoard slot captions', () => {
  it('shows the name and sockets only while the slot is hovered, and keeps the accessible name', async () => {
    const { tile, caption, getByRole } = setup();
    expect(caption()).toBeNull();
    await fireEvent.mouseEnter(tile);
    expect(caption()?.textContent).toContain('Doom Twig');
    expect(caption()?.querySelector('[data-sockets]')?.getAttribute('data-sockets')).toBe('B-B-R');
    await fireEvent.mouseLeave(tile);
    expect(caption()).toBeNull();
    expect(getByRole('button', { name: 'Weapon 1: Doom Twig' })).toBe(tile);
  });

  it('a first tap shows the caption, the second opens the item', async () => {
    const { tile, caption, onedit } = setup();
    await tap(tile);
    expect(caption()).not.toBeNull();
    expect(onedit).not.toHaveBeenCalled();
    await tap(tile);
    expect(onedit).toHaveBeenCalledWith('Weapon 1');
  });

  it('focus shows the caption and a click from the keyboard or a mouse acts at once', async () => {
    const { tile, caption, onedit } = setup();
    await fireEvent.focus(tile);
    expect(caption()).not.toBeNull();
    await fireEvent.click(tile);
    expect(onedit).toHaveBeenCalledTimes(1);
    await fireEvent(tile, new PointerEvent('pointerdown', { pointerType: 'mouse', bubbles: true }));
    await fireEvent.click(tile);
    expect(onedit).toHaveBeenCalledTimes(2);
  });
});

describe('ItemsBoard socket overlay', () => {
  const at = (root: Element) =>
    [...root.querySelectorAll<HTMLElement>('[data-socket]')].map((s) => [parseFloat(s.style.left) * 16, parseFloat(s.style.top) * 16].map(Math.round));

  it('draws a 1-wide item’s sockets down one column with its links, only on hover', async () => {
    const { tile } = setup();
    expect(tile.parentElement!.querySelector('[data-item-sockets]')).toBeNull();
    await fireEvent.mouseEnter(tile);
    const overlay = tile.parentElement!.querySelector('[data-item-sockets]')!;
    // Weapon 1 is 168x322; a 1x3 wand fits 144x298 at 99.3 per cell
    expect(at(overlay)).toEqual([
      [50, 50],
      [50, 149],
      [50, 248]
    ]);
    expect([...overlay.querySelectorAll('[data-socket-link]')].map((l) => l.getAttribute('data-socket-link'))).toEqual(['0', '1']);
    expect([...overlay.querySelectorAll('[role="img"]')].map((s) => s.getAttribute('aria-label'))).toEqual(['Blue socket', 'Blue socket', 'Red socket']);
  });

  it('sizes sockets and links as the game does per inventory cell: 35px sockets, 15x38 links between them', async () => {
    const { tile } = setup();
    await fireEvent.mouseEnter(tile);
    const overlay = tile.parentElement!.querySelector('[data-item-sockets]')!;
    const cell = 298 / 3;
    const px = (v: string) => Math.round(parseFloat(v) * 160) / 10;
    const links = [...overlay.querySelectorAll<SVGElement>('[data-socket-link]')].map((l) =>
      [l.style.left, l.style.top, l.style.width, l.style.height].map(px)
    );
    expect(links).toEqual([
      [49.7, 99.3, 31.4, 79.5],
      [49.7, 198.7, 31.4, 79.5]
    ]);
    const scale = /scale\(([\d.]+)\)/.exec(overlay.querySelector<HTMLElement>('[data-socket] > span')!.style.transform)![1];
    expect(Number(scale) * 24).toBeCloseTo((cell * 35) / 47.4645, 3);
  });

  it('snakes a 2-wide item’s sockets in rows of two, centred on the art', async () => {
    const chest: ItemSummary = { ...wand, name: 'Carcass Jack', base: 'Varnished Coat', sockets: 'R-R-G-G B-B', cells: [2, 3] };
    const r = render(ItemsBoard, {
      slots: { 'Body Armour': { id: 2, item: chest } },
      flasks: {},
      calcFlasks: {},
      fitting: () => [],
      jewels: [],
      jewelOptions: [],
      onedit: () => {},
      onflask: () => {},
      onequip: () => {}
    });
    const tile = r.getByRole('button', { name: 'Body Armour: Carcass Jack' });
    await fireEvent.mouseEnter(tile);
    const overlay = tile.parentElement!.querySelector('[data-item-sockets]')!;
    // Body Armour is 168x252; a 2x3 coat fits 144x228 at 72 per cell
    expect(at(overlay)).toEqual([
      [36, 36],
      [108, 36],
      [108, 108],
      [36, 108],
      [36, 180],
      [108, 180]
    ]);
    expect([...overlay.querySelectorAll('[data-socket-link]')].map((l) => l.getAttribute('data-socket-link'))).toEqual(['0', '1', '2', '4']);
  });
});

describe('ItemsBoard socketed gems', () => {
  it("puts the slot's gems in its sockets in order, and hovering one shows its gem tooltip", async () => {
    const gemtip = vi.fn((group: number, gem: number) => Promise.resolve([{ Text: `tip ${group}.${gem}`, Separator: false }]));
    const r = render(ItemsBoard, {
      slots: { 'Weapon 1': { id: 1, item: wand } },
      flasks: {},
      calcFlasks: {},
      fitting: () => [],
      jewels: [],
      jewelOptions: [],
      onedit: () => {},
      onflask: () => {},
      onequip: () => {},
      socketed: {
        'Weapon 1': [
          { group: 3, gems: [{ index: 0, name: 'Fireball' }] },
          { group: 5, gems: [{ index: 1, name: 'Spell Echo' }] }
        ]
      },
      gemtip
    });
    const tile = r.getByRole('button', { name: 'Weapon 1: Doom Twig' });
    await fireEvent.mouseEnter(tile);
    const gems = [...tile.parentElement!.querySelectorAll<HTMLElement>('[data-socket]')].map((s) =>
      s.querySelector('[data-socket-gem]')?.getAttribute('data-socket-gem')
    );
    expect(gems).toEqual(['Fireball', 'Spell Echo', undefined]);
    await fireEvent.mouseEnter(tile.parentElement!.querySelector('[data-socket-gem="Spell Echo"]')!);
    expect(gemtip).toHaveBeenCalledWith(5, 1);
    await vi.waitFor(() => expect(document.querySelector('[aria-label="Gem tooltip"]')?.textContent).toContain('tip 5.1'));
    await fireEvent.mouseLeave(tile.parentElement!.querySelector('[data-socket-gem="Spell Echo"]')!);
    expect(document.querySelector('[aria-label="Gem tooltip"]')).toBeNull();
  });

  it('reaches the gems from the focused slot with the arrow keys, showing each gem tooltip, and Escape returns to the slot', async () => {
    const gemtip = vi.fn((group: number, gem: number) => Promise.resolve([{ Text: `tip ${group}.${gem}`, Separator: false }]));
    const r = render(ItemsBoard, {
      slots: { 'Weapon 1': { id: 1, item: wand } },
      flasks: {},
      calcFlasks: {},
      fitting: () => [],
      jewels: [],
      jewelOptions: [],
      onedit: () => {},
      onflask: () => {},
      onequip: () => {},
      socketed: {
        'Weapon 1': [
          { group: 3, gems: [{ index: 0, name: 'Fireball' }] },
          { group: 5, gems: [{ index: 1, name: 'Spell Echo' }] }
        ]
      },
      gemtip
    });
    const tile = r.getByRole('button', { name: 'Weapon 1: Doom Twig' });
    const tipText = () => document.querySelector('[aria-label="Gem tooltip"]')?.textContent;
    tile.focus();
    await tick();
    await fireEvent.keyDown(tile, { key: 'ArrowDown' });
    const fireball = tile.parentElement!.querySelector<HTMLElement>('[data-socket-gem="Fireball"]')!;
    expect(document.activeElement).toBe(fireball);
    await vi.waitFor(() => expect(tipText()).toContain('tip 3.0'));
    expect(fireball.getAttribute('aria-describedby')).toBe(document.querySelector('[aria-label="Gem tooltip"]')!.closest('[id]')!.id);
    await fireEvent.keyDown(fireball, { key: 'ArrowDown' });
    expect(document.activeElement?.getAttribute('data-socket-gem')).toBe('Spell Echo');
    await vi.waitFor(() => expect(tipText()).toContain('tip 5.1'));
    await fireEvent.keyDown(document.activeElement!, { key: 'Escape' });
    expect(document.activeElement).toBe(tile);
    expect(tipText()).toBeUndefined();
    tile.blur();
    await vi.waitFor(() => expect(tile.parentElement!.querySelector('[data-item-sockets]')).toBeNull());
  });
});

describe('ItemsBoard abyss jewels', () => {
  it("shows the jewel equipped in the item's abyssal socket there; a first tap shows its tooltip, the second opens it", async () => {
    const onedit = vi.fn();
    const belt: ItemSummary = { ...wand, name: 'Dire Cord', base: 'Stygian Vise', sockets: 'A', cells: [2, 1] };
    const eye: ItemSummary = { ...wand, name: 'Dread Gaze', base: 'Murderous Eye Jewel', sockets: '', cells: [1, 1] };
    const r = render(ItemsBoard, {
      slots: { Belt: { id: 1, item: belt }, 'Belt Abyssal Socket 1': { id: 2, item: eye } },
      flasks: {},
      calcFlasks: {},
      fitting: () => [],
      jewels: [],
      jewelOptions: [],
      onedit,
      onflask: () => {},
      onequip: () => {}
    });
    const tile = r.getByRole('button', { name: 'Belt: Dire Cord' });
    await fireEvent.mouseEnter(tile);
    const jewel = r.getByRole('button', { name: 'Dread Gaze' });
    expect(jewel.closest('[data-socket]')?.querySelector('[aria-label="Abyssal socket"]')).not.toBeNull();
    await tap(jewel);
    expect(onedit).not.toHaveBeenCalled();
    expect(document.getElementById(jewel.getAttribute('aria-describedby') ?? '')?.textContent).toContain('Dread Gaze');
    await tap(jewel);
    expect(onedit).toHaveBeenCalledWith('Belt Abyssal Socket 1');
  });
});
