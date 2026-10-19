<script lang="ts">
  import Pill from './Pill.svelte';
  import { besideAnchor } from './items';

  type Props = {
    label: string;
    a: string;
    b: string;
    lead: string;
    winner?: 'a' | 'b';
    pA: number;
    pB: number;
    caps?: [number, number];
    zero?: number;
    /* CompareTab.lua:3328-3341 */
    diff?: string;
    diffBetter?: boolean;
  };
  let { label, a, b, lead, winner, pA, pB, caps, zero, diff, diffBetter = false }: Props = $props();

  const uid = $props.id();
  let row = $state<HTMLElement>();
  let tip = $state(false);

  const lo = $derived(Math.min(pA, pB));
  const gap = $derived(Math.max(pA, pB) - lo);
  const pct = (v: number) => v.toFixed(2) + '%';
</script>

<!-- eslint-disable-next-line svelte/no-unused-svelte-ignore -- svelte-check warns on both, eslint-plugin-svelte sees one -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_static_element_interactions -->
<div
  bind:this={row}
  class="butterfly-row"
  tabindex={diff ? 0 : undefined}
  aria-describedby={diff && tip ? `${uid}-diff` : undefined}
  onmouseenter={() => (tip = true)}
  onmouseleave={() => (tip = false)}
  onfocusin={() => (tip = true)}
  onfocusout={() => (tip = false)}>
  <span class="butterfly-value text-right" data-win={winner === 'a'}>{a}</span>
  <div aria-hidden="true" class="bf-track bf-a">
    <div class="bf-bed"></div>
    <div class="bf-bar tone-build-a" style:--from="0%" style:--w={pct(winner === 'b' ? lo : pA)}></div>
    {#if winner === 'b'}<div class="bf-bar bf-gap tone-build-b" style:--from={pct(lo)} style:--w={pct(gap)}></div>{/if}
    {#if zero != null}<div class="bf-tick bf-tick-zero" style:--at={pct(zero)}></div>{/if}
    {#if caps}<div class="bf-tick" style:--at={pct(caps[0])}></div>{/if}
    <div class="bf-frame"></div>
  </div>
  <span class="butterfly-label">{label}<Pill {winner}>{lead}</Pill></span>
  <div aria-hidden="true" class="bf-track bf-b">
    <div class="bf-bed"></div>
    <div class="bf-bar tone-build-b" style:--from="0%" style:--w={pct(winner === 'a' ? lo : pB)}></div>
    {#if winner === 'a'}<div class="bf-bar bf-gap tone-build-a" style:--from={pct(lo)} style:--w={pct(gap)}></div>{/if}
    {#if zero != null}<div class="bf-tick bf-tick-zero" style:--at={pct(zero)}></div>{/if}
    {#if caps}<div class="bf-tick" style:--at={pct(caps[1])}></div>{/if}
    <div class="bf-frame"></div>
  </div>
  <span class="butterfly-value" data-win={winner === 'b'}>{b}</span>
</div>
{#if diff && tip}
  <div
    role="tooltip"
    id="{uid}-diff"
    class="tooltip pointer-events-none fixed top-(--y) left-(--x) z-float rounded-md px-2.5 py-1.5 text-12 shadow-tip"
    use:besideAnchor={row}>
    {label} · B − A: <span class={['font-mono', diffBetter ? 'text-better' : 'text-worse']} data-better={diffBetter}>{diff}</span>
  </div>
{/if}
