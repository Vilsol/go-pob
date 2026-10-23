<script lang="ts">
  import PowerReport from '$lib/design/PowerReport.svelte';
  import NodeTipSections from '$lib/design/NodeTipSections.svelte';
  import { untrack } from 'svelte';
  import { get } from 'svelte/store';
  import { goto as gotoPage } from '$app/navigation';
  import { base as appBase } from '$app/paths';
  import TreePage from '$lib/design/TreePage.svelte';
  import { treeManageRequest } from './tree_manage';
  import type { TreeFixture, TreeNodeFx } from '$lib/design/fixtures';
  import {
    showPowerRows,
    type CanvasApi,
    type PowerFilter,
    type PowerRow,
    type PowerSort,
    type ShownPowerRow,
    type SpecOp,
    type NodeTip,
    type NodeTipCost
  } from '$lib/design/tree';
  import { displayStats } from '$lib/display/stats';
  import { formatNumSep, luaFormat } from '$lib/display/format_stat';
  import SkillTree from '$lib/components/skill-tree/SkillTree.svelte';
  import { allocationCosts, calculateAllocationPath, type PrecalculatedAllocationPaths } from '$lib/components/skill-tree/paths';
  import { clusterNodeAsTreeNode } from '$lib/components/skill-tree/scene';
  import JewelRing from '$lib/components/skill-tree/JewelRing.svelte';
  import { drawnNodes, loadSkillTree, skillTree, skillTreeVersion } from '$lib/skill_tree';
  import { ascendancyOffsets, drawnByPoB, nodeCentre, searchRingSize, toScreen, type Point } from '$lib/skill_tree/geometry';
  import type { Node, TreeClusterNode } from '$lib/skill_tree/types';
  import type { NodeStatDiff, StatDiffBlock } from '$lib/skill_tree/stat_compare';
  import { estimatePlayerProgress } from '$lib/display/points';
  import { currentBuild, devMode, outputs } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { powerWorkers } from '$lib/go/power_workers';
  import { logError } from '$lib/utils';
  import Dialog from '$lib/design/Dialog.svelte';
  import Button from '$lib/design/Button.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import TattooPopup from '$lib/design/TattooPopup.svelte';
  import { tattooEditKind, type TattooChoiceFx } from '$lib/skill_tree/tattoo';
  import { tone, type Tone } from '$lib/design/tone';
  import type { calculator, exposition, pob } from '$lib/types';
  import type { ClassChangeResult } from '$lib/skill_tree/class_change';
  import { heatMixValues, heatValues } from '$lib/skill_tree/heat';
  import TimelessDialog from './TimelessDialog.svelte';
  import { PowerFlashes, powerRingSize } from './power_flash';
  import { resolvePoeUrl, shrinkTreeLink } from './link_import';
  import { colorHighlight, edgeSearchHighlight, hexColour } from '$lib/skill_tree/options';
  import { phoneMedia } from '$lib/design/mobile/mobile';
  import UndoToast from '$lib/design/mobile/UndoToast.svelte';

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
    syncWrap?.PowerPoolWarm($powerWorkers).catch(logError);
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
  // TreeTab.lua:1116, PassiveTreeView.lua:907-933
  let powerAscendancy = $state.raw<PowerRow[]>([]);
  let powerMs = $state<number>();
  let powerProgress = $state(0);
  let powerFocus = $state<number>();
  let powerHover = $state<number>();
  let powerStarted = $state(false);
  const powerFlashes = new PowerFlashes();
  let powerCurrent = $state.raw<number[]>([]);
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
            powerCurrent = l.current;
            if (l.rows) {
              powerPartial = l.rows;
              powerFlashes.arrive(l.rows.map((r) => r.ID));
            }
          })
          .catch(logError);
      }, 250);
      syncWrap
        ?.BuildPowerReportLive(stat, depth, get(powerWorkers))
        .then((r) => {
          if (!live || !r) return;
          powerRows = r.rows;
          powerAscendancy = r.ascendancy ?? [];
          powerMs = r.ms;
          powerFlashes.arrive(r.rows.filter((row) => row.Power !== 0).map((row) => row.ID));
        })
        .catch(logError)
        .finally(() => {
          clearInterval(poll);
          if (live) {
            powerPartial = undefined;
            powerCurrent = [];
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
      powerCurrent = [];
    };
  });
  const closePower = () => {
    powerOpen = false;
    powerStarted = false;
    void syncWrap?.CancelPowerReport();
  };
  $effect(() => () => void syncWrap?.CancelPowerReport());
  // TreeTab.lua:310-313
  const stopPower = () => {
    powerStarted = false;
    void syncWrap?.CancelPowerReport();
  };
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
  const heatRows = $derived(heatOn && powerRows ? [...powerRows, ...powerAscendancy] : powerRows === undefined ? powerPartial : undefined);
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
  // PassiveSpec.lua:1614-1635
  let tattooTips = $state.raw<Map<number, { name: string; stats: string[]; reminder: string[] }>>(new Map());
  $effect(() => {
    void treeTattoos;
    syncWrap
      ?.GetTreeTattooNodes()
      .then((t) => (tattooTips = new Map((t ?? []).map((x) => [x.nodeId, x]))))
      .catch(logError);
  });
  // PassiveSpec.lua:1185-1372, 2380-2388
  let conqueredTips = $state.raw<Map<number, { name: string; stats: string[]; replaced: boolean }>>(new Map());
  $effect(() => {
    void $outputs;
    syncWrap
      ?.GetTreeConqueredNodes()
      .then((c) => (conqueredTips = new Map((c ?? []).map((x) => [x.nodeId, x]))))
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
  const powerPick = $derived(
    new Map((powerOpen || heatOn ? (powerRows ?? []) : []).filter((r) => r.Picked && r.Effect !== undefined).map((r) => [r.ID, r.Effect!]))
  );
  const masteryFx = (n: TreeNodeFx): Partial<TreeNodeFx> => {
    if (n.k !== 'm') return {};
    const pick = powerPick.get(n.id);
    if (options?.node === n.id) {
      return {
        effects: options.opts.map((o) => [o.Label]),
        effect: options.opts.findIndex((o) => o.Selected),
        picked: pick === undefined ? undefined : options.opts.findIndex((o) => o.Effect === pick)
      };
    }
    if (n.id === masteryOpen) return { effects: [], effect: -1 };
    // PassiveTreeView.lua:1584-1592
    const held = new Set(Object.entries(selections).flatMap(([node, e]) => (Number(node) === n.id ? [] : [e])));
    const shown = (base.byId.get(n.id)?.masteryEffects ?? []).filter((e) => !held.has(e.effect));
    return {
      effects: shown.map((e) => [e.stats.join(' / ')]),
      effect: shown.findIndex((e) => e.effect === selections[n.id]),
      picked: pick === undefined ? undefined : shown.findIndex((e) => e.effect === pick)
    };
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
    const empty = blocks.length ? undefined : `No changes from ${isAlloc ? 'unallocating' : 'allocating'} this node.`;
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
    base.nodes.map((node): TreeNodeFx => {
      const tattoo = conqueredTips.get(node.id) ?? tattooTips.get(node.id);
      const n = tattoo ? { ...node, name: tattoo.name + ($devMode ? ` (${node.id})` : ''), stats: tattoo.stats } : node;
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
    fitActive: () => void;
  }>();
  const fxById = $derived(new Map(nodes.map((n) => [n.id, n])));
  const fxOf = (n: Node) => (n.skill === undefined ? undefined : fxById.get(n.skill));
  let hoveredFx: TreeNodeFx | undefined;
  // phone: a double tap allocates straight away, so it offers an undo
  const phone = phoneMedia();
  let undoToast = $state<string>();
  let undoTimer: ReturnType<typeof setTimeout> | undefined;
  const offerUndo = (fx: TreeNodeFx) => {
    clearTimeout(undoTimer);
    undoToast = `${alloc[fx.id] ? 'Unallocated' : 'Allocated'} ${fx.name}`;
    undoTimer = setTimeout(() => (undoToast = undefined), 5000);
  };
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
  let inRadiusOf = $state.raw(new Map<number, number[]>());
  $effect(() => {
    const own = rings.filter((r) => socketed.some((j) => j.NodeID === r.nodeId));
    Promise.all(
      own.map(async (r) => ({ socket: r.nodeId, nodes: ((await syncWrap?.GetTreeSocketRadii(r.nodeId)) ?? []).find((x) => x.outer === r.outer)?.nodes ?? [] }))
    )
      .then((list) => {
        const out = new Map<number, number[]>();
        for (const { socket, nodes: ids } of list) for (const id of ids) if (id !== socket) out.set(id, [...(out.get(id) ?? []), socket]);
        inRadiusOf = out;
      })
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
        if (phone.current) el.fitActive();
        else if (t.ZoomLevel || t.ZoomX || t.ZoomY) el.setPobView({ zoomLevel: t.ZoomLevel, zoomX: t.ZoomX, zoomY: t.ZoomY });
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
    // restored when the build opens, not once the canvas is up: a Ctrl+D pressed before it arrives wins over it
    const shown = untrack(() => showDiff);
    syncWrap
      ?.GetTreeViewState()
      .then((t) => {
        if (t && q === typed) q = t.SearchStr ?? '';
        if (t && showDiff === shown) showDiff = t.ShowStatDifferences ?? true;
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
      const was = savedView;
      // phone: the fitted view stays off the build's saved view
      const p = phone.current && was ? { zoomLevel: was.ZoomLevel, zoomX: was.ZoomX, zoomY: was.ZoomY } : el.pobView();
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
  let tipExtra = $state.raw<{
    id: number;
    skills: calculator.TooltipLine[][];
    jewel?: calculator.ItemTooltip;
    distance?: number;
    affecting?: { socket: number; title: string[]; lines: calculator.TooltipLine[] }[];
  }>();
  let tipNode = $state<number | null>(null);
  $effect(() => {
    const id = tipNode;
    if (id === null) return;
    // PassiveTreeView.lua:1479
    const isSocket = !!base.byId.get(id)?.isJewelSocket;
    const socket = isSocket && socketed.some((j) => j.NodeID === id);
    // PassiveTreeView.lua:1755-1762
    const comparedSocket = untrack(() => compareCluster.find((c) => c.id === id)?.type === 'Socket');
    const reaching = inRadiusOf.get(id) ?? [];
    Promise.all([
      syncWrap?.GetNodeSkillTooltips(id),
      socket ? syncWrap?.SocketJewelTooltip(id) : comparedSocket ? syncWrap?.CompareSpecJewelTooltip(untrack(() => compareSpec) + 1, id) : undefined,
      isSocket && untrack(() => alloc[id]) ? syncWrap?.GetSocketDistance(id) : undefined,
      Promise.all(reaching.map(async (s) => ({ socket: s, tip: await syncWrap?.SocketJewelTooltip(s) })))
    ])
      .then(([skills, jewel, distance, reach]) => {
        const affecting = reach.flatMap(({ socket: s, tip }) =>
          tip ? [{ socket: s, title: tip.Title ?? [], lines: (tip.Lines ?? []).filter((l) => !l.Separator && /radius|ring/i.test(l.Text)) }] : []
        );
        if (tipNode === id) tipExtra = { id, skills: skills ?? [], jewel, distance, affecting };
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

  // PassiveTreeView.lua:1478-1513, 1550-1576, 1601-1656, 1712-1746, 1804-1805
  const nodeTip = (n: TreeNodeFx): NodeTip => {
    const node = base.byId.get(n.id);
    const theirs = comparedMastery.get(n.id);
    const extra = tipExtra?.id === n.id ? tipExtra : undefined;
    const socket = n.k === 'j' && !!alloc[n.id];
    return {
      pick: n.k === 'm' && n.picked !== undefined && n.effects?.[n.picked] ? { index: n.picked, text: n.effects[n.picked].join(' / ') } : undefined,
      comparedMastery:
        compared && n.k === 'm' && theirs && theirs.effect !== selections[n.id]
          ? { remove: n.effect !== undefined && n.effect >= 0 ? (n.effects?.[n.effect] ?? []) : [], add: theirs.stats }
          : undefined,
      jewel: extra?.jewel
        ? [...(extra.jewel.Title ?? []).map((t) => ({ Text: t, Separator: false })), { Text: '', Separator: true }, ...(extra.jewel.Lines ?? [])]
        : undefined,
      comparedEmptySocket: socket && !!compared?.nodes.includes(n.id) && !socketed.some((j) => j.NodeID === n.id),
      distance: socket ? extra?.distance : undefined,
      affecting: extra?.affecting?.map((a) => ({ socket: a.socket, title: a.title[0] ?? '', lines: a.lines })),
      skills: extra?.skills,
      reminder: tattooTips.has(n.id) ? tattooTips.get(n.id)!.reminder : conqueredTips.get(n.id)?.replaced ? [] : node?.reminderText,
      flavour: node?.flavourText,
      tattooKind: tattooEditKind(n, !!treeTattoos.tattoos[n.id]) || undefined,
      compareOnly: compareOnly.has(n.id)
    };
  };
  const nodeCost = (n: TreeNodeFx): NodeTipCost | undefined => {
    if (compareOnly.has(n.id)) return undefined;
    const path = alloc[n.id] ? 0 : (n.pts ?? 0);
    const depends = alloc[n.id] ? (n.lose ?? 0) + 1 : 0;
    const cost = gold * (base.byId.get(n.id)?.ascendancyName ? 5 : 1);
    return {
      traced: path > 0 && tracePath.includes(n.id) ? tracePath.length : undefined,
      path,
      leap: leapHeld.has(n.id),
      depends,
      gold: formatNumSep(String(cost)),
      dependsGold: formatNumSep(String(depends * cost)),
      socket: n.k === 'j',
      allocated: !!alloc[n.id]
    };
  };

  const copy = (n: TreeNodeFx) => {
    navigator.clipboard?.writeText([n.name, ...(n.stats ?? [])].join('\n')).catch(logError);
  };
</script>

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
  onshrinktree={shrinkTreeLink}
  onimporttree={async (title: string, link: string) => {
    // TreeTab.lua:759-789
    let resolved = link;
    if (/poeurl\.com\//.test(link)) {
      try {
        resolved = await resolvePoeUrl(link);
      } catch (err) {
        return err instanceof Error ? err.message : String(err);
      }
    }
    editSeq++;
    const msg = (await syncWrap?.ImportTreeLink(title, resolved)) ?? '';
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
      <PowerReport
        stats={heatOn ? nodePowerStats : powerStats}
        bind:stat={powerStat}
        bind:depth={depthChoice}
        bind:custom={powerCustom}
        bind:filter={powerFilter}
        bind:masteries={powerMasteries}
        bind:sort={powerSort}
        statLabel={powerLabel}
        heat={heatOn}
        started={powerStarted}
        rows={powerRows ? shownPower : undefined}
        progress={powerProgress}
        ms={powerMs}
        focus={powerFocus}
        onheat={setHeat}
        onstart={() => (powerStarted = true)}
        onstop={stopPower}
        onclose={closePower}
        onhover={(id: number | undefined) => (powerHover = id)}
        onpick={(r: ShownPowerRow) => {
          powerFocus = r.ID;
          if (!r.Cluster) canvasEl?.focus(r.ID);
        }} />
    {/if}
  {/snippet}
  {#snippet nodeTipExtra(n: TreeNodeFx)}<NodeTipSections tip={nodeTip(n)} />{/snippet}
  {#snippet nodeTipTail(n: TreeNodeFx)}<NodeTipSections cost={nodeCost(n)} />{/snippet}
  {#snippet canvas(api: CanvasApi)}
    <div class="absolute inset-0">
      {#if view}
        <div class="pointer-events-none absolute inset-0 z-above overflow-hidden" aria-hidden="true">
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
          {#if powerOpen && powerHover !== undefined}
            {@const p = project(powerHover)}
            {#if p}
              <div
                class="absolute -translate-x-1/2 -translate-y-1/2 rounded-full border-3 border-accent shadow-[0_0_12px_var(--color-accent)]"
                data-power-hover={powerHover}
                style:left="{p.x}px"
                style:top="{p.y}px"
                style:width="{powerRingSize(view.scaling)}px"
                style:height="{powerRingSize(view.scaling)}px">
              </div>
            {/if}
          {/if}
          {#each powerCurrent as id (id)}
            {@const p = project(id)}
            {#if p}
              <div
                class="absolute size-8 -translate-x-1/2 -translate-y-1/2 animate-pulse rounded-full border-2 border-accent"
                data-power-current={id}
                style:left="{p.x}px"
                style:top="{p.y}px">
              </div>
            {/if}
          {/each}
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
                style:width="{searchRingSize(view.scaling)}px"
                style:height="{searchRingSize(view.scaling)}px"
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
            gestures={phone.current}
            onpick={(n: Node) => {
              const fx = fxOf(n);
              if (!fx) return;
              if (phone.current) offerUndo(fx);
              api.pick(fx);
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
    {#if undoToast}
      <UndoToast
        text={undoToast}
        onundo={() => {
          undoToast = undefined;
          void syncWrap?.Undo().catch(logError);
        }} />
    {/if}
  {/snippet}
</TreePage>

<TimelessDialog bind:open={timelessOpen} />

<Dialog bind:open={socketOpen} title={socketing ? `Socket a jewel: ${socketing.node.name}` : 'Socket a jewel'} size="sm">
  {#if socketing}
    <div role="group" aria-label="Jewels for this socket" class="flex max-h-90 flex-col gap-1 overflow-y-auto">
      {#each [{ ItemID: 0, Name: 'None', Rarity: '' }, ...socketing.choices] as j (j.ItemID)}
        <button
          class="choice-card justify-between gap-2 px-2.5 py-1.5 text-left text-13"
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

<Dialog phone="centred" open={classConfirm !== undefined} onclose={() => (classConfirm = undefined)} title="Class Change" size="sm">
  {#if classConfirm}
    <div class="flex flex-col gap-3 text-13">
      <p class="m-0">Changing class to {classConfirm.className} will reset your passive tree.</p>
      <p class="m-0">This can be avoided by connecting one of the {classConfirm.className} starting nodes to your tree.</p>
    </div>
  {/if}
  {#snippet actions()}
    <DialogActions confirm="Continue" tone="danger" oncancel={() => (classConfirm = undefined)} onconfirm={() => void answerClass(1).catch(logError)}>
      {#snippet extra()}<Button size="md" onclick={() => void answerClass(2).catch(logError)}>Connect Path</Button>{/snippet}
    </DialogActions>
  {/snippet}
</Dialog>
