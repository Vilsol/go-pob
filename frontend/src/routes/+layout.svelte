<script lang="ts">
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
  import { proxy } from 'comlink';
  import { currentBuildPath, buildChannel, devMode, markBackendAsLoaded, backendLoaded, engineBoot, poe2Enabled } from '$lib/global.js';
  import { engines, game, lastGame } from '$lib/go/engine';
  import { poe2Worker } from '$lib/go/engine_poe2.svelte';
  import { dataErrorText, poe2DataError } from '$lib/go/data_error';
  import { setArtResolver } from '$lib/design/item_art';
  import { showTitlebarName, windowTitle } from '$lib/display/window_title';
  import { onTick } from '$lib/calc_state';
  import { logError } from '$lib/utils';
  import type { Snippet } from 'svelte';
  import { derived as derivedStore, get } from 'svelte/store';
  import AppBuildList from '$lib/shell/AppBuildList.svelte';
  import { WASM_FALLBACK, WASM_FILE, engineFile } from '$lib/go/wasm_file';
  import { fallBackToGo, runningEngine, takeFallbackNotice } from '$lib/go/engine_choice';
  import { addToast, removeToast } from '$lib/design/toast';
  import { checkForUpdate, followConnection, startPwa } from '$lib/pwa';
  import Toasts from '$lib/design/Toasts.svelte';
  import GameVersionDialog from '$lib/design/GameVersionDialog.svelte';
  import { buildName } from '$lib/shell/build_name.svelte';
  import { beforeNavigate, goto } from '$app/navigation';
  import { importLanding, importParam, openImportLink } from '$lib/shell/link_import';
  import Dialog from '$lib/design/Dialog.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import { phoneMedia, phonePage } from '$lib/design/mobile/mobile';
  import AppImportModal from '$lib/shell/AppImportModal.svelte';
  import { openImport } from '$lib/shell/import_modal.svelte';
  import ActivityBar from '$lib/design/ActivityBar.svelte';
  import { startJob, track } from '$lib/activity/jobs';
  import { stageReporter } from '$lib/activity/worker_jobs';
  import { followLongJobs, jobToastKey } from '$lib/activity/job_toasts';
  import type { WasmReport } from '$lib/activity/read_body';

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
  const routeOf = (url: URL) => url.pathname.slice(base.length).replace(/^\/+|\/+$/g, '');
  const route = $derived(routeOf(page.url));
  const LABELS: Record<string, string> = {
    tree: 'Tree',
    skills: 'Skills',
    items: 'Items',
    calcs: 'Calcs',
    configuration: 'Config',
    notes: 'Notes',
    party: 'Party'
  };

  // a page's code loads before it opens
  beforeNavigate((nav) => {
    if (!nav.to || nav.willUnload || nav.to.url.pathname === nav.from?.url.pathname) return;
    const to = routeOf(nav.to.url);
    if (!LABELS[to]) return;
    const job = startJob(`page:${to}`, `Opening ${LABELS[to]}`).phase('Loading the page');
    nav.complete.then(
      () => job.end(),
      () => job.end('cancelled')
    );
  });

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
    track(openImportLink(text), 'import', 'Importing the linked build', { long: true })
      .then(() => goto(importLanding(new URL(location.href), base), { replaceState: true }))
      .catch((e: Error) => {
        logError(e);
        linkError = e.message;
      });
  };

  const boot = browser ? startJob('boot', 'Starting the calculation engine', { long: true }).phase('Starting the worker') : undefined;
  const dataJob = () => startJob('data', 'Loading game data', { long: true });
  let dataLoad: ReturnType<typeof dataJob> | undefined;
  const dataStage = stageReporter(() => dataLoad);

  let wasmLoading = $state(true);
  let dataLoading = $state(false);
  // without a top bar, the activity indicator floats under the toasts (Toasts.svelte)
  const floatingActivity = $derived(!($game && !wasmLoading) || phone.current);

  let loadingMessage = $state('Initializing...');
  let loadingStage = $state('');
  // a boot takes a few seconds even on a loaded machine; one still running after this is stuck, and a reload starts it
  // over with fresh workers
  const BOOT_STUCK_MS = 30_000;
  let bootStuck = $state(false);

  // without its game data a build opens empty, so the reason stays on screen with a retry; a PoE1 one replaces the
  // failed data job's toast
  const dataErrorToast = (key: string, heading: string, error: string) =>
    addToast(`${heading}\n${error}`, { key, role: 'alert', tone: 'error', dismiss: false, actions: [{ label: 'Retry', run: () => location.reload() }] });
  const failData = (e: Error) => {
    dataLoad?.end('error', e);
    logError(e);
    dataLoading = false;
    dataErrorToast(dataLoad ? jobToastKey(dataLoad.id) : 'data-error', 'The game data could not be loaded', dataErrorText(e.message));
  };

  // the worker downloads the wasm itself, so the page never holds or copies it; the boot job follows the download
  // a handle the job calls return cannot cross back to the worker, so these return nothing
  const wasmReport: WasmReport = {
    phase: (name) => {
      boot?.phase(name);
      if (name === 'Decompressing the engine') loadingMessage = 'Decompressing wasm runtime...';
    },
    bytes: (done, total) => {
      boot?.progress(done, total, 'bytes');
    },
    fetched: () => {}
  };
  const assetUrl = (path: string) => new URL(assets + path, location.href).href;

  // Launch.lua:158-161
  const updateKey = (e: KeyboardEvent) => {
    if (e.defaultPrevented || !(e.ctrlKey || e.metaKey) || e.altKey || e.shiftKey || e.key.toLowerCase() !== 'u' || get(devMode)) return;
    e.preventDefault();
    checkForUpdate().catch(logError);
  };

  if (browser) {
    followLongJobs();
    followConnection();
    poe2DataError.subscribe((e) => (e ? dataErrorToast('poe2-data-error', 'The PoE2 game data could not be loaded', e) : removeToast('poe2-data-error')));
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
      boot?.end('error', 'Failed to initialize worker');
    } else {
      const worker = syncWrap;
      setArtResolver(async (queries) => {
        await backendLoaded;
        return worker.ItemArtPaths(queries);
      });
      const silent = setTimeout(() => {
        loadingMessage = `The calculation worker did not start in ${BOOT_STUCK_MS / 1000} s.`;
        bootStuck = true;
      }, BOOT_STUCK_MS);
      syncWrap.booted
        .then((booted) => {
          clearTimeout(silent);
          if (booted) {
            boot?.end();
            wasmLoading = false;
            dataLoad = dataJob();
            syncWrap
              ?.loadData(
                // eslint-disable-next-line @typescript-eslint/require-await
                proxy(async (stage: string, done: number, total: number) => dataStage(stage, done, total))
              )
              .then(() => {
                dataLoad?.end();
                get(markBackendAsLoaded)();
              })
              .catch(failData);
            return;
          }

          syncWrap
            .fetchWasm(assetUrl(engineFile(WASM_FILE, runningEngine)), assetUrl(engineFile(WASM_FALLBACK, runningEngine)), proxy(wasmReport))
            .then((size) => {
              console.log('wasm runtime size:', size);

              loadingMessage = 'Booting...';
              boot?.phase('Booting the engine');
              const stuck = setTimeout(() => {
                if (fallBackToGo(`it took over ${BOOT_STUCK_MS / 1000} s to boot`)) return;
                loadingMessage = 'Booting is taking much longer than it should.';
                bootStuck = true;
              }, BOOT_STUCK_MS);

              syncWrap
                ?.bootFetched(proxy(onTick), proxy(buildChannel), runningEngine)
                .then(async () => {
                  clearTimeout(stuck);
                  bootStuck = false;
                  console.log('worker booted');
                  performance.mark('pob:wasm-booted');

                  boot?.end();
                  wasmLoading = false;
                  dataLoading = true;
                  dataLoad = dataJob();
                  await syncWrap?.loadData(
                    // eslint-disable-next-line @typescript-eslint/require-await
                    proxy(async (stage: string, done: number, total: number) => {
                      loadingStage = stage;
                      dataStage(stage, done, total);
                    })
                  );

                  dataLoading = false;
                  dataLoad?.end();
                  performance.mark('pob:data-loaded');

                  get(markBackendAsLoaded)();
                })
                .catch((e: Error) => {
                  if (dataLoading) return failData(e);
                  logError(e);
                  if (!wasmLoading) return;
                  clearTimeout(stuck);
                  boot?.end('error', e);
                  if (fallBackToGo(`it did not boot (${e.message})`)) return;
                  loadingMessage = `Booting failed: ${e.message}`;
                  bootStuck = true;
                });
            })
            .catch((e: Error) => {
              logError(e);
              boot?.end('error', e);
              if (fallBackToGo(`its wasm did not load (${e.message})`)) return;
              loadingMessage = e.message;
              bootStuck = true;
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

<Dialog open={!!linkError} title="Build import" size="md" onclose={() => (linkError = undefined)}>
  <p role="alert" class="m-0 text-13">The linked build could not be opened: {linkError}</p>
  {#snippet actions()}
    <DialogActions confirm="OK" cancel={false} onconfirm={() => (linkError = undefined)} />
  {/snippet}
</Dialog>

<svelte:window onkeydown={updateKey} />
<Toasts activity={floatingActivity} />
<ActivityBar />
