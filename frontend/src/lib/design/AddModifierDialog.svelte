<script lang="ts">
  import Input from './Input.svelte';
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import Dropdown from './Dropdown.svelte';
  import ItemTooltip from './ItemTooltip.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import PobText from './PobText.svelte';
  import FloatingModTip from './FloatingModTip.svelte';
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

  // ItemsTab.lua:3842-3849
  const uid = $props.id();
  let hover = $state.raw<{ el: HTMLElement; lines: string[]; blocks: ImpactBlock[]; pending: boolean }>();
  let hoverSeq = 0;
  const hoverMod = (index: number | null, el: HTMLElement | null) => {
    const seq = ++hoverSeq;
    const m = index === null ? undefined : mods[order[index]];
    if (!el || !m) {
      hover = undefined;
      return;
    }
    hover = { el, lines: m.lines.map((l) => '^7' + l), blocks: [], pending: !!craft.modCompare };
    craft
      .modCompare?.(m.lines.map((l) => (m.modTags.length ? `{tags:${m.modTags.join(',')}}` : '') + l))
      .then((blocks) => {
        if (seq === hoverSeq && hover) hover = { ...hover, blocks, pending: false };
      })
      .catch(() => {
        if (seq === hoverSeq && hover) hover = { ...hover, pending: false };
      });
  };

  const result = () => (sourceId === 'CUSTOM' ? craft.customModText?.('CUSTOM', '', custom) : mod ? craft.customModText?.(sourceId, mod.id, '') : undefined);
  $effect(() => {
    void [sourceId, mod, custom, choices];
    if (!open || !choices) return;
    void result()?.then(async (t) => {
      preview = t === undefined ? undefined : await craft.tipOf?.(t);
    });
  });
</script>

<Dialog bind:open title="Add Modifier to Item" size="lg">
  <div class="flex flex-col gap-2.5 text-13">
    <div class="grid grid-cols-2 gap-3">
      <FloatingLabel label="Source"
        ><Select
          size="sm"
          aria-label="Source"
          disabled={(choices?.sources.length ?? 0) <= 1}
          options={(choices?.sources ?? []).map((s, i) => ({ value: String(i), label: s.label }))}
          value={String(source)}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (source = Number(e.currentTarget.value))} /></FloatingLabel>
      {#if sourceId !== 'CUSTOM'}
        <FloatingLabel label="Sort by"
          ><Select
            size="sm"
            aria-label="Sort by"
            aria-busy={sorting}
            options={[{ value: '', label: 'Default' }, ...stats.map((s) => ({ value: s.stat, label: s.label }))]}
            bind:value={sort} /></FloatingLabel>
      {/if}
    </div>
    {#if sourceId === 'CUSTOM'}
      <Input size="sm" aria-label="Custom modifier" bind:value={custom} />
    {:else}
      <FloatingLabel label="Modifier"
        ><Dropdown
          class="block h-7.5 w-full"
          searchable
          aria-label="Modifier"
          options={order.map((k, i) => ({ value: String(i), label: plainText(mods[k].label) }))}
          value={String(pick)}
          onhover={hoverMod}
          onchange={(v: string) => {
            pick = Number(v);
          }} /></FloatingLabel>
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
    {#if hover}
      <FloatingModTip anchor={hover.el} id="{uid}-tip" label="Modifier tooltip" lines={hover.lines} blocks={hover.blocks} pending={hover.pending} />
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
