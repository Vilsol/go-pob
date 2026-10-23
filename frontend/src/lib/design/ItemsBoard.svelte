<script lang="ts">
  import Checkbox from './Checkbox.svelte';
  import EmptyState from './EmptyState.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import SlotMenu from './SlotMenu.svelte';
  import Dropdown from './Dropdown.svelte';
  import { tone as toneClass, type Tone } from './tone';
  import { BOARD_SLOTS, besideAnchor, itemTip, type TipProps, type WhatIf } from './items';
  import { draggedItem, type ItemDrag } from './item_drag';
  import { dropZone, type DropZone } from './reorder';
  import SocketPreview from './SocketPreview.svelte';
  import ItemIcon from './ItemIcon.svelte';
  import ItemSockets from './ItemSockets.svelte';
  import { artBox, placeGems, type SocketedGem, type SocketedGroup } from './item_sockets';
  import TooltipLines from './TooltipLines.svelte';
  import type { Tree } from '$lib/skill_tree/types';
  import type { ItemSummary, ItemTip, StatDiff } from './fixtures';
  import Tabs from './Tabs.svelte';
  import { panelId, tabId } from './tab_ids';
  import { BOARD_PANES, phoneMedia } from './mobile/mobile';

  type Equipped = { id: number; item: ItemSummary; impact?: StatDiff };
  type Option = { id: number; label: string; tone: Tone };
  type Props = {
    slots: Record<string, Equipped | undefined>;
    flasks: Record<string, boolean>;
    calcFlasks: Record<string, boolean>;
    fitting: (slot: string) => Option[];
    jewels: { node: number; id: number; item?: ItemSummary; choices?: number[] }[];
    /* ItemSlotControl.lua:98-111 */
    abyssal?: string[];
    /* ItemsTab.lua:166-174 */
    extra?: string[];
    /* ItemSlotControl.lua:48-56 */
    tipFor?: (id: number, slot: string) => TipProps | undefined;
    onhoveritem?: (id: number | null, slot: string) => void;
    socketTip?: { node: number; tip: ItemTip };
    jewelOptions: Option[];
    hover?: string | null;
    whatIf?: WhatIf;
    onedit: (slot: string) => void;
    /* an empty slot opens Craft Item for it */
    oncraft?: (slot: string) => void;
    onflask: (slot: string, on: boolean) => void;
    onequip: (slot: string, id: number | null) => void;
    onjewel?: (node: number, id: number | null) => void;
    /* ItemListControl.lua:331-369 */
    ondelete?: (id: number) => void;
    /* ItemsTab.lua:147-162 */
    swap?: boolean;
    /* ItemSlotControl.lua:114-130 */
    ondropitem?: (slot: string, drag: ItemDrag) => void;
    /* ItemSlotControl.lua:132-149 */
    socketView?: { node: number; tree: Tree; radii: { inner: number; outer: number; col: string; nodes: number[] }[]; alloc: Set<number> };
    /* the skill groups socketed in each slot, in the skill list's order */
    socketed?: Record<string, SocketedGroup[]>;
    /* GemSelectControl.lua:629-646 */
    gemtip?: (group: number, gem: number) => Promise<{ Text: string; Separator: boolean }[] | undefined>;
  };
  let {
    slots,
    flasks,
    calcFlasks,
    fitting,
    jewels,
    jewelOptions,
    hover = $bindable(null),
    whatIf,
    onedit,
    oncraft,
    onflask,
    onequip,
    onjewel,
    ondelete,
    abyssal = [],
    extra = [],
    tipFor,
    onhoveritem,
    socketTip,
    swap = false,
    ondropitem,
    socketView,
    socketed = {},
    gemtip
  }: Props = $props();
  let dropOver = $state<string | null>(null);
  const dropTarget = (slot: string): DropZone | undefined =>
    ondropitem && {
      accepts: (d) => !!draggedItem(d),
      over: () => (dropOver = slot),
      leave: () => (dropOver = null),
      drop: (d) => {
        dropOver = null;
        const item = draggedItem(d);
        if (item) ondropitem(slot, item);
      }
    };
  const abyssalLabel = (slot: string) => slot.replace(/ Swap/, '').replace(/ Abyssal Socket (\d+)$/, ' Abyssal #$1');
  const key = (slot: string) => (swap && slot.startsWith('Weapon') ? `${slot} Swap` : slot);
  // a touch has no hover: the first tap on an item shows its name, sockets and tooltip, the second opens it
  let tap: { touch: boolean; shown: boolean } = { touch: false, shown: false };

  let gemHover = $state<{ slot: string; gem: SocketedGem; el: HTMLElement; lines?: { Text: string; Separator: boolean }[] } | null>(null);
  // a pointer that leaves while the overlay redraws can miss mouseleave; the tip only shows for the slot still hovered
  const gemShown = $derived(gemHover && gemHover.slot === hover ? gemHover : null);
  const hoverGem = (slot: string, gem: SocketedGem | null, el: HTMLElement | null) => {
    if (!gem || !el) {
      gemHover = null;
      return;
    }
    const at = { slot, gem, el };
    gemHover = at;
    void gemtip?.(gem.group, gem.gem).then((lines) => {
      if (gemHover?.el === el) gemHover = { ...at, lines };
    });
  };

  const uid = $props.id();
  const els: Record<string, HTMLElement> = $state({});
  const overlays: Record<string, { focusGem: (last?: boolean) => void } | undefined> = $state({});
  // the pointer or focus moving between a slot and its socket overlay keeps the slot shown
  const inSlot = (k: string, to: EventTarget | null) =>
    to instanceof Element && (to === els[k] || (!!els[k]?.parentElement?.contains(to) && !!to.closest('[data-item-sockets]')));
  const leaveSlot = (k: string, to: EventTarget | null) => {
    if (inSlot(k, to)) return;
    if (hover === k) hover = null;
    gemHover = null;
  };
  const tip = $derived.by(() => {
    const jewel = hover ? /^Jewel (\d+)$/.exec(hover) : null;
    if (jewel) {
      const j = jewels.find((x) => x.node === Number(jewel[1]));
      if (!j?.item) return undefined;
      return itemTip(socketTip?.node === j.node ? { ...j.item, tooltip: socketTip.tip } : j.item, undefined, 'equipped');
    }
    const e = hover ? slots[hover] : undefined;
    if (!e || !hover) return undefined;
    const t = itemTip(e.item, e.impact, 'equipped', e.item.flask ? flasks[hover] : undefined, calcFlasks[hover]);
    return whatIf?.key === hover ? { ...t, whatIf } : t;
  });
  // phones: the board's slots in sideways panes at the board's own scale, with a section strip above
  const phone = phoneMedia();
  const paneUid = `${uid}-panes`;
  let pane = $state<string>(BOARD_PANES[0].id);
  let strip = $state<HTMLElement>();
  const showPane = (id: string) => {
    pane = id;
    const i = BOARD_PANES.findIndex((x) => x.id === id);
    strip?.scrollTo({ left: i * strip.clientWidth, behavior: 'smooth' });
  };
  const bySlot = new Map(BOARD_SLOTS.map((b) => [b.slot, b]));

  // ItemSlotControl.lua:48-56: an entry under the pointer or the arrow keys shows its tooltip beside the open list
  let jewelPick = $state<{ id: number; slot: string; el: HTMLElement } | null>(null);
  const pickTip = $derived(jewelPick ? tipFor?.(jewelPick.id, jewelPick.slot) : undefined);
  const pickJewel = (node: number, value: string | undefined, el: HTMLElement | null) => {
    const slot = `Jewel ${node}`;
    const id = Number(value);
    if (!el || !id) {
      jewelPick = null;
      hover = els[slot]?.contains(document.activeElement) ? slot : null;
      return;
    }
    jewelPick = { id, slot, el };
    onhoveritem?.(id, slot);
  };
  const jewelChoices = (choices?: number[]) => [
    { value: '0', label: 'None' },
    ...jewelOptions.filter((o) => !choices || choices.includes(o.id)).map((o) => ({ value: String(o.id), label: o.label }))
  ];
</script>

{#snippet board(b: (typeof BOARD_SLOTS)[number], placed: boolean)}
  {@const k = key(b.slot)}
  {@const e = slots[k]}
  {@const flask = b.slot.startsWith('Flask')}
  <div
    role="group"
    aria-label="{b.slot} slot"
    class={[placed ? 'absolute top-(--y) left-(--x)' : 'relative shrink-0', 'h-(--h) w-(--w)', !placed && flask && 'mb-5.5']}
    style="--x: {b.x / 16}rem; --y: {b.y / 16}rem; --w: {b.w / 16}rem; --h: {b.h / 16}rem"
    use:dropZone={dropTarget(k)}>
    <button
      bind:this={els[k]}
      class={[
        'relative box-border flex size-full flex-col items-start justify-between overflow-hidden rounded border border-(--tone)/60 bg-tile p-1.5 pr-6 text-left font-sans',
        toneClass(e ? e.item.rarity : 'muted'),
        (hover === k || dropOver === k) && 'border-text',
        flask && e && !flasks[k] && 'border-dashed'
      ]}
      aria-label="{b.slot}: {e ? e.item.name : 'Empty'}"
      aria-haspopup={e || oncraft ? 'dialog' : undefined}
      title={e || !oncraft ? undefined : `Craft an item for ${b.slot}`}
      aria-describedby={hover === k && tip ? `${uid}-tip` : undefined}
      data-off={flask && e && !flasks[k] ? 'true' : undefined}
      onpointerdown={(ev: PointerEvent) => (tap = { touch: ev.pointerType === 'touch', shown: hover === k })}
      onclick={() => {
        const first = tap.touch && !tap.shown;
        tap = { touch: false, shown: false };
        if (e && first) hover = k;
        else if (e) onedit(k);
        else oncraft?.(k);
      }}
      onmouseenter={() => (hover = k)}
      onmouseleave={(ev: MouseEvent) => leaveSlot(k, ev.relatedTarget)}
      onfocus={() => (hover = k)}
      onblur={(ev: FocusEvent) => leaveSlot(k, ev.relatedTarget)}
      onkeydown={(ev: KeyboardEvent) => {
        if (!overlays[k] || !['ArrowRight', 'ArrowDown', 'ArrowLeft', 'ArrowUp'].includes(ev.key)) return;
        ev.preventDefault();
        overlays[k].focusGem(ev.key === 'ArrowLeft' || ev.key === 'ArrowUp');
      }}>
      {#if e}
        {@const art = artBox(e.item.cells, b.w - 24, b.h - 24)}
        <span class="pointer-events-none absolute inset-0 flex items-center justify-center">
          <span class="relative flex">
            <ItemIcon
              name={e.item.name}
              base={e.item.base}
              rarity={e.item.rarity}
              size={art.w}
              height={art.h}
              unit="rem"
              decorative
              class={[flask && !flasks[k] && 'opacity-50']} />
          </span>
        </span>
      {/if}
      <span class="eyebrow eyebrow-xs relative font-normal text-text-3">{flask ? '' : b.slot}</span>
      {#if !e || hover === k}
        <span data-slot-caption class="relative flex flex-col items-start">
          <span
            data-pob-color={e && (!flask || flasks[k]) ? '' : undefined}
            class={['rounded-tight bg-tile/80 text-12 leading-[0.9375rem] font-semibold', e && (!flask || flasks[k]) ? 'text-(--tone)' : 'text-text-3']}
            >{e ? e.item.name : 'Empty'}</span>
          {#if e?.item.sockets}<span class="note-sm rounded-tight bg-tile/80 font-mono before:content-[attr(data-sockets)]" data-sockets={e.item.sockets}></span
            >{/if}
        </span>
      {/if}
    </button>
    {#if e && hover === k && e.item.sockets}
      {@const art = artBox(e.item.cells, b.w - 24, b.h - 24)}
      <span class="pointer-events-none absolute inset-0 flex items-center justify-center">
        <span class="relative" style:width="{art.w / 16}rem" style:height="{art.h / 16}rem"
          ><ItemSockets
            bind:this={overlays[k]}
            line={e.item.sockets}
            w={art.w}
            h={art.h}
            cell={art.cell}
            cols={art.cols}
            gems={placeGems(e.item.sockets, socketed[k] ?? [])}
            shown={gemShown?.slot === k ? gemShown.gem : undefined}
            tipId="{uid}-gem-tip"
            ongem={(g: SocketedGem | null, el: HTMLElement | null) => hoverGem(k, g, el)}
            onleave={(to: EventTarget | null) => leaveSlot(k, to)}
            onescape={() => els[k]?.focus()}
            onopen={() => onedit(k)} /></span>
      </span>
    {/if}
    {#if e || fitting(k).length}
      <div class="absolute top-1 right-1">
        <SlotMenu
          slot={b.slot}
          options={fitting(k)}
          equipped={e?.id}
          onequip={(id: number | null) => onequip(k, id)}
          ondelete={e && ondelete ? () => ondelete(e.id) : undefined}
          tipFor={tipFor && ((id: number) => tipFor(id, k))}
          onhoveritem={onhoveritem && ((id: number | null) => onhoveritem(id, k))} />
      </div>
    {/if}
    {#if flask && e}
      <label class="absolute -bottom-5.5 left-1 flex items-center gap-0.75 text-10 text-text-2" title="Flask active in calculations">
        <Checkbox
          class="size-3.5"
          checked={flasks[k]}
          aria-label="{k} active"
          onchange={(ev: Event & { currentTarget: HTMLInputElement }) => onflask(k, ev.currentTarget.checked)} />
        on
      </label>
    {/if}
  </div>
{/snippet}

{#snippet treeJewels(cols: string)}
  {#if jewels.length}
    <div class={['grid gap-2', cols]}>
      {#each jewels as j, i (j.node)}
        {@const choices = jewelChoices(j.choices)}
        <div
          role="group"
          aria-label="Socket #{i + 1}"
          data-socket-node={j.node}
          bind:this={els[`Jewel ${j.node}`]}
          aria-describedby={hover === `Jewel ${j.node}` && tip ? `${uid}-tip` : undefined}
          onmouseenter={() => (hover = `Jewel ${j.node}`)}
          onmouseleave={() => (hover = null)}
          onfocusin={() => (hover = `Jewel ${j.node}`)}
          onfocusout={() => (hover = null)}>
          <div class="flex items-center gap-1.5">
            {#if j.item}<ItemIcon name={j.item.name} base={j.item.base} rarity={j.item.rarity} size={30} decorative />{/if}
            <FloatingLabel label="Socket #{i + 1}" class="min-w-0 grow [--float-bg:var(--color-surface-1)]"
              ><Dropdown
                aria-label="Socket #{i + 1}"
                title={onjewel ? `Passive node ${j.node}` : `Passive node ${j.node} · socketed on the tree (changing it here is not in the app yet)`}
                class="block h-7.5 w-full {toneClass(j.item?.rarity)} [&>button]:bg-surface-1 [&>button]:px-1.5"
                options={choices}
                value={String(j.id)}
                disabled={!onjewel}
                previewOnTap
                onhover={(k: number | null, el: HTMLElement | null) => pickJewel(j.node, k === null ? undefined : choices[k]?.value, el)}
                onchange={(v: string) => onjewel?.(j.node, Number(v) || null)} /></FloatingLabel>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <EmptyState text="No jewel sockets allocated on the tree." />
  {/if}
  {#if socketView && (hover === `Jewel ${socketView.node}` || hover?.endsWith(`@Jewel ${socketView.node}`))}
    {@const n = jewels.findIndex((j) => j.node === socketView.node)}
    <SocketPreview
      tree={socketView.tree}
      node={socketView.node}
      radii={socketView.radii}
      alloc={socketView.alloc}
      anchor={els[`Jewel ${socketView.node}`]}
      label="Passive tree around socket #{n + 1}" />
  {/if}
{/snippet}

<div class="flex shrink-0 flex-col gap-3 max-phone:w-full max-phone:min-w-0">
  {#if phone.current}
    <Tabs id={paneUid} variant="segmented" fill label="Board sections" tabs={[...BOARD_PANES]} selected={pane} onchange={showPane} />
    <div
      bind:this={strip}
      role="tabpanel"
      id={panelId(paneUid)}
      aria-labelledby={tabId(paneUid, pane)}
      class="flex snap-x snap-mandatory overflow-x-auto overflow-y-hidden rounded-md border border-line bg-well [scrollbar-width:none]"
      onscroll={() => {
        if (strip) pane = BOARD_PANES[Math.round(strip.scrollLeft / strip.clientWidth)]?.id ?? pane;
      }}>
      {#each BOARD_PANES as p (p.id)}
        <div role="group" aria-label={p.label} class="box-border flex w-full shrink-0 snap-start flex-wrap content-start items-start gap-2 p-2.5">
          {#if p.id === 'jewels'}
            <div class="w-full">{@render treeJewels('grid-cols-2')}</div>
          {:else}
            {#each p.slots as slot (slot)}{@render board(bySlot.get(slot)!, false)}{/each}
          {/if}
        </div>
      {/each}
    </div>
  {:else}
    <section aria-label="Equipped items" class="relative h-185.5 w-217 rounded-md border border-line bg-well">
      {#each BOARD_SLOTS as b (b.slot)}{@render board(b, true)}{/each}
    </section>
  {/if}
  {#snippet smallSlot(slot: string, label: string)}
    {@const e = slots[slot]}
    <div
      role="group"
      aria-label="{label} slot"
      class={[
        'flex h-7.5 items-center gap-1 rounded border bg-surface-1 px-1.5 text-12',
        dropOver === slot ? 'border-text' : 'border-line',
        toneClass(e?.item.rarity ?? 'muted')
      ]}
      use:dropZone={dropTarget(slot)}>
      <button
        bind:this={els[slot]}
        class="min-w-0 grow truncate border-0 bg-transparent p-0 text-left font-sans text-12"
        aria-label="{label}: {e ? e.item.name : 'Empty'}"
        aria-haspopup={!e && oncraft ? 'dialog' : undefined}
        title={e || !oncraft ? undefined : `Craft an item for ${label}`}
        onclick={() => (e ? onedit(slot) : oncraft?.(slot))}
        onmouseenter={() => (hover = slot)}
        onmouseleave={() => (hover = null)}
        onfocus={() => (hover = slot)}
        onblur={() => (hover = null)}
        ><span class="text-text-3">{label}</span>
        <span data-pob-color class={e ? 'text-(--tone)' : 'text-text-3'}>{e ? e.item.name : 'Empty'}</span></button>
      {#if e || fitting(slot).length}
        <SlotMenu
          slot={label}
          options={fitting(slot)}
          equipped={e?.id}
          onequip={(id: number | null) => onequip(slot, id)}
          ondelete={e && ondelete ? () => ondelete(e.id) : undefined}
          tipFor={tipFor && ((id: number) => tipFor(id, slot))}
          onhoveritem={onhoveritem && ((id: number | null) => onhoveritem(id, slot))} />
      {/if}
    </div>
  {/snippet}
  {#if extra.length}
    <section aria-label="Other slots" class="flex w-217 flex-col gap-1.5 max-phone:w-full">
      <h2 class="caps">Other slots</h2>
      <div class="grid grid-cols-4 gap-2 max-phone:grid-cols-2">
        {#each extra as slot (slot)}{@render smallSlot(slot, slot)}{/each}
      </div>
    </section>
  {/if}
  {#if abyssal.length}
    <section aria-label="Abyssal sockets" class="flex w-217 flex-col gap-1.5 max-phone:w-full">
      <h2 class="caps">Abyssal sockets</h2>
      <div class="grid grid-cols-4 gap-2 max-phone:grid-cols-2">
        {#each abyssal as slot (slot)}{@render smallSlot(slot, abyssalLabel(slot))}{/each}
      </div>
    </section>
  {/if}
  {#if !phone.current}
    <section aria-label="Tree jewels" class="flex w-217 flex-col gap-1.5">
      <h2 class="caps">Tree jewels</h2>
      {@render treeJewels('grid-cols-4')}
    </section>
  {/if}
</div>
{#if gemShown}
  {#if gemShown.lines?.length}
    <div id="{uid}-gem-tip" data-floating class="pointer-events-none fixed top-(--y) left-(--x) z-float w-max max-w-[480px]" use:besideAnchor={gemShown.el}>
      <div class="tooltip rounded-md px-3 py-2.5 shadow-tip"><TooltipLines lines={gemShown.lines} center label="Gem tooltip" /></div>
    </div>
  {/if}
{:else if tip && hover}
  <FloatingItemTip anchor={els[hover]} id="{uid}-tip" {...tip} />
{/if}
{#if jewelPick && pickTip}
  <FloatingItemTip anchor={jewelPick.el} id="{uid}-pick-tip" {...pickTip} />
{/if}
