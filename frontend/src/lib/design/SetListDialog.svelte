<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import { untrack } from 'svelte';
  import type { SetListOp } from './set_list';
  import { Reorder, reorderState } from './reorder';

  // ConfigSetListControl.lua:13-110
  // ConfigSetListControl.lua:71-110
  type Props = { open?: boolean; title: string; noun: string; sets: string[]; current: number; onactivate: (i: number) => void; onop: (op: SetListOp) => void };
  let { open = $bindable(false), title, noun, sets, current, onactivate, onop }: Props = $props();

  let sel = $state(0);
  let naming = $state<{ op: 'new' | 'copy' | 'rename'; title: string } | null>(null);
  let confirmDelete = $state(false);
  $effect(() => {
    if (open)
      untrack(() => {
        sel = current;
        naming = null;
        confirmDelete = false;
      });
  });
  const save = () => {
    if (!naming || !/\S/.test(naming.title)) return;
    const { op, title: t } = naming;
    if (op === 'new') onop({ op, title: t });
    else onop({ op, index: sel, title: t });
    if (op !== 'rename') sel = sets.length;
    naming = null;
  };
  let box = $state<HTMLElement>();
  const dragState = $state(reorderState());
  const drag = new Reorder(dragState, {
    box: () => box,
    name: (i) => sets[i] ?? '',
    onmove: (from, to) => {
      onop({ op: 'move', from, to });
      sel = to;
    },
    onactivate: (i) => {
      if (i !== current) onactivate(i);
    }
  });
  const move = (by: number) => {
    const to = sel + by;
    if (to < 0 || to >= sets.length) return;
    onop({ op: 'move', from: sel, to });
    sel = to;
  };
</script>

<Dialog bind:open {title} class="w-105 p-4.5">
  <div class="flex flex-col gap-2">
    <div role="listbox" aria-label={`${noun}s`} class="flex max-h-75 flex-col gap-1 overflow-y-auto" bind:this={box}>
      {#each sets as s, i (i)}
        <div {...drag.row(i)} class="relative">
          <button
            data-reorder-press
            tabindex="-1"
            role="option"
            aria-selected={i === sel}
            class="choice-card justify-between gap-2 px-2.5 py-1.5 text-left text-13 aria-selected:border-accent aria-selected:bg-surface-3"
            onclick={() => {
              sel = i;
              naming = null;
              confirmDelete = false;
            }}
            ondblclick={() => {
              if (i !== current) onactivate(i);
            }}
            ><span class="truncate">{s}</span>{#if i === current}<span class="font-mono text-11 text-text-2">(Current)</span>{/if}</button>
        </div>
      {/each}
    </div>
    <div aria-live="assertive" class="sr-only">{dragState.message}</div>
    <span id={drag.hint} class="sr-only">{drag.hintText}; Enter makes it current</span>
    {#if naming}
      <form
        class="flex items-center gap-1.5"
        onsubmit={(e) => {
          e.preventDefault();
          save();
        }}>
        <input class="input h-7.5 grow" aria-label={`Name for this ${noun}`} bind:value={naming.title} maxlength="100" />
        <Button size="sm" variant="primary" type="submit" disabled={!/\S/.test(naming.title)}>Save</Button>
        <Button size="sm" type="button" onclick={() => (naming = null)}>Cancel</Button>
      </form>
    {:else if confirmDelete}
      <div class="flex items-center gap-1.5 text-13" role="group" aria-label="Confirm delete">
        <span class="grow">Are you sure you want to delete '{sets[sel]}'?</span>
        <Button
          size="sm"
          variant="danger"
          onclick={() => {
            onop({ op: 'delete', index: sel });
            confirmDelete = false;
            sel = Math.max(0, sel - 1);
          }}>Delete</Button>
        <Button size="sm" onclick={() => (confirmDelete = false)}>Cancel</Button>
      </div>
    {:else}
      <div class="flex flex-wrap gap-1.5">
        <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'new', title: '' })}>New</Button>
        <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'copy', title: sets[sel] ?? '' })}>Copy</Button>
        <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'rename', title: sets[sel] ?? '' })}>Rename</Button>
        <Button size="sm" variant="danger" disabled={sets.length < 2} onclick={() => (confirmDelete = true)}>Delete</Button>
        <Button size="sm" class="bg-surface-1" disabled={sel <= 0} onclick={() => move(-1)}>Move up</Button>
        <Button size="sm" class="bg-surface-1" disabled={sel >= sets.length - 1} onclick={() => move(1)}>Move down</Button>
      </div>
    {/if}
  </div>
  {#snippet actions()}<Button onclick={() => (open = false)}>Done</Button>{/snippet}
</Dialog>
