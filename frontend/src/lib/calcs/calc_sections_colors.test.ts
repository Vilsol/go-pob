import { describe, expect, it } from 'vitest';
import { calcSections } from './calc_sections';
import { pobPlain } from '$lib/design/pob_text';

const strings = (v: unknown, out: string[] = []): string[] => {
  if (typeof v === 'string') out.push(v);
  else if (v && typeof v === 'object') for (const x of Object.values(v)) strings(x, out);
  return out;
};

// CalcSections.lua:1430, 1608
describe('calc section text', () => {
  it('carries only colour codes the PoB-text renderer draws', () => {
    expect(strings(calcSections).filter((s) => pobPlain(s).includes('^'))).toEqual([]);
  });
});
