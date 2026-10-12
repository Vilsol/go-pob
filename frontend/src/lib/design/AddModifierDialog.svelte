<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import ItemTooltip from './ItemTooltip.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import PobText from './PobText.svelte';
  import { plainText, sortByValue, type CraftApi, type CustomModChoices } from './crafting';
  import type { ImpactBlock, TipProps } from './items';

  // ItemsTab.lua:3344-3659
  type Props = { open?: boolean; craft: CraftApi };
  let { open = $bindable(false), craft }: Props = $props();

  let choices = $state.raw<CustomModChoices>();
  let stats = $state.raw<{ stat: string; label: string }[]>([]);
  let source = $state(0);
  let sort = $state('');
  let values = $state.raw<Record<string, number>>();
  let sorting = $state(false);
  let pick = $state(0);
  let custom = $state('');
  let compare = $state.raw<ImpactBlock[]>([]);
  let preview = $state.raw<TipProps>();

  $effect(() => {
    if (!open) return;
    source = pick = 0;
    sort = custom = '';
    void craft.customModChoices?.().then((c) => (choices = c));
    void craft.sortStats().then((s) => (stats = s));
  });

  const sourceId = $derived(choices?.sources[source]?.id ?? 'CUSTOM');
  const mods = $derived(sourceId === 'CUSTOM' ? [] : (choices?.mods[sourceId] ?? []));
  const order = $derived(sortByValue(sort && values ? mods.map((m) => values![m.id] ?? 0) : undefined, mods.length));
  const mod = $derived(mods[order[pick]]);

  $effect(() => {
    const stat = sort;
    const id = sourceId;
    const ids = mods.map((m) => m.id);
    values = undefined;
    pick = 0;
    if (!stat || !ids.length) return;
    sorting = true;
    void craft.customModSort?.(id, ids, stat).then((v) => {
      if (stat === sort && id === sourceId) values = Object.fromEntries(ids.map((x, k) => [x, v[k] ?? 0]));
      sorting = false;
    });
  });

  $effect(() => {
    const m = mod;
    compare = [];
    if (!m || !open) return;
    void craft.modCompare?.(m.lines.map((l) => (m.modTags.length ? `{tags:${m.modTags.join(',')}}` : '') + l)).then((b) => {
      if (m === mod) compare = b;
    });
  });

  const result = () => (sourceId === 'CUSTOM' ? craft.customModText?.('CUSTOM', '', custom) : mod ? craft.customModText?.(sourceId, mod.id, '') : undefined);
  $effect(() => {
    void [sourceId, mod, custom, choices];
    if (!open || !choices) return;
    void result()?.then(async (t) => {
      preview = t === undefined ? undefined : await craft.tipOf?.(t);
    });
  });
</script>

<Dialog bind:open title="Add Modifier to Item" class="w-160 p-4.5">
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
      <input class="input h-7.5" aria-label="Custom modifier" bind:value={custom} />
    {:else}
      <FloatingLabel label="Modifier"
        ><Select
          class="h-7.5"
          aria-label="Modifier"
          options={order.map((k, i) => ({ value: String(i), label: plainText(mods[k].label) }))}
          value={String(pick)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (pick = Number(e.currentTarget.value))} /></FloatingLabel>
      {#if mod}
        <div role="group" aria-label="Modifier comparison" class="flex flex-col gap-1 font-mono text-13">
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
    <Button
      size="md"
      variant="primary"
      onclick={async () => {
        const t = await result();
        if (t !== undefined) craft.settext?.(t);
        open = false;
      }}>Add</Button>
    <Button size="md" onclick={() => (open = false)}>Cancel</Button>
  {/snippet}
</Dialog>
