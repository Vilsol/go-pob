<script lang="ts">
  import '../app.postcss';
  import Header from '../lib/components/Header.svelte';
  import Sidebar from '../lib/components/Sidebar.svelte';

  import { assets } from '$app/paths';
  import { browser } from '$app/environment';
  import { syncWrap } from '$lib/go/worker';
  import { proxy } from 'comlink';
  import type { Outputs } from '$lib/custom_types';
  import { outputs, currentBuild, markBackendAsLoaded } from '$lib/global.js';
  import OverlayController from '$lib/components/overlays/OverlayController.svelte';
  import { fontScaling } from '$lib/global.js';
  import { logError } from '$lib/utils';
  import type { Snippet } from 'svelte';
  import { get } from 'svelte/store';
  import BuildSelectorPage from '$lib/components/build-selector/BuildSelectorPage.svelte';

  let {
    children
  }: {
    children?: Snippet;
  } = $props();

  let wasmLoading = $state(true);

  let loadingMessage = $state('Initializing...');
  let loadingStage = $state('');

  if (browser) {
    if (!syncWrap || syncWrap === null) {
      loadingMessage = 'Failed to initialize worker';
    } else {
      syncWrap.booted
        .then((booted) => {
          if (booted) {
            wasmLoading = false;
            return;
          }

          fetch(assets + (import.meta.env.MODE === 'development' ? '/go-pob.wasm' : '/go-pob.wasm.gz'))
            .then(async (data) => {
              if (!data.body) {
                loadingMessage = 'Failed to load wasm runtime';
                throw new Error('Failed to load wasm runtime');
              }

              return data.arrayBuffer();
            })
            .then((data) => {
              if (data.byteLength < 2) {
                loadingMessage = 'Failed to load wasm runtime';
                throw new Error('Failed to load wasm runtime');
              }

              const dataArray = new Uint8Array(data);
              if (dataArray[0] === 0x1f && dataArray[1] === 0x8b) {
                loadingMessage = 'Decompressing wasm runtime...';

                const decompressedStream = new Response(data).body!.pipeThrough(new DecompressionStream('gzip'));
                return new Response(decompressedStream).arrayBuffer();
              }

              return data;
            })
            .then(async (data) => {
              console.log('wasm runtime size:', data.byteLength);

              loadingMessage = 'Booting...';

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
                    // eslint-disable-next-line @typescript-eslint/require-await
                    proxy(async (stage: string) => {
                      loadingMessage = 'Loading data:';
                      loadingStage = stage;
                    })
                  );

                  wasmLoading = false;

                  get(markBackendAsLoaded)();
                })
                .catch(logError);
            })
            .catch(logError);
        })
        .catch(logError);
    }
  }
</script>

<div class="w-screen h-screen max-w-screen max-h-screen overflow-hidden flex flex-col" style="font-size: {$fontScaling}pt">
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
  {:else if !$currentBuild}
    <BuildSelectorPage />
  {:else}
    <Header />

    <div class="flex flex-row h-full full-page">
      <Sidebar />

      <div class="h-full w-full overflow-hidden">
        {@render children?.()}
      </div>
    </div>
  {/if}

  <OverlayController />
</div>
