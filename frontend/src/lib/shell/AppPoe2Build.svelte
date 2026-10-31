<script lang="ts">
  import Button from '$lib/design/Button.svelte';
  import Checkbox from '$lib/design/Checkbox.svelte';
  import Input from '$lib/design/Input.svelte';
  import PobText from '$lib/design/PobText.svelte';
  import Select from '$lib/design/Select.svelte';
  import { currentBuildPath, poe2Build, type Poe2View } from '$lib/global.js';
  import SaveAsDialog from './SaveAsDialog.svelte';
  import { savePoe2Build } from './poe2_save';
  import { saveErrorText } from './save_error';
  import { logError } from '$lib/utils';
  import { NOTE_COLOR, type StatLine } from '$lib/display/stat_list';
  import { buildStatList2, displayStats2, minionDisplayStats2 } from '$lib/display/stats2';
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
  import type { Poe2Calc, Worker2Type } from '$lib/go/sync_worker2';
  import type { Remote } from 'comlink';

  let { view }: { view: Poe2View } = $props();

  // the PoE2 worker hands back a Poe2Calc; a view without its fields renders the stats it has
  const calc = $derived(view as Partial<Poe2Calc> & Poe2View);

  let busy = $state(false);
  let error = $state<string | undefined>();
  let showAll = $state(false);

  const edit = async (op: (w: Remote<Worker2Type>) => Promise<Poe2Calc>) => {
    busy = true;
    error = undefined;
    try {
      const { poe2Worker } = await import('$lib/go/worker2');
      poe2Build.set(await op(await poe2Worker()));
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busy = false;
    }
  };

  // Build.lua:2773-2787 (PoB-PoE2)
  let saveAs = $state(false);
  let saved = $state(false);
  const onsave = () => {
    const path = currentBuildPath.current;
    if (!path) {
      saveAs = true;
      return;
    }
    busy = true;
    error = undefined;
    savePoe2Build(path)
      .then(() => (saved = true))
      .catch((e: Error) => {
        logError(e);
        error = saveErrorText(path);
      })
      .finally(() => (busy = false));
  };

  const warnings = $derived.by(() => {
    const out: string[] = [];
    if (calc.minion) buildStatList2({ Output: calc.minion, SkillFlags: calc.minionSkillFlags }, minionDisplayStats2, out);
    buildStatList2({ Output: calc.output, SkillFlags: calc.skillFlags }, displayStats2, out);
    return out;
  });
  const minionGroups = $derived(calc.minion ? buildStatList2({ Output: calc.minion, SkillFlags: calc.minionSkillFlags }, minionDisplayStats2) : undefined);
  const playerGroups = $derived(buildStatList2({ Output: calc.output, SkillFlags: calc.skillFlags }, displayStats2));

  const groups = $derived(calc.socketGroups ?? []);
  const mainGroup = $derived(groups[(calc.mainSocketGroup ?? 1) - 1]);
  const inputs = $derived((calc.configInputs ?? {}) as Record<string, ConfigValue2 | undefined>);
  const rel = $derived(calc.relevance ? relevanceSets2(calc.relevance) : undefined);
  const sections = $derived.by(() => {
    const r = rel;
    if (!r) return [];
    return configurations2.map((s) => ({ ...s, rows: s.rows.filter((row) => isShown2(row, inputs, r, showAll)) })).filter((s) => s.rows.some(isVariable2));
  });
  const invalid = (row: Option2) => rel !== undefined && isVariable2(row) && !isEligible2(row, inputs, rel) && isModified2(row, inputs);

  const setMain = (which: 'group' | 'skill', index: number) => edit((w) => (which === 'group' ? w.SetMainSocketGroup(index) : w.SetMainActiveSkill(index)));
  const setOption = (name: string, value: ConfigValue2 | undefined) => edit((w) => w.SetConfigOption(name, value));
  const valueOf = (e: Event) => (e.currentTarget as HTMLInputElement | HTMLSelectElement).value;
  const numberOf = (raw: string) => (raw.trim() === '' || Number.isNaN(Number(raw)) ? undefined : Number(raw));
  const listValue = (row: Option2, index: string) => row.list?.[Number(index)]?.val as ConfigValue2 | undefined;
  const listIndex = (row: Option2) => {
    const cur = isVariable2(row) ? currentValue(row, inputs) : undefined;
    const i = row.list?.findIndex((e) => e.val === cur) ?? -1;
    return String(Math.max(i, 0));
  };
</script>

{#snippet statGroups(list: StatLine[][])}
  {#each list as g, gi (gi)}
    <dl class="m-0 grid grid-cols-[1fr_max-content] gap-x-3 text-14">
      {#each g as line, li (li)}
        <dt style:color={line.labelColor}>{line.label}:</dt>
        <dd class="m-0 text-right tabular-nums" data-stat={line.stat} style:color={line.valueColor}>
          {line.value}{#if line.note}<span style:color={NOTE_COLOR}>{line.note}</span>{/if}
        </dd>
      {/each}
    </dl>
  {/each}
{/snippet}

<section class="page gap-3 p-4" aria-label="Path of Exile 2 build">
  <div class="flex items-center gap-3">
    <span class="text-18 font-semibold">Path of Exile 2 build (preview)</span>
    <Button disabled={busy} onclick={onsave}>Save</Button>
    <Button disabled={busy} onclick={() => (saveAs = true)}>Save as…</Button>
    <Button onclick={() => poe2Build.set(undefined)}>Close</Button>
    {#if saved && !busy}<span class="note" role="status">Saved</span>{/if}
    {#if busy}<span class="note" role="status">Calculating…</span>{/if}
  </div>
  <p class="m-0 text-14">
    Level {view.level}
    {view.ascendClassName && view.ascendClassName !== 'None' ? view.ascendClassName : view.className}
  </p>
  {#if error}<p class="m-0 text-14" role="alert">{error}</p>{/if}

  <div class="flex min-h-0 flex-1 flex-wrap gap-4">
    <aside class="flex w-full max-w-[300px] shrink-0 flex-col gap-2 overflow-y-auto" aria-label="Stats">
      <!-- Build.lua:363-381, 2059-2090 (PoB-PoE2) -->
      <span class="text-14">Main Skill:</span>
      <Select
        size="sm"
        aria-label="Main socket group"
        value={String(calc.mainSocketGroup ?? 1)}
        disabled={busy || groups.length === 0}
        options={groups.length ? groups.map((g, i) => ({ value: String(i + 1), label: g.label })) : [{ value: '1', label: '<No skills added yet>' }]}
        onchange={(e: Event) => setMain('group', Number(valueOf(e)))} />
      {#if mainGroup}
        <Select
          size="sm"
          aria-label="Main active skill"
          value={String(calc.mainActiveSkill || 1)}
          disabled={busy || mainGroup.skills.length <= 1}
          options={mainGroup.skills.map((s, i) => ({ value: String(i + 1), label: s }))}
          onchange={(e: Event) => setMain('skill', Number(valueOf(e)))} />
      {/if}

      {#if minionGroups}
        <span class="text-14 font-semibold">Minion:</span>
        {@render statGroups(minionGroups)}
        <span class="text-14 font-semibold">Player:</span>
      {/if}
      {#if calc.skillFlags?.disable}<span class="text-14">Skill disabled:</span>{/if}
      {@render statGroups(playerGroups)}
      {#if warnings.length}
        <ul class="m-0 pl-4 text-14" aria-label="Warnings">
          {#each warnings as w (w)}<li>{w}</li>{/each}
        </ul>
      {/if}
    </aside>

    {#if rel}
      <div class="flex min-w-0 flex-1 flex-col gap-3 overflow-y-auto" aria-label="Configuration">
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
  </div>
  <SaveAsDialog bind:open={saveAs} save={savePoe2Build} onsaved={() => (saved = true)} />
</section>
