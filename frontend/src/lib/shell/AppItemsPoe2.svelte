<script lang="ts">
  import { untrack } from 'svelte';
  import ItemsPage from '$lib/design/ItemsPage.svelte';
  import FloatingLabel from '$lib/design/FloatingLabel.svelte';
  import Select from '$lib/design/Select.svelte';
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { BOARD_SLOTS_POE2, type ImpactBlock, type ItemSetOp } from '$lib/design/items';
  import { BOARD_PANES_POE2 } from '$lib/design/mobile/mobile';
  import type { RuneSocketsEdit } from '$lib/design/RuneSockets.svelte';
  import type { ItemSet, ItemSummary, ItemTip, ModLine } from '$lib/design/fixtures';
  import { itemsLayout, phoneTables } from '$lib/display/appearance';
  import { poe2Build } from '$lib/global';
  import { poe2Busy, poe2Error, poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import type { Poe2DisplayEdit, Poe2ItemsEdit, Poe2ItemsPage } from '$lib/go/sync_worker2';

  // ItemsTab.lua, ItemSlotControl.lua, ItemSetListControl.lua (PoB-PoE2): the Items page over the open PoE2 build.
  // Crafting, the unique and rare databases, shared items, the equip impact, item variants, influences, quality and
  // drag and drop are not here yet

  type Item = Poe2ItemsPage['items'][number];
  type Editing = { item: ItemSummary; slot?: string; isNew?: boolean; key?: string };

  const AFFIXES = { tiers: [], sources: [], sorts: [], rows: [] };
  const READONLY = 'Edit as text to change it in the paste panel, under its controls';

  const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));
  const plain = (t: string) => t.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '');

  let page = $state.raw<Poe2ItemsPage>();
  const read = () =>
    poe2Worker()
      .then((w) => w.ItemsPage())
      .then((p) => (page = p))
      .catch(fail);
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
  const blocks = $derived((display?.fits ?? []).map((slot): ImpactBlock => ({ header: label(slot), slot, sections: [], unmeasured: true })));
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

  let layout = $state($itemsLayout);
  let editing = $state<Editing | null>(null);
  let hover = $state<string | null>(null);
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
      bind:editing
      craft={false}
      {slotItems}
      readonly={READONLY}
      paste={{ item: pasteItem, name: pasteItem?.name, error, pending: $poe2Busy, blocks, message, editing: display?.id ? pasteItem?.name : undefined }}
      bind:pasteText={text}
      onpasteitem={(t: string) => (text = t)}
      runes={display && { canHaveSockets: display.canHaveSockets, sockets: display.runes, choices: display.runeChoices }}
      onrunes={(e: RuneSocketsEdit) => void onrunes(e)}
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
      {ondelete}
      oncopy={(id: number) =>
        void poe2Worker()
          .then((w) => w.ItemText(id))
          .then((t) => navigator.clipboard?.writeText(t.replace(/\r?\n/g, '\r\n')))
          .catch(fail)} />
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
