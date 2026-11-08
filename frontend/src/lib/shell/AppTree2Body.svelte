<script lang="ts">
  import { poe2Build, poe2Edited, type Poe2View } from '$lib/global.js';
  import { get } from 'svelte/store';
  import Textarea from '$lib/design/Textarea.svelte';
  import type { Poe2Calc } from '$lib/go/sync_worker2';
  import type { TreeModel } from '$lib/go/engine';
  import { poe2Busy, poe2Error, poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import TreePage from '$lib/design/TreePage.svelte';
  import type { TreeFixture, TreeNodeFx } from '$lib/design/fixtures';
  import type { CanvasApi, SpecOp } from '$lib/design/tree';
  import SkillTree2 from '$lib/components/skill-tree/SkillTree2.svelte';
  import type { Tree2, Tree2Node } from '$lib/components/skill-tree/scene2';
  import { searchResults } from '$lib/components/skill-tree/tree2_search';
  import Button from '$lib/design/Button.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { logError } from '$lib/utils';

  // the PoE2 worker hands back a Poe2Calc; a view without its fields renders the stats it has
  const view = $derived($poe2Build as (Partial<Poe2Calc> & Poe2View) | undefined);
  const busy = $derived($poe2Busy);
  const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

  let { make }: { make: () => TreeModel } = $props();
  // made while the page initialises, as it follows the build for as long as the page lives
  const model: TreeModel | undefined = make();

  let treeData = $state.raw<Tree2 | undefined>();
  void poe2Worker()
    .then((w) => w.TreeData())
    .then((d) => (treeData = d))
    .catch(fail);

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
  const progress = $derived(view?.tree?.progress);
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

  // the passive tree's nodes as the shared page lists and searches them, with the spec's switched text
  const KIND: Record<string, TreeNodeFx['k']> = { Notable: 'n', Keystone: 'k', Socket: 'j', Mastery: 'm', ClassStart: 'c', AscendClassStart: 'c' };
  const byId = $derived(new Map((treeData?.nodes ?? []).map((n) => [n.id, n])));
  const alloc = $derived(Object.fromEntries((view?.tree?.alloc ?? []).map((id) => [id, true])) as Record<number, boolean>);
  const nodes = $derived(
    (treeData?.nodes ?? [])
      .filter((n) => n.type !== 'OnlyImage')
      .map((n): TreeNodeFx => {
        const text = view?.tree?.text[n.id];
        return {
          id: n.id,
          k: KIND[n.type] ?? 's',
          x: 0,
          y: 0,
          name: text?.name ?? n.name,
          stats: text?.stats ?? n.stats ?? [],
          ...(alloc[n.id] ? { a: 1 } : {})
        };
      })
  );
  // PassiveTreeView.lua:858-882 (PoB-PoE2)
  const search = (q: string, list: TreeNodeFx[]) =>
    searchResults(
      q,
      list.map((n) => ({ id: n.id, type: byId.get(n.id)?.type ?? '', name: n.name, stats: n.stats ?? [] }))
    );
  // TreeTab.lua:188 (PoB-PoE2)
  const SEARCH_HELP2: [string, string][] = [
    ['fire damage', 'every word must match (name, stat lines, node type)'],
    ['"minion life"', 'quotes keep a phrase together'],
    ['(increased.fire|area)', 'several terms; Lua patterns work too']
  ];

  const tree = $derived<TreeFixture>({
    version: model?.version ?? '0_5',
    classes: model?.classes ?? [],
    ascendancies: model?.ascendancies ?? ['None'],
    specs: model?.specs ?? [],
    activeSpec: model?.active ?? 0,
    aspect: 1,
    radius: 0,
    nodes,
    edges: [],
    tips: {},
    sockets: []
  });

  // a search result picked from the list allocates as a click on the tree does; the chooser below stands in for the
  // canvas's popups, which a list pick does not reach
  let chooseAttribute = $state<TreeNodeFx>();
  let confirmClass = $state<TreeNodeFx>();
  const pick = async (n: TreeNodeFx, confirm = false) => {
    const res = await treeClick(n.id, false, 0, confirm, 0);
    if (res === 2) chooseAttribute = n;
    else if (res === 3) confirmClass = n;
  };

  // the canvas reports the node under the pointer, so the page's keys (Ctrl+C, F1) act on it
  let hoveredFx: TreeNodeFx | undefined;
  const fxById = $derived(new Map(nodes.map((n) => [n.id, n])));
  const hover = (api: CanvasApi, id: number | undefined) => {
    if (hoveredFx) api.leave(hoveredFx);
    hoveredFx = id === undefined ? undefined : fxById.get(id);
    if (hoveredFx) api.enter(hoveredFx);
  };
  // PassiveTreeView.lua:556-564, Main.lua:1767-1823 (PoB-PoE2), without the BuildPlanner markup buttons
  let note = $state<{ node: TreeNodeFx; text: string }>();
  const noteKey = (e: MouseEvent) => {
    if (!e.shiftKey || !hoveredFx) return;
    e.preventDefault();
    e.stopPropagation();
    const node = hoveredFx;
    void poe2Worker()
      .then((w) => w.TreeNodeNote(node.id))
      .then((text) => (note = { node, text }))
      .catch(fail);
  };
  const saveNote = async () => {
    const n = note;
    note = undefined;
    if (!n) return;
    await (await poe2Worker()).SetTreeNodeNote(n.node.id, n.text);
    const v = get(poe2Build);
    if (v) poe2Edited(v);
  };
  const ATTRIBUTES = ['Strength', 'Dexterity', 'Intelligence'];
  const nodeLabel = (n: Tree2Node | TreeNodeFx | undefined) => n?.name ?? '';
</script>

{#if view}
  {#if $poe2Error}<p class="m-0 px-4 py-2 text-14" role="alert">{$poe2Error}</p>{/if}
  <div class="flex min-h-[480px] min-w-0 flex-1 flex-col" aria-label="Tree" data-poe2-page>
    {#if model}
      {@const m = model}
      <TreePage
        {tree}
        className={m.className}
        ascendancy={m.ascendancy}
        {alloc}
        level={m.level}
        levelAuto={m.levelAuto}
        onlevel={(l: number) => m.setLevel(l)}
        onlevelauto={(a: boolean) => m.setLevelAuto(a)}
        onclass={(c: string) => m.setClass(c)}
        onascendancy={(a: string) => m.setAscendancy(a)}
        whole
        zoomControls={false}
        edgeMarkers={false}
        hint="Click: allocate / deallocate · Alt+Wheel: weapon set · Drag: pan · Wheel: zoom"
        {search}
        searchHelp={{ rows: SEARCH_HELP2 }}
        versions={[{ value: m.version, label: m.version }]}
        specVersion={m.version}
        spec={m.active}
        specDetails={m.specDetails}
        onspec={(i: number) => m.setSpec(i)}
        onspecs={(op: SpecOp) => m.editSpecs(op)}
        onspecsopen={() => m.readSpecRows()}
        onspechover={(i: number | null) => m.hoverSpec(i)}
        specTip={m.specTip}
        onpick={(n: TreeNodeFx) => void pick(n).catch(logError)}
        onresettree={() => m.reset()}>
        {#snippet points()}
          {#if progress}
            <!-- Build.lua:1087-1097 (PoB-PoE2) -->
            <output class="text-13 whitespace-nowrap text-text-2" aria-label="Points used" title={progressTip}
              >Points <span class={['font-mono', progress.normalPassives > progress.usedMax ? 'text-worse' : 'text-text']}
                >{progress.normalPassives} / {progress.usedMax}</span>
              · Weapon set 1 <span class="font-mono text-worse">{progress.weaponSet1Used} / {progress.weaponSetMax}</span>
              · Weapon set 2 <span class="font-mono text-better">{progress.weaponSet2Used} / {progress.weaponSetMax}</span>
              · Ascendancy
              <span class={['font-mono', progress.ascUsed > progress.ascMax ? 'text-worse' : 'text-text']}>{progress.ascUsed} / {progress.ascMax}</span
              ></output>
          {/if}
          {#if busy}<span class="note" role="status">Calculating…</span>{/if}
        {/snippet}
        {#snippet canvas(api: CanvasApi)}
          <div class="absolute inset-0 flex" role="presentation" onpointerleave={() => hover(api, undefined)} oncontextmenucapture={noteKey}>
            {#if treeData && view.tree}
              <SkillTree2
                data={treeData}
                spec={view.tree}
                {busy}
                onclick={treeClick}
                onattribute={treeAttribute}
                onhover={(id: number, allocMode: number) => {
                  hover(api, id);
                  return treeHover(id, allocMode);
                }}
                onundo={() => treeUndo(false)}
                onredo={() => treeUndo(true)} />
            {:else}
              <EmptyState busy text="Loading the passive tree…" />
            {/if}
          </div>
        {/snippet}
      </TreePage>
    {:else}
      <EmptyState busy text="Loading the passive tree…" />
    {/if}
  </div>
{/if}

<!-- Build.lua:255-269 (PoB-PoE2) -->
<Dialog phone="centred" open={model?.classConfirm !== undefined} onclose={() => model?.answerClass('cancel')} title="Class Change" size="sm">
  {#if model?.classConfirm}
    <div class="flex flex-col gap-3 text-13">
      <p class="m-0">Changing class to {model.classConfirm} will reset your passive tree.</p>
      <p class="m-0">This can be avoided by connecting one of the {model.classConfirm} starting nodes to your tree.</p>
    </div>
  {/if}
  {#snippet actions()}
    <DialogActions confirm="Continue" tone="danger" oncancel={() => model?.answerClass('cancel')} onconfirm={() => model?.answerClass('continue')}>
      {#snippet extra()}<Button size="md" onclick={() => model?.answerClass('connect')}>Connect Path</Button>{/snippet}
    </DialogActions>
  {/snippet}
</Dialog>

<Dialog phone="centred" open={note !== undefined} onclose={() => (note = undefined)} title="Note: {note?.node.name ?? ''}" size="md">
  {#if note}
    <div class="flex flex-col gap-2 text-13">
      <p class="m-0">Note shown on this entry in the exported .build (BuildPlanner) file.<br />Leave blank to remove.</p>
      <Textarea class="h-40 font-mono" aria-label="Node note" bind:value={note.text} />
    </div>
  {/if}
  {#snippet actions()}
    <DialogActions confirm="Save" oncancel={() => (note = undefined)} onconfirm={() => void saveNote().catch(logError)} />
  {/snippet}
</Dialog>

<!-- TreeTab.lua:851-876 (PoB-PoE2) -->
<Dialog phone="centred" open={chooseAttribute !== undefined} onclose={() => (chooseAttribute = undefined)} title="Choose Attribute" size="sm">
  <p class="m-0 text-13">Choose an attribute for {nodeLabel(chooseAttribute)}:</p>
  {#snippet actions()}
    {#each ATTRIBUTES as a, i (a)}
      <Button
        size="md"
        onclick={() => {
          const n = chooseAttribute;
          chooseAttribute = undefined;
          if (n) void treeAttribute(n.id, i + 1, 0);
        }}>{a}</Button>
    {/each}
    <Button size="md" onclick={() => (chooseAttribute = undefined)}>Cancel</Button>
  {/snippet}
</Dialog>

<!-- PassiveTreeView.lua:511-531 (PoB-PoE2) -->
<Dialog phone="centred" open={confirmClass !== undefined} onclose={() => (confirmClass = undefined)} title="Class Change" size="sm">
  <p class="m-0 text-13">Changing class to {byId.get(confirmClass?.id ?? -1)?.ascendancy ?? ''} will reset your passive tree.</p>
  {#snippet actions()}
    <DialogActions
      confirm="Continue"
      tone="danger"
      oncancel={() => (confirmClass = undefined)}
      onconfirm={() => {
        const n = confirmClass;
        confirmClass = undefined;
        if (n) void pick(n, true).catch(logError);
      }} />
  {/snippet}
</Dialog>
