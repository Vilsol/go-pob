<script lang="ts">
  import { isTypingTarget } from './keys';
  import FloatingLabel from './FloatingLabel.svelte';
  import Segmented from './Segmented.svelte';
  import Tabs from './Tabs.svelte';
  import { panelId, tabId } from './tab_ids';
  import Select from './Select.svelte';
  import Button from './Button.svelte';
  import ItemsBoard from './ItemsBoard.svelte';
  import ItemsList from './ItemsList.svelte';
  import ItemsPool from './ItemsPool.svelte';
  import ItemsBrowse from './ItemsBrowse.svelte';
  import ManageSetsDialog from './ManageSetsDialog.svelte';
  import ItemEditor from './ItemEditor.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import Dialog from './Dialog.svelte';
  import DisplayItemControls from './DisplayItemControls.svelte';
  import CraftItemDialog from './CraftItemDialog.svelte';
  import ItemSetTip from './ItemSetTip.svelte';
  import { draggedItem, type ItemDrag } from './item_drag';
  import { dropZone, type DropZone } from './reorder';
  import { tone as toneClass } from './tone';
  import {
    BOARD_SLOTS,
    baseSlot,
    impactRows,
    itemTip,
    replacing,
    type CraftChoice,
    type DisplayEdit,
    type DisplayInfo,
    type ImpactBlock,
    type ItemSetOp,
    type TipProps,
    type WhatIf
  } from './items';
  import type { ItemSet, ItemSummary, ItemTip, ItemsFixture, StatDiff } from './fixtures';
  import type { BrowseKind, BrowseRow, PoolItem, AffixRow } from './fixtures/items';
  import type { CraftApi } from './crafting';
  import type { Snippet } from 'svelte';

  // ItemSlotControl.lua:82-93
  // ItemsTab.lua:335-340
  // ItemListControl.lua:331-369
  type View = 'equipped' | 'stash' | 'shared' | 'browse';
  type Editing = { item: ItemSummary; slot?: string; impact?: StatDiff; isNew?: boolean; key?: string };
  type LivePaste = {
    item?: ItemSummary;
    name?: string;
    error?: string;
    pending: boolean;
    blocks: ImpactBlock[];
    message?: string;
    editing?: string;
  };
  type Props = {
    data: Omit<ItemsFixture, 'paste'> & { paste?: ItemsFixture['paste'] };
    phoneLayout?: 'cards' | 'table';
    sets: ItemSet[];
    setId: string;
    flasks: Record<string, boolean>;
    jewels: { node: number; item: number; choices?: number[] }[];
    /* ItemSlotControl.lua:98-111 */
    abyssal?: string[];
    /* ItemsTab.lua:166-174 */
    extraSlots?: string[];
    /* ItemsTab.lua:4580-4592 */
    socketTip?: { node: number; tip: ItemTip };
    added: string[];
    shared: PoolItem[];
    browse: BrowseKind[];
    affixes: { tiers: string[]; sources: string[]; sorts: string[]; rows: AffixRow[] };
    hover?: string | null;
    view?: View;
    layout?: 'board' | 'list';
    manageOpen?: boolean;
    editing?: Editing | null;
    whatIf?: WhatIf;
    craft?: boolean;
    equipFromList?: boolean;
    slotItems?: Record<string, number[]>;
    readonly?: string;
    paste?: LivePaste;
    pasteText?: string;
    /* ItemsTab.lua:1454-1458 */
    onpasteitem?: (text: string) => void;
    onset?: (id: string) => void;
    onsets?: (op: ItemSetOp) => void;
    onswap?: (swap: boolean) => void;
    onflask: (slot: string, on: boolean) => void;
    onequip: (slot: string, id: number | null) => void;
    onjewel?: (node: number, id: number | null) => void;
    onadd?: (name: string, equip: boolean) => void;
    onsave?: (item: ItemSummary, slot: string | undefined, equip: boolean) => void;
    onedittext?: (editing: Editing) => void;
    editorModTable?: boolean;
    onmodtable?: (editing: Editing) => void;
    onclone?: (editing: Editing) => void;
    customRequest?: boolean;
    oncancelpaste?: () => void;
    /* ItemsTab.lua:342-350 */
    onbuysimilar?: () => void;
    ondelete?: (id: number, confirmed: boolean) => Promise<string | undefined>;
    ondeleteunused?: () => void;
    ondeleteall?: () => void;
    onsort?: () => void;
    ontoggle?: (id: number, second: boolean) => void;
    oncopy?: (id: number) => void;
    display?: DisplayInfo;
    ondisplayedit?: (edit: DisplayEdit) => void;
    displayCraft?: CraftApi;
    craftBases?: { type: string; bases: string[] }[];
    craftPreview?: TipProps;
    oncraftpreview?: (c: CraftChoice) => void;
    oncraftcreate?: (c: CraftChoice) => void;
    craftFrom?: string;
    itemDB?: Snippet;
    /* ItemSlotControl.lua:114-130, ItemListControl.lua:275-284 */
    ondropslot?: (slot: string, drag: ItemDrag) => void;
    /* ListControl.lua:158-163 */
    ondropstash?: (drag: ItemDrag, before?: number) => void;
    /* ItemListControl.lua:77-183 */
    stashFilters?: string[];
    stashFilter?: string;
    onstashfilter?: (filter: string) => void;
    /* ItemsTab.lua:193-206 */
    specs?: string[];
    spec?: number;
    onspec?: (spec: number) => void;
    /* ItemsTab.lua:203-205 */
    onspecmanage?: () => void;
    sharedSets?: { title: string; tip: string[] }[];
    /* ItemSlotControl.lua:132-149 */
    socketView?: import('svelte').ComponentProps<typeof ItemsBoard>['socketView'];
    onshared?: (op: { op: 'share'; id: string } | { op: 'import' | 'delete'; index: number } | { op: 'rename'; index: number; title: string }) => void;
    onshareditem?: (op: { op: 'delete' | 'copy' | 'open'; index: number }) => void;
    ondropshared?: (drag: ItemDrag) => void;
  };
  let {
    data,
    phoneLayout,
    sets,
    setId,
    flasks,
    jewels,
    abyssal = [],
    extraSlots = [],
    socketTip,
    added,
    shared,
    browse,
    affixes,
    hover = $bindable(null),
    view = $bindable('equipped'),
    layout = $bindable('board'),
    manageOpen = $bindable(false),
    editing = $bindable(null),
    whatIf,
    craft = true,
    equipFromList = true,
    slotItems,
    readonly,
    paste: live,
    pasteText = $bindable(data.paste?.text ?? ''),
    onpasteitem,
    onset,
    onsets,
    onswap,
    onflask,
    onequip,
    onjewel,
    onadd,
    onsave,
    onedittext,
    editorModTable,
    onmodtable,
    onclone,
    customRequest = $bindable(false),
    oncancelpaste,
    onbuysimilar,
    ondelete,
    ondeleteunused,
    ondeleteall,
    onsort,
    ontoggle,
    oncopy,
    display,
    ondisplayedit,
    displayCraft,
    craftBases = [],
    craftPreview,
    oncraftpreview,
    oncraftcreate,
    itemDB,
    craftFrom = $bindable(),
    ondropslot,
    ondropstash,
    stashFilters,
    stashFilter,
    onstashfilter,
    specs = [],
    spec = 1,
    onspec,
    onspecmanage,
    sharedSets = [],
    onshared,
    onshareditem,
    ondropshared,
    socketView
  }: Props = $props();
  let crafting = $state(false);
  $effect(() => {
    if (craftFrom) crafting = true;
  });
  $effect(() => {
    if (!crafting) craftFrom = undefined;
  });
  // ItemsTab.lua:105-112
  const pageUid = $props.id();
  let setTip = $state(false);
  let setSelect = $state<HTMLElement>();

  let confirm = $state<{ title: string; message: string; run: () => void } | null>(null);
  const deleteItem = async (id: number) => {
    const prompt = await ondelete?.(id, false);
    if (prompt) confirm = { title: 'Delete Item', message: prompt, run: () => void ondelete?.(id, true) };
  };
  const hasItems = $derived(Object.keys(data.items).length > 0);

  const calcSet = $derived(data.sets.find((s) => s.id === data.activeSet) ?? data.sets[0]);
  const calcFlasks = $derived(Object.fromEntries((calcSet?.slots ?? []).map((x) => [x.slot, x.active])));
  const calcItem = $derived(Object.fromEntries((calcSet?.slots ?? []).map((x) => [x.slot, x.item])));

  const set = $derived(sets.find((s) => s.id === setId) ?? sets[0]);
  const slots = $derived(
    Object.fromEntries(
      (set?.slots ?? []).map((x) => [
        x.slot,
        { id: x.item, item: data.items[x.item], impact: set.id === data.activeSet && calcItem[x.slot] === x.item ? data.impact[x.slot] : undefined }
      ])
    )
  );
  const order = $derived([
    ...new Set([
      ...BOARD_SLOTS.map((b) => b.slot),
      ...extraSlots,
      ...abyssal,
      ...(set?.slots ?? []).map((x) => x.slot).filter((s) => !BOARD_SLOTS.some((b) => b.slot === s))
    ])
  ]);
  const options = $derived(Object.entries(data.items).map(([id, it]) => ({ id: Number(id), label: it.name, tone: it.rarity, it })));
  const fitting = (slot: string) =>
    !equipFromList ? [] : slotItems ? options.filter((o) => slotItems[slot]?.includes(o.id)) : options.filter((o) => o.it.fits.includes(baseSlot(slot)));
  const jewelOptions = $derived(options.filter((o) => /Jewel$/.test(o.it.base)));
  const jewelRows = $derived(jewels.map((j) => ({ node: j.node, id: j.item, item: data.items[j.item], choices: j.choices })));
  // ItemSlotControl.lua:48-56
  const tipFor = (id: number, slot: string) => {
    const it = data.items[id];
    if (!it) return undefined;
    const t = itemTip(it, undefined, 'candidate');
    return whatIf?.key === `${id}@${slot}` ? { ...t, whatIf } : t;
  };
  const onhoveritem = (id: number | null, slot: string) => (hover = id === null ? null : `${id}@${slot}`);

  const stashRows = $derived([
    ...added.map((name) => {
      const r = browse.flatMap((k) => k.rows).find((x) => x.name === name)!;
      const tip = {
        item: r.name,
        base: r.note,
        rarity: r.rarity,
        mods: [],
        impactTitle: `EQUIP IMPACT (IN ${r.type.toUpperCase()})`,
        impact: r.impact ? [r.impact] : []
      };
      return { key: `added:${name}`, tip, slot: r.type, col: 'just now' };
    }),
    ...data.stash.map((s) => {
      const it = data.items[s.item];
      return {
        key: String(s.item),
        tip: itemTip(it, s.impact, 'candidate'),
        slot: s.impact?.slot ?? it.fits[0] ?? (/Jewel$/.test(it.base) ? 'Jewel' : '—'),
        col: s.used ?? 'Unused'
      };
    })
  ]);
  const sharedRows = $derived(
    shared.map((s, i) => ({
      key: String(i),
      tip: { item: s.name, base: s.base, rarity: s.rarity, mods: s.mods, impactTitle: `EQUIP IMPACT (IN ${s.slot.toUpperCase()})`, impact: s.impact },
      slot: s.slot,
      col: s.scope
    }))
  );
  const tabs = $derived([
    { id: 'equipped', label: 'Equipped' },
    { id: 'stash', label: 'Stash', count: String(stashRows.length) },
    ...(shared.length || ondropshared ? [{ id: 'shared', label: 'Shared' }] : []),
    ...(onadd || itemDB ? [{ id: 'browse', label: 'Browse items' }] : [])
  ]);

  const tabDragOver: DropZone = {
    accepts: (d) => !!draggedItem(d),
    over: (_d, hit) => {
      const tab = hit.closest('[role="tab"]');
      const to = tab ? tabs.find((t) => tab.textContent?.trim().startsWith(t.label)) : undefined;
      if (to && (to.id === 'equipped' || to.id === 'stash') && to.id !== view) view = to.id;
    }
  };

  let note = $state('');
  const sample = $derived(data.paste);
  const pasteSlot = $derived(sample ? (sample.impact?.slot ?? sample.item.fits[0]) : undefined);

  const editSlot = (slot: string) => {
    const e = slots[slot];
    if (e) editing = { item: e.item, slot, impact: e.impact, key: slot };
  };
  // ItemsTab.lua:1459-1464
  const onkeydown = (e: KeyboardEvent) => {
    if (e.key !== 'e' || e.ctrlKey || e.metaKey || e.altKey) return;
    if (isTypingTarget(e.target)) return;
    if (hover && slots[hover]) editSlot(hover);
  };
  const editStash = (key: string) => {
    const s = data.stash.find((x) => String(x.item) === key);
    if (s) editing = { item: data.items[s.item], slot: s.impact?.slot, impact: s.impact, key };
  };
  const openStash = (key: string) => {
    const s = data.stash.find((x) => String(x.item) === key);
    if (s) onedittext?.({ item: data.items[s.item], slot: s.impact?.slot, impact: s.impact, key });
  };
  const blank = (name: string, base: string, slot?: string): Editing => ({
    item: { name, base, rarity: 'rare', sockets: '', implicits: [], explicits: [], flask: false, fits: slot ? [slot] : [] },
    slot,
    isNew: true
  });
  const addRow = (r: BrowseRow, equip: boolean) => {
    onadd?.(r.name, equip);
    note = equip ? `Equipped ${r.name} in ${r.type} (the previous item moved to the stash)` : `Added ${r.name} to this build’s stash`;
  };
  const slotLabel = (slot: string) => {
    const socket = /Abyssal Socket (\d+)$/.exec(slot);
    return socket ? `${slot.replace(/ ?Swap/, '').replace(/ Abyssal Socket \d+$/, '')} Abyssal #${socket[1]}` : slot.replace(/ Swap$/, '');
  };
</script>

<svelte:window {onkeydown} />
<div class="flex flex-wrap items-center gap-2.5">
  <div role="presentation" use:dropZone={tabDragOver}>
    <Tabs id={pageUid} variant="segmented" label="Item views" {tabs} selected={view} onchange={(id) => (view = id as View)} />
  </div>
  <FloatingLabel label="Item set" class="[--float-bg:var(--color-surface-1)]"
    ><Select
      class="bg-surface-1"
      aria-describedby={setTip && set?.tip ? `${pageUid}-set-tip` : undefined}
      onmouseenter={(e: MouseEvent & { currentTarget: HTMLSelectElement }) => {
        setSelect = e.currentTarget;
        setTip = true;
      }}
      onmouseleave={() => (setTip = false)}
      onfocus={(e: FocusEvent & { currentTarget: HTMLSelectElement }) => {
        setSelect = e.currentTarget;
        setTip = true;
      }}
      onblur={() => (setTip = false)}
      options={sets.map((s) => ({ value: s.id, label: s.title }))}
      value={setId}
      disabled={!onset || sets.length < 2}
      title={onset ? undefined : 'The item set in use (switching sets is not in the app yet)'}
      onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onset?.(e.currentTarget.value)} /></FloatingLabel>
  {#if onsets}
    <Button size="sm" class="px-2.5 text-12" aria-haspopup="dialog" onclick={() => (manageOpen = true)}>Manage…</Button>
  {/if}
  {#if onspec && specs.length}
    <FloatingLabel label="Passive tree" class="[--float-bg:var(--color-surface-1)]"
      ><Select
        class="max-w-55 bg-surface-1"
        options={specs.map((label, i) => ({ value: String(i + 1), label }))}
        value={String(spec)}
        disabled={specs.length < 2}
        onchange={(e: Event & { currentTarget: HTMLSelectElement }) => onspec(Number(e.currentTarget.value))} /></FloatingLabel>
    {#if onspecmanage}
      <Button size="sm" class="px-2.5 text-12" aria-label="Manage passive trees" aria-haspopup="dialog" onclick={onspecmanage}>Manage…</Button>
    {/if}
  {/if}
  {#if onswap}
    <Segmented
      label="Weapon set"
      options={[
        { id: 'I', label: 'I' },
        { id: 'II', label: 'II' }
      ]}
      value={set?.swap ? 'II' : 'I'}
      onchange={(id) => onswap(id === 'II')} />
  {:else}
    <span class="note" title="The weapon set in use (switching it is not in the app yet)">Weapon set {set?.swap ? 'II' : 'I'}</span>
  {/if}
  {#if craft}
    <Button size="sm" variant="primary" class="px-3 text-12" aria-haspopup="dialog" onclick={() => (editing = blank('New item', 'Choose a base…'))}
      >New item</Button>
  {/if}
  {#if oncraftcreate}
    <Button size="sm" variant="primary" class="px-3 text-12" aria-haspopup="dialog" onclick={() => (crafting = true)}>Craft item…</Button>
  {/if}
  <div class="grow"></div>
  {#if view === 'equipped'}
    <Segmented
      label="Layout"
      options={[
        { id: 'board', label: 'Board' },
        { id: 'list', label: 'List' }
      ]}
      bind:value={layout} />
  {/if}
</div>

<div role="tabpanel" id={panelId(pageUid)} aria-labelledby={tabId(pageUid, view)} class="contents">
  {#if view === 'equipped'}
    <div class="flex flex-wrap items-start gap-4.5 max-phone:flex-col max-phone:flex-nowrap max-phone:items-stretch">
      {#if layout === 'board'}
        <ItemsBoard
          {slots}
          {flasks}
          {calcFlasks}
          {fitting}
          jewels={jewelRows}
          {jewelOptions}
          bind:hover
          {whatIf}
          swap={!!set?.swap}
          onedit={editSlot}
          {onflask}
          {onequip}
          {onjewel}
          {abyssal}
          extra={extraSlots}
          {socketTip}
          {tipFor}
          {onhoveritem}
          ondelete={ondelete && deleteItem}
          ondropitem={ondropslot}
          {socketView} />
      {:else}
        <ItemsList
          {order}
          {slots}
          {flasks}
          {calcFlasks}
          {fitting}
          jewels={jewelRows}
          bind:hover
          {whatIf}
          {phoneLayout}
          onedit={editSlot}
          {onflask}
          {onequip}
          ondelete={ondelete && deleteItem} />
      {/if}
      <section aria-label={craft ? 'Paste or craft an item' : 'Paste an item'} class="panel flex min-w-75 flex-[1_1_0] flex-col gap-2.5 p-3.5">
        <div class="flex items-center justify-between">
          <h2 class="caps">{live?.editing ? `Editing ${live.editing}` : craft ? 'Paste or craft an item' : 'Paste an item'}</h2>
          {#if live?.editing}
            <Button size="sm" variant="ghost" title="Discard the edit; the build keeps the item as it was" onclick={() => oncancelpaste?.()}>Cancel</Button>
          {:else if craft && sample}
            <button
              class="btn-bare text-12 text-accent"
              aria-haspopup="dialog"
              onclick={() => (editing = { item: sample.item, slot: pasteSlot, impact: sample.impact, isNew: true })}>Craft item…</button>
          {/if}
        </div>
        <textarea
          class="input h-27.5 font-mono text-12 whitespace-pre text-text-2"
          aria-label="Item text"
          title="Copy an item in game with Ctrl+C, or from Path of Building"
          spellcheck="false"
          onpaste={(e) => {
            const t = e.clipboardData?.getData('text');
            if (!onpasteitem || !t) return;
            e.preventDefault();
            onpasteitem(t);
          }}
          bind:value={pasteText}></textarea>
        {#if live}
          <!-- ItemsTab.lua:2635-2653 -->
          {#if live.error}
            <p role="alert" class="text-13 text-error">
              The item is invalid. For Rare and Unique items, the first 2 lines must be the title and base name (e.g. "Abberath's Horn" then "Goat's Horn"). For
              Normal and Magic items, the base name must be somewhere in the first line.
            </p>
          {/if}
          {#if live.item}
            {@const it = live.item}
            {#snippet itemHead()}
              <div role="article" aria-label="Item: {live.name ?? it.name}" class={['flex flex-col gap-0.5 text-13', toneClass(it.rarity)]}>
                <span data-pob-color class="toned font-semibold">{it.name}</span>
                <span class="text-text-2"
                  >{it.rarity[0].toUpperCase() + it.rarity.slice(1)}
                  <span data-pob-color class={['toned', toneClass(it.rarity)]}>{it.base}</span>
                  → {it.fits.length ? `fits ${it.fits.join(', ')}` : 'fits no slot'}</span>
              </div>
              {#if onbuysimilar}<Button size="sm" class="self-start" aria-haspopup="dialog" onclick={onbuysimilar}>Buy similar</Button>{/if}
            {/snippet}
            {#snippet itemResult()}
              <section aria-label="Equip impact" aria-busy={live.pending} class="flex flex-col gap-2.5 font-mono text-13">
                <ImpactBlocks blocks={live.blocks}>
                  {#snippet action(b: ImpactBlock)}
                    {#if b.slot}
                      <Button size="sm" variant="primary" class="shrink-0 font-sans" onclick={() => onsave?.(it, b.slot, true)}
                        >Equip in {slotLabel(b.slot)}</Button>
                    {/if}
                  {/snippet}
                </ImpactBlocks>
                {#if live.pending}<span class="font-sans text-12 text-text-3">Calculating…</span>
                {:else if !live.blocks.length}<span class="font-sans text-12 text-text-2">This item fits no slot of the active item set.</span>{/if}
              </section>
              <div class="flex flex-wrap items-center gap-2">
                {#if live.editing}
                  <Button size="md" title="Replace {live.editing} with this text, wherever it is equipped" onclick={() => onsave?.(it, undefined, false)}
                    >Save</Button>
                {:else}
                  <Button size="md" title="Add it to the build's items, equipped in the first free slot it fits" onclick={() => onsave?.(it, undefined, false)}
                    >Save to build</Button>
                {/if}
                {#if live.message}<p role="status" class="text-13 text-text-2">{live.message}</p>{/if}
              </div>
            {/snippet}
            {#if display && ondisplayedit}
              <DisplayItemControls info={display} onedit={ondisplayedit} craft={displayCraft} bind:customRequest head={itemHead} result={itemResult} />
            {:else}
              {@render itemHead()}
              <div class="flex flex-col gap-2.5 border-t border-line pt-2.5">{@render itemResult()}</div>
            {/if}
          {:else if live.message}
            <p role="status" class="text-13 text-text-2">{live.message}</p>
          {/if}
        {:else if sample}
          <div class="text-13 text-text-2">
            {sample.item.rarity[0].toUpperCase() + sample.item.rarity.slice(1)}
            <span data-pob-color class={['toned', toneClass(sample.item.rarity)]}>{sample.item.base}</span>
            → {pasteSlot ? `fits ${pasteSlot}` : 'fits no slot'}
          </div>
          {#if sample.impact}
            <div class="flex flex-col gap-1 border-t border-line pt-2.5 font-mono text-13">
              <div class="font-sans text-12 text-text-2">
                Equip impact{pasteSlot ? ` (in ${pasteSlot}${replacing(sample.impact) ? `, ${replacing(sample.impact)}` : ''})` : ''}
              </div>
              {#each impactRows(sample.impact, 6) as d, i (i)}
                <div class="kv"><span>{d.label}</span><span class={['toned', toneClass(d.dir)]}>{d.text}</span></div>
              {/each}
            </div>
          {/if}
          <div class="flex flex-wrap gap-2">
            {#if pasteSlot}<Button size="md" variant="primary" onclick={() => onsave?.(sample.item, pasteSlot, true)}>Equip in {pasteSlot}</Button>{/if}
            <Button size="md" onclick={() => onsave?.(sample.item, undefined, false)}>Save to stash</Button>
          </div>
        {/if}
      </section>
    </div>
  {:else if view === 'stash'}
    <ItemsPool
      title="Stash"
      rows={stashRows}
      col="USED"
      footer={equipFromList ? 'Hover an item for its equip impact; ⋯ on a slot equips it.' : 'Hover an item for its equip impact in each slot it fits.'}
      empty={stashFilter && stashFilter !== 'Any Loadout' ? `No item is in ${stashFilter}.` : 'This build has no items yet.'}
      bind:hover
      {whatIf}
      onedit={editStash}
      ondelete={ondelete && ((key: string) => void deleteItem(Number(key)))}
      ondeleteunused={hasItems ? ondeleteunused : undefined}
      {onsort}
      ontoggle={ontoggle && ((key: string, second: boolean) => /^\d+$/.test(key) && ontoggle(Number(key), second))}
      onopen={onedittext && openStash}
      oncopy={oncopy && ((key: string) => /^\d+$/.test(key) && oncopy(Number(key)))}
      onclone={onclone && ((key: string) => data.items[key] && onclone({ item: data.items[key], key }))}
      onshare={ondropshared && ((key: string) => ondropshared({ id: Number(key) }))}
      ondropitem={ondropstash && ((d: ItemDrag, before?: string) => ondropstash(d, before === undefined ? undefined : Number(before)))}
      dropAt={!stashFilter || stashFilter === 'Any Loadout'}
      filters={stashFilters}
      filter={stashFilter}
      onfilter={onstashfilter}
      ondeleteall={hasItems && ondeleteall
        ? () => (confirm = { title: 'Delete All', message: 'Are you sure you want to delete all items in this build?', run: ondeleteall })
        : undefined} />
  {:else if view === 'shared'}
    <ItemsPool
      title="Shared"
      rows={sharedRows}
      col="SCOPE"
      footer="Hover for its equip impact; copy one to this build’s stash to equip it."
      empty="This is a list of items that will be shared between all of your builds. You can add items to this list by dragging them from one of the other lists."
      onedit={() => {}}
      ondelete={onshareditem &&
        ((key: string) => {
          const index = Number(key);
          confirm = {
            title: 'Delete Item',
            message: `Are you sure you want to remove '${shared[index]?.name}' from the shared item list?`,
            run: () => onshareditem({ op: 'delete', index })
          };
        })}
      oncopy={onshareditem && ((key: string) => onshareditem({ op: 'copy', index: Number(key) }))}
      onopen={onshareditem && ((key: string) => onshareditem({ op: 'open', index: Number(key) }))}
      ondropitem={ondropshared} />
  {:else if itemDB}
    {@render itemDB()}
  {:else}
    <ItemsBrowse kinds={browse} {added} {note} onadd={addRow} oncraft={(r: BrowseRow) => (editing = blank(`New ${r.name}`, r.name, r.type))} />
  {/if}
</div>

{#if setTip && set?.tip}
  <ItemSetTip anchor={setSelect} id="{pageUid}-set-tip" title={set.title} lines={set.tip} />
{/if}
{#if oncraftcreate}
  <CraftItemDialog bind:open={crafting} groups={craftBases} start={craftFrom} preview={craftPreview} onpreview={oncraftpreview} oncreate={oncraftcreate} />
{/if}
{#if confirm}
  {@const c = confirm}
  <Dialog open title={c.title} onclose={() => (confirm = null)}>
    <p class="text-13 whitespace-pre-line">{c.message}</p>
    {#snippet actions()}
      <Button
        size="md"
        variant="danger"
        onclick={() => {
          const run = c.run;
          confirm = null;
          run();
        }}>Delete</Button>
      <Button size="md" onclick={() => (confirm = null)}>Cancel</Button>
    {/snippet}
  </Dialog>
{/if}
{#if onsets}
  <ManageSetsDialog bind:open={manageOpen} {sets} {setId} onpick={(id: string) => onset?.(id)} {onsets} shared={sharedSets} {onshared} />
{/if}
{#if editing}
  {@const ed = editing}
  {#key ed}
    <ItemEditor
      open
      item={ed.item}
      slot={ed.slot}
      impact={ed.impact}
      isNew={ed.isNew}
      {affixes}
      {readonly}
      whatIf={whatIf && whatIf.key === ed.key ? whatIf : undefined}
      onedittext={onedittext && (() => onedittext(ed))}
      modTable={editorModTable}
      onmodtable={onmodtable && (() => onmodtable(ed))}
      onclone={onclone && (() => onclone(ed))}
      onsave={(it: ItemSummary, equip: boolean) => onsave?.(it, ed.slot, equip)}
      onclose={() => (editing = null)} />
  {/key}
{/if}
