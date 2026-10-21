<script lang="ts">
  import Listbox from './Listbox.svelte';
  import SearchField from './SearchField.svelte';
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';

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

<Dialog bind:open title="Mod browser" size="xl" anchor="top">
  <div class="flex flex-col gap-2">
    <Listbox
      items={lines}
      key={(_: string, i: number) => i}
      label="Modifiers"
      selected={sel}
      class="h-113.5"
      onselect={(_: string, i: number) => (sel = i)}
      onactivate={(_: string, i: number) => {
        sel = i;
        add();
      }}
      onhover={(i: number) => (i < 0 ? (tip = undefined) : hover(lines[i]))}>
      {#snippet row(l: string)}{l}{/snippet}
    </Listbox>
    <label class="field field-inline text-13"
      >Search:
      <SearchField size="sm" class="grow" label="Search modifiers" autofocus bind:value={search} />
    </label>
    <p class="m-0 min-h-5 text-12" role="status">
      {#if tip}{tip.from ? tip.from.join(' · ') : '…'}{/if}
    </p>
  </div>
  {#snippet actions()}
    <DialogActions confirm="Add" disabled={!canAdd} oncancel={() => (open = false)} onconfirm={add}>
      {#snippet extra()}
        <!-- ConfigModBrowser.lua:267-270 -->
        <span class="cursor-help px-1 text-13 text-text-2" title={HELP}>?</span>
      {/snippet}
    </DialogActions>
  {/snippet}
</Dialog>
