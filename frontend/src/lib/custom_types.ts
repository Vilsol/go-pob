import type { pob } from './types';

export type Outputs = {
  Output?: Record<string, number>;
  OutputTable?: Record<string, Record<string, number> | undefined>;
  SkillFlags?: Record<string, boolean>;
};

export type SkillGroupUpdate = Pick<pob.Skill, 'Enabled' | 'IncludeInFullDPS' | 'Slot' | 'Gems'>;
