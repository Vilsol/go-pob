import { writable } from 'svelte/store';

const DIGITS = ['#000000', '#FF0000', '#00FF00', '#0000FF', '#FFFF00', '#FF00FF', '#00FFFF', '#FFFFFF', '#B2B2B2', '#666666'];
const CODE = /\^(?:[x#]([0-9A-Fa-f]{6})|(\d))/g;
// Main.lua:117
const NOT_SUPPORTED = ' ^8(Not supported in PoB yet)';

/* Main.lua:1127-1131 */
export const notSupportedNote = writable(true);

export type PobSegment = { text: string; color?: string };

export function pobSegments(text: string, note = true): PobSegment[] {
  if (!note) text = text.replaceAll(NOT_SUPPORTED, '');
  const out: PobSegment[] = [];
  let color: string | undefined;
  let last = 0;
  for (const m of text.matchAll(CODE)) {
    if (m.index > last) out.push({ text: text.slice(last, m.index), color });
    color = m[1] ? `#${m[1].toUpperCase()}` : DIGITS[Number(m[2])];
    last = m.index + m[0].length;
  }
  if (last < text.length) out.push({ text: text.slice(last), color });
  return out;
}

export const pobPlain = (text: string) => text.replace(CODE, '');
