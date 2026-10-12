export type GemOptions = {
  defaultLevel: string;
  defaultQuality: number;
  showSupport: string;
  showAlt: boolean;
  showLegacy: boolean;
};

/* SkillSetListControl.lua:14-110 */
export type SkillSetOp = { op: 'new'; title: string } | { op: 'copy' | 'rename'; index: number; title: string } | { op: 'delete'; index: number };
