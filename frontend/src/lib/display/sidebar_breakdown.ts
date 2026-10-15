// Build.lua:1764-1908
import type { CalcDataCol, CalcDataColProp, CalcDataRow, CalcSection } from '$lib/calcs/calc_sections';
import { rowsOf, type FlagGated } from '$lib/calcs/check_flag';
import { elementsOf, type CalcBreakdown } from './calcs';
import type { Stat } from './stats';
import type { calculator } from '$lib/types';

export type SidebarStatRef = Pick<Stat, 'childStat' | 'breakdown' | 'modNames' | 'ignoredSections'> & { stat: string; minion?: boolean };

export type SidebarCell = Omit<CalcDataColProp, 'modName'> & FlagGated & { modName?: string | string[] };

export type SidebarSection = { label?: string; bd?: CalcBreakdown; mods?: calculator.ModSectionRow[] };

type IndexEntry = { colData: CalcDataCol; modSectionsByName: Record<string, SidebarCell[]> };

const rowFields = (row: CalcDataRow) => Object.fromEntries(Object.entries(row).filter(([k]) => isNaN(parseInt(k))));

const indexes = new WeakMap<CalcSection[], Map<string, IndexEntry>>();

// src/Modules/Build.lua:1764-1811
export function breakdownIndex(sections: CalcSection[]): Map<string, IndexEntry> {
  const cached = indexes.get(sections);
  if (cached) return cached;
  const index = new Map<string, IndexEntry>();
  for (const section of sections) {
    for (const subSection of section[4]) {
      const modSectionsByName: Record<string, SidebarCell[]> = {};
      const breakdownCells = new Map<string, CalcDataCol>();
      for (const row of rowsOf(subSection.data)) {
        for (const colData of elementsOf<CalcDataCol>(row)) {
          for (const cell of elementsOf<CalcDataColProp>(colData)) {
            if (!cell.skipSideBarIndex && cell.breakdown !== undefined && !breakdownCells.has(cell.breakdown)) {
              breakdownCells.set(cell.breakdown, colData);
            }
            if (cell.modName) {
              const copy: SidebarCell = { ...rowFields(row), ...cell };
              for (const name of [cell.modName].flat()) {
                (modSectionsByName[name] ??= []).push(copy);
              }
            }
          }
        }
      }
      for (const [key, colData] of breakdownCells) {
        if (!index.has(key)) index.set(key, { colData, modSectionsByName });
      }
    }
  }
  indexes.set(sections, index);
  return index;
}

// src/Modules/Build.lua:1816-1869
export function sidebarBreakdown(sections: CalcSection[], key: string, modNames?: string[], ignoredSections?: string[]): SidebarCell[] {
  const entry = breakdownIndex(sections).get(key);
  if (!entry) {
    const output: SidebarCell[] = [{ breakdown: key }];
    if (modNames) output.push({ modName: modNames });
    return output;
  }
  const displayData: SidebarCell[] = elementsOf<CalcDataColProp>(entry.colData).map((cell) => ({ ...cell }));
  for (const modKey of modNames ?? []) {
    displayData.push(...(entry.modSectionsByName[modKey] ?? []));
  }
  const visitedSections = new Set<string>();
  for (let i = displayData.length - 1; i >= 0; i--) {
    const label = displayData[i].label;
    if (label) {
      if (visitedSections.has(label) || ignoredSections?.includes(label)) displayData.splice(i, 1);
      visitedSections.add(label);
    }
  }
  return displayData.length ? displayData : [{ breakdown: key }];
}
