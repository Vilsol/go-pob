<script lang="ts">
  import Badge from './Badge.svelte';
  import { rovingIndex } from './keys';
  import type { Snippet } from 'svelte';
  import type { Tone } from './tone';
  import { panelId, tabId } from './tab_ids';

  type Tab = { id: string; label: string; count?: string; badge?: { text: string; tone: Tone } };
  type Props = {
    tabs: Tab[];
    selected?: string;
    label: string;
    variant?: 'underline' | 'segmented' | 'chip' | 'vertical';
    fill?: boolean;
    id?: string;
    class?: string;
    panelClass?: string;
    panel?: Snippet<[string]>;
    onchange?: (id: string) => void;
  };
  let {
    tabs,
    selected = $bindable(tabs[0]?.id),
    label,
    variant = 'underline',
    fill = false,
    id,
    class: extra = variant === 'underline' ? 'border-b border-line px-3' : '',
    panelClass,
    panel,
    onchange
  }: Props = $props();

  const uid = $props.id();
  const base = $derived(id ?? uid);
  const btns = $state<HTMLButtonElement[]>([]);

  const pick = (to: string) => {
    selected = to;
    onchange?.(to);
  };

  function onkeydown(e: KeyboardEvent, i: number) {
    const to = rovingIndex(e, i, tabs.length, { orientation: variant === 'vertical' ? 'vertical' : 'horizontal' });
    if (to === undefined) return;
    e.preventDefault();
    pick(tabs[to].id);
    btns[to]?.focus();
  }

  const strip = { underline: 'tabs', segmented: 'seg', chip: 'flex gap-1', vertical: 'flex flex-col gap-0.5' };
  const btn = { underline: 'tab tab-sm', segmented: 'seg-btn', chip: 'chip', vertical: 'tab-v' };
</script>

<div role="tablist" aria-label={label} aria-orientation={variant === 'vertical' ? 'vertical' : undefined} class={[strip[variant], fill && 'w-full', extra]}>
  {#each tabs as t, i (t.id)}
    <button
      bind:this={btns[i]}
      type="button"
      role="tab"
      id={tabId(base, t.id)}
      class={[btn[variant], fill && 'flex-1 justify-center']}
      aria-selected={t.id === selected}
      aria-controls={panelId(base)}
      tabindex={t.id === selected ? 0 : -1}
      onclick={() => pick(t.id)}
      onkeydown={(e) => onkeydown(e, i)}>
      {t.label}{#if t.count}<span class={variant === 'vertical' ? 'font-mono text-11 font-normal text-text-3' : 'ml-1 font-mono text-11 text-text-2'}
          >{t.count}</span
        >{/if}{#if t.badge}<Badge variant="solid" size="sm" mono tone={t.badge.tone}>{t.badge.text}</Badge>{/if}
    </button>
  {/each}
</div>
{#if panel}
  <div role="tabpanel" id={panelId(base)} aria-labelledby={tabId(base, selected ?? '')} class={panelClass}>
    {@render panel(selected ?? '')}
  </div>
{/if}
