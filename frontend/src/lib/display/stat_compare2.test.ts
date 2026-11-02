import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { gunzipSync } from 'node:zlib';
import { fileURLToPath } from 'node:url';
import { addStatCompares2 } from './stat_compare2';
import { lineText } from '../skill_tree/stat_compare';

// calculator2/testdata/item_compare_pob.jsonl.gz: PoB-PoE2's item tooltips over calculator2/testdata/items_edit.xml,
// each with the calculations it ran
type Calc = { slot: string; toggle: boolean; output: Record<string, number>; minion: Record<string, number> };
type Row = { base?: Record<string, number>; skillFlags?: Record<string, boolean>; minion?: boolean; item?: string; lines?: string[]; calcs?: Calc[] };
const rows = gunzipSync(readFileSync(fileURLToPath(new URL('../../../../calculator2/testdata/item_compare_pob.jsonl.gz', import.meta.url))))
  .toString('utf8')
  .trim()
  .split('\n')
  .map((l) => JSON.parse(l) as Row);
const plain = (s: string) => s.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '');

describe('the PoE2 stat differences', () => {
  it("lists PoB-PoE2's lines for each item tooltip block", () => {
    const [head, ...items] = rows;
    let blocks = 0;
    for (const r of items) {
      // split the tooltip into its blocks: a header, an optional "(replacing …)" line, its stat lines
      const want: { header: string; lines: string[] }[] = [];
      for (const line of r.lines!) {
        if (/will give you:$/.test(plain(line))) want.push({ header: plain(line), lines: [] });
        else if (!/^\(replacing /.test(plain(line))) want.at(-1)!.lines.push(line);
      }
      for (const b of want) {
        const slot = /in (.+) will give you:$/.exec(b.header)?.[1] ?? '';
        const calc = r.calcs!.filter((c) => c.slot === slot).at(-1)!;
        const sections = addStatCompares2(head.skillFlags!, {}, head.minion!, { output: head.base! }, { output: calc.output });
        const got = sections.flatMap((s) => s.lines).map((l) => `${l.better ? '^x33FF77' : '^xDD0022'}${lineText(l)}^7`);
        expect(got, `${r.item}: ${b.header}`).toEqual(b.lines);
        blocks++;
      }
    }
    expect(blocks).toBeGreaterThan(4);
  });
});
