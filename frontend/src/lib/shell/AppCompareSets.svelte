<script lang="ts">
  import CompareSetsPicker from '$lib/design/CompareSetsPicker.svelte';
  import { buildRevision, poe2Build } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { get } from 'svelte/store';
  import { game } from '$lib/go/engine';
  import { poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import type { Poe2Calc } from '$lib/go/sync_worker2';
  import { compareSets2, type Poe2CompareSides } from './compare_poe2';
  import { logError } from '$lib/utils';
  import type { CompareInfo, CompareSetKind, CompareSets } from './compare_types';

  // Build.lua:301-414
  // CompareTab.lua:290-340
  // CompareTab.lua:342-478
  type Props = { info: CompareInfo };
  let { info = $bindable() }: Props = $props();

  type Loadouts = { loadouts: string[]; selected: number };
  // the frame is keyed on the game, so the picker reads it once
  const poe2 = get(game) === 'poe2';

  // CompareTab.lua:290-389
  let sets = $state<CompareSets | undefined>();
  let bLoadouts = $state<Loadouts>({ loadouts: [], selected: -1 });
  $effect(() => {
    if (!info.seq || poe2) return;
    syncWrap
      ?.GetCompareSets()
      .then((s) => (sets = s))
      .catch(logError);
    syncWrap
      ?.GetCompareLoadouts()
      .then((l) => (bLoadouts = l))
      .catch(logError);
  });
  // CompareTab.lua:395-435
  const setCount = (kind: 'stage' | 'mine', text: string) => {
    const v = text.trim() === '' ? undefined : Number(text);
    if (v !== undefined && !Number.isInteger(v)) return;
    syncWrap
      ?.SetCompareCount(kind, v)
      .then((i) => (info = i))
      .catch(logError);
  };
  // CompareTab.lua:941-959
  type ConfigSets = { sets: { id: number; title?: string }[]; active: number };
  let aConfigSets = $state<ConfigSets>({ sets: [], active: 0 });
  let aLoadouts = $state<Loadouts>({ loadouts: [], selected: -1 });
  $effect(() => {
    void $buildRevision;
    if (!info.seq || poe2) return;
    syncWrap
      ?.GetConfigSets()
      .then((c) => (aConfigSets = c))
      .catch(logError);
    syncWrap
      ?.GetLoadouts()
      .then((l) => (aLoadouts = l))
      .catch(logError);
  });
  // CompareTab.lua:775-831
  // TreeTab.lua:649-666
  let primarySets = $state<Awaited<ReturnType<NonNullable<typeof syncWrap>['GetComparePrimarySets']>>>();
  $effect(() => {
    void $buildRevision;
    if (!info.seq || poe2) return;
    syncWrap
      ?.GetComparePrimarySets()
      .then((p) => (primarySets = p))
      .catch(logError);
  });
  let versionList = $state<{ value: string; label: string }[]>([]);
  // CompareTab.lua:279-389 (PoB-PoE2): both builds' sets, read again whenever either calculates; PoE2 has one tree
  // version and no loadouts
  $effect(() => {
    const aView = $poe2Build as Poe2Calc | undefined;
    if (!poe2 || !info.seq || !aView) return;
    const at = info.seq;
    poe2Worker()
      .then((w) => w.CompareSides() as Promise<Poe2CompareSides | undefined>)
      .then((sides) => {
        if (!sides || at !== info.seq) return;
        sets = compareSets2(sides.b);
        const versions = { a: sides.a.treeVersion, b: sides.b.treeVersion };
        versionList = [...new Set([versions.a, versions.b])].map((v) => ({ value: v, label: v }));
        const itemIds = sides.a.itemSets.sets.map((s) => s.id);
        primarySets = {
          tree: { labels: aView.treeSpecs?.labels ?? [], active: (aView.treeSpecs?.active ?? 1) - 1 },
          item: {
            labels: sides.a.itemSets.sets.map((s) => s.title || 'Default'),
            ids: itemIds,
            active: Math.max(0, itemIds.indexOf(sides.a.itemSets.active))
          },
          versions
        };
        const cs = aView.configSets ?? { sets: [], active: 1 };
        aConfigSets = {
          sets: cs.sets,
          active: Math.max(
            0,
            cs.sets.findIndex((c) => c.id === cs.active)
          )
        };
      })
      .catch(logError);
  });
  $effect(() => {
    if (poe2) return;
    syncWrap
      ?.GetTreeVersionOptions()
      .then((v) => (versionList = v ?? []))
      .catch(logError);
  });
  let convertMsg = $state('');
  const convertTo = async (side: 'a' | 'b', version: string, replace: boolean) => {
    if (side === 'a') convertMsg = (await syncWrap?.ConvertTreeSpec(version, replace)) ?? '';
    else {
      const r = await syncWrap?.ConvertCompareTreeSpec(version, replace);
      if (r) {
        info = r.info;
        convertMsg = r.msg;
      }
    }
  };
  // this build's tree, item or config set, by index in its list
  const pickPrimary = (kind: 'tree' | 'item' | 'config', index: number) => {
    if (poe2) {
      const id = kind === 'item' ? primarySets?.item.ids[index] : aConfigSets.sets[index]?.id;
      if (kind === 'tree') void poe2Run((w) => w.SetActiveTreeSpec(index + 1));
      else if (kind === 'item' && id !== undefined) void poe2Run((w) => w.EditItems({ op: 'set', id }));
      else if (kind === 'config' && id !== undefined) void poe2Run((w) => w.SelectConfigSet(id));
      return;
    }
    if (kind === 'tree') syncWrap?.SetActiveTreeSpec(index + 1)?.catch(logError);
    else if (kind === 'item' && primarySets) syncWrap?.SetActiveItemSet(primarySets.item.ids[index])?.catch(logError);
    else if (kind === 'config') {
      const set = aConfigSets.sets[index];
      if (set) syncWrap?.SetActiveConfigSet(set.id)?.catch(logError);
    }
  };
  const pickSet = (kind: CompareSetKind, index: number) => {
    if (poe2) {
      if (kind === 'parts' || kind === 'minion' || kind === 'minionSkill') return;
      poe2Worker()
        .then((w) => w.SetCompareSet(kind, index))
        .then((i) => (info = i))
        .catch(logError);
      return;
    }
    syncWrap
      ?.SetCompareSet(kind, index)
      .then((i) => (info = i))
      .catch(logError);
  };
  const pickLoadout = (side: 'a' | 'b', name: string) => {
    if (side === 'a') syncWrap?.SelectLoadout(name).catch(logError);
    else
      syncWrap
        ?.SelectCompareLoadout(name)
        .then((i) => (info = i))
        .catch(logError);
  };
</script>

<CompareSetsPicker
  {sets}
  {primarySets}
  {aConfigSets}
  {aLoadouts}
  {bLoadouts}
  {versionList}
  message={convertMsg}
  onloadout={pickLoadout}
  onprimary={pickPrimary}
  onset={pickSet}
  oncount={setCount}
  onconvert={(side: 'a' | 'b', version: string, replace: boolean) => void convertTo(side, version, replace).catch(logError)} />
