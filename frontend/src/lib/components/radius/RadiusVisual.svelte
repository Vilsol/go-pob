<script lang="ts">
  import { RADIUS_BOX, circleRows } from './radius_visual';

  // CalcBreakdownControl.lua:140-147, 626-641
  let { radius }: { radius: number } = $props();
  const rows = $derived(circleRows(radius));
</script>

<svg
  role="img"
  aria-label="Radius {radius / 10} metres"
  class="block rounded border border-line bg-[#101010]"
  width={RADIUS_BOX.width + 8}
  height={RADIUS_BOX.height + 4}
  viewBox="-4 -2 {RADIUS_BOX.width + 8} {RADIUS_BOX.height + 4}">
  {#each rows as [y, x, w] (y)}<rect {x} {y} width={w} height="1" fill="rgb(127 255 127 / 0.33)" />{/each}
</svg>
