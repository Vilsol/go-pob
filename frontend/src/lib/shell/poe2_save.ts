import { get } from 'svelte/store';
import { syncWrap } from '$lib/go/worker';
import { poe2Build } from '$lib/global';
import { displayStats2, minionDisplayStats2 } from '$lib/display/stats2';
import type { Poe2Calc } from '$lib/go/sync_worker2';
import type { exposition } from '$lib/types2';

export type SavedStats2 = { player: exposition.SavedStat[]; fullDPS: exposition.SavedFullDPS[]; minion: exposition.SavedStat[] };

// src/Modules/BuildDisplayStats.lua:295-304 (PoB-PoE2)
const EXTRA_SAVE_STATS = [
  'PowerCharges',
  'PowerChargesMax',
  'FrenzyCharges',
  'FrenzyChargesMax',
  'EnduranceCharges',
  'EnduranceChargesMax',
  'ActiveTotemLimit',
  'ActiveMinionLimit'
];

// src/Modules/Build.lua:1196-1244 (PoB-PoE2); its condFunc test ends in `or true`, so it never drops a row.
// The output table holds numbers only, so a childStat row and the SkillDPS table read as nil.
export const savedStats2 = (calc: Pick<Poe2Calc, 'output' | 'skillFlags' | 'minion'>): SavedStats2 => {
  const player: exposition.SavedStat[] = [];
  const added = new Set<string>();
  for (const s of displayStats2) {
    if (s.flag && !calc.skillFlags[s.flag]) continue;
    const name = s.stat && s.stat + (s.childStat ?? '');
    if (!name || added.has(name) || s.stat === 'SkillDPS' || s.childStat) continue;
    const v = calc.output[s.stat!];
    if (v === undefined) continue;
    player.push({ Stat: name, Value: v });
    added.add(name);
  }
  for (const stat of EXTRA_SAVE_STATS) {
    const v = calc.output[stat];
    if (v !== undefined) player.push({ Stat: stat, Value: v });
  }
  const minion: exposition.SavedStat[] = [];
  if (calc.minion) {
    for (const s of minionDisplayStats2) {
      const v = s.stat ? calc.minion[s.stat] : undefined;
      if (v !== undefined) minion.push({ Stat: s.stat!, Value: v });
    }
  }
  return { player, fullDPS: [], minion };
};

// the PoE2 worker serializes; the PoE1 worker owns the build storage
export const savePoe2Build = async (path: string) => {
  const { poe2Worker } = await import('$lib/go/worker2');
  const view = get(poe2Build) as Poe2Calc | undefined;
  const xml = await (await poe2Worker()).Serialize(view?.skillFlags ? savedStats2(view) : undefined);
  await syncWrap.SaveForeignBuild(path, xml);
};
