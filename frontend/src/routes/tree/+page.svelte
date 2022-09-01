<script>
  import SkillTree from '$lib/components/SkillTree.svelte';
  import { loadSkillTree, skillTree } from '$lib/skill_tree';
  import { onMount } from 'svelte';

  onMount(() => {
    if (!$skillTree) {
      loadSkillTree('3_18');
    }
  });
</script>

{#if $skillTree}
  <SkillTree />
{/if}
