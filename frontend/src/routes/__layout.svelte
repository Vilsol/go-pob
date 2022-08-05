<script lang="ts">
  import '../app.postcss';
  import Header from '../lib/components/Header.svelte';
  import Sidebar from '../lib/components/Sidebar.svelte';

  import '../wasm_exec.js';
  import { assets } from '$app/paths';
  import { browser } from '$app/env';
  import { syncWrap } from '../lib/go/worker';
  // import { initializeCrystalline } from "../lib/types";

  let wasmLoading = true;

  // eslint-disable-next-line no-undef
  // const go = new Go();

  let loadingMessage = 'Initializing...';

  if (browser) {
    fetch(assets + '/go-pob.wasm')
      .then((data) => data.arrayBuffer())
      .then((data) => {
        // WebAssembly.instantiate(data, go.importObject).then((result) => {
        //   go.run(result.instance);
        //   wasmLoading = false;
        //   initializeCrystalline();
        // });

        syncWrap.boot(data).then(async () => {
          console.log('worker booted');

          loadingMessage = 'Loading data...';
          await syncWrap.loadData();

          wasmLoading = false;
        });
      });
  }
</script>

<div class="w-screen h-screen max-w-screen max-h-screen overflow-hidden flex flex-col">
  {#if wasmLoading}
    <div class="flex flex-row justify-center h-full">
      <div class="flex flex-col justify-center text-5xl">
        {loadingMessage}
      </div>
    </div>
  {:else}
    <Header />

    <div class="flex flex-row h-full full-page">
      <Sidebar />

      <div>
        <slot />
      </div>
    </div>
  {/if}
</div>
