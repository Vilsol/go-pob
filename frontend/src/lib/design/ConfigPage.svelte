<script lang="ts" module>
  const collapsed = $state<Record<string, boolean>>({});
</script>

<script lang="ts">
  import OptionGroup from './OptionGroup.svelte';
  import OptionRow from './OptionRow.svelte';
  import NumberInput from './NumberInput.svelte';
  import Select from './Select.svelte';
  import Dropdown from './Dropdown.svelte';
  import Button from './Button.svelte';
  import PobText from './PobText.svelte';
  import ConfigCustom from './ConfigCustom.svelte';
  import { besideAnchor } from './items';
  import { pobPlain } from './pob_text';
  import { configView, listIndex, INVALID, type ConfigFilter, type CustomOps, type Inputs, type OptionView } from './config';
  import { optionTooltip } from '$lib/display/configurations';
  import type { ConfigValue } from '$lib/display/configurations';
  import type { calculator } from '$lib/types';
  import type { CustomGroup } from './fixtures';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import SetListDialog from './SetListDialog.svelte';
  import type { SetListOp } from './set_list';
  import type { StatCompareSection } from '$lib/skill_tree/stat_compare';

  type Props = {
    inputs: Inputs;
    placeholders: Record<string, number | string>;
    relevance: calculator.ConfigRelevance;
    sets: string[];
    activeSet: number;
    custom: CustomGroup[];
    customSelected: number;
    customOps: CustomOps;
    filter: ConfigFilter;
    onset: (key: string, value: ConfigValue) => void;
    onunset?: (key: string) => void;
    onreset: () => void;
    onconfigset?: (i: number) => void;
    /* ConfigOptions.lua:61-72 */
    output?: Record<string, number>;
    onconfigsets?: (op: SetListOp) => void;
    /* ConfigTab.lua:721-766 */
    whatif?: (key: string, value: ConfigValue) => Promise<StatCompareSection[] | undefined>;
    revision?: unknown;
  };
  let {
    inputs,
    placeholders,
    relevance,
    sets,
    activeSet,
    custom,
    customSelected,
    customOps,
    filter = $bindable(),
    onset,
    onunset,
    onreset,
    onconfigset,
    output,
    onconfigsets,
    whatif,
    revision
  }: Props = $props();
  let managing = $state(false);

  const view = $derived(configView(inputs, placeholders, relevance, filter));
  // ConfigTab.lua:988-989
  let searchBox = $state<HTMLInputElement>();
  const onwindowkey = (e: KeyboardEvent) => {
    if (e.key.toLowerCase() === 'f' && (e.ctrlKey || e.metaKey) && !e.altKey && searchBox) {
      e.preventDefault();
      searchBox.focus();
      searchBox.select();
    }
  };

  let tip = $state<{ var: string; anchor: HTMLElement; id: string; entry?: number }>();
  const tipLines = (o: OptionView) =>
    [o.modified && '• modified from default', !o.relevant && 'does not affect this build', o.invalid && INVALID].filter(Boolean) as string[];
  // ConfigTab.lua:334-349
  const tipValue = (o: OptionView) => (o.v.type === 'list' && tip?.var === o.v.var && tip.entry !== undefined ? o.v.list[tip.entry]?.value : o.value);
  const tooltipOf = (o: OptionView) => optionTooltip(o.v, tipValue(o) as ConfigValue, output);
  const canCompare = (o: OptionView) =>
    !!whatif && !o.invalid && (o.v.type === 'check' || (o.v.type === 'list' && tip?.var === o.v.var && tip.entry !== undefined && tipValue(o) !== o.value));
  const hasTip = (o: OptionView) => !!tooltipOf(o) || tipLines(o).length > 0 || canCompare(o);
  const enter = (o: OptionView, id: string) => (e: Event) => {
    tip = { var: o.v.var, anchor: e.currentTarget as HTMLElement, id };
  };
  const leave = (id: string) => () => {
    if (tip?.id === id) tip = undefined;
  };
  const hoverEntry = (o: OptionView, id: string) => (i: number | null, el: HTMLElement | null) => {
    if (i === null || !el) {
      if (tip?.id === id) tip = undefined;
      return;
    }
    tip = { var: o.v.var, anchor: el, id, entry: i };
  };
  const tipOption = $derived.by(() => {
    if (!tip) return undefined;
    const key = tip.var;
    const o = [...view.most, ...view.sections.flatMap((s) => s.rows)].find((r): r is OptionView => r.kind === 'option' && r.v.var === key);
    return o && hasTip(o) ? o : undefined;
  });
  let compare = $state.raw<{ key: string; rev: unknown; sections?: StatCompareSection[] }>();
  $effect(() => {
    const rev = revision;
    const o = tipOption;
    if (!o || !canCompare(o) || !whatif) {
      compare = undefined;
      return;
    }
    const value = (o.v.type === 'check' ? o.value !== true : tipValue(o)) as ConfigValue;
    const key = `${o.v.var}:${String(o.value)}:${String(value)}`;
    if (compare?.key === key && compare.rev === rev) return;
    const asked = { key, rev, sections: compare?.key === key ? compare.sections : undefined };
    compare = asked;
    whatif(o.v.var, value)
      .then((sections) => {
        if (compare === asked) compare = { key, rev, sections };
      })
      .catch(() => {});
  });
</script>

<svelte:window onkeydown={onwindowkey} />

{#snippet option(o: OptionView, where: string)}
  {@const id = `${where}-${o.v.var}`}
  {@const described = tip?.id === id && tipOption ? 'config-option-tip' : undefined}
  <OptionRow
    modified={o.modified}
    inactive={!o.relevant}
    class={o.invalid ? 'text-worse' : ''}
    data-var={o.v.var}
    onmouseenter={enter(o, id)}
    onmouseleave={leave(id)}
    onfocusin={enter(o, id)}
    onfocusout={leave(id)}>
    {#snippet label()}<PobText text={o.v.label} plain={!o.relevant} />{/snippet}
    {#snippet control()}
      {#if o.v.type === 'check'}
        <input
          type="checkbox"
          class="check size-4 shrink-0"
          checked={o.value === true}
          aria-describedby={described}
          onchange={(e) => onset(o.v.var, e.currentTarget.checked)} />
      {:else if o.v.type === 'list'}
        {@const list = o.v.list}
        <Dropdown
          size="xs"
          class="max-w-37.5 shrink-0"
          aria-describedby={described}
          options={list.map((l, i) => ({ value: String(i), label: pobPlain(l.label) }))}
          value={String(listIndex(o.v, o.value))}
          onchange={(v: string) => onset(o.v.var, list[Number(v)].value as ConfigValue)}
          onhover={hoverEntry(o, id)} />
      {:else if o.v.type === 'text'}
        <input
          class="input input-xs w-37.5 shrink-0"
          value={String(o.value ?? '')}
          aria-describedby={described}
          onchange={(e) => onset(o.v.var, e.currentTarget.value)} />
      {:else}
        <NumberInput
          size="xs"
          class="w-20 shrink-0 font-mono"
          value={o.set && typeof o.value === 'number' ? o.value : null}
          step={o.v.type === 'float' ? 0.1 : 1}
          placeholder={o.placeholder}
          onchange={(n: number) => onset(o.v.var, n)}
          onclear={onunset && (() => onunset(o.v.var))} />
      {/if}
    {/snippet}
  </OptionRow>
{/snippet}

<div class="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-2">
  <input
    type="search"
    class="input h-8.5 w-60 bg-surface-1 px-2.5 text-14 max-phone:w-full"
    placeholder="Filter options"
    aria-label="Filter options"
    bind:this={searchBox}
    bind:value={filter.q} />
  <label class="check-label gap-2 text-13"><input type="checkbox" class="check" bind:checked={filter.modOnly} /> Modified only</label>
  <label class="check-label gap-2 text-13"
    ><input type="checkbox" class="check" bind:checked={filter.showAll} /> Show options that don’t affect this build</label>
  <div class="grow max-phone:hidden"></div>
  <span class="note" aria-live="polite">{view.summary}</span>
  <label class="field field-inline"
    >Config set
    <Select
      class="h-8.5 bg-surface-1"
      disabled={(sets.length < 2 && !onconfigsets) || !onconfigset}
      options={[...sets.map((s, i) => ({ value: String(i), label: s })), ...(onconfigsets ? [{ value: '-1', label: 'Manage config sets…' }] : [])]}
      value={String(activeSet)}
      onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
        const v = Number(e.currentTarget.value);
        if (v === -1) {
          e.currentTarget.value = String(activeSet);
          managing = true;
          return;
        }
        onconfigset?.(v);
      }} />
  </label>
  <Button size="md" variant="ghost" class="px-2.5" onclick={onreset}>Reset to defaults</Button>
</div>
<div class="scroll flex grow flex-col gap-2.5">
  <OptionGroup title="Most used" count="the same options as in their sections below · edit either" class="mb-0 shrink-0">
    <div class="grid grid-cols-4 gap-x-4 max-wide:grid-cols-2 max-phone:grid-cols-1">
      {#each view.most as o (o.v.var)}{@render option(o, 'most')}{/each}
    </div>
  </OptionGroup>
  <div class="columns-3 gap-2.5 max-desk:columns-2 max-phone:columns-1">
    {#each view.sections as s (s.name)}
      {#if s.rows.length}
        <OptionGroup
          title={s.name}
          collapsed={collapsed[s.name] && !/\S/.test(filter.q)}
          ontoggle={() => (collapsed[s.name] = !collapsed[s.name])}
          count={`${s.total} options${s.modified ? ` · ${s.modified} modified` : ''}`}
          more={!filter.showAll && s.hidden ? `+ ${s.hidden} more (hidden: only shown when the build uses them)` : undefined}>
          {#each s.rows as r, i (r.kind === 'option' ? r.v.var : `h${i}`)}
            {#if r.kind === 'heading'}
              <h4 class="border-b border-surface-2 px-2.5 pt-1.5 pb-1 text-11 font-semibold text-text-2"><PobText text={r.label} /></h4>
            {:else}
              {@render option(r, 'sec')}
            {/if}
          {/each}
        </OptionGroup>
      {/if}
    {/each}
  </div>
  <ConfigCustom groups={custom} selected={customSelected} ops={customOps} {revision} />
</div>

{#if tip && tipOption}
  <div
    role="tooltip"
    id="config-option-tip"
    class="tooltip pointer-events-none fixed top-(--y) left-(--x) z-float flex max-w-90 flex-col gap-1 rounded-md px-2.5 py-2 text-12 whitespace-pre-line shadow-tip"
    use:besideAnchor={tip.anchor}>
    <span class="font-semibold">{pobPlain(tipOption.v.label)}</span>
    {#if tooltipOf(tipOption)}<span class="text-text-2"><PobText text={tooltipOf(tipOption)!} /></span>{/if}
    {#each tipLines(tipOption) as l, k (k)}<span class={tipOption.invalid ? 'text-worse' : 'text-text-3'}>{l}</span>{/each}
    {#if compare?.sections?.length}
      <div class="mt-1 border-t border-line pt-1.5 whitespace-normal" data-whatif>
        <ImpactBlocks
          blocks={[
            {
              header: tipOption.v.type === 'check' ? 'Toggling this option will give you:' : 'Selecting this option will give you:',
              sections: compare.sections
            }
          ]} />
      </div>
    {/if}
  </div>
{/if}
{#if onconfigsets}
  <SetListDialog
    bind:open={managing}
    title="Manage Config Sets"
    noun="config set"
    {sets}
    current={activeSet}
    onactivate={(i: number) => onconfigset?.(i)}
    onop={onconfigsets} />
{/if}
