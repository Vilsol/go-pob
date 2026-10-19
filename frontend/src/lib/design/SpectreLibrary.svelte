<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import Select from './Select.svelte';
  import PobText from './PobText.svelte';
  import { spectreMatches, type SpectreRow } from './spectres';
  import { Reorder, dragSource, dropZone, reorderState, type DropZone } from './reorder';

  // Build.lua:1453-1482
  // MinionListControl.lua:79-89
  type Props = { open?: boolean; library: SpectreRow[]; spectres: SpectreRow[]; onsave: (ids: string[]) => void };
  let { open = $bindable(false), library, spectres, onsave }: Props = $props();

  let dest = $state<SpectreRow[]>([]);
  let destSel = $state(-1);
  let srcSel = $state<string>();
  let hover = $state.raw<SpectreRow>();
  let search = $state('');
  let mode = $state('1');
  $effect(() => {
    if (open) {
      dest = [...spectres];
      destSel = -1;
      srcSel = undefined;
      search = '';
    }
  });
  const shown = $derived(library.filter((r) => spectreMatches(r, search, Number(mode))));
  const has = (id: string) => dest.some((d) => d.id === id);
  const add = (id: string | undefined) => {
    const row = library.find((r) => r.id === id);
    if (row && !has(row.id)) dest = [...dest, row];
  };
  const tip = $derived(hover ?? (srcSel ? library.find((r) => r.id === srcSel) : dest[destSel]));

  let destBox = $state<HTMLElement>();
  const dragState = $state(reorderState());
  const reorder = new Reorder(dragState, {
    box: () => destBox,
    name: (i) => dest[i]?.name ?? '',
    onmove: (from, to) => {
      const next = [...dest];
      next.splice(to, 0, ...next.splice(from, 1));
      dest = next;
      destSel = to;
    }
  });
  // MinionListControl.lua:83-89
  const spectreOf = (d: unknown) => (d && typeof d === 'object' && 'spectre' in d ? String(d.spectre) : undefined);
  const destZone: DropZone = {
    accepts: (d) => {
      const id = spectreOf(d);
      return !!id && !has(id);
    },
    drop: (d, hit, e) => {
      const row = library.find((r) => r.id === spectreOf(d));
      if (!row || has(row.id)) return;
      const rows = [...(destBox?.querySelectorAll<HTMLElement>('[data-reorder-item]') ?? [])];
      let at = rows.findIndex((el) => el.contains(hit));
      if (at < 0) at = dest.length;
      else {
        const r = rows[at].getBoundingClientRect();
        if (r.height && e.clientY > r.top + r.height / 2) at++;
      }
      dest = [...dest.slice(0, at), row, ...dest.slice(at)];
      destSel = at;
      srcSel = undefined;
    }
  };
</script>

<Dialog bind:open title="Spectre Library" class="w-160 p-4">
  <div class="grid grid-cols-2 gap-3 text-13">
    <div class="flex flex-col gap-1.5">
      <div class="flex items-center justify-between">
        <span class="subhead">Spectres in Build:</span>
        <Button
          size="xs"
          disabled={destSel < 0}
          onclick={() => {
            dest = dest.filter((_, i) => i !== destSel);
            destSel = -1;
          }}>Remove</Button>
      </div>
      <div
        role="listbox"
        aria-label="Spectres in Build"
        tabindex="-1"
        class="scroll flex h-62.5 flex-col border border-line"
        bind:this={destBox}
        use:dropZone={destZone}>
        {#each dest as r, i (i)}
          <div {...reorder.row(i)}>
            <button
              data-reorder-press
              tabindex="-1"
              role="option"
              aria-selected={i === destSel}
              class="border-0 bg-transparent px-2 py-0.5 text-left font-sans text-13 text-text aria-selected:bg-surface-3"
              onclick={() => {
                destSel = i;
                srcSel = undefined;
              }}
              onmouseenter={() => (hover = r)}
              onmouseleave={() => (hover = undefined)}>{r.name}</button>
          </div>
        {/each}
      </div>
      <div aria-live="assertive" class="sr-only">{dragState.message}</div>
      <span id={reorder.hint} class="sr-only">{reorder.hintText}</span>
    </div>
    <div class="flex flex-col gap-1.5">
      <div class="flex items-center justify-between">
        <span class="subhead">Available Spectres:</span>
        <Button size="xs" disabled={!srcSel || has(srcSel)} onclick={() => add(srcSel)}>Add</Button>
      </div>
      <div class="flex gap-1">
        <input class="input h-7 grow" placeholder="Search" aria-label="Search spectres" bind:value={search} />
        <Select
          size="xs"
          aria-label="Search in"
          options={[
            { value: '1', label: 'Names' },
            { value: '2', label: 'Skills' },
            { value: '3', label: 'Both' }
          ]}
          bind:value={mode} />
      </div>
      <div role="listbox" aria-label="Available Spectres" class="scroll flex h-55 flex-col border border-line">
        {#each shown as r (r.id)}
          <button
            role="option"
            aria-selected={r.id === srcSel}
            class="border-0 bg-transparent px-2 py-0.5 text-left font-sans text-13 text-text aria-selected:bg-surface-3"
            onclick={() => {
              srcSel = r.id;
              destSel = -1;
            }}
            ondblclick={() => add(r.id)}
            use:dragSource={{ data: () => ({ spectre: r.id }), label: r.name }}
            onmouseenter={() => (hover = r)}
            onmouseleave={() => (hover = undefined)}>{r.name}</button>
        {/each}
      </div>
    </div>
  </div>
  <p class="note mt-1.5">Spectres in your Library must be assigned to an active Raise Spectre gem for their buffs and curses to activate</p>
  {#if tip}
    <div role="tooltip" class="tooltip mt-1.5 flex flex-col gap-0.5 rounded-md px-3 py-2 text-12">
      {#each tip.tooltip as line, i (i)}<span><PobText text={line} /></span>{/each}
    </div>
  {/if}
  {#snippet actions()}
    <Button
      onclick={() => {
        onsave(dest.map((d) => d.id));
        open = false;
      }}>Save</Button>
    <Button onclick={() => (open = false)}>Cancel</Button>
  {/snippet}
</Dialog>
