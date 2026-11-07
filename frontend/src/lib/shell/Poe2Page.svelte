<script lang="ts">
  import { poe2Build, type Poe2View } from '$lib/global.js';
  import type { Poe2Calc } from '$lib/go/sync_worker2';
  import { poe2Busy, poe2Error, poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import SkillTree2 from '$lib/components/skill-tree/SkillTree2.svelte';
  import type { Tree2 } from '$lib/components/skill-tree/scene2';
  import Button from '$lib/design/Button.svelte';
  import FloatingLabel from '$lib/design/FloatingLabel.svelte';
  import NumberInput from '$lib/design/NumberInput.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';

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
  // Build.lua:202-216 (PoB-PoE2)
  const setLevel = (level: number) => poe2Run((w) => w.SetLevel(level));
  const setLevelAuto = (auto: boolean) => poe2Run((w) => w.SetLevelAutoMode(auto));
  const progress = $derived(view?.tree?.progress);
  // TreeTab.lua:145-161 (PoB-PoE2)
  let resetting = $state(false);
  const resetTree = () => {
    resetting = false;
    void poe2Run((w) => w.TreeReset());
  };
  // Build.lua:1046-1051, 1098-1101 (PoB-PoE2)
  const labSuggest = (level: number) =>
    level < 33
      ? ''
      : level < 55
        ? '\nLabyrinth: Normal Lab'
        : level < 68
          ? '\nLabyrinth: Cruel Lab'
          : level < 75
            ? '\nLabyrinth: Merciless Lab'
            : level < 90
              ? '\nLabyrinth: Uber Lab'
              : '';
  const progressTip = $derived(
    progress &&
      `Required Level: ${progress.level}\nEstimated Progress:\nAct: ${progress.act}\nExtra Skillpoints: ${progress.extra}${labSuggest(progress.level)}`
  );
</script>

{#if view}
  {#if $poe2Error}<p class="m-0 px-4 py-2 text-14" role="alert">{$poe2Error}</p>{/if}
  {#if route === 'tree'}
    <div class="flex min-h-[480px] min-w-0 flex-1 flex-col" aria-label="Tree" data-poe2-page>
      <div class="flex flex-wrap items-center gap-3 px-3 py-1.5 text-14">
        <span>{view.ascendClassName && view.ascendClassName !== 'None' ? view.ascendClassName : view.className}</span>
        <FloatingLabel label="Level" class="[--float-bg:var(--color-surface-1)]">
          <NumberInput
            size="md"
            class="w-20 bg-surface-1 font-mono"
            label="Character level"
            min={1}
            max={100}
            value={view.level}
            disabled={busy}
            onchange={(v: number) => {
              if (v !== view.level) void setLevel(v);
            }} />
        </FloatingLabel>
        {#if view.tree}
          <Button
            size="md"
            class="px-3"
            aria-pressed={!!view.tree.levelAuto}
            disabled={busy}
            title="Auto: the level follows the passive points used; Manual: the level you type"
            onclick={() => void setLevelAuto(!view.tree?.levelAuto)}>{view.tree.levelAuto ? 'Auto' : 'Manual'}</Button>
        {/if}
        {#if progress}
          <!-- Build.lua:1087-1097 (PoB-PoE2) -->
          <output class="text-13 whitespace-nowrap text-text-2" aria-label="Points used" title={progressTip}
            >Points <span class={['font-mono', progress.normalPassives > progress.usedMax ? 'text-worse' : 'text-text']}
              >{progress.normalPassives} / {progress.usedMax}</span>
            · Weapon set 1 <span class="font-mono text-worse">{progress.weaponSet1Used} / {progress.weaponSetMax}</span>
            · Weapon set 2 <span class="font-mono text-better">{progress.weaponSet2Used} / {progress.weaponSetMax}</span>
            · Ascendancy
            <span class={['font-mono', progress.ascUsed > progress.ascMax ? 'text-worse' : 'text-text']}>{progress.ascUsed} / {progress.ascMax}</span></output>
        {:else if view.tree}
          <span aria-label="Passive points used">Points: {view.tree.used}{view.tree.ascUsed ? ` / Ascendancy: ${view.tree.ascUsed}` : ''}</span>
        {/if}
        <Button size="md" class="px-3" disabled={busy} onclick={() => (resetting = true)}>Reset Tree</Button>
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

<Dialog bind:open={resetting} title="Reset Tree">
  <p class="m-0 text-13 text-text-2">Warning: resetting your passive tree cannot be undone.</p>
  {#snippet actions()}
    <DialogActions confirm="Reset" tone="danger" oncancel={() => (resetting = false)} onconfirm={resetTree} />
  {/snippet}
</Dialog>
