import type { pob, calculator } from './types';
import type { SkillGemCacheItem } from '$lib/cache';

export type Outputs = {
  Output?: Record<string, number>;
  OutputTable?: Record<string, Record<string, number> | undefined>;
  SkillFlags?: Record<string, boolean>;
  Breakdown?: Record<string, calculator.EntryPlain | undefined>;
  Calcs?: Record<string, number>;
  MainSkill?: calculator.MainSkillControls;
  PointsUsed?: [number, number, number, number];
  ConfigRelevance?: calculator.ConfigRelevance;
  OutputStrings?: Record<string, string>;
  /* Calcs.lua:176-388 */
  SkillDPS?: calculator.SkillDPS[];
  CalcsTab?: Outputs;
  /* Build.lua:1849-1867 */
  Minion?: {
    Output?: Record<string, number>;
    OutputTable?: Record<string, Record<string, number> | undefined>;
    SkillFlags?: Record<string, boolean>;
  };
};

export type TickWarnings = {
  seq: number;
  warnings: calculator.SidebarWarnings;
  /* Calcs.lua:450 */
  costWarning: string[];
  ms: number;
};

export type TickMeta = {
  seq: number;
  pending: number;
  source: string;
  startedAt: number;
  calcMs: number;
  dumpMs: number;
  sentAt: number;
};

export type TickCallback = (out: Outputs | undefined, meta: TickMeta) => void;

export interface GemListValue {
  label: string;
  value: string;
  data: SkillGemCacheItem;
}

interface GemWithStuff extends pob.Gem {
  GemListValue: GemListValue;
}

export type SkillGroupUpdate = Pick<pob.Skill, 'Enabled' | 'IncludeInFullDPS' | 'Slot'> & {
  Gems: Array<GemWithStuff>;
};
