import type { Outputs } from '$lib/custom_types';
import type { CalcData, CalcDataRow, CalcSection } from './calc_sections';

/* Classes/CalcSectionControl.lua:86, CalcSectionControl.lua:186-204 */
export type FlagGated = {
  flag?: string;
  flagList?: string[];
  playerFlag?: string;
  notFlag?: string;
  notFlagList?: string[];
  haveOutput?: string;
};

const namespaced = /^([a-zA-Z]+)\.([a-zA-Z]+)$/;

const absent = (value: unknown): boolean => value === undefined || value === null || value === false || value === 0;

/* Classes/CalcsTab.lua:361-398 */
export const checkFlag = (obj: FlagGated, outputs: Outputs | undefined): boolean => {
  if (!outputs) {
    return true;
  }

  const skillFlags = outputs.SkillFlags ?? {};

  if (obj.flag && !skillFlags[obj.flag]) {
    return false;
  }
  if (obj.flagList) {
    for (const flag of obj.flagList) {
      if (!skillFlags[flag]) {
        return false;
      }
    }
  }
  if (obj.playerFlag && !skillFlags[obj.playerFlag]) {
    return false;
  }
  if (obj.notFlag && skillFlags[obj.notFlag]) {
    return false;
  }
  if (obj.notFlagList) {
    for (const flag of obj.notFlagList) {
      if (skillFlags[flag]) {
        return false;
      }
    }
  }
  if (obj.haveOutput) {
    const match = namespaced.exec(obj.haveOutput);
    if (match) {
      const [, ns, key] = match;
      const table = outputs.OutputTable?.[ns];
      if (!table || absent(table[key])) {
        return false;
      }
    } else if (absent(outputs.Output?.[obj.haveOutput])) {
      return false;
    }
  }

  return true;
};

export const rowsOf = (data: CalcData): CalcDataRow[] =>
  Object.entries(data)
    .filter(([k]) => !isNaN(parseInt(k)))
    .sort(([a], [b]) => parseInt(a) - parseInt(b))
    .map(([, row]) => row as CalcDataRow);

/* Classes/CalcSectionControl.lua:17-18 */
export const sectionEnabled = (section: CalcSection, outputs: Outputs | undefined): boolean => {
  const first = section[4][0];
  if (!first) {
    return false;
  }

  if (!checkFlag({ flag: first.data.flag, notFlag: first.data.notFlag }, outputs)) {
    return false;
  }

  return section[4].some((block) => rowsOf(block.data).some((row) => checkFlag(row, outputs)));
};
