<script>
  import SkillTree from '$lib/components/SkillTree.svelte';
  import { loadSkillTree, skillTree, skillTreeVersion } from '$lib/skill_tree';
  import { onMount } from 'svelte';

  onMount(() => {
    if (!$skillTree || !$skillTreeVersion) {
      loadSkillTree('3_18');
    }
  });
</script>

{#if $skillTree && $skillTreeVersion}
  <SkillTree />
{/if}
