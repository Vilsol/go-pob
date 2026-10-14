<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import Dropdown from './Dropdown.svelte';
  import PobText from './PobText.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import { plainText, type AffixControls, type CraftApi } from './crafting';
  import type { ImpactBlock } from './items';

  // ItemsTab.lua:717-981
  // ItemsTab.lua:2219-2297
  type Props = { affixes: AffixControls; craft: CraftApi };
  let { affixes, craft }: Props = $props();

  let stats = $state.raw<{ stat: string; label: string }[]>([]);
  let sort = $state('');
  let powers = $state.raw<number[][]>();
  let sorting = $state(false);
  let focus = $state<number>();
  let roll = $state.raw<{ slot: number; lines: string[] }>();
  let compare = $state.raw<ImpactBlock[]>([]);
  let dropped = $state.raw<{ base: string; mods: string[] }>();

  $effect(() => {
    void craft.sortStats().then((s) => (stats = s));
  });

  $effect(() => {
    const stat = sort;
    const a = affixes;
    powers = undefined;
    if (!stat || !a.sorting) return;
    sorting = true;
    void Promise.all(a.controls.map((_, i) => craft.affixPowers?.(i, stat) ?? Promise.resolve([]))).then((p) => {
      if (stat === sort && a === affixes) powers = p;
      sorting = false;
    });
  });

  const orderOf = (i: number) => {
    const n = affixes.controls[i].series.length;
    const idx = Array.from({ length: n }, (_, k) => k);
    const p = powers?.[i];
    if (!p) return idx;
    return [0, ...idx.slice(1).sort((a, b) => (p[b] ?? 0) - (p[a] ?? 0))];
  };

  const focused = $derived(focus === undefined ? undefined : affixes.controls[focus]?.series[affixes.controls[focus].sel]);
  $effect(() => {
    const s = focused;
    compare = [];
    if (!s?.mid.length) return;
    void craft.modCompare?.($state.snapshot(s.mid)).then((b) => {
      if (s === focused) compare = b;
    });
  });
</script>

<div role="group" aria-label="Affixes" class="flex flex-col gap-1.5">
  {#if affixes.bases && affixes.bases.length > 1 && affixes.base}
    <FloatingLabel label="Base"
      ><Dropdown
        class="block h-7.5 w-full"
        searchable
        aria-label="Item base"
        options={affixes.bases.map((b) => ({ value: b, label: b }))}
        value={affixes.base}
        onchange={(v: string) => {
          void craft.rebase?.(v).then((mods) => (dropped = mods?.length ? { base: v, mods } : undefined));
        }} /></FloatingLabel>
    {#if dropped && dropped.base === affixes.base}
      <div role="status" aria-label="Dropped affixes" class="note flex flex-col">
        <span>{affixes.base} cannot roll, so they were removed:</span>
        {#each dropped.mods as m, k (k)}<span>{plainText(m)}</span>{/each}
      </div>
    {/if}
  {/if}
  {#if affixes.sorting}
    <FloatingLabel label="Modifier sorting"
      ><Select
        class="h-7.5"
        aria-label="Modifier sorting"
        aria-busy={sorting}
        options={[{ value: '', label: 'Default' }, ...stats.map((s) => ({ value: s.stat, label: s.label }))]}
        bind:value={sort} /></FloatingLabel>
  {/if}
  {#each affixes.controls as c, i (i)}
    <div class="contents" onfocusin={() => (focus = i)}>
      <FloatingLabel label="{c.type}:"
        ><Dropdown
          class="block h-7.5 w-full"
          searchable
          aria-label="Affix {i + 1}"
          options={orderOf(i).map((k) => ({ value: String(k), label: plainText(c.series[k].label) }))}
          value={String(c.sel)}
          onchange={(v: string) => {
            focus = i;
            void craft.affixEdit?.(i, Number(v), c.sliderVal);
          }} /></FloatingLabel>
    </div>
    {#if c.sliderShown}
      <label class="field field-inline gap-2 text-12"
        >Roll<input
          type="range"
          class="w-60"
          min="0"
          max="1"
          step="0.001"
          aria-label="Roll of affix {i + 1}"
          value={c.sliderVal}
          oninput={async (e) => {
            const v = Number(e.currentTarget.value);
            roll = { slot: i, lines: (await craft.affixRollTooltip?.(i, v)) ?? [] };
          }}
          onchange={(e) => void craft.affixEdit?.(i, c.sel, Number(e.currentTarget.value))} /></label>
      {#if roll?.slot === i}
        <div role="tooltip" aria-label="Roll of affix {i + 1}" class="tooltip flex flex-col gap-0.5 rounded-md px-3 py-2 text-12">
          {#each roll.lines as l, k (k)}<span><PobText text={l} /></span>{/each}
        </div>
      {/if}
    {/if}
  {/each}
  {#if focused && focus !== undefined && affixes.controls[focus].sel > 0}
    <div role="group" aria-label="Affix comparison" class="flex flex-col gap-0.5 font-mono text-12">
      {#each focused.tooltip as l, k (k)}
        <!-- ItemsTab.lua:860-914 -->
        {#if l === '--'}<hr class="my-1 border-line" />{:else}<span class="min-h-2 font-sans"><PobText text={l} /></span>{/if}
      {/each}
      <ImpactBlocks blocks={compare} />
    </div>
  {/if}
</div>
