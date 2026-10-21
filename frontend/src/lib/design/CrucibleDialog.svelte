<script lang="ts">
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import ItemTooltip from './ItemTooltip.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import PobText from './PobText.svelte';
  import type { CraftApi, CrucibleChoices } from './crafting';
  import type { ImpactBlock, TipProps } from './items';

  // ItemsTab.lua:3662-3771
  type Props = { open?: boolean; craft: CraftApi };
  let { open = $bindable(false), craft }: Props = $props();

  let choices = $state.raw<CrucibleChoices>();
  let picks = $state<string[]>(['', '', '', '', '']);
  let focus = $state(0);
  let compare = $state.raw<ImpactBlock[]>([]);
  let preview = $state.raw<TipProps>();

  $effect(() => {
    if (!open) return;
    void craft.crucibleChoices?.().then((c) => {
      choices = c;
      picks = [...(c?.selected ?? ['', '', '', '', ''])];
    });
  });

  const focused = $derived(choices?.nodes[focus]?.find((e) => e.id === picks[focus]));
  $effect(() => {
    const e = focused;
    compare = [];
    if (!e || !open) return;
    void craft.modCompare?.([...e.lines]).then((b) => {
      if (e === focused) compare = b;
    });
  });

  $effect(() => {
    const ids = [...picks];
    if (!open || !choices) return;
    void craft.crucibleText?.(ids).then(async (t) => {
      preview = t === undefined ? undefined : await craft.tipOf?.(t);
    });
  });
</script>

<Dialog bind:open title="Add Crucible Modifier to Item" size="lg" anchor="top">
  <div class="flex flex-col gap-2.5 text-13">
    {#each choices?.nodes ?? [] as list, i (i)}
      <FloatingLabel label="Node {i + 1}"
        ><Select
          size="sm"
          aria-label="Node {i + 1}"
          options={[{ value: '', label: 'None' }, ...list.map((e) => ({ value: e.id, label: e.label }))]}
          value={picks[i]}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
            picks[i] = e.currentTarget.value;
            focus = i;
          }} /></FloatingLabel>
    {/each}
    {#if focused}
      <div role="group" aria-label="Crucible comparison" class="flex flex-col gap-1 font-mono text-13">
        {#each focused.lines as l, k (k)}<span class="font-sans"><PobText text={'^7' + l} /></span>{/each}
        <ImpactBlocks blocks={compare} />
      </div>
    {/if}
    {#if preview}
      <div aria-label="New item" role="group" class="self-start"><ItemTooltip {...preview} /></div>
    {/if}
  </div>
  {#snippet actions()}
    <DialogActions
      confirm="Add"
      oncancel={() => (open = false)}
      onconfirm={async () => {
        const t = await craft.crucibleText?.([...picks]);
        if (t !== undefined) craft.settext?.(t);
        open = false;
      }} />
  {/snippet}
</Dialog>
