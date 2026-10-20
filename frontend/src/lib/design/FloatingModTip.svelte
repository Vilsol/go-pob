<script lang="ts">
  import ImpactBlocks from './ImpactBlocks.svelte';
  import PobText from './PobText.svelte';
  import Tooltip from './Tooltip.svelte';
  import type { ImpactBlock } from './items';

  // ItemsTab.lua:818-930
  type Props = { anchor: HTMLElement | undefined; id: string; label: string; lines: string[]; blocks: ImpactBlock[]; pending: boolean };
  let { anchor, id, label, lines, blocks, pending }: Props = $props();
</script>

<Tooltip {id} {label} aria-busy={pending} placement="beside" {anchor} class="flex w-max max-w-110 flex-col gap-0.5">
  {#each lines as l, k (k)}
    {#if l === '--'}<hr class="my-1 border-line" />{:else}<span class="min-h-2"><PobText text={l} /></span>{/if}
  {/each}
  <div class="font-mono"><ImpactBlocks {blocks} /></div>
  {#if pending}<span class="text-text-3">Calculating…</span>{/if}
</Tooltip>
