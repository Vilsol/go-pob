<script lang="ts">
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { currentEngine } from '$lib/go/engine';
  import { logError } from '$lib/utils';
  import AppSkillsBody from './AppSkillsBody.svelte';

  // the frame is keyed on the game, so the page reads its engine once
  const page = currentEngine()
    .skills()
    .catch((err: Error) => {
      logError(err);
      return undefined;
    });
</script>

{#await page}
  <EmptyState busy text="Loading the skills…" />
{:then make}
  {#if make}<AppSkillsBody {make} />{/if}
{/await}
