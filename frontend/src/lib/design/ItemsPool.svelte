<script lang="ts">
  import FloatingItemTip from './FloatingItemTip.svelte';
  import Select from './Select.svelte';
  import Button from './Button.svelte';
  import { tone as toneClass } from './tone';
  import { besideAnchor, type TipProps, type WhatIf } from './items';
  import { acceptItemDrag, readItemDrag, startItemDrag, type ItemDrag } from './item_drag';
  import { itemWikiName } from './wiki';

  // ItemListControl.lua:23-75
  // ItemListControl.lua:290-329
  type Row = { key: string; tip: TipProps; slot: string; col: string };
  type Props = {
    title: 'Stash' | 'Shared';
    rows: Row[];
    col: string;
    footer: string;
    empty: string;
    hover?: string | null;
    whatIf?: WhatIf;
    onedit: (key: string) => void;
    ondelete?: (key: string) => void;
    ondeleteunused?: () => void;
    ondeleteall?: () => void;
    onsort?: () => void;
    ontoggle?: (key: string, second: boolean) => void;
    onopen?: (key: string) => void;
    oncopy?: (key: string) => void;
    /* ItemListControl.lua:275-284 */
    ondropitem?: (drag: ItemDrag, before?: string) => void;
    dropAt?: boolean;
    /* ItemListControl.lua:77-111 */
    filters?: string[];
    filter?: string;
    onfilter?: (filter: string) => void;
  };
  let {
    title,
    rows,
    col,
    footer,
    empty,
    hover = $bindable(null),
    whatIf,
    onedit,
    ondelete,
    ondeleteunused,
    ondeleteall,
    onsort,
    ontoggle,
    onopen,
    oncopy,
    ondropitem,
    dropAt = false,
    filters = [],
    filter = '',
    onfilter
  }: Props = $props();
  let dropOver = $state(false);
  let dropBefore = $state<string | null>(null);
  let pending: ReturnType<typeof setTimeout> | undefined;
  const click = (e: MouseEvent, key: string) => {
    clearTimeout(pending);
    if ((e.ctrlKey || e.metaKey) && ontoggle) {
      ontoggle(key, e.shiftKey);
      return;
    }
    if (!onopen || e.detail === 0) onedit(key);
    else if (e.detail === 1) pending = setTimeout(() => onedit(key), 250);
  };
  const dblclick = (key: string) => {
    clearTimeout(pending);
    onopen?.(key);
  };
  const deletable = (key: string) => !!ondelete && /^\d+$/.test(key);

  const uid = $props.id();
  let q = $state('');
  let info = $state(false);
  let infoBtn = $state<HTMLButtonElement>();
  const els: Record<string, HTMLButtonElement> = $state({});
  const shown = $derived(rows.filter((r) => r.tip.item.toLowerCase().includes(q.trim().toLowerCase())));
  const hovered = $derived(shown.find((r) => r.key === hover));
  const hoveredTip = $derived(hovered && (whatIf?.key === hovered.key ? { ...hovered.tip, whatIf } : hovered.tip));
</script>

<div class="flex min-h-0 grow gap-3.5">
  <section
    aria-label="{title} items"
    class={['panel w-175 max-w-full self-start overflow-hidden text-12', dropOver && 'outline-1 outline-accent']}
    ondragover={(e) => {
      if (ondropitem && acceptItemDrag(e)) dropOver = true;
    }}
    ondragleave={() => {
      dropOver = false;
      dropBefore = null;
    }}
    ondrop={(e) => {
      dropOver = false;
      const before = dropBefore ?? undefined;
      dropBefore = null;
      const d = ondropitem && readItemDrag(e);
      if (d) ondropitem?.(d, before);
    }}>
    <div class="flex items-center gap-2 border-b border-line px-3 py-2.5">
      <span class="flex items-center gap-1.5 text-13 font-semibold whitespace-nowrap"
        >{title}
        {#if title === 'Shared'}
          <button
            bind:this={infoBtn}
            class="info-btn"
            aria-label="About shared items"
            aria-describedby={info ? `${uid}-info` : undefined}
            onmouseenter={() => (info = true)}
            onmouseleave={() => (info = false)}
            onfocus={() => (info = true)}
            onblur={() => (info = false)}>i</button>
        {/if}
      </span>
      <input class="input h-7.5 grow" type="search" placeholder="Search {title.toLowerCase()} items" aria-label="Search {title}" bind:value={q} />
      {#if filters.length && onfilter}
        <Select
          class="h-7.5 max-w-45 text-12"
          aria-label="Loadout filter"
          title="Show the items of a loadout, or those nothing uses"
          options={filters}
          value={filter || filters[0]}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onfilter(e.currentTarget.value)} />
      {/if}
      {#if onsort}
        <Button size="sm" variant="ghost" class="shrink-0 px-2 text-12" title="Sort the build's items by slot, equipped state, set and name" onclick={onsort}
          >Sort</Button>
      {:else}
        <Select class="h-7.5 text-12" aria-label="Sort" options={['Sort by build impact', 'Sort by slot', 'Sort by name']} />
      {/if}
      {#if ondeleteunused}
        <Button size="sm" variant="ghost" class="shrink-0 px-2 text-12" title="Delete every item no set or tree uses" onclick={ondeleteunused}
          >Del Unused</Button>
      {/if}
      {#if ondeleteall}
        <Button size="sm" variant="ghost" class="shrink-0 px-2 text-12" aria-haspopup="dialog" onclick={ondeleteall}>Del All…</Button>
      {/if}
    </div>
    <div role="table" aria-label={title}>
      <div role="row" class="data-row data-head grid-cols-24">
        <span role="columnheader" class="col-span-9">ITEM</span><span role="columnheader" class="col-span-7">BASE</span><span
          role="columnheader"
          class="col-span-4">SLOT</span
        ><span role="columnheader" class="col-span-4 text-right">{col}</span>
      </div>
      {#each shown as r, i (r.key)}
        <div
          role="row"
          tabindex="-1"
          class={[
            'data-row grid-cols-24',
            hover === r.key ? 'bg-surface-3' : i % 2 && 'bg-alt',
            dropBefore === r.key && 'shadow-[inset_0_2px_0_var(--color-accent)]'
          ]}
          ondragover={(e) => {
            if (dropAt && ondropitem && /^\d+$/.test(r.key) && acceptItemDrag(e)) dropBefore = r.key;
          }}
          onmouseenter={() => (hover = r.key)}
          onmouseleave={() => (hover = null)}>
          <span role="cell" class={[deletable(r.key) ? 'col-span-8' : 'col-span-9', 'min-w-0']}>
            <button
              bind:this={els[r.key]}
              data-pob-color
              data-wiki={itemWikiName(r.tip.rarity, r.tip.item, r.tip.base) || undefined}
              class={['cell-link max-w-full truncate', toneClass(r.tip.rarity)]}
              aria-haspopup="dialog"
              aria-describedby={hover === r.key ? `${uid}-tip` : undefined}
              onclick={(e) => click(e, r.key)}
              ondblclick={() => dblclick(r.key)}
              draggable={/^\d+$/.test(r.key) ? 'true' : undefined}
              ondragstart={(e) => /^\d+$/.test(r.key) && startItemDrag(e, title === 'Shared' ? { shared: Number(r.key) } : { id: Number(r.key) }, r.tip.item)}
              onkeydown={(e) => {
                if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'c' && oncopy) {
                  e.preventDefault();
                  oncopy(r.key);
                }
              }}
              onfocus={() => (hover = r.key)}
              onblur={() => (hover = null)}>{r.tip.item}</button>
          </span>
          <span role="cell" class="col-span-7 truncate text-text-2">{r.tip.base}</span>
          <span role="cell" class={['col-span-4 truncate', hover === r.key ? 'text-text-2' : 'text-text-3']}>{r.slot}</span>
          <span role="cell" class={['col-span-4 truncate text-right', hover === r.key ? 'text-text-2' : 'text-text-3']} title={r.col}>{r.col}</span>
          {#if deletable(r.key)}
            <span role="cell" class="col-span-1 text-right">
              <Button
                size="2xs"
                variant="ghost"
                class="px-1"
                aria-label="Delete {r.tip.item}"
                title={title === 'Shared' ? 'Remove this item from the shared item list' : 'Delete this item from the build'}
                onclick={() => ondelete?.(r.key)}>×</Button>
            </span>
          {/if}
        </div>
      {:else}
        <div role="row" class="px-3 py-3"><span role="cell" class="note">{rows.length ? `No ${title.toLowerCase()} item matches “${q}”.` : empty}</span></div>
      {/each}
    </div>
    <div class="note px-3 py-2 text-11">{footer}</div>
  </section>
</div>
{#if info}
  <div
    role="tooltip"
    id="{uid}-info"
    class="tooltip pointer-events-none fixed top-(--y) left-(--x) z-30 w-70 rounded-md px-2.5 py-2 text-12 shadow-pop"
    use:besideAnchor={infoBtn}>
    Shared items are a library kept across all your builds. Copy one into this build’s stash or equip it directly.
  </div>
{/if}
{#if hovered && hoveredTip}
  <FloatingItemTip anchor={els[hovered.key]} id="{uid}-tip" {...hoveredTip} />
{/if}
