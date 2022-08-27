<script lang="ts">
  import { overlays } from '$lib/overlay';
  import { get } from 'svelte/store';
  import Overlay from '$lib/components/overlays/Overlay.svelte';

  const closeOverlay = (index: number) => {
    get(overlays).splice(index, 1);
    overlays.set(get(overlays));
  };
</script>

{#each $overlays as overlay, i}
  <Overlay config={overlay} closeOverlay={() => closeOverlay(i)} />
{/each}
