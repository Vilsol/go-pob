<script lang="ts">
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import Dropdown from './Dropdown.svelte';
  import ItemTooltip from './ItemTooltip.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import PobText from './PobText.svelte';
  import { sortByValue, type CraftApi, type ImplicitChoices } from './crafting';
  import type { ImpactBlock, TipProps } from './items';

  type Props = { open?: boolean; craft: CraftApi };
  let { open = $bindable(false), craft }: Props = $props();

  let choices = $state.raw<ImplicitChoices>();
  let stats = $state.raw<{ stat: string; label: string }[]>([]);
  let source = $state(0);
  let sort = $state('');
  let values = $state.raw<Record<string, number>>();
  let sorting = $state(false);
  let group = $state(0);
  let pick = $state(0);
  let custom = $state('');
  let compare = $state.raw<ImpactBlock[]>([]);
  let preview = $state.raw<TipProps>();

  $effect(() => {
    if (!open) return;
    source = group = pick = 0;
    sort = custom = '';
    void craft.implicitChoices?.().then((c) => (choices = c));
    void craft.sortStats().then((s) => (stats = s));
  });

  const sourceId = $derived(choices?.sources[source]?.id ?? 'CUSTOM');
  const groups = $derived(sourceId === 'CUSTOM' ? [] : (choices?.groups[sourceId] ?? []));
  const groupOrder = $derived(sortByValue(sort && values ? groups.map((g) => Math.max(...g.mods.map((m) => values![m.id] ?? 0))) : undefined, groups.length));
  const current = $derived(groups[groupOrder[group]]);
  const modOrder = $derived(sortByValue(sort && values && current ? current.mods.map((m) => values![m.id] ?? 0) : undefined, current?.mods.length ?? 0));
  const mod = $derived(current?.mods[modOrder[pick]]);

  $effect(() => {
    const stat = sort;
    const id = sourceId;
    const ids = groups.flatMap((g) => g.mods.map((m) => m.id));
    values = undefined;
    group = pick = 0;
    if (!stat || !ids.length) return;
    sorting = true;
    void craft.implicitSort?.(id, ids, stat).then((v) => {
      if (stat === sort && id === sourceId) values = Object.fromEntries(ids.map((x, k) => [x, v[k] ?? 0]));
      sorting = false;
    });
  });

  $effect(() => {
    const m = mod;
    compare = [];
    if (!m || !open) return;
    const lines = m.lines.map((l) => (m.modTags.length ? `{tags:${m.modTags.join(',')}}` : '') + l);
    void craft.modCompare?.(lines, sourceId === 'VESTIGIAL').then((b) => {
      if (m === mod) compare = b;
    });
  });

  const result = () => (sourceId === 'CUSTOM' ? craft.implicitText?.('CUSTOM', '', custom) : mod ? craft.implicitText?.(sourceId, mod.id, '') : undefined);
  $effect(() => {
    void [sourceId, mod, custom, choices];
    if (!open || !choices) return;
    void result()?.then(async (t) => {
      preview = t === undefined ? undefined : await craft.tipOf?.(t);
    });
  });
</script>

<Dialog bind:open title="Add Implicit to Item" class="w-160 p-4.5">
  <div class="flex flex-col gap-2.5 text-13">
    <div class="grid grid-cols-2 gap-3">
      <FloatingLabel label="Source"
        ><Select
          class="h-7.5"
          aria-label="Source"
          disabled={(choices?.sources.length ?? 0) <= 1}
          options={(choices?.sources ?? []).map((s, i) => ({ value: String(i), label: s.label }))}
          value={String(source)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (source = Number(e.currentTarget.value))} /></FloatingLabel>
      {#if sourceId !== 'CUSTOM'}
        <FloatingLabel label="Sort by"
          ><Select
            class="h-7.5"
            aria-label="Sort by"
            aria-busy={sorting}
            options={[{ value: '', label: 'Default' }, ...stats.map((s) => ({ value: s.stat, label: s.label }))]}
            bind:value={sort} /></FloatingLabel>
      {/if}
    </div>
    {#if sourceId === 'CUSTOM'}
      <input class="input h-7.5" aria-label="Custom implicit" bind:value={custom} />
    {:else}
      <FloatingLabel label="Type"
        ><Select
          class="h-7.5"
          aria-label="Type"
          options={groupOrder.map((k, i) => ({ value: String(i), label: groups[k].label.replace(/\^x[0-9A-Fa-f]{6}|\^\d/g, '') }))}
          value={String(group)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
            group = Number(e.currentTarget.value);
            pick = 0;
          }} /></FloatingLabel>
      {#if sourceId !== 'VESTIGIAL'}
        <FloatingLabel label="Modifier"
          ><Dropdown
            class="block h-7.5 w-full"
            searchable
            aria-label="Modifier"
            options={modOrder.map((k, i) => ({ value: String(i), label: current.mods[k].label }))}
            value={String(pick)}
            onchange={(v: string) => {
              pick = Number(v);
            }} /></FloatingLabel>
      {/if}
      {#if mod}
        <div role="group" aria-label="Implicit comparison" class="flex flex-col gap-1 font-mono text-13">
          {#each mod.lines as l, k (k)}<span class="font-sans"><PobText text={'^7' + l} /></span>{/each}
          <ImpactBlocks blocks={compare} />
        </div>
      {/if}
    {/if}
    {#if preview}
      <div aria-label="New item" role="group" class="self-start"><ItemTooltip {...preview} /></div>
    {/if}
  </div>
  {#snippet actions()}
    <DialogActions
      confirm="Add"
      oncancel={() => (open = false)}
      onconfirm={async () => {
        const t = await result();
        if (t !== undefined) craft.settext?.(t);
        open = false;
      }} />
  {/snippet}
</Dialog>
