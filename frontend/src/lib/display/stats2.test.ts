import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { buildStatList2, displayStats2 } from './stats2';

// PoB-PoE2's own sidebar for the wasm2 fixture build (oracle/poe2_sidebar.lua, `task config-ui-oracles-poe2`)
type Oracle = { output: Record<string, number | true>; skillFlags: Record<string, true>; groups: [string, string][][]; warnings: string[] };
const oracle = JSON.parse(
  readFileSync(fileURLToPath(new URL('../../../../testdata/oracle/poe2_sidebar_sorceress_spark.json', import.meta.url)), 'utf8')
) as Oracle;

// a true output is present to the condition functions, as it is truthy in Lua
const output = Object.fromEntries(Object.entries(oracle.output).map(([k, v]) => [k, v === true ? 1 : v]));

describe('the PoE2 sidebar stat list', () => {
  it("lists PoB-PoE2's rows, values and groups for its own outputs", () => {
    const warnings: string[] = [];
    const groups = buildStatList2({ Output: output, SkillFlags: oracle.skillFlags }, displayStats2, warnings);
    expect(groups.map((g) => g.map((l) => [`${l.label}:`, l.value + (l.note ?? '')]))).toEqual(oracle.groups);
    // the requirement warnings name their source item or gem, a table output the calculator does not hold
    expect(warnings.map((w) => w.replace(/ of .*$/, ' of'))).toEqual(oracle.warnings.map((w) => w.replace(/ of .*$/, ' of')));
  });

  it('drops a flag-gated row when the main skill lacks the flag', () => {
    const groups = buildStatList2({ Output: output, SkillFlags: { ...oracle.skillFlags, spell: false } }, displayStats2);
    expect(groups.flat().some((l) => l.label === 'Cast Rate')).toBe(false);
  });
});
