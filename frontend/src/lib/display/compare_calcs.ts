// CompareTab.lua:4655-4902
import type { Outputs } from '$lib/custom_types';
import type { CalcDataCol, CalcDataColProp, CalcSection } from '$lib/calcs/calc_sections';
import { checkFlag, rowsOf } from '$lib/calcs/check_flag';
import { FormatStr } from '$lib/calcs/formatting';
import { cellId, elementsOf } from './calcs';
import { colorCodesX } from './colors';
import type { calculator } from '$lib/types';

/* CompareCalcsHelpers.lua:185-210 */
export type CompareCalcMods = { label: string; sd: calculator.ModSectionData };
export type CompareCalcRow = { label: string; a: string; b: string; tip?: string[]; mods?: CompareCalcMods[] };
export type CompareCalcBlock = { label: string; extra?: { a: string; b: string }; rows: CompareCalcRow[] };
export type CompareCalcSection = { id: string; color: string; blocks: CompareCalcBlock[] };

const sameLines = (x: string[] | undefined, y: string[] | undefined) => (x?.length ?? 0) === (y?.length ?? 0) && (x ?? []).every((l, i) => l === y?.[i]);

// src/Classes/CompareCalcsHelpers.lua:141-163
const linesOf = (out: Outputs, bd: string) => {
  const lines = out.Breakdown?.[bd]?.Lines ?? [];
  return lines.length ? [...lines] : undefined;
};

export const modKey = (sd: calculator.ModSectionData) => JSON.stringify(sd);

const sdOf = (e: CalcDataColProp): calculator.ModSectionData => ({
  ModName: typeof e.modName === 'string' ? [e.modName] : [...e.modName!],
  Multi: typeof e.modName !== 'string',
  ModType: e.modType ?? '',
  Cfg: e.cfg ?? '',
  ModSource: e.modSource ?? '',
  Enemy: !!e.enemy,
  Actor: e.actor ?? ''
});

const modsOf = (col: CalcDataCol): CompareCalcMods[] =>
  elementsOf<CalcDataColProp>(col)
    .filter((e) => e.modName)
    .map((e) => ({ label: e.label ?? '', sd: sdOf(e) }));

export const modEntries = (sections: CalcSection[]) => {
  const out = new Map<string, calculator.ModSectionData>();
  for (const section of sections)
    for (const block of section[4]) for (const row of rowsOf(block.data)) if (row[0]) for (const m of modsOf(row[0])) out.set(modKey(m.sd), m.sd);
  return [...out.values()];
};

// src/Classes/CompareTab.lua:4610-4646
const rowMatches = (col: CalcDataCol, a: Outputs, b: Outputs, va: string, vb: string, modsEqual?: Map<string, boolean>) => {
  if (va !== vb) return false;
  for (const prop of elementsOf<CalcDataColProp>(col)) {
    if (prop.modName && !modsEqual?.get(modKey(sdOf(prop)))) return false;
    if (prop.breakdown && !sameLines(linesOf(a, prop.breakdown), linesOf(b, prop.breakdown))) return false;
  }
  return true;
};

// src/Classes/CompareCalcsHelpers.lua:185-210
const tipOf = (label: string, col: CalcDataCol, a: Outputs, b: Outputs, nameA: string, nameB: string) => {
  const tip = [label];
  for (const prop of elementsOf<CalcDataColProp>(col)) {
    if (!prop.breakdown) continue;
    const la = linesOf(a, prop.breakdown);
    const lb = linesOf(b, prop.breakdown);
    if (la) tip.push(`${nameA}:`, ...la.map((l) => `  ${l}`));
    if (lb) tip.push(`${nameB}:`, ...lb.map((l) => `  ${l}`));
  }
  return tip.length > 1 ? tip : undefined;
};

// src/Classes/CompareCalcsHelpers.lua:212-310
export const modTipLines = (label: string, pRows: calculator.CompareModRow[], cRows: calculator.CompareModRow[], nameA: string, nameB: string) => {
  if (!pRows.length && !cRows.length) return [];
  const cByKey = new Map(cRows.map((r) => [r.Key, r]));
  const common: [calculator.CompareModRow, calculator.CompareModRow][] = [];
  const pOnly: calculator.CompareModRow[] = [];
  const cMatched = new Set<string>();
  for (const p of pRows) {
    const c = cByKey.get(p.Key);
    if (c) {
      common.push([p, c]);
      cMatched.add(p.Key);
    } else pOnly.push(p);
  }
  const cOnly = cRows.filter((r) => !cMatched.has(r.Key));
  const pad = (v: string, n: number) => v.padEnd(n);
  const tail = (r: calculator.CompareModRow) => `^7${pad(r.SourceType, 6)} ^7${r.SourceName}${r.ModName}`;
  const only = (r: calculator.CompareModRow) => `    ^7${pad(r.Display, 10)} ${tail(r)}`;
  const lines = [`^7${label || 'Player modifiers'}:`];
  if (common.length) {
    lines.push('^x808080  Common:');
    for (const [p, c] of common.sort((x, y) => y[0].Value - x[0].Value)) {
      const val =
        p.Display === c.Display ? `^7${pad(p.Display, 10)}` : `${colorCodesX.POSITIVE}${pad(p.Display, 5)}^7/${colorCodesX.WARNING}${pad(c.Display, 5)}`;
      lines.push(`    ${val} ${tail(p)}`);
    }
  }
  if (pOnly.length) lines.push(`${colorCodesX.POSITIVE}  ${nameA} only:`, ...pOnly.sort((x, y) => y.Value - x.Value).map(only));
  if (cOnly.length) lines.push(`${colorCodesX.WARNING}  ${nameB} only:`, ...cOnly.sort((x, y) => y.Value - x.Value).map(only));
  return lines;
};

// src/Classes/CompareTab.lua:4692-4740
export function compareCalcsView(
  sections: CalcSection[],
  a: Outputs,
  b: Outputs,
  onlyDifferences: boolean,
  nameA = 'A',
  nameB = 'B',
  modsEqual?: Map<string, boolean>
): CompareCalcSection[] {
  const view: CompareCalcSection[] = [];
  for (const section of sections) {
    const [id, color, blocks] = [section[1], section[3], section[4]];
    if (!blocks[0] || !checkFlag(blocks[0].data, a)) continue;
    const kept: CompareCalcBlock[] = [];
    blocks.forEach((block, j) => {
      const rows: CompareCalcRow[] = [];
      rowsOf(block.data).forEach((row, k) => {
        const col = row[0];
        if (!row.label || !col?.format) return;
        const shownA = checkFlag(row, a);
        const shownB = checkFlag(row, b);
        if (!shownA && !shownB) return;
        const cid = cellId(id, j, k, 0);
        const va = FormatStr(col.format, a, col, cid);
        const vb = FormatStr(col.format, b, col, cid);
        if (onlyDifferences && shownA && shownB && rowMatches(col, a, b, va, vb, modsEqual)) return;
        const tip = tipOf(row.label, col, a, b, nameA, nameB);
        const mods = modsOf(col);
        rows.push({ label: row.label, a: va, b: vb, ...(tip && { tip }), ...(mods.length && { mods }) });
      });
      const extra = block.data.extra ? { a: FormatStr(block.data.extra, a), b: FormatStr(block.data.extra, b) } : undefined;
      if (rows.length || (onlyDifferences && extra && extra.a !== extra.b))
        kept.push(extra ? { label: block.label, extra, rows } : { label: block.label, rows });
    });
    if (kept.length) view.push({ id, color, blocks: kept });
  }
  return view;
}
