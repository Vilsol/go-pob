import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { buildStatList2, displayStats2 } from './stats2';
import type { SkillDPSEntry } from './stat_list';

// PoB-PoE2's Full DPS over the PoE2 fixture builds with every group included (oracle/poe2_full_dps.lua,
// `mise run full-dps-poe2-fixtures`): its SkillDPS rows and the sidebar rows from Full DPS on
type Row = {
  build: string;
  FullDPS: number;
  FullDotDPS: number;
  skills: { name: string; dps: number; count: number; trigger?: string; skillPart?: string; source?: string }[];
  statBox?: [string, string][];
};
const rows = readFileSync(fileURLToPath(new URL('../../../../testdata/oracle/poe2_full_dps.jsonl', import.meta.url)), 'utf8')
  .split('\n')
  .filter(Boolean)
  .map((l) => JSON.parse(l) as Row);

describe('the PoE2 sidebar Full DPS rows', () => {
  const shown = rows.filter((r) => r.statBox?.length);
  it('have fixtures with skill rows, a skill part and a source', () => {
    expect(shown.length).toBeGreaterThan(20);
    expect(shown.some((r) => r.skills.some((s) => s.skillPart))).toBe(true);
    expect(shown.some((r) => r.skills.some((s) => s.source))).toBe(true);
  });

  it.each(shown.map((r) => [r.build, r] as const))("are PoB-PoE2's for %s", (_, r) => {
    const skillDPS: SkillDPSEntry[] = r.skills.map((s) => ({
      Name: s.name,
      DPS: s.dps,
      Count: s.count,
      Trigger: s.trigger ?? '',
      SkillPart: s.skillPart ?? '',
      Source: s.source ?? ''
    }));
    const groups = buildStatList2({ Output: { FullDPS: r.FullDPS, FullDotDPS: r.FullDotDPS }, SkillFlags: {}, SkillDPS: skillDPS }, displayStats2);
    const lines = groups.flat();
    const from = lines.findIndex((l) => l.label === 'Full DPS');
    const ours = lines.slice(from).map((l): [string, string] => (l.detail ? [l.value, ''] : [`${l.label}${l.labelNote ?? ''}:`, l.value + (l.note ?? '')]));
    expect(ours).toEqual(r.statBox);
  });
});
