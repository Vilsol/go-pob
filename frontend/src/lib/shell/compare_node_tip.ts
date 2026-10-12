import { lineText, type StatCompareSection } from '$lib/skill_tree/stat_compare';

export type TipLine = { text: string; sep: boolean };

/* Build.lua:1924-1936 */
export type WhatIfBlock = { header: string; sections: StatCompareSection[] };

const POSITIVE = '^x33FF77';
const NEGATIVE = '^xDD0022';
const TIP = '^x80A080';

// Build.lua:1924-1936, 1889-1913
const blockLines = (b: WhatIfBlock): TipLine[] => {
  const out: TipLine[] = [];
  let header = b.header;
  const split = b.sections.length > 1;
  for (const s of b.sections) {
    if (split && s.actor) header = `${header}${header ? '\n' : ''}^7${s.actor}:`;
    for (const h of header ? header.split('\n') : []) out.push({ text: h.startsWith('^') ? h : `^7${h}`, sep: false });
    header = '';
    for (const l of s.lines) out.push({ text: `${l.better ? POSITIVE : NEGATIVE}${lineText(l)}${l.perPoint ? ` ^8${l.perPoint}` : ''}`, sep: false });
  }
  return out;
};

/* PassiveTreeView.lua:1695-1724 */
export const nodeWhatIfLines = (blocks: WhatIfBlock[], pathLength: number): TipLine[] => {
  const out: TipLine[] = [{ text: '', sep: true }];
  for (const b of blocks) out.push(...blockLines(b));
  if (!blocks.some((b) => b.sections.some((s) => s.lines.length))) {
    out.push({ text: `^7No changes from allocating this node${pathLength > 1 ? ' or the nodes leading to it' : ''}.`, sep: false });
  }
  if (pathLength > 0) {
    out.push({ text: '', sep: true }, { text: `^7${pathLength} points to node`, sep: false });
    if (pathLength > 1)
      out.push({ text: `${TIP}Tip: To reach this node by a different path, hold Shift, then trace the path and click this node`, sep: false });
  }
  return out;
};
