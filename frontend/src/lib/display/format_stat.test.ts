import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { formatNumSep, formatStat, VALUE_COLOR, SUFFIX_COLOR } from './format_stat';
import { numberFormat } from './number_format';

type Case = { fmt: string; pc: boolean; mod: boolean; label: string; value: number; overCap: number | null; out: string };
const cases = JSON.parse(readFileSync(fileURLToPath(new URL('../../../../testdata/oracle/format_stat_expected.json', import.meta.url)), 'utf8')) as Case[];

const code = (color: string) => (color === VALUE_COLOR ? '^7' : `^x${color.slice(1).toUpperCase()}`);

describe('formatStat against PoB', () => {
  it('has cases', () => {
    expect(cases.length).toBeGreaterThan(100);
  });

  it('renders every case the way PoB does', () => {
    const diffs: string[] = [];
    for (const c of cases) {
      const f = formatStat({ fmt: c.fmt, pc: c.pc, mod: c.mod, label: c.label }, c.value, c.overCap ?? undefined);
      const got = code(f.color) + f.value + (f.overCap ? `^x${SUFFIX_COLOR.slice(1)}${f.overCap}` : '');
      if (got !== c.out) {
        diffs.push(`${c.fmt} pc=${c.pc} mod=${c.mod} ${c.value}: got ${got}, PoB ${c.out}`);
      }
    }
    expect(diffs).toEqual([]);
  });
});

// Main.lua:1056-1079, Common.lua:741-766, Build.lua:1658-1667
describe('the number display options', () => {
  const on = { showThousandsSeparators: true, thousandsSeparator: ',', decimalSeparator: '.', useCompactValues: false };
  it('separators off, other separator characters', () => {
    expect(formatNumSep('1234567.5', { ...on, showThousandsSeparators: false })).toBe('1234567.5');
    expect(formatNumSep('-1234567.25 (x)', { ...on, thousandsSeparator: '.', decimalSeparator: ',' })).toBe('-1.234.567,25 (x)');
    expect(formatNumSep('1234.5', { ...on, showThousandsSeparators: false, decimalSeparator: ',' })).toBe('1234,5');
  });
  it('compact large numbers on a stat that allows it', () => {
    numberFormat.set({ ...on, useCompactValues: true });
    try {
      expect(formatStat({ fmt: '.1f', compactValue: true }, 12345.6).value).toBe('12.3K');
      expect(formatStat({ fmt: '.1f', compactValue: true }, 4567890).value).toBe('4.6M');
      expect(formatStat({ fmt: '.1f', compactValue: true }, 2.5e9).value).toBe('2.5B');
      expect(formatStat({ fmt: '.1f', compactValue: true }, 9999).value).toBe('9,999');
      expect(formatStat({ fmt: '.1f' }, 12345.6).value, 'a stat without compactValue').toBe('12,345.6');
    } finally {
      numberFormat.set(on);
    }
    expect(formatStat({ fmt: '.1f', compactValue: true }, 12345.6).value).toBe('12,345.6');
  });
});
