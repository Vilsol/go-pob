<script lang="ts">
  import Dialog from './Dialog.svelte';
  import DialogActions from './DialogActions.svelte';
  import NumberInput from './NumberInput.svelte';
  import Tabs from './Tabs.svelte';
  import { panelId, tabId } from './tab_ids';
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
</script>

<Dialog bind:open title="Settings" sheet note="Saved in this browser as you change them" class="h-170 max-h-[calc(100vh-24px)] w-195">
  <div class="flex min-h-0 flex-[1_1_0] max-phone:flex-col">
    <Tabs
      id={uid}
      variant="vertical"
      label="Settings sections"
      class="w-47.5 shrink-0 border-r border-line bg-inset px-2 py-2.5 max-phone:w-full max-phone:flex-row max-phone:overflow-x-auto max-phone:border-r-0 max-phone:border-b max-phone:[scrollbar-width:none] max-phone:[&>*]:shrink-0 max-phone:[&>*]:gap-1.5"
      tabs={sections.map((s) => ({ id: s.id, label: s.label, count: String(s.rows.length) }))}
      selected={cur.id}
      onchange={(id: string) => (section = id)} />
    <div role="tabpanel" id={panelId(uid)} aria-labelledby={tabId(uid, cur.id)} class="scroll flex min-w-0 flex-[1_1_0] flex-col px-4.5 pt-2 pb-4">
      {#each cur.rows as r (r.id)}
        {@const x = val(r)}
        {@const id = `${uid}-${r.id}`}
        <div
          class="data-row grid-cols-[minmax(0,1fr)_15rem] gap-3 rounded max-phone:grid-cols-1 max-phone:gap-1.5 px-2 py-2.25 hover:bg-row-hover"
          data-setting={r.id}>
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
              <select {id} class="select input-sm w-full px-1.5 text-12" value={x} onchange={(e) => onchange(r.id, e.currentTarget.value)}>
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
  {#snippet footer()}
    {#if sample !== undefined}<span aria-live="polite" class="font-mono text-11 text-text-3">Preview: Full DPS {preview}</span>{/if}
    <div class="grow"></div>
    <DialogActions
      cancel={oncancel ? 'Cancel' : false}
      oncancel={() => {
        oncancel?.();
        open = false;
      }}
      confirm={oncancel ? 'Save' : 'Done'}
      onconfirm={() => (open = false)}>
      {#snippet extra()}
        <button class="btn btn-md bg-transparent px-3 text-text-2" onclick={() => onreset(cur.id)}>Reset this section</button>
      {/snippet}
    </DialogActions>
  {/snippet}
</Dialog>
