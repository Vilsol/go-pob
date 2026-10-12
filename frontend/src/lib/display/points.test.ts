import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { autoLevel, estimatePlayerProgress, progressTooltip } from './points';

type Case = { file: string; used: number; ascUsed: number; secondaryAscUsed: number; sockets: number; extraPoints: number; pointStr: string; pointReq: string };
const cases = readFileSync(fileURLToPath(new URL('../../../../testdata/oracle/sidebar_controls_expected.jsonl', import.meta.url)), 'utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as Case);

const NEGATIVE = '^xDD0022';

describe('estimatePlayerProgress against PoB', () => {
  it('has cases', () => {
    expect(cases.length).toBeGreaterThan(10);
  });

  it('reproduces the point display and its tooltip', () => {
    const diffs: string[] = [];
    for (const c of cases) {
      const p = estimatePlayerProgress([c.used, c.ascUsed, c.secondaryAscUsed, c.sockets], c.extraPoints);
      const str = `${p.used > p.usedMax ? NEGATIVE : '^7'}${String(p.used).padStart(3)} / ${String(p.usedMax).padStart(3)}   ${p.ascUsed > p.ascMax ? NEGATIVE : '^7'}${p.ascUsed} / ${p.ascMax}`;
      if (str !== c.pointStr) diffs.push(`${c.file}: str ${JSON.stringify(str)} PoB ${JSON.stringify(c.pointStr)}`);
      if (progressTooltip(p) !== c.pointReq) diffs.push(`${c.file}: req ${JSON.stringify(progressTooltip(p))} PoB ${JSON.stringify(c.pointReq)}`);
    }
    expect(diffs).toEqual([]);
  });

  // Build.lua:886-890
  it('estimates the act and lab below endgame', () => {
    const p = estimatePlayerProgress([40, 2, 0, 0], 1);
    expect(p.level).toBe(34);
    expect(p.act).toBe(4);
    expect(progressTooltip(p)).toBe('Required Level: 34\nEstimated Progress:\nAct: 4\nQuestpoints: 6\nExtra Skillpoints: 1\nLabyrinth: Normal Lab');
  });
});

// Build.lua:892-896
describe('autoLevel', () => {
  it('returns the estimated level when auto mode is on and it differs', () => {
    const p = estimatePlayerProgress([100, 6, 0, 0], 1);
    expect(autoLevel(p, true, 50)).toBe(p.level);
    expect(autoLevel(p, true, p.level)).toBeUndefined();
    expect(autoLevel(p, false, 50)).toBeUndefined();
  });
});
