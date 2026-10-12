import type { exposition } from '$lib/types';

export type TextCells = { head: string[]; cells: string[]; tail: string[]; blank?: string };

const isBlank = (l: string) => l.trim() === '';

export const textToCells = (text: string): TextCells => {
  const lines = text.split('\n');
  const first = lines.findIndex((l) => !isBlank(l));
  if (first < 0) return { head: [], cells: [''], tail: [], blank: text };
  const last = lines.length - 1 - [...lines].reverse().findIndex((l) => !isBlank(l));
  return { head: lines.slice(0, first), cells: lines.slice(first, last + 1), tail: lines.slice(last + 1) };
};

export const cellsToText = (t: TextCells): string => {
  if (t.cells.every(isBlank)) {
    return t.blank ?? (t.head.length || t.tail.length ? [...t.head, ...t.tail].join('\n') : '');
  }
  return [...t.head, ...t.cells, ...t.tail].join('\n');
};

export const lineLabel = (line: exposition.ModLine): string => {
  switch (line.Status) {
    case 'parsed':
      return 'parsed';
    case 'unsupported':
      return 'no match';
    case 'comment':
      return 'comment';
    default:
      return '';
  }
};

export const modSummary = (line: exposition.ModLine): string =>
  (line.Mods ?? []).map((m) => `${m.Name} ${m.Type} ${m.Value}${m.Tags?.length ? ` (${m.Tags.join(', ')})` : ''}`).join('; ');
