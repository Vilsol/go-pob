<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import FloatingItemTip from './FloatingItemTip.svelte';
  import Segmented from './Segmented.svelte';
  import Select from './Select.svelte';
  import Checkbox from './Checkbox.svelte';
  import Button from './Button.svelte';
  import { tone as toneClass } from './tone';
  import type { BrowseKind, BrowseRow } from './fixtures/items';

  type Props = {
    kinds: BrowseKind[];
    added: string[];
    note?: string;
    onadd: (row: BrowseRow, equip: boolean) => void;
    oncraft: (row: BrowseRow) => void;
  };
  let { kinds, added, note = '', onadd, oncraft }: Props = $props();

  const uid = $props.id();
  let kind = $state(kinds[0].id);
  let q = $state('');
  let hover = $state<number | null>(null);
  const els: HTMLElement[] = $state([]);
  const b = $derived(kinds.find((k) => k.id === kind) ?? kinds[0]);
  const rows = $derived(b.rows.filter((r) => r.name.toLowerCase().includes(q.trim().toLowerCase())));
  const h = $derived(hover === null ? undefined : rows[hover]);
</script>

<div class="flex min-h-0 grow flex-col gap-2.5">
  <Segmented tabs chip label="Item type" options={kinds.map((k) => ({ id: k.id, label: k.label }))} bind:value={kind} onchange={() => (hover = null)} />
  <div class="flex min-h-0 grow gap-3.5">
    <aside aria-label="Filters" class="panel flex w-62.5 shrink-0 flex-col gap-2.5 self-start p-3 text-12">
      <input class="input" type="search" placeholder={b.search} aria-label="Search {b.title}" bind:value={q} />
      {#each b.filters as f (f.label)}
        <FloatingLabel label={f.label}><Select class="h-7.5 text-12" options={f.options} /></FloatingLabel>
      {/each}
      {#each b.checks as c (c.label)}
        <Checkbox compact checked={c.on}>{c.label}</Checkbox>
      {/each}
      <span class="text-text-3">{b.note}</span>
    </aside>
    <section aria-label={b.title} class="panel relative min-w-0 grow self-start overflow-hidden text-12">
      <div role="table" aria-label="{b.title} list">
        <div role="row" class="data-row data-head grid-cols-24 py-2">
          <span role="columnheader" class="col-span-7">ITEM</span><span role="columnheader" class="col-span-4">TYPE</span><span
            role="columnheader"
            class="col-span-6">{b.col}</span
          ><span role="columnheader" class="col-span-2 text-right">REQ.</span><span role="columnheader" class="col-span-5"
            ><span class="sr-only">Actions</span></span>
        </div>
        {#each rows as r, i (i)}
          {@const isAdded = added.includes(r.name)}
          <!-- svelte-ignore a11y_interactive_supports_focus -->
          <div
            role="row"
            class={['data-row grid-cols-24 py-1.5', hover === i && 'bg-hl']}
            onmouseenter={() => (hover = i)}
            onmouseleave={() => (hover = null)}
            onfocusin={() => (hover = i)}
            onfocusout={() => (hover = null)}>
            <span bind:this={els[i]} role="cell" data-pob-color class={['toned col-span-7 truncate font-semibold', toneClass(r.rarity)]}>{r.name}</span>
            <span role="cell" class="col-span-4 truncate text-text-2">{r.type}</span>
            <span role="cell" class="col-span-6 truncate text-text-3"
              >{r.note}{#if r.impact}&nbsp;·&#32;<span class={['toned', toneClass(r.impact.dir)]}>{r.impact.text}</span>{/if}</span>
            <span role="cell" class="col-span-2 text-right font-mono text-text-2">{r.req}</span>
            <span role="cell" class="col-span-5 flex justify-end gap-1">
              {#if r.action === 'craft'}
                <Button size="xs" class="text-accent" aria-haspopup="dialog" title="Open the item editor with this base" onclick={() => oncraft(r)}
                  >Craft from base</Button>
              {:else}
                <Button
                  size="xs"
                  variant="ghost"
                  disabled={isAdded}
                  aria-describedby={hover === i ? `${uid}-tip` : undefined}
                  title="Add to this build’s stash"
                  onclick={() => onadd(r, false)}>{isAdded ? 'Added ✓' : 'Add'}</Button>
                <Button size="xs" variant="primary" title="Add and equip in {r.type}" onclick={() => onadd(r, true)}>Equip</Button>
              {/if}
            </span>
          </div>
        {/each}
      </div>
      <div role="status" class={note ? 'border-t border-line bg-better/8 px-3 py-1.5 text-12 text-better' : 'sr-only'}>{note}</div>
    </section>
  </div>
</div>
{#if h?.impact && hover !== null}
  <FloatingItemTip
    anchor={els[hover]}
    id="{uid}-tip"
    item={h.name}
    base={h.note}
    rarity={h.rarity}
    mods={[]}
    impactTitle="EQUIP IMPACT (IN {h.type.toUpperCase()})"
    impact={[h.impact]} />
{/if}
