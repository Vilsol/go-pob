<script lang="ts">
  import type { Node } from '$lib/skill_tree/types';
  import type { Point } from '$lib/skill_tree/geometry';
  import { lineText, type NodeStatDiff } from '$lib/skill_tree/stat_compare';

  interface Props {
    node: Node;
    at: Point;
    width: number;
    height: number;
    devMode: boolean;
    diff?: NodeStatDiff;
  }

  let { node, at, width, height, devMode, diff }: Props = $props();

  let name = $derived((node.name || 'N/A') + (devMode ? ` (${node.skill})` : ''));
  let flipX = $derived(at.x > width / 2);
  let flipY = $derived(at.y > height / 2);
</script>

<div
  role="tooltip"
  aria-label="Node tooltip"
  class="pointer-events-none absolute z-20 max-w-[36rem] min-w-[18rem] rounded-md border border-line-strong bg-ground/90 text-13 text-text shadow-tip"
  style="left: {at.x}px; top: {at.y}px; transform: translate({flipX ? 'calc(-100% - 12px)' : '12px'}, {flipY ? 'calc(-100% - 12px)' : '12px'})">
  <h2 class="bg-warn-bg px-3 py-2 text-center text-15 font-semibold">{name}</h2>
  {#if node.stats?.length}
    <ul class="m-0 list-none space-y-1 px-3 py-2">
      {#each node.stats as stat, i (i)}
        <li class="whitespace-pre-line">{stat}</li>
      {/each}
    </ul>
  {:else if node.isJewelSocket}
    <p class="m-0 px-3 py-2 text-relic">Click to select this socket</p>
  {/if}
  {#if diff && diff.node === node.skill}
    <section aria-label="Stat differences" aria-busy={diff.pending} data-node={diff.node} class="space-y-2 border-t border-line px-3 py-2">
      {#each diff.blocks as block (block.header)}
        <div data-ready-at={block.readyAt} data-calc-ms={block.calcMs}>
          <p class="m-0 text-text-2">{block.header}</p>
          {#each block.sections as section, s (s)}
            {#if section.actor}
              <p class="m-0 text-text-2">{section.actor}:</p>
            {/if}
            <ul class="m-0 list-none p-0" aria-label={block.header + (section.actor ? ` ${section.actor}` : '')}>
              {#each section.lines as line, l (l)}
                <li class={line.better ? 'text-better' : 'text-worse'}>
                  {lineText(line)}{#if line.perPoint}<span class="text-text-3">{' ' + line.perPoint}</span>{/if}
                </li>
              {/each}
            </ul>
          {/each}
        </div>
      {/each}
      {#if diff.pending}
        <p class="m-0 text-text-3">Calculating…</p>
      {:else if diff.empty}
        <p class="m-0 text-text-2">{diff.empty}</p>
      {/if}
    </section>
  {/if}
</div>
