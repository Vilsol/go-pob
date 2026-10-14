// PassiveTreeView.lua:783-797
import type { TreeNodeFx } from './fixtures';

export const KIND_WORD: Record<TreeNodeFx['k'], string> = { s: 'normal', n: 'notable', k: 'keystone', m: 'mastery', j: 'socket', c: 'class start' };
export const KIND_NAME: Record<TreeNodeFx['k'], string> = { s: 'Passive', n: 'Notable', k: 'Keystone', m: 'Mastery', j: 'Jewel socket', c: 'Class start' };

export const searchWords = (q: string) => {
  const words: string[] = [];
  const rest = q.toLowerCase().replace(/"([^"]*)"/g, (_m, w: string) => {
    if (w) words.push(w);
    return ' ';
  });
  for (const w of rest.split(/\s+/)) if (w) words.push(w);
  return words;
};

const has = (hay: string, w: string) => {
  try {
    return new RegExp(w).test(hay);
  } catch {
    return hay.includes(w);
  }
};

const ID_WORD = /^#\d+$/;

export function nodeMatches(n: TreeNodeFx, words: string[]) {
  if (n.k === 'c' || !words.length) return false;
  if (words[0] === 'oil:' || words[0].startsWith('oil:')) {
    const oils = [words[0].slice(4), ...words.slice(1)].filter(Boolean);
    const recipe = (n.recipe ?? []).map((o) => o.replace(/Oil$/, '').toLowerCase());
    return oils.length > 0 && recipe.length > 0 && oils.every((w) => recipe.some((o) => has(o, w)));
  }
  const hays = [n.name.toLowerCase(), KIND_WORD[n.k], ...(n.stats ?? []).map((s) => s.toLowerCase())];
  return words.every((w) => (ID_WORD.test(w) ? Number(w.slice(1)) === n.id : hays.some((x) => has(x, w))));
}

export const SEARCH_HELP: [string, string][] = [
  ['life mana', 'every word must match (name, stat lines, node type)'],
  ['"minion life"', 'quotes keep a phrase together'],
  ['oil:golden silver', 'notables whose anoint recipe uses these oils'],
  ['keystone', 'a node type: notable, keystone, mastery, socket'],
  ['min.*life', 'patterns work too'],
  ['#12345', 'the node with that id']
];

export type CanvasApi = {
  hovered: number | null;
  describedBy: string;
  path: number[];
  lose: number[];
  matches: Set<number>;
  enter: (n: TreeNodeFx) => void;
  leave: (n: TreeNodeFx) => void;
  pick: (n: TreeNodeFx) => void;
  menu: (n: TreeNodeFx, at: { x: number; y: number }) => void;
};

export type SpecOp =
  | { op: 'new'; title: string }
  | { op: 'copy'; index: number; title: string }
  | { op: 'rename'; index: number; title: string }
  | { op: 'delete'; index: number }
  /* ListControl.lua:428-436 */
  | { op: 'move'; from: number; to: number };

export type PowerRow = {
  ID: number;
  Name: string;
  Type: string;
  Power: number;
  PathPower: number;
  PathDist: number;
  Allocated: boolean;
  Cluster?: boolean;
  Stats?: string[];
  /* CalcsTab.lua:541 */
  Offence?: number;
  Defence?: number;
  /* CalcsTab.lua:684 */
  Effect?: number;
  Picked?: boolean;
};
/* PowerReportListControl.lua:31 */
export type PowerFilter = 'unallocated' | 'clusters' | 'allocated';
/* PowerReportListControl.lua:16-22 */
export type PowerSort = 'type' | 'power' | 'points' | 'perPoint';
export type ShownPowerRow = PowerRow & { power: string; perPoint: string; points: string; dir: 'up' | 'down' | '' };

/* TreeTab.lua:1059-1111, PowerReportListControl.lua:58-135 */
export function showPowerRows(
  rows: PowerRow[],
  display: { fmt?: string; pc?: boolean; mod?: boolean; lowerIsBetter?: boolean } | undefined,
  filter: PowerFilter,
  masteries: boolean,
  format: (fmt: string, v: number) => string,
  sort: PowerSort = 'power'
): ShownPowerRow[] {
  const ds = display ?? { fmt: '.1f' };
  const mult = ds.pc || ds.mod ? 100 : 1;
  const dir = (v: number): ShownPowerRow['dir'] => (v === 0 ? '' : v > 0 !== !!ds.lowerIsBetter ? 'up' : 'down');
  const out = rows
    .filter((r) => (filter === 'allocated' ? r.Allocated : r.Power > 0 && (filter === 'clusters' || !r.Cluster)) && (masteries || r.Type !== 'Mastery'))
    .map((r) => ({
      ...r,
      power: format(ds.fmt ?? '.1f', r.Power * mult),
      perPoint: r.Cluster ? '--' : format(ds.fmt ?? '.1f', r.PathPower * mult),
      points: r.Cluster ? 'Cluster' : r.PathDist === 1000 ? 'Anoint' : String(r.PathDist),
      dir: dir(r.Power)
    }));
  const compare = (a: number, b: number) => (filter === 'allocated' ? a - b : b - a);
  const byPower = (a: ShownPowerRow, b: ShownPowerRow) => compare(a.Power, b.Power);
  const sorters: Record<PowerSort, (a: ShownPowerRow, b: ShownPowerRow) => number> = {
    type: (a, b) => (a.Type === b.Type ? byPower(a, b) : a.Type < b.Type ? -1 : 1),
    power: byPower,
    points: (a, b) => (!!a.Cluster !== !!b.Cluster ? (a.Cluster ? 1 : -1) : a.PathDist === b.PathDist ? byPower(a, b) : a.PathDist - b.PathDist),
    perPoint: (a, b) => (a.PathPower === b.PathPower && !a.Cluster && !b.Cluster ? a.PathDist - b.PathDist : compare(a.PathPower, b.PathPower))
  };
  return out.sort(sorters[sort]);
}
