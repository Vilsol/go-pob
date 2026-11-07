// The PoE2 Calcs page: PoB-PoE2's CalcsTab:CheckFlag, CalcFormat.lua's formatCalcStr and the CalcSectionControl
// layout, over calc_sections2.ts, with the breakdowns calculator2 builds.

import type { CalcSection2, CalcSectionColumn2, CalcSectionEntry2, CalcSectionRow2, FlagGated2 } from '$lib/calcs/calc_section_types2';
import {
  cellId,
  elementsOf,
  type CalcBlock,
  type CalcBreakdown,
  type CalcCell,
  type CalcModSection,
  type CalcRow,
  type CalcSectionView,
  type CalcsView
} from './calcs';
import type { exposition } from '$lib/types2';
import { formatNumSep } from './format_stat';
import { luaToString } from './saved_stats';
import { breakdownIndex, type SidebarCell } from './sidebar_breakdown';
import type { CalcSection } from '$lib/calcs/calc_sections';

export type CalcsOutputValue2 = number | boolean | string;

export type CalcsActor2 = {
  // output with its true outputs, and the MainHand and OffHand tables when the main skill attacks with that hand
  output: Record<string, CalcsOutputValue2 | Record<string, CalcsOutputValue2>>;
  skillFlags: Record<string, boolean>;
  // the main skill's skillData keys Lua reads as truthy
  skillData: Record<string, boolean>;
  // the {p:mod:n} values, by cellId:<entry index>
  mods: Record<string, number>;
  // the actor's breakdowns, by the name a cell entry gives
  breakdowns?: Record<string, CalcBreakdown>;
};

const GEAR_ONLY = '#gearOnly';

/* CalcBreakdownControl.lua:147-297 (PoB-PoE2): the worker's breakdowns as the breakdown panel shows them; one with
   slots also as a gearOnly cell shows it, under <name>#gearOnly */
export const calcsBreakdowns2 = (bds: Record<string, Readonly<exposition.CalcsBreakdown>>): Record<string, CalcBreakdown> => {
  const out: Record<string, CalcBreakdown> = {};
  for (const [k, b] of Object.entries(bds)) {
    const shown = (gearOnly: boolean) => {
      const bd: CalcBreakdown = { lines: [...(b.Lines ?? [])] };
      if (b.Radius) bd.radius = b.Radius;
      // the first table without a label is the panel's sources table; the reservations table is named for it
      (b.Tables ?? []).forEach((table, i) => {
        const t = gearOnly && table.Columns?.[0]?.Label === 'Base' && table.Columns[0].Key === 'base' ? b.GearTable : table;
        const columns = (t.Columns ?? []).map((c) => ({ label: c.Label, key: c.Key }));
        const rows = (t.Rows ?? []).map((r) => Object.fromEntries(columns.map((c) => [c.key, r?.[c.key] ?? ''])));
        const label = t.Label || (columns[0]?.key === 'skillName' ? 'Reservations' : '');
        if (!label && !bd.columns) {
          bd.columns = columns;
          bd.rows = rows;
        } else (bd.tables ??= []).push({ label: label || `Table ${i + 1}`, columns, rows });
      });
      return bd;
    };
    out[k] = shown(false);
    if (b.Gear) out[k + GEAR_ONLY] = shown(true);
  }
  return out;
};

export type CalcsModProps2 = { Cfg: string; ModSource: string; Enemy: boolean; ModName: string[]; ModType: string; Actor: string };

const namespaced = /^([a-zA-Z]+)\.([a-zA-Z]+)$/;

/* CalcFormat.lua:17-23 (PoB-PoE2): output[c], or output[ns][var] for an ns.var name */
const outputOf = (actor: CalcsActor2, c: string): CalcsOutputValue2 | undefined => {
  const m = namespaced.exec(c);
  if (!m) {
    const v = actor.output[c];
    return typeof v === 'object' ? undefined : v;
  }
  const table = actor.output[m[1]];
  return typeof table === 'object' ? table[m[2]] : undefined;
};

/* CalcsTab.lua:433-478 (PoB-PoE2) */
export const checkFlag2 = (obj: FlagGated2, actor: CalcsActor2): boolean => {
  const skillFlags = actor.skillFlags;
  const skillData = actor.skillData;
  if (obj.flag && !skillFlags[obj.flag]) return false;
  if (obj.skillData && !skillData[obj.skillData]) return false;
  if (obj.flagList) for (const flag of obj.flagList) if (!skillFlags[flag]) return false;
  if (obj.playerFlag && !skillFlags[obj.playerFlag]) return false;
  if (obj.notFlag && skillFlags[obj.notFlag]) return false;
  if (obj.notSkillData && skillData[obj.notSkillData]) return false;
  if (obj.notFlagList) for (const flag of obj.notFlagList) if (skillFlags[flag]) return false;
  if (obj.haveOutput) {
    const v = outputOf(actor, obj.haveOutput);
    if (v === undefined || v === false || v === 0) return false;
  }
  return true;
};

const rowsOf = <T>(t: { [n: number]: T }): T[] => elementsOf<T>(t);

/* Common.lua:722-728 (PoB-PoE2) */
const round = (val: number, dec: number) => Math.floor(val * 10 ** dec + 0.5) / 10 ** dec;

/* CalcFormat.lua:12-14 (PoB-PoE2) */
const formatCalcVal = (val: number, p: number) => formatNumSep(luaToString(round(val, p)));

const numberOf = (v: number | boolean | string | undefined) => (typeof v === 'number' ? v : 0);

/* CalcFormat.lua:16-80 (PoB-PoE2); id is the cell's cellId, the key its entries' mod values are under */
export const formatCalcStr2 = (str: string, actor: CalcsActor2, colData?: CalcSectionColumn2, id?: string): string => {
  str = str.replace(/\{output:([a-zA-Z.:]+)\}/g, (_, c: string) => {
    const v = outputOf(actor, c);
    return v === undefined || v === false ? '' : typeof v === 'number' ? luaToString(v) : String(v);
  });
  str = str.replace(/\{(\d+):output:([a-zA-Z.:]+)\}/g, (_, p: string, c: string) => formatCalcVal(numberOf(outputOf(actor, c)), Number(p)));
  str = str.replace(/\{(\d+):mod:([\d,]+)\}/g, (_, p: string, n: string) => {
    const numList = (n.match(/\d+/g) ?? []).map(Number);
    const first = colData?.[numList[0] - 1];
    if (!first || !first.modType) return '?';
    const modType = first.modType;
    let modTotal = modType === 'MORE' ? 1 : 0;
    for (const num of numList) {
      if (!colData?.[num - 1]) break;
      const modVal = actor.mods[`${id}:${num - 1}`] ?? 0;
      modTotal = modType === 'MORE' ? modTotal * modVal : modTotal + modVal;
    }
    if (modType === 'MORE') modTotal = (modTotal - 1) * 100;
    return formatCalcVal(modTotal, Number(p));
  });
  return str;
};

/* CalcFormat.lua:39-60 (PoB-PoE2): the entries a {p:mod:n} cell combines, for the worker to evaluate */
export const calcsModProps2 = (sections: CalcSection2[]): Record<string, CalcsModProps2> => {
  const out: Record<string, CalcsModProps2> = {};
  for (const [, id, , , subs] of sections)
    subs.forEach((sub, j) =>
      rowsOf<CalcSectionRow2>(sub.data).forEach((row, k) =>
        rowsOf<CalcSectionColumn2>(row).forEach((col, l) =>
          rowsOf<CalcSectionEntry2>(col).forEach((e, m) => {
            if (!e.modName || !e.modType) return;
            out[`${cellId(id, j, k, l)}:${m}`] = {
              Cfg: e.cfg ?? '',
              ModSource: e.modSource ?? '',
              Enemy: !!e.enemy,
              ModName: [e.modName].flat(),
              ModType: e.modType,
              Actor: e.actor ?? ''
            };
          })
        )
      )
    );
  return out;
};

/* CalcBreakdownControl.lua:300-325 (PoB-PoE2): what AddModSection reads of an entry */
export const modSectionData2 = (e: CalcSectionEntry2): CalcModSection['sd'] => ({
  ModName: [e.modName ?? []].flat(),
  Multi: typeof e.modName !== 'string',
  ModType: e.modType ?? '',
  Cfg: e.cfg ?? '',
  ModSource: e.modSource ?? '',
  Enemy: !!e.enemy,
  Actor: e.actor ?? ''
});

const modLabel = (e: CalcSectionEntry2) => {
  const all = [e.modName ?? []].flat();
  return `${all.slice(0, 3).join(', ')}${all.length > 3 ? ` +${all.length - 3}` : ''} ${e.modType}${e.enemy ? ' (enemy)' : ''}`;
};

/* CalcSectionControl.lua:113-160, 500-600 (PoB-PoE2) */
export const calcsView2 = (sections: CalcSection2[], actor: CalcsActor2): CalcsView => {
  const breakdowns: Record<string, CalcBreakdown> = {};
  const view = sections.map(([width, id, col, color, subs]): CalcSectionView => {
    const first = subs[0];
    const head = { id, title: first?.label ?? id, wide: width === 3, col, color };
    // CalcSectionControl.lua:28-29, 115: the section's flag and notFlag are its first subsection's
    if (!first || !checkFlag2({ flag: first.data.flag, notFlag: first.data.notFlag }, actor)) return { ...head, applicable: false };
    let enabled = false;
    const blocks = subs.map((sub, j): CalcBlock => {
      const rows: CalcRow[] = [];
      rowsOf<CalcSectionRow2>(sub.data).forEach((row, k) => {
        if (!checkFlag2(row, actor)) return;
        enabled = true;
        const cells = rowsOf<CalcSectionColumn2>(row).map((c, l): CalcCell => {
          if (c.format === undefined || !checkFlag2(c, actor)) return { text: '' };
          const cell: CalcCell = { text: formatCalcStr2(c.format, actor, c, cellId(id, j, k, l)) };
          // CalcBreakdownControl.lua:75-83, 244-257: the first breakdown entry a cell names
          const entry = rowsOf<CalcSectionEntry2>(c).find((e) => e.breakdown);
          const bd = entry && (entry.gearOnly && actor.breakdowns?.[entry.breakdown + GEAR_ONLY] ? entry.breakdown + GEAR_ONLY : entry.breakdown);
          if (bd && actor.breakdowns?.[bd]) {
            breakdowns[bd] = actor.breakdowns[bd];
            cell.bd = bd;
          }
          const mods = rowsOf<CalcSectionEntry2>(c).filter((e) => e.modName && e.modType);
          if (mods.length) cell.mods = mods.map(modLabel);
          // CalcBreakdownControl.lua:75-83: each mod name entry whose flags hold is a mod table
          const modEntries = rowsOf<CalcSectionEntry2>(c).filter((e) => e.modName && !e.breakdown && checkFlag2(e, actor));
          if (modEntries.length) cell.sections = modEntries.map((e) => ({ label: e.label, sd: modSectionData2(e) }));
          return cell;
        });
        rows.push({ label: row.label ?? '', cells });
      });
      return { label: sub.label, extra: sub.data.extra ? formatCalcStr2(sub.data.extra, actor) : undefined, rows };
    });
    if (!enabled) return { ...head, applicable: false };
    // CalcSectionControl.lua:506-525: every subsection draws its label and extra, rows or none
    return { ...head, applicable: true, blocks };
  });
  return { sections: view, breakdowns };
};

/* Build.lua:2278-2331 (PoB-PoE2): the cells a sidebar line shows for breakdown key, whose Calcs cell is source's */
export const sidebarBreakdown2 = (sections: CalcSection2[], key: string, source: string, modNames?: string[], ignoredSections?: string[]): SidebarCell[] => {
  const entry = breakdownIndex(sections as unknown as CalcSection[]).get(source);
  if (!entry) {
    const output: SidebarCell[] = [{ breakdown: key }];
    if (modNames) output.push({ modName: modNames });
    return output;
  }
  const displayData: SidebarCell[] = elementsOf<SidebarCell>(entry.colData as never).map((cell) => ({
    ...cell,
    ...(cell.breakdown === source && { breakdown: key })
  }));
  for (const modKey of modNames ?? []) displayData.push(...(entry.modSectionsByName[modKey] ?? []));
  const visitedSections = new Set<string>();
  for (let i = displayData.length - 1; i >= 0; i--) {
    const label = displayData[i].label;
    if (label) {
      if (visitedSections.has(label) || ignoredSections?.includes(label)) displayData.splice(i, 1);
      visitedSections.add(label);
    }
  }
  return displayData.length ? displayData : [{ breakdown: key }];
};
