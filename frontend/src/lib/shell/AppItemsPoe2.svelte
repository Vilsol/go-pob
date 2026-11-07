<script lang="ts">
  import { untrack } from 'svelte';
  import ItemsPage from '$lib/design/ItemsPage.svelte';
  import ItemDB from '$lib/design/ItemDB.svelte';
  import { defaultItemDBFilter, type ItemDBFilterInput, type ItemDBKind, type ItemDBLists, type ItemDBRowPlain } from '$lib/design/item_db';
  import FloatingLabel from '$lib/design/FloatingLabel.svelte';
  import Select from '$lib/design/Select.svelte';
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { BOARD_SLOTS_POE2, itemTip, type CraftChoice, type ImpactBlock, type ItemSetOp, type TipProps, type WhatIf } from '$lib/design/items';
  import { BOARD_PANES_POE2 } from '$lib/design/mobile/mobile';
  import type { RuneSocketsEdit } from '$lib/design/RuneSockets.svelte';
  import type { ItemRangesEdit, ItemRangesInfo } from '$lib/design/ItemRanges.svelte';
  import type { ItemSet, ItemSummary, ItemTip, ModLine } from '$lib/design/fixtures';
  import { itemsLayout, phoneTables } from '$lib/display/appearance';
  import { poe2Build } from '$lib/global';
  import { poe2Busy, poe2Error, poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import type { Poe2AffixControl, Poe2CompareBlock, Poe2CompareTarget, Poe2DisplayEdit, Poe2ItemsEdit, Poe2ItemsPage } from '$lib/go/sync_worker2';
  import type { AffixControls, CraftApi } from '$lib/design/crafting';
  import { slotOnlyTooltips } from '$lib/display/item_options';

  // ItemsTab.lua, ItemSlotControl.lua, ItemSetListControl.lua (PoB-PoE2): the Items page over the open PoE2 build.
  // The rare database, shared items, item variants, influences, quality and drag and drop are not here yet

  type Item = Poe2ItemsPage['items'][number];
  type Editing = { item: ItemSummary; slot?: string; isNew?: boolean; key?: string };

  const AFFIXES = { tiers: [], sources: [], sorts: [], rows: [] };
  const READONLY = 'Edit as text to change it in the paste panel, under its controls';

  const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));
  const plain = (t: string) => t.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '');

  let page = $state.raw<Poe2ItemsPage>();
  let affixControls = $state.raw<Poe2AffixControl[]>([]);
  let ranges = $state.raw<ItemRangesInfo>();
  let whatIf = $state<WhatIf>();
  // a read that makes fewer worker calls can finish before an older one; only the newest may land
  let readSeq = 0;
  const read = () => {
    const mine = ++readSeq;
    return poe2Worker()
      .then(async (w) => {
        const p = w.ItemsPage();
        const shown = !!(await p).display;
        const a = shown ? await w.DisplayItemAffixes() : [];
        const r = shown ? await w.DisplayItemRanges() : undefined;
        if (mine !== readSeq) return;
        affixControls = a;
        ranges = r;
        page = await p;
      })
      .catch(fail);
  };
  // read again after every calculation
  $effect(() => {
    void $poe2Build;
    void untrack(read);
  });

  const edit = (e: Poe2ItemsEdit) => poe2Run((w) => w.EditItems(e));
  const editDisplay = async (e: Poe2DisplayEdit) => {
    try {
      await (await poe2Worker()).EditDisplayItem(e);
      return true;
    } catch (err) {
      fail(err);
      return false;
    } finally {
      await read();
    }
  };

  // the slots each item fits, and where the active set holds it
  const fitsOf = (id: number) => (page?.slots ?? []).filter((s) => s.choices.includes(id)).map((s) => s.name);
  const usedIn = (id: number) => page?.slots.find((s) => s.itemId === id);
  const isFlask = (fits: string[]) => fits.some((s) => /^(Flask|Charm) /.test(s));

  // ItemsTab.lua:3720-3729, 4039-4141 (PoB-PoE2)
  const tipOf = (it: Pick<Item, 'name' | 'lines'>): ItemTip => ({
    influence: [],
    title: it.name,
    lines: it.lines.flatMap((g) => [{ text: '', sep: true }, ...g.map((text) => ({ text, sep: false }))])
  });
  const modLines = (g: string[] | undefined): ModLine[] => (g ?? []).map((l) => ({ text: plain(l), kind: 'explicit' }));
  const summary = (it: Pick<Item, 'rarity' | 'name' | 'lines'>, fits: string[]): ItemSummary => ({
    name: plain(it.name[0] ?? ''),
    base: plain(it.name[1] ?? it.name[0] ?? ''),
    rarity: (it.rarity.toLowerCase() || 'normal') as ItemSummary['rarity'],
    sockets: '',
    implicits: it.lines.length > 1 ? modLines(it.lines.slice(0, -1).flat()) : [],
    explicits: modLines(it.lines.at(-1)),
    flask: isFlask(fits),
    fits,
    tooltip: tipOf(it)
  });

  const items = $derived(Object.fromEntries((page?.items ?? []).map((i) => [String(i.id), summary(i, fitsOf(i.id))])));
  const boardSlots = new Set(BOARD_SLOTS_POE2.map((b) => b.slot));
  const itemSlots = $derived((page?.slots ?? []).filter((s) => !s.nodeId));
  const sets = $derived(
    (page?.sets ?? []).map(
      (s): ItemSet => ({
        id: String(s.id),
        title: s.title,
        swap: s.id === page?.activeSet && page.useSecondWeaponSet,
        slots: s.id === page?.activeSet ? itemSlots.filter((x) => x.itemId).map((x) => ({ slot: x.name, item: x.itemId, active: x.active })) : []
      })
    )
  );
  const activeSet = $derived(String(page?.activeSet ?? 1));
  const flasks = $derived(Object.fromEntries(itemSlots.filter((s) => s.activatable).map((s) => [s.name, s.active])));
  const jewels = $derived((page?.slots ?? []).filter((s) => s.nodeId).map((s) => ({ node: s.nodeId, item: s.itemId, choices: s.choices })));
  const extraSlots = $derived(itemSlots.filter((s) => !boardSlots.has(s.name.replace(/ Swap$/, ''))).map((s) => s.name));
  const slotItems = $derived(Object.fromEntries((page?.slots ?? []).map((s) => [s.name, s.choices])));
  const stash = $derived(
    (page?.items ?? []).map((i) => {
      const at = usedIn(i.id);
      return { item: i.id, used: at ? at.label : 'Unused' };
    })
  );

  // the paste panel: its text opens the item in the editor, which the page shows with its runes
  let text = $state('');
  let pushed = '';
  let error = $state<string>();
  let message = $state<string>();
  const display = $derived(page?.display);
  const pasteItem = $derived(display && summary(display.item, display.fits));
  const label = (slot: string) => page?.slots.find((s) => s.name === slot)?.label ?? slot;
  // ItemsTab.lua:4425-4538 (PoB-PoE2): the editor's item's stat differences, one block per slot it fits
  let blocks = $state<ImpactBlock[]>([]);
  let blocksPending = $state(false);
  let compareSeq = 0;
  const stamp = (b: Poe2CompareBlock[]): ImpactBlock[] => {
    const readyAt = performance.now();
    return b.map((x) => ({ ...x, readyAt }));
  };
  $effect(() => {
    const d = display;
    void $poe2Build;
    const mine = ++compareSeq;
    if (!d) {
      blocks = [];
      return;
    }
    blocksPending = true;
    void untrack(() =>
      poe2Worker()
        .then((w) => w.ItemCompare({ kind: 'display' }))
        .then((b) => {
          if (mine === compareSeq) blocks = stamp(b);
        })
        .catch(fail)
        .finally(() => {
          if (mine === compareSeq) blocksPending = false;
        })
    );
  });
  let seq = 0;
  $effect(() => {
    const t = text;
    if (t === pushed) return;
    const mine = ++seq;
    void untrack(async () => {
      pushed = t;
      error = undefined;
      message = undefined;
      if (!t.trim()) {
        await editDisplay({ op: 'close' });
        return;
      }
      const w = await poe2Worker();
      try {
        // ItemsTab.lua:2871-2899 (PoB-PoE2): an item opened from the build keeps its id
        if (display?.id) await w.EditDisplayItem({ op: 'text', text: t, rarity: display.rarity });
        else await w.EditDisplayItem({ op: 'paste', text: t });
      } catch (err) {
        if (mine === seq) error = err instanceof Error ? err.message : String(err);
      }
      if (mine === seq) await read();
    });
  });
  const openText = (t: string) => {
    pushed = t;
    text = t;
  };

  // ItemsTab.lua:447-458, ItemSlotControl.lua:154-166 (PoB-PoE2)
  const onsave = async (slot: string | undefined, equip: boolean) => {
    const name = pasteItem?.name;
    await edit({ op: 'add', slot: equip ? slot : undefined });
    if (!$poe2Error) {
      message = equip && slot ? `Equipped ${name} in ${label(slot)}.` : `Saved ${name}.`;
      openText('');
    }
  };
  const oncancelpaste = () => {
    openText('');
    void editDisplay({ op: 'close' });
  };
  // ItemListControl.lua:321-328 (PoB-PoE2)
  const openItem = async (id: number) => {
    if (await editDisplay({ op: 'edit', id })) openText(page?.display ? `Rarity: ${page.display.rarity}\n${page.display.text}` : '');
  };
  const onrunes = async (e: RuneSocketsEdit) => {
    if (e.op === 'sockets') await editDisplay({ op: 'sockets', count: e.count });
    else await editDisplay({ op: 'rune', socket: e.socket, rune: e.rune });
    if (page?.display) openText(`Rarity: ${page.display.rarity}\n${page.display.text}`);
  };

  // ItemsTab.lua:932-947, 1067-1104, 2232-2518 (PoB-PoE2): the editor's affix drop-downs and roll sliders
  const displayAffixes = $derived<AffixControls | undefined>(
    affixControls.length
      ? {
          sorting: false,
          controls: affixControls.map((c) => ({
            type: c.type,
            sel: c.sel,
            sliderShown: c.sliderShown,
            sliderVal: c.sliderVal,
            series: c.list.map((e) => ({ label: e.label, tiers: e.modList.length, haveRange: e.haveRange, tooltip: e.tooltip, mid: [] }))
          }))
        }
      : undefined
  );
  const affixCraft: CraftApi = {
    enchantChoices: async () => undefined,
    notables: async () => [],
    sortStats: async () => [],
    preview: async () => undefined,
    enchantSort: async () => [],
    anointPowers: async () => [],
    anointCompare: async () => undefined,
    affixEdit: async (slot, series, slider) => {
      try {
        await (await poe2Worker()).EditDisplayItemAffix(slot, series, slider);
      } catch (err) {
        fail(err);
      } finally {
        await read();
      }
      if (page?.display) openText(`Rarity: ${page.display.rarity}\n${page.display.text}`);
      return undefined;
    },
    affixRollTooltip: async (slot, slider) => (await poe2Worker()).DisplayItemAffixRoll(slot, slider)
  };

  const onranges = async (e: ItemRangesEdit) => {
    try {
      await (await poe2Worker()).EditDisplayItemRange(e);
    } catch (err) {
      fail(err);
    } finally {
      await read();
    }
    if (page?.display) openText(`Rarity: ${page.display.rarity}\n${page.display.text}`);
  };

  // ItemListControl.lua:337-375 (PoB-PoE2)
  const ondelete = async (id: number, confirmed: boolean) => {
    const at = usedIn(id);
    if (at && !confirmed) return `${items[id]?.name} is currently equipped in ${at.label}.\nAre you sure you want to delete it?`;
    await edit({ op: 'delete', id });
    return undefined;
  };

  // ItemSetListControl.lua, ItemSetService.lua (PoB-PoE2)
  const onsets = (op: ItemSetOp) => {
    switch (op.op) {
      case 'new':
        return void edit({ op: 'newSet', title: op.title });
      case 'copy':
        return void edit({ op: 'copySet', id: Number(op.id), title: op.title });
      case 'rename':
        return void edit({ op: 'renameSet', id: Number(op.id), title: op.title });
      case 'delete':
        return void edit({ op: 'deleteSet', id: Number(op.id) });
      case 'move':
        return void edit({ op: 'moveSet', from: op.from + 1, to: op.to + 1 });
    }
  };

  // ItemsTab.lua:2762-2868 (PoB-PoE2): the Craft Item popup; its rarity choice is 0-based here, 1-based in Lua
  let craftBases = $state<{ type: string; bases: string[] }[]>([]);
  let craftPreview = $state<TipProps>();
  $effect(() => {
    void untrack(() =>
      poe2Worker()
        .then((w) => w.CraftBases())
        .then((g) => (craftBases = g))
        .catch(fail)
    );
  });
  let craftSeq = 0;
  const oncraftpreview = async (c: CraftChoice) => {
    const mine = ++craftSeq;
    try {
      const w = await poe2Worker();
      const it = await w.CraftItemPreview(c.rarity + 1, c.title, c.base);
      if (mine !== craftSeq) return;
      const tip = { ...itemTip(summary(it, []), undefined, 'candidate'), impactTitle: '', impactSub: undefined, impact: [] };
      craftPreview = { ...tip, whatIf: { key: 'craft', pending: true, blocks: [] } };
      const b = await w.ItemCompare({ kind: 'craft', rarity: c.rarity + 1, title: c.title, base: c.base });
      if (mine === craftSeq) craftPreview = { ...tip, whatIf: { key: 'craft', pending: false, blocks: stamp(b) } };
    } catch {
      if (mine === craftSeq) craftPreview = undefined;
    }
  };
  const oncraftcreate = async (c: CraftChoice, slot?: string) => {
    try {
      (await poe2Worker()).CraftDisplayItem(c.rarity + 1, c.title, c.base);
    } catch (err) {
      fail(err);
      return;
    }
    await read();
    if (slot) await edit({ op: 'add', slot });
    else if (page?.display) openText(`Rarity: ${page.display.rarity}\n${page.display.text}`);
  };

  // ItemsTab.lua:379-414, ItemDBControl.lua (PoB-PoE2): the unique database; the rare one needs the affix data
  const DB_KINDS: ItemDBKind[] = ['UNIQUE', 'RARE'];
  let dbKind = $state<ItemDBKind>('UNIQUE');
  let dbFilter = $state<ItemDBFilterInput>(defaultItemDBFilter());
  let dbSort = $state('');
  let dbLists = $state<ItemDBLists>();
  let dbRows = $state<ItemDBRowPlain[]>([]);
  let dbBusy = $state(false);
  let dbHover = $state<string | null>(null);
  let dbTip = $state<{ name: string; tip: TipProps }>();
  let dbNote = $state('');
  let dbListsKind: ItemDBKind | undefined;
  $effect(() => {
    const kind = dbKind;
    if (pageView !== 'browse' || dbListsKind === kind) return;
    void untrack(() =>
      poe2Worker()
        .then((w) => w.ItemDBLists(kind))
        .then((l) => {
          dbLists = l;
          dbListsKind = kind;
        })
        .catch(fail)
    );
  });
  let dbSeq = 0;
  // ItemDBControl.lua:300-325 (PoB-PoE2): the list builds again on a filter change and after every calculation
  $effect(() => {
    const filter = $state.snapshot(dbFilter);
    const kind = dbKind;
    void $poe2Build;
    if (pageView !== 'browse') return;
    const mine = ++dbSeq;
    dbBusy = true;
    void untrack(() =>
      poe2Worker()
        .then((w) => w.ItemDBRows(kind, filter))
        .then((rows) => {
          if (mine !== dbSeq) return;
          dbRows = rows;
          dbBusy = false;
        })
        .catch(fail)
    );
  });
  $effect(() => {
    const name = dbHover;
    if (name === null) return;
    void untrack(() =>
      poe2Worker()
        .then((w) => w.ItemDBTooltip(dbKind, name))
        .then((it) => {
          if (dbHover === name)
            dbTip = { name, tip: { ...itemTip(summary(it, []), undefined, 'candidate'), impactTitle: '', impactSub: undefined, impact: [] } };
        })
        .catch(fail)
    );
  });
  const dbTipProps = $derived(
    dbTip && dbTip.name === dbHover ? { ...dbTip.tip, whatIf: whatIf?.key === `db:${dbKind}:${dbTip.name}` ? whatIf : undefined } : undefined
  );
  const dbAdd = async (name: string, equip: boolean, second: boolean) => {
    if (!(await poe2Run((w) => w.AddDBItem(dbKind, name, equip, second)))) return;
    dbNote = equip ? `Added ${name} and equipped it in its slot` : `Added ${name} to the build's items`;
  };
  const dbOpen = async (name: string) => {
    try {
      (await poe2Worker()).OpenDBItem(dbKind, name);
    } catch (err) {
      fail(err);
      return;
    }
    await read();
    if (page?.display) openText(`Rarity: ${page.display.rarity}\n${page.display.text}`);
    dbHover = null;
    pageView = 'equipped';
  };
  const dbCopy = (name: string) =>
    void poe2Worker()
      .then((w) => w.ItemDBRaw(dbKind, name))
      .then((t) => navigator.clipboard?.writeText(t.replace(/\r?\n/g, '\r\n')))
      .catch(fail);
  let craftFrom = $state<string>();

  let layout = $state($itemsLayout);
  let editing = $state<Editing | null>(null);
  let hover = $state<string | null>(null);

  // ItemsTab.lua:4425-4538, ItemDBControl.lua:333-341 (PoB-PoE2): the hovered item's stat differences; an equipped
  // slot, and an item in a slot's list with "slot only tooltips", show only that slot's block
  const target = $derived(editing?.key ?? hover ?? (dbHover === null ? null : `db:${dbKind}:${dbHover}`));
  const whatIfOf = (key: string): { t: Poe2CompareTarget; slot?: string } | undefined => {
    const inSlot = page?.slots.find((s) => s.name === key && s.itemId);
    if (inSlot) return { t: { kind: 'item', id: inSlot.itemId }, slot: inSlot.name };
    const m = /^(\d+)@(.+)$/.exec(key);
    if (m) return { t: { kind: 'item', id: Number(m[1]) }, slot: $slotOnlyTooltips ? m[2] : undefined };
    const db = /^db:(UNIQUE|RARE):(.+)$/.exec(key);
    if (db) return { t: { kind: 'db', db: db[1] as ItemDBKind, name: db[2] } };
    return /^\d+$/.test(key) && items[key] ? { t: { kind: 'item', id: Number(key) } } : undefined;
  };
  let whatIfSeq = 0;
  $effect(() => {
    const key = target;
    void $poe2Build;
    const mine = ++whatIfSeq;
    const job = key ? untrack(() => whatIfOf(key)) : undefined;
    if (!key || !job) {
      whatIf = undefined;
      return;
    }
    whatIf = { key, pending: true, blocks: [] };
    void untrack(() =>
      poe2Worker()
        .then((w) => w.ItemCompare(job.t))
        .then((b) => {
          if (mine === whatIfSeq) whatIf = { key, pending: false, blocks: stamp(job.slot ? b.filter((x) => !x.slot || x.slot === job.slot) : b) };
        })
        .catch(fail)
    );
  });
  let pageView = $state<'equipped' | 'stash' | 'shared' | 'browse'>('equipped');
</script>

{#if page}
  <div class="flex min-h-0 min-w-0 flex-1 flex-col gap-3" data-poe2-page aria-label="Items">
    <ItemsPage
      bind:layout
      phoneLayout={$phoneTables}
      board={{ slots: BOARD_SLOTS_POE2, panes: BOARD_PANES_POE2 }}
      data={{ activeSet, items, sets, impact: {}, jewels, stash }}
      {sets}
      setId={activeSet}
      {flasks}
      {jewels}
      {extraSlots}
      added={[]}
      shared={[]}
      browse={[]}
      affixes={AFFIXES}
      bind:view={pageView}
      bind:hover
      {whatIf}
      bind:editing
      craft={false}
      {slotItems}
      readonly={READONLY}
      paste={{
        item: pasteItem,
        name: pasteItem?.name,
        error,
        pending: $poe2Busy || blocksPending,
        blocks,
        message,
        editing: display?.id ? pasteItem?.name : undefined
      }}
      bind:pasteText={text}
      onpasteitem={(t: string) => (text = t)}
      runes={display && { canHaveSockets: display.canHaveSockets, sockets: display.runes, choices: display.runeChoices }}
      onrunes={(e: RuneSocketsEdit) => void onrunes(e)}
      {displayAffixes}
      {affixCraft}
      {ranges}
      onranges={(e: ItemRangesEdit) => void onranges(e)}
      onset={(id: string) => void edit({ op: 'set', id: Number(id) })}
      {onsets}
      onswap={(second: boolean) => void edit({ op: 'swap', second })}
      onflask={(slot: string, on: boolean) => void edit({ op: 'active', slot, state: on })}
      onequip={(slot: string, id: number | null) => void edit({ op: 'equip', slot, id: id ?? 0 })}
      onjewel={(node: number, id: number | null) => void edit({ op: 'equip', slot: `Jewel ${node}`, id: id ?? 0 })}
      onsave={(_: ItemSummary, slot: string | undefined, equip: boolean) => void onsave(slot, equip)}
      onedittext={(ed: Editing) => {
        editing = null;
        const id = Number(ed.key && /^\d+$/.test(ed.key) ? ed.key : ed.slot && page?.slots.find((s) => s.name === ed.slot)?.itemId);
        if (id) void openItem(id);
      }}
      {oncancelpaste}
      {craftBases}
      {craftPreview}
      bind:craftFrom
      oncraftpreview={(c: CraftChoice) => void oncraftpreview(c)}
      oncraftcreate={(c: CraftChoice, slot?: string) => void oncraftcreate(c, slot)}
      {ondelete}
      oncopy={(id: number) =>
        void poe2Worker()
          .then((w) => w.ItemText(id))
          .then((t) => navigator.clipboard?.writeText(t.replace(/\r?\n/g, '\r\n')))
          .catch(fail)}>
      {#snippet itemDB()}
        <ItemDB
          bind:kind={dbKind}
          kinds={DB_KINDS}
          lists={dbLists}
          bind:filter={dbFilter}
          bind:sort={dbSort}
          rows={dbRows}
          busy={dbBusy}
          bind:hover={dbHover}
          tip={dbTipProps}
          note={dbNote}
          onadd={(name: string, equip: boolean, second: boolean) => void dbAdd(name, equip, second)}
          onopen={(name: string) => void dbOpen(name)}
          oncopy={dbCopy}
          oncraft={(base: string) => (craftFrom = base)} />
      {/snippet}
    </ItemsPage>
    {#if page.runeSlots.length}
      <!-- ItemsTab.lua:267-310 (PoB-PoE2): the character's rune slots, shown with Martial Artist's SocketRunesOnCharacter -->
      <section aria-label="Character runes" class="panel flex flex-wrap gap-2.5 p-3.5">
        {#each page.runeSlots as r (r.name)}
          <FloatingLabel label={r.label} class="[--float-bg:var(--color-surface-1)]"
            ><Select
              class="bg-surface-1"
              aria-label={r.label}
              options={r.choices}
              value={r.rune}
              onchange={(e: Event & { currentTarget: HTMLSelectElement }) =>
                void edit({ op: 'characterRune', slot: r.name, rune: e.currentTarget.value })} /></FloatingLabel>
        {/each}
      </section>
    {/if}
  </div>
{:else}
  <EmptyState busy text="Loading the items…" />
{/if}
