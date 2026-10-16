import type { ImpactBlock, TipProps } from './items';

export type EnchantSource = { name: string; label: string; lines: string[] };
export type EnchantChoices = { haveSkills: boolean; skills: { skill: string; sources: EnchantSource[] }[]; used: string[] };
export type AnointNotable = { id: number; name: string; stats: string[]; reminder: string[]; recipe: string[]; unsupported?: boolean[] };
export type SortStat = { stat: string; label: string };
export type AnointComparison = { note?: string; blocks: ImpactBlock[] };

export type CorruptMod = { id: string; label: string; lines: string[]; group: string; modTags: string[] };
export type CorruptChoices = {
  unique: boolean;
  implicitsCannotBeChanged: boolean;
  /* ItemsTab.lua:3129-3137 */
  maxImplicits: number;
  /* ItemsTab.lua:3138-3155 */
  glimpse: CorruptMod[];
  corrupted: CorruptMod[];
  scourgeUpside: CorruptMod[];
  scourgeDownside: CorruptMod[];
  rolls: { index: number; line: string; value: number }[];
};

export type ImplicitChoices = {
  sources: { id: string; label: string }[];
  groups: Record<string, { label: string; mods: { id: string; label: string; lines: string[]; modTags: string[] }[] }[]>;
};

export type CustomModChoices = {
  sources: { id: string; label: string }[];
  mods: Record<string, { id: string; label: string; lines: string[]; modTags: string[] }[]>;
};

export const plainText = (s: string) => s.replace(/\^(x[0-9A-Fa-f]{6}|[0-9])/g, '');

export type AffixSeries = { label: string; tiers: number; haveRange: boolean; tooltip: string[]; mid: string[] };
export type AffixControls = {
  sorting: boolean;
  controls: { type: string; series: AffixSeries[]; sel: number; sliderShown: boolean; sliderVal: number }[];
  base?: string;
  bases?: string[];
};

const RANGE = /\(-?[\d.]+--?[\d.]+\)/g;
const NUMBER = /-?\d+\.?\d*/g;
export const statTemplate = (line: string) => plainText(line).replace('[Retained] ', '').replace(RANGE, '#').replace(NUMBER, '#');

/* Item.lua:2085-2093 */
export const affixLines = (affixes: AffixControls, lines: string[], explicit: number[]) => {
  const stats = affixes.controls.map((c) => {
    const want = c.sel > 0 ? c.series[c.sel].label.split('/').map(statTemplate) : [];
    return explicit.filter((i) => want.includes(statTemplate(lines[i])));
  });
  const seen = new Set<number>();
  const shared = new Set<number>();
  const slots = stats.map((list) =>
    list.filter((i) => {
      if (seen.has(i)) shared.add(i);
      else seen.add(i);
      return !shared.has(i);
    })
  );
  return { stats, slots, shared, rest: explicit.filter((i) => !seen.has(i)) };
};

export type CrucibleChoices = { nodes: { id: string; label: string; lines: string[] }[][]; selected: string[] };

export type CraftApi = {
  enchantChoices: () => Promise<EnchantChoices | undefined>;
  notables: () => Promise<AnointNotable[]>;
  sortStats: () => Promise<SortStat[]>;
  preview: (edit: { kind: string; index: number; text: string }) => Promise<TipProps | undefined>;
  enchantSort: (slot: number, lines: string[], stat: string) => Promise<number[]>;
  anointPowers: (slot: number, ids: number[], stat: string) => Promise<number[]>;
  anointCompare: (slot: number, id: number) => Promise<AnointComparison | undefined>;
  corruptChoices?: () => Promise<CorruptChoices | undefined>;
  rangeLine?: (index: number, value: number) => Promise<string>;
  corruptText?: (source: string, ids: string[], downside: string[], rolls?: { index: number; value: number }[]) => Promise<string | undefined>;
  corruptSort?: (source: string, modType: string, ids: string[], stat: string) => Promise<number[]>;
  /* ItemsTab.lua:2503-2525 */
  modCompare?: (lines: string[], replaceImplicits?: boolean) => Promise<ImpactBlock[]>;
  implicitChoices?: () => Promise<ImplicitChoices | undefined>;
  implicitText?: (source: string, modId: string, custom: string) => Promise<string | undefined>;
  implicitSort?: (source: string, ids: string[], stat: string) => Promise<number[]>;
  customModChoices?: () => Promise<CustomModChoices | undefined>;
  customModText?: (source: string, modId: string, custom: string) => Promise<string | undefined>;
  customModSort?: (source: string, ids: string[], stat: string) => Promise<number[]>;
  affixEdit?: (slot: number, series: number, slider: number) => Promise<string | undefined>;
  affixRollTooltip?: (slot: number, slider: number) => Promise<string[]>;
  affixPowers?: (slot: number, stat: string) => Promise<number[]>;
  rebase?: (base: string) => Promise<string[] | undefined>;
  rebaseTip?: (base: string) => Promise<TipProps | undefined>;
  crucibleChoices?: () => Promise<CrucibleChoices | undefined>;
  crucibleText?: (ids: string[]) => Promise<string | undefined>;
  tipOf?: (text: string) => Promise<TipProps | undefined>;
  settext?: (text: string) => void;
};

/* ItemsTab.lua:3177-3218 */
export function corruptOptions(list: CorruptMod[], others: (string | undefined)[]): CorruptMod[] {
  const groups = new Set(others.flatMap((o) => (o ? (list.find((m) => m.id === o)?.group ?? []) : [])));
  return list.filter((m) => !groups.has(m.group));
}

/* ItemsTab.lua:2688-2696 */
export function enchantSkills(c: EnchantChoices, allSkills: boolean): string[] {
  const all = c.skills.map((s) => s.skill);
  return allSkills || !c.used.length ? all : all.filter((s) => c.used.includes(s));
}

/* ItemsTab.lua:2780-2814 */
export function sortByValue(values: number[] | undefined, count: number): number[] {
  const order = Array.from({ length: count }, (_, i) => i);
  if (!values) return order;
  return order.sort((a, b) => (values[a] !== values[b] ? values[b] - values[a] : a - b));
}

/* NotableDBControl.lua:50-79 */
export function notableMatches(n: AnointNotable, search: string, mode: 'Anywhere' | 'Names' | 'Modifiers'): boolean {
  const s = search.toLowerCase();
  if (!/\S/.test(s)) return true;
  const alternatives = s.split('|').filter((a) => a !== '');
  const match = (text: string) => alternatives.some((a) => text.toLowerCase().includes(a));
  if (mode !== 'Modifiers' && match(n.name)) return true;
  if (mode !== 'Names' && n.stats.some(match)) return true;
  return false;
}

/* NotableDBControl.lua:162-182 */
export function sortNotables(list: AnointNotable[], power?: Map<number, number>): AnointNotable[] {
  return [...list].sort((a, b) => {
    if (power) {
      const pa = power.get(a.id) ?? 0;
      const pb = power.get(b.id) ?? 0;
      if (pa !== pb) return pb - pa;
    }
    return a.name < b.name ? -1 : a.name > b.name ? 1 : 0;
  });
}

/* NotableDBControl.lua:140-159 */
export function finitePowers(power: Map<number, number>): Map<number, number> {
  let max = 0;
  let infinite = false;
  for (const v of power.values()) {
    if (v === Infinity) infinite = true;
    else max = Math.max(max, v);
  }
  if (!infinite) return power;
  return new Map([...power].map(([id, v]) => [id, v === Infinity ? max * 2 : v]));
}

export function maxOf(power: Map<number, number>): number {
  return Math.max(0, ...power.values());
}

/* NotableDBControl.lua:215-230 */
export function notableColour(power: number | undefined, maxPower: number | undefined, byStat: boolean): string {
  if (!byStat) return '^xB8DAF1';
  if (maxPower && maxPower > 0 && power && power > 0) {
    const red = Math.floor((power / maxPower) * (0xff - 0x80) + 0x80);
    return `^x${red.toString(16).toUpperCase()}8080`;
  }
  return '^x808080';
}
