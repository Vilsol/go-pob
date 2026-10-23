import { describe, expect, it } from 'vitest';
import { abyssalSlots, artBox, placeGems, socketGrid } from './item_sockets';
import { BOARD_CELL } from './items';

describe('artBox', () => {
  it('draws the art in its inventory proportions at the largest size that fits', () => {
    expect(artBox([1, 3], 144, 298)).toMatchObject({ w: 298 / 3, h: 298, cols: 1, rows: 3 });
    expect(artBox([2, 4], 144, 298)).toMatchObject({ w: 144, h: 288, cell: 72 });
    expect(artBox([2, 1], 144, 60)).toMatchObject({ w: 120, h: 60 });
  });
  it('treats an unknown size as one cell', () => {
    expect(artBox(undefined, 144, 60)).toMatchObject({ w: 60, h: 60, cols: 1, rows: 1 });
    expect(artBox([0, 0], 60, 144)).toMatchObject({ w: 60, h: 60 });
  });
});

describe('socketGrid', () => {
  const cells = (n: number, w: number) => socketGrid(n, w).at.map((p) => `${p.col},${p.row}`);
  it('runs down one column on a 1-wide item', () => {
    expect(cells(3, 1)).toEqual(['0,0', '0,1', '0,2']);
    expect(socketGrid(3, 1)).toMatchObject({ cols: 1, rows: 3 });
  });
  it('snakes in rows of two on a 2-wide item', () => {
    expect(cells(6, 2)).toEqual(['0,0', '1,0', '1,1', '0,1', '0,2', '1,2']);
    expect(cells(3, 2)).toEqual(['0,0', '1,0', '1,1']);
    expect(socketGrid(4, 2)).toMatchObject({ cols: 2, rows: 2 });
  });
  it('centres a lone socket on a 2-wide item', () => {
    expect(socketGrid(1, 2)).toMatchObject({ cols: 1, rows: 1 });
  });
});

describe('placeGems', () => {
  const g = (group: number, names: string[], source?: string) => ({ group, source, gems: names.map((name, index) => ({ index, name })) });
  it("fills the sockets in order with the slot's groups in list order", () => {
    const at = placeGems('R-G-B B-B', [g(0, ['Fireball', 'Spell Echo']), g(2, ['Clarity', 'Flame Dash', 'Frostblink'])]);
    expect(at.map((x) => x && `${x.group}.${x.gem}`)).toEqual(['0.0', '0.1', '2.0', '2.1', '2.2']);
  });
  it('leaves sockets past the last gem empty and drops gems past the last socket', () => {
    expect(placeGems('R-R-R', [g(0, ['Cleave'])]).map((x) => x?.name)).toEqual(['Cleave', undefined, undefined]);
    expect(placeGems('R', [g(0, ['Cleave', 'Ruthless'])]).map((x) => x?.name)).toEqual(['Cleave']);
  });
  it('skips abyssal sockets and groups granted by an item', () => {
    const at = placeGems('A R-G', [g(0, ['Curse on Hit'], 'Item:Doedre'), g(1, ['Cleave', 'Ruthless'])]);
    expect(at.map((x) => x?.name)).toEqual([undefined, 'Cleave', 'Ruthless']);
  });
});

describe('abyssalSlots', () => {
  it("names the slot each abyssal socket's jewel is equipped in, counting abyssal sockets only", () => {
    expect(abyssalSlots('G-W-W-W-R A', 'Body Armour')).toEqual([undefined, undefined, undefined, undefined, undefined, 'Body Armour Abyssal Socket 1']);
    expect(abyssalSlots('A R-G A', 'Weapon 1 Swap')).toEqual(['Weapon 1 Swap Abyssal Socket 1', undefined, undefined, 'Weapon 1 Swap Abyssal Socket 2']);
    expect(abyssalSlots('', 'Belt')).toEqual([]);
  });
});

describe('artBox on the board', () => {
  it('draws every item at the board cell, never stretched to fill a larger slot', () => {
    expect(BOARD_CELL).toBe(72);
    expect(artBox([1, 3], 144, 298, BOARD_CELL)).toEqual({ w: 72, h: 216, cell: 72, cols: 1, rows: 3 });
    expect(artBox([2, 4], 144, 298, BOARD_CELL)).toEqual({ w: 144, h: 288, cell: 72, cols: 2, rows: 4 });
    // a slot smaller than the board cell still shrinks the item to fit
    expect(artBox([1, 1], 60, 60, BOARD_CELL).cell).toBe(60);
  });
});
