import { describe, expect, it } from 'vitest';

import { MAX_SCALING, MIN_SCALING, searchRingSize } from './geometry';

describe('the search ring', () => {
  it('is 25% smaller than max(20, 350 / scaling) fully zoomed in, the same fully zoomed out, and shrinks as the view zooms out', () => {
    const before = (s: number) => Math.max(20, 350 / s);
    expect(searchRingSize(MIN_SCALING)).toBeCloseTo(before(MIN_SCALING) * 0.75, 6);
    expect(searchRingSize(MAX_SCALING)).toBeCloseTo(before(MAX_SCALING), 6);
    for (let s = MIN_SCALING; s < MAX_SCALING; s += 0.5) expect(searchRingSize(s + 0.5)).toBeLessThan(searchRingSize(s));
  });
});
