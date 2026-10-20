import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { KINDS, census, scanSource } from './raw_controls';

type Allow = { file: string; kind: string; count: number; reason: string };
const config = JSON.parse(readFileSync(resolve(import.meta.dirname, 'raw_controls.json'), 'utf8')) as { ceiling: Record<string, number>; allow: Allow[] };

describe('raw control census', () => {
  it('finds planted controls across lines, in expressions and inside blocks', () => {
    const planted = `<script>let x = $state(false);</script>
{#if x}
  <button
    type="button"
    class={['btn', x && 'btn-primary']}>go</button>
{/if}
<div class="btn-row"><span class:btn-ghost={x}></span></div>
<Button class="btn-bare">component</Button>
<input
  class="input input-sm" />
<input type="text" class="mod-cell" />
<textarea class="input"></textarea>
{#each [1] as i (i)}<select><option>{i}</option></select>{/each}
<label><input type="checkbox" class="check" /> a</label>`;
    const hits = scanSource(planted).map((h) => `${h.line} ${h.kind} ${h.element}`);
    expect(hits).toEqual(['3 btn button', '7 btn div', '7 btn span', '9 input input', '12 input textarea', '13 select select', '14 checkbox input']);
  });

  it('stays at or under the ceiling, and the ceiling follows it down', () => {
    const hits = census();
    const allowed = new Map(config.allow.map((a) => [`${a.file}\t${a.kind}`, a]));
    const per = new Map<string, number>();
    for (const h of hits) per.set(`${h.file}\t${h.kind}`, (per.get(`${h.file}\t${h.kind}`) ?? 0) + 1);

    const stale = config.allow
      .filter((a) => (per.get(`${a.file}\t${a.kind}`) ?? 0) !== a.count)
      .map((a) => `${a.file} ${a.kind}: allowed ${a.count}, found ${per.get(`${a.file}\t${a.kind}`) ?? 0}`);
    expect(stale, 'allow-list counts must match the file exactly').toEqual([]);
    expect(config.allow.every((a) => a.reason.trim().length > 0)).toBe(true);

    const counted = hits.filter((h) => !allowed.has(`${h.file}\t${h.kind}`));
    const got = Object.fromEntries(KINDS.map((k) => [k, counted.filter((h) => h.kind === k).length]));
    const over = KINDS.filter((k) => got[k] > config.ceiling[k]).flatMap((k) => counted.filter((h) => h.kind === k).map((h) => `src/${h.file}:${h.line} ${k}`));
    expect(over, `raw controls over the ceiling ${JSON.stringify(config.ceiling)}; use the primitives`).toEqual([]);
    expect(got, 'the count went down: lower the ceiling in harness/raw_controls.json').toEqual(config.ceiling);
  });
});
