<script lang="ts">
  import '$lib/design/styles/spec.css';
  import '$lib/display/appearance';
  import PageFrame from '$lib/design/PageFrame.svelte';
  import AppTopBar from '$lib/shell/AppTopBar.svelte';
  import AppSidebar from '$lib/shell/AppSidebar.svelte';
  import { page } from '$app/state';

  import { assets, base } from '$app/paths';
  import { browser } from '$app/environment';
  import { syncWrap } from '$lib/go/worker';
  import { proxy, transfer } from 'comlink';
  import { currentBuild, currentBuildPath, buildChannel, devMode, markBackendAsLoaded, outputs } from '$lib/global.js';
  import { showTitlebarName, windowTitle } from '$lib/display/window_title';
  import { onTick } from '$lib/calc_state';
  import { logError } from '$lib/utils';
  import type { Snippet } from 'svelte';
  import { get } from 'svelte/store';
  import AppBuildList from '$lib/shell/AppBuildList.svelte';
  import { WASM_FALLBACK, WASM_FILE } from '$lib/go/wasm_file';
  import { checkForUpdate, startPwa } from '$lib/pwa';
  import PwaStatus from '$lib/pwa/PwaStatus.svelte';
  import Toasts from '$lib/design/Toasts.svelte';
  import GameVersionDialog from '$lib/design/GameVersionDialog.svelte';
  import { buildName } from '$lib/shell/build_name.svelte';
  import { goto } from '$app/navigation';
  import { importLanding, importParam, openImportLink } from '$lib/shell/link_import';
  import Dialog from '$lib/design/Dialog.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import { phoneMedia, phonePage } from '$lib/design/mobile/mobile';

  let {
    children
  }: {
    children?: Snippet;
  } = $props();

  // one shell at a time, so the page, the sidebar and their worker reads never run twice; crossing the breakpoint
  // remounts the page
  const phone = phoneMedia();
  const FRAMES = import.meta.glob<{ mainClass: string }>('./*/frame.ts', { eager: true });
  const MAIN_CLASS = Object.fromEntries(Object.entries(FRAMES).map(([path, m]) => [path.split('/')[1], m.mainClass]));
  const route = $derived(page.url.pathname.slice(base.length).replace(/^\/+|\/+$/g, ''));
  const LABELS: Record<string, string> = {
    tree: 'Tree',
    skills: 'Skills',
    items: 'Items',
    calcs: 'Calcs',
    configuration: 'Config',
    notes: 'Notes',
    party: 'Party',
    import: 'Import / Export'
  };

  // Main.lua:1750-1756, PassiveSpec.lua:2302-2308
  let titleBuild = $state<{ name: string; className: string; ascendClassName: string }>();
  $effect(() => {
    void $outputs;
    if (!$currentBuild) {
      titleBuild = undefined;
      return;
    }
    const name = buildName(currentBuildPath.current);
    syncWrap
      ?.GetBuildInfo()
      .then((i) => (titleBuild = i ? { name, className: i.className, ascendClassName: i.ascendClassName } : undefined))
      .catch(logError);
  });
  const title = $derived(windowTitle($showTitlebarName, titleBuild));
  // Build.lua:99-106
  let conversion = $state<string>();
  let conversionOpen = $state(false);
  let conversionChecked: unknown;
  $effect(() => {
    const build = $currentBuild;
    if (!build || build === conversionChecked) return;
    conversionChecked = build;
    Promise.resolve(syncWrap?.GetVersionConversion())
      .then((v) => {
        conversion = v;
        conversionOpen = !!v;
      })
      .catch(logError);
  });

  // Main.lua:56-85
  let linkError = $state<string>();
  const linkText = browser ? importParam(new URL(location.href)) : undefined;
  const linkImport = () => {
    const text = linkText;
    if (!text) return;
    openImportLink(text)
      .then(() => goto(importLanding(new URL(location.href), base), { replaceState: true }))
      .catch((e: Error) => {
        logError(e);
        linkError = e.message;
      });
  };

  let wasmLoading = $state(true);
  let dataLoading = $state(false);

  let loadingMessage = $state('Initializing...');
  let loadingStage = $state('');

  async function wasmBytes(path: string): Promise<ArrayBuffer> {
    const res = await fetch(assets + path);
    if (!res.ok) {
      throw new Error(`Failed to load wasm runtime: ${res.status} ${path}`);
    }
    const data = await res.arrayBuffer();
    performance.mark('pob:wasm-fetched');
    const head = new Uint8Array(data, 0, Math.min(4, data.byteLength));
    if (head[0] === 0x00 && head[1] === 0x61 && head[2] === 0x73 && head[3] === 0x6d) {
      return data;
    }
    if (head[0] === 0x1f && head[1] === 0x8b) {
      loadingMessage = 'Decompressing wasm runtime...';
      return new Response(new Response(data).body!.pipeThrough(new DecompressionStream('gzip'))).arrayBuffer();
    }
    if (path !== WASM_FALLBACK) {
      console.warn(`${path} arrived still encoded; falling back to ${WASM_FALLBACK}`);
      return wasmBytes(WASM_FALLBACK);
    }
    loadingMessage = 'Failed to load wasm runtime';
    throw new Error('Failed to load wasm runtime: not a wasm module');
  }

  // Launch.lua:158-161
  const updateKey = (e: KeyboardEvent) => {
    if (!(e.ctrlKey || e.metaKey) || e.altKey || e.shiftKey || e.key.toLowerCase() !== 'u' || get(devMode)) return;
    e.preventDefault();
    checkForUpdate().catch(logError);
  };

  if (browser) {
    startPwa().catch(logError);
    if (!syncWrap || syncWrap === null) {
      loadingMessage = 'Failed to initialize worker';
    } else {
      syncWrap.booted
        .then((booted) => {
          if (booted) {
            wasmLoading = false;
            syncWrap
              ?.loadData(proxy(async () => {}))
              .then(() => get(markBackendAsLoaded)())
              .catch(logError);
            return;
          }

          wasmBytes(WASM_FILE)
            .then((data) => {
              performance.mark('pob:wasm-decoded');
              console.log('wasm runtime size:', data.byteLength);

              loadingMessage = 'Booting...';

              syncWrap
                ?.boot(transfer(data, [data]), proxy(onTick), proxy(buildChannel))
                .then(async () => {
                  console.log('worker booted');
                  performance.mark('pob:wasm-booted');

                  wasmLoading = false;
                  dataLoading = true;
                  await syncWrap?.loadData(
                    // eslint-disable-next-line @typescript-eslint/require-await
                    proxy(async (stage: string) => {
                      loadingStage = stage;
                    })
                  );

                  dataLoading = false;
                  performance.mark('pob:data-loaded');

                  get(markBackendAsLoaded)();
                  linkImport();
                })
                .catch(logError);
            })
            .catch(logError);
        })
        .catch(logError);
    }
  }
</script>

<svelte:head><title>{title}</title></svelte:head>

{#if $currentBuild && !wasmLoading && phone.current}
  <AppTopBar phone label={phonePage(route)?.label ?? LABELS[route]} mainClass={phonePage(route)?.main}>{@render children?.()}</AppTopBar>
{:else if $currentBuild && !wasmLoading}
  <PageFrame label={LABELS[route]} mainClass={MAIN_CLASS[route]}>
    {#snippet topbar()}<AppTopBar />{/snippet}
    {#snippet sidebar()}<AppSidebar />{/snippet}
    {@render children?.()}
  </PageFrame>
{:else if wasmLoading}
  <div class="page items-center justify-center gap-2" role="status">
    <span class="text-18 font-semibold">go-pob</span>
    <span class="text-14 text-text-2">{loadingMessage}</span>
    {#if loadingStage !== ''}<span class="note">{loadingStage}</span>{/if}
  </div>
{:else}
  <AppBuildList />
{/if}

{#if conversion}
  <GameVersionDialog
    bind:open={conversionOpen}
    version={conversion}
    onconvert={() => {
      conversion = undefined;
      void Promise.resolve(syncWrap?.ConvertBuildVersion()).catch(logError);
    }}
    oncancel={() => {
      conversion = undefined;
      void Promise.resolve(syncWrap?.ClearBuild()).catch(logError);
      currentBuildPath.current = undefined;
    }} />
{/if}

{#if dataLoading}
  <div role="status" class="fixed right-2 bottom-2 rounded-md border border-line bg-surface-2 px-3 py-1 text-13 text-text-2 shadow-pop">
    Loading game data{loadingStage ? `: ${loadingStage}` : '...'}
  </div>
{/if}

<Dialog open={!!linkError} title="Build import" class="w-110 p-4.5" onclose={() => (linkError = undefined)}>
  <p role="alert" class="m-0 text-13">The linked build could not be opened: {linkError}</p>
  {#snippet actions()}
    <DialogActions confirm="OK" cancel={false} onconfirm={() => (linkError = undefined)} />
  {/snippet}
</Dialog>

<svelte:window onkeydown={updateKey} />
<PwaStatus />
<Toasts />
