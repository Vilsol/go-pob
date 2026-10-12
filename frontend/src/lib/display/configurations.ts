import generated from './config_options_generated';
import type { calculator } from '$lib/types';

type OneOrMany = string | readonly string[];

interface RawEntry {
  var?: string;
  type?: string;
  label?: string;
  section?: string;
  col?: number;
  list?: readonly { val: unknown; label: string }[];
  defaultIndex?: number;
  defaultState?: boolean | number | string;
  defaultPlaceholderState?: number | string;
  tooltip?: string;
  dynamicTooltip?: boolean;
  valueTooltips?: Readonly<Record<string, string>>;
  ifOption?: OneOrMany;
  ifCond?: OneOrMany;
  ifMinionCond?: OneOrMany;
  ifEnemyCond?: OneOrMany;
  ifCondTrue?: OneOrMany;
  ifMult?: OneOrMany;
  ifEnemyMult?: OneOrMany;
  ifStat?: OneOrMany;
  ifEnemyStat?: OneOrMany;
  ifTagType?: OneOrMany;
  ifFlag?: OneOrMany;
  ifMod?: OneOrMany;
  ifSkill?: OneOrMany;
  ifSkillFlag?: OneOrMany;
  ifSkillData?: OneOrMany;
  ifNode?: number | readonly number[];
  includeTransfigured?: boolean;
  implyCond?: string;
  implyCondList?: readonly string[];
  implyMinionCond?: string;
  implyEnemyCond?: string;
  hideIfInvalid?: boolean;
  doNotHighlight?: boolean;
  legacy?: boolean;
}

/* ConfigTab.lua:350-630 */
export type Predicates = Omit<
  RawEntry,
  | 'var'
  | 'type'
  | 'label'
  | 'section'
  | 'col'
  | 'list'
  | 'defaultIndex'
  | 'defaultState'
  | 'defaultPlaceholderState'
  | 'tooltip'
  | 'dynamicTooltip'
  | 'valueTooltips'
>;

interface Variable extends Predicates {
  label: string;
  var: string;
  defaultState?: boolean | number | string;
  defaultPlaceholderState?: number | string;
  tooltip?: string;
  valueTooltips?: Readonly<Record<string, string>>;
}

interface VariableList extends Variable {
  type: 'list';
  defaultIndex?: number;
  list: { label: string; value: unknown }[];
}

interface VariableNumber extends Variable {
  type: 'count' | 'integer' | 'countAllowZero' | 'float';
}

interface VariableCheck extends Variable {
  type: 'check';
}

interface VariableText extends Variable {
  type: 'text';
}

export type AllVarTypes = VariableList | VariableNumber | VariableCheck | VariableText;

export interface ConfigLabel extends Predicates {
  label: string;
}

export interface ConfigSection {
  name: string;
  col: number;
  rows: (AllVarTypes | ConfigLabel)[];
  variables: AllVarTypes[];
}

export const isVariable = (row: AllVarTypes | ConfigLabel): row is AllVarTypes => 'var' in row;

const predicateKeys = [
  'ifOption',
  'ifCond',
  'ifMinionCond',
  'ifEnemyCond',
  'ifCondTrue',
  'ifMult',
  'ifEnemyMult',
  'ifStat',
  'ifEnemyStat',
  'ifTagType',
  'ifFlag',
  'ifMod',
  'ifSkill',
  'ifSkillFlag',
  'ifSkillData',
  'ifNode',
  'includeTransfigured',
  'implyCond',
  'implyCondList',
  'implyMinionCond',
  'implyEnemyCond',
  'hideIfInvalid',
  'doNotHighlight',
  'legacy'
] as const;

const predicatesOf = (e: RawEntry): Predicates => {
  const out: Record<string, unknown> = {};
  for (const k of predicateKeys) {
    if (e[k] !== undefined) out[k] = e[k];
  }
  return out as Predicates;
};

const toVariable = (e: RawEntry): AllVarTypes => {
  const base = {
    ...predicatesOf(e),
    var: e.var!,
    label: e.label ?? '',
    defaultState: e.defaultState,
    defaultPlaceholderState: e.defaultPlaceholderState,
    tooltip: e.tooltip,
    valueTooltips: e.valueTooltips
  };
  if (e.type === 'list') {
    return { ...base, type: 'list', defaultIndex: e.defaultIndex, list: (e.list ?? []).map((l) => ({ label: l.label, value: l.val })) };
  }
  return { ...base, type: e.type as Exclude<AllVarTypes['type'], 'list'> };
};

export const configurations: ConfigSection[] = [];
for (const e of generated as readonly RawEntry[]) {
  if (e.section) {
    configurations.push({ name: e.section, col: e.col ?? 1, rows: [], variables: [] });
    continue;
  }
  const section = configurations[configurations.length - 1];
  if (e.var) {
    const v = toVariable(e);
    section.rows.push(v);
    section.variables.push(v);
  } else if (e.label) {
    section.rows.push({ ...predicatesOf(e), label: e.label });
  }
}

export const reverseConfigOptions: Record<string, AllVarTypes> = Object.fromEntries(configurations.flatMap((s) => s.variables.map((v) => [v.var, v])));

export type ConfigValue = boolean | number | string;

/* ConfigTab.lua:650-660 */
export const defaultValue = (v: AllVarTypes): ConfigValue | undefined => {
  switch (v.type) {
    case 'check':
      return v.defaultState ?? false;
    case 'list':
      return v.list[(v.defaultIndex ?? 1) - 1]?.value as ConfigValue | undefined;
    case 'text':
      return v.defaultState ?? '';
    default:
      return v.defaultState ?? 0;
  }
};

/* ConfigTab.lua:890-908, ConfigTab.lua:668 */
export const isModified = (v: AllVarTypes, inputs: Record<string, ConfigValue | undefined>, placeholders: Record<string, ConfigValue | undefined>) => {
  const cur = inputs[v.var];
  if (cur === undefined) return false;
  const def = placeholders[v.var] ?? v.defaultPlaceholderState ?? defaultValue(v);
  return cur !== def;
};

const many = (x: OneOrMany | undefined): readonly string[] => (x === undefined ? [] : typeof x === 'string' ? [x] : x);

export const configRelevanceQuery: calculator.ConfigRelevance = (() => {
  const sets = {
    Conds: new Set<string>(),
    MinionConds: new Set<string>(),
    EnemyConds: new Set<string>(),
    CondTrue: new Set<string>(),
    Mults: new Set<string>(),
    EnemyMults: new Set<string>(),
    Stats: new Set<string>(),
    EnemyStats: new Set<string>(),
    TagTypes: new Set<string>(),
    Flags: new Set<string>(),
    Mods: new Set<string>(),
    Skills: new Set<string>(),
    TransfiguredSkills: new Set<string>(),
    SkillFlags: new Set<string>(),
    SkillData: new Set<string>()
  };
  const add = (set: Set<string>, x: OneOrMany | undefined) => many(x).forEach((n) => set.add(n));
  for (const s of configurations) {
    for (const r of s.rows) {
      add(sets.Conds, r.ifCond);
      add(sets.Conds, r.implyCond);
      add(sets.Conds, r.implyCondList);
      add(sets.MinionConds, r.ifMinionCond);
      add(sets.MinionConds, r.implyMinionCond);
      add(sets.EnemyConds, r.ifEnemyCond);
      add(sets.EnemyConds, r.implyEnemyCond);
      add(sets.CondTrue, r.ifCondTrue);
      add(sets.Mults, r.ifMult);
      add(sets.EnemyMults, r.ifEnemyMult);
      add(sets.Stats, r.ifStat);
      add(sets.EnemyStats, r.ifEnemyStat);
      add(sets.TagTypes, r.ifTagType);
      add(sets.Flags, r.ifFlag);
      add(sets.Mods, r.ifMod);
      add(r.includeTransfigured ? sets.TransfiguredSkills : sets.Skills, r.ifSkill);
      add(sets.SkillFlags, r.ifSkillFlag);
      add(sets.SkillData, r.ifSkillData);
    }
  }
  return Object.fromEntries(Object.entries(sets).map(([k, v]) => [k, [...v].sort()])) as unknown as calculator.ConfigRelevance;
})();

export type RelevanceSets = Record<keyof calculator.ConfigRelevance, Set<string>>;

export const relevanceSets = (r: calculator.ConfigRelevance | undefined): RelevanceSets | undefined =>
  r && (Object.fromEntries(Object.keys(configRelevanceQuery).map((k) => [k, new Set(r[k as keyof calculator.ConfigRelevance] ?? [])])) as RelevanceSets);

/* ConfigTab.lua:641 */
const inputTruthy = (name: string, inputs: Record<string, ConfigValue | undefined>) => {
  let value = inputs[name];
  if (value === undefined) {
    const v = reverseConfigOptions[name];
    value = v && (v.type === 'list' ? (v.defaultIndex ? defaultValue(v) : undefined) : v.defaultState);
  }
  return value !== undefined && value !== false;
};

/* ConfigTab.lua:350-630 */
export const isRelevant = (row: AllVarTypes | ConfigLabel, inputs: Record<string, ConfigValue | undefined>, rel: RelevanceSets): boolean => {
  const implied = () => {
    if (!isVariable(row) || !inputTruthy(row.var, inputs)) return false;
    return (
      many(row.implyCondList).some((c) => rel.Conds.has(c)) ||
      (row.implyCond !== undefined && rel.Conds.has(row.implyCond)) ||
      (row.implyMinionCond !== undefined && rel.MinionConds.has(row.implyMinionCond)) ||
      (row.implyEnemyCond !== undefined && rel.EnemyConds.has(row.implyEnemyCond))
    );
  };
  const check = (x: OneOrMany | undefined, pass: (n: string) => boolean) => x === undefined || many(x).some(pass);
  const withImply = (x: OneOrMany | undefined, set: Set<string>) => check(x, (n) => implied() || set.has(n));
  return (
    check(row.ifOption, (n) => inputTruthy(n, inputs)) &&
    withImply(row.ifCond, rel.Conds) &&
    withImply(row.ifMinionCond, rel.MinionConds) &&
    withImply(row.ifEnemyCond, rel.EnemyConds) &&
    check(row.ifCondTrue, (n) => rel.CondTrue.has(n)) &&
    withImply(row.ifMult, rel.Mults) &&
    withImply(row.ifEnemyMult, rel.EnemyMults) &&
    withImply(row.ifStat, rel.Stats) &&
    withImply(row.ifEnemyStat, rel.EnemyStats) &&
    withImply(row.ifTagType, rel.TagTypes) &&
    check(row.ifFlag, (n) => rel.Flags.has(n)) &&
    withImply(row.ifMod, rel.Mods) &&
    check(row.ifSkill, (n) => (row.includeTransfigured ? rel.TransfiguredSkills : rel.Skills).has(n)) &&
    check(row.ifSkillFlag, (n) => rel.SkillFlags.has(n)) &&
    check(row.ifSkillData, (n) => rel.SkillData.has(n))
  );
};

const SHOW_ALL_EXCLUDED_KEYWORDS = ['recently', 'in the last', 'in the past', 'in last', 'in past', 'pvp'];

/* ConfigVisibility.lua:144-155 */
export const isShowAllExcluded = (row: AllVarTypes | ConfigLabel) => {
  if (row.ifOption || row.ifSkill || row.ifSkillData || row.ifSkillFlag || row.legacy) return true;
  const label = row.label.toLowerCase();
  return SHOW_ALL_EXCLUDED_KEYWORDS.some((kw) => label.includes(kw));
};

// ConfigOptions.lua:61-72
const OUTPUT_TOOLTIPS: Record<string, (output: Record<string, number>) => string> = {
  // src/Modules/ConfigOptions.lua:61-65
  conditionLowLife: (o) =>
    `You will automatically be considered to be on Low ^xE05030Life ^7if you have at least ${100 - (o.LowLifePercentage ?? 0)}% ^xE05030Life ^7reserved` +
    '\nbut you can use this option to force it if necessary.',
  // src/Modules/ConfigOptions.lua:67-72
  conditionFullLife: (o) =>
    `You can be considered to be on Full ^xE05030Life ^7if you have at least ${o.FullLifePercentage ?? 0}% ^xE05030Life ^7left.` +
    '\nYou will automatically be considered to be on Full ^xE05030Life ^7if you have Chaos Inoculation,' +
    '\nbut you can use this option to force it if necessary.'
};

/* ConfigTab.lua:334-349 */
export function optionTooltip(v: AllVarTypes, value: ConfigValue | undefined, output: Record<string, number> = {}): string | undefined {
  const parts = [OUTPUT_TOOLTIPS[v.var]?.(output) ?? v.tooltip, value === undefined ? undefined : v.valueTooltips?.[String(value)]];
  const text = parts.filter((p): p is string => !!p).join('\n');
  return text || undefined;
}
