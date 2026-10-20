<script lang="ts">
  import CompareSetsPicker from '$lib/design/CompareSetsPicker.svelte';
  import { buildRevision } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import type { CompareInfo, CompareSetKind, CompareSets } from './compare_types';

  // Build.lua:301-414
  // CompareTab.lua:290-340
  // CompareTab.lua:342-478
  type Props = { info: CompareInfo };
  let { info = $bindable() }: Props = $props();

  type Loadouts = { loadouts: string[]; selected: number };

  // CompareTab.lua:290-389
  let sets = $state<CompareSets | undefined>();
  let bLoadouts = $state<Loadouts>({ loadouts: [], selected: -1 });
  $effect(() => {
    if (!info.seq) return;
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
    if (!info.seq) return;
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
    if (!info.seq) return;
    syncWrap
      ?.GetComparePrimarySets()
      .then((p) => (primarySets = p))
      .catch(logError);
  });
  let versionList = $state<{ value: string; label: string }[]>([]);
  $effect(() => {
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
    if (kind === 'tree') syncWrap?.SetActiveTreeSpec(index + 1)?.catch(logError);
    else if (kind === 'item' && primarySets) syncWrap?.SetActiveItemSet(primarySets.item.ids[index])?.catch(logError);
    else if (kind === 'config') {
      const set = aConfigSets.sets[index];
      if (set) syncWrap?.SetActiveConfigSet(set.id)?.catch(logError);
    }
  };
  const pickSet = (kind: CompareSetKind, index: number) => {
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
