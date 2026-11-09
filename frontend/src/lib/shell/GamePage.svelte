<script lang="ts">
  import type { Snippet } from 'svelte';
  import { game } from '$lib/go/engine';
  import { GAME_PAGES } from './game_pages';
  import PageLoading from '$lib/design/PageLoading.svelte';

  let { route, children }: { route: string; children: Snippet } = $props();

  // with no build open the route's own page renders, as it did before there were two games
  const body = $derived(GAME_PAGES[route]?.[$game ?? 'poe1']);
</script>

{#if body === 'route'}
  {@render children()}
{:else if body}
  {#await body()}
    <PageLoading label="Loading the {GAME_PAGES[route]?.label ?? route} page…" />
  {:then page}
    <page.default {route} />
  {/await}
{:else}
  <p class="note p-5" role="status">The {GAME_PAGES[route]?.label ?? route} page is not available for Path of Exile 2 builds yet.</p>
{/if}
