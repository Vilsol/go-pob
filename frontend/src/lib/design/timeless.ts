// TreeTab.lua:1301-1331
import type { TreeSocketFx } from './fixtures';
import type { calculator } from '$lib/types';
import type { Snippet } from 'svelte';

const CLASS_SOCKETS: Record<number, string> = { 26725: 'Marauder', 54127: 'Duelist', 7960: 'Templar/Witch' };

/* TreeTab.lua:1295-1336 */
export type TimelessSocket = TreeSocketFx & { label?: string };

export const socketLabel = (s: TimelessSocket) => s.label ?? `${s.pts === 0 ? '# ' : ''}${CLASS_SOCKETS[s.id] ?? s.keystones[0] ?? 'Unknown'}: ${s.id}`;

export const socketCovers = (s: TreeSocketFx) =>
  `${s.notables.length} notables, ${s.keystones.length} keystone${s.keystones.length === 1 ? '' : 's'}, ${s.smalls} small in radius`;

export const socketCost = (s: TreeSocketFx) => (s.pts < 0 ? 'not reachable' : s.pts === 0 ? 'allocated' : `+${s.pts} points to reach`);

export const keystoneOf = (conqueror: string) => /\((.*)\)/.exec(conqueror)?.[1] ?? conqueror;

/* TreeTab.lua:1864-1918 */
export type TimelessStat = { id: string; label: string; descriptions: string[]; total?: boolean };

// TreeTab.lua:1404-1505, 1828-1918

const split = (line: string) => line.match(/[^,\s]+/g) ?? [];
const rowsOf = (text: string) =>
  text
    .split(/[\r\n]+/)
    .filter((l) => l !== '')
    .map(split);

const round3 = (v: number) => String(Math.floor(v * 1000 + 0.5) / 1000);

/* TreeTab.lua:1855-1863 */
export const statCount = (m: calculator.TimelessMod) => (m.TotalMod || !m.ID ? 0 : (m.Descriptions ?? []).length);

/* TreeTab.lua:1404-1416 */
export const nodeWeightLabels = (w1: number, w2: number, w3: number): [string, string, string] => [
  round3(Number(w1.toFixed(3))),
  round3(Number(w2.toFixed(3))),
  w3 >= 500 ? 'required' : round3(Number(w3.toFixed(0)))
];

/* TreeTab.lua:1893-1914 */
export const insertSearchNode = (text: string, id: string, w: [string, string, string]) =>
  text + (text.length > 0 ? '\n' : '') + `${id}, ${w[0]}, ${w[1]}, ${w[2]}`;

const slidersOf = (row: string[]) => {
  const num = (s: string | undefined, max: number) => Math.min(Math.max(Number.isFinite(Number(s)) && s !== undefined ? Number(s) : 0, 0), max);
  return {
    w1: row[1] === 'required' ? 10 : num(row[1], 10),
    w2: row[2] === 'required' ? 10 : num(row[2], 10),
    w3: row[3] === 'required' ? 500 : num(row[3], 500)
  };
};

/* TreeTab.lua:1828-1852 */
export const slidersFromRow = (text: string, id: string): { w1: number; w2: number; w3: number } | undefined => {
  const row = rowsOf(text).find((r) => r[0] === id);
  return row && slidersOf(row);
};

/* TreeTab.lua:1445-1491 */
export const rewriteSearchRow = (text: string, id: string, w: [string, string, string]) =>
  rowsOf(text)
    .map((r) => (r[0] === id ? [r[0], ...w, ...r.slice(4)] : r).join(', '))
    .join('\n');

export const searchRows = (text: string) =>
  rowsOf(text)
    .filter((r) => r.length)
    .map((r) => ({ id: r[0] as string, ...slidersOf(r) }));

export const removeSearchRow = (text: string, id: string) =>
  rowsOf(text)
    .filter((r) => r[0] !== id)
    .map((r) => r.join(', '))
    .join('\n');

export const statMatches = (stats: TimelessStat[], query: string) => {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (!words.length) return stats;
  return stats.filter((s) => {
    const text = [s.label, ...s.descriptions].join('\n').toLowerCase();
    return words.every((w) => text.includes(w));
  });
};

/* TreeTab.lua:1161-2400 */
export type TimelessForm = {
  jewelType: number;
  conqueror: number;
  devotion1: number;
  devotion2: number;
  socket: number;
  filter: boolean;
  distance: number;
  protect: string[];
  searchList: string;
  searchListFallback: string;
  fallbackMode: number;
  totalMinimum: string;
  realm: string;
  league: string;
  tradeType: number;
  searchMore: boolean;
};

export type TimelessRow = { label: string; seed: number; total: number; socket: number; nodes: { name: string; targets: string[] }[] };

export type TimelessTreeArgs = {
  rings: { id: number; hot?: boolean }[];
  picked?: Set<number>;
  centre: number | null;
  hovered: number | null;
  sockets: Set<number>;
  enter: (id: number) => void;
  leave: () => void;
  pick: (id: number) => void;
  tip: Snippet<[number]>;
  legend: Snippet;
};
