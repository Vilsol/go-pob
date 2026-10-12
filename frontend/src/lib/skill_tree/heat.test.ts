import { describe, expect, it } from 'vitest';
import { heatChannel, heatValues } from './heat';
import type { PowerRow } from '$lib/design/tree';

const row = (ID: number, Power: number, extra: Partial<PowerRow> = {}): PowerRow => ({
  ID,
  Name: `n${ID}`,
  Type: 'Normal',
  Power,
  PathPower: 0,
  PathDist: 1,
  Allocated: false,
  ...extra
});

// PassiveTreeView.lua:906-919
describe('the node power heat map', () => {
  it('colours by (power / max * 1.5) ^ 0.5, a negative power as none, at most full', () => {
    expect(heatChannel(10, 10)).toBe(1);
    expect(heatChannel(2, 12)).toBeCloseTo(0.5);
    expect(heatChannel(-4, 12)).toBe(0);
    expect(heatChannel(3, 0)).toBe(0);
  });

  it('takes the unallocated nodes from the report, a mastery at its best effect, and the largest', () => {
    const { values, max } = heatValues([
      row(1, 5),
      row(2, 3, { Type: 'Mastery' }),
      row(2, 8, { Type: 'Mastery' }),
      row(3, -2),
      row(4, 50, { Allocated: true }),
      row(5, 99, { Cluster: true })
    ]);
    expect([...values.entries()]).toEqual([
      [1, 5],
      [2, 8],
      [3, -2]
    ]);
    expect(max).toBe(8);
  });
});

// PassiveTreeView.lua:920-933
// CalcsTab.lua:543-547
describe('the Offence/Defence heat map', () => {
  it('takes each maximum over reachable unallocated nodes and mixes the channels as PoB does', async () => {
    const { heatMixValues, heatMixColour } = await import('./heat');
    const mixRow = (ID: number, Offence: number, Defence: number, PathDist = 1, Allocated = false) => ({
      ID,
      Name: '',
      Type: 'Normal',
      Power: Offence,
      PathPower: 0,
      PathDist,
      Allocated,
      Offence,
      Defence
    });
    const { mix, maxOffence, maxDefence } = heatMixValues([mixRow(1, 0.2, 0.1), mixRow(2, 0.1, 0.4), mixRow(3, 9, 9, 1000), mixRow(4, 5, 5, 1, true)]);
    expect([maxOffence, maxDefence]).toEqual([0.2, 0.4]);
    expect(mix.has(4)).toBe(false);
    const [r, g, b] = heatMixColour(0.2, 0.1, maxOffence, maxDefence);
    const dps = Math.sqrt(1.5);
    const def = Math.sqrt((0.1 / 0.4) * 1.5);
    expect(r).toBe(1);
    expect(b).toBeCloseTo(def, 12);
    expect(g).toBeCloseTo((Math.max(dps - 0.5, 0) + Math.max(def - 0.5, 0)) / 2, 12);
  });
});

// Main.lua:957, PassiveTreeView.lua:913-918
describe('the node power theme', () => {
  it('moves the single-stat colour and the offence / defence channels as PoB does', async () => {
    const { heatSingleColour, heatMixColour } = await import('./heat');
    const c = Math.sqrt((2 / 12) * 1.5);
    expect(heatSingleColour(2, 12, 'RED/BLUE')).toEqual([c, 0, 0]);
    expect(heatSingleColour(2, 12, 'RED/GREEN')).toEqual([0, c, 0]);
    expect(heatSingleColour(2, 12, 'GREEN/BLUE')).toEqual([0, 0, c]);
    const [dps, mixed, def] = heatMixColour(0.1, 0.4, 0.2, 0.4, 'RED/BLUE');
    expect(heatMixColour(0.1, 0.4, 0.2, 0.4, 'RED/GREEN')).toEqual([dps, def, mixed]);
    expect(heatMixColour(0.1, 0.4, 0.2, 0.4, 'GREEN/BLUE')).toEqual([mixed, dps, def]);
  });
});

// CalcsTab.lua:533-534, 543-547
// PassiveTreeView.lua:908
describe('ascendancy nodes in the heat map', () => {
  it('colours them, and leaves them and unreachable nodes out of the largest power', () => {
    const { values, max } = heatValues([row(1, 4), row(2, 40), row(3, 90, { PathDist: 1000 })], new Set([2]));
    expect(values.get(2)).toBe(40);
    expect(max).toBe(4);
  });
});
