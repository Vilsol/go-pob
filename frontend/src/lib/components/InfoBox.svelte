<script lang="ts">
  import { openOverlay } from '$lib/overlay';
  import Options from '$lib/components/overlays/Options.svelte';
  import Version from '$lib/components/overlays/Version.svelte';
  import { syncWrap } from '$lib/go/worker';

  const openOptions = () => {
    openOverlay({
      component: Options,
      props: {}
    });
  };

  const openVersion = () => {
    openOverlay({
      component: Version,
      props: {}
    });
  };
</script>

<div class="flex flex-row p-2">
  <div class="flex flex-col flex-1 gap-2">
    <button class="container min-w-full flex-1" onclick={openOptions}>Options</button>
    <button class="container min-w-full flex-1">About</button>
  </div>
  <div class="flex flex-col flex-1 items-center">
    <button class="flex-1 flex place-items-center cursor-pointer" onclick={openVersion}>
      {#await syncWrap?.BuildInfo() then buildInfo}
        Version: {buildInfo?.Main?.Version}
      {/await}
    </button>
  </div>
</div>
