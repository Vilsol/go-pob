<script module lang="ts">
  import './heat_story_worker';
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import SkillTree from './SkillTree.svelte';
  import { loadSkillTree, skillTree } from '$lib/skill_tree';
  import type { Tree } from '$lib/skill_tree/types';

  // PassiveTreeView.lua:831-833, 906-919
  const { Story } = defineMeta({ title: 'Live/Tree heat map', component: SkillTree, tags: ['!test'] });

  const heatOf = (tree: Tree) => {
    const values = new Map<number, number>();
    for (const [id, n] of Object.entries(tree.nodes)) {
      if (n.classStartIndex !== undefined || n.isAscendancyStart || n.ascendancyName || n.group === undefined) continue;
      const g = tree.groups[n.group];
      if (!g) continue;
      values.set(Number(id), Math.max(0, 100 - Math.hypot(g.x, g.y) / 80) * ((Number(id) % 7) / 6));
    }
    return { values, max: Math.max(...values.values()) };
  };
  void loadSkillTree('3_29');
</script>

<Story name="Heat map" asChild>
  <div class="h-[720px] w-full bg-tree-ground">
    {#if $skillTree}
      <SkillTree skillTree={$skillTree} skillTreeVersion="3_29" chrome={false} heat={heatOf($skillTree)} />
    {:else}
      <p class="p-4 text-text-2">Loading the 3.29 tree…</p>
    {/if}
  </div>
</Story>
