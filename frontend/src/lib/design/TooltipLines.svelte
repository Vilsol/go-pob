<script lang="ts">
  import PobText from './PobText.svelte';

  // GemTooltip.lua:217
  type Line = { Text: string; Separator: boolean };
  type Props = { lines: Line[]; center?: boolean; label?: string };
  let { lines, center = false, label }: Props = $props();
</script>

<div class={['flex flex-col gap-0.5 text-13', center && 'items-center text-center']} aria-label={label} data-tooltip-lines>
  {#each lines as l, i (i)}
    {#if l.Separator}<hr class="my-1 w-full border-line" />{:else}<span data-line><PobText text={l.Text} /></span>{/if}
  {/each}
</div>
