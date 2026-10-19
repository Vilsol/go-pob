<script lang="ts">
  import { buildRevision } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import Dialog from '$lib/design/Dialog.svelte';
  import { logError } from '$lib/utils';
  import type { CompareInfo, CompareSetKind, CompareSets } from './compare_types';

  // Build.lua:301-414
  // CompareTab.lua:290-340
  // CompareTab.lua:342-478
  type Props = { info: CompareInfo };
  let { info = $bindable() }: Props = $props();

  type Loadouts = { loadouts: string[]; selected: number };
  const uid = $props.id();
  let open = $state({ a: false, b: false });

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
  const SET_SELECTS: { kind: CompareSetKind; label: string }[] = [
    { kind: 'tree', label: 'Tree set' },
    { kind: 'skill', label: 'Skill set' },
    { kind: 'item', label: 'Item set' },
    { kind: 'config', label: 'Config set' }
  ];
  const SKILL_SELECTS: { kind: CompareSetKind; label: string }[] = [
    { kind: 'group', label: 'Socket group' },
    { kind: 'skills', label: 'Main skill' },
    { kind: 'parts', label: 'Skill part' },
    { kind: 'minion', label: 'Minion' },
    { kind: 'minionSkill', label: 'Minion skill' }
  ];
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
  let converting = $state<{ side: 'a' | 'b'; version: string } | null>(null);
  let convertMsg = $state('');
  const convertTo = async (replace: boolean) => {
    const c = converting;
    converting = null;
    if (!c) return;
    if (c.side === 'a') convertMsg = (await syncWrap?.ConvertTreeSpec(c.version, replace)) ?? '';
    else {
      const r = await syncWrap?.ConvertCompareTreeSpec(c.version, replace);
      if (r) {
        info = r.info;
        convertMsg = r.msg;
      }
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

{#snippet loadout(side: 'a' | 'b', l: Loadouts)}
  <label class="field field-inline"
    >Loadout:
    <select
      class="select input-xs max-w-60"
      aria-label={side === 'a' ? 'Primary loadout' : 'Comparison loadout'}
      disabled={!l.loadouts.length}
      value={String(l.selected)}
      onchange={(e) => {
        const name = l.loadouts[Number(e.currentTarget.value)];
        e.currentTarget.value = String(l.selected);
        if (name !== undefined) pickLoadout(side, name);
      }}>
      {#if l.selected < 0}<option value="-1" disabled>{l.loadouts.length ? 'Custom sets' : 'No loadouts'}</option>{/if}
      {#each l.loadouts as n, i (i)}<option value={String(i)}>{n}</option>{/each}
    </select></label>
  <button class="btn btn-2xs" aria-expanded={open[side]} aria-controls="{uid}-sets-{side}" onclick={() => (open[side] = !open[side])}
    >{side === 'a' ? "This build's sets" : 'Comparison sets'} {open[side] ? '▴' : '▾'}</button>
{/snippet}

{#if sets}
  {@const cur = sets}
  <div role="group" aria-label="Comparison build sets" class="grid shrink-0 grid-cols-[repeat(auto-fit,minmax(min(100%,420px),1fr))] gap-2 text-12">
    <section aria-label="This build" class="flex min-w-0 flex-col gap-1.5 rounded-md border border-line border-l-3 border-l-build-a px-2.5 py-1.5">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-11 font-semibold tracking-[0.08em] text-build-a">THIS BUILD</span>
        {@render loadout('a', aLoadouts)}
      </div>
      <div id="{uid}-sets-a" hidden={!open.a} class="flex flex-wrap items-center gap-2">
        {#if primarySets}
          {@const ps = primarySets}
          <label class="field field-inline"
            >Tree set:
            <select
              class="select input-xs max-w-50"
              aria-label="Primary tree set"
              disabled={ps.tree.labels.length < 2}
              value={String(ps.tree.active)}
              onchange={(e) => syncWrap?.SetActiveTreeSpec(Number(e.currentTarget.value) + 1)?.catch(logError)}
              >{#each ps.tree.labels as l, i (i)}<option value={String(i)}>{l}</option>{/each}</select
            ></label>
          <label class="field field-inline"
            >Item set:
            <select
              class="select input-xs max-w-50"
              aria-label="Primary item set"
              disabled={ps.item.labels.length < 2}
              value={String(ps.item.active)}
              onchange={(e) => syncWrap?.SetActiveItemSet(ps.item.ids[Number(e.currentTarget.value)])?.catch(logError)}
              >{#each ps.item.labels as l, i (i)}<option value={String(i)}>{l}</option>{/each}</select
            ></label>
        {/if}
        <label class="field field-inline"
          >Config set:
          <select
            class="select input-xs max-w-50"
            aria-label="Primary config set"
            disabled={aConfigSets.sets.length < 2}
            value={String(aConfigSets.active)}
            onchange={(e) => {
              const set = aConfigSets.sets[Number(e.currentTarget.value)];
              if (set) syncWrap?.SetActiveConfigSet(set.id)?.catch(logError);
            }}
            >{#each aConfigSets.sets as c, i (i)}<option value={String(i)}>{c.title || 'Default'}</option>{/each}</select
          ></label>
        {#if primarySets}
          {@const ps = primarySets}
          <label class="field field-inline"
            >Tree version:
            <select
              class="select input-xs max-w-50"
              aria-label="Primary tree version"
              value={ps.versions.a}
              onchange={(e) => {
                const version = e.currentTarget.value;
                e.currentTarget.value = ps.versions.a;
                if (version !== ps.versions.a) converting = { side: 'a', version };
              }}
              >{#each versionList as o (o.value)}<option value={o.value}>{o.label}</option>{/each}</select
            ></label>
        {/if}
      </div>
    </section>
    <section aria-label="Comparison" class="flex min-w-0 flex-col gap-1.5 rounded-md border border-line border-l-3 border-l-build-b px-2.5 py-1.5">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-11 font-semibold tracking-[0.08em] text-build-b">COMPARISON</span>
        {@render loadout('b', bLoadouts)}
      </div>
      <div id="{uid}-sets-b" hidden={!open.b} class="flex flex-wrap items-center gap-2">
        {#each SET_SELECTS as sel (sel.kind)}
          {@const list = cur[sel.kind]}
          {#if list}
            <label class="field field-inline"
              >{sel.label}:
              <select
                class="select input-xs max-w-50"
                aria-label="Comparison {sel.label.toLowerCase()}"
                disabled={list.labels.length < 2}
                value={String(list.active)}
                onchange={(e) => pickSet(sel.kind, Number(e.currentTarget.value))}
                >{#each list.labels as l, i (i)}<option value={String(i)}>{l}</option>{/each}</select
              ></label>
          {/if}
        {/each}
        {#if primarySets}
          {@const ps = primarySets}
          <label class="field field-inline"
            >Tree version:
            <select
              class="select input-xs max-w-50"
              aria-label="Comparison tree version"
              value={ps.versions.b}
              onchange={(e) => {
                const version = e.currentTarget.value;
                e.currentTarget.value = ps.versions.b;
                if (version !== ps.versions.b) converting = { side: 'b', version };
              }}
              >{#each versionList as o (o.value)}<option value={o.value}>{o.label}</option>{/each}</select
            ></label>
        {/if}
      </div>
      <div class="flex flex-wrap items-center gap-2">
        {#each SKILL_SELECTS as sel (sel.kind)}
          {@const list = cur[sel.kind]}
          {#if list}
            <label class="field field-inline"
              >{sel.label}:
              <select
                class="select input-xs max-w-50"
                aria-label="Comparison {sel.label.toLowerCase()}"
                disabled={list.labels.length < 2 ||
                  (sel.kind === 'skills' && !cur.skills.enabled) ||
                  ((sel.kind === 'minion' || sel.kind === 'minionSkill') && !cur[sel.kind]?.enabled)}
                value={String(list.active)}
                onchange={(e) => pickSet(sel.kind, Number(e.currentTarget.value))}
                >{#each list.labels as l, i (i)}<option value={String(i)}>{l}</option>{/each}</select
              ></label>
          {/if}
        {/each}
        {#each [{ kind: 'stage' as const, label: 'Stages' }, { kind: 'mine' as const, label: 'Mines' }] as box (box.kind)}
          {#if cur[box.kind] !== undefined}
            <label class="field field-inline"
              >{box.label}:
              <input
                class="input input-xs w-13"
                inputmode="numeric"
                aria-label="Comparison {box.label.toLowerCase()}"
                value={cur[box.kind]}
                onchange={(e) => setCount(box.kind, e.currentTarget.value)} /></label>
          {/if}
        {/each}
      </div>
    </section>
  </div>
{/if}
{#if convertMsg}<div role="alert" class="note text-12">{convertMsg}</div>{/if}
<Dialog
  small
  open={converting !== null}
  onclose={() => (converting = null)}
  title={converting ? `Convert to Version ${versionList.find((o) => o.value === converting?.version)?.label ?? converting.version}` : 'Convert'}
  class="w-115 p-4.5">
  <div class="flex flex-col gap-3 text-13">
    <p class="m-0">Warning: some or all of the passives may be de-allocated due to changes in the tree.</p>
    <p class="m-0">Convert will replace your current tree.<br />Copy + Convert will backup your current tree.</p>
    <div class="flex gap-1.5">
      <button class="btn btn-sm btn-primary" onclick={() => void convertTo(true)}>Convert</button>
      <button class="btn btn-sm" onclick={() => void convertTo(false)}>Copy + Convert</button>
      <button class="btn btn-sm" onclick={() => (converting = null)}>Cancel</button>
    </div>
  </div>
</Dialog>
