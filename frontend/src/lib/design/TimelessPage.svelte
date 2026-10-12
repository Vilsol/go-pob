<script lang="ts">
  import type { Snippet } from 'svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import ChoiceCard from './ChoiceCard.svelte';
  import NumberInput from './NumberInput.svelte';
  import StatPicker from './StatPicker.svelte';
  import TreeView from './TreeView.svelte';
  import type { TreeFixture, TreeNodeFx } from './fixtures';
  import {
    insertSearchNode,
    keystoneOf,
    nodeWeightLabels,
    removeSearchRow,
    rewriteSearchRow,
    searchRows,
    socketCost,
    socketCovers,
    socketLabel,
    type TimelessForm,
    type TimelessTreeArgs,
    type TimelessRow,
    type TimelessSocket,
    type TimelessStat
  } from './timeless';

  // TreeTab.lua:1161-2863
  type Jewel = { name: string; kind: string; seeds: string; conquerors: string[] };
  type Props = {
    tree?: TreeFixture;
    treeView?: Snippet<[TimelessTreeArgs]>;
    sockets: TimelessSocket[];
    jewels: Jewel[];
    devotion: string[];
    stats: TimelessStat[];
    protectable?: TimelessStat[];
    fallbackModes: string[];
    realms: string[];
    leagues: string[];
    tradeTypes: string[];
    form: TimelessForm;
    step?: number;
    count?: number;
    rows?: Map<number, TimelessRow>;
    /* TimelessJewelListControl.lua:25-60 */
    sel?: number;
    highlight?: number;
    added?: Set<number>;
    searching?: boolean;
    progress?: number;
    generating?: boolean;
    fallbackFresh?: boolean;
    tradeLabel?: string;
    onchange?: () => void;
    onjewel?: (type: number) => void;
    onsearch?: () => void;
    onreset?: () => void;
    ongenerate?: () => void;
    onrows?: (first: number) => void;
    onadd?: (row: number) => void;
    ontrade?: () => void;
    onclose?: () => void;
    ondone?: () => void;
  };
  let {
    tree,
    treeView,
    sockets,
    jewels,
    devotion,
    stats,
    protectable = [],
    fallbackModes,
    realms,
    leagues,
    tradeTypes,
    form = $bindable(),
    step = $bindable(1),
    count = 0,
    rows = new Map(),
    sel = $bindable(1),
    highlight = $bindable(0),
    added = new Set(),
    searching = false,
    progress = 0,
    generating = false,
    fallbackFresh = $bindable(false),
    tradeLabel = 'Open Trade URL',
    onchange,
    onjewel,
    onsearch,
    onreset,
    ongenerate,
    onrows,
    onadd,
    ontrade,
    onclose,
    ondone
  }: Props = $props();

  const STEPS = ['Jewel', 'Sockets', 'Requirements', 'Ranking', 'Results'];
  const ROW = 28;
  const VISIBLE = 12;
  const uid = $props.id();
  let hover = $state<number | null>(null);
  let hoverRow = $state<number | null>(null);
  let width = $state(0);
  let scrollTop = $state(0);
  let showFallback = $state(false);

  const J = $derived(jewels[form.jewelType - 1] ?? jewels[0]);
  const conqName = $derived(J.conquerors[form.conqueror - 1] ?? J.conquerors[0]);
  const keystone = $derived(conqName === 'Any' ? 'any conqueror’s keystone' : keystoneOf(conqName));
  const isMF = $derived(J.name === 'Militant Faith');
  const socketById = $derived(new Map(sockets.map((s) => [s.id, s])));
  const inView = $derived(new Set(tree?.nodes.map((n) => n.id) ?? []));
  const alloc = $derived(Object.fromEntries((tree?.nodes ?? []).filter((n) => n.a).map((n) => [n.id, true])));
  const row = $derived(rows.get(sel - 1));
  const focus = $derived(step === 5 ? ((hoverRow !== null ? rows.get(hoverRow)?.socket : undefined) ?? row?.socket ?? null) : hover);
  const picked = $derived(new Set(form.socket === -1 ? sockets.map((s) => s.id) : [form.socket]));
  const ringIds = $derived.by(() => {
    const ids = new Set<number>(step === 2 ? [...picked] : step < 5 && form.socket !== -1 ? [form.socket] : []);
    if (focus) ids.add(focus);
    return [...ids].map((id) => ({ id, hot: id === focus }));
  });
  const rings = $derived(ringIds.filter((r) => inView.has(r.id)));
  const conquered = $derived.by(() => {
    const c = tree?.nodes.find((n) => n.id === focus);
    if (!tree || !c) return new Set<number>();
    const r = tree.radius;
    const d2 = (n: TreeNodeFx) => (n.x - c.x) ** 2 + ((n.y - c.y) / tree.aspect) ** 2;
    return new Set(tree.nodes.filter((n) => (n.k === 'n' || n.k === 'k') && d2(n) <= r * r).map((n) => n.id));
  });
  const tipOf = $derived.by(() => {
    const s = hover !== null ? socketById.get(hover) : undefined;
    const n = tree?.nodes.find((x) => x.id === hover);
    if (!tree || !s || !n) return null;
    const h = width / tree.aspect;
    const nx = n.x * width;
    return {
      s,
      x: nx + 18 + 262 <= width ? nx + 18 : Math.max(nx - 18 - 262, 0),
      y: Math.max(Math.min(n.y * h - 30, h - 120), 6)
    };
  });

  const statById = $derived(new Map(stats.map((s) => [s.id, s])));
  const reqs = $derived(searchRows(form.searchList));
  const unpicked = $derived(stats.filter((s) => !reqs.some((r) => r.id === s.id)));
  const twoStats = (id: string) => {
    const s = statById.get(id);
    return !!s && !s.total && s.descriptions.length > 1;
  };
  const changed = () => onchange?.();
  const setSocket = (id: number) => {
    form.socket = id;
    changed();
  };
  // TreeTab.lua:1893-1914
  const addReq = (s: TimelessStat) => {
    form.searchList = insertSearchNode(form.searchList, s.id, nodeWeightLabels(1, twoStats(s.id) ? 1 : 0, 0));
    changed();
  };
  const setReq = (id: string, w: { w1: number; w2: number; w3: number }) => {
    form.searchList = rewriteSearchRow(form.searchList, id, nodeWeightLabels(w.w1, w.w2, w.w3));
    changed();
  };
  const dropReq = (id: string) => {
    form.searchList = removeSearchRow(form.searchList, id);
    changed();
  };
  const setJewel = (i: number) => {
    if (form.jewelType === i + 1) return;
    form.jewelType = i + 1;
    form.conqueror = 1;
    form.searchList = '';
    form.searchListFallback = '';
    form.protect = [];
    onjewel?.(i + 1);
    changed();
  };

  const firstRow = $derived(Math.max(0, Math.floor(scrollTop / ROW) - 2));
  const shown = $derived(Array.from({ length: Math.max(0, Math.min(count, firstRow + VISIBLE + 4) - firstRow) }, (_, i) => firstRow + i));
  const highlighted = (i: number) => highlight > 0 && i + 1 >= Math.min(sel, highlight) && i + 1 <= Math.max(sel, highlight);
  // TimelessJewelListControl.lua:25-60
  function clickRow(i: number, e: MouseEvent) {
    if (e.shiftKey && sel) highlight = i + 1;
    else {
      sel = i + 1;
      highlight = 0;
    }
  }
  function rowKey(i: number, e: KeyboardEvent) {
    if (e.key === 'Enter') onadd?.(i);
    else if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && count) {
      e.preventDefault();
      sel = Math.min(Math.max(sel + (e.key === 'ArrowDown' ? 1 : -1), 1), count);
      highlight = 0;
    }
  }
  $effect(() => {
    const list = document.getElementById(`${uid}-results`);
    if (!list || !sel) return;
    const top = (sel - 1) * ROW;
    if (top < list.scrollTop || top + ROW > list.scrollTop + list.clientHeight) list.scrollTop = top;
  });
  const next = () => {
    if (step === 4) {
      onsearch?.();
      step = 5;
    } else if (step === 5) (ondone ?? onclose)?.();
    else step = step + 1;
  };
</script>

<div class="panel flex shrink-0 items-center gap-3.5 px-3.5 py-2.5">
  <div class="flex flex-col">
    <span class="text-11 tracking-caps text-text-3">TIMELESS JEWEL SEARCH</span>
    <span class="text-15 font-semibold"><span class="toned tone-unique" data-pob-color>{J.name}</span> · {conqName}</span>
  </div>
  <div class="grow"></div>
  <ol aria-label="Steps" class="m-0 flex list-none gap-1 p-0">
    {#each STEPS as label, i (label)}
      <li>
        <button
          class={['chip text-13 aria-[current=step]:border-accent', i + 1 < step ? 'bg-surface-3 text-text' : i + 1 === step ? 'text-text' : 'text-text-3']}
          aria-current={i + 1 === step ? 'step' : undefined}
          onclick={() => (step = i + 1)}><span class="font-mono">{i + 1}</span> {label}</button>
      </li>
    {/each}
  </ol>
  {#if onclose}<button class="btn-close" aria-label="Close timeless search" onclick={onclose}>×</button>{/if}
</div>

{#if step === 1}
  <div class="flex gap-3.5">
    <div role="group" aria-label="Jewels" class="grid grow grid-cols-4 content-start gap-2">
      {#each jewels as j, i (j.name)}
        <ChoiceCard
          pressed={i === form.jewelType - 1}
          class="flex-col items-start gap-0.75 rounded-md bg-surface-1 px-3 py-2.5 aria-pressed:bg-surface-2"
          onclick={() => setJewel(i)}
          ><span class="toned tone-unique font-semibold" data-pob-color>{j.name}</span><span class="text-11 tracking-caps text-text-3">{j.kind}</span><span
            class="text-12 text-text-2"
            >{j.conquerors
              .filter((c) => c !== 'Any')
              .map((c) => c.split(' (')[0])
              .join(', ')}</span
          ><span class="font-mono text-11 text-text-3">seeds {j.seeds}</span></ChoiceCard>
      {/each}
    </div>
    <section aria-label="Conqueror" class="panel flex w-80 shrink-0 flex-col gap-2.5 p-3.5">
      <h2 class="subhead text-12">CONQUEROR</h2>
      {#each J.conquerors as c, i (c)}
        <ChoiceCard
          pressed={i === form.conqueror - 1}
          class="flex-col items-start gap-0.75 rounded-md px-3 py-2.5"
          onclick={() => {
            form.conqueror = i + 1;
            changed();
          }}
          ><span class="font-semibold">{c.split(' (')[0]}</span><span class="text-12 text-text-2">{c === 'Any' ? 'search every conqueror' : keystoneOf(c)}</span
          ></ChoiceCard>
      {/each}
      {#if isMF}
        {#each [1, 2] as k (k)}
          <FloatingLabel label="Devotion modifier {k}"
            ><select
              class="select h-8.5"
              aria-label="Devotion modifier {k}"
              value={k === 1 ? form.devotion1 : form.devotion2}
              onchange={(e) => {
                if (k === 1) form.devotion1 = Number(e.currentTarget.value);
                else form.devotion2 = Number(e.currentTarget.value);
                changed();
              }}
              >{#each devotion as o, i (i)}<option value={i + 1}>{o}</option>{/each}</select
            ></FloatingLabel>
        {/each}
      {/if}
    </section>
  </div>
{:else}
  {#snippet socketTip(id: number)}
    {@const s = socketById.get(id)}
    {#if s}
      <div class="font-semibold">{socketLabel(s)}</div>
      <div class="text-text-2">{socketCost(s)}</div>
      <div class="text-text-2">{socketCovers(s)}</div>
      <div class="text-node-conq">{s.keystones.length ? `${s.keystones.join(', ')} → ${keystone}` : 'no keystone in radius'}</div>
    {/if}
  {/snippet}
  {#snippet legend()}
    <span><span class="inline-block size-2.5 rounded-[5px] bg-accent align-[-1px]"></span> allocated</span>
    <span><span class="box-border inline-block size-2.5 rounded-[5px] border-2 border-node-jewel align-[-1px]"></span> jewel socket</span>
    <span><span class="box-border inline-block size-2.5 rounded-[5px] border-2 border-node-conq align-[-1px]"></span> conquered in radius</span>
  {/snippet}
  <div class="flex items-start gap-3">
    {#if treeView}
      {@render treeView({
        rings: ringIds,
        picked: step === 2 ? picked : undefined,
        centre: step === 5 ? (row?.socket ?? null) : form.socket === -1 ? null : form.socket,
        hovered: hover,
        sockets: new Set(socketById.keys()),
        enter: (id: number) => (hover = id),
        leave: () => (hover = null),
        pick: setSocket,
        tip: socketTip,
        legend
      })}
    {:else if tree}
      <TreeView
        {tree}
        label="Passive tree around the build"
        class="flex-[0_1_1000px] rounded-md border border-line bg-ground"
        {alloc}
        {rings}
        {conquered}
        picked={step === 2 ? picked : undefined}
        active={(n: TreeNodeFx) => socketById.has(n.id)}
        describedBy="{uid}-tip"
        hovered={hover}
        bind:width
        onenter={(n: TreeNodeFx) => (hover = n.id)}
        onleave={() => (hover = null)}
        onpick={(n: TreeNodeFx) => setSocket(n.id)}>
        {#snippet overlay()}
          {#if tipOf}
            <div
              role="tooltip"
              id="{uid}-tip"
              class="tooltip pointer-events-none absolute top-(--y) left-(--x) z-3 flex w-65.5 flex-col gap-0.75 rounded-md px-2.5 py-2 text-12 shadow-pop"
              style:--x="{tipOf.x}px"
              style:--y="{tipOf.y}px">
              {@render socketTip(tipOf.s.id)}
            </div>
          {/if}
          <div class="map-note bottom-2.5 left-2.5 flex flex-wrap gap-3 text-text-2">
            {@render legend()}
            <span>{tree.version.replace('_', '.')} tree · Large radius (1800)</span>
          </div>
        {/snippet}
      </TreeView>
    {:else}
      <div class="panel flex-[0_1_1000px] p-3 text-13 text-text-3">Loading the tree…</div>
    {/if}

    {#if step === 2}
      <section aria-label="Sockets" class="panel flex max-h-[70vh] min-w-0 grow flex-col gap-2 overflow-y-auto p-3 text-13">
        <div class="flex items-center justify-between">
          <h2 class="subhead text-12">JEWEL SOCKET</h2>
          <ChoiceCard pressed={form.socket === -1} class="h-6.5 px-2.5" onclick={() => setSocket(-1)}>All sockets</ChoiceCard>
        </div>
        {#each sockets as s (s.id)}
          <ChoiceCard
            pressed={form.socket === s.id}
            class="flex-col items-start gap-0.5 bg-surface-1 px-2.5 py-2 aria-pressed:border-better aria-pressed:bg-surface-2"
            onclick={() => setSocket(s.id)}
            onmouseenter={() => (hover = s.id)}
            onmouseleave={() => (hover = null)}
            ><span class="flex w-full justify-between gap-2"
              ><span class="font-semibold">{socketLabel(s)}</span><span
                class={['font-mono text-12 whitespace-nowrap', s.pts === 0 ? 'text-accent' : 'text-text-2']}
                >{s.pts < 0 ? '—' : s.pts === 0 ? 'alloc.' : `+${s.pts} pts`}</span
              ></span
            ><span class="note">{socketCovers(s)}{s.jewel ? ` · holds ${s.jewel}` : ''}{inView.has(s.id) ? '' : ' · off the view'}</span></ChoiceCard>
        {/each}
        <h2 class="subhead mt-1.5 text-12">NODES</h2>
        <label class="flex items-center gap-2" title="Exclude nodes that you do not have allocated on your active passive skill tree."
          ><input
            type="checkbox"
            class="check"
            bind:checked={form.filter}
            onchange={() => {
              if (!form.filter) form.protect = [];
              changed();
            }} />Filter nodes: only the ones allocated on the tree</label>
        {#if form.filter}
          <label class="flex items-center justify-between gap-2"
            >Node distance<NumberInput
              bind:value={form.distance}
              min={0}
              max={10}
              size="xs"
              class="w-16"
              label="Node Distance"
              onchange={() => changed()} /></label>
          {#if isMF}
            <StatPicker
              stats={protectable.filter((p) => !form.protect.includes(p.label))}
              label="Protect an allocated node"
              placeholder="Protect an allocated node from changing…"
              onpick={(p: TimelessStat) => (form.protect = [...form.protect, p.label])} />
            {#if form.protect.length}
              <div class="flex flex-wrap items-center gap-1" aria-label="Protected nodes">
                {#each form.protect as p (p)}<span class="chip text-12">{p}</span>{/each}
                <button class="btn btn-sm" onclick={() => (form.protect = [])}>Clear</button>
              </div>
            {/if}
          {/if}
        {/if}
        <div class="note mt-auto">
          Click a socket on the tree or in the list. Hover shows the radius and the points needed to reach it. “#” marks an allocated socket.
        </div>
      </section>
    {:else if step === 3}
      <section aria-label="Requirements" class="panel flex min-w-0 grow flex-col gap-2.5 p-3 text-13">
        <h2 class="subhead text-12">REQUIRED NODES</h2>
        <StatPicker stats={unpicked} label="Add a required stat" placeholder="+ Add a required stat — type to search" onpick={addReq} />
        <div class="flex max-h-[50vh] flex-col gap-2 overflow-y-auto">
          {#each reqs as r, i (`${i}:${r.id}`)}
            {@const s = statById.get(r.id)}
            {@const name = s?.label ?? r.id}
            <div class="flex flex-col gap-1 border-b border-line pb-2" data-req={r.id}>
              <div class="flex items-start justify-between gap-2">
                <span class="flex min-w-0 flex-col"
                  ><span class="font-semibold">{name}</span>{#if s?.descriptions.length}<span class="note-sm whitespace-pre-line"
                      >{s.descriptions.join('\n')}</span
                    >{/if}</span>
                <button class="btn-close" aria-label="Remove {name}" onclick={() => dropReq(r.id)}>×</button>
              </div>
              <div class="flex flex-wrap items-center gap-3 text-11 text-text-3">
                <label class="flex items-center gap-1"
                  >weight<NumberInput
                    value={r.w1}
                    min={0}
                    max={10}
                    step={0.1}
                    size="xs"
                    class="w-16"
                    label="{name} weight"
                    onchange={(v: number) => setReq(r.id, { ...r, w1: v })} /></label>
                {#if twoStats(r.id)}
                  <label class="flex items-center gap-1"
                    >second stat<NumberInput
                      value={r.w2}
                      min={0}
                      max={10}
                      step={0.1}
                      size="xs"
                      class="w-16"
                      label="{name} secondary weight"
                      onchange={(v: number) => setReq(r.id, { ...r, w2: v })} /></label>
                {/if}
                <label class="flex items-center gap-1"
                  >min<NumberInput
                    value={r.w3 >= 500 ? 0 : r.w3}
                    min={0}
                    max={490}
                    step={10}
                    size="xs"
                    class="w-16"
                    disabled={r.w3 >= 500}
                    label="{name} minimum"
                    onchange={(v: number) => setReq(r.id, { ...r, w3: v })} /></label>
                <label class="flex items-center gap-1"
                  ><input
                    type="checkbox"
                    class="check"
                    aria-label="{name} required"
                    checked={r.w3 >= 500}
                    onchange={(e) => setReq(r.id, { ...r, w3: e.currentTarget.checked ? 500 : 0 })} />required</label>
              </div>
            </div>
          {:else}
            <span class="note">No required nodes yet: pick the stats the jewel should give.</span>
          {/each}
        </div>
        <details class="text-12">
          <summary class="cursor-pointer text-text-2">Edit as text (PoB's desired nodes list)</summary>
          <textarea class="input mt-1 h-32 w-full font-mono text-12" aria-label="Desired nodes" bind:value={form.searchList} onchange={changed}></textarea>
        </details>
      </section>
    {:else if step === 4}
      <section aria-label="Ranking" class="panel flex min-w-0 grow flex-col gap-2.5 p-3 text-13">
        <h2 class="subhead text-12">FALLBACK NODES</h2>
        <div class="note">
          Seeds that tie on the required nodes are ranked by the fallback list: generate it from how much each node changes a stat of the build.
        </div>
        <div class="flex items-end gap-2">
          <FloatingLabel label="Fallback weight mode" class="grow"
            ><select class="select h-8.5" aria-label="Fallback Weight Mode" bind:value={form.fallbackMode} onchange={changed}
              >{#each fallbackModes as m, i (i)}<option value={i + 1}>{m}</option>{/each}</select
            ></FloatingLabel>
          <button
            class="btn btn-md"
            disabled={form.jewelType === 1 || generating}
            title={form.jewelType === 1
              ? "Glorious Vanity's rolled stats are not weighed yet"
              : 'Weighs every node of the list by how much it changes the chosen stat when added to the build'}
            onclick={() => ongenerate?.()}>{generating ? 'Generating…' : 'Generate'}</button>
        </div>
        <div class="flex gap-1">
          <button class={['btn btn-sm', !showFallback && 'btn-primary']} aria-pressed={!showFallback} onclick={() => (showFallback = false)}
            >Desired Nodes</button>
          <button
            class={['btn btn-sm', showFallback && 'btn-primary']}
            aria-pressed={showFallback}
            onclick={() => {
              showFallback = true;
              fallbackFresh = false;
            }}><span class={fallbackFresh ? 'text-accent' : ''}>Fallback Nodes</span></button>
        </div>
        {#if showFallback}
          <textarea class="input h-40 font-mono text-12" aria-label="Fallback nodes" bind:value={form.searchListFallback} onchange={changed}></textarea>
        {:else}
          <textarea class="input h-40 font-mono text-12" aria-label="Desired nodes list" bind:value={form.searchList} onchange={changed}></textarea>
        {/if}
        <label class="flex items-center justify-between gap-2"
          >Minimum total weight<input
            class="input h-7 w-24"
            inputmode="numeric"
            aria-label="Total Minimum Weight"
            title="Optional: only show results whose total weight meets this"
            placeholder="none"
            bind:value={form.totalMinimum}
            oninput={() => (form.totalMinimum = form.totalMinimum.replace(/\D/g, ''))} /></label>
      </section>
    {:else}
      <section aria-label="Results" class="panel flex min-w-0 grow flex-col overflow-hidden text-12">
        <div class="flex items-center justify-between gap-2 px-3 py-2">
          <span class="note-sm" aria-live="polite">{searching ? `Searching… ${Math.round(progress * 100)}%` : `${count} results`}</span>
          <span class="flex gap-1.5">
            <button class="btn btn-sm" disabled={searching} onclick={() => onsearch?.()}>Search again</button>
            <button class="btn btn-sm" onclick={() => onreset?.()}>Reset</button>
          </span>
        </div>
        <div
          class="grid grid-cols-[72px_minmax(0,150px)_64px_minmax(0,1fr)] gap-2 border-t border-line px-2 py-1 text-11 tracking-caps text-text-3 uppercase"
          aria-hidden="true">
          <span>Seed</span><span>Socket</span><span class="text-right">Weight</span><span>Matched nodes</span>
        </div>
        <div
          id="{uid}-results"
          class="relative h-[264px] overflow-y-auto border-y border-line"
          role="listbox"
          aria-label="Search results"
          tabindex="0"
          onscroll={(e) => {
            scrollTop = e.currentTarget.scrollTop;
            onrows?.(Math.floor(scrollTop / ROW));
          }}
          onmouseleave={() => (hoverRow = null)}>
          <div style:height="{count * ROW}px">
            {#each shown as i (i)}
              {@const r = rows.get(i)}
              {@const rs = r && socketById.get(r.socket)}
              {@const plain = !highlighted(i) && !added.has(i)}
              <div
                role="option"
                aria-selected={sel === i + 1}
                aria-label={r?.label.trim()}
                tabindex="-1"
                data-on={sel === i + 1}
                data-seed={r?.seed}
                class={[
                  'absolute inset-x-0 grid cursor-default grid-cols-[72px_minmax(0,150px)_64px_minmax(0,1fr)] items-center gap-2 px-2 whitespace-nowrap hover:bg-hover data-[on=true]:bg-surface-3 data-[on=true]:shadow-[inset_2px_0_0_var(--color-node-conq)]',
                  i % 2 && 'bg-bar',
                  highlighted(i) && (sel === i + 1 || highlight === i + 1 ? 'text-[#ff8000]' : 'text-[#ffff00]'),
                  added.has(i) && 'text-[#b2b2b2]'
                ]}
                style:top="{i * ROW}px"
                style:height="{ROW}px"
                onclick={(e) => clickRow(i, e)}
                ondblclick={() => onadd?.(i)}
                onkeydown={(e) => rowKey(i, e)}
                onmouseenter={() => (hoverRow = i)}>
                {#if r}
                  <span class="font-mono font-semibold" data-cell="seed">{r.seed}</span>
                  <span class={['truncate', plain && 'text-text-2']} data-cell="socket">{rs ? socketLabel(rs) : ''}</span>
                  <span class={['text-right font-mono', plain && 'text-accent']} data-cell="weight">{r.total}</span>
                  <span class="flex min-w-0 gap-1 overflow-hidden" data-cell="nodes">
                    {#each r.nodes as n, k (k)}
                      <span
                        class={[
                          'shrink-0 rounded-sm px-1.5 leading-[18px]',
                          n.targets.length ? 'bg-surface-3' : 'border border-dashed border-line',
                          plain && (n.targets.length ? 'text-text' : 'text-text-3')
                        ]}
                        title="{n.name}: {n.targets.length ? n.targets.join(', ') : 'None'}"
                        >{n.name} <span class="font-mono font-semibold">{n.targets.length}</span></span>
                    {/each}
                  </span>
                {/if}
              </div>
            {/each}
          </div>
        </div>
        {#if row}
          {@const s = socketById.get(row.socket)}
          <div class="flex flex-col gap-1 px-3 py-2.5" aria-label="Result preview" role="group">
            <span class="font-semibold">Seed {row.seed}{s ? ` in ${socketLabel(s)}` : ''}</span>
            <span class="text-node-conq">{s?.keystones.length ? `${s.keystones.join(', ')} → ${keystone}` : `→ ${keystone}`}</span>
            {#each row.nodes as n, i (i)}<span class="text-text-2">{n.name}: {n.targets.length ? n.targets.join(', ') : 'None'}</span>{/each}
            {#if row.total > 0}<span class="text-text-2">Combined node weight: {row.total}</span>{/if}
            <div class="mt-1 flex gap-1.5">
              <button class="btn btn-sm btn-primary px-3" disabled={added.has(sel - 1)} onclick={() => onadd?.(sel - 1)}
                >{added.has(sel - 1) ? 'Added to the build' : 'Add to the build'}</button>
            </div>
          </div>
        {/if}
        <div class="flex flex-wrap items-center gap-2 border-t border-line px-3 py-2.5" role="group" aria-label="Trade search">
          <select class="select h-7" aria-label="Realm" bind:value={form.realm}
            >{#each realms as r (r)}<option>{r}</option>{/each}</select>
          <input class="input h-7 w-36" list="{uid}-leagues" aria-label="League" bind:value={form.league} />
          <datalist id="{uid}-leagues"
            >{#each leagues as l (l)}<option value={l}></option>{/each}</datalist>
          <select class="select h-7" aria-label="Buyout" bind:value={form.tradeType}
            >{#each tradeTypes as t, i (i)}<option value={i + 1}>{t}</option>{/each}</select>
          <label
            class="flex items-center gap-1"
            title="By default, this tool only searches for 4 or 10 (depending on conqueror) seeds at once to preserve sorting. This option searches up to 180 or 60 seeds at once if you don't care about the sorting."
            ><input type="checkbox" class="check" bind:checked={form.searchMore} /><span>Search maximum amount</span></label>
          <button class="btn btn-sm" disabled={count === 0} onclick={() => ontrade?.()}>{tradeLabel}</button>
        </div>
        <div class="note-sm px-3 pb-2">Click selects, shift+click marks a range for the trade search; double-click or Enter adds the jewel to the build.</div>
      </section>
    {/if}
  </div>
{/if}

<div class="grow"></div>
<div class="flex shrink-0 justify-end gap-2">
  <button class="btn btn-md" disabled={step === 1} onclick={() => (step = Math.max(1, step - 1))}>Back</button>
  <button class="btn btn-md btn-primary px-4" disabled={step === 4 && searching} onclick={next}>{step === 4 ? 'Search' : step === 5 ? 'Done' : 'Next'}</button>
</div>
