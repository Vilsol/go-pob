import {
  configurations,
  defaultValue,
  isModified,
  isRelevant,
  isShowAllExcluded,
  isVariable,
  optionTooltip,
  relevanceSets,
  reverseConfigOptions,
  type AllVarTypes,
  type ConfigLabel,
  type ConfigSection,
  type ConfigValue,
  type RelevanceSets
} from '$lib/display/configurations';
import type { calculator } from '$lib/types';
import type { StatCompareSection } from '$lib/skill_tree/stat_compare';
import { pobPlain } from './pob_text';

export type Inputs = Record<string, ConfigValue | undefined>;

export const MOST_USED = [
  'usePowerCharges',
  'overridePowerCharges',
  'useFrenzyCharges',
  'useEnduranceCharges',
  'enemyIsBoss',
  'presetBossSkills',
  'enemyLevel',
  'bandit',
  'conditionMoving',
  'conditionFullLife'
];

// a game's options table and the ConfigTab rules that show it; rel is what relevanceSets makes of the calculation's
// relevance report
export type ConfigTable<Rel = unknown, Report = unknown> = {
  sections: ConfigSection[];
  options: Record<string, AllVarTypes>;
  relevanceSets(report: Report): Rel;
  isRelevant(row: AllVarTypes | ConfigLabel, inputs: Inputs, rel: Rel): boolean;
  isModified(v: AllVarTypes, inputs: Inputs, placeholders: Record<string, number | string>): boolean;
  isShowAllExcluded(row: AllVarTypes | ConfigLabel): boolean;
  // the value the control shows, and whether it is the build's own
  value(v: AllVarTypes, inputs: Inputs): ConfigValue | undefined;
  isSet(v: AllVarTypes, inputs: Inputs): boolean;
  tooltip(v: AllVarTypes, value: ConfigValue | undefined, output?: Record<string, number>): string | undefined;
};

export const poe1ConfigTable: ConfigTable<RelevanceSets, calculator.ConfigRelevance> = {
  sections: configurations,
  options: reverseConfigOptions,
  relevanceSets: (r) => relevanceSets(r)!,
  isRelevant,
  isModified,
  isShowAllExcluded,
  value: (v, inputs) => inputs[v.var] ?? defaultValue(v),
  isSet: (v, inputs) => inputs[v.var] !== undefined,
  tooltip: optionTooltip
};

export const INVALID = 'This config option is conditional with missing source and is invalid.';

export type OptionView = {
  kind: 'option';
  v: AllVarTypes;
  /* ConfigTab.lua:650-660 */
  value: ConfigValue | undefined;
  set: boolean;
  placeholder?: string;
  relevant: boolean;
  modified: boolean;
  /* ConfigTab.lua:679-698 */
  invalid: boolean;
};
export type HeadingView = { kind: 'heading'; label: string };
export type SectionView = { name: string; total: number; modified: number; hidden: number; rows: (OptionView | HeadingView)[] };

export type ConfigFilter = { q: string; showAll: boolean; modOnly: boolean };

export function configView<Rel, Report>(
  t: ConfigTable<Rel, Report>,
  inputs: Inputs,
  placeholders: Record<string, number | string>,
  relevance: Report,
  f: ConfigFilter
) {
  const rel = t.relevanceSets(relevance);
  const needle = f.q.trim().toLowerCase();
  const option = (v: AllVarTypes): OptionView => {
    const modified = t.isModified(v, inputs, placeholders);
    const relevant = t.isRelevant(v, inputs, rel);
    const ph = placeholders[v.var] ?? v.defaultPlaceholderState;
    return {
      kind: 'option',
      v,
      value: t.value(v, inputs),
      set: t.isSet(v, inputs),
      placeholder: ph === undefined ? undefined : String(ph),
      relevant,
      modified,
      invalid: modified && !relevant
    };
  };
  let shown = 0;
  let modifiedAll = 0;
  let total = 0;
  const sections: SectionView[] = [];
  for (const s of t.sections) {
    if (!s.variables.length) continue;
    const rows: (OptionView | HeadingView)[] = [];
    let modified = 0;
    let hidden = 0;
    let heading: HeadingView | undefined;
    for (const row of s.rows) {
      if (!isVariable(row)) {
        heading = t.isRelevant(row, inputs, rel) || f.showAll ? { kind: 'heading', label: row.label } : undefined;
        continue;
      }
      total++;
      const o = option(row);
      if (o.modified) modified++;
      // ConfigTab.lua:320-332
      const shownRaw = o.relevant || (f.showAll && !t.isShowAllExcluded(row));
      o.invalid = o.modified && !shownRaw;
      const visible = shownRaw || (o.invalid && !row.hideIfInvalid);
      if (!visible) hidden++;
      if (!visible || (needle && !pobPlain(row.label).toLowerCase().includes(needle)) || (f.modOnly && !o.modified)) continue;
      if (heading) rows.push(heading);
      heading = undefined;
      rows.push(o);
    }
    modifiedAll += modified;
    shown += rows.filter((r) => r.kind === 'option').length;
    sections.push({ name: s.name, total: s.variables.length, modified, hidden, rows });
  }
  const most = MOST_USED.map((k) => t.options[k])
    .filter(Boolean)
    .map(option);
  return { sections, most, summary: `${shown} shown · ${modifiedAll} modified · ${total} in total` };
}

export const listIndex = (v: AllVarTypes & { type: 'list' }, value: ConfigValue | undefined) =>
  Math.max(
    0,
    v.list.findIndex((l) => l.value === value)
  );

export type CustomOps = {
  onselect: (g: number) => void;
  ontoggle: (g: number, enabled: boolean) => void;
  onrename: (g: number, title: string) => void;
  onaddgroup: () => void;
  ondeletegroup: (g: number) => void;
  oncell: (g: number, c: number, text: string) => void;
  oncommit: (g: number, c: number, text: string) => void;
  onmove: (g: number, c: number, by: number) => void;
  onremove: (g: number, c: number) => void;
  onaddcell: (g: number) => void;
  /* ConfigTab.lua:53-68 */
  togglewhatif?: (g: number) => Promise<StatCompareSection[] | undefined>;
  /* ConfigTab.lua:42-44, 1305-1550 */
  modbrowser?: { list: (search: string) => Promise<string[]>; sources: (line: string) => Promise<string[]>; onadd: (g: number, line: string) => void };
};
