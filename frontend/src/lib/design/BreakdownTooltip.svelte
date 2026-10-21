<script lang="ts">
  import { tone as toneClass, type Tone } from './tone';
  import BreakdownSteps from './BreakdownSteps.svelte';
  import type { Snippet } from 'svelte';

  type Source = { src: string; val: string; tone?: Tone };
  type Props = {
    title: string;
    value: string;
    tone?: Tone;
    formula?: string;
    groups: { title?: string; rows: Source[] }[];
    lines?: string[];
    footer?: string;
    compact?: boolean;
    id?: string;
    class?: string;
    children?: Snippet;
  };
  let {
    title,
    value,
    tone,
    formula,
    groups,
    lines,
    footer,
    compact = false,
    id,
    class: extra = 'relative w-[290px] rounded-lg px-3 py-2.5',
    children
  }: Props = $props();
</script>

<div role="tooltip" {id} class={['tooltip flex flex-col gap-0.75', extra]}>
  <div class={['kv', compact ? 'gap-0' : 'gap-2.5']}>
    <span class={['font-semibold', compact ? 'text-12' : 'text-13']}>{title}</span><span class={['toned font-mono', toneClass(tone)]}>{value}</span>
  </div>
  {#if formula}<span class="font-mono text-text-2">{formula}</span>{/if}
  {#if lines?.length}<div class="mt-1 text-12"><BreakdownSteps {lines} /></div>{/if}
  {@render children?.()}{#each groups as g, i (i)}
    {#if g.title}<span class="mt-1 text-10 tracking-caps text-text-3">{g.title}</span>{/if}
    {#each g.rows as x, j (j)}
      <div class={['kv', compact ? 'gap-2' : 'gap-2.5']}><span class={['toned', toneClass(x.tone)]}>{x.src}</span><span class="font-mono">{x.val}</span></div>
    {/each}
  {/each}{#if footer}<span class={['text-text-3', compact ? 'mt-0' : 'mt-0.75']}>{footer}</span>{/if}
</div>
