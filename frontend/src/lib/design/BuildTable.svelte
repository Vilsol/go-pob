<script lang="ts">
  import Checkbox from './Checkbox.svelte';
  import Button from './Button.svelte';
  import Popover from './Popover.svelte';
  import SortHeader from './SortHeader.svelte';
  import SearchField from './SearchField.svelte';
  import Icon from './Icon.svelte';
  import Notice from './Notice.svelte';
  import EmptyState from './EmptyState.svelte';
  import Segmented from './Segmented.svelte';
  import { isTypingTarget } from './keys';
  import { COLS, DEFAULT_COLS, buildId, childrenOf, countIn, leaf, nextSort, subpath, visibleRows, type ColKey, type Drag, type Sort } from './builds';
  import type { SavedBuild } from './fixtures/app';
  import Menu from './Menu.svelte';
  import { dragSource, dropZone, type DropZone } from './reorder';
  import { tone as toneClass } from './tone';

  type Recalc = { done: number; total: number; name: string };
  type Props = {
    builds: SavedBuild[];
    phoneLayout?: 'cards' | 'table';
    folders: string[];
    folder: string;
    classes: string[];
    gameVersion: string;
    fresh?: Record<string, boolean>;
    recalc?: Recalc | null;
    drag: Drag | null;
    q?: string;
    columns?: ColKey[];
    cols?: ColKey[];
    sort?: Sort;
    colMenuOpen?: boolean;
    selected?: string;
    href?: (b: SavedBuild) => string;
    onfolder: (path: string) => void;
    ondrag: (drag: Drag | null) => void;
    onmove: (id: string, dest: string) => void;
    onopen?: (id: string) => void;
    oncopy?: (id: string) => void;
    onrename?: (id: string) => void;
    ondelete?: (id: string) => void;
    /* BuildList.lua:135-167 */
    onpaste?: (id: string, cut: boolean) => void;
    onrecalc?: () => void;
  };
  let {
    builds,
    phoneLayout = 'cards',
    folders,
    folder,
    classes,
    gameVersion,
    fresh = {},
    recalc = null,
    drag,
    q = $bindable(''),
    columns,
    cols = $bindable(columns ? [...columns] : [...DEFAULT_COLS]),
    sort = $bindable({ key: 'edited', dir: 'desc' }),
    colMenuOpen = $bindable(false),
    selected = $bindable(''),
    href = (b) => `#build=${encodeURIComponent(b.name)}`,
    onfolder,
    ondrag,
    onmove,
    onopen,
    oncopy,
    onrename,
    ondelete,
    onpaste,
    onrecalc
  }: Props = $props();

  // BuildListControl.lua:260-280, BuildList.lua:135-167
  let marked = $state<{ id: string; cut: boolean } | null>(null);
  const rowKey = (e: KeyboardEvent, id: string) => {
    const ctrl = (e.ctrlKey || e.metaKey) && !e.altKey && !e.shiftKey;
    const k = e.key.toLowerCase();
    if (e.key === 'F2' && onrename) onrename(id);
    else if (e.key === 'Delete' && ondelete) ondelete(id);
    else if (ctrl && (k === 'c' || k === 'x') && onpaste) marked = { id, cut: k === 'x' };
    else return false;
    e.preventDefault();
    return true;
  };
  const pasteKey = (e: KeyboardEvent) => {
    if (!marked || !onpaste || !(e.ctrlKey || e.metaKey) || e.altKey || e.shiftKey || e.key.toLowerCase() !== 'v') return;
    if (isTypingTarget(e.target)) return;
    if (document.querySelector('dialog[open]')) return;
    e.preventDefault();
    const m = marked;
    marked = null;
    onpaste(m.id, m.cut);
  };

  let cls = $state('All');
  let menuFor = $state<string | null>(null);
  let pickerRoot = $state<HTMLElement>();

  const avail = $derived(columns ? COLS.filter((c) => columns.includes(c.key)) : COLS);
  const shown = $derived(avail.filter((c) => cols.includes(c.key)));
  const tracks = $derived(`20px ${shown.map((c) => `minmax(0, ${c.fr}fr)`).join(' ')} 64px`);
  const ql = $derived(q.trim().toLowerCase());
  const rows = $derived(visibleRows(builds, folder, q, cls, sort));
  const subs = $derived(childrenOf(folders, folder).filter((c) => !ql || leaf(c).toLowerCase().includes(ql)));
  const parts = $derived(folder ? folder.split('/') : []);
  const crumbs = $derived([{ name: 'All builds', path: '' }, ...parts.map((p, i) => ({ name: p, path: parts.slice(0, i + 1).join('/') }))]);
  const stale = (b: SavedBuild) => b.version !== gameVersion && !fresh[b.name];
  const staleCount = $derived(builds.filter(stale).length);

  const toggleCol = (k: ColKey) => (cols = cols.includes(k) ? cols.filter((x) => x !== k) : COLS.map((c) => c.key).filter((x) => x === k || cols.includes(x)));

  const moveItems = (b: SavedBuild) => [
    { label: 'Open', onselect: () => onopen?.(buildId(b)) },
    { label: 'Copy', onselect: () => oncopy?.(buildId(b)) },
    ...(onrename ? [{ label: 'Rename…', onselect: () => onrename(buildId(b)) }] : []),
    ...(ondelete ? [{ label: 'Delete…', onselect: () => ondelete(buildId(b)) }] : []),
    ...folders.map((f, i) => ({
      label: leaf(f) + (b.folder === f ? ' (current)' : ''),
      caption: i === 0 ? 'MOVE TO…' : undefined,
      disabled: b.folder === f,
      indent: `${8 + (f.split('/').length - 1) * 14}px`,
      onselect: () => onmove(buildId(b), f)
    }))
  ];

  const dropOn = (path: string): DropZone => ({
    accepts: () => !!drag,
    over: () => {
      if (drag && (drag.over !== path || drag.where !== 'list')) ondrag({ ...drag, over: path, where: 'list' });
    },
    leave: () => {
      if (drag?.over === path) ondrag({ ...drag, over: null, where: null });
    },
    drop: () => {
      if (drag) onmove(drag.id, path);
    }
  });
</script>

<svelte:window onkeydown={pasteKey} />

<main class="scroll flex min-w-0 grow flex-col gap-3 px-6 py-4 max-phone:px-3" aria-label="Builds in {folder ? leaf(folder) : 'all folders'}">
  <nav aria-label="Breadcrumb" class="flex items-center gap-1.5 text-13">
    {#each crumbs as c, i (c.path)}
      <button
        class="border-0 bg-transparent p-0 font-sans text-13 text-accent aria-[current=page]:font-semibold aria-[current=page]:text-text"
        aria-current={i === crumbs.length - 1 ? 'page' : undefined}
        onclick={() => onfolder(c.path)}
        use:dropZone={dropOn(c.path)}>{c.name}</button
      >{#if i < crumbs.length - 1}<span class="text-text-3" aria-hidden="true">/</span>{/if}
    {/each}
  </nav>
  <div class="flex items-center gap-2.5 max-phone:flex-wrap">
    <SearchField
      size="lg"
      class="w-75 bg-surface-1 max-phone:w-full"
      bind:value={q}
      placeholder={folder ? `Search in ${leaf(folder)} and its subfolders` : 'Search builds'}
      label="Search builds"
      title="Words match name, folder, class, ascendancy and skill; class:witch matches class or ascendancy only" />
    <Segmented chip label="Filter by class" options={classes.map((c) => ({ id: c, label: c }))} bind:value={cls} class="flex-wrap" itemClass="px-2.5" />
    <div class="grow max-phone:hidden"></div>
    {#if staleCount || recalc}
      <Button
        class="h-9 px-3 whitespace-nowrap"
        disabled={!!recalc}
        onclick={onrecalc}
        title="Builds saved with older game data show muted stats until they are recalculated with {gameVersion} data"
        >↻ Recalculate all · {staleCount} stale</Button>
    {/if}
    <div class="relative" bind:this={pickerRoot}>
      <Button
        class="h-9 px-3 whitespace-nowrap"
        aria-expanded={colMenuOpen}
        aria-controls="build-cols"
        onclick={() => (colMenuOpen = !colMenuOpen)}
        onkeydown={(e: KeyboardEvent) => {
          if (e.key === 'Escape') colMenuOpen = false;
        }}>Columns · {shown.length} / {avail.length} ▾</Button>
      <Popover
        id="build-cols"
        open={colMenuOpen}
        anchor={pickerRoot}
        role="group"
        label="Visible columns"
        placement="bottom-end"
        gap={4}
        onclose={() => (colMenuOpen = false)}
        class="flex w-55 flex-col p-1.5 text-13">
        <span class="px-2 py-1 text-11 tracking-caps text-text-3">SHOW COLUMNS</span>
        {#each avail as c (c.key)}
          <label class={['menu-item', c.key === 'name' ? 'text-text-3' : 'text-text']}
            ><Checkbox class="size-4" checked={cols.includes(c.key)} disabled={c.key === 'name'} onchange={() => toggleCol(c.key)} />{c.label}</label>
        {/each}
        <div class="my-1 border-t border-line"></div>
        <button class="menu-item h-7 text-12 font-medium text-accent" onclick={() => (cols = columns ? [...columns] : [...DEFAULT_COLS])}
          >Reset to default</button>
      </Popover>
    </div>
  </div>
  {#if recalc}
    <Notice row>
      <span>Recalculating {recalc.done} / {recalc.total} in the background · {recalc.name}…</span>
      <span class="relative h-1.5 w-40 overflow-hidden rounded-full bg-surface-3"
        ><span class="absolute inset-y-0 left-0 w-(--w) bg-accent" style="--w: {Math.round((recalc.done / recalc.total) * 100)}%"></span></span>
      <span class="note ml-auto">the list stays usable</span>
    </Notice>
  {/if}
  <div class="max-phone:@container max-phone:relative max-phone:overflow-x-auto">
    <div role="grid" aria-label="Builds" data-phone={phoneLayout} class="phone-grid panel max-phone:w-max max-phone:min-w-full" style="--tracks: {tracks}">
      <div role="row" class="data-row data-head grid-cols-(--tracks) gap-2 px-3.5 py-1.5 tracking-caps">
        <span role="columnheader" data-phone-hide><span class="sr-only">Drag</span></span>
        {#each shown as c (c.key)}
          <SortHeader
            label={c.label.toUpperCase()}
            active={sort.key === c.key}
            dir={sort.dir}
            align={c.right ? 'end' : 'start'}
            title="Sort by {c.label}"
            buttonClass="h-6 font-medium"
            data-col-head={c.key === 'name' ? 'name' : undefined}
            onsort={() => (sort = nextSort(sort, c))} />
        {/each}
        <span role="columnheader" data-phone-hide><span class="sr-only">Actions</span></span>
      </div>
      {#each subs as c (c)}
        {@const hot = !!drag && drag.where === 'list' && drag.over === c}
        {@const n = childrenOf(folders, c).length}
        {@const cut = marked?.cut && marked.id === `folder:${c}`}
        <div
          role="row"
          data-folder={c}
          data-cut={cut ? '' : undefined}
          class={[
            'relative box-border flex items-center gap-3 border-b border-surface-2 py-2 pr-3.5 pl-10.5 text-13 max-phone:in-data-[phone=cards]:pl-3.5 max-phone:in-data-[phone=table]:sticky max-phone:in-data-[phone=table]:left-0 max-phone:in-data-[phone=table]:z-raised max-phone:in-data-[phone=table]:w-[100cqw] max-phone:in-data-[phone=table]:pl-3.5',
            hot ? 'drop-target bg-accent/14' : 'bg-alt',
            cut && 'opacity-50'
          ]}
          use:dropZone={dropOn(c)}>
          <span role="gridcell"
            ><button
              class="flex items-center gap-2 border-0 bg-transparent p-0 text-left font-sans text-13 font-semibold text-accent"
              onclick={() => onfolder(c)}
              onkeydown={(e) => {
                const k = e.key.toLowerCase();
                if (!onpaste || !(e.ctrlKey || e.metaKey) || e.altKey || e.shiftKey || (k !== 'c' && k !== 'x')) return;
                e.preventDefault();
                marked = { id: `folder:${c}`, cut: k === 'x' };
              }}><Icon name="folder" />{leaf(c)}</button
            ></span>
          <span role="gridcell" class="text-text-3">folder · {countIn(builds, c)} builds{n ? ` · ${n} subfolder${n > 1 ? 's' : ''}` : ''}</span>
          {#if hot}
            <span role="gridcell" class="ml-auto text-12 font-semibold text-accent">Drop to move into {leaf(c)}</span>
            <div aria-hidden="true" class="drag-ghost top-5 left-65"><span class="grip">⋮⋮</span><span>{drag?.name}</span></div>
          {/if}
        </div>
      {/each}
      {#each rows as b (buildId(b))}
        {@const id = buildId(b)}
        {@const dragged = drag?.id === id}
        <div
          role="row"
          tabindex="0"
          aria-selected={selected === id}
          data-cut={marked?.cut && marked.id === id ? '' : undefined}
          data-build={b.name}
          use:dragSource={{
            data: () => ({ build: id }),
            label: b.name,
            onstart: () => ondrag({ id, name: b.name, over: null, where: null }),
            onend: () => ondrag(null)
          }}
          class={[
            'group data-row cursor-default grid-cols-(--tracks) gap-2 px-3.5 py-1.25 text-13 hover:bg-row-hover focus-visible:-outline-offset-2',
            dragged ? 'opacity-40 outline-1 -outline-offset-1 outline-accent outline-dashed' : 'aria-selected:bg-row-selected aria-selected:shadow-sel-bar',
            marked?.cut && marked.id === id && 'opacity-50'
          ]}
          onclick={(e: MouseEvent) => {
            if (!(e.target as HTMLElement).closest('button, a, [role="menu"]')) selected = id;
          }}
          ondblclick={(e: MouseEvent) => {
            if (!(e.target as HTMLElement).closest('button, [role="menu"]')) onopen?.(id);
          }}
          onkeydown={(e) => {
            if (e.target !== e.currentTarget) return;
            if (rowKey(e, id)) return;
            if (e.key === 'Enter') {
              e.preventDefault();
              onopen?.(id);
            } else if (e.key === ' ') {
              e.preventDefault();
              selected = id;
            }
          }}>
          <span role="gridcell" data-phone-hide
            ><button
              class="drag-handle group-hover:opacity-100 group-focus-within:opacity-100"
              aria-label="Drag {b.name} to a folder"
              title="Drag onto a folder to move it (keyboard: ⋯ → Move to…)">⋮⋮</button
            ></span>
          {#each shown as c (c.key)}
            {#if c.key === 'name'}
              <span role="gridcell" class="truncate font-semibold" data-col="name"
                >{#if ql && subpath(b, folder)}<span class="font-normal text-text-3" data-subpath>{subpath(b, folder)}</span>{/if}<a
                  class="text-inherit no-underline hover:underline"
                  href={href(b)}
                  title="Open {b.name}"
                  onclick={(e) => {
                    if (!onopen) return;
                    e.preventDefault();
                    onopen(id);
                  }}>{b.name}</a
                ></span>
            {:else}
              {@const old = !!c.stat && stale(b)}
              <span
                role="gridcell"
                data-col={c.key}
                data-label={c.right && c.key !== 'level' ? c.label : c.key === 'level' ? 'Lv' : undefined}
                data-stale={old || undefined}
                class={['toned truncate', toneClass(old ? 'muted' : c.tone(b)), c.right && 'text-right font-mono']}
                title={old ? `Saved with ${b.version} data · Recalculate all refreshes it` : undefined}>{old ? '↻ ' : ''}{c.text(b)}</span>
            {/if}
          {/each}
          <span role="gridcell" data-col="aside" class="relative flex items-center justify-end gap-1">
            <Button size="xs" icon aria-label="Copy {b.name}" title="Copy" onclick={() => oncopy?.(id)}><Icon name="copy" size={14} /></Button>
            <Menu label="Actions for {b.name}" items={moveItems(b)} class="top-8 w-57.5" bind:open={() => menuFor === id, (v) => (menuFor = v ? id : null)}>
              {#snippet trigger(attrs)}
                <Button size="xs" icon aria-label="Actions for {b.name}" title={onrename || ondelete ? 'Open, rename, delete, move' : 'Open, move'} {...attrs}
                  ><Icon name="more" /></Button>
              {/snippet}
              {#snippet row(it: { label: string; indent?: string })}<span class="pl-(--indent)" style={it.indent ? `--indent: ${it.indent}` : undefined}
                  >{it.label}</span
                >{/snippet}
            </Menu>
          </span>
        </div>
      {/each}
      {#if !rows.length && !subs.length}<EmptyState inset="page" text="No builds match. Clear the search or pick another class." />{/if}
    </div>
  </div>
  <div class="note">
    {rows.length} builds in {folder || 'all folders'}{ql && folder ? ' and its subfolders' : folder ? ' (subfolders listed above)' : ''} · click selects, double-click
    / Enter / the name opens · drag ⋮⋮ onto a folder to move, or ⋯ → Move to… · click a column header to sort
  </div>
</main>
