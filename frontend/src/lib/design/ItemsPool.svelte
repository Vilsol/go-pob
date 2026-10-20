<script lang="ts">
  import Tooltip from './Tooltip.svelte';
  import SearchField from './SearchField.svelte';
  import Icon from './Icon.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import Select from './Select.svelte';
  import Button from './Button.svelte';
  import Menu from './Menu.svelte';
  import { tone as toneClass } from './tone';
  import { type TipProps, type WhatIf } from './items';
  import { draggedItem, itemSource, type ItemDrag } from './item_drag';
  import { dragSource, dropZone } from './reorder';
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
    onclone?: (key: string) => void;
    onshare?: (key: string) => void;
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
    onclone,
    onshare,
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
  const rowEls: Record<string, HTMLElement> = $state({});
  const id = (key: string) => /^\d+$/.test(key);
  const rowActions = (r: Row) => [
    ...(ontoggle && id(r.key)
      ? [
          { label: 'Equip or unequip (Ctrl+click)', onselect: () => ontoggle(r.key, false) },
          ...(/ 1$/.test(r.slot)
            ? [{ label: `Equip or unequip in ${r.slot.replace(/1$/, '2')} (Ctrl+Shift+click)`, onselect: () => ontoggle(r.key, true) }]
            : [])
        ]
      : []),
    ...(onopen ? [{ label: 'Edit as text (double-click)', onselect: () => onopen(r.key) }] : []),
    ...(onclone && id(r.key) ? [{ label: 'Clone as new item', onselect: () => onclone(r.key) }] : []),
    ...(oncopy ? [{ label: 'Copy (Ctrl+C)', onselect: () => oncopy(r.key) }] : []),
    ...(onshare && id(r.key) ? [{ label: 'Share with all builds', onselect: () => onshare(r.key) }] : []),
    ...(deletable(r.key) ? [{ label: 'Delete (Del)', separator: true, onselect: () => ondelete?.(r.key) }] : [])
  ];
  const shown = $derived(rows.filter((r) => r.tip.item.toLowerCase().includes(q.trim().toLowerCase())));
  const step = (key: string, by: number) => {
    const i = shown.findIndex((r) => r.key === key);
    const to = shown[i + by];
    if (to) els[to.key]?.focus();
  };
  const hovered = $derived(shown.find((r) => r.key === hover));
  const hoveredTip = $derived(hovered && (whatIf?.key === hovered.key ? { ...hovered.tip, whatIf } : hovered.tip));
</script>

<div class="flex min-h-0 grow gap-3.5">
  <section
    aria-label="{title} items"
    class={['panel w-175 max-w-full self-start text-12', dropOver && 'outline-1 outline-accent']}
    use:dropZone={ondropitem && {
      accepts: (d) => !!draggedItem(d),
      over: () => (dropOver = true),
      leave: () => {
        dropOver = false;
        dropBefore = null;
      },
      drop: (d) => {
        dropOver = false;
        const before = dropBefore ?? undefined;
        dropBefore = null;
        const item = draggedItem(d);
        if (item) ondropitem?.(item, before);
      }
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
      <SearchField size="sm" class="grow" placeholder="Search {title.toLowerCase()} items" label="Search {title}" bind:value={q} />
      {#if filters.length && onfilter}
        <Select
          size="sm"
          class="max-w-45 text-12"
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
        <Select size="sm" class="text-12" aria-label="Sort" options={['Sort by build impact', 'Sort by slot', 'Sort by name']} />
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
        <span role="columnheader" class="col-span-7">ITEM</span><span role="columnheader" class="col-span-7">BASE</span><span
          role="columnheader"
          class="col-span-4">SLOT</span
        ><span role="columnheader" class="col-span-4 text-right">{col}</span><span role="columnheader" class="col-span-2 sr-only">Actions</span>
      </div>
      {#each shown as r, i (r.key)}
        {@const acts = rowActions(r)}
        <div
          bind:this={rowEls[r.key]}
          role="row"
          tabindex="-1"
          class={[
            'data-row grid-cols-24',
            hover === r.key ? 'bg-surface-3' : i % 2 && 'bg-alt',
            dropBefore === r.key && 'shadow-[inset_0_2px_0_var(--color-accent)]'
          ]}
          use:dropZone={dropAt && ondropitem && /^\d+$/.test(r.key) && { accepts: (d) => !!draggedItem(d), over: () => (dropBefore = r.key) }}
          onmouseenter={() => (hover = r.key)}
          onmouseleave={() => (hover = null)}>
          <span role="cell" class="col-span-7 min-w-0">
            <button
              bind:this={els[r.key]}
              data-pob-color
              data-wiki={itemWikiName(r.tip.rarity, r.tip.item, r.tip.base) || undefined}
              class={['cell-link max-w-full truncate', toneClass(r.tip.rarity)]}
              aria-haspopup="dialog"
              aria-describedby={hover === r.key ? `${uid}-tip` : undefined}
              onclick={(e) => click(e, r.key)}
              ondblclick={() => dblclick(r.key)}
              use:dragSource={/^\d+$/.test(r.key) && itemSource(title === 'Shared' ? { shared: Number(r.key) } : { id: Number(r.key) }, r.tip.item)}
              onkeydown={(e) => {
                if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'c' && oncopy) {
                  e.preventDefault();
                  oncopy(r.key);
                } else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
                  e.preventDefault();
                  step(r.key, e.key === 'ArrowDown' ? 1 : -1);
                } else if (e.key === 'Delete' && deletable(r.key)) {
                  e.preventDefault();
                  const at = shown.findIndex((x) => x.key === r.key);
                  (els[shown[at + 1]?.key] ?? els[shown[at - 1]?.key])?.focus();
                  ondelete?.(r.key);
                }
              }}
              onfocus={() => (hover = r.key)}
              onblur={() => (hover = null)}>{r.tip.item}</button>
          </span>
          <span role="cell" class="col-span-7 truncate text-text-2">{r.tip.base}</span>
          <span role="cell" class={['col-span-4 truncate', hover === r.key ? 'text-text-2' : 'text-text-3']}>{r.slot}</span>
          <span role="cell" class={['col-span-4 truncate text-right', hover === r.key ? 'text-text-2' : 'text-text-3']} title={r.col}>{r.col}</span>
          <span role="cell" class="col-span-2 flex justify-end gap-0.5">
            {#if acts.length}
              <Menu label={r.tip.item} items={acts} class="top-full right-0 z-dock mt-1 w-max max-w-[calc(100vw-16px)]" itemClass="whitespace-nowrap">
                {#snippet trigger(a)}
                  <Button
                    {...a}
                    size="2xs"
                    variant="ghost"
                    class="px-1 font-mono"
                    aria-label="Actions for {r.tip.item}"
                    title="Equip, edit, clone, copy, share or delete"><Icon name="more" size={14} /></Button>
                {/snippet}
              </Menu>
            {/if}
            {#if deletable(r.key)}
              <Button
                size="2xs"
                variant="ghost"
                class="px-1"
                aria-label="Delete {r.tip.item}"
                title={title === 'Shared' ? 'Remove this item from the shared item list' : 'Delete this item from the build'}
                onclick={() => ondelete?.(r.key)}><Icon name="close" size={12} /></Button>
            {/if}
          </span>
        </div>
      {:else}
        <div role="row" class="px-3 py-3"><span role="cell" class="note">{rows.length ? `No ${title.toLowerCase()} item matches “${q}”.` : empty}</span></div>
      {/each}
    </div>
    <div class="note px-3 py-2 text-11">{footer}</div>
  </section>
</div>
{#if info}
  <Tooltip id="{uid}-info" placement="beside" anchor={infoBtn} class="w-70">
    Shared items are a library kept across all your builds. Copy one into this build’s stash or equip it directly.
  </Tooltip>
{/if}
{#if hovered && hoveredTip}
  <FloatingItemTip anchor={rowEls[hovered.key]} id="{uid}-tip" {...hoveredTip} />
{/if}
