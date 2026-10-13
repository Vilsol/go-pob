import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { circleRows, ringLabelAt, ringPath } from './radius_visual';

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

describe('radius visual rings', () => {
  it('lie on the filled circle, and label it right of the character', () => {
    for (const radius of [8, 25, 44, 80]) {
      const rows = new Map(circleRows(radius).map(([y, x, w]) => [y, [x, x + w - 1]]));
      const points = [...ringPath(radius).matchAll(/[ML](-?\d+) (-?\d+)/g)].map((m) => [Number(m[1]), Number(m[2])]);
      expect(points.length, `radius ${radius}`).toBeGreaterThan(100);
      for (const [x, y] of points.filter(([px, py]) => py >= 0 && py < 270 && px > 0 && px < 480)) {
        const [lo, hi] = rows.get(y) ?? [NaN, NaN];
        expect(x >= lo - 1 && x <= hi + 1, `radius ${radius} at ${x},${y}: row ${lo}..${hi}`).toBe(true);
      }
      const [lx, ly] = ringLabelAt(radius);
      expect(rows.get(ly)?.[1], `the label of ${radius} sits on the edge`).toBeCloseTo(lx, -0.5);
    }
  });
});
