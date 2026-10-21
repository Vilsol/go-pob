import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { census, scanSource } from './edge_whitespace';

type Allow = { file: string; count: number; reason: string };
const config = JSON.parse(readFileSync(resolve(import.meta.dirname, 'edge_whitespace.json'), 'utf8')) as { ceiling: number; allow: Allow[] };

const hits = (src: string) => scanSource(src).map((h) => `${h.line} ${h.side}`);

describe('edge whitespace census', () => {
  it('flags the Notice that rendered " Offline…"', () => {
    const notice = `<script>let { title, actions, children } = $props();</script>
<div
  role="status"
  class={['rounded border', title && 'flex']}>
  {#if title}<span class="eyebrow">{title}</span>{/if}
  {#if actions}
    <span>{@render children()}</span>
    <div class="grow"></div>
    <div class="flex">{@render actions()}</div>
  {:else}
    {@render children()}
  {/if}
</div>`;
    expect(hits(notice)).toEqual(['5 start']);
  });

  it('flags the heading whose name ended in a space without a count', () => {
    const heading = `<script>let { level, count, children } = $props();</script>
{#snippet heading()}
  <svelte:element this={\`h\${level}\`} class="subhead"
    >{@render children()}
    {#if count}<span class="text-text-3">{count}</span>{/if}</svelte:element>
{/snippet}
{@render heading()}`;
    expect(hits(heading)).toEqual(['4 end']);
  });

  it('flags whitespace kept beside an empty sibling in blocks, snippets, each and component children', () => {
    const planted = `<script>let { a, xs, c } = $props();</script>
<p>{#if a}<i></i>{/if}
  Name</p>
{#snippet s()}
  {#each xs as x}<b>{x}</b>{/each}
  {c}
{/snippet}
<Button
  variant="bare">{#if a}<Icon />{:else}
    {#if c}<i></i>{/if}
    Label{/if}</Button>
<h3><!-- note -->{#if a}<i></i>{/if}<!-- note -->
  Title</h3>`;
    expect(hits(planted)).toEqual(['2 start', '5 start', '10 start', '12 start']);
  });

  it('leaves the whitespace Svelte trims, keeps on purpose, or that only touches elements', () => {
    const harmless = `<script>let { a, c, xs, children, footer } = $props();</script>
<div>
  <b>a</b>
  <i>b</i>
</div>
<p>
  <!-- comment -->
  Name
</p>
{#snippet s(i)}
  {@const m = i}
  Name
{/snippet}
<h2>Name{#if c}{' '}<span>{c}</span>{/if}</h2>
<div>
  {#each xs as x}<i>{x}</i>{:else}<b>none</b>{/each}
  Name
</div>
<pre>{#if a}<i></i>{/if}
  kept</pre>
<section>
  {#if a}<header></header>{/if}
  <main></main>
  {@render footer?.()}
</section>
<dialog>
  {@render children()}{#if a}<div>foot</div>{/if}
</dialog>
<svelte:head><title>x</title></svelte:head>
<p>{a} {c}</p>`;
    expect(hits(harmless)).toEqual([]);
  });

  it('stays at or under the ceiling, and the ceiling follows it down', () => {
    const all = census();
    const per = new Map<string, number>();
    for (const h of all) per.set(h.file, (per.get(h.file) ?? 0) + 1);
    const allowed = new Set(config.allow.map((a) => a.file));
    const stale = config.allow.filter((a) => (per.get(a.file) ?? 0) !== a.count).map((a) => `${a.file}: allowed ${a.count}, found ${per.get(a.file) ?? 0}`);
    expect(stale, 'allow-list counts must match the file exactly').toEqual([]);
    expect(config.allow.every((a) => a.reason.trim().length > 0)).toBe(true);

    const counted = all.filter((h) => !allowed.has(h.file));
    expect(
      counted.length > config.ceiling ? counted.map((h) => `src/${h.file}:${h.line} ${h.side}`) : [],
      "whitespace kept beside a block that can render nothing; join the nodes, or put {' '} inside the block"
    ).toEqual([]);
    expect(counted.length, 'the count went down: lower the ceiling in harness/edge_whitespace.json').toBe(config.ceiling);
  });
});
