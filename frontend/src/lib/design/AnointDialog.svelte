<script lang="ts">
  import Tooltip from './Tooltip.svelte';
  import SearchList from './SearchList.svelte';
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import Select from './Select.svelte';
  import PobText from './PobText.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import { finitePowers, maxOf, notableColour, notableMatches, sortNotables, type AnointComparison, type AnointNotable, type CraftApi } from './crafting';
  import type { DisplayEdit } from './items';

  // ItemsTab.lua:2978-3015
  type Props = { open?: boolean; slot: number; anointed: string[]; craft: CraftApi; onedit: (edit: DisplayEdit) => void };
  let { open = $bindable(false), slot, anointed, craft, onedit }: Props = $props();

  const MODES = ['Anywhere', 'Names', 'Modifiers'] as const;
  const CHUNK = 20;

  let notables = $state.raw<AnointNotable[]>([]);
  let stats = $state.raw<{ stat: string; label: string }[]>([]);
  let sort = $state('');
  let search = $state('');
  let mode = $state<(typeof MODES)[number]>('Anywhere');
  let selected = $state.raw<AnointNotable>();
  let power = $state.raw<Map<number, number>>();
  let progress = $state<number>();
  let comparison = $state.raw<AnointComparison>();

  $effect(() => {
    if (!open) return;
    sort = search = '';
    mode = 'Anywhere';
    selected = undefined;
    void craft.notables().then((n) => (notables = n));
    void craft.sortStats().then((s) => (stats = s));
  });

  const filtered = $derived(notables.filter((n) => notableMatches(n, search, mode)));
  const list = $derived(sortNotables(filtered, sort ? power : undefined));

  $effect(() => {
    const stat = sort;
    const ids = filtered.map((n) => n.id);
    power = undefined;
    progress = undefined;
    if (!stat || !open) return;
    const run = { live: true };
    const out = new Map<number, number>();
    void (async () => {
      for (let i = 0; i < ids.length && run.live; i += CHUNK) {
        progress = Math.floor((i / ids.length) * 100);
        const part = ids.slice(i, i + CHUNK);
        const v = await craft.anointPowers(slot, part, stat);
        part.forEach((id, k) => out.set(id, v[k] ?? 0));
      }
      if (run.live) {
        power = finitePowers(out);
        progress = undefined;
      }
    })();
    return () => {
      run.live = false;
    };
  });

  $effect(() => {
    const id = selected?.id ?? 0;
    comparison = undefined;
    if (!open) return;
    void craft.anointCompare(slot, id).then((c) => {
      if ((selected?.id ?? 0) === id) comparison = c;
    });
  });

  // ItemsTab.lua:2984-2994
  const saveLabel = $derived(selected ? `Anoint ${selected.name}` : anointed.length >= slot ? `Remove ${anointed[slot - 1]}` : 'No Anoint');
</script>

<Dialog bind:open title="Anoint Item" size="md">
  <div class="flex flex-col gap-1.5 text-13">
    <Select
      size="sm"
      aria-label="Sort notables"
      options={[{ value: '', label: 'Sort by Name' }, ...stats.map((s) => ({ value: s.stat, label: `Sort by ${s.label}` }))]}
      bind:value={sort} />
    <SearchList
      bind:query={search}
      searchLabel="Search notables"
      items={list}
      key={(n: AnointNotable) => n.id}
      label="Notables"
      busy={progress !== undefined}
      selected={selected ? list.findIndex((n: AnointNotable) => n.id === selected?.id) : -1}
      emptyText="No notables found that match those filters."
      listClass="h-70"
      onselect={(n: AnointNotable) => (selected = n)}>
      {#snippet filters()}<Select size="xs" aria-label="Search in" options={[...MODES]} bind:value={mode} />{/snippet}
      {#snippet busyNote()}<span class="note px-2 py-0.5">Sorting... ({progress}%)</span>{/snippet}
      {#snippet row(n: AnointNotable)}<PobText text={notableColour(power?.get(n.id), power && maxOf(power), !!sort && !!power) + n.name} />{/snippet}
    </SearchList>
    {#if selected}
      <Tooltip label="Notable" interactive flat class="flex flex-col gap-0.5">
        <span class="font-semibold">{selected.name}</span>
        {#each selected.stats as line, i (i)}<span data-unsupported={selected.unsupported?.[i] || undefined}
            ><PobText text={(selected.unsupported?.[i] ? '^xF05050' : '^x8888FF') + line} /></span
          >{/each}
        {#each selected.reminder as line, i (i)}<span><PobText text={'^xA0A080' + line} /></span>{/each}
      </Tooltip>
    {/if}
    {#if comparison?.note}
      <p class="note" role="status"><PobText text={comparison.note} /></p>
    {:else if comparison}
      <section aria-label="Anoint comparison" class="flex flex-col gap-2 font-mono text-13"><ImpactBlocks blocks={comparison.blocks} /></section>
    {/if}
  </div>
  {#snippet actions()}
    <DialogActions
      confirm={saveLabel}
      oncancel={() => (open = false)}
      onconfirm={() => {
        onedit({ kind: 'anoint', index: slot, text: selected?.name ?? '' });
        open = false;
      }} />
  {/snippet}
</Dialog>
