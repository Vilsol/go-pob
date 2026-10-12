// Build.lua:486-561
import { get } from 'svelte/store';
import { currentBuild, UITick } from '$lib/global.js';
import { logError } from '$lib/utils';
import type { calculator } from '$lib/types';

export type MainSkillField = 'MainActiveSkill' | 'SkillPart' | 'StageCount' | 'MineCount' | 'Minion' | 'MinionSkill';

/* CalcsTab.lua:54-118 */
export function writeMainSkill(c: calculator.MainSkillControls | undefined, field: MainSkillField, value: number | undefined, calcs = false) {
  const b = get(currentBuild);
  if (!b || !c) return;
  if (field !== 'MainActiveSkill' && c.GemIndex < 0) return;
  let p: Promise<unknown> | undefined;
  let source: string = field;
  const sfx = calcs ? 'Calcs' : '';
  switch (field) {
    case 'MainActiveSkill':
      source = 'SetMainActiveSkill' + sfx;
      if (value !== undefined) p = calcs ? b.SetMainActiveSkillCalcs(c.SkillSet, c.SocketGroup, value) : b.SetMainActiveSkill(c.SkillSet, c.SocketGroup, value);
      break;
    case 'SkillPart':
      source = 'SetGemSkillPart' + sfx;
      if (value !== undefined)
        p = calcs ? b.SetGemSkillPartCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, value) : b.SetGemSkillPart(c.SkillSet, c.SocketGroup, c.GemIndex, value);
      break;
    case 'StageCount':
      source = 'SetGemSkillStageCount' + sfx;
      p = calcs
        ? b.SetGemSkillStageCountCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? 0)
        : b.SetGemSkillStageCount(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? 0);
      break;
    case 'MineCount':
      source = 'SetGemSkillMineCount' + sfx;
      p = calcs
        ? b.SetGemSkillMineCountCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? -1)
        : b.SetGemSkillMineCount(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? -1);
      break;
    case 'Minion': {
      const option = value === undefined ? undefined : c.Minions?.[value - 1];
      if (option?.ItemSetID) {
        source = 'SetGemSkillMinionItemSet' + sfx;
        p = calcs
          ? b.SetGemSkillMinionItemSetCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, option.ItemSetID)
          : b.SetGemSkillMinionItemSet(c.SkillSet, c.SocketGroup, c.GemIndex, option.ItemSetID);
      } else if (option?.MinionID) {
        source = 'SetGemSkillMinion' + sfx;
        p = calcs
          ? b.SetGemSkillMinionCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, option.MinionID)
          : b.SetGemSkillMinion(c.SkillSet, c.SocketGroup, c.GemIndex, option.MinionID);
      }
      break;
    }
    case 'MinionSkill':
      source = 'SetGemSkillMinionSkill' + sfx;
      if (value !== undefined)
        p = calcs
          ? b.SetGemSkillMinionSkillCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, value)
          : b.SetGemSkillMinionSkill(c.SkillSet, c.SocketGroup, c.GemIndex, value);
      break;
  }
  p?.then(() => UITick(source)).catch(logError);
}
