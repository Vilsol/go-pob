<script lang="ts">
  import type { OverlayConfig } from '$lib/overlay';

  let {
    config,
    closeOverlay
  }: {
    config: OverlayConfig;
    closeOverlay: () => void;
  } = $props();

  let backdrop = $state<HTMLElement>();

  const close = (event: Event) => {
    if (config.backdropClose && event.target === backdrop) {
      closeOverlay();
    }
  };
</script>

<button class="overlay" bind:this={backdrop} onclick={close} onkeyup={close}>
  <config.component onclose={closeOverlay} {...config.props} />
</button>

<style lang="postcss">
  .overlay {
    @apply absolute top-0 left-0 w-screen h-screen z-50 bg-black/75 flex items-center justify-center;
  }
</style>
