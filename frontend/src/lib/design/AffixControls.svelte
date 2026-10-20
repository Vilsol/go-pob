<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import Dropdown from './Dropdown.svelte';
  import PobText from './PobText.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import FloatingModTip from './FloatingModTip.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import { plainText, type AffixControls, type CraftApi } from './crafting';
  import type { ImpactBlock, TipProps } from './items';
  import type { Snippet } from 'svelte';

  // ItemsTab.lua:717-981
  // ItemsTab.lua:2219-2297
  type Props = {
    affixes: AffixControls;
    craft: CraftApi;
    slots?: number[][];
    changes?: number[][];
    lineId?: (index: number) => string;
    line?: Snippet<[index: number, preview: string[] | undefined, also?: boolean]>;
  };
  let { affixes, craft, slots, changes, lineId, line }: Props = $props();

  let stats = $state.raw<{ stat: string; label: string }[]>([]);
  let sort = $state('');
  let powers = $state.raw<number[][]>();
  let sorting = $state(false);
  let focus = $state<number>();
  let roll = $state.raw<{ slot: number; lines: string[] }>();
  let preview = $state.raw<{ slot: number; lines: string[]; from: AffixControls }>();
  let dragged = $state.raw<{ slot: number; value: number; from: AffixControls }>();
  const percent = (v: number) => `${Math.round(v * 100)}%`;
  const GROUP: Record<string, string> = { Prefix: 'Prefixes', Suffix: 'Suffixes' };
  const groups = $derived.by(() => {
    const out: { type: string; label: string; slots: number[] }[] = [];
    affixes.controls.forEach((c, i) => {
      const g = out.find((x) => x.type === c.type) ?? out[out.push({ type: c.type, label: GROUP[c.type] ?? c.type, slots: [] }) - 1];
      g.slots.push(i);
    });
    return out;
  });
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

  // ItemsTab.lua:818-930
  const uid = $props.id();
  let hover = $state.raw<{ el: HTMLElement; lines?: string[]; blocks: ImpactBlock[]; pending: boolean; item?: TipProps }>();
  let hoverSeq = 0;
  let compares = new Map<string, Promise<ImpactBlock[]>>();
  $effect(() => {
    void affixes;
    compares = new Map();
  });
  const compareOf = (mid: string[]) => {
    const key = mid.join('\n');
    let p = compares.get(key);
    if (!p) compares.set(key, (p = craft.modCompare?.($state.snapshot(mid)) ?? Promise.resolve([])));
    return p;
  };
  const hoverAffix = (i: number) => (index: number | null, el: HTMLElement | null) => {
    const seq = ++hoverSeq;
    const k = index === null ? 0 : orderOf(i)[index];
    const s = affixes.controls[i].series[k];
    if (!el || !k || !s) {
      hover = undefined;
      return;
    }
    hover = { el, lines: s.tooltip, blocks: [], pending: !!s.mid.length };
    if (!s.mid.length) return;
    compareOf(s.mid)
      .then((blocks) => {
        if (seq === hoverSeq && hover) hover = { ...hover, blocks, pending: false };
      })
      .catch(() => {
        if (seq === hoverSeq && hover) hover = { ...hover, pending: false };
      });
  };
  const hoverBase = (index: number | null, el: HTMLElement | null) => {
    const seq = ++hoverSeq;
    const b = index === null ? undefined : affixes.bases?.[index];
    if (!el || !b || !craft.rebaseTip) {
      hover = undefined;
      return;
    }
    hover = { el, blocks: [], pending: true };
    craft
      .rebaseTip(b)
      .then((item) => {
        if (seq === hoverSeq && hover) hover = { ...hover, item, pending: false };
      })
      .catch(() => {
        if (seq === hoverSeq) hover = undefined;
      });
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

<div role="group" aria-label="Affixes" class="flex flex-col gap-2.5">
  {#if affixes.bases && affixes.bases.length > 1 && affixes.base}
    <FloatingLabel label="Base"
      ><Dropdown
        class="block h-7.5 w-full"
        searchable
        aria-label="Item base"
        options={affixes.bases.map((b) => ({ value: b, label: b }))}
        value={affixes.base}
        onhover={hoverBase}
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
  {#each groups as g (g.type)}
    <div role="group" aria-label={g.label} class="flex flex-col gap-2 border-t border-line-faint pt-2.5">
      <h4 class="caps text-11 text-text-3">{g.label}</h4>
      {#each g.slots as i, n (i)}
        {@const c = affixes.controls[i]}
        {@const ids = (changes ?? slots)?.[i]?.map((k) => lineId?.(k)).join(' ') || undefined}
        <div
          role="group"
          aria-label="{c.type} {n + 1}"
          class="flex flex-col gap-1.5 rounded-md border border-line-faint bg-surface-2 p-2"
          onfocusin={() => (focus = i)}>
          <FloatingLabel label="{c.type} {n + 1}"
            ><Dropdown
              class="block h-7.5 w-full"
              searchable
              aria-label="Affix {i + 1}"
              options={orderOf(i).map((k) => ({ value: String(k), label: plainText(c.series[k].label) }))}
              value={String(c.sel)}
              onhover={hoverAffix(i)}
              onchange={(v: string) => {
                focus = i;
                void craft.affixEdit?.(i, Number(v), c.sliderVal);
              }} /></FloatingLabel>
          {#if c.sliderShown}
            <label class="flex items-center gap-2 text-12 text-text-2"
              ><span class="w-20 shrink-0">Tier and roll</span><input
                type="range"
                class="min-w-0 grow"
                min="0"
                max="1"
                step="0.001"
                aria-label="Roll of affix {i + 1}"
                aria-describedby={ids}
                value={c.sliderVal}
                oninput={async (e) => {
                  const v = Number(e.currentTarget.value);
                  dragged = { slot: i, value: v, from: affixes };
                  const lines = (await craft.affixRollTooltip?.(i, v)) ?? [];
                  roll = { slot: i, lines };
                  preview = { slot: i, lines: lines.slice(0, lines.indexOf('') < 0 ? lines.length : lines.indexOf('')), from: affixes };
                }}
                onchange={(e) => void craft.affixEdit?.(i, c.sel, Number(e.currentTarget.value))} /><output class="w-10 shrink-0 text-right font-mono"
                >{percent(dragged?.slot === i && dragged.from === affixes ? dragged.value : c.sliderVal)}</output
              ></label>
            {#if roll?.slot === i}
              <div role="tooltip" aria-label="Roll of affix {i + 1}" class="tooltip flex flex-col gap-0.5 rounded-md px-3 py-2 text-12">
                {#each roll.lines as l, k (k)}<span><PobText text={l} /></span>{/each}
              </div>
            {/if}
          {/if}
          {#if line && (changes ?? slots)?.[i]?.length}
            <ul aria-label="Lines of affix {i + 1}" class="ml-1 flex flex-col gap-1 border-l-2 border-line-strong pl-2.5">
              {#each (changes ?? slots)?.[i] ?? [] as k (k)}{@render line(
                  k,
                  preview?.slot === i && preview.from === affixes ? preview.lines : undefined,
                  !slots?.[i]?.includes(k)
                )}{/each}
            </ul>
          {/if}
        </div>
      {/each}
    </div>
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
{#if hover?.lines}
  <FloatingModTip anchor={hover.el} id="{uid}-tip" label="Affix tooltip" lines={hover.lines} blocks={hover.blocks} pending={hover.pending} />
{:else if hover?.item}
  <div role="group" aria-label="Base tooltip"><FloatingItemTip anchor={hover.el} id="{uid}-tip" {...hover.item} /></div>
{/if}
