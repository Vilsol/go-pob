<script>
  import SkillTree from '$lib/components/SkillTree.svelte';
  import { drawnGroups, loadSkillTree, skillTree, skillTreeVersion } from '$lib/skill_tree';
  import { onMount } from 'svelte';
  import { logError } from '$lib/utils';

  onMount(() => {
    if (!$skillTree || !$skillTreeVersion || drawnGroups.size === 0) {
      loadSkillTree('3_18').catch(logError);
    }
  });
</script>

{#if $skillTree && $skillTreeVersion && drawnGroups.size > 0}
  <SkillTree skillTree={$skillTree} skillTreeVersion={$skillTreeVersion} />
{/if}
