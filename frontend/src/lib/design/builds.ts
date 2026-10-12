import type { SavedBuild } from './fixtures/app';
import type { Tone } from './tone';

export const under = (path: string, folder: string) => folder === '' || path === folder || path.startsWith(folder + '/');
export const childrenOf = (folders: string[], f: string) =>
  folders.filter((x) => x.split('/').length === (f === '' ? 1 : f.split('/').length + 1) && (f === '' || x.startsWith(f + '/')));
export const countIn = (builds: SavedBuild[], f: string) => builds.filter((b) => under(b.folder, f)).length;
export const leaf = (f: string) => f.split('/').pop() ?? f;

export const fmtShort = (v: number) => (v >= 1e6 ? (v / 1e6).toFixed(2) + 'M' : v >= 1e3 ? (v / 1e3).toFixed(1) + 'K' : String(v));
export const fmtInt = (v: number) => Math.round(v).toLocaleString('en-US');
export const fmtAgo = (m: number) =>
  m < 60
    ? `${m} min ago`
    : m < 1440
      ? `${Math.round(m / 60)} h ago`
      : m < 2880
        ? 'yesterday'
        : m < 10080
          ? `${Math.round(m / 1440)} days ago`
          : m < 43200
            ? `${Math.round(m / 10080)} weeks ago`
            : 'last month';

export type ColKey = 'name' | 'cls' | 'asc' | 'level' | 'skill' | 'dps' | 'ehp' | 'life' | 'es' | 'tree' | 'edited';
export type Col = {
  key: ColKey;
  label: string;
  fr: number;
  right: boolean;
  default: boolean;
  stat?: boolean;
  sort: (b: SavedBuild) => string | number;
  text: (b: SavedBuild) => string;
  tone: (b: SavedBuild) => Tone;
};

const clsTone = (b: SavedBuild) => b.className.toLowerCase() as Tone;

export const COLS: Col[] = [
  { key: 'name', label: 'Name', fr: 7, right: false, default: true, sort: (b) => b.name.toLowerCase(), text: (b) => b.name, tone: () => 'text' },
  { key: 'cls', label: 'Class', fr: 3, right: false, default: false, sort: (b) => b.className, text: (b) => b.className, tone: clsTone },
  { key: 'asc', label: 'Ascendancy', fr: 4, right: false, default: true, sort: (b) => b.ascendancy, text: (b) => b.ascendancy, tone: clsTone },
  { key: 'level', label: 'Level', fr: 2, right: true, default: true, sort: (b) => b.level, text: (b) => String(b.level), tone: () => 'text' },
  { key: 'skill', label: 'Main skill', fr: 4, right: false, default: true, sort: (b) => b.skill, text: (b) => b.skill, tone: () => 'gem' },
  { key: 'dps', label: 'Full DPS', fr: 3, right: true, default: true, stat: true, sort: (b) => b.dps, text: (b) => fmtShort(b.dps), tone: () => 'text' },
  { key: 'ehp', label: 'EHP', fr: 3, right: true, default: true, stat: true, sort: (b) => b.ehp, text: (b) => fmtShort(b.ehp), tone: () => 'text' },
  { key: 'life', label: 'Life', fr: 2, right: true, default: true, stat: true, sort: (b) => b.life, text: (b) => fmtInt(b.life), tone: () => 'life' },
  { key: 'es', label: 'ES', fr: 2, right: true, default: true, stat: true, sort: (b) => b.es, text: (b) => fmtInt(b.es), tone: () => 'es' },
  { key: 'tree', label: 'Tree version', fr: 2, right: false, default: false, sort: (b) => b.version, text: (b) => b.version, tone: () => 'dim' },
  { key: 'edited', label: 'Modified', fr: 3, right: false, default: true, sort: (b) => -b.edited, text: (b) => fmtAgo(b.edited), tone: () => 'muted' }
];
export const DEFAULT_COLS = COLS.filter((c) => c.default).map((c) => c.key);

export const subpath = (b: SavedBuild, folder: string) => {
  const r = folder === '' ? b.folder : b.folder.slice(folder.length + 1);
  return r ? r + '/ ' : '';
};

// BuildListHelpers.lua:37-58
export const matches = (b: SavedBuild, q: string, folder: string) =>
  q
    .trim()
    .toLowerCase()
    .split(/\s+/)
    .every((t) => {
      const m = /^class:(.*)$/.exec(t);
      if (m) return !m[1] || [b.className, b.ascendancy, b.folder].some((x) => x.toLowerCase().includes(m[1]));
      return [b.name, b.folder, b.className, b.ascendancy, b.skill, subpath(b, folder)].some((x) => x.toLowerCase().includes(t));
    });

export const buildId = (b: SavedBuild) => (b.folder ? `${b.folder}/${b.name}` : b.name);

export type Drag = { id: string; name: string; over: string | null; where: 'tree' | 'list' | null };

export type Sort = { key: ColKey; dir: 'asc' | 'desc' };

export const visibleRows = (builds: SavedBuild[], folder: string, q: string, cls: string, sort: Sort) => {
  const col = COLS.find((c) => c.key === sort.key) ?? COLS[COLS.length - 1];
  const ql = q.trim();
  return builds
    .filter((b) => (ql ? under(b.folder, folder) && matches(b, ql, folder) : folder === '' || b.folder === folder) && (cls === 'All' || b.className === cls))
    .sort((a, b) => {
      const x = col.sort(a);
      const y = col.sort(b);
      const r = x < y ? -1 : x > y ? 1 : 0;
      return sort.dir === 'asc' ? r : -r;
    });
};

export const nextSort = (sort: Sort, col: Col): Sort =>
  sort.key === col.key ? { key: col.key, dir: sort.dir === 'asc' ? 'desc' : 'asc' } : { key: col.key, dir: col.right || col.key === 'edited' ? 'desc' : 'asc' };
