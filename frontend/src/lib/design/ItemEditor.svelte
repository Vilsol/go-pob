<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import DialogActions from './DialogActions.svelte';
  import Select from './Select.svelte';
  import NumberInput from './NumberInput.svelte';
  import Checkbox from './Checkbox.svelte';
  import ModRow from './ModRow.svelte';
  import ModPicker from './ModPicker.svelte';
  import ImpactBlocks from './ImpactBlocks.svelte';
  import { tone as toneClass } from './tone';
  import { impactRows, modTone, type WhatIf } from './items';
  import type { ItemSummary, ModLine, StatDiff } from './fixtures';
  import type { AffixRow } from './fixtures/items';

  // ItemsTab.lua:988-993
  type Mods = { implicits: ModLine[]; explicits: ModLine[] };
  type Props = {
    open?: boolean;
    item: ItemSummary;
    slot?: string;
    impact?: StatDiff;
    isNew?: boolean;
    affixes: { tiers: string[]; sources: string[]; sorts: string[]; rows: AffixRow[] };
    readonly?: string;
    whatIf?: WhatIf;
    onedittext?: () => void;
    modTable?: boolean;
    onmodtable?: () => void;
    onclone?: () => void;
    onsave?: (item: ItemSummary, equip: boolean) => void;
    onclose?: () => void;
  };
  let {
    open = $bindable(false),
    item,
    slot,
    impact,
    isNew = false,
    affixes,
    readonly,
    whatIf,
    onedittext,
    modTable,
    onmodtable,
    onclone,
    onsave,
    onclose
  }: Props = $props();

  const RARITIES = ['Normal', 'Magic', 'Rare', 'Unique', 'Relic'];
  const start = (): Mods => ({ implicits: [...item.implicits], explicits: [...item.explicits] });
  let cur = $state<Mods>(start());
  let past = $state<Mods[]>([]);
  let future = $state<Mods[]>([]);
  let picker = $state(false);
  let current = $state<string | null>(null);
  let ilvl = $state(84);
  let quality = $state(20);
  let rarity = $state(item.rarity[0].toUpperCase() + item.rarity.slice(1));

  const commit = (next: Mods) => {
    past = [...past, cur];
    future = [];
    cur = next;
  };
  const undo = () => {
    if (!past.length) return;
    future = [cur, ...future];
    cur = past[past.length - 1];
    past = past.slice(0, -1);
  };
  const redo = () => {
    if (!future.length) return;
    past = [...past, cur];
    cur = future[0];
    future = future.slice(1);
  };
  function onkeydown(e: KeyboardEvent) {
    if (!(e.ctrlKey || e.metaKey) || picker || readonly) return;
    const k = e.key.toLowerCase();
    if (k === 'z' && !e.shiftKey) undo();
    else if ((k === 'z' && e.shiftKey) || k === 'y') redo();
    else return;
    e.preventDefault();
  }
  const saved = (): ItemSummary => ({ ...item, ...cur, rarity: rarity.toLowerCase() as ItemSummary['rarity'] });
  const sections = $derived([
    { title: 'IMPLICIT', key: 'implicits' as const, lines: cur.implicits },
    { title: 'EXPLICIT', key: 'explicits' as const, lines: cur.explicits }
  ]);
  const rows = $derived(impactRows(impact, 6));
</script>

<Dialog
  bind:open
  {onclose}
  title="{readonly ? 'Item' : isNew ? 'Craft' : 'Edit item'} {item.name}"
  sheet
  class="max-h-[calc(100vh-48px)] w-245 max-w-[calc(100vw-48px)]">
  {#snippet head()}
    <Button
      size="sm"
      disabled={readonly !== undefined && !onedittext}
      title={onedittext ? 'Copy its text into the paste panel, where it is parsed, compared and saved' : undefined}
      onclick={() => {
        if (!onedittext) return;
        onedittext();
        open = false;
      }}>Edit as text</Button>
    <Button
      size="sm"
      aria-haspopup="dialog"
      disabled={readonly !== undefined && !(onmodtable && modTable)}
      title={readonly === undefined ? undefined : !onmodtable ? readonly : modTable ? undefined : 'PoB adds modifiers to magic and rare items only'}
      onclick={() => {
        if (readonly === undefined) picker = true;
        else if (onmodtable) {
          onmodtable();
          open = false;
        }
      }}>Mod table</Button>
    <Button
      size="sm"
      disabled={!onclone}
      title={onclone ? 'Open a copy in the paste panel, saved as a new item' : readonly}
      onclick={() => {
        onclone?.();
        open = false;
      }}>Clone as new item</Button>
    {#if !readonly}
      <div role="group" aria-label="Item editor history" class="ml-auto flex items-center gap-1.5 rounded-md border border-line bg-surface-1 px-1.5 py-0.75">
        <span class="text-11 text-text-3">Item history</span>
        <Button size="xs" variant="ghost" disabled={!past.length} aria-label="Undo item edit (Ctrl+Z)" onclick={undo}>↶ Undo</Button>
        <Button size="xs" variant="ghost" disabled={!future.length} aria-label="Redo item edit (Ctrl+Shift+Z)" onclick={redo}>↷ Redo</Button>
        <span aria-live="polite" class="font-mono text-11 text-text-2">{past.length} / {past.length + future.length}</span>
      </div>
    {:else}
      <span class="note ml-auto">{readonly}</span>
    {/if}
  {/snippet}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="scroll flex gap-4 p-4 max-phone:flex-1 max-phone:flex-col max-phone:p-3" {onkeydown}>
    <div class="flex min-w-0 flex-[1_1_0] flex-col gap-3">
      <section aria-label="Item" class={['overflow-hidden rounded-md border border-(--tone)/33 bg-inset text-13', toneClass(item.rarity)]}>
        <div class="item-head px-3.5 py-2.5">
          <span data-pob-color class="toned text-16 font-semibold">{item.name}</span><span class="text-12 text-text-2">{item.base} · {rarity}</span>
        </div>
        {#each sections as sec (sec.key)}
          <div class="flex flex-col gap-0.5 border-t border-line px-3.5 py-2">
            <div class="flex justify-between text-11 tracking-caps text-text-3"><span>{sec.title}</span><span>{sec.lines.length}</span></div>
            {#each sec.lines as m, i (i)}
              <div class="group -mx-1.5 flex items-center gap-1 rounded pr-1.5 hover:bg-surface-2">
                <span class="grid min-w-0 grow">
                  {#if readonly}
                    <span class={['flex justify-between px-1.5 py-1.25 font-sans text-13 text-(--tone)', toneClass(modTone(m.kind))]}
                      ><span>{m.text}</span><span class="note-sm font-mono">{m.kind === 'magic' ? '' : m.kind}</span></span>
                  {:else}
                    <ModRow
                      text={m.text}
                      meta={m.kind === 'magic' ? '' : m.kind}
                      tone={modTone(m.kind)}
                      current={current === `${sec.key}${i}`}
                      onclick={() => (current = `${sec.key}${i}`)} />
                  {/if}
                </span>
                {#if !readonly}<button
                    class="size-5.5 shrink-0 rounded border border-danger-line bg-danger-bg p-0 font-sans text-14 leading-none font-semibold text-error opacity-0 group-focus-within:opacity-100 group-hover:opacity-100 focus:opacity-100"
                    aria-label="Remove {m.text}"
                    title="Remove modifier"
                    onclick={() => commit({ ...cur, [sec.key]: cur[sec.key].filter((_, j) => j !== i) })}>×</button
                  >{/if}
              </div>
            {/each}
            {#if sec.key === 'explicits' && !readonly}
              <button class="btn-bare self-start py-1 text-12 text-accent" aria-haspopup="dialog" onclick={() => (picker = true)}>+ Add modifier</button>
            {/if}
          </div>
        {/each}
      </section>
      <section
        aria-label="Impact"
        aria-busy={whatIf?.pending}
        data-slot={whatIf?.key}
        class="flex flex-col gap-1 rounded-md border border-line bg-inset px-3.5 py-2.5 font-mono text-13">
        {#if whatIf}
          <ImpactBlocks blocks={whatIf.blocks} />
          {#if whatIf.pending}<div class="font-sans text-12 text-text-3">Calculating…</div>
          {:else if !whatIf.blocks.length}<div class="font-sans text-12 text-text-2">Fits no slot of the active item set</div>{/if}
        {:else}
          <div class="font-sans text-12 text-text-2">
            {#if impact}{impact.header}{:else if slot}No what-if for {slot} yet{:else}Fits no equipment slot{/if}
          </div>
          {#each rows as d, i (i)}
            <div class="kv"><span>{d.label}</span><span class={['toned', toneClass(d.dir)]}>{d.text}</span></div>
          {/each}
        {/if}
      </section>
    </div>
    <section aria-label="Item properties" class="flex flex-[0_0_300px] flex-col gap-2.5 max-phone:flex-none">
      <h3 class="caps">Properties</h3>
      {#if readonly}
        <div class="flex flex-col gap-1 text-13">
          <div class="kv"><span class="text-text-2">Base type</span><span>{item.base}</span></div>
          <div class="kv">
            <span class="text-text-2">Rarity</span><span data-pob-color class={['text-(--tone)', toneClass(item.rarity)]}>{rarity}</span>
          </div>
          {#if item.sockets}<div class="kv"><span class="text-text-2">Sockets</span><span class="font-mono">{item.sockets}</span></div>{/if}
        </div>
      {:else}
        <FloatingLabel label="Base type"><Select class="h-8.5" options={[item.base]} /></FloatingLabel>
        <div class="grid grid-cols-2 gap-2">
          <label class="field">Item level<NumberInput bind:value={ilvl} min={1} max={100} /></label>
          <label class="field">Quality<NumberInput bind:value={quality} min={0} max={30} /></label>
        </div>
        <FloatingLabel label="Rarity"
          ><Select
            data-pob-color
            class={['h-8.5', toneClass(rarity.toLowerCase() as ItemSummary['rarity'])]}
            options={RARITIES}
            bind:value={rarity} /></FloatingLabel>
        <FloatingLabel label="Influence"
          ><Select class="h-8.5" options={['None', 'Shaper', 'Elder', 'Crusader', 'Redeemer', 'Hunter', 'Warlord']} /></FloatingLabel>
        <Checkbox>Corrupted</Checkbox>
      {/if}
    </section>
  </div>
  <div class="flex items-center gap-2 border-t border-line bg-bar px-4 py-3 max-phone:flex-wrap max-phone:px-3">
    <span class="note max-phone:hidden">{readonly ? 'Esc closes' : 'Esc closes · Ctrl+Z / Ctrl+Shift+Z undo and redo inside the editor only'}</span>
    <div class="grow"></div>
    <DialogActions
      cancel={readonly ? 'Close' : 'Cancel'}
      oncancel={() => (open = false)}
      confirm={readonly ? undefined : slot ? `Save and equip in ${slot}` : 'Save to stash'}
      onconfirm={() => {
        onsave?.(saved(), !!slot);
        open = false;
      }}>
      {#snippet extra()}
        {#if !readonly && slot}
          <Button
            size="md"
            onclick={() => {
              onsave?.(saved(), false);
              open = false;
            }}>Save to stash</Button>
        {/if}
      {/snippet}
    </DialogActions>
  </div>
  <ModPicker bind:open={picker} base={item.base} {affixes} onadd={(text) => commit({ ...cur, explicits: [...cur.explicits, { text, kind: 'magic' }] })} />
</Dialog>
