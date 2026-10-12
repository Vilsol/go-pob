<script lang="ts">
  import type { Snippet } from 'svelte';
  import { tone as toneClass, type Tone } from './tone';

  type Tab = { id: string; label: string; badge?: { text: string; tone: Tone } };
  type Props = { tabs: Tab[]; selected?: string; label: string; class?: string; panelClass?: string; panel: Snippet<[string]> };
  let { tabs, selected = $bindable(tabs[0]?.id), label, class: extra = 'border-b border-line px-3', panelClass, panel }: Props = $props();

  const uid = $props.id();
  const btns = $state<HTMLButtonElement[]>([]);

  function onkeydown(e: KeyboardEvent, i: number) {
    const n = tabs.length;
    const to = { ArrowRight: (i + 1) % n, ArrowLeft: (i - 1 + n) % n, Home: 0, End: n - 1 }[e.key];
    if (to === undefined) return;
    e.preventDefault();
    selected = tabs[to].id;
    btns[to]?.focus();
  }
</script>

<div role="tablist" aria-label={label} class={['tabs', extra]}>
  {#each tabs as t, i (t.id)}
    <button
      bind:this={btns[i]}
      role="tab"
      id="{uid}-tab-{t.id}"
      class="tab tab-sm"
      aria-selected={t.id === selected}
      aria-controls="{uid}-panel"
      tabindex={t.id === selected ? 0 : -1}
      onclick={() => (selected = t.id)}
      onkeydown={(e) => onkeydown(e, i)}>
      {t.label}{#if t.badge}<span class={['count-badge', toneClass(t.badge.tone)]}>{t.badge.text}</span>{/if}
    </button>
  {/each}
</div>
<div role="tabpanel" id="{uid}-panel" aria-labelledby="{uid}-tab-{selected}" class={panelClass}>
  {@render panel(selected ?? '')}
</div>
