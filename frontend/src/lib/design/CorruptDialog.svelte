<script lang="ts">
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import Dropdown from './Dropdown.svelte';
  import Tabs from './Tabs.svelte';
  import { panelId, tabId } from './tab_ids';
  import ItemTooltip from './ItemTooltip.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import PobText from './PobText.svelte';
  import { corruptOptions, sortByValue, type CorruptChoices, type CorruptMod, type CraftApi } from './crafting';
  import type { ImpactBlock, TipProps } from './items';

  // ItemsTab.lua:3121-3480
  type Props = { open?: boolean; craft: CraftApi };
  let { open = $bindable(false), craft }: Props = $props();

  const GLIMPSE = 'Glimpse of Chaos';
  let choices = $state.raw<CorruptChoices>();
  let stats = $state.raw<{ stat: string; label: string }[]>([]);
  const uid = $props.id();
  let tab = $state<'implicits' | 'rolls'>('implicits');
  let source = $state('Corrupted');
  let sort = $state('');
  let values = $state.raw<Record<string, Record<string, number>>>({});
  let sorting = $state(false);
  const none = () => ['', '', '', '', ''];
  let picks = $state<string[]>(none());
  let focus = $state(0);
  let rolls = $state<number[]>([]);
  let labels = $state<string[]>([]);
  let compare = $state.raw<ImpactBlock[]>([]);
  let preview = $state.raw<TipProps>();

  $effect(() => {
    if (!open) return;
    tab = 'implicits';
    source = 'Corrupted';
    sort = '';
    picks = none();
    void craft.corruptChoices?.().then((c) => {
      choices = c;
      if (c?.glimpse.length) source = GLIMPSE;
      rolls = (c?.rolls ?? []).map((r) => r.value);
      labels = (c?.rolls ?? []).map((r) => r.line);
    });
    void craft.sortStats().then((s) => (stats = s));
  });

  const sources = $derived([...(choices?.glimpse.length ? [GLIMPSE] : []), 'Corrupted', 'Scourge']);
  const count = $derived(
    source === 'Scourge' ? (choices?.unique ? 4 : 3) : source === 'Corrupted' && choices?.implicitsCannotBeChanged ? 0 : (choices?.maxImplicits ?? 2)
  );
  const typeOf = (i: number) => (source !== 'Scourge' ? source : i < 2 ? 'ScourgeUpside' : 'ScourgeDownside');
  const listOf = (type: string): CorruptMod[] =>
    (type === GLIMPSE
      ? choices?.glimpse
      : type === 'Corrupted'
        ? choices?.corrupted
        : type === 'ScourgeUpside'
          ? choices?.scourgeUpside
          : choices?.scourgeDownside) ?? [];
  const sorted = (type: string) => {
    const list = listOf(type);
    const v = sort ? values[type] : undefined;
    return sortByValue(v && list.map((m) => v[m.id] ?? 0), list.length).map((k) => list[k]);
  };

  $effect(() => {
    const stat = sort;
    const src = source;
    if (!stat || !choices) return;
    sorting = true;
    const types = src !== 'Scourge' ? [src] : ['ScourgeUpside', 'ScourgeDownside'];
    void Promise.all(
      types.map(async (type) => {
        const ids = listOf(type).map((m) => m.id);
        const v = (await craft.corruptSort?.(src, type, ids, stat)) ?? [];
        return [type, Object.fromEntries(ids.map((id, k) => [id, v[k] ?? 0]))] as const;
      })
    ).then((r) => {
      if (stat === sort && src === source) values = Object.fromEntries(r);
      sorting = false;
    });
  });

  const focused = $derived(listOf(typeOf(focus)).find((m) => m.id === picks[focus]));
  $effect(() => {
    const mod = focused;
    compare = [];
    if (!mod || !open) return;
    const tagged = mod.lines.map((l) => (mod.modTags.length ? `{tags:${mod.modTags.join(',')}}` : '') + l);
    void craft.modCompare?.(tagged).then((b) => {
      if (mod === focused) compare = b;
    });
  });

  const result = () => {
    if (tab === 'rolls')
      return craft.corruptText?.(
        source,
        [],
        [],
        (choices?.rolls ?? []).map((r, k) => ({ index: r.index, value: rolls[k] }))
      );
    const shown = picks.slice(0, count);
    return craft.corruptText?.(
      source,
      (source !== 'Scourge' ? shown : shown.slice(0, 2)).filter(Boolean),
      source !== 'Scourge' ? [] : shown.slice(2).filter(Boolean)
    );
  };
  $effect(() => {
    void [tab, source, picks.join('|'), rolls.join('|'), choices];
    if (!open || !choices) return;
    void result()?.then(async (t) => {
      preview = t === undefined ? undefined : await craft.tipOf?.(t);
    });
  });
</script>

<Dialog bind:open title="Corrupted Item" size="lg">
  <div class="flex flex-col gap-2.5 text-13">
    {#if choices?.unique}
      <Tabs
        id={uid}
        variant="segmented"
        label="Corruption"
        tabs={[
          { id: 'implicits', label: 'Implicits' },
          { id: 'rolls', label: 'Roll Ranges' }
        ]}
        selected={tab}
        onchange={(id) => (tab = id as 'implicits' | 'rolls')} />
    {/if}
    <div
      role={choices?.unique ? 'tabpanel' : undefined}
      id={choices?.unique ? panelId(uid) : undefined}
      aria-labelledby={choices?.unique ? tabId(uid, tab) : undefined}
      class="flex flex-col gap-2.5">
      {#if tab === 'implicits'}
        <div class="grid grid-cols-2 gap-3">
          <FloatingLabel label="Source"
            ><Select
              size="sm"
              aria-label="Source"
              options={sources}
              value={source}
              onchange={(e: Event & { currentTarget: HTMLSelectElement }) => {
                source = e.currentTarget.value;
                picks = none();
              }} /></FloatingLabel>
          <FloatingLabel label="Sort by"
            ><Select
              size="sm"
              aria-label="Sort by"
              aria-busy={sorting}
              options={[{ value: '', label: 'Default' }, ...stats.map((s) => ({ value: s.stat, label: s.label }))]}
              bind:value={sort} /></FloatingLabel>
        </div>
        {#if source === 'Corrupted' && choices?.implicitsCannotBeChanged}
          <p class="note">This Items Implicits Cannot Be Changed</p>
        {/if}
        {#each Array.from({ length: count }, (_, k) => k) as i (i)}
          {@const others = source === 'Scourge' ? [picks[i % 2 === 0 ? i + 1 : i - 1]] : picks.slice(0, count).filter((_, k) => k !== i)}
          <FloatingLabel label="Implicit #{i + 1}"
            ><Dropdown
              class="block h-7.5 w-full"
              searchable
              aria-label="Implicit #{i + 1}"
              options={[{ value: '', label: 'None' }, ...corruptOptions(sorted(typeOf(i)), others).map((m) => ({ value: m.id, label: m.label }))]}
              value={picks[i]}
              onchange={(v: string) => {
                picks[i] = v;
                focus = i;
              }} /></FloatingLabel>
        {/each}
        {#if focused}
          <div role="group" aria-label="Implicit comparison" class="flex flex-col gap-1 font-mono text-13">
            {#each focused.lines as l, k (k)}<span class="font-sans"><PobText text={'^7' + l} /></span>{/each}
            <ImpactBlocks blocks={compare} />
          </div>
        {/if}
      {:else}
        <div role="group" aria-label="Roll Ranges" class="flex flex-col gap-1.5">
          {#each choices?.rolls ?? [] as r, k (k)}
            <div class="flex items-center gap-2">
              <span class="w-9 font-mono text-12">{rolls[k].toFixed(2)}</span>
              <input
                type="range"
                class="w-20"
                min="0.78"
                max="1.22"
                step="0.01"
                aria-label="Roll of {r.line}"
                value={rolls[k]}
                onchange={async (e) => {
                  rolls[k] = Number(e.currentTarget.value);
                  labels[k] = (await craft.rangeLine?.(r.index, rolls[k])) ?? labels[k];
                }} />
              <span class="grow">{labels[k]}</span>
            </div>
          {/each}
        </div>
      {/if}
    </div>
    {#if preview}
      <div aria-label="Corrupted item" role="group" class="self-start"><ItemTooltip {...preview} /></div>
    {/if}
  </div>
  {#snippet actions()}
    <DialogActions
      confirm="Corrupted"
      oncancel={() => (open = false)}
      onconfirm={async () => {
        const t = await result();
        if (t !== undefined) craft.settext?.(t);
        open = false;
      }} />
  {/snippet}
</Dialog>
