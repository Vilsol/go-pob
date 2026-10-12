<script lang="ts">
  import ItemTooltip from './ItemTooltip.svelte';
  import TooltipLines from './TooltipLines.svelte';
  import { besideAnchor, type TipProps } from './items';

  // ItemsTab.lua:4748-4794
  type Props = TipProps & { anchor: HTMLElement | undefined; id: string };
  let { anchor, id, ...tip }: Props = $props();
  let shift = $state(false);
  const children = $derived((tip.tooltip?.children ?? []).filter((c) => c.length));
</script>

<svelte:window onkeydown={(e: KeyboardEvent) => (shift = e.shiftKey)} onkeyup={(e: KeyboardEvent) => (shift = e.shiftKey)} onblur={() => (shift = false)} />
<div class="pointer-events-none fixed top-(--y) left-(--x) z-30 flex items-start gap-1" use:besideAnchor={anchor}>
  <div class="rounded-md shadow-tip"><ItemTooltip {id} {...tip} /></div>
  {#if shift}
    {#each children as c, i (i)}
      <div role="group" aria-label="Granted skill" class="tooltip w-[300px] rounded-md px-3 py-1.5 shadow-tip">
        <TooltipLines center lines={c.map((l) => ({ Text: l.text, Separator: l.sep }))} />
      </div>
    {/each}
  {/if}
</div>
