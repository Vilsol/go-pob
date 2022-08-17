<script lang="ts">
  import '../app.postcss';
  import Header from '../lib/components/Header.svelte';
  import Sidebar from '../lib/components/Sidebar.svelte';

  import { assets } from '$app/paths';
  import { browser } from '$app/env';
  import { syncWrap } from '../lib/go/worker';
  import { proxy } from 'comlink';
  import type { Outputs } from '../lib/custom_types';
  import { outputs, currentBuild } from '../lib/global';

  let wasmLoading = true;

  let loadingMessage = 'Initializing...';
  let loadingStage = '';

  if (browser) {
    if (!syncWrap || syncWrap === null) {
      loadingMessage = 'Failed to initialize worker';
    } else {
      syncWrap.booted.then((booted) => {
        if (booted) {
          wasmLoading = false;
          return;
        }

        fetch(assets + '/go-pob.wasm')
          .then((data) => data.arrayBuffer())
          .then((data) => {
            syncWrap
              ?.boot(
                data,
                proxy((out: Outputs) => {
                  outputs.set(out);
                }),
                proxy(currentBuild)
              )
              .then(async () => {
                console.log('worker booted');

                loadingMessage = 'Loading data...';
                await syncWrap?.loadData(
                  proxy(async (stage: string) => {
                    loadingMessage = 'Loading data:';
                    loadingStage = stage;
                  })
                );

                wasmLoading = false;
              });
          });
      });
    }
  }
</script>

<div class="w-screen h-screen max-w-screen max-h-screen overflow-hidden flex flex-col">
  {#if wasmLoading}
    <div class="flex flex-row justify-center h-full">
      <div class="flex flex-col justify-center text-5xl text-center">
        {loadingMessage}
        {#if loadingStage !== ''}
          <br />
          {loadingStage}
        {/if}
      </div>
    </div>
  {:else}
    <Header />

    <div class="flex flex-row h-full full-page">
      <Sidebar />

      <div class="h-full w-full">
        <slot />
      </div>
    </div>
  {/if}
</div>
