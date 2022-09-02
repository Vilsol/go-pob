<script>
  import SkillTree from '$lib/components/SkillTree.svelte';
  import { drawnGroups, loadSkillTree, skillTree, skillTreeVersion } from '$lib/skill_tree';
  import { onMount } from 'svelte';

  onMount(() => {
    if (!$skillTree || !$skillTreeVersion || Object.keys(drawnGroups).length === 0) {
      loadSkillTree('3_18');
    }
  });
</script>

{#if $skillTree && $skillTreeVersion}
  <SkillTree />
{/if}
