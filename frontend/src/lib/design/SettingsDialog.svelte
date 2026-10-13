<script lang="ts">
  import Dialog from './Dialog.svelte';
  import NumberInput from './NumberInput.svelte';
  import type { Setting, SettingsSection } from './fixtures/app';

  // Main.lua:1166-1225
  type Value = Setting['value'];
  type Props = {
    open?: boolean;
    sections: SettingsSection[];
    values: Record<string, Value>;
    sample?: number;
    section?: string;
    onchange: (id: string, value: Value) => void;
    onreset: (section: string) => void;
    oncancel?: () => void;
    onaction?: (id: string) => void;
  };
  let { open = $bindable(false), sections, values, sample, section = $bindable(sections[0]?.id), onchange, onreset, oncancel, onaction }: Props = $props();

  const uid = $props.id();
  const tabs = $state<HTMLButtonElement[]>([]);
  let asking = $state<string>();
  const drafts = $state<Record<string, number>>({});
  const val = (r: Setting) => values[r.id] ?? r.value;
  const cur = $derived(sections.find((s) => s.id === section) ?? sections[0]);
  const byId = (id: string) => sections.flatMap((s) => s.rows).find((r) => r.id === id);
  const preview = $derived.by(() => {
    const on = byId('thousands');
    const sep = byId('thousandsSep');
    const s = sep ? String(val(sep)) : ',';
    return String(sample).replace(/\B(?=(\d{3})+(?!\d))/g, on && val(on) ? (s === 'space' ? ' ' : s) : '');
  });

  function onkeydown(e: KeyboardEvent, i: number) {
    const n = sections.length;
    const to = { ArrowDown: (i + 1) % n, ArrowUp: (i - 1 + n) % n, Home: 0, End: n - 1 }[e.key];
    if (to === undefined) return;
    e.preventDefault();
    section = sections[to].id;
    tabs[to]?.focus();
  }
</script>

<Dialog bind:open title="Settings" sheet note="Saved in this browser as you change them" class="h-170 max-h-[calc(100vh-24px)] w-195">
  <div class="flex min-h-0 flex-[1_1_0]">
    <div
      role="tablist"
      aria-orientation="vertical"
      aria-label="Settings sections"
      class="flex w-47.5 shrink-0 flex-col gap-0.5 border-r border-line bg-inset px-2 py-2.5">
      {#each sections as s, i (s.id)}
        <button
          bind:this={tabs[i]}
          role="tab"
          id="{uid}-tab-{s.id}"
          aria-selected={s.id === cur.id}
          aria-controls="{uid}-panel"
          tabindex={s.id === cur.id ? 0 : -1}
          class="flex h-8.5 items-center justify-between rounded border-0 bg-transparent px-2.5 text-left font-sans text-13 text-text-2 aria-selected:bg-surface-3 aria-selected:font-semibold aria-selected:text-text aria-selected:shadow-[inset_2px_0_0_var(--color-accent)]"
          onclick={() => (section = s.id)}
          onkeydown={(e) => onkeydown(e, i)}>{s.label}<span class="font-mono text-11 font-normal text-text-3">{s.rows.length}</span></button>
      {/each}
    </div>
    <div role="tabpanel" id="{uid}-panel" aria-labelledby="{uid}-tab-{cur.id}" class="scroll flex min-w-0 flex-[1_1_0] flex-col px-4.5 pt-2 pb-4">
      {#each cur.rows as r (r.id)}
        {@const x = val(r)}
        {@const id = `${uid}-${r.id}`}
        <div class="data-row grid-cols-[minmax(0,1fr)_240px] gap-3 rounded px-2 py-2.25 hover:bg-bar" data-setting={r.id}>
          <span class="flex min-w-0 flex-col gap-0.5"
            >{#if r.kind === 'number' || r.kind === 'info'}<span class="text-13">{r.label}</span>{:else}<label for={id} class="text-13">{r.label}</label
              >{/if}{#if r.hint}<span class="note-sm">{r.hint}</span>{/if}</span>
          <span class="flex min-w-0 items-center gap-2">
            {#if r.kind === 'range'}
              <input
                {id}
                type="range"
                min={r.min}
                max={r.max}
                step={r.step}
                value={x}
                class="min-w-0 flex-[1_1_0] accent-accent"
                oninput={(e) => (r.onRelease ? (drafts[r.id] = Number(e.currentTarget.value)) : onchange(r.id, Number(e.currentTarget.value)))}
                onchange={(e) => {
                  if (!r.onRelease) return;
                  delete drafts[r.id];
                  onchange(r.id, Number(e.currentTarget.value));
                }} />
              <span class="w-11 text-right font-mono text-12 text-text-2">{drafts[r.id] ?? x}{r.unit}</span>
            {:else if r.kind === 'check'}
              <input {id} type="checkbox" class="check" checked={!!x} onchange={() => onchange(r.id, !x)} />
            {:else if r.kind === 'select'}
              <select {id} class="select h-7.5 w-full px-1.5 text-12" value={x} onchange={(e) => onchange(r.id, e.currentTarget.value)}>
                {#each r.opts ?? [] as o (o)}<option value={o}>{o}</option>{/each}
              </select>
            {:else if r.kind === 'number'}
              <NumberInput value={Number(x)} min={r.min} max={r.max} label={r.label} class="w-24" onchange={(v: number) => onchange(r.id, v)} />
            {:else if r.kind === 'color'}
              <input
                {id}
                type="color"
                class="h-7 w-9 rounded border border-line-strong bg-ground p-0"
                value={x}
                oninput={(e) => onchange(r.id, e.currentTarget.value)} />
              <span class="font-mono text-12 text-text-2">{x}</span>
            {:else if r.kind === 'info'}
              <span {id} class="font-mono text-12 text-text-2">{x}</span>
            {:else if asking === r.id}
              <span role="alert" class="note-sm min-w-0">{r.confirm}</span>
              <button class="btn btn-sm" onclick={() => (asking = undefined)}>Keep</button>
              <button
                class={['btn btn-sm', r.danger && 'border-warn-line text-worse']}
                onclick={() => {
                  asking = undefined;
                  onaction?.(r.id);
                }}>{r.label}</button>
            {:else}
              <button {id} class={['btn btn-sm', r.danger && 'border-warn-line text-worse']} onclick={() => (r.confirm ? (asking = r.id) : onaction?.(r.id))}
                >{x}</button>
            {/if}
          </span>
        </div>
      {/each}
    </div>
  </div>
  <div class="flex items-center gap-2 border-t border-line bg-bar px-4.5 py-3">
    {#if sample !== undefined}<span aria-live="polite" class="font-mono text-11 text-text-3">Preview: Full DPS {preview}</span>{/if}
    <div class="grow"></div>
    <button class="btn btn-md bg-transparent px-3 text-text-2" onclick={() => onreset(cur.id)}>Reset this section</button>
    {#if oncancel}
      <button
        class="btn btn-md px-3"
        onclick={() => {
          oncancel();
          open = false;
        }}>Cancel</button>
    {/if}
    <button class="btn btn-md btn-primary px-4" onclick={() => (open = false)}>{oncancel ? 'Save' : 'Done'}</button>
  </div>
</Dialog>
