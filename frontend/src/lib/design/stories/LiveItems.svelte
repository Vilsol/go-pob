<script lang="ts">
  import ItemsPage from '../ItemsPage.svelte';
  import type { BuildFixture } from '../fixtures';
  import type { ItemSetOp, WhatIf } from '../items';
  import { sharedItems, browseKinds, modPicker } from '../fixtures/items';

  type View = 'equipped' | 'stash' | 'shared' | 'browse';
  type Props = {
    fx: BuildFixture;
    view?: View;
    layout?: 'board' | 'list';
    manage?: boolean;
    edit?: string;
    hover?: string;
    app?: boolean;
    onedittext?: () => void;
    trees?: boolean;
    phoneLayout?: 'cards' | 'table';
  };
  let { fx, view = 'equipped', layout = 'board', manage = false, edit, hover, app = false, onedittext, trees = false, phoneLayout }: Props = $props();
  let treeSpec = $state(1);
  let treeManageAsks = $state(0);
  const treeProps = $derived(
    trees ? { specs: ['Leveling', 'Endgame'], spec: treeSpec, onspec: (i: number) => (treeSpec = i), onspecmanage: () => treeManageAsks++ } : {}
  );

  const d = fx.items;
  const start = d.sets.find((s) => s.id === d.activeSet) ?? d.sets[0];
  const onEdit = start.slots.find((x) => x.slot === edit);

  let s = $state({
    view,
    layout,
    manage,
    sets: structuredClone(d.sets),
    setId: start.id,
    flasks: Object.fromEntries(start.slots.filter((x) => x.slot.startsWith('Flask')).map((x) => [x.slot, x.active])),
    added: [] as string[],
    jewels: structuredClone(d.jewels),
    editing: onEdit ? { item: d.items[onEdit.item], slot: onEdit.slot, impact: d.impact[onEdit.slot], key: onEdit.slot } : null,
    hover: hover ?? null
  });

  const target = $derived(s.editing?.key ?? s.hover);
  const whatIf = $derived.by((): WhatIf | undefined => {
    if (!app || !target) return undefined;
    const stash = d.stash.find((x) => String(x.item) === target);
    const diff = stash ? stash.impact : d.impact[target];
    return { key: target, pending: false, blocks: diff ? [diff] : [] };
  });
  const paste = { item: d.paste.item, name: d.paste.item.name, pending: false, blocks: d.paste.impact ? [d.paste.impact] : [] };

  const freeId = () => {
    let id = 1;
    while (s.sets.some((x) => x.id === String(id))) id++;
    return String(id);
  };
  const setOp = (op: ItemSetOp) => {
    if (op.op === 'new') s.sets = [...s.sets, { id: freeId(), title: op.title, swap: false, slots: [] }];
    else if (op.op === 'copy') {
      const from = s.sets.find((x) => x.id === op.id);
      if (from) s.sets = [...s.sets, { ...$state.snapshot(from), id: freeId(), title: op.title }];
    } else if (op.op === 'rename') s.sets = s.sets.map((x) => (x.id === op.id ? { ...x, title: op.title } : x));
    else if (op.op === 'delete' && s.sets.length > 1) {
      const i = s.sets.findIndex((x) => x.id === op.id);
      s.sets = s.sets.filter((x) => x.id !== op.id);
      if (op.id === s.setId) s.setId = s.sets[Math.max(0, i - 1)].id;
    } else if (op.op === 'move') {
      const next = [...s.sets];
      next.splice(op.to, 0, ...next.splice(op.from, 1));
      s.sets = next;
    }
  };

  const setSlots = (slot: string, item: number | null) => {
    s.sets = s.sets.map((set) =>
      set.id !== s.setId ? set : { ...set, slots: [...set.slots.filter((x) => x.slot !== slot), ...(item ? [{ slot, item, active: true }] : [])] }
    );
  };
</script>

{#if app}
  <ItemsPage
    {phoneLayout}
    data={d}
    sets={s.sets}
    setId={s.setId}
    flasks={s.flasks}
    jewels={s.jewels}
    added={[]}
    shared={[]}
    browse={[]}
    affixes={modPicker}
    bind:hover={s.hover}
    bind:view={s.view}
    bind:layout={s.layout}
    bind:editing={s.editing}
    {whatIf}
    craft={false}
    equipFromList={false}
    readonly="Crafting is not in the app yet"
    {paste}
    onflask={(slot: string, on: boolean) => (s.flasks[slot] = on)}
    onequip={setSlots}
    onedittext={() => onedittext?.()} />
{:else}
  <ItemsPage
    {phoneLayout}
    data={d}
    sets={s.sets}
    setId={s.setId}
    flasks={s.flasks}
    jewels={s.jewels}
    added={s.added}
    shared={sharedItems}
    browse={browseKinds}
    affixes={modPicker}
    {hover}
    bind:view={s.view}
    bind:layout={s.layout}
    bind:manageOpen={s.manage}
    bind:editing={s.editing}
    onset={(id: string) => (s.setId = id)}
    onsets={setOp}
    onswap={(swap: boolean) => (s.sets = s.sets.map((x) => (x.id === s.setId ? { ...x, swap } : x)))}
    onflask={(slot: string, on: boolean) => (s.flasks[slot] = on)}
    onequip={setSlots}
    onjewel={(node: number, id: number | null) => (s.jewels = s.jewels.map((j) => (j.node === node ? { node, item: id ?? 0 } : j)))}
    onadd={(name: string) => {
      if (!s.added.includes(name)) s.added = [...s.added, name];
    }}
    {...treeProps} />
  {#if trees}<output aria-label="Tree manager asked" class="sr-only">{treeManageAsks}</output>{/if}
{/if}
