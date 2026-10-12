<script lang="ts">
  import { unsavedName } from './build_name.svelte';
  import { onMount } from 'svelte';
  import { goto, pushState } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import BuildListPage from '$lib/design/BuildListPage.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import Button from '$lib/design/Button.svelte';
  import { buildId, leaf } from '$lib/design/builds';
  import type { SavedBuild } from '$lib/design/fixtures/app';
  import { syncWrap } from '$lib/go/worker';
  import { currentBuildPath, devMode } from '$lib/global.js';
  import { logError } from '$lib/utils';
  import PromptDialog from './PromptDialog.svelte';
  import AppAbout from './AppAbout.svelte';
  import AppSettings from './AppSettings.svelte';
  import { CLASSES, LIVE_COLUMNS, loadTree, pathOf } from './build_list';
  import { loadBuildSort, saveBuildSort } from '$lib/display/build_sort';
  import { defaultCharLevel, defaultGemQuality } from '$lib/display/item_options';
  import { get } from 'svelte/store';
  import { applyUpdate, checkForUpdate, updateAvailable, updateCheckRunning } from '$lib/pwa';

  let builds = $state<SavedBuild[]>([]);
  let folders = $state<string[]>([]);
  let gameVersion = $state('');
  let open = $state<Record<string, boolean>>({});
  let about = $state(false);
  let settings = $state(false);
  const folder = $derived(page.state.folder ?? '');

  type Prompt = { title: string; label: string; value: string; confirm: string; run: (value: string) => void };
  let prompt = $state<Prompt>({ title: '', label: '', value: '', confirm: '', run: () => {} });
  let prompting = $state(false);
  type Confirm = { title: string; body: string; confirm: string; run: () => void };
  let ask = $state<Confirm>({ title: '', body: '', confirm: '', run: () => {} });
  let asking = $state(false);

  const reload = () =>
    loadTree((dir) => syncWrap.ListBuilds(dir), gameVersion)
      .then((t) => {
        builds = t.builds;
        folders = t.folders;
      })
      .catch(logError);

  onMount(() => {
    syncWrap
      .GameDataVersion()
      .then((v) => {
        gameVersion = v;
        return reload();
      })
      .catch(logError);
  });

  const ancestors = (f: string) => f.split('/').map((_, i, a) => a.slice(0, i + 1).join('/'));
  const go = (f: string) => {
    for (const a of f ? ancestors(f) : []) open[a] = true;
    if (f !== folder) pushState('', { folder: f });
  };

  const byId = (id: string) => builds.find((b) => buildId(b) === id);
  const ask_ = (c: Confirm) => {
    ask = c;
    asking = true;
  };
  const prompt_ = (p: Prompt) => {
    prompt = p;
    prompting = true;
  };

  const toTarget = (b: SavedBuild, dest: string, name: string, act: (from: string, to: string) => Promise<unknown>) => {
    const from = pathOf(b.folder, b.name);
    const to = pathOf(dest, name);
    if (to === from) return;
    const run = () => void act(from, to).then(reload).catch(logError);
    if (builds.some((x) => x.folder === dest && x.name === name)) {
      ask_({
        title: `Replace “${name}”?`,
        body: `${dest ? leaf(dest) : 'All builds'} already holds a build named ${name}. It is replaced, and cannot be brought back.`,
        confirm: 'Replace',
        run
      });
    } else run();
  };

  const copyPrompt = (b: SavedBuild) =>
    prompt_({
      title: `Copy “${b.name}”`,
      label: 'Name of the copy',
      value: `${b.name} (copy)`,
      confirm: 'Copy',
      run: (name) => toTarget(b, b.folder, name, (from, to) => syncWrap.Copy(from, to))
    });

  // BuildList.lua:135-167
  // BuildList.lua:141-145, 156-160
  const pasteFolder = (f: string, cut: boolean) => {
    const name = leaf(f);
    const from = pathOf(f.slice(0, Math.max(0, f.length - name.length - 1)), name);
    if (folder === f || folder.startsWith(`${f}/`)) {
      ask_({ title: 'Error', body: `A folder cannot be ${cut ? 'moved' : 'copied'} into itself.`, confirm: 'OK', run: () => {} });
      return;
    }
    if (from === pathOf(folder, name)) {
      if (cut) return;
      prompt_({
        title: `Copy “${name}”`,
        label: 'Name of the copy',
        value: `${name} (copy)`,
        confirm: 'Copy',
        run: (to) => void syncWrap.CopyFolder(from, pathOf(folder, to)).then(reload).catch(logError)
      });
      return;
    }
    void (cut ? syncWrap.MoveFolder(from, pathOf(folder, name)) : syncWrap.CopyFolder(from, pathOf(folder, name))).then(reload).catch(logError);
  };
  const paste = (id: string, cut: boolean) => {
    if (id.startsWith('folder:')) {
      pasteFolder(id.slice('folder:'.length), cut);
      return;
    }
    const b = byId(id);
    if (!b) return;
    if (cut) {
      if (b.folder !== folder) toTarget(b, folder, b.name, (from, to) => syncWrap.Rename(from, to));
      return;
    }
    if (b.folder === folder) {
      copyPrompt(b);
      return;
    }
    let name = b.name;
    for (let i = 2; builds.some((x) => x.folder === folder && x.name === name); i++) name = `${b.name}[${i}]`;
    void syncWrap.Copy(pathOf(b.folder, b.name), pathOf(folder, name)).then(reload).catch(logError);
  };
  const openBuild = (id: string) => {
    const b = byId(id);
    if (!b) return;
    const path = pathOf(b.folder, b.name);
    currentBuildPath.current = path;
    unsavedName.imported = false;
    syncWrap.OpenBuild(path).catch(logError);
  };

  // Main.lua:539-541, 747
  let sort = $state(loadBuildSort());
  $effect(() => saveBuildSort($state.snapshot(sort)));

  const newBuild = () => {
    void syncWrap.NewBuild({ level: get(defaultCharLevel), gemQuality: get(defaultGemQuality) });
    unsavedName.imported = false;
    currentBuildPath.current = '';
  };
</script>

<BuildListPage
  update={$devMode ? undefined : { running: $updateCheckRunning, ready: $updateAvailable, oncheck: () => void checkForUpdate(), onapply: applyUpdate }}
  bind:sort
  {builds}
  {folders}
  classes={CLASSES}
  {gameVersion}
  {folder}
  {open}
  columns={[...LIVE_COLUMNS]}
  onfolder={go}
  ontoggle={(f: string) => (open[f] = !open[f])}
  onmove={(id: string, dest: string) => {
    const b = byId(id);
    if (b && b.folder !== dest) toTarget(b, dest, b.name, (from, to) => syncWrap.Rename(from, to));
  }}
  onopen={openBuild}
  oncopy={(id: string) => {
    const b = byId(id);
    if (b) copyPrompt(b);
  }}
  onpaste={paste}
  onrename={(id: string) => {
    const b = byId(id);
    if (!b) return;
    prompt_({
      title: `Rename “${b.name}”`,
      label: 'New name',
      value: b.name,
      confirm: 'Rename',
      run: (name) => toTarget(b, b.folder, name, (from, to) => syncWrap.Rename(from, to))
    });
  }}
  ondelete={(id: string) => {
    const b = byId(id);
    if (!b) return;
    ask_({
      title: `Delete “${b.name}”?`,
      body: 'The build is removed from this browser. Export it first to keep a copy.',
      confirm: 'Delete',
      run: () => void syncWrap.Delete(pathOf(b.folder, b.name)).then(reload).catch(logError)
    });
  }}
  onnewfolder={() =>
    prompt_({
      title: 'New folder',
      label: 'Folder name',
      value: '',
      confirm: 'Create',
      run: (name) => void syncWrap.NewFolder(pathOf(folder, name)).then(reload).catch(logError)
    })}
  onnew={newBuild}
  onimport={() => {
    newBuild();
    void goto(`${base}/import`);
  }}
  onabout={() => (about = true)}
  onsettings={() => (settings = true)}>
  {#snippet dialogs()}
    <AppAbout bind:open={about} />
    <AppSettings bind:open={settings} />
    <PromptDialog bind:open={prompting} title={prompt.title} label={prompt.label} value={prompt.value} confirm={prompt.confirm} onconfirm={prompt.run} />
    <Dialog bind:open={asking} title={ask.title}>
      <span class="text-13 text-text-2">{ask.body}</span>
      {#snippet actions()}
        <Button size="md" variant="ghost" class="px-3" onclick={() => (asking = false)}>Cancel</Button>
        <Button
          size="md"
          variant="danger-solid"
          onclick={() => {
            asking = false;
            ask.run();
          }}>{ask.confirm}</Button>
      {/snippet}
    </Dialog>
  {/snippet}
</BuildListPage>
