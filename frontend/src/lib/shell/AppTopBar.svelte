<script lang="ts">
  import { buildName, unsavedName } from './build_name.svelte';
  import { resetAccountSession } from './account_session.svelte';
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
  import { WIKI_KEY, hoveredWikiName, openWiki } from '$lib/design/wiki';

  const tabs = [
    ['Tree', 'tree'],
    ['Skills', 'skills'],
    ['Items', 'items'],
    ['Calcs', 'calcs'],
    ['Config', 'configuration'],
    ['Notes', 'notes'],
    ['Party', 'party'],
    ['Compare', 'compare']
  ].map(([label, path], i) => ({ label, href: `${base}/${path}`, key: String(i + 1) }));

  const name = $derived(buildName(currentBuildPath.current));

  let dirty = $state(false);
  let seen = 0;
  $effect(() => {
    if (!$currentBuild) {
      seen = 0;
      dirty = false;
      return;
    }
    if (seen++ > 0) dirty = true;
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
  let closeAfterSave = false;
  const closeBuild = () => {
    closeAfterSave = false;
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
  $effect(() => {
    if (!saveAs && !savePrompt) closeAfterSave = false;
  });
  const onsave = () => {
    const path = currentBuildPath.current;
    if (!path) {
      saveAs = true;
      return;
    }
    syncWrap
      .SaveBuildAs(path)
      .then(() => {
        dirty = false;
        if (closeAfterSave) closeBuild();
      })
      .catch((e: Error) => {
        logError(e);
        closeAfterSave = false;
        saveError = saveErrorText(path);
      });
  };

  const leave = () => {
    if (dirty) savePrompt = true;
    else closeBuild();
  };
  const unload = (e: BeforeUnloadEvent) => {
    if (!dirty) return;
    e.preventDefault();
  };
</script>

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
<SaveAsDialog
  bind:open={saveAs}
  onsaved={() => {
    dirty = false;
    if (closeAfterSave) closeBuild();
  }} />
<Dialog bind:open={savePrompt} title="Save Changes" class="w-100 p-4.5">
  <div class="flex flex-col gap-3 text-13">
    <p class="m-0">This build has unsaved changes.<br />Do you want to save them now?</p>
    <div class="flex gap-1.5">
      <Button
        size="sm"
        variant="primary"
        onclick={() => {
          closeAfterSave = true;
          savePrompt = false;
          onsave();
        }}>Save</Button>
      <Button
        size="sm"
        onclick={() => {
          savePrompt = false;
          closeBuild();
        }}>Don't Save</Button>
      <Button size="sm" variant="ghost" onclick={() => (savePrompt = false)}>Cancel</Button>
    </div>
  </div>
</Dialog>
<Dialog open={!!saveError} title="Error" class="w-100 p-4.5" onclose={() => (saveError = undefined)}>
  <p role="alert" class="m-0 text-13 whitespace-pre-line">{saveError}</p>
  {#snippet actions()}<Button onclick={() => (saveError = undefined)}>OK</Button>{/snippet}
</Dialog>
<Dialog bind:open={naming} title="Set Name" class="w-100 p-4.5">
  <form
    class="flex flex-col gap-2 text-13"
    onsubmit={(e) => {
      e.preventDefault();
      saveLoadout();
    }}>
    <label class="flex flex-col gap-1"
      ><span>Enter name for this loadout:</span><input
        class="input h-7.5"
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
