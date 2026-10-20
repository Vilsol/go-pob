<script lang="ts">
  import Badge from './Badge.svelte';
  import type { Snippet } from 'svelte';
  import { untrack } from 'svelte';
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import DialogActions from './DialogActions.svelte';
  import InlineConfirm from './InlineConfirm.svelte';
  import type { SetListOp } from './set_list';
  import { Reorder, reorderState } from './reorder';

  // ConfigSetListControl.lua:13-110, ItemSetListControl.lua, SkillSetListControl.lua, PassiveSpecListControl.lua
  type Props = {
    open?: boolean;
    title: string;
    noun: string;
    sets: string[];
    current: number;
    sel?: number;
    reorderable?: boolean;
    movable?: boolean;
    nameOf?: (i: number) => string;
    deleteMessage?: (i: number) => string;
    note?: string;
    activateHint?: string;
    class?: string;
    row?: Snippet<[number]>;
    rowProps?: (i: number) => Record<string, unknown>;
    aside?: Snippet;
    panel?: Snippet;
    tools?: Snippet;
    children?: Snippet;
    onselect?: (i: number) => void;
    onclose?: () => void;
    onactivate: (i: number) => void;
    onop: (op: SetListOp) => void;
  };
  let {
    open = $bindable(false),
    title,
    noun,
    sets,
    current,
    sel = $bindable(0),
    reorderable = true,
    movable = false,
    nameOf = (i) => sets[i] ?? '',
    deleteMessage = (i) => `Are you sure you want to delete '${sets[i] ?? ''}'?`,
    note,
    activateHint = 'Enter makes it current',
    class: extra = 'w-105 p-4.5',
    row,
    rowProps,
    aside,
    panel,
    tools,
    children,
    onselect,
    onclose,
    onactivate,
    onop
  }: Props = $props();

  const listLabel = $derived(`${noun[0].toUpperCase()}${noun.slice(1)}s`);
  let naming = $state<{ op: 'new' | 'copy' | 'rename'; title: string } | null>(null);
  let confirmDelete = $state(false);
  $effect(() => {
    if (open)
      untrack(() => {
        sel = Math.max(0, current);
        naming = null;
        confirmDelete = false;
      });
  });
  const select = (i: number) => {
    sel = i;
    naming = null;
    confirmDelete = false;
    onselect?.(i);
  };
  const activate = (i: number) => {
    if (i !== current) onactivate(i);
  };
  const save = () => {
    if (!naming || !/\S/.test(naming.title)) return;
    const { op, title: t } = naming;
    if (op === 'new') onop({ op, title: t });
    else onop({ op, index: sel, title: t });
    if (op !== 'rename') sel = sets.length;
    naming = null;
  };
  const remove = () => {
    onop({ op: 'delete', index: sel });
    confirmDelete = false;
    sel = Math.max(0, sel - 1);
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
    onactivate: activate
  });
  const move = (by: number) => {
    const to = sel + by;
    if (to < 0 || to >= sets.length) return;
    onop({ op: 'move', from: sel, to });
    sel = to;
  };
  // PassiveSpecListControl.lua:95-117
  const listKey = (e: KeyboardEvent) => {
    if (e.key !== 'Delete' && e.key !== 'F2') return;
    const rows = [...(box?.querySelectorAll(reorderable ? '[data-reorder-item]' : '[role="option"]') ?? [])];
    const i = rows.indexOf((e.target as HTMLElement).closest(reorderable ? '[data-reorder-item]' : '[role="option"]')!);
    if (i < 0) return;
    e.preventDefault();
    select(i);
    if (e.key === 'F2') naming = { op: 'rename', title: nameOf(i) };
    else if (sets.length > 1) confirmDelete = true;
  };
</script>

{#snippet option(i: number, label: string)}
  <button
    data-reorder-press={reorderable ? '' : undefined}
    tabindex={reorderable ? -1 : undefined}
    role="option"
    aria-selected={i === sel}
    class="choice-card justify-between gap-2 px-2.5 py-1.5 text-left text-13"
    {...rowProps?.(i)}
    onclick={() => select(i)}
    ondblclick={() => activate(i)}
    >{#if row}{@render row(i)}{:else}<span class="truncate">{label}</span>{/if}{#if i === current}<Badge tone="accent">current</Badge>{/if}</button>
{/snippet}

<Dialog bind:open {title} class={extra} {onclose}>
  <div class="flex flex-col gap-2">
    <div role="listbox" aria-label={listLabel} tabindex="-1" class="flex max-h-75 flex-col gap-1 overflow-y-auto" bind:this={box} onkeydown={listKey}>
      {#each sets as s, i (i)}
        {#if reorderable}
          <div {...drag.row(i)} class="relative">{@render option(i, s)}</div>
        {:else}
          {@render option(i, s)}
        {/if}
      {/each}
    </div>
    {#if reorderable}
      <div aria-live="assertive" class="sr-only">{dragState.message}</div>
      <span id={drag.hint} class="sr-only">{drag.hintText}; {activateHint}</span>
    {/if}
    {@render aside?.()}
    {#if panel}
      {@render panel()}
    {:else if naming}
      <form
        class="flex items-center gap-1.5"
        onsubmit={(e) => {
          e.preventDefault();
          save();
        }}>
        <input class="input input-sm grow" aria-label={`Name for this ${noun}`} bind:value={naming.title} maxlength="100" />
        <Button size="sm" variant="primary" type="submit" disabled={!/\S/.test(naming.title)}>Save</Button>
        <Button size="sm" type="button" onclick={() => (naming = null)}>Cancel</Button>
      </form>
    {:else if confirmDelete}
      <InlineConfirm message={deleteMessage(sel)} onconfirm={remove} oncancel={() => (confirmDelete = false)} />
    {:else}
      <div class="flex flex-wrap gap-1.5">
        <Button size="sm" class="bg-surface-1" onclick={() => (naming = { op: 'new', title: '' })}>New</Button>
        <Button size="sm" class="bg-surface-1" disabled={!sets.length} onclick={() => (naming = { op: 'copy', title: nameOf(sel) })}>Copy</Button>
        <Button size="sm" class="bg-surface-1" disabled={!sets.length} onclick={() => (naming = { op: 'rename', title: nameOf(sel) })}>Rename</Button>
        <Button size="sm" variant="danger" disabled={sets.length < 2} onclick={() => (confirmDelete = true)}>Delete</Button>
        {#if movable}
          <Button size="sm" class="bg-surface-1" disabled={sel <= 0} onclick={() => move(-1)}>Move up</Button>
          <Button size="sm" class="bg-surface-1" disabled={sel >= sets.length - 1} onclick={() => move(1)}>Move down</Button>
        {/if}
      </div>
      {@render tools?.()}
    {/if}
    {#if note}<span class="note">{note}</span>{/if}
  </div>
  {@render children?.()}
  {#snippet actions()}
    <DialogActions confirm="Done" cancel={false} onconfirm={() => (open = false)} />
  {/snippet}
</Dialog>
