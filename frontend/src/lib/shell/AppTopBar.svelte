<script lang="ts">
  import { buildName, opening, unsavedName } from './build_name.svelte';
  import { resetAccountSession } from './account_session.svelte';
  import { beforeReplace } from './link_import';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { currentBuild, currentBuildPath, devMode } from '$lib/global.js';
  import { showWarnings } from '$lib/display/item_options';
  import { sidebar } from '$lib/build_warnings';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import TopBar from '$lib/design/TopBar.svelte';
  import { onMount, tick } from 'svelte';
  import type { UndoView } from '$lib/go/undo';
  import SaveAsDialog from './SaveAsDialog.svelte';
  import { saveErrorText } from './save_error';
  import AppSettings from './AppSettings.svelte';
  import AppAbout from './AppAbout.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import Button from '$lib/design/Button.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import { WIKI_KEY, hoveredWikiName, openWiki } from '$lib/design/wiki';
  import type { Snippet } from 'svelte';
  import MobileFrame from '$lib/design/mobile/MobileFrame.svelte';
  import StatsStrip from '$lib/design/mobile/StatsStrip.svelte';
  import BuildSheets, { type BuildSheet } from '$lib/design/mobile/BuildSheets.svelte';
  import { moreTabs, primaryTabs, routeTabs } from '$lib/design/mobile/mobile';
  import AppSidebar from './AppSidebar.svelte';
  import TimelessDialog from './TimelessDialog.svelte';
  import { statPins } from './stat_pins.svelte';
  import { editDelta } from '$lib/display/stat_delta';

  // phone: this bar becomes the phone frame around the page, with the sidebar in its stats sheet
  type Props = { phone?: boolean; label?: string; mainClass?: string; children?: Snippet };
  let { phone = false, label = '', mainClass = '', children }: Props = $props();

  const tabs = [
    ['Tree', 'tree'],
    ['Skills', 'skills'],
    ['Items', 'items'],
    ['Calcs', 'calcs'],
    ['Config', 'configuration'],
    ['Notes', 'notes'],
    ['Party', 'party'],
    ['Compare', 'compare']
  ].map(([text, path], i) => ({ label: text, href: `${base}/${path}`, key: String(i + 1) }));

  const name = $derived(buildName(currentBuildPath.current));

  let dirty = $state(false);
  let shown: unknown;
  $effect(() => {
    const build = $currentBuild;
    if (!build) {
      shown = undefined;
      dirty = false;
      return;
    }
    if (build === shown) dirty = true;
    else {
      shown = build;
      dirty = opening.modFlag;
    }
  });

  let undo = $state<UndoView>({ steps: [], undone: 0, max: 50 });
  let undoSeq = 0;
  const showUndo = (read: Promise<UndoView> | undefined) => {
    const seq = ++undoSeq;
    read
      ?.then((v) => {
        if (seq === undoSeq) undo = v;
      })
      .catch(logError);
  };
  $effect(() => {
    void $currentBuild;
    showUndo(syncWrap?.GetUndoHistoryWhenRecorded());
  });
  let now = $state(Date.now());
  onMount(() => {
    const ticks = new MutationObserver(() => showUndo(syncWrap?.GetUndoHistoryWhenRecorded()));
    ticks.observe(document.documentElement, { attributes: true, attributeFilter: ['data-calc-seq'] });
    const clock = setInterval(() => (now = Date.now()), 30_000);
    return () => {
      ticks.disconnect();
      clearInterval(clock);
    };
  });
  const ago = (at: number) => {
    const min = Math.floor((now - at) / 60_000);
    return min < 1 ? 'just now' : min < 60 ? `${min} min ago` : new Date(at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  };
  const history = $derived(undo.steps.map((s) => ({ action: s.action, page: s.page, when: ago(s.at) })));
  let saveAs = $state(false);
  let settings = $state(false);
  let about = $state(false);
  let aboutTab = $state('about');
  let helpSection = $state('Loadouts');
  // Main.lua:439-448, ItemTools.lua:380-409
  const VIEW_MODES: Record<string, string> = { configuration: 'config' };
  const onhelpkey = (e: KeyboardEvent) => {
    if (e.key !== WIKI_KEY || e.ctrlKey || e.altKey || e.metaKey || e.shiftKey) return;
    e.preventDefault();
    const wiki = hoveredWikiName();
    if (wiki) {
      openWiki(wiki);
      return;
    }
    const segment = page.url.pathname.slice(base.length).split('/').filter(Boolean)[0] ?? '';
    helpSection = `${VIEW_MODES[segment] ?? segment} tab`;
    aboutTab = 'help';
    about = true;
  };
  // Main.lua:230-241
  let version = $state<string>();
  onMount(() => {
    Promise.resolve(syncWrap?.BuildInfo())
      .then((bi) => (version = bi?.Main?.Version || '(devel)'))
      .catch(logError);
  });

  // Build.lua:301-415, 753-878
  const HEADER = 'Loadouts:';
  const NEW = 'New Loadout';
  const SYNC = 'Sync';
  const HELP = 'Help >>';
  let loadoutList = $state.raw<{ loadouts: string[]; selected: number }>({ loadouts: [], selected: -1 });
  let loadoutShown = $state<string>(HEADER);
  const readLoadouts = () =>
    syncWrap
      ?.GetLoadouts()
      .then((l) => {
        loadoutList = l ?? { loadouts: [], selected: -1 };
        loadoutShown = loadoutList.selected >= 0 ? loadoutList.loadouts[loadoutList.selected] : HEADER;
      })
      .catch(logError);
  $effect(() => {
    void $currentBuild;
    void readLoadouts();
  });
  const loadoutOptions = $derived([HEADER, ...loadoutList.loadouts, NEW, SYNC, HELP]);
  let naming = $state(false);
  let loadoutName = $state('New Loadout');
  const chooseLoadout = (choice: string) => {
    loadoutShown = choice;
    if (choice === NEW) {
      loadoutName = 'New Loadout';
      naming = true;
    } else if (choice === HELP) {
      // Build.lua:311-315
      aboutTab = 'help';
      helpSection = 'Loadouts';
      about = true;
    } else if (choice !== HEADER && choice !== SYNC) {
      void syncWrap
        ?.SelectLoadout(choice)
        .then(() => currentBuild.set($currentBuild))
        .catch(logError);
      return;
    }
    void tick().then(readLoadouts);
  };
  const saveLoadout = () => {
    if (!/\S/.test(loadoutName)) return;
    naming = false;
    void syncWrap
      ?.NewLoadout(loadoutName)
      .then(() => currentBuild.set($currentBuild))
      .catch(logError);
  };

  // Build.lua:1352-1379
  let savePrompt = $state(false);
  // Build.lua:2069-2072
  let saveError = $state<string>();
  let leaving: () => void;
  let afterSave: (() => void) | undefined;
  const closeBuild = () => {
    void tick().then(() =>
      setTimeout(() => {
        void syncWrap.ClearBuild();
        unsavedName.imported = false;
        unsavedName.predefined = undefined;
        resetAccountSession();
        currentBuildPath.current = undefined;
      }, 100)
    );
  };
  let saving = $state(false);
  $effect(() => {
    if (!saveAs && !savePrompt && !saving) afterSave = undefined;
  });
  const saved = () => {
    dirty = false;
    const next = afterSave;
    afterSave = undefined;
    next?.();
  };
  const onsave = () => {
    const path = currentBuildPath.current;
    if (!path) {
      saveAs = true;
      return;
    }
    saving = true;
    syncWrap
      .SaveBuildAs(path)
      .then(saved)
      .catch((e: Error) => {
        logError(e);
        afterSave = undefined;
        saveError = saveErrorText(path);
      })
      .finally(() => (saving = false));
  };

  const ask = (then: () => void) => {
    leaving = then;
    if (dirty) savePrompt = true;
    else then();
  };
  const leave = () => ask(closeBuild);
  onMount(() => {
    beforeReplace.ask = ask;
    return () => {
      if (beforeReplace.ask === ask) beforeReplace.ask = undefined;
    };
  });
  let phoneSheet = $state<BuildSheet>();
  let timelessOpen = $state(false);
  const phoneActions = $derived([
    { key: 'history', label: 'Undo history', note: `${undo.steps.length - undo.undone} steps` },
    { key: 'loadout', label: 'Loadout', note: loadoutShown === HEADER ? 'None' : loadoutShown },
    ...(currentBuildPath.current ? [{ key: 'saveas', label: 'Save as…' }] : []),
    { key: 'issues', label: 'Build issues', note: String($showWarnings ? ($sidebar?.warnings.length ?? 0) : 0) },
    { key: 'settings', label: 'Settings' },
    { key: 'about', label: 'About' }
  ]);
  const phoneAction = (key: string) => {
    if (key === 'history' || key === 'loadout' || key === 'issues') phoneSheet = key;
    else if (key === 'saveas') saveAs = true;
    else if (key === 'settings') settings = true;
    else if (key === 'about') {
      aboutTab = 'about';
      about = true;
    }
  };
  const unload = (e: BeforeUnloadEvent) => {
    if (!dirty || beforeReplace.reloading) return;
    e.preventDefault();
  };
</script>

{#if phone}
  <MobileFrame
    {name}
    current={page.url.pathname}
    {label}
    {mainClass}
    tabs={routeTabs(primaryTabs, base)}
    more={routeTabs(moreTabs, base)}
    {dirty}
    canUndo={undo.undone < undo.steps.length}
    canRedo={undo.undone > 0}
    actions={phoneActions}
    onnavigate={(href: string) => {
      if (href === '#timeless') timelessOpen = true;
      else void goto(href);
    }}
    onback={leave}
    onundo={() => showUndo(syncWrap?.Undo())}
    onredo={() => showUndo(syncWrap?.Redo())}
    {onsave}
    onaction={phoneAction}>
    {#snippet strip(open: () => void)}
      {#if $sidebar}<StatsStrip sidebar={$sidebar} pinned={statPins.ids} delta={$editDelta} onopen={open} />{/if}
    {/snippet}
    {#snippet stats()}<AppSidebar fill />{/snippet}
    {@render children?.()}
  </MobileFrame>
  <BuildSheets
    bind:open={phoneSheet}
    {history}
    undone={undo.undone}
    loadouts={loadoutOptions.filter((o) => o !== HEADER)}
    loadout={loadoutShown}
    issues={$showWarnings ? ($sidebar?.warnings ?? []) : []}
    onhistory={(i: number) => showUndo(syncWrap?.UndoTo(i))}
    onloadout={chooseLoadout} />
  <TimelessDialog bind:open={timelessOpen} />
{:else}
  <TopBar
    {name}
    {tabs}
    current={page.url.pathname}
    backHref="{base}/"
    importHref="{base}/import"
    {history}
    undone={undo.undone}
    historyMax={undo.max}
    onundo={() => showUndo(syncWrap?.Undo())}
    onredo={() => showUndo(syncWrap?.Redo())}
    onhistory={(i: number) => showUndo(syncWrap?.UndoTo(i))}
    loadouts={loadoutOptions}
    loadout={loadoutShown}
    onloadout={chooseLoadout}
    issues={$showWarnings ? ($sidebar?.warnings ?? []) : []}
    logs={[]}
    {dirty}
    onnavigate={(t: { href: string }, e: Event) => {
      if (e instanceof KeyboardEvent) void goto(t.href);
    }}
    onback={(e: MouseEvent) => {
      e.preventDefault();
      leave();
    }}
    {onsave}
    onimport={() => void goto(`${base}/import`, { state: { focusImport: true } })}
    onsaveas={currentBuildPath.current ? () => (saveAs = true) : undefined}
    onsettings={() => (settings = true)}
    onabout={() => {
      aboutTab = 'about';
      about = true;
    }}
    {version}
    devMode={$devMode} />
{/if}
<SaveAsDialog bind:open={saveAs} onsaved={saved} />
<Dialog phone="centred" bind:open={savePrompt} title="Save Changes" size="sm">
  <div class="flex flex-col gap-3 text-13">
    <p class="m-0">This build has unsaved changes.<br />Do you want to save them now?</p>
    <div class="flex gap-1.5">
      <Button
        size="sm"
        variant="primary"
        onclick={() => {
          afterSave = leaving;
          savePrompt = false;
          onsave();
        }}>Save</Button>
      <Button
        size="sm"
        onclick={() => {
          savePrompt = false;
          leaving();
        }}>Don't Save</Button>
      <Button size="sm" variant="ghost" onclick={() => (savePrompt = false)}>Cancel</Button>
    </div>
  </div>
</Dialog>
<Dialog phone="centred" open={!!saveError} title="Error" size="sm" onclose={() => (saveError = undefined)}>
  <p role="alert" class="m-0 text-13 whitespace-pre-line">{saveError}</p>
  {#snippet actions()}
    <DialogActions confirm="OK" cancel={false} onconfirm={() => (saveError = undefined)} />
  {/snippet}
</Dialog>
<Dialog bind:open={naming} title="Set Name" size="sm">
  <form
    class="flex flex-col gap-2 text-13"
    onsubmit={(e) => {
      e.preventDefault();
      saveLoadout();
    }}>
    <label class="flex flex-col gap-1"
      ><span>Enter name for this loadout:</span><input
        class="input input-sm"
        aria-label="Enter name for this loadout"
        bind:value={loadoutName}
        maxlength="100" /></label>
    <div class="flex gap-1.5">
      <Button size="sm" variant="primary" type="submit" disabled={!/\S/.test(loadoutName)}>Save</Button>
      <Button size="sm" type="button" variant="ghost" onclick={() => (naming = false)}>Cancel</Button>
    </div>
  </form>
</Dialog>
<svelte:window onbeforeunload={unload} onkeydown={onhelpkey} />
<AppSettings bind:open={settings} />
<AppAbout bind:open={about} bind:tab={aboutTab} helpSection={aboutTab === 'help' ? helpSection : undefined} />
