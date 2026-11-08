<script lang="ts">
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { currentEngine } from '$lib/go/engine';
  import { logError } from '$lib/utils';
  import AppTree2Body from './AppTree2Body.svelte';

  // the frame is keyed on the game, so the page reads its engine once
  const page = currentEngine()
    .tree?.()
    .catch((err: Error) => {
      logError(err);
      return undefined;
    });
</script>

{#await page}
  <EmptyState busy text="Loading the passive tree…" />
{:then make}
  {#if make}<AppTree2Body {make} />{/if}
{/await}
