<script lang="ts">
  import { untrack } from 'svelte';
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import { Reorder, reorderState } from './reorder';
  import type { ItemSetOp } from './items';
  import type { ItemSet } from './fixtures';
  import ItemSetTip from './ItemSetTip.svelte';

  type SharedOp = { op: 'share'; id: string } | { op: 'import' | 'delete'; index: number } | { op: 'rename'; index: number; title: string };
  type Props = {
    open?: boolean;
    sets: ItemSet[];
    setId: string;
    onpick: (id: string) => void;
    onsets: (op: ItemSetOp) => void;
    shared?: { title: string; tip: string[] }[];
    onshared?: (op: SharedOp) => void;
  };
  let { open = $bindable(false), sets, setId, onpick, onsets, shared = [], onshared }: Props = $props();
  let sharedRow = $state<number | null>(null);
  let sharedNaming = $state<string | null>(null);
  let sharedConfirm = $state(false);
  const sharedSel = $derived(sharedRow === null ? undefined : shared[sharedRow]);

  let selRow = $state(0);
  let naming = $state<{ op: 'new' | 'copy' | 'rename'; title: string } | null>(null);
  let confirmDelete = $state(false);
  const selected = $derived(sets[selRow]);
  $effect(() => {
    if (!open) return;
    untrack(() => {
      selRow = Math.max(
        0,
        sets.findIndex((s) => s.id === setId)
      );
      naming = null;
      confirmDelete = false;
    });
  });

  const saveName = () => {
    if (!naming || !/\S/.test(naming.title)) return;
    const { op, title } = naming;
    if (op === 'new') {
      onsets({ op, title });
      selRow = sets.length;
    } else if (selected) {
      onsets({ op, id: selected.id, title });
      if (op === 'copy') selRow = sets.length;
    }
    naming = null;
  };

  // ItemSetListControl.lua:80-83
  const uid = $props.id();
  let tipRow = $state<number | null>(null);
  const rowEls: HTMLElement[] = $state([]);
  const sharedEls: HTMLElement[] = $state([]);
  let sharedTip = $state<number | null>(null);
  const tipSet = $derived(tipRow === null ? undefined : sets[tipRow]);

  let box = $state<HTMLElement>();
  const dragState = $state(reorderState());
  const drag = new Reorder(dragState, {
    box: () => box,
    name: (i) => sets[i]?.title ?? '',
    onmove: (from, to) => {
      onsets({ op: 'move', from, to });
      selRow = to;
    },
    onactivate: (i) => {
      if (sets[i] && sets[i].id !== setId) onpick(sets[i].id);
    }
  });
</script>

<Dialog bind:open title="Manage item sets" class="w-105 p-4.5">
  <div class="flex flex-col gap-2">
    <div role="listbox" aria-label="Item sets" class="flex max-h-75 flex-col gap-1 overflow-y-auto" bind:this={box}>
      {#each sets as s, i (i)}
        <div {...drag.row(i)} class="relative">
          <button
            bind:this={rowEls[i]}
            data-reorder-press
            tabindex="-1"
            role="option"
            aria-selected={i === selRow}
            aria-describedby={tipRow === i && s.tip ? `${uid}-tip` : undefined}
            onmouseenter={() => (tipRow = i)}
            onmouseleave={() => (tipRow = null)}
            onfocus={() => (tipRow = i)}
            onblur={() => (tipRow = null)}
            class="choice-card justify-between gap-2 px-2.5 py-1.5 text-left text-13 aria-selected:border-accent aria-selected:bg-surface-3"
            onclick={() => {
              selRow = i;
              naming = null;
              confirmDelete = false;
            }}
            ondblclick={() => {
              if (s.id !== setId) onpick(s.id);
            }}
            ><span class="truncate">{s.title}</span>{#if s.id === setId}<span class="font-mono text-11 text-accent">current</span>{/if}</button>
        </div>
      {/each}
    </div>
    <div aria-live="assertive" class="sr-only">{dragState.message}</div>
    <span id={drag.hint} class="sr-only">{drag.hintText}; Enter switches to the set</span>
    {#if naming}
      <form
        class="flex items-center gap-1.5"
        onsubmit={(e) => {
          e.preventDefault();
          saveName();
        }}>
        <input class="input h-7.5 grow" aria-label="Name for this item set" bind:value={naming.title} maxlength="100" />
        <Button size="sm" variant="primary" type="submit" disabled={!/\S/.test(naming.title)}>Save</Button>
        <Button size="sm" type="button" onclick={() => (naming = null)}>Cancel</Button>
      </form>
    {:else if confirmDelete && selected}
      <div class="flex items-center gap-1.5 text-13" role="group" aria-label="Confirm delete">
        <span class="grow">Are you sure you want to delete '{selected.title}'? This will not delete any items used by the set.</span>
        <Button
          size="sm"
          variant="danger"
          onclick={() => {
            onsets({ op: 'delete', id: selected.id });
            confirmDelete = false;
            selRow = Math.max(0, selRow - 1);
          }}>Delete</Button>
        <Button size="sm" onclick={() => (confirmDelete = false)}>Cancel</Button>
      </div>
    {:else}
      <div class="flex gap-1.5">
        <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'new', title: '' })}>New</Button>
        <Button size="sm" class="bg-surface-1" disabled={!selected} onclick={() => (naming = { op: 'copy', title: selected?.title ?? '' })}>Copy</Button>
        <Button size="sm" class="bg-surface-1" disabled={!selected} onclick={() => (naming = { op: 'rename', title: selected?.title ?? '' })}>Rename</Button>
        <Button size="sm" variant="danger" disabled={!selected || sets.length < 2} onclick={() => (confirmDelete = true)}>Delete</Button>
      </div>
    {/if}
    <span class="note">Double-click an item set to switch to it; drag to reorder.</span>
  </div>
  {#if onshared}
    <section aria-label="Shared item sets" class="mt-3 flex flex-col gap-2 border-t border-line pt-3">
      <h3 class="caps">Shared item sets</h3>
      <div role="listbox" aria-label="Shared sets" class="flex max-h-50 flex-col gap-1 overflow-y-auto">
        {#each shared as s, i (i)}
          <button
            bind:this={sharedEls[i]}
            role="option"
            aria-selected={i === sharedRow}
            aria-describedby={sharedTip === i && s.tip.length ? `${uid}-shared-tip` : undefined}
            class="choice-card px-2.5 py-1.5 text-left text-13 aria-selected:border-accent aria-selected:bg-surface-3"
            onclick={() => {
              sharedRow = i;
              sharedNaming = null;
              sharedConfirm = false;
            }}
            onmouseenter={() => (sharedTip = i)}
            onmouseleave={() => (sharedTip = null)}
            onfocus={() => (sharedTip = i)}
            onblur={() => (sharedTip = null)}>{s.title}</button>
        {:else}
          <span class="note">Item sets shared here are kept for all your builds: share one of this build's sets to add it.</span>
        {/each}
      </div>
      {#if sharedNaming !== null && sharedRow !== null}
        {@const row = sharedRow}
        <form
          class="flex items-center gap-1.5"
          onsubmit={(e) => {
            e.preventDefault();
            if (sharedNaming && /\S/.test(sharedNaming)) onshared({ op: 'rename', index: row, title: sharedNaming });
            sharedNaming = null;
          }}>
          <input class="input h-7.5 grow" aria-label="Name for this shared set" bind:value={sharedNaming} maxlength="100" />
          <Button size="sm" variant="primary" type="submit" disabled={!/\S/.test(sharedNaming)}>Save</Button>
          <Button size="sm" type="button" onclick={() => (sharedNaming = null)}>Cancel</Button>
        </form>
      {:else if sharedConfirm && sharedSel && sharedRow !== null}
        {@const row = sharedRow}
        <div class="flex items-center gap-1.5 text-13" role="group" aria-label="Confirm shared delete">
          <span class="grow">Are you sure you want to delete '{sharedSel.title}' from the shared item set list?</span>
          <Button
            size="sm"
            variant="danger"
            onclick={() => {
              onshared({ op: 'delete', index: row });
              sharedConfirm = false;
              sharedRow = null;
            }}>Delete</Button>
          <Button size="sm" onclick={() => (sharedConfirm = false)}>Cancel</Button>
        </div>
      {:else}
        <div class="flex flex-wrap gap-1.5">
          <Button size="sm" class="bg-surface-1" disabled={!selected} onclick={() => selected && onshared({ op: 'share', id: selected.id })}>Share set</Button>
          <Button
            size="sm"
            class="bg-surface-1"
            disabled={sharedRow === null}
            onclick={() => sharedRow !== null && onshared({ op: 'import', index: sharedRow })}>Import</Button>
          <Button size="sm" class="bg-surface-1" disabled={!sharedSel} onclick={() => (sharedNaming = sharedSel?.title ?? '')}>Retitle</Button>
          <Button size="sm" variant="danger" disabled={!sharedSel} onclick={() => (sharedConfirm = true)}>Remove</Button>
        </div>
      {/if}
    </section>
  {/if}
  {#if open && sharedTip !== null && shared[sharedTip]?.tip.length}
    <ItemSetTip anchor={sharedEls[sharedTip]} id="{uid}-shared-tip" title={shared[sharedTip].title} lines={shared[sharedTip].tip} />
  {/if}
  {#if open && tipRow !== null && tipSet?.tip}
    <ItemSetTip anchor={rowEls[tipRow]} id="{uid}-tip" title={tipSet.title} lines={tipSet.tip} />
  {/if}
  {#snippet actions()}
    <Button size="md" variant="primary" class="px-4" onclick={() => (open = false)}>Done</Button>
  {/snippet}
</Dialog>
