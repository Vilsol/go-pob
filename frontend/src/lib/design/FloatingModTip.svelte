<script lang="ts">
  import ImpactBlocks from './ImpactBlocks.svelte';
  import PobText from './PobText.svelte';
  import { besideAnchor, type ImpactBlock } from './items';

  // ItemsTab.lua:818-930
  type Props = { anchor: HTMLElement | undefined; id: string; label: string; lines: string[]; blocks: ImpactBlock[]; pending: boolean };
  let { anchor, id, label, lines, blocks, pending }: Props = $props();
</script>

<div class="pointer-events-none fixed top-(--y) left-(--x) z-30" use:besideAnchor={anchor}>
  <div {id} role="tooltip" aria-label={label} aria-busy={pending} class="tooltip flex w-max max-w-110 flex-col gap-0.5 rounded-md px-3 py-2 text-12 shadow-tip">
    {#each lines as l, k (k)}
      {#if l === '--'}<hr class="my-1 border-line" />{:else}<span class="min-h-2"><PobText text={l} /></span>{/if}
    {/each}
    <div class="font-mono"><ImpactBlocks {blocks} /></div>
    {#if pending}<span class="text-text-3">Calculating…</span>{/if}
  </div>
</div>
