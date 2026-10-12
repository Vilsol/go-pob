// Build.lua:1649-1686

import { colorCodes } from './colors';
import { currentNumberFormat, type NumberFormat } from './number_format';

export interface FormatSpec {
  fmt?: string;
  pc?: boolean;
  mod?: boolean;
  label?: string;
  chaosInoc?: boolean;
  /* Build.lua:1658-1667 */
  compactValue?: boolean;
}

export interface FormattedStat {
  value: string;
  color: string;
  overCap?: string;
}

export const VALUE_COLOR = colorCodes.WHITE;
export const SUFFIX_COLOR = '#808080';
const CHAOS_INOC_COLOR = '#808080';

const FORMAT_RE = /^(\+?)(?:\.(\d+))?([df])(.*)$/;
const INT64_MIN = '-9223372036854775808';

const toFixedLua = (v: number, p: number): string => {
  const abs = Math.abs(v);
  const out = abs >= 1e21 ? abs.toLocaleString('en-US', { useGrouping: false, minimumFractionDigits: p, maximumFractionDigits: p }) : abs.toFixed(p);
  return v < 0 || Object.is(v, -0) ? `-${out}` : out;
};

export const luaFormat = (fmt: string, v: number): string => {
  const m = FORMAT_RE.exec(fmt);
  if (!m) {
    throw new Error(`unsupported stat format: ${fmt}`);
  }
  const [, plus, precision, conv, rest] = m;
  const suffix = rest.replace(/%%/g, '%');
  let body: string;
  if (conv === 'd') {
    if (!Number.isFinite(v) || Math.abs(v) >= 2 ** 63) {
      body = INT64_MIN;
    } else {
      const n = Math.trunc(v);
      body = (plus && n >= 0 ? '+' : '') + String(n === 0 ? 0 : n);
    }
  } else if (Number.isNaN(v)) {
    body = (plus ? '+' : '') + 'nan';
  } else if (!Number.isFinite(v)) {
    body = v > 0 ? `${plus}inf` : '-inf';
  } else {
    const s = toFixedLua(v, precision === undefined ? 6 : Number(precision));
    body = plus && !s.startsWith('-') ? `+${s}` : s;
  }
  return body + suffix;
};

/* Common.lua:741-766 */
export const formatNumSep = (str: string, opts: NumberFormat = currentNumberFormat()): string =>
  str.replace(/-?\d+\.?\d+/g, (m) => {
    const [, minus, integer, fraction] = /(-?)(\d+)(\.?\d*)/.exec(m)!;
    const grouped = opts.showThousandsSeparators ? integer.replace(/\B(?=(\d{3})+(?!\d))/g, opts.thousandsSeparator) : integer;
    return minus + grouped + fraction.replace('.', opts.decimalSeparator);
  });

// src/Modules/Build.lua:1649-1686
export const formatStat = (statData: FormatSpec, statVal: number, overCapStatVal?: number, colorOverride?: string): FormattedStat => {
  const val = statVal * (statData.pc || statData.mod ? 100 : 1) - (statData.mod ? 100 : 0);
  let color = colorOverride ?? (statVal >= 0 ? VALUE_COLOR : statData.chaosInoc ? CHAOS_INOC_COLOR : colorCodes.NEGATIVE);
  if (statData.label === 'Unreserved Life' && statVal === 0) {
    color = colorCodes.NEGATIVE;
  }

  let valStr: string | undefined;
  if (statData.compactValue && currentNumberFormat().useCompactValues && Number.isFinite(val)) {
    const abs = Math.abs(val);
    if (abs >= 1e9) valStr = luaFormat('.1f', val / 1e9) + 'B';
    else if (abs >= 1e6) valStr = luaFormat('.1f', val / 1e6) + 'M';
    else if (abs >= 1e4) valStr = luaFormat('.1f', val / 1e3) + 'K';
  }
  if (valStr === undefined) {
    valStr = luaFormat(statData.fmt ?? 'd', val);
    const trimmed = /^([+-]?\d+\.\d+)(\D*)$/.exec(valStr);
    if (trimmed) {
      valStr = trimmed[1].replace(/0+$/, '').replace(/\.$/, '') + trimmed[2];
    }
  }

  const out: FormattedStat = { value: formatNumSep(valStr), color };
  if (overCapStatVal !== undefined && overCapStatVal > 0) {
    out.overCap = ` (+${luaFormat('d', overCapStatVal)}%)`;
  }
  return out;
};
