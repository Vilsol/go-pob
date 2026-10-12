<script lang="ts">
  import { base } from '$app/paths';
  import type { Sprite } from '$lib/skill_tree/types';

  // PassiveTreeView.lua:1158-1204
  type Props = { x: number; y: number; outer: number; inner: number; art?: string; keystone?: boolean; sheet?: Sprite; nodeId: number };
  let { x, y, outer, inner, art = '', keystone = false, sheet, nodeId }: Props = $props();

  // PassiveTreeView.lua:1180-1197
  const TIMELESS: Record<string, string> = {
    'Brutal Restraint': 'Maraketh',
    'Elegant Hubris': 'EternalEmpire',
    'Glorious Vanity': 'Vaal',
    'Lethal Pride': 'Karui',
    'Militant Faith': 'Templar',
    'Heroic Tragedy': 'Kalguur'
  };
  type Layer = { src?: string; sprite?: string; size: number; angle: number };
  const layers = $derived.by((): Layer[] => {
    const prefix = TIMELESS[art];
    if (prefix && sheet?.coords[`${prefix}JewelCircle1`]) {
      return [
        { sprite: `${prefix}JewelCircle1`, size: outer, angle: -0.7 },
        { sprite: `${prefix}JewelCircle2`, size: outer, angle: 0.7 }
      ];
    }
    const [a, b, c, d] = keystone ? [-0.8, 1, -1.2, 1.0] : [-0.7, 0.7, -0.7, 0.7];
    return [
      { src: 'ShadedOuterRing', size: outer, angle: a },
      { src: 'ShadedOuterRingFlipped', size: outer, angle: b },
      { src: 'ShadedInnerRing', size: inner, angle: c },
      { src: 'ShadedInnerRingFlipped', size: inner, angle: d }
    ];
  });
  // PassiveTreeView.lua:1250-1273
  const period = (angle: number) => `${((2 * Math.PI) / (Math.abs(angle) * 0.03)).toFixed(1)}s`;
  const spriteStyle = (name: string, size: number) => {
    const c = sheet!.coords[name];
    const k = (2 * size) / c.w;
    return `background-image:url(${sheet!.filename});background-size:${sheet!.w * k}px ${sheet!.h * k}px;background-position:${-c.x * k}px ${-c.y * k}px`;
  };
</script>

<div class="jewel-ring" data-jewel-ring={nodeId} data-art={art || undefined}>
  {#each layers as l, i (i)}
    {#if l.size > 0}
      <div
        data-ring-layer={l.src ?? l.sprite}
        class="turn absolute bg-no-repeat"
        style:left="{x - l.size}px"
        style:top="{y - l.size}px"
        style:width="{2 * l.size}px"
        style:height="{2 * l.size}px"
        style:--turn-period={period(l.angle)}
        style:--turn-dir={l.angle < 0 ? 'reverse' : 'normal'}
        style={l.sprite ? spriteStyle(l.sprite, l.size) : `background-image:url(${base}/tree-rings/${l.src}.png);background-size:100% 100%`}>
      </div>
    {/if}
  {/each}
</div>

<style>
  /* PassiveTreeView.lua:1251-1254 */
  .turn {
    animation: turn var(--turn-period) linear infinite var(--turn-dir);
  }
  @keyframes turn {
    to {
      transform: rotate(1turn);
    }
  }
</style>
