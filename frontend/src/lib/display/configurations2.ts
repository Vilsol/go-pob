// The PoE2 config page's options (PoB-PoE2's ConfigOptions.lua) and ConfigTab's visibility rules.

import generated from './config_options_generated2';
import type { calculator2 } from '$lib/types2';

type OneOrMany = string | number | readonly (string | number)[];

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
  ifMult?: OneOrMany;
  ifEnemyMult?: OneOrMany;
  ifStat?: OneOrMany;
  ifEnemyStat?: OneOrMany;
  ifTagType?: OneOrMany;
  ifFlag?: OneOrMany;
  ifPlayerFlag?: OneOrMany;
  ifMinionFlag?: OneOrMany;
  ifMod?: OneOrMany;
  ifSkill?: OneOrMany;
  ifSkillType?: OneOrMany;
  ifSkillFlag?: OneOrMany;
  ifSkillData?: OneOrMany;
  ifGemFamily?: OneOrMany;
  includeTransfigured?: boolean;
  implyCond?: string;
  implyCondList?: readonly string[];
  implyMinionCond?: string;
  implyEnemyCond?: string;
  hideIfInvalid?: boolean;
  legacy?: boolean;
}

export type ConfigValue2 = boolean | number | string;

export interface Option2 extends RawEntry {
  label: string;
}

export interface Variable2 extends Option2 {
  var: string;
  type: 'check' | 'count' | 'integer' | 'countAllowZero' | 'float' | 'list';
}

export interface ConfigSection2 {
  name: string;
  col: number;
  rows: Option2[];
}

export const isVariable2 = (row: Option2): row is Variable2 => row.var !== undefined;

export const configurations2: ConfigSection2[] = [];
for (const e of generated as readonly RawEntry[]) {
  if (e.section) {
    configurations2.push({ name: e.section, col: e.col ?? 1, rows: [] });
  } else if (e.var || e.label) {
    configurations2.at(-1)!.rows.push({ ...e, label: e.label ?? '' });
  }
}

export const options2: Record<string, Variable2> = Object.fromEntries(configurations2.flatMap((s) => s.rows.filter(isVariable2).map((v) => [v.var, v])));

/* ConfigTab.lua:1323-1330 */
const seededInput = (v: Variable2): ConfigValue2 | undefined =>
  v.defaultIndex ? (v.list?.[v.defaultIndex - 1]?.val as ConfigValue2 | undefined) : v.defaultState;

/* ConfigTab.lua:722-732 */
const defaultState = (v: Variable2): ConfigValue2 | undefined => {
  switch (v.type) {
    case 'check':
      return v.defaultState ?? false;
    case 'list':
      return v.list?.[(v.defaultIndex ?? 1) - 1]?.val as ConfigValue2 | undefined;
    default:
      return v.defaultState ?? 0;
  }
};

/* ConfigTab.lua:980-998 */
export const defaultStateOf = (name: string, value: ConfigValue2 | undefined): ConfigValue2 | undefined => {
  const v = options2[name];
  if (v?.defaultPlaceholderState !== undefined) return v.defaultPlaceholderState;
  if (v) {
    const d = defaultState(v);
    if (d !== undefined) return d;
  }
  switch (typeof value) {
    case 'number':
      return 0;
    case 'boolean':
      return false;
    case 'string':
      return '';
  }
  return undefined;
};

// the value a control shows: the saved input, else the one CreateConfigSet seeds
export const currentValue = (v: Variable2, inputs: Record<string, ConfigValue2 | undefined>): ConfigValue2 | undefined =>
  inputs[v.var] !== undefined ? inputs[v.var] : seededInput(v);

const inputTruthy = (name: string, inputs: Record<string, ConfigValue2 | undefined>) => {
  const v = options2[name];
  const value = inputs[name] !== undefined ? inputs[name] : v && seededInput(v);
  return value !== undefined && value !== false;
};

const many = (x: OneOrMany | undefined): string[] => (x === undefined ? [] : Array.isArray(x) ? x.map(String) : [String(x)]);

export const relevanceQuery2: calculator2.ConfigRelevance = (() => {
  const sets: Record<keyof calculator2.ConfigRelevance, Set<string>> = {
    Conds: new Set(),
    MinionConds: new Set(),
    EnemyConds: new Set(),
    Mults: new Set(),
    EnemyMults: new Set(),
    Stats: new Set(),
    EnemyStats: new Set(),
    TagTypes: new Set(),
    Flags: new Set(),
    PlayerFlags: new Set(),
    MinionFlags: new Set(),
    Mods: new Set(),
    Skills: new Set(),
    TransfiguredSkills: new Set(),
    SkillTypes: new Set(),
    SkillFlags: new Set(),
    SkillData: new Set(),
    GemFamilies: new Set()
  };
  const add = (k: keyof calculator2.ConfigRelevance, x: OneOrMany | undefined) => many(x).forEach((n) => sets[k].add(n));
  for (const s of configurations2) {
    for (const r of s.rows) {
      add('Conds', r.ifCond);
      add('Conds', r.implyCond);
      add('Conds', r.implyCondList);
      add('MinionConds', r.ifMinionCond);
      add('MinionConds', r.implyMinionCond);
      add('EnemyConds', r.ifEnemyCond);
      add('EnemyConds', r.implyEnemyCond);
      add('Mults', r.ifMult);
      add('EnemyMults', r.ifEnemyMult);
      add('Stats', r.ifStat);
      add('EnemyStats', r.ifEnemyStat);
      add('TagTypes', r.ifTagType);
      if (r.ifPlayerFlag !== undefined) add('PlayerFlags', r.ifPlayerFlag);
      else add('Flags', r.ifFlag);
      add('MinionFlags', r.ifMinionFlag);
      add('Mods', r.ifMod);
      add(r.includeTransfigured ? 'TransfiguredSkills' : 'Skills', r.ifSkill);
      add('SkillTypes', r.ifSkillType);
      add('SkillFlags', r.ifSkillFlag);
      add('SkillData', r.ifSkillData);
      add('GemFamilies', r.ifGemFamily);
    }
  }
  return Object.fromEntries(Object.entries(sets).map(([k, v]) => [k, [...v].sort()])) as unknown as calculator2.ConfigRelevance;
})();

export type RelevanceSets2 = Record<keyof calculator2.ConfigRelevance, Set<string>>;

export const relevanceSets2 = (r: calculator2.ConfigRelevance): RelevanceSets2 =>
  Object.fromEntries(Object.keys(relevanceQuery2).map((k) => [k, new Set(r[k as keyof calculator2.ConfigRelevance] ?? [])])) as RelevanceSets2;

/* ConfigTab.lua:221-239, 362-374, 392-702 */
export const isEligible2 = (row: Option2, inputs: Record<string, ConfigValue2 | undefined>, rel: RelevanceSets2): boolean => {
  const implied = () =>
    isVariable2(row) &&
    inputTruthy(row.var, inputs) &&
    ((row.implyCondList ?? []).some((c) => rel.Conds.has(c)) ||
      (row.implyCond !== undefined && rel.Conds.has(row.implyCond)) ||
      (row.implyMinionCond !== undefined && rel.MinionConds.has(row.implyMinionCond)) ||
      (row.implyEnemyCond !== undefined && rel.EnemyConds.has(row.implyEnemyCond)));
  const check = (x: OneOrMany | undefined, pass: (n: string) => boolean) => x === undefined || many(x).some(pass);
  const withImply = (x: OneOrMany | undefined, set: Set<string>) => check(x, (n) => implied() || set.has(n));
  const flag = row.ifPlayerFlag ?? row.ifFlag;
  return (
    check(row.ifOption, (n) => inputTruthy(n, inputs)) &&
    withImply(row.ifCond, rel.Conds) &&
    withImply(row.ifMinionCond, rel.MinionConds) &&
    withImply(row.ifEnemyCond, rel.EnemyConds) &&
    withImply(row.ifMult, rel.Mults) &&
    withImply(row.ifEnemyMult, rel.EnemyMults) &&
    check(row.ifStat, (n) => implied() || rel.Stats.has(n)) &&
    withImply(row.ifEnemyStat, rel.EnemyStats) &&
    withImply(row.ifTagType, rel.TagTypes) &&
    check(flag, (n) => (row.ifPlayerFlag !== undefined ? rel.PlayerFlags : rel.Flags).has(n)) &&
    check(row.ifMinionFlag, (n) => rel.MinionFlags.has(n)) &&
    withImply(row.ifMod, rel.Mods) &&
    check(row.ifSkill, (n) => (row.includeTransfigured ? rel.TransfiguredSkills : rel.Skills).has(n)) &&
    check(row.ifSkillType, (n) => rel.SkillTypes.has(n)) &&
    check(row.ifSkillFlag, (n) => rel.SkillFlags.has(n)) &&
    check(row.ifSkillData, (n) => rel.SkillData.has(n)) &&
    check(row.ifGemFamily, (n) => rel.GemFamilies.has(n))
  );
};

/* ConfigTab.lua:211-219 */
const showAllApplies = (row: Option2) => row.ifOption === undefined && !row.legacy;

/* ConfigTab.lua:736-760 */
export const isModified2 = (v: Variable2, inputs: Record<string, ConfigValue2 | undefined>) => {
  const cur = inputs[v.var];
  return cur !== undefined && cur !== defaultStateOf(v.var, cur);
};

/* ConfigTab.lua:362-374, 751-760 */
export const isShown2 = (row: Option2, inputs: Record<string, ConfigValue2 | undefined>, rel: RelevanceSets2, showAll: boolean): boolean => {
  const shown = isEligible2(row, inputs, rel) || (showAll && showAllApplies(row));
  if (isVariable2(row) && !row.hideIfInvalid) {
    return shown || isModified2(row, inputs);
  }
  return shown;
};
