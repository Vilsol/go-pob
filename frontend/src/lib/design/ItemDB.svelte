<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import Tabs from './Tabs.svelte';
  import { panelId, tabId } from './tab_ids';
  import Select from './Select.svelte';
  import Button from './Button.svelte';
  import { tone as toneClass, type Tone } from './tone';
  import type { TipProps } from './items';
  import { startItemDrag } from './item_drag';
  import { itemWikiName } from './wiki';
  import type { ItemDBFilterInput, ItemDBKind, ItemDBLists, ItemDBRowPlain } from './item_db';

  // ItemsTab.lua:273-296
  type Props = {
    kind: ItemDBKind;
    lists?: ItemDBLists;
    filter: ItemDBFilterInput;
    sort: string;
    rows: ItemDBRowPlain[];
    busy?: boolean;
    hover?: string | null;
    tip?: TipProps;
    note?: string;
    onadd: (name: string, equip: boolean, second: boolean) => void;
    onopen: (name: string) => void;
    oncopy: (name: string) => void;
    oncraft?: (base: string) => void;
  };
  let {
    kind = $bindable(),
    lists,
    filter = $bindable(),
    sort = $bindable(),
    rows,
    busy = false,
    hover = $bindable(null),
    tip,
    note = '',
    onadd,
    onopen,
    oncopy,
    oncraft
  }: Props = $props();

  const uid = $props.id();
  const els: Record<string, HTMLButtonElement> = $state({});
  const unique = $derived(kind === 'UNIQUE');
  const opts = (list: string[] | undefined) => (list ?? []).map((label, i) => ({ value: String(i + 1), label }));
  const pick = (key: keyof Omit<ItemDBFilterInput, 'search'>) => (e: Event & { currentTarget: HTMLSelectElement }) =>
    (filter = { ...filter, [key]: Number(e.currentTarget.value) });
  const sortOptions = $derived([
    { value: '', label: 'Sort by Name' },
    ...(lists?.sortStats ?? []).map((s) => ({ value: s.stat, label: `Sort by ${s.label}` }))
  ]);
  const fmt = (v: number) => (Number.isFinite(v) ? (Math.abs(v) >= 100 ? Math.round(v).toLocaleString('en-US') : String(Math.round(v * 100) / 100)) : '—');
  const hovered = $derived(rows.find((r) => r.name === hover));
</script>

<div class="flex min-h-0 grow flex-col gap-2.5 max-phone:shrink-0">
  <Tabs
    id={uid}
    variant="chip"
    label="Item database"
    tabs={[
      { id: 'UNIQUE', label: 'Uniques' },
      { id: 'RARE', label: 'Templates' }
    ]}
    selected={kind}
    onchange={(id) => {
      kind = id as ItemDBKind;
      hover = null;
    }} />
  <div
    role="tabpanel"
    id={panelId(uid)}
    aria-labelledby={tabId(uid, kind)}
    class="flex min-h-0 grow flex-wrap gap-3.5 max-phone:shrink-0 max-phone:flex-col max-phone:flex-nowrap">
    <aside aria-label="Filters" class="panel flex w-62.5 shrink-0 flex-col gap-2.5 self-start p-3 text-12 max-phone:w-auto max-phone:self-stretch">
      <input
        class="input"
        type="search"
        placeholder="Search"
        aria-label="Search {unique ? 'uniques' : 'templates'}"
        value={filter.search}
        oninput={(e) => (filter = { ...filter, search: e.currentTarget.value })} />
      <FloatingLabel label="Search in"
        ><Select
          class="h-7.5 text-12"
          aria-label="Search in"
          options={opts(lists?.searchModes)}
          value={String(filter.searchMode)}
          onchange={pick('searchMode')} /></FloatingLabel>
      <FloatingLabel label="Slot"
        ><Select class="h-7.5 text-12" aria-label="Slot" options={opts(lists?.slots)} value={String(filter.slot)} onchange={pick('slot')} /></FloatingLabel>
      <FloatingLabel label="Type"
        ><Select class="h-7.5 text-12" aria-label="Type" options={opts(lists?.types)} value={String(filter.type)} onchange={pick('type')} /></FloatingLabel>
      {#if unique}
        <FloatingLabel label="Sort"><Select class="h-7.5 text-12" aria-label="Sort" options={sortOptions} bind:value={sort} /></FloatingLabel>
        <FloatingLabel label="League"
          ><Select
            class="h-7.5 text-12"
            aria-label="League"
            options={opts(lists?.leagues)}
            value={String(filter.league)}
            onchange={pick('league')} /></FloatingLabel>
        <FloatingLabel label="Requirements"
          ><Select
            class="h-7.5 text-12"
            aria-label="Requirements"
            options={opts(lists?.requirements)}
            value={String(filter.requirement)}
            onchange={pick('requirement')} /></FloatingLabel>
        <FloatingLabel label="Source"
          ><Select
            class="h-7.5 text-12"
            aria-label="Source"
            options={opts(lists?.obtainable)}
            value={String(filter.obtainable)}
            onchange={pick('obtainable')} /></FloatingLabel>
      {/if}
    </aside>
    <section
      aria-label={unique ? 'Uniques' : 'Templates'}
      aria-busy={busy}
      class="panel relative min-w-0 grow self-start overflow-hidden text-12 max-phone:shrink-0 max-phone:self-stretch">
      <div role="table" aria-label="{unique ? 'Unique' : 'Template'} list" class="max-h-[70vh] overflow-y-auto max-phone:max-h-none">
        <div role="row" class="data-row data-head sticky top-0 z-1 grid-cols-24 py-2 max-phone:hidden">
          <span role="columnheader" class="col-span-9">ITEM</span><span role="columnheader" class="col-span-5">TYPE</span><span
            role="columnheader"
            class="col-span-3 text-right">{sort ? 'STAT' : 'REQ.'}</span
          ><span role="columnheader" class="col-span-7"><span class="sr-only">Actions</span></span>
        </div>
        {#each rows as r, i (i)}
          <div
            role="row"
            tabindex="-1"
            class={[
              'data-row grid-cols-24 py-1 max-phone:grid-cols-[minmax(0,1fr)_auto] max-phone:gap-y-0.5',
              hover === r.name ? 'bg-surface-3' : i % 2 && 'bg-alt'
            ]}
            onmouseenter={() => (hover = r.name)}
            onmouseleave={() => (hover = null)}>
            <span role="cell" class="col-span-9 min-w-0 max-phone:col-span-1 max-phone:col-start-1 max-phone:row-start-1">
              <button
                bind:this={els[r.name]}
                data-pob-color
                data-wiki={itemWikiName(r.rarity, r.title || r.name, r.base)}
                class={['cell-link max-w-full truncate font-semibold', toneClass(r.rarity.toLowerCase() as Tone)]}
                title="Ctrl+click: add and equip (Shift: the second slot) · double-click: open in the paste panel · Ctrl+C: copy · drag onto a slot or the stash"
                aria-describedby={hover === r.name ? `${uid}-tip` : undefined}
                onclick={(e) => {
                  if (e.ctrlKey || e.metaKey) onadd(r.name, true, e.shiftKey);
                }}
                ondblclick={() => onopen(r.name)}
                draggable="true"
                ondragstart={(e) => startItemDrag(e, { db: kind, name: r.name }, r.name)}
                onkeydown={(e) => {
                  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'c') {
                    e.preventDefault();
                    oncopy(r.name);
                  }
                }}
                onfocus={() => (hover = r.name)}
                onblur={() => (hover = null)}>{unique ? r.title || r.name : r.name}</button>
            </span>
            <span role="cell" class="col-span-5 truncate text-text-2 max-phone:col-span-1 max-phone:col-start-1 max-phone:row-start-2" title={r.base}
              >{r.type}</span>
            <span role="cell" class="col-span-3 text-right font-mono text-text-2 max-phone:col-span-1 max-phone:col-start-2 max-phone:row-start-2"
              >{sort ? fmt(r.power) : r.level || ''}</span>
            <span role="cell" class="col-span-7 flex justify-end gap-1 max-phone:col-span-1 max-phone:col-start-2 max-phone:row-start-1">
              {#if oncraft && r.base}<Button
                  size="xs"
                  variant="ghost"
                  title="Craft a new item on this base"
                  aria-label="Craft from {r.base}"
                  onclick={() => oncraft(r.base)}>Craft</Button
                >{/if}
              <Button size="xs" variant="ghost" title="Add to this build's items" aria-label="Add {r.name}" onclick={() => onadd(r.name, false, false)}
                >Add</Button>
              <Button
                size="xs"
                variant="primary"
                title="Add and equip in its slot (Shift-click: the second slot)"
                aria-label="Equip {r.name}"
                onclick={(e: MouseEvent) => onadd(r.name, true, e.shiftKey)}>Equip</Button>
            </span>
          </div>
        {:else}
          <div role="row" class="px-3 py-3">
            <span role="cell" class="note">{busy ? 'Loading…' : 'No items found that match those filters.'}</span>
          </div>
        {/each}
      </div>
      <div class="note flex justify-between px-3 py-1.5 text-11">
        <span>{busy ? (sort ? 'Sorting…' : 'Loading…') : `${rows.length} items`}</span>
        <span role="status" class={note ? 'text-better' : 'sr-only'}>{note}</span>
      </div>
    </section>
  </div>
</div>
{#if hovered && tip}
  <FloatingItemTip anchor={els[hovered.name]} id="{uid}-tip" {...tip} />
{/if}
