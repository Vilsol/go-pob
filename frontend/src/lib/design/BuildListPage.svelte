<script lang="ts">
  import type { Snippet } from 'svelte';
  import BuildTable from './BuildTable.svelte';
  import FolderTree from './FolderTree.svelte';
  import { buildId, leaf, type ColKey, type Drag, type Sort } from './builds';
  import Icon from './Icon.svelte';
  import Sheet from './mobile/Sheet.svelte';
  import { phoneMedia } from './mobile/mobile';
  import type { SavedBuild } from './fixtures/app';

  type Recalc = { done: number; total: number; name: string };
  type Props = {
    builds: SavedBuild[];
    phoneLayout?: 'cards' | 'table';
    folders: string[];
    classes: string[];
    gameVersion: string;
    folder: string;
    open: Record<string, boolean>;
    fresh?: Record<string, boolean>;
    recalc?: Recalc | null;
    q?: string;
    sort?: Sort;
    colMenuOpen?: boolean;
    importHref?: string;
    newHref?: string;
    columns?: ColKey[];
    onfolder: (path: string) => void;
    ontoggle: (path: string) => void;
    onmove: (id: string, dest: string) => void;
    onopen?: (id: string) => void;
    oncopy?: (id: string) => void;
    onrename?: (id: string) => void;
    ondelete?: (id: string) => void;
    onpaste?: (id: string, cut: boolean) => void;
    onnewfolder?: () => void;
    onimport?: () => void;
    onnew?: () => void;
    onrecalc?: () => void;
    onabout?: () => void;
    onsettings?: () => void;
    /* Main.lua:212-229 */
    update?: { running: boolean; ready: boolean; oncheck: () => void; onapply: () => void };
    dialogs?: Snippet;
  };
  let {
    builds,
    phoneLayout,
    folders,
    classes,
    gameVersion,
    folder,
    open,
    fresh = {},
    recalc = null,
    q = $bindable(''),
    sort = $bindable({ key: 'edited', dir: 'desc' }),
    colMenuOpen = $bindable(false),
    importHref = '#import',
    newHref = '#new',
    columns,
    onfolder,
    ontoggle,
    onmove,
    onopen,
    oncopy,
    onrename,
    ondelete,
    onpaste,
    onnewfolder,
    onimport,
    onnew,
    onrecalc,
    onabout,
    onsettings,
    update,
    dialogs
  }: Props = $props();

  let drag = $state<Drag | null>(null);
  let foldersOpen = $state(false);
  let note = $state('');
  const move = (id: string, dest: string) => {
    drag = null;
    note = `${builds.find((b) => buildId(b) === id)?.name ?? id} moved to ${dest}`;
    onmove(id, dest);
  };
  // the tree's own New folder button lives in the folders sheet on a phone; this one sits beside the sheet's button
  const phone = phoneMedia();
</script>

<div class="page max-phone:h-dvh">
  <header class="box-border flex h-14 shrink-0 items-center gap-4 border-b border-line bg-surface-1 px-6 max-phone:gap-2 max-phone:px-3">
    <span class="text-18 font-semibold">go-pob</span>
    <span class="note max-phone:hidden">game data {gameVersion}</span>
    <div class="grow"></div>
    {#if onimport}<button class="btn btn-md" onclick={onimport}>Import build</button>{:else}<a href={importHref} class="btn btn-md">Import build</a>{/if}
    {#if onnew}<button class="btn btn-md btn-primary px-4" onclick={onnew}>New build</button>{:else}<a href={newHref} class="btn btn-md btn-primary px-4"
        >New build</a
      >{/if}
    {#if update?.ready}
      <button class="btn btn-md text-code-green" title="Switch to the new version (reloads the page)" onclick={update.onapply}>Update Ready</button>
    {:else if update}
      <button class="btn btn-md" disabled={update.running} onclick={update.oncheck}>{update.running ? 'Checking...' : 'Check for Update'}</button>
    {/if}
    <button
      class="btn btn-icon btn-md"
      aria-haspopup="dialog"
      aria-label="About go-pob"
      title="About: version, game data, build settings, dependencies"
      onclick={onabout}><Icon name="info" /></button>
    <button class="btn btn-icon btn-md" aria-haspopup="dialog" aria-label="Settings" title="Settings" onclick={onsettings}><Icon name="gear" /></button>
  </header>
  <div class="flex h-11 shrink-0 items-center gap-2 border-b border-line pr-3 phone:hidden">
    <button type="button" class="btn h-full min-w-0 grow justify-start rounded-none border-0 px-3" aria-haspopup="dialog" onclick={() => (foldersOpen = true)}
      ><span class="truncate">Folders · {folder || 'All builds'} ▾</span></button>
    {#if onnewfolder && phone.current}<button class="btn btn-sm shrink-0" title="Make a folder in {folder ? leaf(folder) : 'All builds'}" onclick={onnewfolder}
        >New folder</button
      >{/if}
  </div>
  <div class="flex min-h-0 grow">
    {#snippet tree()}
      <FolderTree
        {folders}
        {builds}
        {folder}
        {open}
        {drag}
        {note}
        onfolder={(path: string) => {
          foldersOpen = false;
          onfolder(path);
        }}
        {ontoggle}
        ondrag={(d) => {
          drag = d;
          if (d) note = '';
        }}
        onmove={move}
        {onnewfolder} />
    {/snippet}
    <div class="contents max-phone:hidden">{@render tree()}</div>
    <Sheet bind:open={foldersOpen} title="Folders"
      ><div class="flex [&>nav]:w-full [&>nav]:border-r-0">
        {#if foldersOpen}{@render tree()}{/if}
      </div></Sheet>
    <BuildTable
      bind:sort
      {phoneLayout}
      {builds}
      {folders}
      {folder}
      {classes}
      {gameVersion}
      {fresh}
      {recalc}
      {drag}
      {columns}
      bind:q
      bind:colMenuOpen
      {onfolder}
      ondrag={(d) => {
        drag = d;
        if (d) note = '';
      }}
      onmove={move}
      {onopen}
      {oncopy}
      {onrename}
      {ondelete}
      {onpaste}
      {onrecalc} />
  </div>
  {@render dialogs?.()}
</div>
