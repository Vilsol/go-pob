<script lang="ts">
  import { untrack } from 'svelte';
  import { goto as gotoPage } from '$app/navigation';
  import { base as appBase } from '$app/paths';
  import TreePage from '$lib/design/TreePage.svelte';
  import { treeManageRequest } from './tree_manage';
  import type { TreeFixture, TreeNodeFx } from '$lib/design/fixtures';
  import { showPowerRows, type CanvasApi, type PowerFilter, type PowerRow, type PowerSort, type SpecOp } from '$lib/design/tree';
  import { displayStats } from '$lib/display/stats';
  import { formatNumSep, luaFormat } from '$lib/display/format_stat';
  import SkillTree from '$lib/components/skill-tree/SkillTree.svelte';
  import { allocationCosts, calculateAllocationPath, type PrecalculatedAllocationPaths } from '$lib/components/skill-tree/paths';
  import { clusterNodeAsTreeNode } from '$lib/components/skill-tree/scene';
  import JewelRing from '$lib/components/skill-tree/JewelRing.svelte';
  import { drawnNodes, loadSkillTree, skillTree, skillTreeVersion } from '$lib/skill_tree';
  import { ascendancyOffsets, drawnByPoB, nodeCentre, toScreen, type Point } from '$lib/skill_tree/geometry';
  import type { Node, TreeClusterNode } from '$lib/skill_tree/types';
  import type { NodeStatDiff, StatDiffBlock } from '$lib/skill_tree/stat_compare';
  import { estimatePlayerProgress } from '$lib/display/points';
  import { currentBuild, devMode, outputs } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import Dialog from '$lib/design/Dialog.svelte';
  import Button from '$lib/design/Button.svelte';
  import TattooPopup from '$lib/design/TattooPopup.svelte';
  import { tattooEditKind, type TattooChoiceFx } from '$lib/skill_tree/tattoo';
  import { tone, type Tone } from '$lib/design/tone';
  import type { calculator, exposition, pob } from '$lib/types';
  import type { ClassChangeResult } from '$lib/skill_tree/class_change';
  import { heatMixValues, heatValues } from '$lib/skill_tree/heat';
  import TimelessDialog from './TimelessDialog.svelte';
  import { PowerFlashes } from './power_flash';
  import TooltipLines from '$lib/design/TooltipLines.svelte';
  import { colorHighlight, edgeSearchHighlight, hexColour } from '$lib/skill_tree/options';

  const POWER_SETTLE_MS = 1000;
  const CLASSES = ['Scion', 'Marauder', 'Ranger', 'Witch', 'Duelist', 'Templar', 'Shadow'];
  const ASCENDANCIES: Record<string, string[]> = {
    Scion: ['Ascendant', 'Reliquarian', 'Luminary'],
    Marauder: ['Juggernaut', 'Berserker', 'Chieftain'],
    Ranger: ['Warden', 'Deadeye', 'Pathfinder'],
    Witch: ['Occultist', 'Elementalist', 'Necromancer'],
    Duelist: ['Slayer', 'Gladiator', 'Champion'],
    Templar: ['Inquisitor', 'Hierophant', 'Guardian'],
    Shadow: ['Assassin', 'Trickster', 'Saboteur']
  };

  let version = $state<string>();
  let info = $state<{ className: string; ascendClassName: string; level: number; recordedTreeVersion?: string }>();
  let readSeq = 0;
  let editSeq = 0;
  let treeSpecs = $state.raw<{ active: number; specs: calculator.TreeSpecRow[] }>();
  $effect(() => {
    void $currentBuild;
    void $outputs;
    const read = ++readSeq;
    syncWrap
      ?.GetTreeSpecs()
      .then((s) => {
        if (s && read === readSeq) treeSpecs = s;
      })
      .catch(logError);
    syncWrap
      ?.GetTreeVersion()
      .then((v) => {
        if (read === readSeq) version = v;
      })
      .catch(logError);
    const edits = editSeq;
    syncWrap
      ?.GetBuildInfo()
      .then((i) => {
        if (!i || read !== readSeq || edits !== editSeq) return;
        info = { className: i.className, ascendClassName: i.ascendClassName, level: i.level, recordedTreeVersion: i.recordedTreeVersion };
      })
      .catch(logError);
  });

  $effect(() => {
    if (version) {
      loadSkillTree(version).catch(logError);
    }
  });

  // Build.lua:263-275, PassiveTreeView.lua:474-490
  let classConfirm = $state<{ className: string; answer: (decision: 1 | 2) => Promise<ClassChangeResult> | undefined }>();
  const answerClass = async (decision: 1 | 2) => {
    const c = classConfirm;
    classConfirm = undefined;
    if (!c) return;
    editSeq++;
    const r = await c.answer(decision);
    if (r?.changed) currentBuild.set($currentBuild);
  };
  const setClass = async (className: string) => {
    editSeq++;
    const previous = info?.className;
    if (info) info.className = className;
    const r = await syncWrap?.ChangeTreeClass(className);
    if (r?.needsConfirm) {
      if (info && previous !== undefined) info.className = previous;
      classConfirm = { className: r.className, answer: (decision) => syncWrap?.ChangeTreeClass(className, decision) };
      return;
    }
    if (info && r?.changed) info.ascendClassName = 'None';
    currentBuild.set($currentBuild);
  };
  const setAscendancy = async (ascendancy: string) => {
    editSeq++;
    if (info) info.ascendClassName = ascendancy;
    await syncWrap?.ChangeTreeAscendancy(ascendancy);
    currentBuild.set($currentBuild);
  };
  let specPoints = $state.raw<calculator.TreeSpecPoints[]>([]);
  let managing = false;
  const readPoints = () => {
    managing = true;
    syncWrap
      ?.GetTreeSpecsPoints()
      .then((p) => (specPoints = p ?? []))
      .catch(logError);
  };
  // TreeTab.lua:39-46
  const setSpec = async (index: number) => {
    editSeq++;
    await syncWrap?.SetActiveTreeSpec(index + 1);
    currentBuild.set($currentBuild);
  };
  const editSpecs = async (op: SpecOp) => {
    editSeq++;
    if (op.op === 'new') await syncWrap?.NewTreeSpec(op.title);
    else if (op.op === 'copy') await syncWrap?.CopyTreeSpec(op.index + 1, op.title);
    else if (op.op === 'rename') await syncWrap?.RenameTreeSpec(op.index + 1, op.title);
    else if (op.op === 'move') await syncWrap?.MoveTreeSpec(op.from + 1, op.to + 1);
    else await syncWrap?.DeleteTreeSpec(op.index + 1);
    currentBuild.set($currentBuild);
    if (managing) readPoints();
  };

  // TreeTab.lua:51-104
  let specTip = $state<{
    index: number;
    title: string;
    lines: string[];
    footer: string;
    pending: boolean;
    empty?: string;
    blocks: StatDiffBlock[];
  }>();
  let specHover: number | null = null;
  const hoverSpec = async (index: number | null) => {
    specHover = index;
    const row = index === null ? undefined : treeSpecs?.specs[index];
    if (index === null || !row) {
      specTip = undefined;
      return;
    }
    const points = (await syncWrap?.GetTreeSpecsPoints())?.[index];
    if (specHover !== index) return;
    const lines = [`Class: ${row.ClassName}`, `Ascendancy: ${row.AscendClassName}`, `Points used: ${points?.Used ?? 0}`];
    if (points?.Sockets) lines.push(`Jewel sockets: ${points.Sockets}`);
    const active = (treeSpecs?.active ?? 1) - 1;
    // TreeTab.lua:69-97
    const respec = index === active ? undefined : await syncWrap?.TreeSpecRespecCost(index);
    if (specHover !== index) return;
    const footer = [
      ...(respec?.Gold ? [`${formatNumSep(String(respec.Gold))} Gold required to switch to this tree.`] : []),
      ...(respec?.Passives ? [`${respec.Passives} ${respec.Passives === 1 ? 'Passive node' : 'Passive nodes'} to be refunded.`] : []),
      ...(respec?.Ascendancy ? [`${respec.Ascendancy} ${respec.Ascendancy === 1 ? 'Ascendancy node' : 'Ascendancy nodes'} to be refunded.`] : []),
      `Game Version: ${row.VersionLabel}`
    ].join('\n');
    const tip = { index, title: row.Title, lines, footer, pending: false, blocks: [] as StatDiffBlock[] };
    if (index === active) {
      specTip = tip;
      return;
    }
    specTip = { ...tip, pending: true };
    let r: Awaited<ReturnType<NonNullable<typeof syncWrap>['StatCompareTreeSpec']>> | undefined;
    for (let tries = 0; tries < 3; tries++) {
      r = await syncWrap?.StatCompareTreeSpec(index + 1);
      if (r || specHover !== index) break;
    }
    if (specHover !== index || !r) return;
    const header = 'Switching to this tree will give you:';
    specTip = r.sections.length
      ? { ...tip, blocks: [{ header, sections: r.sections, readyAt: performance.now(), calcMs: r.calcMs }] }
      : { ...tip, empty: 'No changes from switching to this tree.' };
  };
  $effect(() => {
    void $outputs;
    specHover = null;
    specTip = undefined;
  });

  // TreeTab.lua:156-175, 588-680
  let versionOptions = $state.raw<{ value: string; label: string }[]>([]);
  $effect(() => {
    syncWrap
      ?.GetTreeVersionOptions()
      .then((v) => (versionOptions = v ?? []))
      .catch(logError);
  });
  const convertTree = async (to: string, replace: boolean) => {
    editSeq++;
    const msg = (await syncWrap?.ConvertTreeSpec(to, replace)) ?? '';
    currentBuild.set($currentBuild);
    return msg;
  };
  const convertAllTrees = async (to: string) => {
    editSeq++;
    const msg = (await syncWrap?.ConvertAllTreeSpecs(to)) ?? '';
    currentBuild.set($currentBuild);
    return msg;
  };

  // Build.lua:211-217
  // Build.lua:285-299, 1093-1138
  let secondary = $state<{ entries: { id: number; label: string }[]; selected: number; enabled: boolean }>();
  $effect(() => {
    void $currentBuild;
    void $outputs;
    syncWrap
      ?.GetSecondaryAscendancies()
      .then((d) => {
        if (d) secondary = { entries: (d.Entries ?? []).map((e) => ({ id: e.AscendClassID, label: e.Label })), selected: d.Selected, enabled: d.Enabled };
      })
      .catch(logError);
  });
  const setSecondary = async (id: number) => {
    if (secondary) secondary.selected = id;
    await syncWrap?.ChangeSecondaryAscendancy(id);
    currentBuild.set($currentBuild);
  };
  type AutoLevelWorker = { SetCharacterLevelAutoMode(auto: boolean): Promise<void> | void };
  let levelAuto = $state(false);
  $effect(() => {
    void $currentBuild;
    void $outputs;
    syncWrap
      ?.GetCharacterLevelAutoMode()
      .then((a) => (levelAuto = !!a))
      .catch(logError);
  });
  const setLevelAuto = async (auto: boolean) => {
    editSeq++;
    levelAuto = auto;
    await (syncWrap as unknown as AutoLevelWorker | undefined)?.SetCharacterLevelAutoMode(auto);
    currentBuild.set($currentBuild);
  };

  const setLevel = async (level: number) => {
    editSeq++;
    if (info) info.level = level;
    await syncWrap?.SetLevel(level);
    currentBuild.set($currentBuild);
  };

  const progress = $derived($outputs?.PointsUsed ? estimatePlayerProgress($outputs.PointsUsed, $outputs.Output?.ExtraPoints) : undefined);

  // PassiveSpec.lua:1885-2271
  let clusterNodes = $state.raw<TreeClusterNode[]>([]);
  $effect(() => {
    void $currentBuild;
    void $outputs;
    syncWrap
      ?.GetTreeClusterNodes()
      .then((c) => (clusterNodes = c ?? []))
      .catch(logError);
  });
  // PassiveTreeView.lua:1102-1107
  let compareCluster = $state.raw<TreeClusterNode[]>([]);
  const loaded = $derived($skillTree && version && $skillTreeVersion === version ? $skillTree : undefined);
  const kind = (n: Node): TreeNodeFx['k'] =>
    n.classStartIndex !== undefined || n.isAscendancyStart ? 'c' : n.isKeystone ? 'k' : n.isMastery ? 'm' : n.isJewelSocket ? 'j' : n.isNotable ? 'n' : 's';
  const base = $derived.by(() => {
    if (!loaded) return { nodes: [] as TreeNodeFx[], byId: new Map<number, Node>(), at: new Map<number, Point>() };
    const offsets = ascendancyOffsets(loaded);
    const byId = new Map<number, Node>();
    const at = new Map<number, Point>();
    const nodes: TreeNodeFx[] = [];
    for (const n of drawnNodes.values()) {
      if (n.skill === undefined || !drawnByPoB(loaded, n)) continue;
      byId.set(n.skill, n);
      const p = nodeCentre(loaded, offsets, n);
      if (p) at.set(n.skill, p);
      nodes.push({ id: n.skill, k: kind(n), x: 0, y: 0, name: (n.name ?? '') + ($devMode ? ` (${n.skill})` : ''), stats: n.stats, recipe: n.recipe });
    }
    for (const c of [...clusterNodes, ...compareCluster]) {
      const n = clusterNodeAsTreeNode(c);
      byId.set(c.id, n);
      at.set(c.id, { x: c.x, y: c.y });
      nodes.push({ id: c.id, k: kind(n), x: 0, y: 0, name: c.name + ($devMode ? ` (${c.id})` : ''), stats: c.stats });
    }
    return { nodes, byId, at };
  });

  let active = $state.raw<number[]>([]);
  let paths = $state.raw<PrecalculatedAllocationPaths>({});
  let dependents = $state<{ node: number; ids: number[] }>();
  let statDiff = $state<NodeStatDiff>();
  let view = $state<{ offsetX: number; offsetY: number; scaling: number; width: number; height: number }>();

  let socketed = $state.raw<exposition.TreeJewel[]>([]);
  $effect(() => {
    void $currentBuild;
    void $outputs;
    syncWrap
      ?.GetTreeSocketedJewels()
      .then((j) => (socketed = j ?? []))
      .catch(logError);
  });
  // PassiveTreeView.lua:1180-1197
  const ringSheet = $derived.by(() => {
    const sheets = loaded?.sprites.jewelRadius;
    return sheets ? Object.values(sheets)[0] : undefined;
  });
  let socketArt = $state.raw<Map<number, string>>(new Map());
  $effect(() => {
    void $currentBuild;
    void $outputs;
    syncWrap
      ?.GetTreeJewelOverlays()
      .then((o) => (socketArt = new Map((o ?? []).map((x) => [x.nodeId, x.art]))))
      .catch(logError);
  });
  // PassiveTreeView.lua:1213-1234, 977-1003
  let hoverRadii = $state.raw<{ node: number; radii: { inner: number; outer: number; col: string; nodes: number[] }[] }>();
  let socketHover: number | undefined;
  const hoverSocket = (n: Node | undefined) => {
    const id = n?.isJewelSocket ? n.skill : undefined;
    socketHover = id;
    if (id === undefined) {
      hoverRadii = undefined;
      return;
    }
    syncWrap
      ?.GetTreeSocketRadii(id)
      .then((radii) => {
        if (socketHover === id) hoverRadii = radii?.length ? { node: id, radii } : undefined;
      })
      .catch(logError);
  };
  const colour = (col: string) => '#' + col.replace(/^\^x/, '');
  const radiusColour = $derived.by(() => {
    const out = new Map<number, string>();
    for (const r of hoverRadii?.radii ?? []) for (const id of r.nodes) if (!out.has(id)) out.set(id, colour(r.col));
    return out;
  });

  let socketing = $state<{ node: TreeNodeFx; choices: exposition.TreeJewel[]; current: number } | null>(null);
  let socketOpen = $state(false);
  const chooseJewel = async (itemId: number) => {
    const s = socketing;
    socketOpen = false;
    if (!s) return;
    const err = await syncWrap?.SocketTreeJewel(s.node.id, itemId);
    if (err) logError(new Error(err));
    currentBuild.set($currentBuild);
  };

  // TreeTab.lua:1038-1159
  let powerOpen = $state(false);
  // TreeTab.lua:186-189
  let timelessOpen = $state(false);
  // TreeTab.lua:194-205
  let heatOn = $state(false);
  let powerStat = $state('Life');
  let powerFilter = $state<PowerFilter>('unallocated');
  // PowerReportListControl.lua:36, 41
  let powerSort = $state<PowerSort>('power');
  let powerMasteries = $state(true);
  // TreeTab.lua:208-248
  let depthChoice = $state(0);
  let powerCustom = $state('0');
  const powerDepth = $derived(depthChoice === -1 ? Number.parseInt(powerCustom, 10) || 0 : depthChoice);
  let powerRows = $state.raw<PowerRow[]>();
  let powerMs = $state<number>();
  let powerProgress = $state(0);
  let powerFocus = $state<number>();
  let powerStarted = $state(false);
  const powerFlashes = new PowerFlashes();
  let powerPartial = $state.raw<PowerRow[]>();
  let q = $state('');
  const powerStats = $derived(powerOpen || heatOn ? (syncWrap?.GetPowerStats() ?? Promise.resolve([])) : Promise.resolve([]));
  // TreeTab.lua:260-265
  const nodePowerStats = $derived(heatOn ? (syncWrap?.GetNodePowerStats() ?? Promise.resolve([])) : Promise.resolve([]));
  let powerAsked: { stat: string; depth: number; out: unknown } | undefined;
  $effect(() => {
    if ((!powerOpen && !heatOn) || !powerStarted) {
      powerAsked = undefined;
      return;
    }
    const out = $outputs;
    const stat = powerStat;
    const depth = powerDepth;
    const recalculated = powerAsked !== undefined && powerAsked.stat === stat && powerAsked.depth === depth && powerAsked.out !== out;
    powerAsked = { stat, depth, out };
    powerRows = undefined;
    powerProgress = 0;
    let live = true;
    let poll: ReturnType<typeof setInterval> | undefined;
    const build = () => {
      let polls = 0;
      poll = setInterval(() => {
        syncWrap
          ?.PowerReportLive(polls++ % 2 === 1)
          .then((l) => {
            if (!live) return;
            powerProgress = l.done;
            if (l.rows) {
              powerPartial = l.rows;
              powerFlashes.arrive(l.rows.map((r) => r.ID));
            }
          })
          .catch(logError);
      }, 250);
      syncWrap
        ?.BuildPowerReportLive(stat, depth)
        .then((r) => {
          if (!live || !r) return;
          powerRows = r.rows;
          powerMs = r.ms;
          powerFlashes.arrive(r.rows.filter((row) => row.Power !== 0).map((row) => row.ID));
        })
        .catch(logError)
        .finally(() => {
          clearInterval(poll);
          if (live) {
            powerPartial = undefined;
          }
        });
    };
    const settle = recalculated ? setTimeout(build, POWER_SETTLE_MS) : undefined;
    if (!recalculated) build();
    return () => {
      live = false;
      clearTimeout(settle);
      clearInterval(poll);
      powerFlashes.reset();
      powerPartial = undefined;
    };
  });
  const closePower = () => {
    powerOpen = false;
    powerStarted = false;
    void syncWrap?.CancelPowerReport();
  };
  $effect(() => () => void syncWrap?.CancelPowerReport());
  const setHeat = (on: boolean) => {
    heatOn = on;
    if (!on && powerStat === '') powerStat = 'Life';
    if (!on && !powerOpen) {
      powerStarted = false;
      void syncWrap?.CancelPowerReport();
    }
  };
  // PassiveTreeView.lua:920-933
  const ascendancyNodes = $derived(new Set(base.nodes.filter((n) => base.byId.get(n.id)?.ascendancyName).map((n) => n.id)));
  const heatRows = $derived(heatOn && powerRows ? powerRows : powerRows === undefined ? powerPartial : undefined);
  const heat = $derived(
    heatRows ? { ...heatValues(heatRows, ascendancyNodes), mix: powerStat === '' ? heatMixValues(heatRows, ascendancyNodes) : undefined } : undefined
  );
  // PowerReportListControl.lua:46
  let powerStatLabels = $state.raw<Record<string, string>>({});
  $effect(() => {
    void powerStats.then((list) => (powerStatLabels = Object.fromEntries((list ?? []).map((st) => [st.Stat, st.Label]))));
  });
  const powerLabel = $derived(powerStatLabels[powerStat] ?? powerStat);
  const shownPower = $derived(
    powerRows
      ? showPowerRows(
          powerRows,
          displayStats.find((d) => d.stat === powerStat),
          powerFilter,
          powerMasteries,
          (f, v) => formatNumSep(luaFormat(f, v)),
          powerSort
        )
      : []
  );
  const showInPower = (n: TreeNodeFx) => {
    powerOpen = true;
    powerFilter = n.a ? 'allocated' : 'unallocated';
    powerFocus = n.id;
  };
  $effect(() => {
    if (powerFocus === undefined || !shownPower.length) return;
    document.querySelector(`[data-power-node="${powerFocus}"]`)?.scrollIntoView({ block: 'nearest' });
  });

  // TreeTab.lua:841-991
  let treeTattoos = $state.raw<{ tattoos: Record<number, string>; count: number }>({ tattoos: {}, count: 0 });
  $effect(() => {
    void $currentBuild;
    void $outputs;
    syncWrap
      ?.GetTreeTattoos()
      .then((t) => (treeTattoos = t ?? { tattoos: {}, count: 0 }))
      .catch(logError);
  });
  let tattooNode = $state<TreeNodeFx>();
  // PassiveTreeView.lua:847-877
  let tattooArt = $state.raw<Map<number, { icon: string; effect: string }>>(new Map());
  $effect(() => {
    void treeTattoos;
    syncWrap
      ?.GetTreeTattooArt()
      .then((a) => (tattooArt = new Map((a ?? []).map((x) => [x.nodeId, { icon: x.icon, effect: x.effect }]))))
      .catch(logError);
  });
  // TreeTab.lua:946-973
  let tattooLines = $state.raw<{ line: string; count: number }[]>([]);
  $effect(() => {
    if (!tattooNode) return;
    void treeTattoos;
    syncWrap
      ?.GetTattooCountLines()
      .then((l) => (tattooLines = l ?? []))
      .catch(logError);
  });
  // TreeTab.lua:917
  const defaultTattoo: Record<string, number | undefined> = {};
  const resetTattoo = async (n: TreeNodeFx) => {
    tattooNode = undefined;
    defaultTattoo[n.name] = undefined;
    await syncWrap?.RemoveTattoo(n.id);
    currentBuild.set($currentBuild);
  };
  // TreeTab.lua:846-878
  let tattooChoices = $state.raw<TattooChoiceFx[]>([]);
  let tattooSel = $state(0);
  let showLegacyTattoo = $state(false);
  $effect(() => {
    const n = tattooNode;
    const legacy = showLegacyTattoo;
    if (!n) return;
    syncWrap
      ?.GetTattooChoices(n.id, legacy)
      .then((c) => {
        if (tattooNode !== n) return;
        tattooChoices = c ?? [];
        tattooSel = Math.min(defaultTattoo[n.name] ?? 0, Math.max(tattooChoices.length - 1, 0));
      })
      .catch(logError);
  });
  const addTattoo = async (n: TreeNodeFx, dn: string) => {
    tattooNode = undefined;
    defaultTattoo[n.name] = tattooSel;
    const err = await syncWrap?.AddTattoo(n.id, dn);
    if (err) throw new Error(err);
    currentBuild.set($currentBuild);
  };

  const alloc = $derived(Object.fromEntries(active.map((id) => [id, true])) as Record<number, boolean>);

  // TreeTab.lua:992-1036
  let selections = $state.raw<Record<number, number>>({});
  let masteryOpen = $state<number | null>(null);
  let options = $state.raw<{ node: number; opts: calculator.MasteryOption[] }>();
  $effect(() => {
    void $currentBuild;
    void $outputs;
    void active;
    const id = masteryOpen;
    syncWrap
      ?.GetMasterySelections()
      .then((s) => (selections = s ?? {}))
      .catch(logError);
    if (id === null) return;
    syncWrap
      ?.GetMasteryOptions(id)
      .then((opts) => {
        if (masteryOpen === id) options = { node: id, opts: opts ?? [] };
      })
      .catch(logError);
  });
  const masteryFx = (n: TreeNodeFx): Partial<TreeNodeFx> => {
    if (n.k !== 'm') return {};
    if (options?.node === n.id) {
      return { effects: options.opts.map((o) => [o.Label]), effect: options.opts.findIndex((o) => o.Selected) };
    }
    if (n.id === masteryOpen) return { effects: [], effect: -1 };
    // PassiveTreeView.lua:1584-1592
    const held = new Set(Object.entries(selections).flatMap(([node, e]) => (Number(node) === n.id ? [] : [e])));
    const shown = (base.byId.get(n.id)?.masteryEffects ?? []).filter((e) => !held.has(e.effect));
    return { effects: shown.map((e) => [e.stats.join(' / ')]), effect: shown.findIndex((e) => e.effect === selections[n.id]) };
  };
  // PassiveMasteryControl.lua:37-43, PassiveTreeView.lua:1671-1705
  let masteryDiff = $state<{ node: number; effect: number; pending: boolean; empty?: string; blocks: StatDiffBlock[] }>();
  let masteryHover: { id: number; effect: number } | undefined;
  const hoverMastery = async (id: number, index: number | null) => {
    const effect = index === null || options?.node !== id ? undefined : options.opts[index]?.Effect;
    if (effect === undefined) {
      masteryHover = undefined;
      masteryDiff = undefined;
      return;
    }
    const want = { id, effect: index! };
    masteryHover = want;
    const current = () => masteryHover === want;
    const isAlloc = !!alloc[id];
    const path = isAlloc ? [id] : (calculateAllocationPath(paths, id) ?? [id]);
    const blocks: StatDiffBlock[] = [];
    masteryDiff = { node: id, effect: index!, pending: true, blocks };
    const header = isAlloc ? 'Reallocating this node will give you:' : 'Allocating this node will give you:';
    const one = await syncWrap?.StatCompareMastery(id, effect, [id]);
    if (!current() || !one) return;
    if (one.sections.length) blocks.push({ header, sections: one.sections, readyAt: performance.now(), calcMs: one.calcMs });
    if (!isAlloc && path.length > 1) {
      masteryDiff = { node: id, effect: index!, pending: true, blocks: [...blocks] };
      const all = await syncWrap?.StatCompareMastery(id, effect, path, path.length);
      if (!current() || !all) return;
      if (all.sections.length) {
        blocks.push({
          header: 'Allocating this node and all nodes leading to it will give you:',
          sections: all.sections,
          readyAt: performance.now(),
          calcMs: all.calcMs
        });
      }
    }
    const empty = blocks.length
      ? undefined
      : `No changes from ${isAlloc ? 'unallocating' : 'allocating'} this node${!isAlloc && path.length > 1 ? ' or the nodes leading to it' : ''}.`;
    masteryDiff = { node: id, effect: index!, pending: false, blocks: [...blocks], empty };
  };
  $effect(() => {
    void $outputs;
    void masteryOpen;
    masteryHover = undefined;
    masteryDiff = undefined;
  });

  const chooseMastery = async (id: number, index: number) => {
    const effect = options?.node === id ? options.opts[index]?.Effect : undefined;
    if (effect === undefined) return;
    const path = alloc[id] ? [] : (calculateAllocationPath(paths, id) ?? [id]);
    const err = await syncWrap?.SelectMasteryEffect(id, effect, path);
    if (err) logError(new Error(err));
    currentBuild.set($currentBuild);
  };

  const costs = $derived(allocationCosts(paths));
  const nodes = $derived(
    base.nodes.map((n): TreeNodeFx => {
      if (alloc[n.id]) {
        const lose = dependents?.node === n.id ? dependents.ids : undefined;
        return { ...n, ...masteryFx(n), a: 1, lose: lose?.length, loseIds: lose };
      }
      const hop = paths[n.id];
      return { ...n, ...masteryFx(n), pts: costs.get(n.id), next: hop !== undefined && hop !== -1 ? hop : undefined };
    })
  );

  const ascendanciesOf = (className: string) => loaded?.classes.find((c) => c.name === className)?.ascendancies.map((a) => a.name) ?? ASCENDANCIES[className];
  const specDetails = $derived(
    treeSpecs?.specs.map(
      (s, i) => `${s.AscendClassName !== 'None' ? s.AscendClassName : s.ClassName}${specPoints[i] ? `, ${specPoints[i].Used} points` : ''}`
    ) ?? []
  );

  const tree = $derived<TreeFixture>({
    version: version ?? '',
    classes: CLASSES,
    ascendancies: ['None', ...(ascendanciesOf(info?.className ?? '') ?? [])],
    specs: treeSpecs?.specs.map((s) => s.Label) ?? [],
    activeSpec: (treeSpecs?.active ?? 1) - 1,
    aspect: 1,
    radius: 0,
    nodes,
    edges: [],
    tips: {},
    sockets: socketed.map((j) => ({ id: j.NodeID, pts: 0, notables: [], keystones: [], smalls: 0, jewel: j.Name }))
  });

  const project = $derived.by(() => {
    const v = view;
    const t = loaded;
    const at = base.at;
    return (id: number) => {
      const p = at.get(id);
      return p && v && t ? toScreen(t, p, v.offsetX, v.offsetY, v.scaling) : undefined;
    };
  });

  type PobView = { zoomLevel: number; zoomX: number; zoomY: number };
  let canvasEl = $state<{
    toggle: (node: Node) => void;
    zoom: (how: 'in' | 'out' | 'fit') => void;
    focus: (id: number) => void;
    pobView: () => PobView;
    setPobView: (v: PobView) => void;
  }>();
  const fxById = $derived(new Map(nodes.map((n) => [n.id, n])));
  const fxOf = (n: Node) => (n.skill === undefined ? undefined : fxById.get(n.skill));
  let hoveredFx: TreeNodeFx | undefined;
  const hover = (api: CanvasApi, n: Node | undefined) => {
    if (hoveredFx) api.leave(hoveredFx);
    hoveredFx = n ? fxOf(n) : undefined;
    if (hoveredFx) api.enter(hoveredFx);
    hoverSocket(n);
  };

  // TreeTab.lua:106-128, PassiveTreeView.lua:166-185
  let compareSpec = $state(-1);
  type Compared = { nodes: number[]; masteries: { node: number; effect: number; stats: string[] }[] };
  let compared = $state.raw<Compared>();
  $effect(() => {
    void $currentBuild;
    void $outputs;
    const i = compareSpec;
    if (i < 0) {
      compared = undefined;
      return;
    }
    syncWrap
      ?.GetTreeSpecCompare(i + 1)
      .then((c) => {
        if (compareSpec === i) compared = c;
      })
      .catch(logError);
  });
  $effect(() => {
    const c = compared;
    const own = new Set(clusterNodes.map((n) => n.id));
    if (!c) {
      compareCluster = [];
      return;
    }
    const theirs = new Set(c.nodes);
    syncWrap
      ?.GetTreeSpecClusterNodes(compareSpec + 1)
      .then((spec) => {
        if (compared === c) compareCluster = (spec ?? []).filter((n) => !own.has(n.id) && theirs.has(n.id));
      })
      .catch(logError);
  });
  const compareOnly = $derived(new Set(compareCluster.map((n) => n.id)));
  const comparedMastery = $derived(new Map((compared?.masteries ?? []).map((m) => [m.node, m])));
  const compareTint = $derived.by(() => {
    if (!compared) return undefined;
    const theirs = new Set(compared.nodes);
    // PassiveSpec.lua:577-580, 608-613
    const start = (id: number) => base.byId.get(id)?.classStartIndex !== undefined || !!base.byId.get(id)?.isAscendancyStart;
    const out = new Map<number, [number, number, number]>();
    for (const id of theirs) if (!alloc[id] && !start(id)) out.set(id, [0, 1, 0]);
    for (const id of active) {
      if (start(id)) continue;
      if (!theirs.has(id)) out.set(id, [1, 0, 0]);
      else if (base.byId.get(id)?.isMastery && comparedMastery.has(id) && comparedMastery.get(id)!.effect !== selections[id]) out.set(id, [0, 0, 1]);
    }
    return out;
  });

  // PassiveTreeView.lua:1157-1248
  let rings = $state.raw<{ nodeId: number; outer: number; inner: number; art: string }[]>([]);
  $effect(() => {
    void $currentBuild;
    void $outputs;
    const theirs = compared?.nodes ?? [];
    syncWrap
      ?.GetTreeJewelRings(theirs)
      .then((r) => (rings = r ?? []))
      .catch(logError);
  });
  // PassiveTreeView.lua:814-816
  let splitPath = $state.raw<number[]>([]);
  $effect(() => {
    void $currentBuild;
    void $outputs;
    syncWrap
      ?.GetSplitPersonalityPath()
      .then((p) => (splitPath = p ?? []))
      .catch(logError);
  });
  const SPLIT_PERSONALITY: [number, number, number] = [0xff / 255, 0xd6 / 255, 0x2a / 255];
  const nodeTint = $derived.by(() => {
    const out = new Map<number, [number, number, number]>(splitPath.map((id) => [id, SPLIT_PERSONALITY]));
    for (const [id, c] of compareTint ?? []) out.set(id, c);
    return out.size ? out : undefined;
  });

  // PassiveTreeView.lua:96-98, 108
  let showDiff = $state(true);
  // ItemsTab.lua:203-205
  let manageTrees = $state(false);
  $effect(() => {
    if (!$treeManageRequest) return;
    treeManageRequest.set(false);
    manageTrees = true;
  });
  // PassiveTreeView.lua:83-111
  let viewRequested: unknown;
  let viewRestored: unknown;
  let savedView: pob.TreeView | undefined;
  $effect(() => {
    const b = $currentBuild;
    const el = canvasEl;
    if (!b || !el || !view || viewRequested === b) return;
    viewRequested = b;
    syncWrap
      ?.GetTreeViewState()
      .then((t) => {
        if (!t || viewRequested !== b) return;
        savedView = t;
        viewRestored = b;
        if (t.ShowHeatMap) setHeat(true);
        showDiff = t.ShowStatDifferences ?? true;
        if (t.ZoomLevel || t.ZoomX || t.ZoomY) el.setPobView({ zoomLevel: t.ZoomLevel, zoomX: t.ZoomX, zoomY: t.ZoomY });
        else {
          const p = el.pobView();
          savedView = { ...t, ZoomLevel: p.zoomLevel, ZoomX: p.zoomX, ZoomY: p.zoomY };
        }
      })
      .catch(logError);
  });
  let searchRestored: unknown;
  $effect(() => {
    const b = $currentBuild;
    if (!b || searchRestored === b) return;
    searchRestored = b;
    const typed = untrack(() => q);
    syncWrap
      ?.GetTreeViewState()
      .then((t) => {
        if (t && q === typed) q = t.SearchStr ?? '';
      })
      .catch(logError);
  });
  let viewSave: ReturnType<typeof setTimeout> | undefined;
  $effect(() => {
    void view;
    const search = q;
    const heatMap = heatOn;
    const diffs = showDiff;
    const el = canvasEl;
    if (!el || viewRestored !== $currentBuild) return;
    clearTimeout(viewSave);
    viewSave = setTimeout(() => {
      const p = el.pobView();
      const was = savedView;
      const same =
        was &&
        was.ZoomLevel === p.zoomLevel &&
        Math.abs(was.ZoomX - p.zoomX) < 1 &&
        Math.abs(was.ZoomY - p.zoomY) < 1 &&
        (was.SearchStr ?? '') === search &&
        !!was.ShowHeatMap === heatMap &&
        (was.ShowStatDifferences ?? true) === diffs;
      if (same) return;
      const next: pob.TreeView = {
        ZoomLevel: p.zoomLevel,
        ZoomX: p.zoomX,
        ZoomY: p.zoomY,
        SearchStr: search,
        ShowHeatMap: heatMap || was?.ShowHeatMap !== undefined ? heatMap : undefined,
        ShowStatDifferences: diffs
      };
      savedView = next;
      // TreeTab.lua:180, Build.lua:1303
      const searched = (was?.SearchStr ?? '') !== search;
      void syncWrap
        ?.SetTreeViewState(next)
        .then(() => {
          if (searched) currentBuild.set($currentBuild);
        })
        .catch(logError);
    }, 500);
  });

  // PassiveTreeView.lua:1478-1513, 1601-1644, 1712-1746
  let tipExtra = $state.raw<{ id: number; skills: calculator.TooltipLine[][]; jewel?: calculator.ItemTooltip; distance?: number }>();
  let tipNode = $state<number | null>(null);
  $effect(() => {
    const id = tipNode;
    if (id === null) return;
    // PassiveTreeView.lua:1479
    const isSocket = !!base.byId.get(id)?.isJewelSocket;
    const socket = isSocket && socketed.some((j) => j.NodeID === id);
    Promise.all([
      syncWrap?.GetNodeSkillTooltips(id),
      socket ? syncWrap?.SocketJewelTooltip(id) : undefined,
      isSocket && untrack(() => alloc[id]) ? syncWrap?.GetSocketDistance(id) : undefined
    ])
      .then(([skills, jewel, distance]) => {
        if (tipNode === id) tipExtra = { id, skills: skills ?? [], jewel, distance };
      })
      .catch(logError);
  });
  // PassiveTreeView.lua:1719
  let leapHeld = $state.raw(new Set<number>());
  $effect(() => {
    void $outputs;
    syncWrap
      ?.GetLeapHeldNodes()
      .then((ids) => (leapHeld = new Set(ids ?? [])))
      .catch(logError);
  });
  // PassiveTreeView.lua:1715-1716
  let tracePath = $state.raw<number[]>([]);
  let gold = $state(0);
  $effect(() => {
    void $outputs;
    syncWrap
      ?.GetGoldRespecPrice()
      .then((g) => (gold = g ?? 0))
      .catch(logError);
  });

  const copy = (n: TreeNodeFx) => {
    navigator.clipboard?.writeText([n.name, ...(n.stats ?? [])].join('\n')).catch(logError);
  };
</script>

{#snippet nodeTipBody(n: TreeNodeFx)}
  {@const node = base.byId.get(n.id)}
  {@const theirs = comparedMastery.get(n.id)}
  {#if compared && n.k === 'm' && theirs && theirs.effect !== selections[n.id]}
    <!-- PassiveTreeView.lua:1550-1576 -->
    <div class="flex flex-col gap-0.5 text-12" aria-label="Compared mastery">
      {#each n.effect !== undefined && n.effect >= 0 ? (n.effects?.[n.effect] ?? []) : [] as l (l)}<span class="text-[#ff7f7f]">&lt;- {l}</span>{/each}
      {#each theirs.stats as l (l)}<span class="text-[#7fff7f]">-&gt; {l}</span>{/each}
    </div>
  {/if}
  {#if tipExtra?.id === n.id && tipExtra.jewel}
    <div class="border-t border-line pt-2" aria-label="Socketed jewel">
      <TooltipLines
        lines={[...(tipExtra.jewel.Title ?? []).map((t) => ({ Text: t, Separator: false })), { Text: '', Separator: true }, ...(tipExtra.jewel.Lines ?? [])]} />
    </div>
  {/if}
  {#if n.k === 'j' && alloc[n.id]}
    <!-- PassiveTreeView.lua:1490-1497 -->
    {#if compared?.nodes.includes(n.id) && !socketed.some((j) => j.NodeID === n.id)}
      <div class="border-t border-line pt-2 text-12" aria-label="Compared socket">
        <span class="text-[#ff9922]">Compared build:</span> (empty socket)
      </div>
    {/if}
    {#if tipExtra?.id === n.id && tipExtra.distance}
      <div class="border-t border-line pt-2 text-12" aria-label="Distance to start">Distance to start: {tipExtra.distance}</div>
    {/if}
  {/if}
  {#if tipExtra?.id === n.id}
    {#each tipExtra.skills as lines, i (i)}
      <div class="border-t border-line pt-2" aria-label="Granted skill"><TooltipLines {lines} center /></div>
    {/each}
  {/if}
  {#if node?.reminderText?.length}
    <div class="flex flex-col gap-0.5 border-t border-line pt-2 text-12 text-[#a0a080]" aria-label="Reminder text">
      {#each node.reminderText as l, i (i)}<span>{l}</span>{/each}
    </div>
  {/if}
  {#if node?.flavourText?.length}
    <div class="flex flex-col gap-0.5 border-t border-line pt-2 italic text-[#af6025]" aria-label="Flavour text">
      {#each node.flavourText as l, i (i)}<span>{l}</span>{/each}
    </div>
  {/if}
  {@const editKind = tattooEditKind(n, !!treeTattoos.tattoos[n.id])}
  {#if editKind}
    <!-- PassiveTreeView.lua:1646-1656 -->
    <div class="border-t border-line pt-2 text-12 text-[#80a080]" aria-label="Tattoo tip">Tip: Right click to edit the {editKind} for this node</div>
  {/if}
  {#if compareOnly.has(n.id)}
    <!-- PassiveTreeView.lua:1804-1805 -->
    <div class="border-t border-line pt-2 text-12 text-[#70ff70]" aria-label="Compared build">This node is only in the compared build</div>
  {/if}
{/snippet}
{#snippet nodeTipCost(n: TreeNodeFx)}
  {@const node = base.byId.get(n.id)}
  {@const pathLen = alloc[n.id] ? 0 : (n.pts ?? 0)}
  {@const depends = alloc[n.id] ? (n.lose ?? 0) + 1 : 0}
  {@const cost = gold * (node?.ascendancyName ? 5 : 1)}
  {#if !compareOnly.has(n.id)}
    <div class="flex flex-col gap-0.5 border-t border-line pt-2 text-12" aria-label="Points and gold">
      {#if pathLen > 0 && tracePath.includes(n.id)}
        <span>{tracePath.length} nodes in trace path</span>
      {:else if pathLen > 0}
        <span
          >{pathLen} points to node{#if leapHeld.has(n.id)}<span class="text-text-3"> (Can be allocated without pathing to it)</span>{/if}</span>
        {#if pathLen > 1 && !leapHeld.has(n.id)}<span class="text-text-3"
            >Tip: To reach this node by a different path, hold Shift, then trace the path and click this node</span
          >{/if}
      {/if}
      {#if depends > 1}
        <span>{depends} points gained from unallocating these nodes</span>
        <span><span class="text-[#ffd700]">{formatNumSep(String(depends * cost))} Gold</span> required to unallocate these nodes</span>
      {:else if depends === 1}
        <span><span class="text-[#ffd700]">{formatNumSep(String(cost))} Gold</span> required to unallocate this node</span>
      {/if}
      {#if n.k !== 'j'}<span class="text-text-3">Tip: Press Ctrl+C to copy this node's text.</span>
      {:else if alloc[n.id]}
        <!-- PassiveTreeView.lua:1499-1501 -->
        <span class="text-text-3">Tip: Right click this socket to choose the jewel for this socket.</span>{/if}
    </div>
  {/if}
{/snippet}

<TreePage
  {tree}
  bind:showDiff
  className={info?.className ?? ''}
  ascendancy={info?.ascendClassName ?? 'None'}
  {progress}
  {alloc}
  {project}
  level={info?.level}
  recordedVersion={info?.recordedTreeVersion}
  whole
  diff={statDiff}
  hint="Click: allocate / deallocate · Right-click: more · Drag: pan · Wheel: zoom"
  onclass={(c: string) => void setClass(c).catch(logError)}
  onascendancy={(a: string) => void setAscendancy(a).catch(logError)}
  {secondary}
  onsecondary={(id: number) => void setSecondary(id).catch(logError)}
  onlevel={(l: number) => void setLevel(l).catch(logError)}
  {levelAuto}
  onlevelauto={(a: boolean) => void setLevelAuto(a).catch(logError)}
  bind:mastery={masteryOpen}
  onmastery={(id: number, i: number) => void chooseMastery(id, i).catch(logError)}
  onmasteryhover={(id: number, i: number | null) => void hoverMastery(id, i).catch(logError)}
  {masteryDiff}
  onpick={(n: TreeNodeFx) => {
    const node = base.byId.get(n.id);
    if (node) canvasEl?.toggle(node);
  }}
  onmenuaction={(action: string, n: TreeNodeFx) => {
    if (action === 'copy') copy(n);
    // PassiveTreeView.lua:511-519
    if (action === 'socket') void gotoPage(`${appBase}/items?socket=${n.id}`).catch(logError);
    if (action === 'power') showInPower(n);
    if (action === 'tattoo') tattooNode = n;
  }}
  onzoom={(how: 'in' | 'out' | 'fit') => canvasEl?.zoom(how)}
  spec={tree.activeSpec}
  {specDetails}
  onspec={(i: number) => void setSpec(i).catch(logError)}
  onspecs={(op: SpecOp) => void editSpecs(op).catch(logError)}
  onspecsopen={readPoints}
  bind:manage={manageTrees}
  onspechover={(i: number | null) => void hoverSpec(i).catch(logError)}
  versions={versionOptions}
  specVersion={info?.recordedTreeVersion}
  onconvert={convertTree}
  onconvertall={convertAllTrees}
  onexporttree={async () => (await syncWrap?.ExportTreeLink()) ?? ''}
  onimporttree={async (title: string, link: string) => {
    editSeq++;
    const msg = (await syncWrap?.ImportTreeLink(title, link)) ?? '';
    if (!msg) currentBuild.set($currentBuild);
    if (managing) readPoints();
    return msg;
  }}
  {specTip}
  menuOffFor={(action: string, n: TreeNodeFx) =>
    action !== 'tattoo' || treeTattoos.tattoos[n.id]
      ? undefined
      : n.k === 'm' && n.a
        ? 'An allocated mastery takes an effect, not a runegraft'
        : tattooEditKind(n, false)
          ? undefined
          : 'Only attribute passives, +30 attribute notables, keystones and masteries take a tattoo'}
  bind:q
  onpowerreport={() => (powerOpen ? closePower() : (powerOpen = true))}
  detailsOpen={powerOpen}
  ontimeless={() => (timelessOpen = true)}
  {compareSpec}
  oncompare={(i: number) => (compareSpec = i)}
  onhovered={(id: number | null) => (tipNode = id)}
  edgeMarkers={$edgeSearchHighlight}
  nodePower={heatOn}
  onnodepower={(on: boolean) => {
    setHeat(on);
    if (on) powerOpen = true;
  }}
  nodePowerBar={false}
  onresettree={() =>
    void syncWrap
      ?.ResetTreeNodes()
      .then(() => currentBuild.set($currentBuild))
      .catch(logError)}
  onremovetattoos={() =>
    void syncWrap
      ?.RemoveAllTreeTattoos()
      .then(() => currentBuild.set($currentBuild))
      .catch(logError)}>
  {#snippet details()}
    {#if powerOpen}
      <section aria-label="Power report" class="flex max-h-[45%] min-h-0 flex-col gap-1.5 border-b border-line px-4 py-3">
        <div class="flex items-baseline justify-between">
          <h2 class="text-14 font-bold">Power report</h2>
          <button class="btn-close" aria-label="Close power report" onclick={closePower}>×</button>
        </div>
        <div class="flex items-center gap-1.5">
          {#await heatOn ? nodePowerStats : powerStats then stats}
            <select class="select h-7.5 grow text-12" aria-label="Power stat" bind:value={powerStat}
              >{#each stats as s (s.Stat)}<option value={s.Stat}>{s.Label}</option>{/each}</select>
          {/await}
          <select
            class="select h-7.5 text-12"
            aria-label="Power report depth"
            title="Limit of node distance to search (lower = faster)"
            bind:value={depthChoice}
            >{#each [0, 5, 10, 15, -1] as d (d)}<option value={d}>{d === 0 ? 'All' : d === -1 ? 'Custom' : d}</option>{/each}</select>
          <select class="select h-7.5 text-12" aria-label="Power report filter" bind:value={powerFilter} onchange={() => (powerSort = 'power')}
            ><option value="unallocated">Show Unallocated</option><option value="clusters">Show Unallocated & Clusters</option><option value="allocated"
              >Show Allocated</option
            ></select>
          <label class="field field-inline gap-1 text-11"
            ><input class="check" type="checkbox" bind:checked={powerMasteries} onchange={() => (powerSort = 'power')} /><span>Show Masteries</span></label>
        </div>
        {#if depthChoice === -1}
          <input
            class="input h-7.5 w-17.5 font-mono text-12"
            inputmode="numeric"
            aria-label="Custom node power depth"
            value={powerCustom}
            onchange={(e) => (powerCustom = e.currentTarget.value.replace(/\D/g, ''))} />
        {/if}
        <div class="flex items-center gap-2">
          <label class="field field-inline gap-1 text-12" title="An estimate of each unallocated passive's power for the chosen stat"
            ><input class="check" type="checkbox" checked={heatOn} onchange={(e) => setHeat(e.currentTarget.checked)} /><span>Show node power</span></label>
          {#if !powerStarted}
            <button class="btn btn-sm ml-auto px-3" onclick={() => (powerStarted = true)}>Start</button>
          {/if}
        </div>
        {#if !powerStarted}
          <span class="note">Choose the stat and depth, then Start: the report calculates every node for it.</span>
        {:else if !powerRows}
          <div
            role="progressbar"
            aria-label="Building the power report"
            aria-valuemin="0"
            aria-valuemax="100"
            aria-valuenow={Math.round(powerProgress * 100)}
            class="note">
            Building Power Report... ({Math.round(powerProgress * 100)}%)
          </div>
        {:else}
          <span class="note" data-power-ms={powerMs !== undefined ? Math.round(powerMs) : undefined}
            >{shownPower.length} nodes · Click to focus node on tree</span>
          <div role="table" aria-label="Node power" class="scroll flex min-h-0 flex-col text-12">
            <div role="row" class="grid grid-cols-[56px_minmax(0,1fr)_64px_44px_64px] gap-1.5 border-b border-line pb-0.5 text-11 text-text-3">
              {#each [['type', 'Type'], ['name', 'Node Name'], ['power', powerLabel], ['points', 'Points'], ['perPoint', 'pp']] as [col, label] (col)}
                <span role="columnheader" class={col === 'type' || col === 'name' ? '' : 'text-right'} aria-sort={col === powerSort ? 'descending' : undefined}
                  >{#if col === 'name'}{label}{:else}<button
                      class="border-0 bg-transparent p-0 font-sans text-11 text-text-3 hover:text-text aria-pressed:text-text"
                      aria-pressed={col === powerSort}
                      onclick={() => (powerSort = col as PowerSort)}>{label}</button
                    >{/if}</span>
              {/each}
            </div>
            {#each shownPower as r (`${r.ID}:${r.Name}`)}
              <button
                role="row"
                class={[
                  'grid grid-cols-[56px_minmax(0,1fr)_64px_44px_64px] gap-1.5 border-0 bg-transparent py-0.5 text-left font-sans text-12 text-text hover:bg-surface-3',
                  r.ID === powerFocus && 'bg-surface-3'
                ]}
                data-power-node={r.ID}
                title={r.Stats?.join('\n')}
                onclick={() => {
                  powerFocus = r.ID;
                  if (!r.Cluster) canvasEl?.focus(r.ID);
                }}
                ><span role="cell" class="truncate text-text-2">{r.Type}</span><span role="cell" class="truncate">{r.Name}</span><span
                  role="cell"
                  class={['text-right font-mono', r.dir === 'up' ? 'text-better' : r.dir === 'down' ? 'text-worse' : '']}>{r.power}</span
                ><span role="cell" class="text-right font-mono">{r.points}</span><span role="cell" class="text-right font-mono">{r.perPoint}</span></button>
            {/each}
          </div>
        {/if}
      </section>
    {/if}
  {/snippet}
  {#snippet nodeTipExtra(n: TreeNodeFx)}
    {@render nodeTipBody(n)}
  {/snippet}
  {#snippet nodeTipTail(n: TreeNodeFx)}
    {@render nodeTipCost(n)}
  {/snippet}
  {#snippet canvas(api: CanvasApi)}
    <div class="absolute inset-0">
      {#if view}
        <div class="pointer-events-none absolute inset-0 z-2 overflow-hidden" aria-hidden="true">
          {#each rings as r, i (`${r.nodeId}:${i}`)}
            {@const p = project(r.nodeId)}
            {#if p}
              <JewelRing
                nodeId={r.nodeId}
                x={p.x}
                y={p.y}
                outer={r.outer / view.scaling}
                inner={r.inner / view.scaling}
                art={r.art}
                keystone={!!base.byId.get(r.nodeId)?.isKeystone}
                sheet={ringSheet} />
            {/if}
          {/each}
          {#if hoverRadii}
            {@const c = project(hoverRadii.node)}
            {#if c}
              {#each hoverRadii.radii as r, i (i)}
                {#each r.inner ? [r.outer, r.inner] : [r.outer] as size, k (k)}
                  <div
                    class="absolute rounded-full border-2"
                    data-socket-radius={r.outer}
                    style:left="{c.x - size / view.scaling}px"
                    style:top="{c.y - size / view.scaling}px"
                    style:width="{(2 * size) / view.scaling}px"
                    style:height="{(2 * size) / view.scaling}px"
                    style:border-color={colour(r.col)}>
                  </div>
                {/each}
              {/each}
              {#each [...radiusColour] as [id, col] (id)}
                {@const p = project(id)}
                {#if p}
                  <div
                    class="absolute size-5 -translate-x-1/2 -translate-y-1/2 rounded-full border-2"
                    data-in-radius={id}
                    style:left="{p.x}px"
                    style:top="{p.y}px"
                    style:border-color={col}>
                  </div>
                {/if}
              {/each}
            {/if}
          {/if}
          {#each [...powerFlashes.flashing] as [id, k] (id + ':' + k)}
            {@const p = project(id)}
            {#if p}
              <div
                class="absolute size-8 -translate-x-1/2 -translate-y-1/2 animate-ping rounded-full border-2 border-accent [animation-iteration-count:1]"
                data-power-flash={id}
                style:left="{p.x}px"
                style:top="{p.y}px">
              </div>
            {/if}
          {/each}
          <!-- PassiveTreeView.lua:1010-1015 -->
          {#each [...api.matches] as id (id)}
            {@const p = project(id)}
            {#if p && p.x > -40 && p.y > -40 && p.x < view.width + 40 && p.y < view.height + 40}
              <div
                class="absolute -translate-x-1/2 -translate-y-1/2 rounded-full border-2"
                data-search-match={id}
                style:left="{p.x}px"
                style:top="{p.y}px"
                style:width="{Math.max(20, 350 / view.scaling)}px"
                style:height="{Math.max(20, 350 / view.scaling)}px"
                style:border-color={hexColour($colorHighlight) ?? '#ff0000'}>
              </div>
            {/if}
          {/each}
        </div>
      {/if}
      {#if loaded && $skillTreeVersion}
        {#key $skillTreeVersion}
          <SkillTree
            bind:this={canvasEl}
            skillTree={loaded}
            skillTreeVersion={$skillTreeVersion}
            chrome={false}
            bind:view
            bind:statDiff
            bind:dependents
            onhover={(n: Node | undefined) => hover(api, n)}
            onpick={(n: Node) => {
              const fx = fxOf(n);
              if (fx) api.pick(fx);
            }}
            onmenu={(n: Node, at: Point) => {
              const fx = fxOf(n);
              if (fx) api.menu(fx, at);
            }}
            onpaths={(a: number[], p: PrecalculatedAllocationPaths) => {
              active = a;
              paths = p;
            }}
            onclassconfirm={(className: string, answer: (decision: 1 | 2) => Promise<ClassChangeResult> | undefined) => (classConfirm = { className, answer })}
            {heat}
            baseTint={nodeTint}
            compared={compared?.nodes}
            split={splitPath}
            ontrace={(t: number[]) => (tracePath = t)}
            {socketArt}
            tattoos={tattooArt}
            cluster={clusterNodes}
            {compareCluster} />
        {/key}
      {/if}
    </div>
  {/snippet}
</TreePage>

<TimelessDialog bind:open={timelessOpen} />

<Dialog bind:open={socketOpen} title={socketing ? `Socket a jewel: ${socketing.node.name}` : 'Socket a jewel'} class="w-105 p-4.5">
  {#if socketing}
    <div role="group" aria-label="Jewels for this socket" class="flex max-h-90 flex-col gap-1 overflow-y-auto">
      {#each [{ ItemID: 0, Name: 'None', Rarity: '' }, ...socketing.choices] as j (j.ItemID)}
        <button
          class="choice-card justify-between gap-2 px-2.5 py-1.5 text-left text-13 aria-pressed:border-accent aria-pressed:bg-surface-3"
          aria-pressed={j.ItemID === socketing.current}
          onclick={() => void chooseJewel(j.ItemID).catch(logError)}
          ><span class={['truncate text-(--tone)', j.Rarity ? tone(j.Rarity.toLowerCase() as Tone) : '']}>{j.Name}</span
          >{#if j.ItemID === socketing.current}<span class="font-mono text-11 text-accent">socketed</span>{/if}</button>
      {/each}
    </div>
    {#if !socketing.choices.length}<span class="note">The build has no jewel that fits this socket; add one on the Items page.</span>{/if}
  {/if}
</Dialog>

<TattooPopup
  open={tattooNode !== undefined}
  nodeName={tattooNode?.name ?? ''}
  choices={tattooChoices}
  bind:selected={tattooSel}
  tattoo={tattooNode ? treeTattoos.tattoos[tattooNode.id] : undefined}
  count={treeTattoos.count}
  countLines={tattooLines}
  bind:showLegacy={showLegacyTattoo}
  onadd={(dn: string) => tattooNode && void addTattoo(tattooNode, dn).catch(logError)}
  onreset={() => tattooNode && void resetTattoo(tattooNode).catch(logError)}
  onclose={() => (tattooNode = undefined)} />

<Dialog open={classConfirm !== undefined} onclose={() => (classConfirm = undefined)} title="Class Change" class="w-105 p-4.5">
  {#if classConfirm}
    <div class="flex flex-col gap-3 text-13">
      <p class="m-0">Changing class to {classConfirm.className} will reset your passive tree.</p>
      <p class="m-0">This can be avoided by connecting one of the {classConfirm.className} starting nodes to your tree.</p>
      <div class="flex gap-1.5">
        <Button size="sm" variant="danger" onclick={() => void answerClass(1).catch(logError)}>Continue</Button>
        <Button size="sm" onclick={() => void answerClass(2).catch(logError)}>Connect Path</Button>
        <Button size="sm" variant="ghost" onclick={() => (classConfirm = undefined)}>Cancel</Button>
      </div>
    </div>
  {/if}
</Dialog>
