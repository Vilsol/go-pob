import type { IconName } from './Icon.svelte';

// characters IBM Plex has no glyph for, drawn as icons so no line falls back to a system font
export const GLYPH_ICONS: Record<string, IconName> = {
  '▲': 'tri-up',
  '▼': 'tri-down',
  '▴': 'tri-up',
  '▾': 'tri-down',
  '●': 'dot',
  '★': 'star',
  '⚠': 'warn',
  '✗': 'close',
  '☐': 'box',
  '☑': 'box-check',
  '❝': 'quote',
  '⋯': 'more',
  '―': 'rule'
};
export const GLYPH_SPLIT = new RegExp(`([${Object.keys(GLYPH_ICONS).join('')}])`);
