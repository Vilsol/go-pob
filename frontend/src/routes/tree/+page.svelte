<script>
  import SkillTree from '$lib/components/SkillTree.svelte';
  import { drawnGroups, loadSkillTree, skillTree, skillTreeVersion } from '$lib/skill_tree';
  import { onMount } from 'svelte';
  import { logError } from '$lib/utils';

  onMount(() => {
    if (!$skillTree || !$skillTreeVersion || Object.keys(drawnGroups).length === 0) {
      loadSkillTree('3_18').catch(logError);
    }
  });
</script>

{#if $skillTree && $skillTreeVersion && Object.keys(drawnGroups).length > 0}
  <SkillTree />
{/if}
