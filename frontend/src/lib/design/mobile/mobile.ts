import { MediaQuery } from 'svelte/reactivity';
import type { Sidebar } from '$lib/display/sidebar';
import type { StatLine } from '$lib/display/stat_list';
import type { StatCompareLine } from '$lib/skill_tree/stat_compare';

export type MobileTab = { label: string; href: string };

const tab = (label: string, href = `#${label.toLowerCase()}`): MobileTab => ({ label, href });

export const primaryTabs: MobileTab[] = ['Tree', 'Skills', 'Items', 'Calcs', 'Config'].map((l) => tab(l));
export const moreTabs: MobileTab[] = [tab('Notes'), tab('Party'), tab('Compare'), tab('Import/Export', '#import'), tab('Timeless')];

export type Headline = { section: string; line: StatLine; delta?: StatCompareLine };

const dps = ['Full DPS', 'Combined DPS', 'Total DPS inc. Ignite', 'Total DPS', 'Hit DPS', 'Average Damage'];
const defaults = ['Total Life', 'Total Mana', 'Energy Shield'];

// `key` is the section part of the Sidebar's pin id, '' for the player's own section
const lines = (sb: Sidebar) => sb.sections.flatMap((s) => s.groups.flat().map((line) => ({ section: s.header ?? 'Player', key: s.header ?? '', line })));

// every pinned stat of the Sidebar (ids `${header}|${label}|${n}`), or Life, Mana, ES and Full DPS when nothing is pinned
export const headlines = (sb: Sidebar, pinned: string[] = [], delta: StatCompareLine[] = []): Headline[] => {
  const all = lines(sb);
  const picked: { section: string; line: StatLine }[] = [];
  const add = (x?: { section: string; line: StatLine }) => {
    if (x && !picked.includes(x)) picked.push(x);
  };
  for (const id of pinned) {
    const [key, label] = id.split('|');
    add(all.find((x) => x.key === key && x.line.label === label));
  }
  if (!picked.length) {
    for (const l of defaults) add(all.find((x) => x.section === 'Player' && x.line.label === l));
    add(dps.map((l) => all.find((x) => x.line.label === l && x.line.value !== '0')).find(Boolean));
  }
  return picked.map((p) => ({ ...p, delta: delta.find((d) => d.label === p.line.label) }));
};

// matches --breakpoint-phone; for the few layouts that need a different DOM, not just different CSS
export const PHONE_QUERY = '(width < 40rem)';
export const phoneMedia = (): { readonly current: boolean } =>
  typeof window !== 'undefined' && typeof window.matchMedia === 'function' ? new MediaQuery(PHONE_QUERY) : { current: false };

export const BOARD_PANES = [
  { id: 'weapons', label: 'Weapons', slots: ['Weapon 1', 'Weapon 2'] },
  { id: 'armour', label: 'Armour', slots: ['Helmet', 'Body Armour', 'Gloves', 'Boots', 'Belt'] },
  { id: 'jewellery', label: 'Jewellery', slots: ['Amulet', 'Ring 1', 'Ring 2', 'Flask 1', 'Flask 2', 'Flask 3', 'Flask 4', 'Flask 5'] },
  { id: 'jewels', label: 'Jewels', slots: [] as string[] }
] as const;
