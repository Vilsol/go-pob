<script lang="ts">
  import Button from '$lib/design/Button.svelte';
  import Checkbox from '$lib/design/Checkbox.svelte';
  import Input from '$lib/design/Input.svelte';
  import PobText from '$lib/design/PobText.svelte';
  import Select from '$lib/design/Select.svelte';
  import { poe2Build, type Poe2View } from '$lib/global.js';
  import {
    configurations2,
    currentValue,
    isShown2,
    isVariable2,
    isModified2,
    isEligible2,
    relevanceSets2,
    type ConfigValue2,
    type Option2
  } from '$lib/display/configurations2';
  import type { Poe2Calc, Poe2Items as Poe2ItemsData, Poe2SkillGroup, Worker2Type } from '$lib/go/sync_worker2';
  import { poe2Busy, poe2Error, poe2Run, poe2Worker } from '$lib/go/engine_poe2.svelte';
  import SkillTree2 from '$lib/components/skill-tree/SkillTree2.svelte';
  import Poe2Calcs from './Poe2Calcs.svelte';
  import Poe2Items from './Poe2Items.svelte';
  import Poe2Skills from './Poe2Skills.svelte';
  import type { Tree2 } from '$lib/components/skill-tree/scene2';
  import type { Remote } from 'comlink';

  // the PoE2 bodies of the frame's page routes
  let { route }: { route: string } = $props();

  // the PoE2 worker hands back a Poe2Calc; a view without its fields renders the stats it has
  const view = $derived($poe2Build as (Partial<Poe2Calc> & Poe2View) | undefined);
  const busy = $derived($poe2Busy);
  const fail = (err: unknown) => poe2Error.set(err instanceof Error ? err.message : String(err));

  const setMain = (which: 'group' | 'skill', index: number) => poe2Run((w) => (which === 'group' ? w.SetMainSocketGroup(index) : w.SetMainActiveSkill(index)));

  // ConfigTab.lua:183-188, 308-331 (PoB-PoE2)
  let showAll = $state(false);
  const inputs = $derived((view?.configInputs ?? {}) as Record<string, ConfigValue2 | undefined>);
  const rel = $derived(view?.relevance ? relevanceSets2(view.relevance) : undefined);
  const sections = $derived.by(() => {
    const r = rel;
    if (!r) return [];
    return configurations2.map((s) => ({ ...s, rows: s.rows.filter((row) => isShown2(row, inputs, r, showAll)) })).filter((s) => s.rows.some(isVariable2));
  });
  const invalid = (row: Option2) => rel !== undefined && isVariable2(row) && !isEligible2(row, inputs, rel) && isModified2(row, inputs);
  const setOption = (name: string, value: ConfigValue2 | undefined) => poe2Run((w) => w.SetConfigOption(name, value));
  const valueOf = (e: Event) => (e.currentTarget as HTMLInputElement | HTMLSelectElement).value;
  const numberOf = (raw: string) => (raw.trim() === '' || Number.isNaN(Number(raw)) ? undefined : Number(raw));
  const listValue = (row: Option2, index: string) => row.list?.[Number(index)]?.val as ConfigValue2 | undefined;
  const listIndex = (row: Option2) => {
    const cur = isVariable2(row) ? currentValue(row, inputs) : undefined;
    const i = row.list?.findIndex((e) => e.val === cur) ?? -1;
    return String(Math.max(i, 0));
  };

  let treeData = $state.raw<Tree2 | undefined>();
  $effect(() => {
    if (route !== 'tree' || treeData) return;
    void poe2Worker()
      .then((w) => w.TreeData())
      .then((d) => (treeData = d))
      .catch(fail);
  });
  // what the Calcs, Skills and Items pages read, fetched again for every calculation while one is open; the last
  // shows meanwhile
  type Fetched<T> = { for: unknown; data: T } | undefined;
  let pageData = $state.raw<{ calcs?: Fetched<Record<string, number>>; skills?: Fetched<Poe2SkillGroup[]>; items?: Fetched<Poe2ItemsData> }>({});
  const fetchers = {
    calcs: (w: Remote<Worker2Type>): Promise<unknown> => w.CalcsMods(),
    skills: (w: Remote<Worker2Type>): Promise<unknown> => w.SkillGroups(),
    items: (w: Remote<Worker2Type>): Promise<unknown> => w.EquippedItems()
  };
  $effect(() => {
    const at = view;
    const p = route;
    if (!at || (p !== 'calcs' && p !== 'skills' && p !== 'items') || pageData[p]?.for === at) return;
    void poe2Worker()
      .then((w) => fetchers[p](w))
      .then((data) => (pageData = { ...pageData, [p]: { for: at, data } }))
      .catch(fail);
  });
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
</script>

{#if view}
  {#if $poe2Error}<p class="m-0 px-4 py-2 text-14" role="alert">{$poe2Error}</p>{/if}
  {#if route === 'tree'}
    <div class="flex min-h-[480px] min-w-0 flex-1 flex-col" aria-label="Tree" data-poe2-page>
      <div class="flex items-center gap-3 px-3 py-1.5 text-14">
        <span>Level {view.level} {view.ascendClassName && view.ascendClassName !== 'None' ? view.ascendClassName : view.className}</span>
        {#if view.tree}
          <span aria-label="Passive points used">Points: {view.tree.used}{view.tree.ascUsed ? ` / Ascendancy: ${view.tree.ascUsed}` : ''}</span>
        {/if}
        {#if busy}<span class="note" role="status">Calculating…</span>{/if}
      </div>
      <div class="flex min-h-0 min-w-0 flex-1">
        {#if treeData && view.tree}
          <SkillTree2
            data={treeData}
            spec={view.tree}
            {busy}
            onclick={treeClick}
            onattribute={treeAttribute}
            onhover={treeHover}
            onundo={() => treeUndo(false)}
            onredo={() => treeUndo(true)} />
        {:else}
          <span class="note" role="status">Loading the passive tree…</span>
        {/if}
      </div>
    </div>
  {:else if route === 'skills'}
    <Poe2Skills groups={pageData.skills?.data} mainGroup={view.mainSocketGroup ?? 1} {busy} onmain={(i: number) => setMain('group', i)} />
  {:else if route === 'items'}
    <Poe2Items items={pageData.items?.data} {busy} />
  {:else if route === 'calcs'}
    <Poe2Calcs
      calc={view}
      {busy}
      mods={pageData.calcs?.data}
      onmaingroup={(i: number) => setMain('group', i)}
      onmainskill={(i: number) => setMain('skill', i)} />
  {:else if route === 'configuration' && rel}
    <div class="flex min-w-0 flex-1 flex-col gap-3 overflow-y-auto" aria-label="Configuration" data-poe2-page>
      <div class="flex items-center gap-3">
        <span class="text-16 font-semibold">Configuration</span>
        <!-- ConfigTab.lua:183-188 (PoB-PoE2) -->
        <Button onclick={() => (showAll = !showAll)}>{showAll ? 'Hide Ineligible Configurations' : 'Show All Configurations'}</Button>
      </div>
      {#each sections as s (s.name)}
        <fieldset class="m-0 flex flex-col gap-1.5 border border-line p-3" aria-label={s.name}>
          <legend class="px-1 text-14 font-semibold">{s.name}</legend>
          {#each s.rows as row, ri (row.var ?? `label-${ri}`)}
            {#if !isVariable2(row)}
              <span class="text-14 text-text-2"><PobText text={row.label} /></span>
            {:else if row.type === 'check'}
              <Checkbox
                checked={currentValue(row, inputs) === true}
                disabled={busy}
                data-config={row.var}
                title={row.tooltip}
                onchange={(e: Event) => setOption(row.var, (e.currentTarget as HTMLInputElement).checked)}>
                <span class:text-error={invalid(row)}><PobText text={row.label} /></span>
              </Checkbox>
            {:else}
              <label class="grid grid-cols-[1fr_140px] items-center gap-2 text-14" title={row.tooltip}>
                <span class:text-error={invalid(row)}><PobText text={row.label} /></span>
                {#if row.type === 'list'}
                  <Select
                    size="sm"
                    data-config={row.var}
                    disabled={busy}
                    value={listIndex(row)}
                    options={(row.list ?? []).map((e, i) => ({ value: String(i), label: e.label }))}
                    onchange={(e: Event) => setOption(row.var, listValue(row, valueOf(e)))} />
                {:else}
                  <Input
                    size="sm"
                    inputmode="decimal"
                    data-config={row.var}
                    disabled={busy}
                    value={(currentValue(row, inputs) as number | undefined) ?? ''}
                    placeholder={row.defaultPlaceholderState === undefined ? undefined : String(row.defaultPlaceholderState)}
                    onchange={(e: Event) => setOption(row.var, numberOf(valueOf(e)))} />
                {/if}
              </label>
            {/if}
          {/each}
        </fieldset>
      {/each}
    </div>
  {/if}
{/if}
