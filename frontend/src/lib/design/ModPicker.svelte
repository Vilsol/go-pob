<script lang="ts">
  import EmptyState from './EmptyState.svelte';
  import FloatingLabel from './FloatingLabel.svelte';
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import Select from './Select.svelte';
  import NumberInput from './NumberInput.svelte';
  import { tone as toneClass } from './tone';
  import type { AffixRow } from './fixtures/items';

  type Props = {
    open?: boolean;
    base: string;
    affixes: { tiers: string[]; sources: string[]; sorts: string[]; rows: AffixRow[] };
    onadd: (text: string) => void;
  };
  let { open = $bindable(false), base, affixes, onadd }: Props = $props();

  let q = $state('');
  let ilvl = $state(86);
  let tier = $state('Any');
  let source = $state('Any');
  let sel = $state(0);
  const rows = $derived(
    affixes.rows.filter(
      (r) =>
        r.text.toLowerCase().includes(q.trim().toLowerCase()) &&
        r.ilvl <= ilvl &&
        (tier === 'Any' || r.tier === 'T1') &&
        (source === 'Any' || r.source === source || (source === 'Influence' && !['Natural', 'Crafted'].includes(r.source)))
    )
  );
  const total = $derived(rows.reduce((a, r) => a + r.weight, 0));
  const picked = $derived(rows[Math.min(sel, rows.length - 1)]);
</script>

<Dialog bind:open title="Add a modifier" note="{base} · item level {ilvl}" sheet class="h-190 max-h-[calc(100vh-48px)] w-225 max-w-[calc(100vw-48px)]">
  <div class="flex items-end gap-2.5 border-b border-line px-5 py-3 max-phone:flex-wrap max-phone:px-3">
    <input
      type="search"
      class="input h-8.5 grow px-2.5 text-14 max-phone:basis-full"
      placeholder="Search modifier text"
      aria-label="Search modifiers"
      bind:value={q}
      oninput={() => (sel = 0)} />
    <label class="field gap-0.75 text-11 text-text-3">Max item level<NumberInput bind:value={ilvl} min={1} max={100} class="w-20" /></label>
    <FloatingLabel label="Tier" class="max-phone:flex-1"><Select class="h-8.5" options={affixes.tiers} bind:value={tier} /></FloatingLabel>
    <FloatingLabel label="Source" class="max-phone:flex-1"><Select class="h-8.5" options={affixes.sources} bind:value={source} /></FloatingLabel>
    <FloatingLabel label="Sort by" class="max-phone:flex-1"><Select class="h-8.5" options={affixes.sorts} /></FloatingLabel>
  </div>
  <div class="scroll grow text-13">
    <div aria-hidden="true" class="data-row data-head grid-cols-24 gap-2 px-5 py-2 tracking-caps max-phone:hidden">
      <span class="col-span-2">TIER</span><span class="col-span-10">MODIFIER</span><span class="col-span-3">SOURCE</span><span class="col-span-2 text-right"
        >WEIGHT</span
      ><span class="col-span-2 text-right">CHANCE</span><span class="col-span-5 text-right">AT MAX ROLL</span>
    </div>
    <div role="listbox" aria-label="Modifiers">
      {#each rows as r, i (r.text + r.tier)}
        <button
          role="option"
          class="data-row w-full grid-cols-24 gap-2 border-0 max-phone:grid-cols-4 max-phone:gap-y-0.5 max-phone:px-3 border-b border-surface-3 bg-transparent px-5 py-1.75 text-left font-sans text-13 text-text aria-selected:bg-surface-3 aria-selected:shadow-[inset_2px_0_0_var(--color-accent)]"
          aria-selected={r === picked}
          onclick={() => (sel = i)}
          ondblclick={() => {
            onadd(r.text);
            open = false;
          }}>
          <span class="col-span-2 font-mono text-text-2 max-phone:col-span-1">{r.tier}</span>
          <span class="col-span-10 flex flex-col max-phone:col-span-3"
            ><span class={['toned', toneClass(r.source === 'Crafted' ? 'crafted' : 'magic')]}>{r.text}</span><span class="note-sm in-aria-selected:text-text-2"
              >requires ilvl {r.ilvl}</span
            ></span>
          <span class="col-span-3 text-text-2 max-phone:col-span-1">{r.source}</span>
          <span class="col-span-2 text-right font-mono max-phone:col-span-1">{r.weight || '—'}</span>
          <span class="col-span-2 text-right font-mono max-phone:col-span-1">{r.weight ? `${((100 * r.weight) / total).toFixed(1)}%` : 'bench'}</span>
          <span class="col-span-5 text-right font-mono text-better max-phone:col-span-1">{r.impact}</span>
        </button>
      {:else}
        <EmptyState inset="row" text="No modifier matches." />
      {/each}
    </div>
  </div>
  <div class="flex items-center justify-between gap-2 border-t border-line px-5 py-3 max-phone:px-3">
    <span class="note truncate">{picked ? `Selected: ${picked.text}` : 'Nothing selected'}</span>
    <div class="flex gap-2">
      <DialogActions
        confirm="Add modifier"
        disabled={!picked}
        oncancel={() => (open = false)}
        onconfirm={() => {
          onadd(picked.text);
          open = false;
        }} />
    </div>
  </div>
</Dialog>
