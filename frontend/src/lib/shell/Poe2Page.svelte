<script lang="ts">
  import { poe2Build, type Poe2View } from '$lib/global.js';
  import type { Poe2Calc } from '$lib/go/sync_worker2';
  import { poe2Busy, poe2Error, poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import SkillTree2 from '$lib/components/skill-tree/SkillTree2.svelte';
  import type { Tree2 } from '$lib/components/skill-tree/scene2';

  // the PoE2 bodies of the frame's page routes
  let { route }: { route: string } = $props();

  // the PoE2 worker hands back a Poe2Calc; a view without its fields renders the stats it has
  const view = $derived($poe2Build as (Partial<Poe2Calc> & Poe2View) | undefined);
  const busy = $derived($poe2Busy);
  const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

  let treeData = $state.raw<Tree2 | undefined>();
  $effect(() => {
    if (route !== 'tree' || treeData) return;
    void poe2Worker()
      .then((w) => w.TreeData())
      .then((d) => (treeData = d))
      .catch(fail);
  });
  // PassiveTreeView.lua:425-594, TreeTab.lua:355-364, 851-876 (PoB-PoE2)
  const treeClick = async (id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number) =>
    (
      await poe2Run(
        (w) => w.TreeClick(id, right, hotkey, confirmClassChange, allocMode),
        (r) => r.calc
      )
    )?.result;
  const treeAttribute = (id: number, index: number, allocMode: number) => poe2Run((w) => w.TreeAllocateAttribute(id, index, allocMode));
  const treeHover = async (id: number, allocMode: number) => (await poe2Worker()).TreeHover(id, allocMode);
  const treeUndo = (redo: boolean) => poe2Run((w) => w.TreeUndo(redo));
</script>

{#if view}
  {#if $poe2Error}<p class="m-0 px-4 py-2 text-14" role="alert">{$poe2Error}</p>{/if}
  {#if route === 'tree'}
    <div class="flex min-h-[480px] min-w-0 flex-1 flex-col" aria-label="Tree" data-poe2-page>
      <div class="flex items-center gap-3 px-3 py-1.5 text-14">
        <span>Level {view.level} {view.ascendClassName && view.ascendClassName !== 'None' ? view.ascendClassName : view.className}</span>
        {#if view.tree}
          <span aria-label="Passive points used">Points: {view.tree.used}{view.tree.ascUsed ? ` / Ascendancy: ${view.tree.ascUsed}` : ''}</span>
        {/if}
        {#if busy}<span class="note" role="status">Calculating…</span>{/if}
      </div>
      <div class="flex min-h-0 min-w-0 flex-1">
        {#if treeData && view.tree}
          <SkillTree2
            data={treeData}
            spec={view.tree}
            {busy}
            onclick={treeClick}
            onattribute={treeAttribute}
            onhover={treeHover}
            onundo={() => treeUndo(false)}
            onredo={() => treeUndo(true)} />
        {:else}
          <span class="note" role="status">Loading the passive tree…</span>
        {/if}
      </div>
    </div>
  {/if}
{/if}
