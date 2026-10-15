<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';

  const EMPTY = 'No matching modifiers found';
  // ConfigModBrowser.lua:271-278
  const HELP =
    'This menu currently contains supported mod lines from tree nodes and item modifiers only.\n\nThis menu is not a representation of what PoB can parse, and this is only a limited list of mods. Additionally, some mods might be local to items and might not function in custom modifiers.\n\nA mod being supported does not necessarily mean that it will be included in calculations, and only means that the mod parser accepts it.';
  type Props = {
    open?: boolean;
    /* ConfigModBrowser.lua:113-150 */
    list: (search: string) => Promise<string[]>;
    /* ConfigModBrowser.lua:243-250 */
    sources: (line: string) => Promise<string[]>;
    onadd: (line: string) => void;
  };
  let { open = $bindable(false), list, sources, onadd }: Props = $props();

  let search = $state('');
  let lines = $state.raw<string[]>([]);
  let sel = $state(0);
  let tip = $state<{ line: string; from?: string[] }>();
  const known = new Map<string, string[]>();
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
    tip = { line, from: known.get(line) };
    if (known.has(line)) return;
    sources(line)
      .then((from) => {
        known.set(line, from);
        if (tip?.line === line) tip = { line, from };
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
      {#if tip}{tip.from ? tip.from.join(' · ') : '…'}{/if}
    </p>
  </div>
  {#snippet actions()}
    <Button variant="primary" disabled={!canAdd} onclick={add}>Add</Button>
    <Button onclick={() => (open = false)}>Cancel</Button>
    <!-- ConfigModBrowser.lua:267-270 -->
    <span class="cursor-help px-1 text-13 text-text-2" title={HELP}>?</span>
  {/snippet}
</Dialog>
