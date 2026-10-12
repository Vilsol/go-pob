<script lang="ts">
  import Dialog from '$lib/design/Dialog.svelte';
  import TimelessPage from '$lib/design/TimelessPage.svelte';
  import type { TreeFixture } from '$lib/design/fixtures';
  import type { TimelessForm, TimelessRow, TimelessSocket, TimelessStat, TimelessTreeArgs } from '$lib/design/timeless';
  import { syncWrap } from '$lib/go/worker';
  import { classStartNodes, loadSkillTree, skillTree, skillTreeVersion } from '$lib/skill_tree';
  import { logError } from '$lib/utils';
  import type { calculator, pob } from '$lib/types';
  import { timelessTree } from './timeless_tree';
  import TimelessTreeView from './TimelessTreeView.svelte';

  // TreeTab.lua:1161-2863
  type Props = { open?: boolean };
  let { open = $bindable(false) }: Props = $props();

  const VISIBLE = 12;
  const LEAGUES = ['Standard', 'Hardcore', 'Ruthless', 'Hardcore Ruthless'];
  const SEEDS: Record<string, string> = {
    'Glorious Vanity': '100–8000',
    'Lethal Pride': '10000–18000',
    'Brutal Restraint': '500–8000',
    'Militant Faith': '2000–10000',
    'Elegant Hubris': '2000–160000',
    'Heroic Tragedy': '100–8000'
  };

  let options = $state<Awaited<ReturnType<NonNullable<typeof syncWrap>['GetTimelessOptions']>>>();
  let offered = $state.raw<calculator.TimelessSocket[]>([]);
  let mods = $state.raw<calculator.TimelessMod[]>([]);
  let protectable = $state.raw<calculator.TimelessMod[]>([]);
  let fallbackModes = $state.raw<{ label: string; stat: string }[]>([]);
  let view = $state.raw<{ tree: TreeFixture; sockets: TimelessSocket[] }>();

  let form = $state<TimelessForm>({
    jewelType: 1,
    conqueror: 1,
    devotion1: 1,
    devotion2: 1,
    socket: -1,
    filter: false,
    distance: 0,
    protect: [],
    searchList: '',
    searchListFallback: '',
    fallbackMode: 1,
    totalMinimum: '',
    realm: 'PC',
    league: 'Standard',
    tradeType: 1,
    searchMore: false
  });
  let step = $state(1);
  let generating = $state(false);
  let fallbackFresh = $state(false);
  let searching = $state(false);
  let progress = $state(0);
  let count = $state(0);
  let desired = $state.raw<calculator.TimelessDesiredNode[]>([]);
  let rows = $state.raw<Map<number, TimelessRow>>(new Map());
  let sel = $state(1);
  let highlight = $state(0);
  let added = $state.raw<Set<number>>(new Set());
  let last = $state<[number, number] | undefined>();
  let tradeLabel = $state('Open Trade URL');
  let loaded = false;

  const jewels = $derived(
    (options?.types ?? []).map((t) => ({ name: t.Label, kind: 'CLASSIC', seeds: SEEDS[t.Label] ?? '', conquerors: options?.conquerors[t.ID] ?? [] }))
  );
  const stats = $derived<TimelessStat[]>(
    mods.filter((m) => m.ID).map((m) => ({ id: m.ID, label: m.Label.split(/\s{2,}/)[0], descriptions: [...(m.Descriptions ?? [])], total: m.TotalMod }))
  );
  const protects = $derived<TimelessStat[]>(protectable.map((p) => ({ id: p.Label, label: p.Label, descriptions: [...(p.Descriptions ?? [])] })));
  const sockets = $derived<TimelessSocket[]>(
    view?.sockets ??
      offered
        .filter((s) => s.ID !== -1)
        .map((s) => ({ id: s.ID, label: s.Label, pts: s.Label.startsWith('# ') ? 0 : -1, notables: [], keystones: [s.Keystone], smalls: 0 }))
  );

  async function readLists() {
    const l = await syncWrap?.GetTimelessLists(form.jewelType);
    offered = l?.sockets ?? [];
    mods = l?.mods ?? [];
    if (!offered.some((s) => s.ID === form.socket)) form.socket = -1;
  }

  async function readTree() {
    if (!syncWrap) return;
    const version = await syncWrap.GetTreeVersion();
    if (version && version !== $skillTreeVersion) await loadSkillTree(version);
    const t = $skillTree;
    if (!t || !version) return;
    const active = (await syncWrap.GetPassiveNodes()) ?? [];
    const info = await syncWrap.GetBuildInfo();
    const roots = classStartNodes[t.classes.findIndex((c) => c.name === info?.className)] ?? [];
    const paths = (await syncWrap.CalculateAllocationPaths(version, [...active], [...roots])) ?? {};
    view = timelessTree(version, t, active, offered, paths);
  }

  // Build.lua:975-993
  $effect(() => {
    if (!open || !syncWrap) return;
    loaded = false;
    void (async () => {
      options = await syncWrap.GetTimelessOptions();
      fallbackModes = await syncWrap.GetTimelessFallbackModes();
      const d = await syncWrap.GetTimelessData();
      const num = (s: string | undefined, f: number) => (s !== undefined && s !== '' && !isNaN(Number(s)) ? Number(s) : f);
      form.jewelType = Math.min(Math.max(num(d?.JewelTypeID, 1), 1), 6);
      form.conqueror = num(d?.ConquerorTypeID, 1);
      form.devotion1 = num(d?.DevotionVariant1, 1);
      form.devotion2 = num(d?.DevotionVariant2, 1);
      form.socket = num(d?.JewelSocketID, -1);
      form.fallbackMode = num(d?.FallbackWeightModeIdx, 1);
      form.filter = d?.SocketFilter === 'true';
      form.distance = num(d?.SocketFilterDistance, 0);
      form.searchList = d?.SearchList ?? '';
      form.searchListFallback = d?.SearchListFallback ?? '';
      await readLists();
      loaded = true;
      await readTree();
    })().catch(logError);
  });

  function save() {
    if (!loaded) return;
    const d: pob.TimelessData = {
      JewelTypeID: String(form.jewelType),
      ConquerorTypeID: String(form.conqueror),
      DevotionVariant1: String(form.devotion1),
      DevotionVariant2: String(form.devotion2),
      JewelSocketID: String(form.socket),
      FallbackWeightModeIdx: String(form.fallbackMode),
      SocketFilter: form.filter ? 'true' : '',
      SocketFilterDistance: String(form.distance),
      SearchList: form.searchList,
      SearchListFallback: form.searchListFallback
    };
    void syncWrap?.SetTimelessData(d);
  }

  // TreeTab.lua:1558-1588
  $effect(() => {
    const socket = form.socket;
    if (!open || !form.filter || socket === -1) {
      protectable = [];
      return;
    }
    void syncWrap
      ?.GetTimelessProtectable(socket)
      .then((p) => (protectable = p))
      .catch(logError);
  });

  // TreeTab.lua:2112-2115
  async function generate() {
    const mode = fallbackModes[form.fallbackMode - 1];
    if (!mode || !syncWrap) return;
    generating = true;
    try {
      const list = await syncWrap.GenerateTimelessFallback(form.jewelType, mode.stat);
      if (list === undefined) return;
      form.searchListFallback = list;
      fallbackFresh = true;
      save();
    } finally {
      generating = false;
    }
  }

  function reset() {
    void syncWrap?.CancelTimelessSearch();
    form.searchList = '';
    form.searchListFallback = '';
    form.protect = [];
    count = 0;
    rows = new Map();
    desired = [];
    save();
  }

  const rowOf = (r: calculator.TimelessSearchResult): TimelessRow => ({
    label: r.Label,
    seed: r.Seed,
    total: r.Total,
    socket: Number(r.SocketID || form.socket),
    nodes: desired.flatMap((d) => {
      const n = (r.Nodes ?? []).find((x) => x.ID === d.ID);
      return n ? [{ name: d.DisplayName, targets: [...(n.TargetNodeNames ?? [])] }] : [];
    })
  });

  async function fetchRows(first: number) {
    const start = Math.max(0, first - VISIBLE);
    if (rows.has(start) && rows.has(Math.min(count - 1, start + 3 * VISIBLE))) return;
    const got = (await syncWrap?.TimelessRows(start, 4 * VISIBLE)) ?? [];
    const next = new Map(rows);
    got.forEach((r, i) => next.set(start + i, rowOf(r)));
    rows = next;
  }

  async function search() {
    if (!syncWrap) return;
    searching = true;
    progress = 0;
    const timer = setInterval(() => {
      void syncWrap
        ?.TimelessSearchProgress()
        .then((p) => (progress = p))
        .catch(logError);
    }, 200);
    try {
      const minimum = form.totalMinimum.trim() === '' ? NaN : Number(form.totalMinimum);
      const r = await syncWrap.TimelessSearch({
        JewelType: form.jewelType,
        ConquerorType: form.conqueror,
        SocketID: form.socket,
        SocketFilter: form.filter,
        SocketFilterDistance: form.filter ? form.distance : 0,
        ProtectedNodes: $state.snapshot(form.protect),
        SearchList: form.searchList,
        SearchListFallback: form.searchListFallback,
        TotalMinimumWeight: isNaN(minimum) ? 0 : minimum,
        HasTotalMinimumWeight: !isNaN(minimum)
      });
      if (!r) return;
      count = r.count;
      desired = r.desired;
      rows = new Map();
      added = new Set();
      sel = 1;
      highlight = 0;
      last = undefined;
      tradeLabel = 'Open Trade URL';
      await fetchRows(0);
    } finally {
      clearInterval(timer);
      searching = false;
    }
  }

  async function addRow(i: number) {
    if (added.has(i) || !syncWrap) return;
    const socket = rows.get(i)?.socket ?? form.socket;
    const keystone = offered.find((s) => s.ID === socket)?.Keystone ?? 'Unknown';
    const id = await syncWrap.AddTimelessResult(i, form.devotion1, form.devotion2, keystone);
    if (id) added = new Set([...added, i]);
  }

  async function openTrade() {
    const t = await syncWrap?.TimelessTradeURL({
      JewelType: form.jewelType,
      ConquerorType: form.conqueror,
      DevotionVariant1: form.devotion1,
      DevotionVariant2: form.devotion2,
      SelIndex: sel,
      HighlightIndex: highlight,
      SearchMore: form.searchMore,
      TradeTypeIndex: form.tradeType,
      Realm: form.realm,
      League: form.league,
      LastSearchStart: last?.[0] ?? 0,
      LastSearchCount: last?.[1] ?? 0,
      LastSearchPresent: !!last
    });
    if (!t?.URL) return;
    sel = t.SelIndex;
    highlight = t.HighlightIndex;
    last = [t.LastSearchStart, t.LastSearchCount];
    void fetchRows(sel - 1).catch(logError);
    window.open(t.URL, '_blank', 'noopener');
    void navigator.clipboard?.writeText(t.URL).catch(() => undefined);
    tradeLabel = 'Open Next Trade URL';
  }
</script>

{#snippet liveTree(args: TimelessTreeArgs)}
  <TimelessTreeView tree={$skillTree!} version={view!.tree.version} {...args} />
{/snippet}

<Dialog bind:open title="Find a Timeless Jewel" sheet class="w-[min(1400px,98vw)]" onclose={() => void syncWrap?.CancelTimelessSearch()}>
  {#if options}
    <div class="flex min-h-[640px] flex-col gap-2.5 p-3.5" data-testid="timeless-dialog">
      <TimelessPage
        tree={view?.tree}
        treeView={view && $skillTree ? liveTree : undefined}
        {sockets}
        {jewels}
        devotion={options.devotion}
        {stats}
        protectable={protects}
        fallbackModes={fallbackModes.map((m) => m.label)}
        realms={options.realms}
        leagues={LEAGUES}
        tradeTypes={options.tradeTypes}
        bind:form
        bind:step
        {count}
        {rows}
        bind:sel
        bind:highlight
        {added}
        {searching}
        {progress}
        {generating}
        bind:fallbackFresh
        {tradeLabel}
        onchange={save}
        onjewel={() => void readLists().catch(logError)}
        onsearch={() => void search().catch(logError)}
        onreset={reset}
        ongenerate={() => void generate().catch(logError)}
        onrows={(first: number) => void fetchRows(first).catch(logError)}
        onadd={(i: number) => void addRow(i).catch(logError)}
        ontrade={() => void openTrade().catch(logError)}
        ondone={() => (open = false)} />
    </div>
  {/if}
</Dialog>
