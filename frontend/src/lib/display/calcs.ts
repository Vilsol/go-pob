import type { Outputs } from '$lib/custom_types';
import type { CalcDataCol, CalcDataColProp, CalcSection } from '$lib/calcs/calc_sections';
import { checkFlag, rowsOf, sectionEnabled } from '$lib/calcs/check_flag';
import { FormatStr } from '$lib/calcs/formatting';
import { type Gauge, statGauge } from './stat_gauge';
import type { calculator } from '$lib/types';

export type CalcModSection = {
  label?: string;
  sd: { ModName: string[]; Multi: boolean; ModType: string; Cfg: string; ModSource: string; Enemy: boolean; Actor: string };
};

export type CalcCell = { text: string; bd?: string; mods?: string[]; sections?: CalcModSection[] };

export type CalcRow = { label: string; cells: CalcCell[]; gauge?: Gauge };
export type CalcBlock = { label: string; extra?: string; rows: CalcRow[] };
export type CalcSectionView = { id: string; title: string; wide: boolean; col: number; color: string; applicable: boolean; blocks?: CalcBlock[] };
/* CalcBreakdownControl.lua:174-209 */
export type CalcTable = { label: string; columns: { label: string; key: string }[]; rows: Record<string, string>[] };
export type CalcBreakdown = {
  label?: string;
  lines: string[];
  columns?: { label: string; key: string }[];
  rows?: Record<string, string>[];
  more?: number;
  tables?: CalcTable[];
  /* CalcBreakdown.lua:107 */
  radius?: number;
};

// src/Classes/CalcBreakdownControl.lua:179-188
const RESERVATION_COLUMNS = [
  { label: 'Skill', key: 'skillName' },
  { label: 'Base', key: 'base' },
  { label: 'MCM', key: 'mult' },
  { label: 'More/less', key: 'more' },
  { label: 'Inc/red', key: 'inc' },
  { label: 'Efficiency', key: 'efficiency' },
  { label: 'Efficiency More/less', key: 'efficiencyMore' },
  { label: 'Reservation', key: 'total' }
];
// src/Classes/CalcBreakdownControl.lua:197-206
const DAMAGE_TYPE_COLUMNS = [
  { label: 'From', key: 'source' },
  { label: 'Base', key: 'base' },
  { label: 'Inc/red', key: 'inc' },
  { label: 'More/less', key: 'more' },
  { label: 'Converted Damage', key: 'convSrc' },
  { label: 'Total', key: 'total' },
  { label: 'Conversion', key: 'convDst' },
  { label: 'Gain', key: 'gainDst' }
];
export type CalcsView = { sections: CalcSectionView[]; breakdowns: Record<string, CalcBreakdown> };

export const elementsOf = <M>(obj: { [key: number]: M } | undefined): M[] =>
  Object.entries(obj ?? {})
    .filter(([k]) => !isNaN(parseInt(k)))
    .map(([, v]) => v);

/* CalcSectionControl.lua:101 */
export const cellId = (section: string, block: number, row: number, col: number) => `${section}:${block}:${row}:${col}`;

const formatOf = (col: CalcDataCol | undefined): string => col?.format ?? elementsOf<CalcDataColProp>(col).find((e) => e.format)?.format ?? '';

export function calcElements(sections: CalcSection[]): Record<string, CalcDataColProp> {
  const out: Record<string, CalcDataColProp> = {};
  for (const section of sections) {
    const id = section[1];
    for (const [j, block] of section[4].entries()) {
      for (const [k, row] of rowsOf(block.data).entries()) {
        for (const [l, col] of elementsOf<CalcDataCol>(row).entries()) {
          for (const [n, prop] of elementsOf<CalcDataColProp>(col).entries()) {
            // CalcFormat.lua:39-60
            if (prop.modName && prop.modType) out[`${cellId(id, j, k, l)}:${n}`] = prop;
          }
        }
      }
    }
  }
  return out;
}

export const colPropsOf = (elements: Record<string, CalcDataColProp>): Record<string, calculator.ColProps> =>
  Object.fromEntries(
    Object.entries(elements).map(([k, v]) => [
      k,
      {
        Cfg: v.cfg ?? '',
        ModSource: v.modSource,
        Enemy: v.enemy,
        ModName: typeof v.modName === 'string' ? [v.modName] : v.modName!,
        ModType: v.modType!,
        Actor: v.actor ?? ''
      } as calculator.ColProps
    ])
  );

const CELL_MAX = 100;
const cellText = (v: unknown) => {
  const t = (typeof v === 'string' ? v : v === undefined || v === null ? '' : String(v as number)).split('\n')[0];
  return t.length > CELL_MAX ? t.slice(0, CELL_MAX) + '…' : t;
};
const modNames = (names: string | string[]) => {
  const all = [names].flat();
  return all.slice(0, 3).join(', ') + (all.length > 3 ? ` +${all.length - 3}` : '');
};

export function breakdownOf(e: calculator.EntryPlain | undefined, maxRows: number): CalcBreakdown | undefined {
  if (!e) return undefined;
  const cols = (e.Columns ?? []).map((c) => ({ label: c.Label, key: c.Key }));
  const rows = ((e.Rows ?? []) as (Record<string, unknown> | null | undefined)[]).filter((r): r is Record<string, unknown> => !!r);
  const out: CalcBreakdown = { label: e.Label || undefined, lines: e.Lines ?? [] };
  if (e.Radius) out.radius = e.Radius;
  if (cols.length) {
    out.columns = cols;
    out.rows = rows.slice(0, maxRows).map((r) => Object.fromEntries(cols.map((c) => [c.key, cellText(r[c.key])])));
    if (rows.length > maxRows) out.more = rows.length - maxRows;
  }
  const tables: CalcTable[] = [];
  if (e.Reservations?.length) {
    tables.push({
      label: 'Reservations',
      columns: RESERVATION_COLUMNS,
      rows: e.Reservations.map((r) => ({
        skillName: r.SkillName,
        base: r.Base,
        mult: r.Mult ?? '',
        more: r.More ?? '',
        inc: r.Inc ?? '',
        efficiency: r.Efficiency ?? '',
        efficiencyMore: r.EfficiencyMore ?? '',
        total: r.Total
      }))
    });
  }
  if (e.DamageTypes?.length) {
    tables.push({
      label: 'Damage types',
      columns: DAMAGE_TYPE_COLUMNS,
      rows: e.DamageTypes.map((d) => ({
        source: d.Source,
        base: d.Base ?? '',
        inc: d.Inc ?? '',
        more: d.More ?? '',
        convSrc: d.ConvSrc ?? '',
        total: d.Total,
        convDst: d.ConvDst ?? '',
        gainDst: d.GainDst ?? ''
      }))
    });
  }
  if (tables.length) out.tables = tables;
  return out.lines.length || out.columns || out.tables ? out : undefined;
}

const statOf = (format: string) => /\{\d+:output:(\w+)\}/.exec(format)?.[1];

export function calcsView(sections: CalcSection[], out: Outputs, maxRows = Infinity): CalcsView {
  const F = (s: string | undefined, col?: CalcDataCol, id?: string) => FormatStr(s ?? '', out, col, id);
  const breakdowns: Record<string, CalcBreakdown | undefined> = {};
  const view = sections.map((section): CalcSectionView => {
    const [width, id, col, color, blocks] = [section[0], section[1], section[2], section[3], section[4]];
    const applicable = sectionEnabled(section, out);
    const entry: CalcSectionView = { id, title: blocks[0]?.label ?? id, wide: width === 3, col, color, applicable };
    if (!applicable) return entry;
    entry.blocks = blocks
      .map((block, j): CalcBlock => {
        const rows: CalcRow[] = [];
        rowsOf(block.data).forEach((row, k) => {
          if (!checkFlag(row, out)) return;
          const columns = elementsOf<CalcDataCol>(row);
          const cells = columns.map((c, l): CalcCell => {
            if (!checkFlag(c, out)) return { text: '' };
            const props = elementsOf<CalcDataColProp>(c);
            const cell: CalcCell = { text: F(formatOf(c), c, cellId(id, j, k, l)) };
            const bd = props.find((e) => e.breakdown)?.breakdown;
            const mods = props.filter((e) => e.modName && e.modType);
            if (bd && out.Breakdown?.[bd]) {
              breakdowns[bd] ??= breakdownOf(out.Breakdown[bd], maxRows);
              if (breakdowns[bd]) cell.bd = bd;
            }
            if (mods.length) cell.mods = mods.map((e) => `${modNames(e.modName!)} ${e.modType}${e.enemy ? ' (enemy)' : ''}`);
            const modEntries = props.filter((e) => e.modName);
            if (modEntries.length)
              cell.sections = modEntries.map((e) => ({
                label: e.label,
                sd: {
                  ModName: typeof e.modName === 'string' ? [e.modName] : [...e.modName!],
                  Multi: typeof e.modName !== 'string',
                  ModType: e.modType ?? '',
                  Cfg: e.cfg ?? '',
                  ModSource: e.modSource ?? '',
                  Enemy: !!e.enemy,
                  Actor: e.actor ?? ''
                }
              }));
            return cell;
          });
          const first = columns[0];
          const stat = statOf(formatOf(first));
          const firstBd = elementsOf<CalcDataColProp>(first).find((e) => e.breakdown)?.breakdown;
          const gauge = stat ? statGauge(stat, out.Output ?? {}, firstBd ? out.Breakdown?.[firstBd]?.Lines : undefined) : undefined;
          rows.push(gauge ? { label: F(row.label), cells, gauge } : { label: F(row.label), cells });
        });
        return { label: F(block.label), extra: block.data.extra ? F(block.data.extra) : undefined, rows };
      })
      .filter((b) => b.rows.some((r) => r.label));
    return entry;
  });
  return {
    sections: view,
    breakdowns: Object.fromEntries(Object.entries(breakdowns).filter((e): e is [string, CalcBreakdown] => !!e[1]))
  };
}
