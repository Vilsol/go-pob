<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import Dropdown from './Dropdown.svelte';
  import ItemTooltip from './ItemTooltip.svelte';
  import type { CraftChoice, TipProps } from './items';

  // ItemsTab.lua:2519-2603
  type Props = {
    open?: boolean;
    groups: { type: string; bases: string[] }[];
    start?: string;
    preview?: TipProps;
    onpreview?: (c: CraftChoice) => void;
    oncreate: (c: CraftChoice) => void;
  };
  let { open = $bindable(false), groups, start, preview, onpreview, oncreate }: Props = $props();

  const RARITIES = ['Normal', 'Magic', 'Rare', 'Unique', 'Relic'];
  let rarity = $state(2);
  let title = $state('');
  let type = $state(0);
  let base = $state(0);
  const bases = $derived(groups[type]?.bases ?? []);
  $effect(() => {
    if (!open || !start) return;
    const t = groups.findIndex((g) => g.bases.includes(start));
    if (t < 0) return;
    type = t;
    base = groups[t].bases.indexOf(start);
  });
  const choice = $derived({ rarity, title, base: bases[base] ?? '' });
  let hovered = $state<number | null>(null);
  $effect(() => {
    const b = hovered === null ? choice.base : (bases[hovered] ?? choice.base);
    if (open && b) onpreview?.({ ...choice, base: b });
  });
</script>

<Dialog bind:open title="Craft Item" class="w-110 p-4.5">
  <div class="flex flex-col gap-2.5">
    <div class="grid grid-cols-[120px_1fr] gap-2">
      <FloatingLabel label="Rarity"
        ><Select
          class="h-7.5"
          aria-label="Rarity"
          options={RARITIES.map((r, i) => ({ value: String(i), label: r }))}
          value={String(rarity)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (rarity = Number(e.currentTarget.value))} /></FloatingLabel>
      {#if rarity >= 2}
        <input class="input h-7.5" aria-label="Name" placeholder="Name" bind:value={title} />
      {/if}
    </div>
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
        onhover={(i: number | null) => (hovered = i)}
        onchange={(v: string) => {
          base = Number(v);
        }} /></FloatingLabel>
    {#if preview}
      <div aria-label="Base preview" role="group" class="self-start"><ItemTooltip {...preview} /></div>
    {/if}
  </div>
  {#snippet actions()}
    <Button
      size="md"
      variant="primary"
      disabled={!choice.base}
      onclick={() => {
        oncreate(choice);
        open = false;
      }}>Create</Button>
    <Button size="md" onclick={() => (open = false)}>Cancel</Button>
  {/snippet}
</Dialog>
