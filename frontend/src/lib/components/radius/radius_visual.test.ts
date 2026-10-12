import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { circleRows } from './radius_visual';

const oracle = JSON.parse(readFileSync(new URL('../../../../../testdata/tree-ui/radius_visual.json', import.meta.url), 'utf8')) as {
  radius: number;
  rows: [number, number, number][];
}[];

describe('radius visual (CalcBreakdownControl.lua:626-641)', () => {
  it('fills the rows Main.lua’s RenderCircle fills, for every graded radius', () => {
    expect(oracle.length).toBeGreaterThan(5);
    for (const { radius, rows } of oracle) expect(circleRows(radius), `radius ${radius}`).toEqual(rows);
  });
});
