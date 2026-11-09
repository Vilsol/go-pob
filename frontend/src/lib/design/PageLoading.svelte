<script lang="ts">
  import { onMount } from 'svelte';
  import { SHOW_AFTER_MS } from '$lib/activity/jobs';
  import Spinner from './Spinner.svelte';

  // what a page shows while its data is on the way; a page that is ready sooner never shows it
  let { label = 'Loading…', class: extra = '' }: { label?: string; class?: string } = $props();
  let late = $state(false);
  onMount(() => {
    const t = setTimeout(() => (late = true), SHOW_AFTER_MS);
    return () => clearTimeout(t);
  });
</script>

<div class={['flex flex-1 items-center justify-center gap-2 p-5 text-13 text-text-2', extra]} role="status" aria-busy="true">
  {#if late}<Spinner />{label}{/if}
</div>
