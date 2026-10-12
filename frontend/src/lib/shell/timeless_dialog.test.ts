import { describe, expect, it } from 'vitest';
import { insertSearchNode, nodeWeightLabels, rewriteSearchRow, slidersFromRow, statCount } from './timeless_dialog';

describe('timeless search list editing (TreeTab.lua:1404-1918)', () => {
  it('reads the sliders as getNodeWeights does: %.3f rounded to 3, the minimum %.f or required', () => {
    expect(nodeWeightLabels(1, 0.1, 0)).toEqual(['1', '0.1', '0']);
    expect(nodeWeightLabels(2.3456, 0, 250.4)).toEqual(['2.346', '0', '250']);
    expect(nodeWeightLabels(10, 10, 500)).toEqual(['10', '10', 'required']);
  });

  it('adds a picked node at the end and reads an existing row back into the sliders', () => {
    expect(insertSearchNode('', 'karui_notable_add_life', ['1', '0', '0'])).toBe('karui_notable_add_life, 1, 0, 0');
    expect(insertSearchNode('a, 1', 'b', ['2', '0', 'required'])).toBe('a, 1\nb, 2, 0, required');
    expect(slidersFromRow('a 1,2 required\nb', 'a')).toEqual({ w1: 1, w2: 2, w3: 500 });
    expect(slidersFromRow('a, 99, x', 'a')).toEqual({ w1: 10, w2: 0, w3: 0 });
    expect(slidersFromRow('a, 1', 'c')).toBeUndefined();
  });

  it('rewrites the selected row and rejoins every row with ", " (parseSearchList mode 1)', () => {
    expect(rewriteSearchRow('a 1 2\n\nb,3,4,5,6', 'b', ['7', '0', '1'])).toBe('a, 1, 2\nb, 7, 0, 1, 6');
  });

  it('counts a total mod as no legion node (the secondary slider stays off)', () => {
    expect(statCount({ Label: 'Total Strength', ID: 'total_strength', Descriptions: ['x'], TotalMod: true })).toBe(0);
    expect(statCount({ Label: 'n', ID: 'vaal_notable_armour_1', Descriptions: ['a', 'b'], TotalMod: false })).toBe(2);
  });
});
