import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { calcSections2 } from '$lib/calcs/calc_sections2';
import { checkFlag2, sidebarBreakdown2, type CalcsActor2 } from './calcs2';
import { displayStats2 } from './stats2';

// PoB-PoE2's sidebar breakdowns over the wasm2 fixture builds' main calculation (oracle/poe2_sidebar_breakdowns.lua and
// oracle/poe2_calcs.lua, `mise run breakdowns-poe2`)
type Cell = { breakdown?: string; modName?: string[]; modType?: string; label?: string };
type Line = { index: number; stat: string; modNames?: string[]; key: string; source?: string; cells: Cell[] };
const load = <T>(file: string) => JSON.parse(readFileSync(fileURLToPath(new URL(`../../../../testdata/oracle/${file}.json`, import.meta.url)), 'utf8')) as T;

describe.each(['sorceress_spark', 'warrior_dual_mace'])('a PoE2 sidebar breakdown: %s', (name) => {
  const lines = load<Line[]>(`poe2_sidebar_breakdowns_${name}`);
  const calcs = load<CalcsActor2>(`poe2_calcs_${name}`);
  const actor: CalcsActor2 = { output: calcs.output, skillFlags: calcs.skillFlags, skillData: calcs.skillData, mods: {} };

  it('shows the cells GetSidebarBreakdown gives for the stat', () => {
    expect(lines.length).toBeGreaterThan(30);
    for (const line of lines) {
      const statData = displayStats2[line.index - 1];
      expect(statData.stat, `display stat ${line.index}`).toBe(line.stat);
      const cells = sidebarBreakdown2(calcSections2, line.key, line.source ?? line.key, statData.modNames, statData.ignoredSections)
        .filter((c) => checkFlag2(c as never, actor))
        .map((c): Cell => {
          const cell: Cell = {};
          if (c.breakdown !== undefined) cell.breakdown = c.breakdown;
          if (c.modName !== undefined) cell.modName = [c.modName].flat();
          if (c.modType !== undefined) cell.modType = c.modType;
          if (c.label !== undefined) cell.label = c.label;
          return cell;
        });
      expect(cells, `${line.stat} (${line.key})`).toEqual(line.cells);
    }
  });
});
