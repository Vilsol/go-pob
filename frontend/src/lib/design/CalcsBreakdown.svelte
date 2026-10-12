<script lang="ts">
  import PobText from './PobText.svelte';
  import Button from './Button.svelte';
  import { pobPlain } from './pob_text';
  import type { CalcBreakdown, ItemTip } from './fixtures';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import RadiusVisual from '$lib/components/radius/RadiusVisual.svelte';
  import type { Snippet } from 'svelte';

  // CalcBreakdownControl.lua:174-209
  export type ModRow = {
    Value: string;
    Name: string;
    Flags: string;
    Tags: string;
    Source: string;
    SourceName: string;
    Totals?: string[];
    ItemName?: string;
    NodeID?: number;
  };
  export type ModSectionRef = { label?: string; sd: unknown };
  export type ModTable = { label?: string; rows: ModRow[] };
  export type BreakdownPart = { head?: string; text: string; bd?: CalcBreakdown; mods?: string[]; modTables?: ModTable[] };
  type Props = {
    id: string;
    title?: string;
    parts: BreakdownPart[];
    pinned: boolean;
    pinnedTitle?: string;
    onunpin: () => void;
    /* CalcBreakdownControl.lua:400-403 */
    itemtip?: (name: string) => Promise<ItemTip | undefined>;
    /* CalcBreakdownControl.lua:405-413, 594-612 */
    nodeview?: Snippet<[number]>;
  };
  let { id, title, parts, pinned, pinnedTitle, onunpin, itemtip, nodeview }: Props = $props();
  let itemHover = $state<{ anchor: HTMLElement; name: string; tip?: ItemTip }>();
  const showItem = (name: string | undefined) => (e: MouseEvent) => {
    if (!name || !itemtip) return;
    const anchor = e.currentTarget as HTMLElement;
    itemHover = { anchor, name };
    itemtip(name)
      .then((tip) => {
        if (itemHover?.name === name && itemHover.anchor === anchor) itemHover = { anchor, name, tip };
      })
      .catch(() => {});
  };
  // CalcBreakdownControl.lua:595-599
  let nodeHover = $state<{ node: number; x: number; y: number }>();
  const showNode = (node: number | undefined) => (e: MouseEvent) => {
    if (!node || !nodeview) return;
    const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const x = r.right + 309 > window.innerWidth ? r.left - 309 : r.right + 5;
    nodeHover = { node, x: Math.max(0, x), y: Math.max(0, Math.min(r.top, window.innerHeight - 304)) };
  };
  const enterSource = (r: ModRow) => (e: MouseEvent) => {
    showItem(r.ItemName)(e);
    showNode(r.NodeID)(e);
  };
  const leaveSource = () => {
    itemHover = undefined;
    nodeHover = undefined;
  };
  // CalcBreakdownControl.lua:299-306
  const COLS: { key: keyof ModRow; label: string }[] = [
    { key: 'Value', label: 'Value' },
    { key: 'Name', label: 'Stat' },
    { key: 'Flags', label: 'Skill types' },
    { key: 'Tags', label: 'Notes' },
    { key: 'Source', label: 'Source' },
    { key: 'SourceName', label: 'Source Name' }
  ];
  const colsOf = (rows: ModRow[]) => COLS.filter((c) => c.key !== 'Name' || rows.some((r) => r.Name));
</script>

<aside {id} aria-label="Breakdown" class="flex w-80 shrink-0 flex-col border-l border-line bg-surface-1 max-[1500px]:w-64">
  <div class="flex min-h-11 items-center gap-2 border-b border-line px-3 py-2">
    <h2 class="min-w-0 grow truncate text-13 font-semibold">{title ?? 'Breakdown'}</h2>
    {#if title}<span class={['rounded-lg px-1.5 text-10 font-semibold', pinned ? 'bg-surface-3 text-accent' : 'bg-surface-2 text-text-2']}
        >{pinned ? 'PINNED' : 'PREVIEW'}</span
      >{/if}
    {#if pinned || pinnedTitle}<Button size="2xs" onclick={onunpin}>Unpin</Button>{/if}
  </div>
  {#if pinnedTitle && !pinned}<div class="border-b border-line px-3 py-1 text-11 text-text-3">Pinned: {pinnedTitle} · leave the row to go back</div>{/if}
  <div class="scroll flex grow flex-col gap-2.5 px-3 py-2.5 text-11">
    {#if !title}
      <p class="m-0 text-12 text-text-3">Hover a row (or focus it) to preview its breakdown here; click it to pin the breakdown while you scan.</p>
    {:else if !parts.length}
      <p class="m-0 text-12 text-text-3">No breakdown for this row.</p>
    {/if}
    {#each parts as p, i (i)}
      <div class="flex flex-col gap-1">
        <div class="kv items-baseline border-b border-line-faint pb-0.5">
          <span class="text-10 font-semibold tracking-caps-tight text-text-2 uppercase"
            >{#if p.head}<PobText text={p.head.replace(/:$/, '')} />{:else}Value{/if}</span>
          <span class="font-mono text-12"><PobText text={p.text} /></span>
        </div>
        {#if p.bd}
          {#each p.bd.lines as l, j (j)}<div class="font-mono leading-[1.45] text-text-2"><PobText text={l} /></div>{/each}
          {#if p.bd.radius}<RadiusVisual radius={p.bd.radius} />{/if}
          {#if p.bd.columns && p.bd.rows}
            <div role="table" aria-label="Sources" class="flex flex-col" style:--n={p.bd.columns.length}>
              <div role="row" class="grid grid-cols-[repeat(var(--n),minmax(0,1fr))] gap-x-2 border-b border-line py-0.5 text-10 text-text-3">
                {#each p.bd.columns as c (c.key)}<span role="columnheader" class="truncate" title={c.label}>{c.label}</span>{/each}
              </div>
              {#each p.bd.rows as r, j (j)}
                <div role="row" class="grid grid-cols-[repeat(var(--n),minmax(0,1fr))] gap-x-2 border-b border-surface-2 py-0.5">
                  {#each p.bd.columns as c, k (c.key)}<span role="cell" class={['truncate', k === 0 && 'font-mono']} title={pobPlain(r[c.key])}
                      ><PobText text={r[c.key]} /></span
                    >{/each}
                </div>
              {/each}
              {#if p.bd.more}<span class="pt-0.5 text-text-3">+ {p.bd.more} more sources</span>{/if}
            </div>
          {/if}
          {#each p.bd.tables ?? [] as t (t.label)}
            <div role="table" aria-label={t.label} class="flex flex-col overflow-x-auto" style:--n={t.columns.length}>
              <div role="row" class="grid grid-cols-[repeat(var(--n),minmax(3.5rem,1fr))] gap-x-2 border-b border-line py-0.5 text-10 text-text-3">
                {#each t.columns as c (c.key)}<span role="columnheader" class="truncate" title={c.label}>{c.label}</span>{/each}
              </div>
              {#each t.rows as r, j (j)}
                <div role="row" class="grid grid-cols-[repeat(var(--n),minmax(3.5rem,1fr))] gap-x-2 border-b border-surface-2 py-0.5">
                  {#each t.columns as c (c.key)}<span role="cell" class="truncate font-mono" title={pobPlain(r[c.key])}><PobText text={r[c.key]} /></span
                    >{/each}
                </div>
              {/each}
            </div>
          {/each}
        {/if}
        {#if p.modTables}
          {#each p.modTables as s, j (j)}
            {@const t = { rows: s.rows, cols: colsOf(s.rows) }}
            {#if t.rows.length}
              <div role="table" aria-label={s.label ?? 'Modifiers'} class="flex flex-col overflow-x-auto" style:--n={t.cols.length}>
                {#if s.label}<div class="text-10 font-semibold text-text-2">{s.label}:</div>{/if}
                <div role="row" class="grid grid-cols-[repeat(var(--n),minmax(3.5rem,1fr))] gap-x-2 border-b border-line py-0.5 text-10 text-text-3">
                  {#each t.cols as c (c.key)}<span role="columnheader" class="truncate" title={c.label}>{c.label}</span>{/each}
                </div>
                {#each t.rows as r, k (k)}
                  <div role="row" class="grid grid-cols-[repeat(var(--n),minmax(3.5rem,1fr))] gap-x-2 border-b border-surface-2 py-0.5">
                    {#each t.cols as c (c.key)}<!-- svelte-ignore a11y_interactive_supports_focus --><span
                        onmouseenter={c.key === 'SourceName' ? enterSource(r) : undefined}
                        onmouseleave={c.key === 'SourceName' ? leaveSource : undefined}
                        role="cell"
                        class={['truncate', c.key === 'Value' && 'font-mono']}
                        title={c.key === 'Source' && r.Totals?.length ? r.Totals.join('\n') : pobPlain(String(r[c.key] ?? ''))}
                        ><PobText text={String(r[c.key] ?? '')} /></span
                      >{/each}
                  </div>
                {/each}
              </div>
            {/if}
          {/each}
        {:else if p.mods}
          <div class="text-text-3">The modifiers summed into it (per-source list not computed yet):</div>
          {#each p.mods as m, k (k)}<div class="font-mono break-all text-text-2">{m}</div>{/each}
        {/if}
      </div>
    {/each}
  </div>
</aside>
{#if nodeHover && nodeview}
  <div class="pointer-events-none fixed z-50 border-2 border-white" style:left="{nodeHover.x}px" style:top="{nodeHover.y}px">
    {@render nodeview(nodeHover.node)}
  </div>
{/if}
{#if itemHover?.tip}
  <FloatingItemTip
    anchor={itemHover.anchor}
    id="{id}-item-tip"
    item={itemHover.name}
    base=""
    rarity="normal"
    mods={[]}
    impactTitle=""
    impact={[]}
    tooltip={itemHover.tip} />
{/if}
