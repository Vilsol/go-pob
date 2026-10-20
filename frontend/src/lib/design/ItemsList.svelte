<script lang="ts">
  import Checkbox from './Checkbox.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import SlotMenu from './SlotMenu.svelte';
  import { tone as toneClass, type Tone } from './tone';
  import { itemTip, type WhatIf } from './items';
  import type { ItemSummary, StatDiff } from './fixtures';

  type Equipped = { id: number; item: ItemSummary; impact?: StatDiff };
  type Option = { id: number; label: string; tone: Tone };
  type Props = {
    order: string[];
    slots: Record<string, Equipped | undefined>;
    flasks: Record<string, boolean>;
    calcFlasks: Record<string, boolean>;
    fitting: (slot: string) => Option[];
    jewels: { node: number; id: number; item?: ItemSummary }[];
    hover?: string | null;
    whatIf?: WhatIf;
    phoneLayout?: 'cards' | 'table';
    onedit: (slot: string) => void;
    onflask: (slot: string, on: boolean) => void;
    onequip: (slot: string, id: number | null) => void;
    ondelete?: (id: number) => void;
  };
  let {
    order,
    slots,
    flasks,
    calcFlasks,
    fitting,
    jewels,
    hover = $bindable(null),
    whatIf,
    phoneLayout = 'cards',
    onedit,
    onflask,
    onequip,
    ondelete
  }: Props = $props();

  const uid = $props.id();
  const els: Record<string, HTMLButtonElement> = $state({});
  const jewelKey = (i: number) => `Socket #${i + 1}`;
  const tip = $derived.by(() => {
    if (!hover) return undefined;
    const j = jewels.findIndex((_, i) => jewelKey(i) === hover);
    if (j >= 0) {
      const it = jewels[j].item;
      return it && itemTip(it, undefined, 'equipped');
    }
    const e = slots[hover];
    if (!e) return undefined;
    const t = itemTip(e.item, e.impact, 'equipped', e.item.flask ? flasks[hover] : undefined, calcFlasks[hover]);
    return whatIf?.key === hover ? { ...t, whatIf } : t;
  });
  const enter = (k: string) => () => (hover = k);
  const leave = () => (hover = null);
</script>

{#snippet row(key: string, i: number, e: { id?: number; item?: ItemSummary } | undefined, flask: boolean, menu: boolean)}
  <div
    role="row"
    tabindex="-1"
    class={['data-row box-border h-8 grid-cols-24 px-2.5 py-0', hover === key ? 'bg-surface-3' : i % 2 && 'bg-alt']}
    onmouseenter={e?.item ? enter(key) : undefined}
    onmouseleave={leave}>
    <span role="cell" class="col-span-2">
      {#if flask && e}<Checkbox
          class="size-3.5"
          checked={flasks[key]}
          aria-label="{key} active"
          onchange={(ev: Event & { currentTarget: HTMLInputElement }) => onflask(key, ev.currentTarget.checked)} />{/if}
    </span>
    <span role="cell" data-col="aside" class={['col-span-5', hover === key ? 'text-text-2' : 'text-text-3']}>{key}</span>
    <span role="cell" data-col="name" class="col-span-10 flex min-w-0 items-center gap-1">
      {#if e?.item}
        <button
          bind:this={els[key]}
          data-pob-color
          class={['cell-link truncate', toneClass(e.item.rarity)]}
          aria-haspopup="dialog"
          aria-describedby={hover === key && tip ? `${uid}-tip` : undefined}
          onclick={() => onedit(key)}
          onfocus={enter(key)}
          onblur={leave}>{e.item.name}</button>
      {:else}
        <span class={hover === key ? 'text-text-2' : 'text-text-3'}>Empty</span>
      {/if}
      {#if menu && (e?.item || fitting(key).length)}<span class="ml-auto"
          ><SlotMenu
            slot={key}
            options={fitting(key)}
            equipped={e?.id}
            onequip={(id: number | null) => onequip(key, id)}
            ondelete={e?.id && ondelete ? () => ondelete(e.id!) : undefined} /></span
        >{/if}
    </span>
    <span role="cell" class="col-span-4 truncate text-text-2">{e?.item?.base ?? ''}</span>
    <span
      role="cell"
      data-label={e?.item?.sockets ? 'Sockets' : undefined}
      class={['col-span-3 text-right font-mono whitespace-nowrap', hover === key ? 'text-text-2' : 'text-text-3']}>{e?.item?.sockets ?? ''}</span>
  </div>
{/snippet}

<div
  role="table"
  aria-label="Equipped items"
  data-phone={phoneLayout}
  class="phone-grid panel w-217 shrink-0 self-start overflow-hidden text-12 max-phone:relative max-phone:w-full max-phone:overflow-x-auto max-phone:[&>*]:w-217">
  <div role="row" data-phone-head class="data-row data-head grid-cols-24 px-2.5">
    <span role="columnheader" class="col-span-2">ON</span><span role="columnheader" class="col-span-5">SLOT</span><span role="columnheader" class="col-span-10"
      >ITEM</span
    ><span role="columnheader" class="col-span-4">BASE</span><span role="columnheader" class="col-span-3 text-right">SOCK.</span>
  </div>
  {#each order as slot, i (slot)}
    {@render row(slot, i, slots[slot], slot.startsWith('Flask'), true)}
  {/each}
  {#if jewels.length}
    <div role="rowgroup" aria-label="Tree jewels">
      <div role="row" class="border-b border-line bg-inset px-2.5 pt-2.5 pb-1 eyebrow text-text-2">
        <span role="columnheader">Tree jewels</span>
      </div>
      {#each jewels as j, i (j.node)}
        {@render row(jewelKey(i), i, j.item ? { id: j.id, item: j.item } : undefined, false, false)}
      {/each}
    </div>
  {/if}
</div>
{#if tip && hover}
  <FloatingItemTip anchor={els[hover]} id="{uid}-tip" {...tip} />
{/if}
