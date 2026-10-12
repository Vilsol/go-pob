<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';

  // ConfigTab.lua:1305-1550
  const EMPTY = 'No matching modifiers found';
  type Props = {
    open?: boolean;
    /* ConfigTab.lua:1450-1489 */
    list: (search: string) => Promise<string[]>;
    /* ConfigTab.lua:1496-1505 */
    supported: (line: string) => Promise<boolean>;
    onadd: (line: string) => void;
  };
  let { open = $bindable(false), list, supported, onadd }: Props = $props();

  let search = $state('');
  let lines = $state.raw<string[]>([]);
  let sel = $state(0);
  let tip = $state<{ line: string; ok?: boolean }>();
  const known = new Map<string, boolean>();
  let seq = 0;
  $effect(() => {
    if (!open) return;
    const q = search;
    const mine = ++seq;
    list(q)
      .then((l) => {
        if (mine !== seq) return;
        lines = l;
        sel = 0;
      })
      .catch(() => {});
  });
  $effect(() => {
    if (open) search = '';
  });
  const hover = (line: string) => {
    if (line === EMPTY) return;
    tip = { line, ok: known.get(line) };
    if (known.has(line)) return;
    supported(line)
      .then((ok) => {
        known.set(line, ok);
        if (tip?.line === line) tip = { line, ok };
      })
      .catch(() => {});
  };
  const canAdd = $derived(lines[sel] !== undefined && lines[sel] !== EMPTY);
  const add = () => {
    if (!canAdd) return;
    onadd(lines[sel]);
    open = false;
  };
</script>

<Dialog bind:open title="Mod Browser" class="w-180 p-4">
  <div class="flex flex-col gap-2">
    <div role="listbox" aria-label="Modifiers" class="scroll flex h-113.5 flex-col border border-line">
      {#each lines as l, i (i)}
        <button
          role="option"
          aria-selected={i === sel}
          class="border-0 bg-transparent px-2 py-0.5 text-left font-sans text-13 text-text aria-selected:bg-surface-3"
          onclick={() => (sel = i)}
          ondblclick={() => {
            sel = i;
            add();
          }}
          onmouseenter={() => hover(l)}
          onmouseleave={() => (tip = undefined)}>{l}</button>
      {/each}
    </div>
    <label class="field field-inline text-13"
      >Search:
      <!-- svelte-ignore a11y_autofocus -->
      <input class="input h-7 grow" type="search" aria-label="Search modifiers" autofocus bind:value={search} />
    </label>
    <p class="m-0 min-h-5 text-12" role="status">
      {#if tip}Supported: {#if tip.ok === undefined}…{:else if tip.ok}<span class="text-better">Yes</span>{:else}<span class="text-worse">No</span>{/if}{/if}
    </p>
  </div>
  {#snippet actions()}
    <Button variant="primary" disabled={!canAdd} onclick={add}>Add</Button>
    <Button onclick={() => (open = false)}>Cancel</Button>
  {/snippet}
</Dialog>
