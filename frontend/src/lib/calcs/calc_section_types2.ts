// The shape of PoB-PoE2's CalcSections.lua tables as calc_sections2.ts carries them: a Lua sequence index n is key
// n-1 beside the table's named fields.

/* CalcsTab.lua:433-478 (PoB-PoE2) */
export type FlagGated2 = {
  flag?: string;
  flagList?: string[];
  playerFlag?: string;
  notFlag?: string;
  notFlagList?: string[];
  skillData?: string;
  notSkillData?: string;
  haveOutput?: string;
};

/* CalcSections.lua:52-63 (PoB-PoE2) */
export type CalcSectionEntry2 = FlagGated2 & {
  breakdown?: string;
  // a table that also holds keys (CalcSections.lua:776-777, 1575) comes as an object, its sequence under "0", "1", ...
  modName?: string | string[] | { [n: number]: string; modType?: string; cfg?: string };
  modType?: 'BASE' | 'INC' | 'MORE' | 'OVERRIDE' | 'FLAG' | 'LIST';
  cfg?: string;
  enemy?: boolean;
  actor?: string;
  modSource?: string;
  label?: string;
  gearOnly?: boolean;
  skipSideBarIndex?: boolean;
};

export type CalcSectionColumn2 = CalcSectionEntry2 & { format?: string } & { [n: number]: CalcSectionEntry2 };

export type CalcSectionRow2 = FlagGated2 & { label?: string; color?: string; bgCol?: string; textSize?: number } & { [n: number]: CalcSectionColumn2 };

export type CalcSectionData2 = { extra?: string; colWidth?: number; flag?: string; notFlag?: string } & { [n: number]: CalcSectionRow2 };

export type CalcSectionSubsection2 = { defaultCollapsed: boolean; label: string; data: CalcSectionData2 };

/* width in columns, id, group, colour, subsections */
export type CalcSection2 = [number, string, number, string, CalcSectionSubsection2[]];
