<script lang="ts">
  import Button from './Button.svelte';
  import InlineConfirm from './InlineConfirm.svelte';
  import SetManager from './SetManager.svelte';
  import type { SetListOp } from './set_list';
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
  const selected = $derived(sets[selRow]);
  const idAt = (i: number) => sets[i]?.id ?? '';
  const onop = (op: SetListOp) => {
    if (op.op === 'new' || op.op === 'move') onsets(op);
    else if (op.op === 'delete') onsets({ op: 'delete', id: idAt(op.index) });
    else onsets({ op: op.op, id: idAt(op.index), title: op.title });
  };

  // ItemSetListControl.lua:80-83
  const uid = $props.id();
  let tipRow = $state<{ i: number; el: HTMLElement } | null>(null);
  const sharedEls: HTMLElement[] = $state([]);
  let sharedTip = $state<number | null>(null);
  const tipSet = $derived(tipRow === null ? undefined : sets[tipRow.i]);
  const rowProps = (i: number) => ({
    'aria-describedby': tipRow?.i === i && sets[i]?.tip ? `${uid}-tip` : undefined,
    onmouseenter: (e: MouseEvent) => (tipRow = { i, el: e.currentTarget as HTMLElement }),
    onmouseleave: () => (tipRow = null),
    onfocus: (e: FocusEvent) => (tipRow = { i, el: e.currentTarget as HTMLElement }),
    onblur: () => (tipRow = null)
  });
</script>

<SetManager
  bind:open
  bind:sel={selRow}
  title="Manage item sets"
  noun="item set"
  sets={sets.map((s) => s.title)}
  current={sets.findIndex((s) => s.id === setId)}
  deleteMessage={(i: number) => `Are you sure you want to delete '${sets[i]?.title ?? ''}'? This will not delete any items used by the set.`}
  note="Double-click an item set to switch to it; drag to reorder."
  activateHint="Enter switches to the set"
  {rowProps}
  onactivate={(i: number) => sets[i] && onpick(sets[i].id)}
  {onop}>
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
            class="choice-card px-2.5 py-1.5 text-left text-13"
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
          <input class="input input-sm grow" aria-label="Name for this shared set" bind:value={sharedNaming} maxlength="100" />
          <Button size="sm" variant="primary" type="submit" disabled={!/\S/.test(sharedNaming)}>Save</Button>
          <Button size="sm" type="button" onclick={() => (sharedNaming = null)}>Cancel</Button>
        </form>
      {:else if sharedConfirm && sharedSel && sharedRow !== null}
        {@const row = sharedRow}
        <InlineConfirm
          label="Confirm shared delete"
          message={`Are you sure you want to delete '${sharedSel.title}' from the shared item set list?`}
          onconfirm={() => {
            onshared({ op: 'delete', index: row });
            sharedConfirm = false;
            sharedRow = null;
          }}
          oncancel={() => (sharedConfirm = false)} />
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
    <ItemSetTip anchor={tipRow.el} id="{uid}-tip" title={tipSet.title} lines={tipSet.tip} />
  {/if}
</SetManager>
