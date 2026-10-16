import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const css = (f: string) => readFileSync(new URL(`./styles/${f}`, import.meta.url), 'utf8');
const block = (text: string, selector: string) => {
  const at = text.indexOf(selector);
  if (at < 0) throw new Error(`no ${selector}`);
  return text.slice(at, text.indexOf('}', at));
};
const tokens = (text: string) => Object.fromEntries([...text.matchAll(/--color-([\w-]+):\s*(#[0-9a-fA-F]{6})/g)].map((m) => [m[1], m[2]]));

const dark = tokens(block(css('tokens.css'), '@theme'));
const THEMES: Record<string, Record<string, string>> = {
  dark,
  light: { ...dark, ...tokens(block(css('themes.css'), ":root[data-theme='light']")) },
  contrast: { ...dark, ...tokens(block(css('themes.css'), ":root[data-theme='contrast']")) }
};

const rgb = (hex: string) => [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16));
const lum = ([r, g, b]: number[]) => {
  const c = (v: number) => ((v /= 255) <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4);
  return 0.2126 * c(r) + 0.7152 * c(g) + 0.0722 * c(b);
};
const contrast = (a: number[], b: number[]) => {
  const [x, y] = [lum(a), lum(b)].sort((p, q) => q - p);
  return (x + 0.05) / (y + 0.05);
};
const tinted = (tone: number[], ground: number[]) => ground.map((g, i) => g + (tone[i] - g) * 0.16);

const SECTION_AND_DAMAGE = [
  'life',
  'mana',
  'es',
  'code-ward',
  'code-evasion',
  'calc-off',
  'calc-def',
  'normal',
  'stat-fire',
  'stat-cold',
  'stat-light',
  'stat-chaos'
];
const TINTS = ['better', 'worse', 'text'];

describe('Calcs colour tokens', () => {
  for (const [theme, t] of Object.entries(THEMES)) {
    it(`read at 4.5:1 or better in the ${theme} theme`, () => {
      const box = rgb(t['surface-1']);
      const low = [...SECTION_AND_DAMAGE, ...TINTS]
        .map((name) => [name, contrast(rgb(t[name]), box)] as const)
        .filter(([, c]) => c < 4.5)
        .map(([n, c]) => `${n} ${c.toFixed(2)}`);
      expect(low).toEqual([]);
      const lowTint = TINTS.map((name) => [name, contrast(rgb(t[name]), tinted(rgb(t[name]), box))] as const)
        .filter(([, c]) => c < 4.5)
        .map(([n, c]) => `${n} ${c.toFixed(2)}`);
      expect(lowTint, 'on its own tint').toEqual([]);
    });
  }
});
