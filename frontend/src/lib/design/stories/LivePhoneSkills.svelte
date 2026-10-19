<script lang="ts">
  import LiveSkills from './LiveSkills.svelte';
  import SkillsPanes from '../mobile/SkillsPanes.svelte';
  import type { BuildFixture } from '../fixtures';

  type Props = { fx: BuildFixture; pane?: 0 | 1 };
  let { fx, pane = 0 }: Props = $props();
</script>

<SkillsPanes {pane}><LiveSkills {fx} /></SkillsPanes>
