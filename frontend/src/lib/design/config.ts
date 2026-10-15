import {
  configurations,
  defaultValue,
  isModified,
  isRelevant,
  isShowAllExcluded,
  isVariable,
  relevanceSets,
  reverseConfigOptions,
  type AllVarTypes,
  type ConfigValue
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

export function configView(inputs: Inputs, placeholders: Record<string, number | string>, relevance: calculator.ConfigRelevance, f: ConfigFilter) {
  const rel = relevanceSets(relevance)!;
  const needle = f.q.trim().toLowerCase();
  const option = (v: AllVarTypes): OptionView => {
    const modified = isModified(v, inputs, placeholders);
    const relevant = isRelevant(v, inputs, rel);
    const ph = placeholders[v.var] ?? v.defaultPlaceholderState;
    return {
      kind: 'option',
      v,
      value: inputs[v.var] ?? defaultValue(v),
      set: inputs[v.var] !== undefined,
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
  for (const s of configurations) {
    if (!s.variables.length) continue;
    const rows: (OptionView | HeadingView)[] = [];
    let modified = 0;
    let hidden = 0;
    let heading: HeadingView | undefined;
    for (const row of s.rows) {
      if (!isVariable(row)) {
        heading = isRelevant(row, inputs, rel) || f.showAll ? { kind: 'heading', label: row.label } : undefined;
        continue;
      }
      total++;
      const o = option(row);
      if (o.modified) modified++;
      // ConfigTab.lua:320-332
      const shownRaw = o.relevant || (f.showAll && !isShowAllExcluded(row));
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
  const most = MOST_USED.map((k) => reverseConfigOptions[k])
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
