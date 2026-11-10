<script module lang="ts">
  import type { CompareView, MetricId } from '$lib/design/compare';

  // CompareTab.lua is created with each build: the view is kept while the build stays open; opens is the PoE1 open
  // count, or a name for the open PoE2 build's session
  const fresh = (opens: number | string) => ({ view: 'summary' as CompareView, metric: 'dps' as MetricId, swapped: false, opens });
  let kept = fresh(0);
  const sessions = new WeakMap<object, string>();
  let sessionCount = 0;
  const sessionName = (s: object | undefined) => {
    if (!s) return 'poe2:none';
    if (!sessions.has(s)) sessions.set(s, `poe2:${++sessionCount}`);
    return sessions.get(s)!;
  };
</script>

<script lang="ts">
  import Select from '$lib/design/Select.svelte';
  import Input from '$lib/design/Input.svelte';
  import Button from '$lib/design/Button.svelte';
  import Field from '$lib/design/Field.svelte';
  import InlineMessage from '$lib/design/InlineMessage.svelte';
  import SectionHeading from '$lib/design/SectionHeading.svelte';
  import { resource } from 'runed';
  import { untrack } from 'svelte';
  import { get } from 'svelte/store';
  import { buildOpens, buildRevision, currentBuild, currentBuildPath, outputs, poe2Build, poe2Session } from '$lib/global.js';
  import { game } from '$lib/go/engine';
  import { poe2Run, poe2Worker, skillSelect2 } from '$lib/go/engine_poe2.svelte';
  import { compareCalcs2, compareFixture2, itemTip2, powerRows2, rowItem2, withPower2, type Poe2CompareSides, type Poe2Power } from './compare_poe2';
  import type { Poe2Calc } from '$lib/go/sync_worker2';
  import { decodeCode, rootElement } from '$lib/go/game_detect';
  import { COMPARE_MIXED_POE1 } from './compare_types';
  import { NOT_A_CODE } from '$lib/design/share';
  import type { Tree2 } from '$lib/components/skill-tree/scene2';
  import type { SkillsFixture } from '$lib/design/fixtures';
  import { proxy } from 'comlink';
  import { syncWrap } from '$lib/go/worker';
  import type { CalcsHeaderField, CalcsHeaderSide } from '$lib/design/compare';
  import { writeMainSkill } from './main_skill';
  import { logError } from '$lib/utils';
  import { compareName, linkCode } from './link_import';
  import { importFailure } from '$lib/design/share';
  import type { CompareInfo } from './compare_types';
  import { GetSkillGems } from '$lib/cache';
  import ComparePage from '$lib/design/ComparePage.svelte';
  import BuySimilarDialog from '$lib/design/BuySimilarDialog.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import { accountSession, canReimport, reimportPrimary } from './account_session.svelte';
  import BuildPicker from '$lib/design/BuildPicker.svelte';
  import AppCompareSets from './AppCompareSets.svelte';
  import type { SavedBuild } from '$lib/design/fixtures/app';
  import { BUILD_SORT_KEY, loadBuildSort, saveBuildSort } from '$lib/display/build_sort';
  import { followOtherTabs } from '$lib/storage/tabs';
  import { LISTED, tradeDefences, type TradeDefence, type TradeItem, type TradeOptions } from '$lib/design/trade';
  import type { CompareFixture, PowerRow } from '$lib/design/fixtures';
  import type { Tree } from '$lib/skill_tree/types';
  import { calculateAllocationPath } from '$lib/components/skill-tree/paths';
  import { loadTree, pathOf } from './build_list';
  import {
    BASE_METRICS,
    classRoots,
    compareMetrics,
    configDiffs,
    configRows,
    inputOf,
    configControl,
    itemRows,
    jewelRows,
    itemsInto,
    metricsOf,
    powerRows,
    skillsFixture,
    summaryStats,
    treeOverlay,
    type ComparePowerRow
  } from './compare';
  import type { StatCompareSection } from '$lib/skill_tree/stat_compare';
  import { identityOf } from './identity';
  import { nodeWhatIfLines, type WhatIfBlock } from './compare_node_tip';
  import { showFlavourText } from '$lib/display/item_options';
  import { calcSections } from '$lib/calcs/calc_sections';
  import { calcElements } from '$lib/display/calcs';
  import { compareCalcsView, modEntries, modKey } from '$lib/display/compare_calcs';
  import type { calculator } from '$lib/types';
  import type { calculator2 } from '$lib/types2';
  import { compareLoaded } from './import_modal.svelte';

  // the frame is keyed on the game, so the page reads it once
  const poe2 = get(game) === 'poe2';
  const opens = poe2 ? sessionName(get(poe2Session)) : get(buildOpens);
  let ui = $state(kept.opens === opens ? kept : fresh(opens));
  $effect(() => {
    const now = poe2 ? sessionName($poe2Session) : $buildOpens;
    if (now !== untrack(() => ui.opens)) ui = fresh(now);
  });
  $effect(() => {
    kept = { view: ui.view, metric: ui.metric, swapped: ui.swapped, opens: ui.opens };
  });

  let info = $state<CompareInfo>({ name: '', seq: 0, names: [], active: -1 });
  // CompareTab.lua:968-974
  let metricList = $state.raw(BASE_METRICS);
  $effect(() => {
    Promise.resolve(poe2 ? poe2Worker().then((w) => w.TreePowerStats()) : syncWrap?.GetPowerStats())
      .then((s) => (metricList = compareMetrics(s ?? [])))
      .catch(logError);
  });
  // CompareTab.lua:239-242, 1240-1253
  let adding = $state(false);
  let loadError = $state<string | null>(null);
  let loading = $state(false);
  const refreshInfo = () => (poe2 ? poe2Worker().then((w) => w.GetCompareInfo()) : syncWrap?.GetCompareInfo())?.then((i) => (info = i)).catch(logError);
  $effect(() => {
    void $currentBuild;
    void $poe2Build;
    void compareLoaded.seq;
    void refreshInfo();
  });

  const saved = resource(
    () => info.seq === 0 || adding,
    async (empty) => {
      const w = syncWrap;
      return empty && w ? await loadTree((dir) => w.ListBuilds(dir), '') : { builds: [], folders: [] };
    }
  );
  $effect(() => {
    if (!(info.seq === 0 || adding) || !syncWrap) return;
    const w = syncWrap;
    void w
      .OnBuildsChanged(
        'compare-picker',
        proxy(() => void saved.refetch())
      )
      .catch(logError);
    return () => void w.OnBuildsChanged('compare-picker', undefined).catch(logError);
  });
  let code = $state('');
  let codeName = $state('');
  let picking = $state(false);
  // CompareTab.lua:1588-1593
  let pickSort = $state(loadBuildSort());
  $effect(() => saveBuildSort(pickSort));
  $effect(() => followOtherTabs(BUILD_SORT_KEY, { set: (s: typeof pickSort) => (pickSort = s) }, loadBuildSort));
  const load = (run: () => Promise<CompareInfo> | undefined) => {
    loadError = null;
    loading = true;
    run()
      ?.then((i) => {
        info = i;
        adding = false;
      })
      .catch((e: unknown) => (loadError = importFailure(e)))
      .finally(() => (loading = false));
  };
  const remove = () => {
    void (poe2 ? poe2Worker().then((w) => w.RemoveCompare()) : syncWrap?.RemoveCompare())?.then((i) => (info = i));
  };
  // CompareTab.lua:1328-1337 (PoB-PoE2): a code whose XML is not a PoE2 build is refused before the worker reads it
  const loadCode2 = async (text: string, name: string) => {
    const xml = await decodeCode(text);
    if (xml === undefined) throw new Error(NOT_A_CODE);
    if (rootElement(xml) === 'PathOfBuilding') throw new Error(COMPARE_MIXED_POE1);
    return (await poe2Worker()).LoadCompare(xml, name);
  };
  // CompareTab.lua:923-933
  let configShowAll = $state(false);
  let tree = $state<Tree | undefined>();
  const treeLoad = async () => {
    if (tree || !syncWrap) return tree;
    tree = JSON.parse(await syncWrap.GetTree(await syncWrap.GetTreeVersion())) as Tree;
    return tree;
  };

  const model = resource(
    () => [identityOf($outputs), $buildRevision, info.seq] as const,
    async ([, , seq]) => {
      const out = get(outputs);
      if (poe2 || !seq || !syncWrap || !out) return undefined;
      const [sides, items, gems, t, jewels] = await Promise.all([
        syncWrap.GetCompareSides(),
        syncWrap.GetCompareItems(),
        GetSkillGems(),
        treeLoad(),
        syncWrap.GetCompareJewels()
      ]);
      if (!sides || !items || !t) return undefined;
      const aOut = { Output: out.Output, SkillFlags: out.SkillFlags, PointsUsed: out.PointsUsed, Minion: out.Minion };
      const stats = summaryStats(aOut, sides.b.outputs);
      const gemById = new Map(gems.map((g) => [g.ID, g]));
      return {
        seq,
        sides,
        items,
        jewels: jewels ?? [],
        aOut,
        stats,
        config: configDiffs(sides.a.inputs, sides.b.inputs),
        relA: out.ConfigRelevance,
        aSkills: skillsFixture(sides.a, gemById),
        bSkills: skillsFixture(sides.b, gemById),
        tree: treeOverlay(t, sides.a.nodes, sides.b.nodes),
        t
      };
    }
  );
  const m = $derived(model.current);

  const aName = $derived(
    currentBuildPath.current
      ?.split('/')
      .at(-1)
      ?.replace(/\.xml$/, '') || 'This build'
  );
  // the PoE2 build against its comparison, read again whenever either calculates
  let tree2: Tree2 | undefined;
  const model2 = resource(
    () => [$poe2Build, info.seq, configShowAll, metricList] as const,
    async ([aView, seq, showAll, metrics]) => {
      if (!poe2 || !seq || !aView) return undefined;
      const w = await poe2Worker();
      tree2 ??= await w.TreeData().catch((e: Error) => (logError(e), undefined));
      const sides = (await w.CompareSides()) as Poe2CompareSides | undefined;
      const items = await w.CompareItemRows();
      if (!sides || !items) return undefined;
      return { sides, items, ...compareFixture2(aView as never, sides, aName, metrics, showAll, tree2, items) };
    }
  );

  const sideTexts = (slot: string): [string, string] => {
    const jewel = m?.jewels.find((j) => j.slot === slot);
    if (jewel) return [jewel.a?.text ?? '', jewel.b?.text ?? ''];
    return [m?.items.a.find((i) => i.slot === slot)?.text ?? '', m?.items.b.find((i) => i.slot === slot)?.text ?? ''];
  };
  // CompareTab.lua:3806-3809, 3950-3952, CompareBuySimilar.lua:281-531
  let buying = $state<{ item: TradeItem; slot?: string; defences: TradeDefence[]; url?: (o: TradeOptions) => Promise<string | undefined> }>();

  type Power = { receiver: 'a' | 'b'; rows: ComparePowerRow[]; configBack: Record<string, number[]> };
  const power = resource(
    () => [identityOf(m), ui.swapped, metricList.length] as const,
    async ([, swapped]): Promise<Power | undefined> => {
      const stats = metricList.map((x) => x.stat);
      const mm = m;
      if (!mm || !syncWrap) return undefined;
      const receiver = swapped ? 'b' : 'a';
      const giverOf = (r: 'a' | 'b') => (r === 'a' ? 'b' : 'a');
      const cfg = (r: 'a' | 'b') => mm.config.map((c) => inputOf(c.key, r === 'a' ? c.bValue : c.aValue));
      const all = { TreeNodes: true, Items: true, SkillGems: true, SupportGems: true, Config: true };
      const entries = await syncWrap.BuildComparePower(receiver, stats, all, cfg(receiver));
      if (!entries) return undefined;
      const other = giverOf(receiver);
      const back = await syncWrap.BuildComparePower(
        other,
        stats,
        { TreeNodes: false, Items: false, SkillGems: false, SupportGems: false, Config: true },
        cfg(other)
      );
      const rows = powerRows(entries, mm.items[other], mm.items[receiver], mm.config, receiver, metricList);
      const configBack = Object.fromEntries((back ?? []).map((e) => [e.Var, [...(e.Impact ?? [])]]));
      return { receiver, rows, configBack };
    }
  );

  // CompareTab.lua:3932-3977 (PoB-PoE2): each slot's Equip comparison into the shown build
  const itemImpacts2 = resource(
    () => [model2.current, ui.swapped, ui.view] as const,
    async ([mm, swapped, view]) => {
      if (!poe2 || !mm || view !== 'items') return undefined;
      const w = await poe2Worker();
      const receiver = swapped ? 'b' : 'a';
      const out: Record<string, StatCompareSection[] | undefined> = {};
      for (const r of mm.items.slots) if (r[swapped ? 'a' : 'b']) out[r.slot] = await w.CompareEquipImpact(receiver, r.slot);
      return { receiver, out };
    }
  );

  // CompareTab.lua:3129-3157 (PoB-PoE2): the report into the shown build, and the config-only report back
  const power2 = resource(
    () => [model2.current, ui.swapped, metricList] as const,
    async ([mm, swapped, metrics]): Promise<Poe2Power | undefined> => {
      if (!poe2 || !mm) return undefined;
      const w = await poe2Worker();
      const stats = metrics.map((x) => x.stat);
      const receiver = swapped ? 'b' : 'a';
      const all = { TreeNodes: true, Items: true, SkillGems: true, SupportGems: true, Config: true };
      const entries = await w.BuildComparePower(receiver, stats, all);
      if (!entries) return undefined;
      const back = await w.BuildComparePower(receiver === 'a' ? 'b' : 'a', stats, {
        TreeNodes: false,
        Items: false,
        SkillGems: false,
        SupportGems: false,
        Config: true
      });
      if (!back) return undefined;
      return {
        receiver,
        rows: powerRows2(entries, mm.sides, mm.fixture.config, receiver, metrics),
        configBack: Object.fromEntries(back.map((e) => [e.Var, [...(e.Impact ?? [])]]))
      };
    }
  );

  // CompareTab.lua:2785-2868
  const jewelPower = resource(
    () => [identityOf(power.current)] as const,
    async (): Promise<{ receiver: 'a' | 'b'; rows: ComparePowerRow[] } | undefined> => {
      const p = power.current;
      const mm = m;
      if (!p || !mm || !syncWrap) return undefined;
      const entries = await syncWrap.BuildCompareJewelPower(
        p.receiver,
        metricList.map((x) => x.stat)
      );
      if (!entries) return undefined;
      const other = p.receiver === 'a' ? 'b' : 'a';
      return { receiver: p.receiver, rows: powerRows(entries, mm.items[other], mm.items[p.receiver], mm.config, p.receiver, metricList) };
    }
  );

  let calcsOnlyDifferences = $state(true);
  $effect(() => {
    if (poe2 || ui.view !== 'calcs' || !syncWrap) return;
    syncWrap.setCalcTabElements(calcElements(calcSections)).catch(logError);
    return () => void syncWrap?.clearCalcTabElements();
  });
  let calcsRev = $state(0);
  const calcs = resource(
    () => [identityOf(m), ui.view, calcsRev] as const,
    async ([, view]) => (!poe2 && view === 'calcs' ? await syncWrap?.GetCompareCalcs() : undefined)
  );

  const itemImpacts = resource(
    () => [identityOf(m), ui.swapped, ui.view] as const,
    async ([, swapped, view]) => {
      const mm = m;
      if (!mm || !syncWrap || view !== 'items') return undefined;
      const receiver = swapped ? 'b' : 'a';
      const giver = swapped ? 'a' : 'b';
      const out: Record<string, StatCompareSection[] | undefined> = {};
      for (const it of itemsInto(mm.items[receiver], mm.items[giver])) {
        out[it.slot] = await syncWrap.CompareItemInto(receiver, it.text, it.slot);
      }
      return { receiver, out };
    }
  );

  // CompareTab.lua:4617-4633
  let modsEqual = $state.raw<Map<string, boolean>>();
  $effect(() => {
    const c = calcs.current;
    modsEqual = undefined;
    if (!c) return;
    const sds = modEntries(calcSections);
    Promise.resolve(syncWrap?.CompareModsEqual(sds))
      .then((eq) => {
        if (calcs.current === c && eq) modsEqual = new Map(sds.map((sd, i) => [modKey(sd), eq[i]]));
      })
      .catch(logError);
  });
  // CompareTab.lua:4583-4652 (PoB-PoE2)
  const calcs2 = resource(
    () => [model2.current, ui.view, calcsOnlyDifferences, ui.swapped] as const,
    async ([mm, view, only, swapped]) => {
      if (!poe2 || !mm || view !== 'calcs') return undefined;
      const pages = await (await poe2Worker()).CompareCalcsPages();
      const aView = get(poe2Build);
      if (!pages || !aView) return undefined;
      const a = { view: aView as Poe2Calc, page: pages.a };
      const b = { view: mm.sides.b.view, page: pages.b };
      const w = await poe2Worker();
      const modsEqual = (sds: calculator2.ModSectionData[]) => w.CompareModsEqual(sds);
      const sections = await (swapped
        ? compareCalcs2(b, a, only, modsEqual, [mm.sides.bName, aName])
        : compareCalcs2(a, b, only, modsEqual, [aName, mm.sides.bName]));
      return { sections, a, b };
    }
  );
  const calcsShown = $derived.by(() => {
    if (poe2) return ui.view === 'calcs' ? calcs2.current?.sections : undefined;
    const c = calcs.current;
    if (!c || ui.view !== 'calcs') return undefined;
    const bName = m?.sides.bName ?? 'Compare Build';
    return ui.swapped
      ? compareCalcsView(calcSections, c.b, c.a, calcsOnlyDifferences, bName, aName, modsEqual)
      : compareCalcsView(calcSections, c.a, c.b, calcsOnlyDifferences, aName, bName, modsEqual);
  });
  // CompareTab.lua:490-718, 4480-4608 (PoB-PoE2): each build's Calcs tab inputs over its own calculation
  const calcsHeader2 = $derived.by(() => {
    const c = calcs2.current;
    const mm = model2.current;
    if (!poe2 || !c || !mm) return undefined;
    const side = (k: 'a' | 'b'): CalcsHeaderSide => {
      const { view, page } = c[k];
      return {
        name: k === 'a' ? aName : mm.sides.bName,
        groups: (view.socketGroups ?? []).map((g) => g.label),
        skillNumber: page.inputs.skillNumber,
        buffMode: page.inputs.buffMode,
        // CompareTab.lua:2309-2329 (PoB-PoE2): RefreshSkillSelectControls with the "Calcs" suffix and no library buttons
        showMinion: page.controls.MinionShown ? page.inputs.showMinion : undefined,
        controls: page.skills.length ? { ...skillSelect2(page.controls), MinionLibraryShown: false, BeastLibraryShown: false } : undefined,
        info: [
          { label: 'Aura/Buff Skills', text: page.strings.BuffList ?? '' },
          { label: 'Combat Buffs', text: page.strings.CombatList ?? '' },
          { label: 'Curses/Debuffs', text: page.strings.CurseList ?? '' }
        ]
      };
    };
    const order: ['a', 'b'] | ['b', 'a'] = ui.swapped ? ['b', 'a'] : ['a', 'b'];
    return {
      sides: [side(order[0]), side(order[1])] as [CalcsHeaderSide, CalcsHeaderSide],
      oncontrol: (shown: 0 | 1, field: CalcsHeaderField, value: number | string | boolean | undefined) => {
        // PoE2's skill data has no multipart skill
        if (field === 'SkillPart' || (value === undefined && field !== 'StageCount' && field !== 'MineCount')) return;
        const n = value === undefined ? undefined : Number(value);
        if (order[shown] === 'b')
          void poe2Worker()
            .then((w) => w.SetCompareCalcsControl(field, value))
            .then((i) => (info = i))
            .catch(logError);
        else if (field === 'group') void poe2Run((w) => w.SetCalcsSkillNumber(Number(value)));
        else if (field === 'mode') void poe2Run((w) => w.SetCalcsBuffMode(String(value)));
        else if (field === 'showMinion') void poe2Run((w) => w.SetCalcsShowMinion(!!value));
        else if (field === 'StageCount') void poe2Run((w) => w.SetSkillStageCount(true, n));
        else if (field === 'MineCount') void poe2Run((w) => w.SetSkillMineCount(true, n));
        else if (n === undefined) return;
        else if (field === 'StatSet') void poe2Run((w) => w.SetSkillStatSet(true, n));
        else if (field === 'Minion') void poe2Run((w) => w.SetCompareCalcsMinion(n));
        else if (field === 'MinionSkill') void poe2Run((w) => w.SetSkillMinionSkill(true, n));
        else if (field === 'MinionSkillStatSet') void poe2Run((w) => w.SetSkillMinionSkillStatSet(true, n));
        else void poe2Run((w) => w.SetMainActiveSkillCalcs(n));
      },
      breakdown: async (shown: 0 | 1, name: string) => (await poe2Worker()).CompareSkillBreakdown(order[shown], name)
    };
  });
  // CompareTab.lua:490-718, 4480-4608
  const calcsHeader = $derived.by(() => {
    if (poe2) return calcsHeader2;
    const c = calcs.current;
    if (!c || !m) return undefined;
    const bName = m.sides.bName ?? 'Compare Build';
    const side = (k: 'a' | 'b'): CalcsHeaderSide => {
      const out = c[k];
      const inp = c.inputs[k];
      const strings = out.OutputStrings ?? {};
      return {
        name: k === 'a' ? aName : bName,
        groups: inp.groups,
        skillNumber: inp.skillNumber,
        buffMode: inp.buffMode,
        showMinion: inp.showMinion || out.SkillFlags?.haveMinion ? inp.showMinion : undefined,
        controls: out.MainSkill,
        info: [
          { label: 'Aura/Buff Skills', text: strings.BuffList ?? '' },
          { label: 'Combat Buffs', text: strings.CombatList ?? '' },
          { label: 'Curses/Debuffs', text: strings.CurseList ?? '' }
        ]
      };
    };
    const order: ['a', 'b'] | ['b', 'a'] = ui.swapped ? ['b', 'a'] : ['a', 'b'];
    return {
      sides: [side(order[0]), side(order[1])] as [CalcsHeaderSide, CalcsHeaderSide],
      oncontrol: (shown: 0 | 1, field: CalcsHeaderField, value: number | string | boolean | undefined) => {
        if (order[shown] === 'b') {
          void Promise.resolve(syncWrap?.SetCompareCalcsControl(field, value))
            .then(() => calcsRev++)
            .catch(logError);
        } else if (field === 'group') void syncWrap?.SetCalcsSkillNumber(Number(value));
        else if (field === 'mode') void syncWrap?.SetCalcsBuffMode(String(value));
        else if (field === 'showMinion') void syncWrap?.SetCalcsShowMinion(!!value);
        else writeMainSkill(c.a.MainSkill, field, typeof value === 'number' ? value : undefined, true);
      },
      breakdown: async (shown: 0 | 1, name: string) => (await syncWrap?.CompareSkillBreakdown(order[shown] === 'a' ? 0 : 1, name)) ?? []
    };
  });
  const compare = $derived.by((): CompareFixture | undefined => {
    if (poe2) {
      const mm = model2.current;
      if (!mm) return undefined;
      const ii = itemImpacts2.current;
      const items = mm.fixture.items.map((r) => ({
        ...r,
        impactA: ii?.receiver === 'a' ? ii.out[r.slot] : undefined,
        impactB: ii?.receiver === 'b' ? ii.out[r.slot] : undefined
      }));
      return withPower2({ ...mm.fixture, items }, power2.current);
    }
    if (!m) return undefined;
    const p = power.current;
    const jp = jewelPower.current;
    const shownPower = (r: 'a' | 'b'): ComparePowerRow[] => (p && p.receiver === r ? [...p.rows, ...(jp?.receiver === r ? jp.rows : [])] : []);
    const configBackInto = (r: 'a' | 'b') => (p ? (p.receiver === r ? undefined : p.configBack) : undefined);
    const fromRows = (r: 'a' | 'b'): Record<string, number[]> =>
      Object.fromEntries<number[]>(
        shownPower(r)
          .filter((x) => x.cat === 'config')
          .map((x): [string, number[]] => [x.key ?? '', x.impact])
      );
    const impactsInto = (r: 'a' | 'b'): Record<string, number[]> => configBackInto(r) ?? fromRows(r);
    const ii = itemImpacts.current;
    const side = (k: 'a' | 'b') => {
      const s = m.sides[k];
      const o = k === 'a' ? m.aOut : s.outputs;
      return {
        name: k === 'a' ? aName : m.sides.bName,
        className: s.className,
        ascendClassName: s.ascendClassName,
        level: s.level,
        points: o?.PointsUsed?.[0] ?? 0,
        metrics: metricsOf(o, metricList),
        stats: m.stats[k],
        configSet: s.configSet,
        power: shownPower(k) as PowerRow[]
      };
    };
    const zero = [0, 0, 0];
    return {
      a: side('a'),
      b: { ...side('b'), skills: m.bSkills },
      items: itemRows(m.items.a, m.items.b, ii?.receiver === 'a' ? ii.out : {}, ii?.receiver === 'b' ? ii.out : {}),
      jewels: jewelRows(m.jewels),
      tree: m.tree,
      config: configRows(m.sides.a.inputs, m.sides.b.inputs, m.relA, m.sides.b.outputs?.ConfigRelevance, configShowAll).map((c) => ({
        key: c.key,
        section: c.section,
        label: c.label,
        a: c.a,
        b: c.b,
        diff: c.diff,
        impactA: impactsInto('a')[c.key] ?? zero,
        impactB: impactsInto('b')[c.key] ?? zero,
        ...configControl(c.key),
        aValue: c.aValue,
        bValue: c.bValue
      }))
    };
  });

  const aSkills = $derived<SkillsFixture | undefined>(poe2 ? model2.current?.aSkills : m?.aSkills);
  const bName = $derived((poe2 ? model2.current?.sides.bName : m?.sides.bName) ?? 'Compare Build');

  const take = async (id: number) => {
    if (!m || !syncWrap) return;
    const nodes = await syncWrap.GetPassiveNodes();
    const paths = (await syncWrap.CalculateAllocationPaths(await syncWrap.GetTreeVersion(), [...nodes], classRoots(m.t, m.sides.a.className))) ?? {};
    const path = calculateAllocationPath(paths, id);
    if (path) await syncWrap.AllocateNodes(path);
  };
  const drop = async (id: number) => {
    if (!m || !syncWrap) return;
    const nodes = await syncWrap.GetPassiveNodes();
    const cut = (await syncWrap.CalculateDependentNodes(await syncWrap.GetTreeVersion(), [...nodes], id, classRoots(m.t, m.sides.a.className))) ?? [];
    await syncWrap.DeallocateNodes([id, ...cut]);
  };
  // PassiveTreeView.lua:425-594 (PoB-PoE2): a click allocates the node and its path, or frees it and its dependents
  const click2 = (id: number) =>
    void poe2Run(
      (w) => w.TreeClick(id, false, 0, false, 0),
      (r) => r.calc
    );
  const onapply = (row: PowerRow) => {
    const r = row as ComparePowerRow;
    if (poe2) {
      if (ui.swapped) return;
      if (r.cat === 'tree' && r.nodeId !== undefined) click2(r.nodeId);
      else if (r.cat === 'items' && r.slot) void poe2Run((w) => w.CopyCompareItem(r.slot!, true));
      else if (r.cat === 'config' && r.key) void poe2Run((w) => w.UseCompareConfig(r.key!));
      return;
    }
    if (!syncWrap || !m || ui.swapped) return;
    if (r.cat === 'tree' && r.nodeId !== undefined) take(r.nodeId).catch(logError);
    else if (r.cat === 'items' && r.slot) {
      const it = m.items.b.find((i) => i.slot === r.slot);
      if (it) syncWrap.EquipItem(it.text, it.slot).catch(logError);
    } else if (r.cat === 'config' && r.key) syncWrap.UseCompareConfig(r.key).catch(logError);
  };
  const onnode = (id: number, action: 'Take' | 'Drop') => {
    if (ui.swapped) return;
    if (poe2) return click2(id);
    (action === 'Take' ? take(id) : drop(id)).catch(logError);
  };

  // CompareTab.lua:244-271
  const reimportTip = $derived.by(() => {
    if (accountSession.stage !== 'SELECTCHAR') return 'No character selected.\nGo to Import/Export Build tab and select a character first.';
    const c = accountSession.selected;
    return `${c ? `Re-import character from the game server:\n${c.name} (${c.class}, ${c.league})` : 'Re-import the currently selected character.'}\nRefreshes passive tree, jewels, items, and skills.`;
  });
  let confirmTree = $state<() => void>();
</script>

{#if confirmTree}<Dialog open title="Character Import" onclose={() => (confirmTree = undefined)}>
    <p class="m-0 text-13">Importing the passive tree will overwrite your current tree.</p>
    {#snippet actions()}
      <DialogActions
        confirm="Import"
        oncancel={() => (confirmTree = undefined)}
        onconfirm={() => {
          const run = confirmTree;
          confirmTree = undefined;
          run?.();
        }} />
    {/snippet}
  </Dialog>{/if}

{#if !$currentBuild && !$poe2Build}
  <p class="note p-5" role="status">Open a build to compare it.</p>
{:else if info.seq === 0 || adding}
  <section aria-label="Choose a build to compare" class="panel flex max-w-180 flex-col gap-3 p-3.5">
    <SectionHeading size="lg">COMPARE WITH</SectionHeading>
    <Field label="Import a build code or a build site link as the comparison:">
      <Input size="lg" class="font-mono text-13" bind:value={code} aria-label="Comparison build code" /></Field>
    <Field label="Name (optional)">
      <Input size="md" class="text-13" maxlength={100} bind:value={codeName} aria-label="Comparison name" /></Field>
    <!-- CompareTab.lua:1462-1525 -->
    <Button
      size="md"
      variant="primary"
      class="self-start"
      disabled={!code.trim() || loading}
      onclick={() =>
        load(async () =>
          poe2
            ? loadCode2(await linkCode(code), compareName(code, codeName))
            : (syncWrap?.LoadCompareCode(await linkCode(code), compareName(code, codeName)) ?? Promise.reject(new Error('no worker')))
        )}>Compare</Button>
    <SectionHeading level={3} class="mt-1">OR A SAVED BUILD</SectionHeading>
    <!-- CompareTab.lua:1529-1640 -->
    <Button size="md" class="self-start" aria-haspopup="dialog" disabled={loading} onclick={() => (picking = true)}>Choose a saved build…</Button>
    <BuildPicker
      bind:open={picking}
      title="Compare with a saved build"
      bind:sort={pickSort}
      action="Compare"
      builds={poe2 ? saved.current?.builds.filter((b) => b.game === 'poe2') : saved.current?.builds}
      folders={saved.current?.folders ?? []}
      onpick={(b: SavedBuild) =>
        load(() =>
          poe2
            ? Promise.resolve(syncWrap?.ReadBuild(pathOf(b.folder, b.name))).then(async (xml) =>
                xml ? (await poe2Worker()).LoadCompare(xml, b.name) : Promise.reject(new Error('the build could not be read'))
              )
            : syncWrap?.LoadCompareSaved(pathOf(b.folder, b.name))
        )} />
    {#if adding}<Button size="sm" class="self-start" onclick={() => (adding = false)}>Cancel</Button>{/if}
    {#if loading}<span role="status" class="note">Loading…</span>{/if}
    {#if loadError}<InlineMessage kind="error">Comparison failed: {loadError}</InlineMessage>{/if}
  </section>
{:else if compare && aSkills && (m || poe2)}
  <div class="flex shrink-0 items-center gap-2 text-12">
    <!-- CompareTab.lua:227-237 -->
    <Field label="Compare with:" inline>
      <Select
        size="xs"
        aria-label="Compare with"
        value={String(info.active)}
        onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
          const i = Number(e.currentTarget.value);
          void (poe2 ? poe2Worker().then((w) => w.SetActiveCompare(i)) : syncWrap?.SetActiveCompare(i))?.then((v) => (info = v));
        }}
        >{#each info.names as n, i (i)}<option value={String(i)}>{n}</option>{/each}</Select
      ></Field>
    <Button size="2xs" onclick={() => (adding = true)}>Import…</Button>
    {#if !poe2}<Button size="2xs" disabled={!canReimport()} title={reimportTip} onclick={() => reimportPrimary((run) => (confirmTree = run))}
        >Re-import Current</Button
      >{/if}
    <Button size="2xs" onclick={remove}>Remove comparison</Button>
  </div>
  <AppCompareSets bind:info />
  <ComparePage
    {compare}
    {aSkills}
    swapped={ui.swapped}
    bind:view={ui.view}
    bind:metric={ui.metric}
    metrics={metricList}
    calcs={calcsShown}
    {calcsHeader}
    bind:calcsOnlyDifferences
    calcsNames={ui.swapped ? [bName, aName] : [aName, bName]}
    powertip={poe2
      ? undefined
      : async (row: PowerRow) => {
          const r = row as ComparePowerRow;
          if (r.cat === 'tree' && r.nodeId !== undefined) {
            // ComparePowerReportListControl.lua:124-135
            const n = (await treeLoad())?.nodes[r.nodeId];
            if (!n) return undefined;
            const lines = (n.stats ?? []).map((text) => ({ text: `^x8888FF${text}`, sep: false }));
            // PassiveTreeView.lua:1631-1645
            if (n.reminderText?.length) lines.push({ text: '', sep: true }, ...n.reminderText.map((text) => ({ text: `^xA0A080${text}`, sep: false })));
            if (n.flavourText?.length && get(showFlavourText))
              lines.push({ text: '', sep: true }, ...n.flavourText.map((text) => ({ text: `^xAF6025${text}`, sep: false })));
            // PassiveTreeView.lua:1658-1724
            if (!ui.swapped && syncWrap && m) {
              const id = r.nodeId;
              const nodes = await syncWrap.GetPassiveNodes();
              const paths = (await syncWrap.CalculateAllocationPaths(await syncWrap.GetTreeVersion(), [...nodes], classRoots(m.t, m.sides.a.className))) ?? {};
              const path = calculateAllocationPath(paths, id) ?? [];
              const blocks: WhatIfBlock[] = [];
              const diffs = (await syncWrap.GetTreeViewState())?.ShowStatDifferences ?? true;
              const one = diffs ? await syncWrap.StatCompareNodes([id], []) : undefined;
              if (one?.sections.length) blocks.push({ header: '^7Allocating this node will give you:', sections: one.sections });
              if (diffs && path.length > 1) {
                const all = await syncWrap.StatCompareNodes(path, [], path.length);
                if (all?.sections.length) blocks.push({ header: '^7Allocating this node and all nodes leading to it will give you:', sections: all.sections });
              }
              lines.push(...nodeWhatIfLines(blocks, path.length, diffs));
            }
            // PassiveTreeView.lua:1607-1627
            const skills = ui.swapped ? [] : ((await syncWrap?.GetNodeSkillTooltips(r.nodeId)) ?? []);
            const children = skills.map((c) => c.map((l) => ({ text: l.Text, sep: l.Separator })));
            return { influence: [], title: [n.name ?? String(r.nodeId)], lines, children, childrenShown: true };
          }
          const t = r.cat === 'items' ? await syncWrap?.CompareItemTooltip(ui.swapped ? 'a' : 'b', r.what) : undefined;
          return t
            ? { influence: [...(t.Influence ?? [])], title: [...(t.Title ?? [])], lines: (t.Lines ?? []).map((l) => ({ text: l.Text, sep: l.Separator })) }
            : undefined;
        }}
    supportpairs={poe2
      ? undefined
      : async (side: 'a' | 'b', group: number) => (await syncWrap?.CompareGemSupportPairs((side === 'a') !== ui.swapped ? 'a' : 'b', group)) ?? []}
    itemtip={poe2
      ? async (side: 'a' | 'b', name: string) => {
          const mm = model2.current;
          const it = mm && rowItem2([...mm.items.slots, ...mm.items.jewels], (side === 'a') !== ui.swapped ? 'a' : 'b', name);
          return it && itemTip2(it);
        }
      : async (side: 'a' | 'b', name: string) => {
          const t = await syncWrap?.CompareItemTooltip((side === 'a') !== ui.swapped ? 'a' : 'b', name);
          return t
            ? { influence: [...(t.Influence ?? [])], title: [...(t.Title ?? [])], lines: (t.Lines ?? []).map((l) => ({ text: l.Text, sep: l.Separator })) }
            : undefined;
        }}
    calcsModRows={poe2
      ? async (side: 0 | 1, sd: calculator.ModSectionData) => (await poe2Worker()).CompareCalcsModRows((side === 0) !== ui.swapped ? 'a' : 'b', sd)
      : async (side: 0 | 1, sd: calculator.ModSectionData) => (await syncWrap?.CompareModRows((ui.swapped ? 1 - side : side) as 0 | 1, sd)) ?? []}
    powerBusy={poe2 ? power2.loading || !power2.current : power.loading || !power.current || jewelPower.loading}
    onswap={() => (ui.swapped = !ui.swapped)}
    {onapply}
    {onnode}
    onuseconfig={(key: string) => {
      if (ui.swapped) return;
      if (poe2) void poe2Run((w) => w.UseCompareConfig(key));
      else syncWrap?.UseCompareConfig(key).catch(logError);
    }}
    onconfigset={(side: 'a' | 'b', key: string, value: boolean | number | string | null) => {
      if (poe2) {
        if ((side === 'a') !== ui.swapped) void poe2Run((w) => w.SetConfigOption(key, value ?? undefined));
        else
          poe2Worker()
            .then((w) => w.SetCompareConfigOption(key, value ?? undefined))
            .then((i) => (info = i))
            .catch(logError);
      } else if ((side === 'a') !== ui.swapped) syncWrap?.SetConfigOption(key, value as boolean | number | string)?.catch(logError);
      else
        syncWrap
          ?.SetCompareConfigOption(key, value)
          .then((i) => (info = i))
          .catch(logError);
    }}
    oncopyconfig={() => {
      if (ui.swapped) return;
      if (poe2) void poe2Run((w) => w.CopyCompareConfig());
      else syncWrap?.CopyCompareConfig().catch(logError);
    }}
    {configShowAll}
    onconfigshowall={() => (configShowAll = !configShowAll)}
    oncopytree={(andUse: boolean) => {
      if (ui.swapped) return;
      if (poe2) void poe2Run((w) => w.CopyCompareSpec(andUse));
      else syncWrap?.CopyCompareSpec(andUse).catch(logError);
    }}
    onitem={(slot: string, equip: boolean) => {
      if (poe2) {
        if (!ui.swapped) void poe2Run((w) => w.CopyCompareItem(slot, equip));
        return;
      }
      const jewel = m?.jewels.find((j) => j.slot === slot);
      const it = jewel ? jewel.b : m?.items.b.find((i) => i.slot === slot);
      if (!it || ui.swapped) return;
      if (jewel && equip) syncWrap?.EquipCompareJewel(it.text, jewel.node).catch(logError);
      else (equip ? syncWrap?.EquipItem(it.text, slot) : syncWrap?.CopyCompareItem(it.text))?.catch(logError);
    }}
    itemcards={poe2
      ? async (slot: string) => {
          const [a, b] = await (await poe2Worker()).CompareItemCards(slot);
          return ui.swapped ? [b, a] : [a, b];
        }
      : async (slot: string) => {
          const [l, r] = sideTexts(slot);
          return (await syncWrap?.CompareItemCards(ui.swapped ? r : l, ui.swapped ? l : r)) ?? [[], []];
        }}
    socketview={poe2
      ? // CompareTab.lua:3827-3842 (PoB-PoE2): the open build's tree around the socket
        async () => {
          const aView = get(poe2Build) as Poe2Calc | undefined;
          return tree2 && aView ? { tree: tree2, radii: [], alloc: new Set(aView.tree.alloc) } : undefined;
        }
      : async (node: number) => {
          if (!m) return undefined;
          const radii = (await syncWrap?.GetTreeSocketRadii(node)) ?? [];
          return { tree: m.t, radii, alloc: new Set(m.sides.a.nodes) };
        }}
    onbuy={poe2
      ? // CompareTab.lua:3776-3778, 3917-3918 and CompareBuySimilar.lua:284-541 (PoB-PoE2): the compared build's item
        async (slot: string) => {
          const w = await poe2Worker();
          const v = await w.CompareBuySimilar(slot);
          if (!v) return;
          const defences = (v.Defences ?? []).map((d) => ({ label: d.Label, value: d.Value, tradeKey: d.TradeKey }));
          const item = { rarity: v.Unique ? 'UNIQUE' : 'RARE', title: '', name: v.Name, baseName: v.BaseName, type: v.Category };
          const url = async (o: TradeOptions) =>
            w.CompareBuySimilarURL(slot, {
              League: o.league,
              Listed: LISTED.findIndex(([, value]) => value === o.listed) + 1,
              BaseType: o.baseType,
              IlvlMin: o.ilvlMin,
              IlvlMax: o.ilvlMax,
              Defences: defences.map((d) => ({ Checked: !!o.defences?.[d.tradeKey], Min: o.defences?.[d.tradeKey]?.min, Max: o.defences?.[d.tradeKey]?.max }))
            });
          buying = { item, slot, defences, url };
        }
      : async (slot: string) => {
          const text = sideTexts(slot)[1];
          const p = text ? await syncWrap?.ParseItem(text) : undefined;
          if (!p?.item) return;
          const item = { rarity: p.item.Rarity, title: p.item.Title, name: p.item.Name, baseName: p.item.BaseName, type: p.item.Type };
          buying = { item, slot: slot.startsWith('Jewel ') ? undefined : slot, defences: tradeDefences(item, p.item.ArmourData) };
        }} />
  <BuySimilarDialog
    bind:open={() => !!buying, (v) => !v && (buying = undefined)}
    item={buying?.item}
    slot={buying?.slot}
    defences={buying?.defences}
    realms={poe2 ? [['PoE2', 'poe2']] : undefined}
    url={buying?.url}
    onsearch={(url: string) => window.open(url, '_blank', 'noopener')} />
{:else}
  <p class="note p-5" role="status">Calculating…</p>
{/if}
