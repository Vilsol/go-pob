<script lang="ts">
  import type { Node } from '$lib/skill_tree/types';
  import { drawnNodes } from '$lib/skill_tree';

  let { activeNodes, onselect }: { activeNodes: number[]; onselect: (node: Node) => void } = $props();

  const MAX_RESULTS = 20;

  const selectable = [...drawnNodes.values()]
    .filter((n) => n.skill !== undefined && !!n.name && n.classStartIndex === undefined && !n.isAscendancyStart && !n.isMastery)
    .map((node) => ({ node, key: `${node.name} #${node.skill}`.toLowerCase() }));

  let query = $state('');

  const matches = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (q.length < 2) {
      return [];
    }
    return selectable
      .filter((c) => c.key.includes(q))
      .slice(0, MAX_RESULTS)
      .map((c) => c.node);
  });

  const active = $derived(new Set(activeNodes));
</script>

<div class="absolute top-2 left-2 z-10 flex max-w-xs flex-col gap-1 rounded-md border border-line-strong bg-surface-1/90 p-2">
  <input type="search" class="input" placeholder="Search passives" aria-label="Search passive nodes" bind:value={query} />
  {#if matches.length > 0}
    <ul aria-label="Matching passive nodes" class="m-0 flex max-h-80 list-none flex-col overflow-y-auto p-0">
      {#each matches as node (node.skill)}
        <li>
          <button
            type="button"
            class="w-full border-0 bg-transparent px-1 text-left font-sans text-13 text-text hover:bg-hover"
            class:text-accent={active.has(node.skill!)}
            aria-pressed={active.has(node.skill!)}
            onclick={() => onselect(node)}>
            {node.name} <span class="opacity-50">#{node.skill}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>
