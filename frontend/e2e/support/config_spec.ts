import { formatColors } from '../../src/lib/display/colors';
import type { Page } from '@playwright/test';
import { configurations, type AllVarTypes } from '../../src/lib/display/configurations';

const GATES = [
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
  'ifNode'
] as const;

/* ConfigTab.lua:350-630 */
const gated = (v: AllVarTypes) => GATES.some((g) => v[g] !== undefined);

const stripColors = (s: string) =>
  formatColors(s)
    .replace(/<[^>]*>/g, '')
    .trim();

export type ConfigCheck = { key: string; name: string; defaultState: boolean };
export type ConfigCount = { key: string; name: string; min: number };

const unconditional = () => configurations.flatMap((section) => section.variables.filter((v) => !gated(v)));

export const unconditionalConfigCounts = (): ConfigCount[] => {
  const all = unconditional();
  const names = all.map((v) => stripColors(v.label));
  return all
    .map((v, i) => ({ v, name: names[i] }))
    .filter(({ v, name }) => (v.type === 'count' || v.type === 'integer' || v.type === 'countAllowZero') && names.indexOf(name) === names.lastIndexOf(name))
    .map(({ v, name }) => ({ key: v.var, name, min: v.type === 'countAllowZero' ? 0 : 1 }));
};

export const unconditionalConfigChecks = (): ConfigCheck[] => {
  const seen = new Map<string, number>();
  const out: ConfigCheck[] = [];
  for (const section of configurations) {
    for (const v of section.variables) {
      if (v.type !== 'check') continue;
      if (gated(v)) continue;
      const name = stripColors(v.label);
      seen.set(name, (seen.get(name) ?? 0) + 1);
      out.push({ key: v.var, name, defaultState: v.defaultState === true });
    }
  }
  return out.filter((c) => seen.get(c.name) === 1);
};

export const optionControl = (page: Page, role: 'checkbox' | 'spinbutton' | 'combobox', name: string) =>
  page.getByRole(role, { name: new RegExp(`^${name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}( \\(changed\\))?$`) }).first();
