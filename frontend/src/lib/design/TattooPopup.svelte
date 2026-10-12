<script lang="ts">
  import type { TattooChoiceFx } from '$lib/skill_tree/tattoo';
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';
  import Select from './Select.svelte';
  import Checkbox from './Checkbox.svelte';

  // TreeTab.lua:841-990
  type Props = {
    open?: boolean;
    nodeName: string;
    choices: TattooChoiceFx[];
    selected?: number;
    tattoo?: string;
    count: number;
    countLines: { line: string; count: number }[];
    showLegacy?: boolean;
    onadd: (dn: string) => void;
    onreset: () => void;
    onclose: () => void;
  };
  let {
    open = false,
    nodeName,
    choices,
    selected = $bindable(0),
    tattoo,
    count,
    countLines,
    showLegacy = $bindable(false),
    onadd,
    onreset,
    onclose
  }: Props = $props();

  const value = $derived(choices[selected]?.dn ?? '');
  const chosen = $derived(choices[selected]);
</script>

<Dialog {open} {onclose} title="Replace modifier of {nodeName}" class="w-150 p-4.5">
  <div class="flex flex-col gap-2.5 text-13">
    {#if tattoo}<span>Replaced by <strong>{tattoo}</strong></span>{/if}
    {#if choices.length}
      <label class="flex items-center gap-2"
        ><span class="w-17 text-right">Modifier:</span>
        <Select
          class="w-80"
          aria-label="Modifier"
          {value}
          options={choices.map((c) => ({ value: c.dn, label: c.dn, title: c.descriptions.join('\n') }))}
          onchange={(e: Event & { currentTarget: HTMLSelectElement }) => (selected = choices.findIndex((c) => c.dn === e.currentTarget.value))} /></label>
      <div class="flex min-h-10 flex-col gap-0.5" aria-label="Modifier lines">
        {#each chosen?.descriptions ?? [] as l, i (i)}<span>{l}</span>{/each}
      </div>
    {:else}
      <span class="note">No {showLegacy ? '' : 'current '}tattoo fits this node{showLegacy ? '' : '; legacy ones may'}.</span>
    {/if}
    <div class="flex gap-1.5">
      <Button size="sm" variant="primary" disabled={!chosen} onclick={() => chosen && onadd(chosen.dn)}>Add</Button>
      <Button size="sm" variant="danger" onclick={onreset}>Reset Node</Button>
      <Button size="sm" variant="ghost" onclick={onclose}>Cancel</Button>
    </div>
    <div class="flex items-center gap-4">
      <span
        class={['note', count > 50 && 'text-worse']}
        aria-label="Tattoo count"
        title={countLines.map((l) => `(${l.count})${l.line}`).join('\n') || undefined}>Tattoo count: {count} / 50</span>
      <Checkbox compact bind:checked={showLegacy}>Show Legacy Tattoos</Checkbox>
    </div>
  </div>
</Dialog>
