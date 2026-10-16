import { writable } from 'svelte/store';
import type { Tone } from './tone';
import type { ItemSummary, ItemTip, StatDiff } from './fixtures';
import type { StatCompareSection } from '$lib/skill_tree/stat_compare';

export type Impact = { label: string; text: string; dir: 'up' | 'down' };

const MOD_TONES: Record<string, Tone> = { crafted: 'crafted', fractured: 'fractured', custom: 'crafted', disabled: 'muted', unsupported: 'muted' };
export const modTone = (kind: string): Tone => MOD_TONES[kind] ?? 'magic';

export const impactText = (l: { value: string; percent?: string }) => `${l.value.startsWith('-') ? '▼' : '▲'} ${l.value}${l.percent ? ` ${l.percent}` : ''}`;

export function impactRows(diff: StatDiff | undefined, max = 4): Impact[] {
  const rows = (diff?.sections ?? []).flatMap((s) =>
    s.lines.map((l) => ({
      label: s.actor === 'Minion' ? `Minion ${l.label}` : l.label,
      text: impactText(l),
      dir: l.better ? ('up' as const) : ('down' as const)
    }))
  );
  return rows.slice(0, max);
}

export const replacing = (diff: StatDiff | undefined) => diff?.header.split('\n')[1];

export type SocketColour = 'R' | 'G' | 'B' | 'W';
export function socketLine(line: string) {
  const colours: SocketColour[] = [];
  const links: boolean[] = [];
  for (const ch of line) {
    if (ch === '-') links[colours.length - 1] = true;
    else if (ch === ' ') links[colours.length - 1] = false;
    else colours.push(ch === 'R' || ch === 'G' || ch === 'B' ? ch : 'W');
  }
  return { colours, links: colours.slice(1).map((_, i) => links[i] ?? false) };
}

export function setLink(line: string, i: number, linked: boolean) {
  let n = -1;
  return [...line].map((ch) => (ch !== '-' && ch !== ' ' ? (n++, ch) : n === i ? (linked ? '-' : ' ') : ch)).join('');
}

export const itemLabel = (it: ItemSummary) => (it.name === it.base ? it.name : `${it.name}, ${it.base}`);

export const baseSlot = (slot: string) => slot.replace(/ Swap$/, '');

export const BOARD = { w: 868, h: 742 };
export const BOARD_SLOTS: { slot: string; x: number; y: number; w: number; h: number }[] = [
  { slot: 'Weapon 1', x: 42, y: 28, w: 168, h: 322 },
  { slot: 'Helmet', x: 350, y: 28, w: 168, h: 168 },
  { slot: 'Weapon 2', x: 658, y: 28, w: 168, h: 322 },
  { slot: 'Body Armour', x: 350, y: 210, w: 168, h: 252 },
  { slot: 'Ring 1', x: 252, y: 266, w: 84, h: 84 },
  { slot: 'Amulet', x: 532, y: 210, w: 84, h: 84 },
  { slot: 'Ring 2', x: 532, y: 308, w: 84, h: 84 },
  { slot: 'Gloves', x: 168, y: 406, w: 168, h: 168 },
  { slot: 'Belt', x: 350, y: 490, w: 168, h: 84 },
  { slot: 'Boots', x: 532, y: 406, w: 168, h: 168 },
  ...[210, 302, 395, 487, 580].map((x, i) => ({ slot: `Flask ${i + 1}`, x, y: 588, w: 78, h: 120 }))
];

export type ImpactBlock = { header: string; slot?: string; lines?: string[]; sections: StatCompareSection[]; readyAt?: number; calcMs?: number };
export type WhatIf = { key: string; pending: boolean; blocks: ImpactBlock[]; tip?: string };

export type ItemSetOp =
  | { op: 'new'; title: string }
  | { op: 'copy' | 'rename'; id: string; title: string }
  | { op: 'delete'; id: string }
  | { op: 'move'; from: number; to: number };

export type TipProps = {
  item: string;
  base: string;
  rarity: Tone;
  mods: string[];
  flask?: { on: boolean; state: string };
  impactTitle: string;
  impactSub?: string;
  impact: Impact[];
  whatIf?: WhatIf;
  tooltip?: ItemTip;
};

export function itemTip(it: ItemSummary, diff: StatDiff | undefined, kind: 'equipped' | 'candidate', flaskOn?: boolean, calcOn?: boolean): TipProps {
  const base = { item: it.name, base: it.base, rarity: it.rarity, mods: [...it.implicits, ...it.explicits].map((m) => m.text), tooltip: it.tooltip };
  if (it.flask && flaskOn !== undefined) {
    const stale = flaskOn !== calcOn;
    return {
      ...base,
      flask: { on: flaskOn, state: flaskOn ? 'Active in calculations' : 'Inactive (not counted)' },
      impactTitle: (diff?.header ?? 'Toggling this flask').replace(/:$/, '').toUpperCase(),
      impactSub: stale ? 'Changed here; the what-if runs on the next calculation' : undefined,
      impact: stale ? [] : impactRows(diff)
    };
  }
  if (kind === 'equipped') return { ...base, impactTitle: 'UNEQUIP IMPACT', impact: impactRows(diff) };
  return {
    ...base,
    impactTitle: diff?.slot ? `EQUIP IMPACT (IN ${diff.slot.toUpperCase()})` : 'EQUIP IMPACT',
    impactSub: replacing(diff) ?? (diff ? undefined : 'Fits no equipment slot'),
    impact: impactRows(diff)
  };
}

export function besideAnchor(tip: HTMLElement, anchor: HTMLElement | undefined) {
  const place = (to: HTMLElement | undefined) => {
    if (!to) return;
    const a = to.getBoundingClientRect();
    const t = tip.getBoundingClientRect();
    const gap = 10;
    const pad = 8;
    let x = a.right + gap;
    if (x + t.width > innerWidth - pad) x = a.left - gap - t.width;
    const y = Math.min(Math.max(pad, a.top), innerHeight - pad - t.height);
    tip.style.setProperty('--x', `${Math.max(pad, x)}px`);
    tip.style.setProperty('--y', `${Math.max(pad, y)}px`);
  };
  place(anchor);
  let current = anchor;
  const ro = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(() => place(current));
  ro?.observe(tip);
  return {
    update: (to: HTMLElement | undefined) => place((current = to)),
    destroy: () => ro?.disconnect()
  };
}

/* ItemsTab.lua:378-715 */
export type DisplayMod = {
  line: string;
  kind: string;
  section?: string;
  hasRange: boolean;
  range: number;
  disabled: boolean;
  removable: boolean;
  removal: string;
};
export type DisplayInfo = {
  quality: number;
  hasQuality: boolean;
  variants: string[];
  variant: number;
  altVariants: number[];
  sockets: string;
  socketLimit: number;
  defaultColor: string;
  canInfluence: boolean;
  influence: string[];
  showsCatalyst: boolean;
  catalyst: number;
  catalystQuality: number;
  mods: DisplayMod[];
  /* ItemsTab.lua:509-553 */
  enchants?: number;
  anoints?: number;
  anointed?: string[];
  /* Item.lua:1026, ItemsTab.lua:571-576 */
  corruptible?: boolean;
  addImplicit?: boolean;
  addCustom?: boolean;
  /* ItemsTab.lua:1002-1004 */
  addCrucible?: boolean;
  /* ItemsTab.lua:717-981 */
  affixes?: import('./crafting').AffixControls;
  /* ItemsTab.lua:1034-1070 */
  foulborn?: { mutable: boolean; mutated: boolean }[];
  /* ItemsTab.lua:378-452 */
  variantGroups?: {
    versions: string[];
    version: number;
    groups: { group: number; options: { variant: number; label: string }[]; selected: number; enabled: boolean }[];
  };
  /* ItemsTab.lua:696-715 */
  cluster?: { skills: { id: string; label: string }[]; skill: string; nodeCount: number; min: number; max: number };
};
/* Main.lua:1032-1035, ItemsTab.lua:1025-1089 */
export const allAffixSliders = writable(true);
export type DisplayEdit = { kind: string; index?: number; value?: number; second?: number; float?: number; text?: string };

/* ItemsTab.lua:2566-2581 */
export type CraftChoice = { rarity: number; title: string; base: string };

const SOCKET_CODE: Record<string, string> = { R: '^xE05030', G: '^x70FF70', B: '^x7070FF', W: '^xFFF0F0', A: '^xB0B0B0' };
export function socketDescription(line: string) {
  let out = '';
  let linked = false;
  let first = true;
  for (const ch of line) {
    if (ch === '-' || ch === ' ') {
      linked = ch === '-';
      continue;
    }
    if (!first) out += linked ? '^7=' : '  ';
    first = false;
    out += (SOCKET_CODE[ch] ?? '') + ch;
  }
  return out;
}
