import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import type { TickWarnings } from '../custom_types';
import { buildSidebar } from './sidebar';
import { buildStatList, type StatLine } from './stat_list';
import { displayStats } from './stats';

type Row = { height: number; center: boolean; cells: string[] };
type Skill = { name: string; dps: number; count: number; trigger: string | null; skillPart: string | null; source: string | null };
type Case = {
  file: string;
  statBox: Row[];
  warnings: string[];
  skillDPS: Skill[];
  energyShieldProtectsMana: boolean;
  lifeCostWarningList: string[];
  manaCostWarningList: string[];
  rageCostWarningList: string[];
  esCostWarningList: string[];
  lifePercentCostWarningList: string[];
  manaPercentCostWarningList: string[];
  jewelLimitWarning: string[];
  socketLimitWarning: string[];
  missingAnointWarning: string[];
};
const cases = readFileSync(fileURLToPath(new URL('../../../../testdata/oracle/sidebar_stat_box_expected.jsonl', import.meta.url)), 'utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as Case);

const plain = (s: string) => s.replace(/\^(x[0-9A-Fa-f]{6}|\d)/g, '');
const CUSTOM = '^x5CF0BB';

const pobSkillRows = (c: Case) => {
  const start = c.statBox.findIndex((r) => !r.center && r.cells[0]?.startsWith(CUSTOM));
  return start < 0
    ? []
    : c.statBox
        .slice(start)
        .filter((r) => r.cells.length > 0)
        .map((r) => (r.center ? `  ${plain(r.cells[0])}` : `${plain(r.cells[0])} ${plain(r.cells[1] ?? '')}`));
};

const ourRows = (lines: StatLine[]) => lines.map((l) => (l.detail ? `  ${l.value}` : `${l.label}${l.labelNote ?? ''}: ${l.value}`));

describe('Skill DPS lines against PoB (Build.lua:1708-1739)', () => {
  it('has cases with skills', () => {
    expect(cases.filter((c) => c.skillDPS.length >= 3).length).toBeGreaterThan(5);
  });

  it('renders output.SkillDPS as PoB does', () => {
    const diffs: string[] = [];
    for (const c of cases) {
      const skills = c.skillDPS.map((s) => ({
        Name: s.name,
        DPS: s.dps,
        Count: s.count,
        Trigger: s.trigger ?? '',
        SkillPart: s.skillPart ?? '',
        Source: s.source ?? ''
      }));
      const lines = buildStatList({ SkillDPS: skills }, displayStats.slice(-1)).flat();
      const got = ourRows(lines);
      const want = pobSkillRows(c);
      if (JSON.stringify(got) !== JSON.stringify(want)) {
        diffs.push(`${c.file}\n  ours ${JSON.stringify(got)}\n  PoB  ${JSON.stringify(want)}`);
      }
    }
    expect(diffs).toEqual([]);
  });
});

describe('warnings line against PoB (Build.lua:1778-1831)', () => {
  const calcSide = /^(You do not have enough|You have too many gems|You have eligible items|You are exceeding jewel limit)/;

  it('has cases with warnings', () => {
    expect(cases.filter((c) => c.warnings.some((w) => calcSide.test(w))).length).toBeGreaterThan(5);
  });

  it('words the cost and item warnings as PoB does', () => {
    const diffs: string[] = [];
    for (const c of cases) {
      const tick: TickWarnings = {
        seq: 1,
        ms: 0,
        costWarning: [],
        warnings: {
          EnergyShieldProtectsMana: c.energyShieldProtectsMana,
          LifeCostWarningList: c.lifeCostWarningList,
          ManaCostWarningList: c.manaCostWarningList,
          RageCostWarningList: c.rageCostWarningList,
          ESCostWarningList: c.esCostWarningList,
          LifePercentCostPercentCostWarningList: c.lifePercentCostWarningList,
          ManaPercentCostPercentCostWarningList: c.manaPercentCostWarningList,
          PlayerAspectCount: 0,
          MinionAspectCount: 0,
          JewelLimitWarning: c.jewelLimitWarning,
          SocketLimitWarning: c.socketLimitWarning,
          MissingAnointWarning: c.missingAnointWarning
        } as TickWarnings['warnings']
      };
      const got = buildSidebar({}, tick)
        .warnings.map((w) => w.text)
        .sort();
      const want = c.warnings.filter((w) => calcSide.test(w)).sort();
      if (JSON.stringify(got) !== JSON.stringify(want)) {
        diffs.push(`${c.file}\n  ours ${JSON.stringify(got)}\n  PoB  ${JSON.stringify(want)}`);
      }
    }
    expect(diffs).toEqual([]);
  });
});

describe('sidebar sections (Build.lua:1849-1872)', () => {
  it('puts the minion first under headers, and the player alone without', () => {
    const withMinion = buildSidebar({ Output: { Life: 10 }, Minion: { Output: { Life: 5 } } });
    expect(withMinion.sections.map((s) => s.header)).toEqual(['Minion', 'Player']);
    expect(withMinion.sections[0].groups.flat().map((l) => l.label)).toContain('Total Life');
    const alone = buildSidebar({ Output: { Life: 10 } });
    expect(alone.sections.map((s) => s.header)).toEqual([undefined]);
  });

  it('opens the player section with "Skill disabled:" and the reason (Build.lua:1868-1871)', () => {
    const off = buildSidebar({ Output: { Life: 10 }, MainSkill: { SkillDisabled: true, DisableReason: 'No usable weapon equipped' } as never });
    expect(off.sections.at(-1)?.info).toEqual(['^7Skill disabled:', 'No usable weapon equipped']);
    const on = buildSidebar({ Output: { Life: 10 }, MainSkill: { SkillDisabled: false, DisableReason: '' } as never });
    expect(on.sections.at(-1)?.info).toBeUndefined();
  });

  it('opens with the main skill info lines, split at colons past 40 characters, and the minion section with its own (Build.lua:1837-1863)', () => {
    const out = buildSidebar({
      Output: { Life: 10 },
      Minion: { Output: { Life: 5 } },
      MainSkill: {
        InfoMessage: "Cast On Critical Strike's Trigger: Lancing Steel of Spraying",
        InfoMessage2: 'DPS reported assuming Self-Cast',
        MinionInfoMessage: 'Short',
        MinionInfoMessage2: ''
      } as never
    });
    expect(out.info).toEqual(["^#5CF0BBCast On Critical Strike's Trigger", '^#5CF0BB Lancing Steel of Spraying', '^8DPS reported assuming Self-Cast']);
    expect(out.sections[0].info).toEqual(['^#5CF0BBShort']);
    expect(buildSidebar({ Output: {}, MainSkill: { InfoMessage: '', InfoMessage2: 'only the second' } as never }).info).toBeUndefined();
  });

  it('collects warnFunc lines in stat order, once each', () => {
    const out = buildSidebar({
      Output: { Str: 10, ReqStr: 20, LifeUnreserved: 0, Life: 100 },
      OutputStrings: { ReqStrItem: "Kaom's Heart, Glorious Plate" }
    });
    expect(out.warnings).toEqual([
      { kind: 'stat', text: "You do not meet the Strength requirement of Kaom's Heart, Glorious Plate" },
      { kind: 'stat', text: 'Your unreserved Life is below 1' }
    ]);
  });
});
