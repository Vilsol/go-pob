// Build.lua:1834-1875
// Build.lua:1778-1814

import type { Outputs, TickWarnings } from '../custom_types';
import { estimatePlayerProgress } from './points';
import { buildStatList, type StatLine, type StatListInput } from './stat_list';
import { displayStats, minionDisplayStats } from './stats';
import { colorCodesX } from './colors';

export interface SidebarSection {
  /* Build.lua:1850, 1866 */
  header?: string;
  /* Build.lua:1868-1871 */
  info?: string[];
  groups: StatLine[][];
}

export type WarningKind = 'stat' | 'cost' | 'skill' | 'item' | 'points';

export interface BuildWarning {
  kind: WarningKind;
  text: string;
}

export interface Sidebar {
  /* Build.lua:1837-1848 */
  info?: string[];
  sections: SidebarSection[];
  warnings: BuildWarning[];
}

type SidebarWarnings = TickWarnings['warnings'];

const push = (warnings: BuildWarning[], kind: WarningKind, text: string) => {
  if (!warnings.some((w) => w.text === text)) {
    warnings.push({ kind, text });
  }
};

// src/Modules/Build.lua:1778-1814
const actorWarnings = (warnings: BuildWarning[], output: Record<string, number>, w: SidebarWarnings | undefined, aspectCount: number) => {
  if (w) {
    const pools: [string, string[] | undefined][] = [
      ['Life', w.LifeCostWarningList],
      ['Mana', w.ManaCostWarningList],
      ['Rage', w.RageCostWarningList],
      ['Energy Shield', w.ESCostWarningList]
    ];
    for (const [pool, list] of pools) {
      if (list?.length) {
        const name = w.EnergyShieldProtectsMana && pool === 'Mana' ? 'Energy Shield and Mana' : pool;
        push(warnings, 'cost', `You do not have enough ${name} to use: ${list.join(', ')}`);
      }
    }
    const percentPools: [string, string[] | undefined][] = [
      ['Unreserved life', w.LifePercentCostPercentCostWarningList],
      ['Unreserved Mana', w.ManaPercentCostPercentCostWarningList]
    ];
    for (const [pool, list] of percentPools) {
      if (list?.length) {
        push(warnings, 'cost', `You do not have enough ${pool}% to use: ${list.join(', ')}`);
      }
    }
  }
  if (output.VixensTooMuchCastSpeedWarn) {
    push(warnings, 'skill', "You may have too much cast speed or too little cooldown reduction to effectively use Vixen's Curse replacement");
  }
  if (output.VixenModeNoVixenGlovesWarn) {
    push(warnings, 'skill', "Vixen's calculation mode for Doom Blast is selected but you do not have Vixen's Entrapment Embroidered Gloves equipped");
  }
  if (aspectCount > 1) {
    push(warnings, 'skill', 'You have more than one Aspect skill active');
  }
};

const statWarnings = (warnings: BuildWarning[], lines: string[]) => {
  for (const line of lines) {
    push(warnings, 'stat', line);
  }
};

// Build.lua:1838-1847
const infoLines = (message: string | undefined, message2: string | undefined): string[] | undefined => {
  if (!message) return undefined;
  const lines = message.length > 40 ? message.split(':').filter((l) => l) : [message];
  return [...lines.map((l) => colorCodesX.CUSTOM + l), ...(message2 ? ['^8' + message2] : [])];
};

// src/Modules/Build.lua:1834-1875
export const buildSidebar = (out: Outputs, tick?: TickWarnings): Sidebar => {
  const sections: SidebarSection[] = [];
  const warnings: BuildWarning[] = [];
  const w = tick?.warnings;

  if (out.Minion) {
    const lines: string[] = [];
    const groups = buildStatList(out.Minion, minionDisplayStats, lines).map((g) => g.map((l) => ({ ...l, minion: true })));
    statWarnings(warnings, lines);
    actorWarnings(warnings, out.Minion.Output ?? {}, undefined, w?.MinionAspectCount ?? 0);
    sections.push({ header: 'Minion', info: infoLines(out.MainSkill?.MinionInfoMessage, out.MainSkill?.MinionInfoMessage2), groups });
  }

  const player: StatListInput = { ...out, CostWarning: tick?.costWarning };
  const lines: string[] = [];
  const groups = buildStatList(player, displayStats, lines);
  statWarnings(warnings, lines);
  actorWarnings(warnings, out.Output ?? {}, w, w?.PlayerAspectCount ?? 0);
  const info = out.MainSkill?.SkillDisabled ? ['^7Skill disabled:', out.MainSkill.DisableReason] : undefined;
  sections.push({ header: out.Minion ? 'Player' : undefined, info, groups });

  // src/Modules/Build.lua:1817-1831
  for (const jewel of w?.JewelLimitWarning ?? []) {
    push(warnings, 'item', `You are exceeding jewel limit with the jewel ${jewel}`);
  }
  for (const slot of w?.SocketLimitWarning ?? []) {
    push(warnings, 'item', `You have too many gems in your ${slot} slot`);
  }
  if (w?.MissingAnointWarning?.length) {
    push(warnings, 'item', `You have eligible items missing an anoint: ${w.MissingAnointWarning.join(', ')}`);
  }

  // src/Modules/Build.lua:907-909
  if (out.PointsUsed) {
    for (const line of estimatePlayerProgress(out.PointsUsed, out.Output?.ExtraPoints).warnings) {
      push(warnings, 'points', line);
    }
  }

  return { info: infoLines(out.MainSkill?.InfoMessage, out.MainSkill?.InfoMessage2), sections, warnings };
};

export const buildWarnings = (out: Outputs, tick?: TickWarnings): BuildWarning[] => buildSidebar(out, tick).warnings;
