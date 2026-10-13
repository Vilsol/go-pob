<script lang="ts">
  import { RADIUS_BOX, circleRows, ringLabelAt, ringPath, worldToScreen } from './radius_visual';

  // CalcBreakdownControl.lua:140-147, 626-641
  let { radius }: { radius: number } = $props();
  const rows = $derived(circleRows(radius));
  const edge = $derived(ringPath(radius));
  const RINGS = Array.from({ length: 13 }, (_, i) => 10 * (i + 1));
  const rings = RINGS.map((r) => ({ r, d: ringPath(r), even: r % 20 === 0 }));
  const labels = RINGS.filter((r) => r <= 80 && r % 20 === 0).map((r) => ({ r, at: ringLabelAt(r) }));
  const feet = worldToScreen(0, 0, 0, RADIUS_BOX.width, RADIUS_BOX.height);
  const head = worldToScreen(0, 0, 18, RADIUS_BOX.width, RADIUS_BOX.height);
  const tall = feet[1] - head[1];
  const mark = $derived(ringLabelAt(radius));
</script>

<figure class="m-0 flex flex-col gap-1">
  <svg
    role="img"
    aria-label="Radius {radius} ({radius / 10} metres) around the character"
    class="block h-auto w-full rounded border border-line"
    viewBox="0 0 {RADIUS_BOX.width} {RADIUS_BOX.height}">
    <defs>
      <radialGradient id="radius-ground" cx="50%" cy="45%" r="75%">
        <stop offset="0" stop-color="#5a5333" />
        <stop offset="0.6" stop-color="#3a3522" />
        <stop offset="1" stop-color="#1b1a12" />
      </radialGradient>
    </defs>
    <rect width={RADIUS_BOX.width} height={RADIUS_BOX.height} fill="url(#radius-ground)" />
    {#each rings as g (g.r)}
      <path d={g.d} fill="none" stroke={g.even ? 'rgb(255 255 255 / 0.55)' : 'rgb(140 140 220 / 0.55)'} stroke-width="1" />
    {/each}
    {#each rows as [y, x, w] (y)}<rect {x} {y} width={w} height="1" fill="rgb(127 255 127 / 0.33)" />{/each}
    <path d={edge} fill="none" stroke="rgb(127 255 127)" stroke-width="1.5" data-radius-edge />
    {#each labels as l (l.r)}
      <g transform="translate({l.at[0]} {l.at[1]})">
        <rect x="-11" y="-8" width="22" height="16" rx="2" fill="rgb(0 0 0 / 0.8)" />
        <text x="0" y="4.5" text-anchor="middle" font-size="12" font-weight="600" font-family="sans-serif" fill="#ff5a4a">{l.r}</text>
      </g>
    {/each}
    <g aria-hidden="true">
      <ellipse cx={feet[0]} cy={feet[1]} rx={tall * 0.32} ry={tall * 0.12} fill="rgb(0 0 0 / 0.45)" />
      <line x1={feet[0]} y1={feet[1] - tall * 0.08} x2={head[0]} y2={head[1] + tall * 0.34} stroke="#e8e2cc" stroke-width={tall * 0.2} stroke-linecap="round" />
      <circle cx={head[0]} cy={head[1] + tall * 0.1} r={tall * 0.1} fill="#e8e2cc" />
    </g>
    {#if mark[0] < RADIUS_BOX.width - 20 && mark[1] > 0 && mark[1] < RADIUS_BOX.height}
      <circle cx={mark[0]} cy={mark[1]} r="2.5" fill="rgb(127 255 127)" />
    {/if}
  </svg>
  <figcaption class="text-10 text-text-3">
    <span class="font-mono text-text">{radius}</span> radius ({radius / 10} m) · rings every 10 · the box is a 1080p screen
  </figcaption>
</figure>
