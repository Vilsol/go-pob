<script lang="ts">
  import PobText from './PobText.svelte';
  import { breakdownStep } from './breakdown_lines';

  let { lines }: { lines: string[] } = $props();
</script>

<div role="group" class="grid grid-cols-[1.1em_max-content_minmax(0,1fr)] items-baseline gap-x-2 gap-y-0.5" aria-label="Calculation">
  {#each lines as l, j (j)}
    {@const st = breakdownStep(l)}
    {#if 'prose' in st}
      <div class="col-span-3 text-11 leading-snug text-text-2"><PobText text={st.prose} /></div>
    {:else}
      <div class={['contents', st.op === '=' && '[&>*]:border-t [&>*]:border-line [&>*]:pt-0.5']} data-step>
        <span class="text-center font-mono text-text-3">{st.op ?? ''}</span>
        <span class={['text-right font-mono', st.op === '=' ? 'font-semibold text-text' : 'text-text']}><PobText text={st.value} /></span>
        <span class="min-w-0 text-11 text-text-3"><PobText text={st.note ?? ''} /></span>
      </div>
    {/if}
  {/each}
</div>
