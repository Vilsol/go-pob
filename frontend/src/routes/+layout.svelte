<script lang="ts">
  import Notice from '$lib/design/Notice.svelte';
  import Button from '$lib/design/Button.svelte';
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
  import { currentBuildPath, buildChannel, devMode, markBackendAsLoaded, backendLoaded, engineBoot, poe2Enabled } from '$lib/global.js';
  import { engines, game, lastGame } from '$lib/go/engine';
  import { poe2Worker } from '$lib/go/engine_poe2.svelte';
  import { setArtResolver } from '$lib/design/item_art';
  import { showTitlebarName, windowTitle } from '$lib/display/window_title';
  import { onTick } from '$lib/calc_state';
  import { logError } from '$lib/utils';
  import type { Snippet } from 'svelte';
  import { derived as derivedStore, get } from 'svelte/store';
  import AppBuildList from '$lib/shell/AppBuildList.svelte';
  import { WASM_FALLBACK, WASM_FILE, engineFile } from '$lib/go/wasm_file';
  import { fallBackToGo, runningEngine, takeFallbackNotice } from '$lib/go/engine_choice';
  import { addToast } from '$lib/design/toast';
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
  import AppImportModal from '$lib/shell/AppImportModal.svelte';
  import { openImport } from '$lib/shell/import_modal.svelte';

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
    party: 'Party'
  };

  // Import / Export is a dialog over the page; its old address opens it over the Tree, or over the build list
  $effect(() => {
    if (route !== 'import') return;
    openImport();
    void goto(`${base}/tree${page.url.search}`, { replaceState: true });
  });

  // Main.lua:1750-1756, PassiveSpec.lua:2302-2308
  let titleBuild = $state<{ name: string; className: string; ascendClassName: string }>();
  $effect(() => {
    const e = $game && engines[$game];
    if (!e) {
      titleBuild = undefined;
      return;
    }
    const name = buildName(currentBuildPath.current);
    return derivedStore([e.build, e.calculated], (both) => both).subscribe(() => {
      Promise.resolve(e.classInfo())
        .then((i) => (titleBuild = i ? { name, className: i.className, ascendClassName: i.ascendClassName } : undefined))
        .catch(logError);
    });
  });
  const title = $derived(windowTitle($showTitlebarName, titleBuild));
  // Build.lua:99-106
  let conversion = $state<string>();
  let conversionOpen = $state(false);
  let conversionChecked: unknown;
  const conversionOf = $derived($game && engines[$game]?.conversion);
  $effect(() => {
    const e = $game && engines[$game];
    const conv = e && e.conversion;
    if (!conv) return;
    return e.build.subscribe((build) => {
      if (!build || build === conversionChecked) return;
      conversionChecked = build;
      Promise.resolve(conv.check())
        .then((v) => {
          conversion = v;
          conversionOpen = !!v;
        })
        .catch(logError);
    });
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
  // a boot takes a few seconds even on a loaded machine; one still running after this is stuck, and a reload starts it
  // over with fresh workers
  const BOOT_STUCK_MS = 30_000;
  let bootStuck = $state(false);

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
    const fallback = engineFile(WASM_FALLBACK, runningEngine);
    if (path !== fallback) {
      console.warn(`${path} arrived still encoded; falling back to ${fallback}`);
      return wasmBytes(fallback);
    }
    loadingMessage = 'Failed to load wasm runtime';
    throw new Error('Failed to load wasm runtime: not a wasm module');
  }

  // Launch.lua:158-161
  const updateKey = (e: KeyboardEvent) => {
    if (e.defaultPrevented || !(e.ctrlKey || e.metaKey) || e.altKey || e.shiftKey || e.key.toLowerCase() !== 'u' || get(devMode)) return;
    e.preventDefault();
    checkForUpdate().catch(logError);
  };

  if (browser) {
    startPwa().catch(logError);
    const fellBack = takeFallbackNotice();
    if (fellBack) addToast(`Switched back to the Go engine\nThe TinyGo engine stopped: ${fellBack}.`);
    // the last build was PoE2: its engine boots and preloads its game data now, so the next PoE2 build opens at once;
    // with both engines asked for, the PoE2 one waits for the PoE1 game data, so a PoE1 build opens as soon as before
    if (get(poe2Enabled) && lastGame() === 'poe2') poe2Worker().catch(logError);
    else if (get(poe2Enabled) && get(engineBoot) === 'both') backendLoaded.then(() => poe2Worker()).catch(logError);
    // a PoE2 build opens without the PoE1 engine; any other waits for its game data in openImportLink
    linkImport();
    if (!syncWrap || syncWrap === null) {
      loadingMessage = 'Failed to initialize worker';
    } else {
      const worker = syncWrap;
      setArtResolver(async (queries) => {
        await backendLoaded;
        return worker.ItemArtPaths(queries);
      });
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

          wasmBytes(engineFile(WASM_FILE, runningEngine))
            .then((data) => {
              performance.mark('pob:wasm-decoded');
              console.log('wasm runtime size:', data.byteLength);

              loadingMessage = 'Booting...';
              const stuck = setTimeout(() => {
                if (fallBackToGo(`it took over ${BOOT_STUCK_MS / 1000} s to boot`)) return;
                loadingMessage = 'Booting is taking much longer than it should.';
                bootStuck = true;
              }, BOOT_STUCK_MS);

              syncWrap
                ?.boot(transfer(data, [data]), proxy(onTick), proxy(buildChannel), runningEngine)
                .then(async () => {
                  clearTimeout(stuck);
                  bootStuck = false;
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
                })
                .catch((e: Error) => {
                  logError(e);
                  if (!wasmLoading) return;
                  clearTimeout(stuck);
                  if (fallBackToGo(`it did not boot (${e.message})`)) return;
                  loadingMessage = `Booting failed: ${e.message}`;
                  bootStuck = true;
                });
            })
            .catch((e: Error) => {
              logError(e);
              fallBackToGo(`its wasm did not load (${e.message})`);
            });
        })
        .catch(logError);
    }
  }
</script>

<svelte:head><title>{title}</title></svelte:head>

{#if $game && !wasmLoading}
  {#key $game}
    {#if phone.current}
      <AppTopBar phone label={phonePage(route)?.label ?? LABELS[route]} mainClass={phonePage(route)?.main}>{@render children?.()}</AppTopBar>
    {:else}
      <PageFrame label={LABELS[route]} mainClass={MAIN_CLASS[route]}>
        {#snippet topbar()}<AppTopBar />{/snippet}
        {#snippet sidebar()}<AppSidebar />{/snippet}
        {@render children?.()}
      </PageFrame>
    {/if}
  {/key}
{:else if wasmLoading}
  <div class="page items-center justify-center gap-2" role="status">
    <span class="text-18 font-semibold">go-pob</span>
    <span class="text-14 text-text-2">{loadingMessage}</span>
    {#if loadingStage !== ''}<span class="note">{loadingStage}</span>{/if}
    {#if bootStuck}<Button onclick={() => location.reload()}>Retry</Button>{/if}
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
      void Promise.resolve(conversionOf?.convert()).catch(logError);
    }}
    oncancel={() => {
      conversion = undefined;
      void Promise.resolve($game && engines[$game]?.clear()).catch(logError);
      currentBuildPath.current = undefined;
    }} />
{/if}

{#if !wasmLoading}<AppImportModal />{/if}

{#if dataLoading}
  <Notice float class="fixed right-2 bottom-2">Loading game data{loadingStage ? `: ${loadingStage}` : '...'}</Notice>
{/if}

<Dialog open={!!linkError} title="Build import" size="md" onclose={() => (linkError = undefined)}>
  <p role="alert" class="m-0 text-13">The linked build could not be opened: {linkError}</p>
  {#snippet actions()}
    <DialogActions confirm="OK" cancel={false} onconfirm={() => (linkError = undefined)} />
  {/snippet}
</Dialog>

<svelte:window onkeydown={updateKey} />
<PwaStatus />
<Toasts />
