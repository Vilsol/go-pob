<script lang="ts">
  import type { Snippet } from 'svelte';
  import { tone as toneClass } from './tone';
  import { impactText, type ImpactBlock } from './items';
  import PobText from './PobText.svelte';

  // ItemsTab.lua:4997-5001
  type Props = { blocks: ImpactBlock[]; action?: Snippet<[ImpactBlock]> };
  let { blocks, action }: Props = $props();
</script>

{#each blocks as b, i (i)}
  <div class="flex flex-col gap-1" data-header={b.header} data-slot={b.slot} data-ready-at={b.readyAt} data-calc-ms={b.calcMs}>
    {#if b.lines?.length}
      <div class="flex flex-col gap-0.5 font-sans text-12" role="group" aria-label="Effective flask stats">
        {#each b.lines as l, k (k)}<span><PobText text={l} /></span>{/each}
      </div>
    {/if}
    <div class="flex items-start justify-between gap-2">
      <span class="font-sans text-12 whitespace-pre-line text-text-2" data-block-header>{b.header}</span>
      {#if action}{@render action(b)}{/if}
    </div>
    {#each b.sections as s, j (j)}
      <div class="flex flex-col gap-0.75" data-section data-actor={s.actor}>
        {#if s.actor}<div class="subhead font-sans">{s.actor}</div>{/if}
        {#each s.lines as l, k (k)}
          <div class="kv" data-line><span>{l.label}</span><span class={['toned', toneClass(l.better ? 'up' : 'down')]}>{impactText(l)}</span></div>
        {/each}
      </div>
    {/each}
    {#if !b.sections.length}<span class="note font-sans">No stat changes.</span>{/if}
  </div>
{/each}
