<script lang="ts">
  import Input from './Input.svelte';
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import Dropdown from './Dropdown.svelte';
  import ItemTooltip from './ItemTooltip.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import type { CraftChoice, TipProps, UniqueChoice } from './items';
  import { craftFitsSlot } from './crafting';

  // ItemsTab.lua:2519-2603
  type Props = {
    open?: boolean;
    groups: { type: string; bases: string[] }[];
    start?: string;
    /* the slot it is crafted for: only the types that fit it are offered */
    slot?: string;
    preview?: TipProps;
    onpreview?: (c: CraftChoice) => void;
    oncreate: (c: CraftChoice) => void;
    /* the Unique rarity picks from the unique database instead of a base */
    uniques?: () => Promise<UniqueChoice[]>;
    onpreviewunique?: (name: string) => void;
    oncreateunique?: (name: string) => void;
  };
  let { open = $bindable(false), groups: all, start, slot, preview, onpreview, oncreate, uniques, onpreviewunique, oncreateunique }: Props = $props();

  const RARITIES = ['Normal', 'Magic', 'Rare', 'Unique', 'Relic'];
  let rarity = $state(2);
  let title = $state('');
  let type = $state(0);
  let base = $state(0);
  const groups = $derived(slot ? all.filter((g) => craftFitsSlot(slot, g.type)) : all);
  const bases = $derived(groups[type]?.bases ?? []);
  $effect(() => {
    if (!open || !slot) return;
    type = 0;
    base = 0;
  });
  $effect(() => {
    if (!open || !start) return;
    const t = groups.findIndex((g) => g.bases.includes(start));
    if (t < 0) return;
    type = t;
    base = groups[t].bases.indexOf(start);
  });
  const choice = $derived({ rarity, title, base: bases[base] ?? '' });
  let hovered = $state<number | null>(null);
  let hoveredEl = $state<HTMLElement | null>(null);
  const uid = $props.id();

  const UNIQUE = 3;
  const uniqueMode = $derived(rarity === UNIQUE && !!uniques && !!oncreateunique);
  let uniqueRows = $state<UniqueChoice[]>();
  let unique = $state(0);
  let asked = false;
  $effect(() => {
    if (!uniqueMode || asked) return;
    asked = true;
    uniques?.()
      .then((rows) => (uniqueRows = rows))
      .catch(() => (uniqueRows = []));
  });
  const shownUniques = $derived((uniqueRows ?? []).filter((u) => !slot || craftFitsSlot(slot, u.type)));
  $effect(() => {
    void slot;
    unique = 0;
  });

  $effect(() => {
    if (!open) return;
    if (uniqueMode) {
      const u = shownUniques[hovered ?? unique] ?? shownUniques[unique];
      if (u) onpreviewunique?.(u.name);
      return;
    }
    const b = hovered === null ? choice.base : (bases[hovered] ?? choice.base);
    if (b) onpreview?.({ ...choice, base: b });
  });
</script>

<Dialog bind:open title={slot ? `Craft Item for ${slot}` : 'Craft Item'} size="md" anchor="top">
  <div class="flex flex-col gap-2.5">
    <div class="grid grid-cols-[120px_1fr] gap-2">
      <FloatingLabel label="Rarity"
        ><Select
          size="sm"
          aria-label="Rarity"
          options={RARITIES.map((r, i) => ({ value: String(i), label: r }))}
          value={String(rarity)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (rarity = Number(e.currentTarget.value))} /></FloatingLabel>
      {#if rarity >= 2 && !uniqueMode}
        <Input size="sm" aria-label="Name" placeholder="Name" bind:value={title} />
      {/if}
    </div>
    {#if uniqueMode}
      <FloatingLabel label="Unique"
        ><Dropdown
          class="block h-7.5 w-full"
          searchable
          aria-label="Unique"
          aria-busy={!uniqueRows}
          disabled={!shownUniques.length}
          options={shownUniques.map((u, i) => ({ value: String(i), label: u.title || u.name, title: u.base }))}
          value={String(unique)}
          onhover={(i: number | null, el: HTMLElement | null) => {
            hovered = i;
            hoveredEl = el;
          }}
          onchange={(v: string) => {
            unique = Number(v);
          }} /></FloatingLabel>
    {:else}
      <FloatingLabel label="Type"
        ><Dropdown
          class="block h-7.5 w-full"
          searchable
          aria-label="Type"
          options={groups.map((g, i) => ({ value: String(i), label: g.type }))}
          value={String(type)}
          onchange={(v: string) => {
            type = Number(v);
            base = 0;
          }} /></FloatingLabel>
      <FloatingLabel label="Base"
        ><Dropdown
          class="block h-7.5 w-full"
          searchable
          aria-label="Base"
          options={bases.map((b, i) => ({ value: String(i), label: b }))}
          value={String(base)}
          onhover={(i: number | null, el: HTMLElement | null) => {
            hovered = i;
            hoveredEl = el;
          }}
          onchange={(v: string) => {
            base = Number(v);
          }} /></FloatingLabel>
    {/if}
    {#if preview && hoveredEl}
      <div aria-label="Base preview" role="group"><FloatingItemTip anchor={hoveredEl} id="{uid}-tip" {...preview} /></div>
    {:else if preview}
      <div aria-label="Base preview" role="group" class="self-start"><ItemTooltip {...preview} /></div>
    {/if}
  </div>
  {#snippet actions()}
    <DialogActions
      confirm="Create"
      disabled={uniqueMode ? !shownUniques[unique] : !choice.base}
      oncancel={() => (open = false)}
      onconfirm={() => {
        const u = shownUniques[unique];
        if (uniqueMode && u) oncreateunique?.(u.name);
        else oncreate(choice);
        open = false;
      }} />
  {/snippet}
</Dialog>
