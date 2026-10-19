<script lang="ts">
  import type { Sidebar } from '$lib/display/sidebar';
  import type { StatCompareLine } from '$lib/skill_tree/stat_compare';
  import { headlines } from './mobile';

  type Props = { sidebar: Sidebar; pinned?: string[]; delta?: StatCompareLine[]; onopen: () => void };
  let { sidebar, pinned = [], delta = [], onopen }: Props = $props();

  const shown = $derived(headlines(sidebar, pinned, delta));
  const short = (label: string) => label.replace('Effective Hit Pool', 'EHP').replace('Total ', '').replace(' inc. Ignite', '');
</script>

<button
  type="button"
  class="flex h-11 w-full shrink-0 items-stretch overflow-x-auto overflow-y-hidden border-0 border-b border-line bg-bar p-0 text-left font-sans text-text [scrollbar-width:none]"
  aria-label="Show all stats"
  aria-haspopup="dialog"
  onclick={onopen}
  data-stats-strip>
  {#each shown as h (h.section + h.line.label)}
    <span
      class="box-border flex min-w-0 shrink-0 basis-[calc((100%-1.5rem)/4)] flex-col justify-center border-r border-line px-2 last:border-r-0"
      data-strip-cell>
      <span class="flex items-baseline justify-between gap-1 text-11">
        <span class="truncate text-text-3">{h.section === 'Minion' ? 'Minion ' : ''}{short(h.line.label)}</span>
        {#if h.delta}
          <span class={['shrink-0 font-mono', h.delta.better ? 'text-better' : 'text-worse']} data-delta>{h.delta.value}</span>
        {/if}
      </span>
      <span class="truncate font-mono text-13 font-semibold">{h.line.value}</span>
    </span>
  {/each}
  <span class="sticky right-0 flex w-6 shrink-0 items-center justify-center bg-bar text-text-3" aria-hidden="true">▴</span>
</button>
