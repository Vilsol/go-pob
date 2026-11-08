<script lang="ts">
  import { nodePowerTheme, poe2Build, poe2Edited, type Poe2View } from '$lib/global.js';
  import { get } from 'svelte/store';
  import Textarea from '$lib/design/Textarea.svelte';
  import type { Poe2Calc } from '$lib/go/sync_worker2';
  import type { TreeModel } from '$lib/go/engine';
  import { poe2Busy, poe2Error, poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import TreePage from '$lib/design/TreePage.svelte';
  import type { TreeFixture, TreeNodeFx } from '$lib/design/fixtures';
  import type { CanvasApi, SpecOp } from '$lib/design/tree';
  import SkillTree2 from '$lib/components/skill-tree/SkillTree2.svelte';
  import { nodeShown2, type Tree2, type Tree2Node } from '$lib/components/skill-tree/scene2';
  import { searchResults } from '$lib/components/skill-tree/tree2_search';
  import Button from '$lib/design/Button.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { logError } from '$lib/utils';
  import PobText from '$lib/design/PobText.svelte';
  import type { Poe2TreeCalc } from '$lib/go/engine_poe2_tree_calc.svelte';
  import PowerReport from '$lib/design/PowerReport.svelte';
  import { showPowerRows, type PowerFilter, type PowerSort, type ShownPowerRow } from '$lib/design/tree';
  import { displayStats2 } from '$lib/display/stats2';
  import { formatNumSep, luaFormat } from '$lib/display/format_stat';

  // the PoE2 worker hands back a Poe2Calc; a view without its fields renders the stats it has
  const view = $derived($poe2Build as (Partial<Poe2Calc> & Poe2View) | undefined);
  const busy = $derived($poe2Busy);
  const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

  let { make }: { make: () => TreeModel } = $props();
  // made while the page initialises, as it follows the build for as long as the page lives
  const model: TreeModel | undefined = make();
  // stat differences, pathing lines, Shift tracing, socket jewels and node power
  const calc = (model as { calc?: Poe2TreeCalc } | undefined)?.calc;

  let treeData = $state.raw<Tree2 | undefined>();
  let tree2 = $state<{
    zoom: (how: 'in' | 'out' | 'fit') => void;
    project: (id: number) => { x: number; y: number } | undefined;
    focus: (id: number) => void;
  }>();
  // TreeTab.lua:121-140, 444, 581-586 (PoB-PoE2): the compared tree, 0-based, -1 for none, and its allocated nodes
  let compareSpec = $state(-1);
  let compared = $state.raw<Set<number>>();
  $effect(() => {
    const index = compareSpec;
    void view;
    if (index < 0) {
      compared = undefined;
      return;
    }
    void poe2Worker()
      .then((w) => w.TreeSpecAlloc(index + 1))
      .then((ids) => {
        if (compareSpec === index) compared = new Set(ids);
      })
      .catch(fail);
  });
  void poe2Worker()
    .then((w) => w.TreeData())
    .then((d) => (treeData = d))
    .catch(fail);

  const byId = $derived(new Map((treeData?.nodes ?? []).map((n) => [n.id, n])));
  const allocSet = $derived(new Set(view?.tree?.alloc ?? []));
  // PassiveTreeView.lua:565, 1704 (PoB-PoE2)
  const isAllocSocket = (id: number) => {
    const n = byId.get(id);
    return allocSet.has(id) && (n?.type === 'Socket' || !!n?.containJewelSocket);
  };

  // PassiveTreeView.lua:425-594, TreeTab.lua:355-364, 851-876 (PoB-PoE2)
  const treeClick = async (id: number, right: boolean, hotkey: number, confirmClassChange: boolean, allocMode: number, trace?: number[]) => {
    // PassiveTreeView.lua:565-572 (PoB-PoE2): right-clicking an allocated socket opens the Items page on its slot
    if (right && calc && isAllocSocket(id)) {
      calc.openSocket(id);
      return undefined;
    }
    return (
      await poe2Run(
        (w) => w.TreeClick(id, right, hotkey, confirmClassChange, allocMode, trace),
        (r) => r.calc
      )
    )?.result;
  };
  const treeTrace = async (trace: number[], id: number) => (await poe2Worker()).TreeTraceNode(trace, id);
  const treeConnect = async (id: number, hotkey: number, allocMode: number) =>
    (
      await poe2Run(
        (w) => w.TreeClickConnect(id, hotkey, allocMode),
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
  // PassiveTreeView.lua:858-882, 1533-1535 (PoB-PoE2): a node the tree hides is never a match
  const search = (q: string, list: TreeNodeFx[]) =>
    searchResults(
      q,
      list
        .filter((n) => nodeShown2({ alloc: allocSet, hovered: undefined }, byId.get(n.id)))
        .map((n) => ({ id: n.id, type: byId.get(n.id)?.type ?? '', name: n.name, stats: n.stats ?? [] }))
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
  // PassiveTreeView.lua:1967-2053 (PoB-PoE2): the tooltip's stat differences and pathing lines; trace while tracing
  const hover = (api: CanvasApi, id: number | undefined, trace?: number[]) => {
    if (hoveredFx) api.leave(hoveredFx);
    hoveredFx = id === undefined || !nodeShown2({ alloc: allocSet, hovered: id }, byId.get(id)) ? undefined : fxById.get(id);
    if (hoveredFx) api.enter(hoveredFx);
    calc?.hoverNode(hoveredFx?.id ?? null, trace);
  };
  const diff = $derived(calc?.diff && !isAllocSocket(calc.diff.node) ? calc.diff : undefined);
  const tipInfo = $derived(calc?.tip);

  // TreeTab.lua:197-283, 932-1048, PowerReportListControl.lua (PoB-PoE2): the heat map's controls and the power report
  let powerOpen = $state(false);
  let powerFilter = $state<PowerFilter>('unallocated');
  let powerSort = $state<PowerSort>('power');
  let powerHover = $state<number>();
  let depthChoice = $state(0);
  let powerCustom = $state('0');
  $effect(() => {
    const depth = depthChoice === -1 ? Number.parseInt(powerCustom, 10) || 0 : depthChoice;
    calc?.setPowerDepth(depth);
  });
  $effect(() => {
    if (powerOpen) void calc?.readPowerStats().catch(fail);
  });
  const setPower = (on: boolean) => {
    if (on)
      void calc
        ?.readPowerStats()
        .then(() => calc.setPowerOn(true))
        .catch(fail);
    else calc?.setPowerOn(false);
    // TreeTab.lua:202-204
    if (!on) powerOpen = false;
  };
  const powerLabel = $derived(calc?.powerStats.find((st) => st.Stat === calc.powerStat)?.Label ?? '');
  const shownPower = $derived(
    calc?.powerRows
      ? showPowerRows(
          calc.powerRows,
          displayStats2.find((d) => d.stat === calc.powerStat),
          powerFilter,
          true,
          (f, v) => formatNumSep(luaFormat(f, v)),
          powerSort
        )
      : undefined
  );
  const power = $derived(
    calc?.powerOn
      ? { nodes: calc.power?.nodes ?? new Map(), max: calc.power?.max ?? [0, 0, 0, 0, 0], single: calc.powerStat !== '', theme: $nodePowerTheme }
      : undefined
  );
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
  /* PassiveTreeView.lua:478-488, 513 (PoB-PoE2): the base class whose ascendancies hold the picked node's */
  const confirmClassName = $derived.by(() => {
    const asc = byId.get(confirmClass?.id ?? -1)?.ascendancy;
    return treeData?.art?.classes?.find((c) => c.ascendancies?.some((a) => (a.id ?? a.name) === asc))?.name ?? asc ?? '';
  });
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
        onzoom={(how: 'in' | 'out' | 'fit') => tree2?.zoom(how)}
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
        onresettree={() => m.reset()}
        {compareSpec}
        oncompare={(i: number) => (compareSpec = i)}
        project={(id: number) => tree2?.project(id)}
        {diff}
        nodeAction={() => ''}
        nodePower={calc?.powerOn ?? false}
        onnodepower={(on: boolean) => {
          setPower(on);
          if (on) powerOpen = true;
        }}
        nodePowerBar={false}
        onpowerreport={() => (powerOpen = !powerOpen)}
        detailsOpen={powerOpen}>
        {#snippet details()}
          {#if powerOpen && calc}
            {@const c = calc}
            <PowerReport
              stats={c.powerStats}
              bind:stat={() => c.powerStat, (v: string) => c.setPowerStat(v)}
              bind:depth={depthChoice}
              bind:custom={powerCustom}
              bind:filter={powerFilter}
              bind:sort={powerSort}
              statLabel={powerLabel}
              heat={c.powerOn}
              started={c.powerOn}
              rows={shownPower}
              progress={c.powerProgress && c.powerProgress.total ? c.powerProgress.done / c.powerProgress.total : 0}
              onheat={setPower}
              onstart={() => setPower(true)}
              onstop={() => setPower(false)}
              onclose={() => (powerOpen = false)}
              onhover={(id: number | undefined) => (powerHover = id)}
              onpick={(r: ShownPowerRow) => tree2?.focus(r.ID)} />
          {/if}
        {/snippet}
        {#snippet nodeTipExtra(n: TreeNodeFx)}
          {@const t = tipInfo?.node === n.id ? tipInfo : undefined}
          {#if byId.get(n.id)?.ascendancy}<p class="m-0 text-text-2">{byId.get(n.id)?.ascendancy}</p>{/if}
          {#if t?.jewel}
            <!-- PassiveTreeView.lua:1704-1715 (PoB-PoE2) -->
            <div class="flex flex-col gap-0.5 border-t border-line pt-2" aria-label="Socketed jewel">
              {#each t.jewel.item.name as line, i (i)}<span class="font-semibold"><PobText text={line} /></span>{/each}
              {#each t.jewel.item.lines as group, g (g)}
                <div class={['flex flex-col gap-0.5', g > 0 && 'border-t border-line pt-1']}>
                  {#each group as line, i (i)}<span class="whitespace-pre-line"><PobText text={line} /></span>{/each}
                </div>
              {/each}
            </div>
            {#if t.distanceToStart > 0}<p class="m-0">Distance to start: {t.distanceToStart}</p>{/if}
          {/if}
        {/snippet}
        {#snippet nodeTipTail(n: TreeNodeFx)}
          {@const t = tipInfo?.node === n.id ? tipInfo : undefined}
          <div class="flex flex-col gap-0.5 border-t border-line pt-2 text-12" aria-label="Node tips">
            {#if isAllocSocket(n.id)}
              <!-- PassiveTreeView.lua:1719-1726 (PoB-PoE2) -->
              {#if t?.jewel?.enabled}<p class="m-0 text-text-2">
                  Tip: Right click this socket to go to the items page and choose the jewel for this socket.
                </p>{/if}
            {:else}
              <!-- PassiveTreeView.lua:2011-2057 (PoB-PoE2), without the gold costs -->
              {#each t?.pathLines ?? [] as line, i (i)}<p class={['m-0', line.startsWith('Tip:') ? 'text-text-2' : 'text-text']}>{line}</p>{/each}
              <!-- PassiveTreeView.lua:2085-2093 (PoB-PoE2) -->
              <p class="m-0 text-text-2">Shift + Right-Click to add a build note (PoE2 .build export)</p>
              {#if view?.tree?.notes?.[n.id]}<p class="m-0 whitespace-pre-line text-text" aria-label="Node note">Note: {view.tree.notes[n.id]}</p>{/if}
            {/if}
          </div>
        {/snippet}
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
                bind:this={tree2}
                data={treeData}
                spec={view.tree}
                {busy}
                onclick={treeClick}
                onattribute={treeAttribute}
                onhover={(id: number, allocMode: number, trace?: number[]) => {
                  hover(api, id, trace);
                  return treeHover(id, allocMode);
                }}
                ontrace={treeTrace}
                onunhover={() => hover(api, undefined)}
                onconnect={treeConnect}
                onundo={() => treeUndo(false)}
                onredo={() => treeUndo(true)}
                matches={api.matches}
                {compared}
                {power} />
              {#if powerOpen && powerHover !== undefined}
                {@const p = tree2?.project(powerHover)}
                {#if p}
                  <div
                    class="pointer-events-none absolute size-7 -translate-x-1/2 -translate-y-1/2 rounded-full border-3 border-accent shadow-[0_0_12px_var(--color-accent)]"
                    data-power-hover={powerHover}
                    style:left="{p.x}px"
                    style:top="{p.y}px">
                  </div>
                {/if}
              {/if}
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
  <div class="flex flex-col gap-3 text-13">
    <p class="m-0">Changing class to {confirmClassName} will reset your passive tree.</p>
    <p class="m-0">This can be avoided by connecting one of the {confirmClassName} starting nodes to your tree.</p>
  </div>
  {#snippet actions()}
    <DialogActions
      confirm="Continue"
      tone="danger"
      oncancel={() => (confirmClass = undefined)}
      onconfirm={() => {
        const n = confirmClass;
        confirmClass = undefined;
        if (n) void pick(n, true).catch(logError);
      }}>
      {#snippet extra()}<Button
          size="md"
          onclick={() => {
            const n = confirmClass;
            confirmClass = undefined;
            if (n) void treeConnect(n.id, 0, 0).catch(logError);
          }}>Connect Path</Button
        >{/snippet}
    </DialogActions>
  {/snippet}
</Dialog>
