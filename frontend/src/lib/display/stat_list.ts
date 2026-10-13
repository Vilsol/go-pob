// Build.lua:1689-1777

import { colorCodes } from './colors';
import { formatStat, SUFFIX_COLOR, VALUE_COLOR } from './format_stat';
import { statGauge, type Gauge } from './stat_gauge';
import { displayStats, type Stat, type StatOutput } from './stats';

export interface StatLine {
  label: string;
  labelColor: string;
  /* Build.lua:1713-1715 */
  labelNote?: string;
  value: string;
  valueColor: string;
  note?: string;
  /* Build.lua:1725-1738 */
  detail?: boolean;
  gauge?: Gauge;
  stat?: string;
  minion?: boolean;
}

export interface SkillDPSEntry {
  Name: string;
  DPS: number;
  Count: number;
  Trigger?: string;
  SkillPart?: string;
  Source?: string;
}

export interface StatListInput {
  Output?: Record<string, number>;
  OutputTable?: Record<string, Record<string, number> | undefined>;
  SkillFlags?: Record<string, boolean>;
  OutputStrings?: Record<string, string>;
  SkillDPS?: SkillDPSEntry[];
  /* Calcs.lua:450 */
  CostWarning?: string[];
}

export const NOTE_COLOR = SUFFIX_COLOR;
const DETAIL_COLOR = SUFFIX_COLOR;

// src/Modules/Build.lua:34-58
export const matchFlags = (reqFlag: string | undefined, notFlag: string | undefined, flags: Record<string, boolean>) =>
  (reqFlag === undefined || flags[reqFlag] === true) && (notFlag === undefined || flags[notFlag] !== true);

export const statValue = (stat: Stat, input: StatListInput): number | undefined => {
  if (stat.childStat) {
    return input.OutputTable?.[stat.stat!]?.[stat.childStat];
  }
  return input.Output?.[stat.stat!];
};

export const insertIfNew = (list: string[], line: string) => {
  if (!list.includes(line)) {
    list.push(line);
  }
};

// src/Modules/Build.lua:1708-1739
const skillDPSLines = (skills: SkillDPSEntry[], compactValue?: boolean): StatLine[] => {
  const lines: StatLine[] = [];
  const sorted = [...skills].sort((a, b) => b.DPS * b.Count - a.DPS * a.Count);
  for (const skill of sorted) {
    const formatted = formatStat({ fmt: '.1f', compactValue }, skill.DPS * skill.Count);
    lines.push({
      label: (skill.Count >= 2 ? `${skill.Count}x ` : '') + skill.Name,
      labelColor: colorCodes.CUSTOM,
      labelNote: skill.Trigger ? ` (${skill.Trigger})` : undefined,
      value: formatted.value,
      valueColor: formatted.color
    });
    if (skill.SkillPart) {
      lines.push({ label: 'Skill part', labelColor: DETAIL_COLOR, value: skill.SkillPart, valueColor: DETAIL_COLOR, detail: true });
    }
    if (skill.Source) {
      lines.push({ label: 'Source', labelColor: colorCodes.WARNING, value: `from ${skill.Source}`, valueColor: colorCodes.WARNING, detail: true });
    }
  }
  return lines;
};

// src/Modules/Build.lua:1689-1777
export const buildStatList = (input: StatListInput, statList: Stat[] = displayStats, warnings: string[] = []): StatLine[][] => {
  const output = (input.Output ?? {}) as StatOutput;
  const flags = input.SkillFlags ?? {};
  const costWarning = new Set(input.CostWarning ?? []);
  const groups: StatLine[][] = [];
  let group: StatLine[] = [];
  const spacer = () => {
    if (group.length > 0) {
      groups.push(group);
      group = [];
    }
  };

  for (const statData of statList) {
    if (!matchFlags(statData.flag, statData.notFlag, flags)) {
      continue;
    }
    if (!statData.stat) {
      spacer();
      continue;
    }
    if (statData.stat === 'SkillDPS') {
      group.push(...skillDPSLines(input.SkillDPS ?? [], statData.compactValue));
      continue;
    }
    const statVal = statValue(statData, input);
    if (statVal === undefined) {
      continue;
    }
    const shown = statData.condFunc ? statData.condFunc(statVal, output) : statVal !== 0;
    if (shown && !statData.hideStat) {
      let overCapStatVal = statData.overCapStat !== undefined ? output[statData.overCapStat] : undefined;
      if (overCapStatVal !== undefined && statData.overCapStatCondFunc && !statData.overCapStatCondFunc(statVal, output)) {
        overCapStatVal = undefined;
      }

      let colorOverride: string | undefined;
      if (
        output[`${statData.stat}Warning`] ||
        costWarning.has(statData.stat) ||
        (statData.warnFunc && statData.warnFunc(statVal, output, input.OutputStrings) && statData.warnColor)
      ) {
        colorOverride = colorCodes.NEGATIVE;
      }
      const formatted = formatStat(statData, statVal, overCapStatVal, colorOverride);

      let note = formatted.overCap ?? '';
      if (statData.suffix && (!statData.suffixCondFunc || statData.suffixCondFunc(statVal, output))) {
        const suffix = typeof statData.suffix === 'function' ? statData.suffix(statVal, output) : statData.suffix;
        if (suffix) {
          note += ` (${suffix})`;
        }
      }

      const gauge = statData.childStat ? undefined : statGauge(statData.stat, output);
      group.push({
        label: statData.label!,
        labelColor: statData.color ?? VALUE_COLOR,
        value: formatted.value,
        valueColor: formatted.color,
        note: note || undefined,
        ...(gauge && { gauge }),
        ...(!statData.childStat && { stat: statData.stat })
      });
    }
    // src/Modules/Build.lua:1763-1768
    if (statData.warnFunc && (!statData.condFunc || shown)) {
      const line = statData.warnFunc(statVal, output, input.OutputStrings);
      if (line) {
        insertIfNew(warnings, line);
      }
    }
  }
  spacer();
  return groups;
};
