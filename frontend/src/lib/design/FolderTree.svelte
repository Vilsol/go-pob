<script lang="ts">
  import { childrenOf, countIn, leaf, type Drag } from './builds';
  import type { SavedBuild } from './fixtures/app';

  type Props = {
    folders: string[];
    builds: SavedBuild[];
    folder: string;
    open: Record<string, boolean>;
    drag: Drag | null;
    note?: string;
    onfolder: (path: string) => void;
    ontoggle: (path: string) => void;
    ondrag: (drag: Drag | null) => void;
    onmove: (id: string, dest: string) => void;
    onnewfolder?: () => void;
  };
  let { folders, builds, folder, open, drag, note = '', onfolder, ontoggle, ondrag, onmove, onnewfolder }: Props = $props();

  type Row = { path: string; name: string; level: number; kids: boolean; isOpen: boolean; lasts: boolean[] };
  const rows = $derived.by(() => {
    const out: Row[] = [{ path: '', name: 'All builds', level: 0, kids: false, isOpen: true, lasts: [] }];
    const walk = (f: string, level: number, lasts: boolean[]) => {
      const cs = childrenOf(folders, f);
      cs.forEach((c, i) => {
        const kids = childrenOf(folders, c).length > 0;
        const last = i === cs.length - 1;
        out.push({ path: c, name: leaf(c), level, kids, isOpen: !!open[c], lasts: [...lasts, last] });
        if (kids && open[c]) walk(c, level + 1, [...lasts, last]);
      });
    };
    walk('', 1, []);
    return out;
  });

  const LN = 'w-4 shrink-0 self-stretch [--ln:linear-gradient(var(--color-line-tree),var(--color-line-tree))_no-repeat]';
  const guide = (lasts: boolean[], i: number) =>
    i === lasts.length - 1
      ? lasts[i]
        ? `${LN} [background:var(--ln)_7px_0_/_1px_50%,var(--ln)_7px_50%_/_9px_1px]`
        : `${LN} [background:var(--ln)_7px_0_/_1px_100%,var(--ln)_7px_50%_/_9px_1px]`
      : lasts[i]
        ? LN
        : `${LN} [background:var(--ln)_7px_0_/_1px_100%]`;

  const target = (r: Row) => !!drag && drag.where === 'tree' && drag.over === r.path && r.path !== '';
</script>

<nav aria-label="Folders" class="box-border flex w-65 shrink-0 flex-col gap-1 border-r border-line bg-inset p-4">
  <div class="mb-1.5 flex items-center justify-between">
    <h2 class="caps">Folders</h2>
    {#if onnewfolder}<button class="btn btn-xs" title="Make a folder in {folder ? leaf(folder) : 'All builds'}" onclick={onnewfolder}>New folder</button>{/if}
  </div>
  <div role="tree" aria-label="Folder tree" class="flex flex-col">
    {#each rows as r (r.path)}
      {@const total = countIn(builds, r.path)}
      {@const direct = r.path === '' ? total : builds.filter((b) => b.folder === r.path).length}
      {@const cur = r.path === folder}
      {@const hot = target(r)}
      <div
        role="treeitem"
        tabindex="-1"
        aria-level={r.level + 1}
        aria-expanded={r.kids ? r.isOpen : undefined}
        aria-selected={cur}
        data-folder={r.path}
        class={[
          'relative flex h-7.5 items-center gap-1 rounded pr-2 pl-1',
          hot ? 'drop-target shadow-none' : cur ? 'bg-surface-2 shadow-[inset_2px_0_0_var(--color-accent)]' : 'hover:bg-bar'
        ]}
        ondragover={(e) => {
          if (!drag || r.path === '') return;
          e.preventDefault();
          if (drag.over !== r.path || drag.where !== 'tree') ondrag({ ...drag, over: r.path, where: 'tree' });
        }}
        ondragleave={() => {
          if (drag?.over === r.path) ondrag({ ...drag, over: null, where: null });
        }}
        ondrop={(e) => {
          if (!drag || r.path === '') return;
          e.preventDefault();
          onmove(drag.id, r.path);
        }}>
        {#each r.lasts.map((_l, i) => guide(r.lasts, i)) as g, i (i)}<span aria-hidden="true" class={g}></span>{/each}
        <button
          class={['h-5 w-4 shrink-0 border-0 bg-transparent p-0 text-10 text-text-2', !r.kids && 'invisible']}
          tabindex={r.kids ? 0 : -1}
          aria-label="{r.isOpen ? 'Collapse' : 'Expand'} {r.name}"
          onclick={() => ontoggle(r.path)}>{r.isOpen ? '▾' : '▸'}</button>
        <button
          class={[
            'flex min-w-0 grow items-center gap-1.5 border-0 bg-transparent p-0 text-left font-sans text-13',
            cur ? 'font-semibold text-text' : 'text-text-soft'
          ]}
          aria-current={cur ? 'page' : undefined}
          title="{direct} {direct === 1 ? 'build' : 'builds'} here, {total} including subfolders"
          onclick={() => onfolder(r.path)}>
          <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true" class={['shrink-0', cur || hot ? 'text-accent' : 'text-text-3']}>
            {#if r.kids && r.isOpen}
              <path d="M1.5 12.5v-9h4.5l1.5 1.5h5.5v2 M1.5 12.5l2-5.5h11l-2 5.5z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"
              ></path>
            {:else}
              <path d="M1.5 3.5h4.5l1.5 1.5h7v7.5h-13z" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"></path>
            {/if}
          </svg>
          <span class="grow truncate">{r.name}</span>
          <span
            class={[
              'box-border flex h-4.5 min-w-4.5 shrink-0 items-center justify-center rounded-[9px] px-1.5',
              hot
                ? 'bg-accent font-sans text-10 font-semibold text-accent-ink'
                : ['font-mono text-11', cur ? 'bg-line' : 'bg-bar', total ? 'text-text-2' : 'text-text-3']
            ]}>{hot ? 'drop here' : total}</span>
        </button>
        {#if hot}<div aria-hidden="true" class="drag-ghost top-5.5 left-17.5"><span class="grip">⋮⋮</span><span>{drag?.name}</span></div>{/if}
      </div>
    {/each}
  </div>
  <div aria-live="polite" class="min-h-4.5 text-12 text-better">{note}</div>
  <div class="note mt-auto">Builds are stored in this browser. Export a build to move it to another device.</div>
</nav>
