import { describe, expect, it } from 'vitest';
import { compareStatList } from './stat_compare';

// Build.lua:1893-1899
describe('compareStatList formatting', () => {
  const cases: [string, number, string][] = [
    ['d', 1234.7, '+1,234'],
    ['d', -0.3, '+0'],
    ['.1f', -0.04, '-0'],
    ['.1f', 12345.25, '+12,345.3'],
    ['.1f', -3.0, '-3'],
    ['.2f', 0.105, '+0.1'],
    ['.2fs', -0.5, '-0.5s'],
    ['d%%', 7, '+7%'],
    ['.0f', 1500.5, '+1,501'],
    ['+.1f', 2.25, '+2.3'],
    ['.2f%%/s', 1.2345, '+1.23%/s'],
    ['.3fs', 0.0333, '+0.033s']
  ];
  for (const [fmt, diff, want] of cases) {
    it(`%+${fmt} of ${diff}`, () => {
      const [line] = compareStatList([{ stat: 'X', label: 'X', fmt }], {}, { X: 0 }, { X: diff });
      expect(line?.value).toBe(want);
    });
  }

  it('skips a change under 0.001 and colours lowerIsBetter', () => {
    const lines = compareStatList(
      [
        { stat: 'A', label: 'A', fmt: 'd' },
        { stat: 'B', label: 'B', fmt: 'd', lowerIsBetter: true }
      ],
      {},
      { A: 1, B: 5 },
      { A: 1.0005, B: 7 }
    );
    expect(lines.map((l) => [l.label, l.better])).toEqual([['B', false]]);
  });

  it('skips a child stat and SkillDPS, as Build.lua:1880 does', () => {
    const lines = compareStatList(
      [
        { stat: 'MainHand', childStat: 'Accuracy', label: 'MH Accuracy', fmt: 'd' },
        { stat: 'SkillDPS', label: 'Skill DPS', fmt: 'd' },
        { stat: 'Life', label: 'Total Life', fmt: 'd' }
      ],
      {},
      { MainHand: 100, SkillDPS: 1, Life: 10 },
      { MainHand: 200, SkillDPS: 2, Life: 20 }
    );
    expect(lines.map((l) => l.label)).toEqual(['Total Life']);
  });

  it('adds the percent and per-point parts', () => {
    const [line] = compareStatList([{ stat: 'L', label: 'Life', fmt: '.1f', compPercent: true }], {}, { L: 200 }, { L: 250 }, 4);
    expect([line.percent, line.perPoint]).toEqual(['(+25.0%)', '[+12.5 (+6.3%) pp]']);
  });
});

// Build.lua:1891
describe('compareStatList colours', () => {
  it("gives a line its stat's colour, none to a stat without one", () => {
    const lines = compareStatList(
      [
        { stat: 'Life', label: 'Total Life', color: '#E05030' },
        { stat: 'X', label: 'X' }
      ],
      {},
      { Life: 0, X: 0 },
      { Life: 10, X: 1 }
    );
    expect(lines.map((l) => l.color)).toEqual(['#E05030', undefined]);
  });
});
