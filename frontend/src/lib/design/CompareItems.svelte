<script lang="ts">
  import Checkbox from './Checkbox.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import PobText from './PobText.svelte';
  import SocketPreview from './SocketPreview.svelte';
  import TooltipLines from './TooltipLines.svelte';
  import { besideAnchor } from './items';
  import { tone as toneClass } from './tone';
  import type { ItemTip } from './fixtures';
  import type { StatCompareSection } from '$lib/skill_tree/stat_compare';
  import type { CompareItemSide } from './fixtures';
  import type { Tree } from '$lib/skill_tree/types';

  type Row = { slot: string; a: CompareItemSide; b: CompareItemSide; impact?: StatCompareSection[] };
  type JewelRow = Row & { label: string; node: number; aWarn: boolean; bWarn: boolean };
  type SocketView = { tree: Tree; radii: { inner: number; outer: number; col: string; nodes: number[] }[]; alloc: Set<number> };
  // CompareTab.lua:1439-1459, 3788-3856
  // CompareTab.lua:3954-3963
  // CompareTab.lua:3466-3637
  // CompareTab.lua:3857-3872, CompareTab.lua:3806-3809, 3950-3952
  type Props = {
    items: Row[];
    jewels?: JewelRow[];
    onitem?: (slot: string, equip: boolean) => void;
    itemtip?: (side: 'a' | 'b', name: string) => Promise<ItemTip | undefined>;
    cards?: (slot: string) => Promise<[string[], string[]]>;
    socketview?: (node: number) => Promise<SocketView | undefined>;
    onbuy?: (slot: string) => void;
  };
  let { items, jewels = [], onitem, itemtip, cards, socketview, onbuy }: Props = $props();

  // CompareTab.lua:766-773
  let expanded = $state(false);

  const lines = (r: Row) =>
    (r.impact ?? []).flatMap((s) =>
      s.lines.map((l) => ({ text: `${s.actor === 'Minion' ? 'Minion ' : ''}${l.label} ${l.value}${l.percent ? ` ${l.percent}` : ''}`, up: l.better }))
    );
  let hover = $state.raw<{ key: string; anchor: HTMLElement; name: string; tip?: ItemTip }>();
  const showItem = async (side: 'a' | 'b', name: string, anchor: HTMLElement) => {
    const key = `${side}:${name}`;
    hover = { key, anchor, name };
    const tip = await itemtip?.(side, name);
    if (hover?.key === key && tip) hover = { ...hover, tip };
  };
  // CompareTab.lua:3965-4010
  let equipHover = $state.raw<{ row: Row; anchor: HTMLElement }>();
  const equipLines = (r: Row) => {
    const head = [
      { Text: `^7Equipping this item in ${r.slot} will give you:`, Separator: false },
      ...(r.a ? [{ Text: `(replacing ${r.a.name})`, Separator: false }] : [])
    ];
    const body = lines(r).map((l) => ({ Text: `${l.up ? '^x33FF77' : '^xDD0022'}${l.text}`, Separator: false }));
    return [...head, ...(body.length ? body : [{ Text: '^7No changes.', Separator: false }])];
  };

  const same = (r: Row) => r.a?.name === r.b?.name;
  // TradeHelpers.lua:359-372
  const diffLabel = (r: Row) =>
    !r.a && !r.b ? '^8(both empty)' : same(r) ? '^x33FF77(match)' : !r.a ? '^xDD0022(missing)' : !r.b ? '^x80A080(extra)' : '^xFF9922(different)';
  const WARN = '^xFF9922  (tree missing allocated node)';
  const GRID = 'grid-cols-[120px_minmax(0,1fr)_minmax(0,1fr)_minmax(0,320px)] max-phone:grid-cols-2 max-phone:gap-y-1';

  let shown = $state.raw<Record<string, [string[], string[]]>>({});
  $effect(() => {
    if (!expanded || !cards) return;
    const want = [...items, ...jewels].map((r) => r.slot);
    for (const slot of want) {
      void cards(slot).then((c) => (shown = { ...shown, [slot]: c }));
    }
  });

  let socket = $state.raw<{ node: number; label: string; anchor: HTMLElement; view?: SocketView }>();
  const showSocket = async (j: JewelRow, anchor: HTMLElement) => {
    socket = { node: j.node, label: j.label, anchor };
    const view = await socketview?.(j.node);
    if (socket?.node === j.node && view) socket = { ...socket, view };
  };
</script>

{#snippet buttons(r: Row)}
  {#if (onitem || onbuy) && r.b}
    <span class="mt-1 flex gap-1 font-sans">
      {#if onbuy}<button class="btn btn-2xs" title="Search the trade site for items like B's" onclick={() => onbuy(r.slot)}>Buy</button>{/if}
      {#if onitem && !same(r)}
        <button class="btn btn-2xs" title="Add B's item to this build without equipping it" onclick={() => onitem(r.slot, false)}>Copy</button>
        <button
          class="btn btn-2xs"
          aria-describedby={equipHover?.row === r ? 'compare-equip-tip' : undefined}
          onmouseenter={(e) => (equipHover = { row: r, anchor: e.currentTarget })}
          onfocus={(e) => (equipHover = { row: r, anchor: e.currentTarget })}
          onmouseleave={() => (equipHover = undefined)}
          onblur={() => (equipHover = undefined)}
          onclick={() => onitem(r.slot, true)}>Equip</button>
      {/if}
    </span>
  {/if}
{/snippet}

{#snippet compactRow(r: Row, jewel?: JewelRow)}
  <div role="row" class={['data-row items-start gap-2.5 py-1.5', GRID]}>
    {#if jewel}
      <span role="cell" class="text-text-3 max-phone:col-span-full"
        ><span
          role="button"
          tabindex="0"
          class="underline decoration-dotted"
          onmouseenter={(e) => socketview && void showSocket(jewel, e.currentTarget)}
          onfocus={(e) => socketview && void showSocket(jewel, e.currentTarget)}
          onmouseleave={() => (socket = undefined)}
          onblur={() => (socket = undefined)}>{jewel.label}</span
        ></span>
    {:else}
      <span role="cell" class="text-text-3 max-phone:col-span-full">{r.slot}</span>
    {/if}
    {#each [r.a, r.b] as it, i (i)}
      <span role="cell" class="flex min-w-0 flex-col">
        {#if it}
          <span
            data-pob-color
            role="button"
            tabindex="0"
            class={['toned truncate font-semibold', toneClass(it.rarity)]}
            aria-describedby={hover?.key === `${i ? 'b' : 'a'}:${it.name}` ? 'compare-item-tip' : undefined}
            onmouseenter={(e) => itemtip && void showItem(i ? 'b' : 'a', it.name, e.currentTarget)}
            onfocus={(e) => itemtip && void showItem(i ? 'b' : 'a', it.name, e.currentTarget)}
            onmouseleave={() => (hover = undefined)}
            onblur={() => (hover = undefined)}>{it.name}</span>
          <span class="truncate text-text-3" title={it.line}>{it.line}</span>
          {#if jewel && (i ? jewel.bWarn : jewel.aWarn)}<span class="text-11"><PobText text={WARN} /></span>{/if}
        {:else}
          <span class="text-text-3">Empty</span>
        {/if}
      </span>
    {/each}
    <span role="cell" class="flex flex-col items-end font-mono max-phone:col-span-full max-phone:items-start">
      {#if same(r)}
        <span class="text-text-3">identical</span>
      {:else if !r.b}
        <span class="text-text-3">B leaves it empty</span>
      {:else if !r.impact}
        <span class="text-text-3">no what-if (flask or socket)</span>
      {:else}
        {#each lines(r).slice(0, 2) as l, i (i)}<span class={['toned max-w-full truncate', l.up ? 'tone-better' : 'tone-worse']} title={l.text}
            >{l.up ? '▲' : '▼'} {l.text}</span
          >{:else}<span class="text-text-3">no change</span>{/each}
      {/if}
      {@render buttons(r)}
    </span>
  </div>
{/snippet}

{#snippet expandedRow(r: Row, label: string, warn: boolean)}
  <div role="group" aria-label={label} class="border-b border-line py-2">
    <div class="flex items-center gap-2">
      <span class="text-13"><PobText text="^7{label}:{warn ? WARN : ''}" /></span>
      <span class="text-12"><PobText text={diffLabel(r)} /></span>
      <span class="grow"></span>
      {@render buttons(r)}
    </div>
    <div class="mt-1 grid grid-cols-2 gap-4 text-12">
      {#each shown[r.slot] ?? [] as card, i (i)}
        <ul class={['flex flex-col', i && 'border-l border-line pl-4']}>
          {#each card as line, k (k)}<li class={k ? '' : 'text-13'}><PobText text={line} /></li>{/each}
        </ul>
      {/each}
    </div>
  </div>
{/snippet}

<section aria-label="Item differences" class="panel text-12">
  <div class="px-3 py-1.5"><Checkbox compact bind:checked={expanded}>Expanded mode</Checkbox></div>
  {#if expanded}
    <div class="px-3">
      {#each items as r (r.slot)}{@render expandedRow(r, r.slot, false)}{/each}
      {#if jewels.length}
        <p class="pt-2 text-13"><PobText text="^7-- Jewels --" /></p>
        {#each jewels as j (j.slot)}{@render expandedRow(j, j.label, j.aWarn)}{/each}
      {/if}
    </div>
  {:else}
    <div role="table" aria-label="Items by slot">
      <div role="row" class={['data-row data-head gap-2.5 py-1.5 text-10 tracking-caps', GRID]}>
        <span role="columnheader" class="max-phone:hidden">SLOT</span><span role="columnheader" class="text-build-a">A</span><span
          role="columnheader"
          class="text-build-b">B</span
        ><span role="columnheader" class="text-right max-phone:col-span-full max-phone:text-left">EQUIPPING B’S ITEM IN A GIVES</span>
      </div>
      {#each items as r (r.slot)}{@render compactRow(r)}{/each}
      {#if jewels.length}
        <div role="rowgroup" aria-label="Jewels">
          <div role="row" class="data-row data-head py-1.5 text-10 tracking-caps"><span role="columnheader">JEWELS</span></div>
          {#each jewels as j (j.slot)}{@render compactRow(j, j)}{/each}
        </div>
      {/if}
    </div>
  {/if}
</section>
{#if hover?.tip}
  <FloatingItemTip
    anchor={hover.anchor}
    id="compare-item-tip"
    item={hover.name}
    base=""
    rarity="normal"
    mods={[]}
    impactTitle=""
    impact={[]}
    tooltip={hover.tip} />
{/if}
{#if equipHover && equipHover.row.impact}
  <div
    id="compare-equip-tip"
    role="tooltip"
    class="tooltip pointer-events-none fixed top-(--y) left-(--x) z-30 max-w-[600px] rounded-md px-3 py-1.5 shadow-tip"
    use:besideAnchor={equipHover.anchor}>
    <TooltipLines lines={equipLines(equipHover.row)} />
  </div>
{/if}
{#if socket?.view}
  <SocketPreview
    tree={socket.view.tree}
    node={socket.node}
    radii={socket.view.radii}
    alloc={socket.view.alloc}
    anchor={socket.anchor}
    label="Passive tree around {socket.label}" />
{/if}
