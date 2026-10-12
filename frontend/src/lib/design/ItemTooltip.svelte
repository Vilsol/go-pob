<script lang="ts">
  import { tone as toneClass, type Tone } from './tone';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import type { WhatIf } from './items';
  import type { ItemTip } from './fixtures';
  import PobText from './PobText.svelte';

  type Impact = { label: string; text: string; dir: 'up' | 'down' };
  type Props = {
    item: string;
    base: string;
    rarity: Tone;
    baseTone?: Tone;
    mods?: string[];
    flask?: { on: boolean; state: string };
    impactTitle: string;
    impactSub?: string;
    impact: Impact[];
    whatIf?: WhatIf;
    tooltip?: ItemTip;
    id?: string;
  };
  let { item, base, rarity, baseTone, mods = [], flask, impactTitle, impactSub, impact, whatIf, tooltip, id }: Props = $props();
  const lines = $derived.by(() => {
    const all = tooltip?.lines ?? [];
    let a = 0;
    let b = all.length;
    while (a < b && all[a].sep) a++;
    while (b > a && all[b - 1].sep) b--;
    return all.slice(a, b);
  });
</script>

<div
  role="tooltip"
  {id}
  class={['tooltip relative overflow-hidden', tooltip ? 'w-[360px]' : 'w-[290px]', 'rounded-md border-(--tone) text-13', toneClass(rarity)]}>
  <div class="item-head">
    {#if tooltip}
      {#if tooltip.influence.length}<span data-influence class="text-11 text-text-2">{tooltip.influence[0]}</span>{/if}
      {#each tooltip.title as t, i (i)}<span data-pob-color class={['toned', i === 0 && 'font-semibold']}>{t}</span>{/each}
      {#if tooltip.influence.length > 1}<span data-influence class="text-11 text-text-2">{tooltip.influence[1]}</span>{/if}
    {:else}
      <span data-pob-color class="toned font-semibold">{item}</span><span data-pob-color class={['text-12 text-(--tone)', baseTone && toneClass(baseTone)]}
        >{base}</span>
    {/if}
  </div>
  {#if tooltip}
    <div data-item-lines class="flex flex-col px-3 py-1.5 text-center text-12 leading-[1.3]">
      {#each lines as l, i (i)}
        {#if l.sep}<hr class="my-0.75 border-line-strong" />{:else}<span class="min-h-2"><PobText text={l.text} /></span>{/if}
      {/each}
    </div>
  {:else if mods.length}
    <div class="flex flex-col gap-0.5 px-3 py-2 text-center text-12 text-magic">
      {#each mods as m, i (i)}<span>{m}</span>{/each}
    </div>
  {/if}
  {#if flask}
    <div class={['border-t border-line-strong px-3 py-1.5 text-12 text-(--tone)', toneClass(flask.on ? 'up' : 'muted')]}>
      {flask.state}
    </div>
  {/if}
  <div class={['flex flex-col gap-0.75 px-3 py-2 font-mono text-12', (tooltip || mods.length || flask) && 'border-t border-line-strong']}>
    <div class="font-sans text-11 font-semibold tracking-caps text-text-2">{impactTitle}</div>
    {#if impactSub && !whatIf}<div class="font-sans text-11 text-text-3">{impactSub}</div>{/if}
    {#if whatIf}
      <div class="flex flex-col gap-1.5" role="group" aria-label="Stat differences" aria-busy={whatIf.pending} data-slot={whatIf.key}>
        <ImpactBlocks blocks={whatIf.blocks} />
        {#if whatIf.pending}<span class="font-sans text-11 text-text-3">Calculating…</span>
        {:else if whatIf.tip}<span class="font-sans text-11 text-text-3">{whatIf.tip}</span>
        {:else if !whatIf.blocks.length}<span class="font-sans text-11 text-text-3">Fits no slot of the active item set</span>{/if}
      </div>
    {:else}
      {#each impact as d, i (i)}
        <div class="kv"><span>{d.label}</span><span class={['toned', toneClass(d.dir)]}>{d.text}</span></div>
      {/each}
    {/if}
  </div>
</div>
