<script lang="ts">
  import type { OverlayConfig } from '$lib/overlay';

  export let config: OverlayConfig;
  export let closeOverlay: () => void;

  let backdrop: HTMLElement;

  const close = (event: MouseEvent) => {
    if (config.backdropClose && event.target === backdrop) {
      closeOverlay();
    }
  };
</script>

<div class="absolute top-0 left-0 w-screen h-screen z-50 bg-black/75 flex items-center justify-center" bind:this={backdrop} on:click={close}>
  <svelte:component this={config.component} on:close={closeOverlay} {...config.props} />
</div>
