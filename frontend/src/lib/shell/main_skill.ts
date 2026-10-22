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
  let action: string | undefined;
  const sfx = calcs ? 'Calcs' : '';
  const at = (names: string[] | undefined) => (value === undefined ? undefined : names?.[value - 1]);
  const count = value === undefined || value < 0 ? 'its default' : String(value);
  switch (field) {
    case 'MainActiveSkill':
      source = 'SetMainActiveSkill' + sfx;
      action = `Set the ${calcs ? 'Calcs' : 'main'} active skill to ${at(c.Skills) ?? value}`;
      if (value !== undefined) p = calcs ? b.SetMainActiveSkillCalcs(c.SkillSet, c.SocketGroup, value) : b.SetMainActiveSkill(c.SkillSet, c.SocketGroup, value);
      break;
    case 'SkillPart':
      source = 'SetGemSkillPart' + sfx;
      action = `Set the skill part to ${at(c.Parts) ?? value}`;
      if (value !== undefined)
        p = calcs ? b.SetGemSkillPartCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, value) : b.SetGemSkillPart(c.SkillSet, c.SocketGroup, c.GemIndex, value);
      break;
    case 'StageCount':
      source = 'SetGemSkillStageCount' + sfx;
      action = `Set the stage count to ${count}`;
      p = calcs
        ? b.SetGemSkillStageCountCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? 0)
        : b.SetGemSkillStageCount(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? 0);
      break;
    case 'MineCount':
      source = 'SetGemSkillMineCount' + sfx;
      action = `Set the mine count to ${count}`;
      p = calcs
        ? b.SetGemSkillMineCountCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? -1)
        : b.SetGemSkillMineCount(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? -1);
      break;
    case 'Minion': {
      const option = value === undefined ? undefined : c.Minions?.[value - 1];
      action = `Set the minion to ${option?.Label ?? value}`;
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
      action = `Set the minion skill to ${at(c.MinionSkills) ?? value}`;
      if (value !== undefined)
        p = calcs
          ? b.SetGemSkillMinionSkillCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, value)
          : b.SetGemSkillMinionSkill(c.SkillSet, c.SocketGroup, c.GemIndex, value);
      break;
  }
  p?.then(() => UITick(source, action)).catch(logError);
}
