<script lang="ts">
  import type { StepDelta } from '$lib/go/step_labels';
  import { DELTA_SHOWN } from './step_kinds';

  // the stat change of one undo step, as the stats strip shows an edit's change
  type Props = { delta: StepDelta[] };
  let { delta }: Props = $props();
</script>

{#if delta.length}
  <ul class="m-0 flex list-none flex-col gap-0.5 p-0" aria-label="Stat change">
    {#each delta.slice(0, DELTA_SHOWN) as d, i (i)}
      <li class="flex justify-between gap-3">
        <span class="text-text-2">{d.section === 'Minion' ? 'Minion: ' : ''}{d.label}</span><span class={['font-mono', d.better ? 'text-better' : 'text-worse']}
          >{d.value}</span>
      </li>
    {/each}
    {#if delta.length > DELTA_SHOWN}<li class="text-text-3">+{delta.length - DELTA_SHOWN} more</li>{/if}
  </ul>
{:else}
  <p class="m-0 text-text-3">No headline stat changed</p>
{/if}
