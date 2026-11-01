import { describe, expect, it } from 'vitest';
import { configView } from './config';
import { poe2ConfigTable } from './config_table2';
import { relevanceQuery2 } from '$lib/display/configurations2';
import type { calculator2 } from '$lib/types2';

const none = Object.fromEntries(Object.keys(relevanceQuery2).map((k) => [k, []])) as unknown as calculator2.ConfigRelevance;
const shown = (relevance: calculator2.ConfigRelevance) =>
  configView(poe2ConfigTable, {}, {}, relevance, { q: '', showAll: false, modOnly: false })
    .sections.flatMap((s) => s.rows)
    .flatMap((r) => (r.kind === 'option' ? [`${r.v.var} ${r.v.label}`] : []));

// ConfigOptions.lua:1669-1674 (PoB-PoE2) gives two rows the var conditionEnemyExitedPresenceRecently; each is shown by its own condition
describe('the PoE2 options table', () => {
  it('judges two rows that share a var by their own conditions', () => {
    const entered = shown({ ...none, EnemyConds: ['EnteredPresenceRecently'] });
    const exited = shown({ ...none, EnemyConds: ['ExitedPresenceRecently'] });
    expect(entered.filter((r) => r.startsWith('conditionEnemyExitedPresenceRecently '))).toEqual([
      'conditionEnemyExitedPresenceRecently Enemy entered Presence Recently?'
    ]);
    expect(exited.filter((r) => r.startsWith('conditionEnemyExitedPresenceRecently '))).toEqual([
      'conditionEnemyExitedPresenceRecently Enemy exited Presence Recently?'
    ]);
  });

  it("shows a list option's seeded default", () => {
    const view = configView(poe2ConfigTable, {}, {}, none, { q: '', showAll: false, modOnly: false });
    const penalty = view.sections.flatMap((s) => s.rows).find((r) => r.kind === 'option' && r.v.var === 'resistancePenalty');
    expect(penalty?.kind === 'option' && penalty.value).toBe(-60);
  });
});
