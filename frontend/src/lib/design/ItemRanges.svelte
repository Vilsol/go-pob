<script lang="ts" module>
  /* ItemsTab.lua:678-694, 1128-1182, 2564-2577 (PoB-PoE2): the editor's quality, and each modifier range line's roll */
  export type ItemRangesInfo = { lines: { label: string; range: number }[]; quality?: number };
  export type ItemRangesEdit = { op: 'range'; line: number; value: number } | { op: 'quality'; value: number };
</script>

<script lang="ts">
  import NumberInput from './NumberInput.svelte';
  import PobText from './PobText.svelte';

  let { info, onedit }: { info: ItemRangesInfo; onedit: (edit: ItemRangesEdit) => void } = $props();

  const percent = (v: number) => `${Math.round(v * 100)}%`;
</script>

{#if info.quality !== undefined || info.lines.length}
  <section aria-label="Modifier ranges" class="flex flex-col gap-2 border-t border-line pt-2.5">
    {#if info.quality !== undefined}
      <NumberInput
        label="Quality"
        size="sm"
        class="w-30"
        min={0}
        max={99}
        value={info.quality}
        onchange={(value: number) => onedit({ op: 'quality', value })} />
    {/if}
    {#each info.lines as l, i (i)}
      <label class="flex items-center gap-2 text-12 text-text-2"
        ><input
          type="range"
          class="w-25 shrink-0"
          min="0"
          max="1"
          step="0.001"
          aria-label="Roll of range line {i + 1}"
          value={l.range}
          onchange={(e) => onedit({ op: 'range', line: i + 1, value: Number(e.currentTarget.value) })} /><output class="w-10 shrink-0 text-right font-mono"
          >{percent(l.range)}</output
        ><span class="min-w-0 grow"><PobText text={l.label} /></span></label>
    {/each}
  </section>
{/if}
