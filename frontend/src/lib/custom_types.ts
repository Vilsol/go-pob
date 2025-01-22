import type { pob, calculator } from './types';
import type { SkillGemCacheItem } from '$lib/cache';

export type Outputs = {
  Output?: Record<string, number>;
  OutputTable?: Record<string, Record<string, number> | undefined>;
  SkillFlags?: Record<string, boolean>;
  Breakdown?: Record<string, calculator.Entry | undefined>;
  Calcs?: Record<string, number>;
};

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
