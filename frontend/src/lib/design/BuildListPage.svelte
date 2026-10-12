<script lang="ts">
  import type { Snippet } from 'svelte';
  import BuildTable from './BuildTable.svelte';
  import FolderTree from './FolderTree.svelte';
  import { buildId, type ColKey, type Drag, type Sort } from './builds';
  import Icon from './Icon.svelte';
  import type { SavedBuild } from './fixtures/app';

  type Recalc = { done: number; total: number; name: string };
  type Props = {
    builds: SavedBuild[];
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
  let note = $state('');
  const move = (id: string, dest: string) => {
    drag = null;
    note = `${builds.find((b) => buildId(b) === id)?.name ?? id} moved to ${dest}`;
    onmove(id, dest);
  };
</script>

<div class="page">
  <header class="box-border flex h-14 shrink-0 items-center gap-4 border-b border-line bg-surface-1 px-6">
    <span class="text-18 font-semibold">go-pob</span>
    <span class="note">game data {gameVersion}</span>
    <div class="grow"></div>
    {#if onimport}<button class="btn btn-md" onclick={onimport}>Import build</button>{:else}<a href={importHref} class="btn btn-md">Import build</a>{/if}
    {#if onnew}<button class="btn btn-md btn-primary px-4" onclick={onnew}>New build</button>{:else}<a href={newHref} class="btn btn-md btn-primary px-4"
        >New build</a
      >{/if}
    {#if update?.ready}
      <button class="btn btn-md text-[#50e050]" title="Switch to the new version (reloads the page)" onclick={update.onapply}>Update Ready</button>
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
  <div class="flex min-h-0 grow">
    <FolderTree
      {folders}
      {builds}
      {folder}
      {open}
      {drag}
      {note}
      {onfolder}
      {ontoggle}
      ondrag={(d) => {
        drag = d;
        if (d) note = '';
      }}
      onmove={move}
      {onnewfolder} />
    <BuildTable
      bind:sort
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
