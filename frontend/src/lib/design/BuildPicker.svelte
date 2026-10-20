<script lang="ts">
  import Select from './Select.svelte';
  import Button from './Button.svelte';
  import SearchField from './SearchField.svelte';
  import Field from './Field.svelte';
  import Icon from './Icon.svelte';
  import Dialog from './Dialog.svelte';
  import { buildId, childrenOf, countIn, leaf, subpath, visibleRows, type ColKey, type Sort } from './builds';
  import type { SavedBuild } from './fixtures/app';

  type Props = {
    open?: boolean;
    title?: string;
    action?: string;
    builds: SavedBuild[] | undefined;
    folders: string[];
    folder?: string;
    sort?: Sort;
    onpick: (b: SavedBuild) => void;
  };
  let {
    open = $bindable(false),
    title = 'Choose a build',
    action = 'Choose',
    builds,
    folders,
    folder = $bindable(''),
    sort = $bindable({ key: 'edited', dir: 'desc' }),
    onpick
  }: Props = $props();

  const SORTS: { key: ColKey; label: string; dir: Sort['dir'] }[] = [
    { key: 'edited', label: 'Modified', dir: 'desc' },
    { key: 'name', label: 'Name', dir: 'asc' },
    { key: 'level', label: 'Level', dir: 'desc' },
    { key: 'cls', label: 'Class', dir: 'asc' }
  ];

  const uid = $props.id();
  let q = $state('');
  let active = $state(0);
  let list = $state<HTMLElement>();

  const ql = $derived(q.trim().toLowerCase());
  const subs = $derived(builds ? childrenOf(folders, folder).filter((c) => !ql || leaf(c).toLowerCase().includes(ql)) : []);
  const found = $derived(visibleRows(builds ?? [], folder, q, 'All', sort));
  const n = $derived(subs.length + found.length);
  const parts = $derived(folder ? folder.split('/') : []);
  const crumbs = $derived([{ name: 'All builds', path: '' }, ...parts.map((p, i) => ({ name: p, path: parts.slice(0, i + 1).join('/') }))]);
  const at = $derived(Math.min(active, n - 1));
  const curBuild = $derived(at >= subs.length ? found[at - subs.length] : undefined);
  const rowId = (i: number) => `${uid}-row-${i}`;
  const ROW =
    'flex w-full cursor-default border-x-0 border-t-0 bg-transparent text-left font-sans text-13 text-text items-center gap-3 border-b border-surface-2 px-3 py-1.5 hover:bg-row-hover aria-selected:bg-row-selected aria-selected:shadow-sel-bar';

  $effect(() => {
    if (active >= n) active = Math.max(0, n - 1);
  });
  $effect(() => {
    if (open) document.getElementById(rowId(active))?.scrollIntoView?.({ block: 'nearest' });
  });

  const go = (path: string) => {
    folder = path;
    q = '';
    active = 0;
    list?.focus();
  };
  const pick = (i: number) => {
    if (i < 0) return;
    if (i < subs.length) return go(subs[i]);
    const b = found[i - subs.length];
    if (!b) return;
    open = false;
    onpick(b);
  };
  const key = (e: KeyboardEvent) => {
    if (e.key === 'ArrowDown') active = Math.min(active + 1, n - 1);
    else if (e.key === 'ArrowUp') active = Math.max(active - 1, 0);
    else if (e.key === 'Home') active = 0;
    else if (e.key === 'End') active = Math.max(n - 1, 0);
    else if (e.key === 'Enter') pick(at);
    else if (e.key === 'Backspace' && folder) go(parts.slice(0, -1).join('/'));
    else return;
    e.preventDefault();
  };
</script>

<Dialog bind:open {title} class="flex h-[min(560px,85vh)] w-[min(640px,calc(100vw-32px))] flex-col gap-2.5 p-4">
  <nav aria-label="Folder" class="flex flex-wrap items-center gap-1 text-13">
    {#each crumbs as c, i (c.path)}
      {#if i}<span aria-hidden="true" class="text-text-3">/</span>{/if}
      {#if c.path === folder}<span aria-current="location" class="font-semibold">{c.name}</span>{:else}<button
          class="border-0 bg-transparent p-0 font-sans text-13 text-accent hover:underline"
          onclick={() => go(c.path)}>{c.name}</button
        >{/if}
    {/each}
  </nav>
  <div class="flex items-center gap-2">
    <SearchField
      size="md"
      class="min-w-0 grow text-13"
      placeholder="Search {folder ? leaf(folder) : 'all builds'}"
      label="Search builds"
      bind:value={q}
      oninput={() => (active = 0)}
      onkeydown={(e: KeyboardEvent) => {
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Enter') {
          key(e);
          list?.focus();
        }
      }} />
    <Field label="Sort by" inline class="text-12">
      <Select
        size="xs"
        aria-label="Sort by"
        value={sort.key}
        onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
          const s = SORTS.find((x) => x.key === e.currentTarget.value);
          if (s) sort = { key: s.key, dir: s.dir };
        }}>
        {#each SORTS as s (s.key)}<option value={s.key}>{s.label}</option>{/each}
      </Select></Field>
  </div>
  <div
    bind:this={list}
    role="listbox"
    tabindex="0"
    aria-label="Builds in {folder ? leaf(folder) : 'all folders'}"
    aria-activedescendant={n ? rowId(at) : undefined}
    aria-busy={!builds}
    class="scroll panel flex min-h-0 grow flex-col text-13 focus-visible:outline-2 focus-visible:outline-accent"
    onkeydown={key}>
    {#each subs as path, i (path)}
      <button id={rowId(i)} tabindex="-1" role="option" aria-selected={i === at} data-folder={path} class={ROW} onclick={() => go(path)}>
        <Icon name="folder" class="text-accent" />
        <span class="font-semibold text-accent">{leaf(path)}</span>
        <span class="note ml-auto">{countIn(builds ?? [], path)} builds</span>
      </button>
    {/each}
    {#each found as b, j (buildId(b))}
      {@const i = subs.length + j}
      <button
        id={rowId(i)}
        tabindex="-1"
        role="option"
        aria-selected={i === at}
        data-build={buildId(b)}
        class={ROW}
        onclick={() => (active = i)}
        ondblclick={() => pick(i)}>
        <span class="min-w-0 truncate font-semibold"
          >{#if ql && subpath(b, folder)}<span class="font-normal text-text-3">{subpath(b, folder)}</span>{/if}{b.name}</span>
        <span class="note ml-auto shrink-0">{b.ascendancy || b.className} · level {b.level}{b.skill ? ` · ${b.skill}` : ''}</span>
      </button>
    {/each}
    {#if !n}<p class="note p-3">{!builds ? 'Loading…' : ql ? 'No build matches.' : 'No saved builds here.'}</p>{/if}
  </div>
  <div class="flex justify-end gap-2">
    <Button size="sm" onclick={() => (open = false)}>Cancel</Button>
    <Button size="sm" variant="primary" disabled={!curBuild} onclick={() => pick(at)}>{action}</Button>
  </div>
</Dialog>
