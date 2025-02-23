<script lang="ts">
  import Input from '$lib/components/Input.svelte';
  import { syncWrap } from '$lib/go/worker';
  import { Emitter, logError } from '$lib/utils';
  import { storage } from '$lib/types';
  import { openOverlay } from '$lib/overlay';
  import TextDialog from '$lib/components/overlays/TextDialog.svelte';

  interface Props {
    path: string;
    refresh?: Emitter<void>;
    selected: storage.DirEntry | undefined;
    hideSearch?: boolean;
    hideSort?: boolean;
    hideBuilds?: boolean;
    showNewFolder?: boolean;
    openBuild?: (entry: storage.DirEntry) => void;
  }

  let { path = $bindable(), refresh, selected = $bindable(), hideSearch, hideSort, hideBuilds, showNewFolder, openBuild }: Props = $props();

  let search = $state('');
  let sortBy = $state<'name' | 'class' | 'last_edit' | 'level'>('name');

  let entries = $state<storage.DirEntry[]>([]);

  let filtered = $derived(entries.filter((e) => e.Name.toLowerCase().includes(search.toLowerCase())));

  let sorted = $derived.by(() =>
    filtered.sort((a, b) => {
      switch (sortBy) {
        case 'name':
          return a.Name.localeCompare(b.Name);
        case 'class':
          if (a.Class === b.Class) {
            return a.Name.localeCompare(b.Name);
          }

          return a.Class.localeCompare(b.Class);
        case 'last_edit':
          if (a.LastEdit === b.LastEdit) {
            return a.Name.localeCompare(b.Name);
          }

          return new Date(a.LastEdit).getTime() - new Date(b.LastEdit).getTime();
        case 'level':
          if (a.Level === b.Level) {
            return a.Name.localeCompare(b.Name);
          }

          return a.Level - b.Level;
      }
      return 0;
    })
  );

  const load = (dir: string) => {
    void syncWrap
      .ListBuilds(dir)
      .then((data) => {
        entries = data || [];
      })
      .catch(logError);
  };

  refresh?.on(() => {
    load(path);
  });

  $effect(() => {
    load(path);
  });

  const gotoIndex = (i: number) => {
    path = path
      .split('/')
      .filter((p) => !!p)
      .slice(0, i + 1)
      .join('/');
    selected = undefined;
  };

  const gotoPath = (entry: storage.DirEntry) => {
    path += '/' + entry.Name;
    selected = undefined;
  };

  const gotoStart = () => {
    path = '';
    selected = undefined;
  };

  const newFolder = () => {
    openOverlay({
      component: TextDialog,
      props: {
        title: 'New Folder',
        subtitle: 'Enter folder name:',
        confirm: 'Create',
        onconfirm: (name: string) => {
          void syncWrap
            .NewFolder(path + '/' + name)
            .then(() => {
              load(path);
            })
            .catch(logError);
        }
      }
    });
  };
</script>

<div class="flex flex-col gap-2 max-h-full h-full">
  <div class="flex flex-row gap-2">
    {#if !hideSearch}
      <Input prefix="Search:" classes="flex-1" bind:value={search} />
    {/if}

    {#if !hideSort}
      <div class="container select-wrapper">
        <select class="input" bind:value={sortBy}>
          <option value="name">Sort by Name</option>
          <option value="class">Sort by Class</option>
          <option value="last_edit">Sort by Last Edited</option>
          <option value="level">Sort by Level</option>
        </select>
      </div>
    {/if}
  </div>

  {#if showNewFolder}
    <div class="flex flex-row gap-16">
      <div>Folder:</div>
      <button class="container" type="button" onclick={newFolder}>New Folder</button>
    </div>
  {/if}

  <div class="flex flex-row gap-2 items-center">
    <button class="container" type="button" onclick={gotoStart}>Builds</button>
    <div>/</div>
    {#if path !== ''}
      {#each path?.split('/').filter((p) => !!p) as elem, i}
        <button class="container" type="button" onclick={() => gotoIndex(i)}>{elem}</button>
        <div>/</div>
      {/each}
    {/if}
  </div>

  <div class="flex flex-col border-2 min-h-32 flex-1 overflow-auto bg-black">
    {#each sorted.filter((e) => e.Type === 'dir') as elem}
      <button class="entry" class:selected={selected === elem} ondblclick={() => gotoPath(elem)} onclick={() => (selected = elem)}>>> {elem.Name}</button>
    {/each}
    {#if !hideBuilds}
      {#each sorted.filter((e) => e.Type === 'file') as elem}
        <button class="entry flex justify-between" class:selected={selected === elem} ondblclick={() => openBuild?.(elem)} onclick={() => (selected = elem)}>
          <span>
            {elem.Name}
          </span>
          <span>
            Level {elem.Level}
            {elem.Class}
          </span>
        </button>
      {/each}
    {/if}
  </div>
</div>

<style lang="postcss">
  .entry {
    @apply px-2 py-0.5 hover:bg-neutral-700 cursor-pointer text-left border-b;
  }

  .selected {
    @apply bg-neutral-800;
  }
</style>
