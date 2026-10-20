import type { Outputs } from '$lib/custom_types';

export type CompareInfo = { name: string; seq: number; names: string[]; active: number };
export type CompareGem = { gemId: string; nameSpec: string; skillId: string; level: number; quality: number; enabled: boolean; count: number };
export type CompareGroup = { label: string; slot: string; enabled: boolean; inFull: boolean; count: number; imbued?: string; gems: CompareGem[] };
export type CompareItem = { slot: string; name: string; rarity: string; text: string; line: string };
/* CompareTab.lua:1372-1436 */
export type CompareJewel = { slot: string; label: string; node: number; a?: CompareItem; b?: CompareItem; aAllocated: boolean; bAllocated: boolean };
export type CompareSideData = {
  className: string;
  ascendClassName: string;
  level: number;
  mainSocketGroup: number;
  nodes: number[];
  configSet: string;
  inputs: Record<string, boolean | number | string>;
  skills: { activeSet: number; sets: { title: string; groups: CompareGroup[] }[] };
  outputs?: Pick<Outputs, 'Output' | 'SkillFlags' | 'PointsUsed' | 'Minion' | 'ConfigRelevance' | 'MainSkill'>;
};
/* CompareTab.lua:290-342, 2199-2230 */
export type { CompareSetKind, CompareSets } from '$lib/design/compare';
/* CompareTab.lua:490-718 */
export type CompareCalcsInputs = { groups: string[]; skillNumber: number; buffMode: string; showMinion: boolean };
