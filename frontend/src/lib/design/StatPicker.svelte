<script lang="ts">
  import { statMatches, type TimelessStat } from './timeless';

  type Props = { stats: TimelessStat[]; label: string; placeholder?: string; class?: string; onpick: (s: TimelessStat) => void };
  let { stats, label, placeholder = 'Search stats…', class: extra = '', onpick }: Props = $props();

  const uid = $props.id();
  const MAX = 60;
  let q = $state('');
  let open = $state(false);
  let active = $state(0);
  let list = $state<HTMLElement>();
  const found = $derived(statMatches(stats, q));
  const shown = $derived(found.slice(0, MAX));

  $effect(() => {
    void q;
    active = 0;
  });

  const pick = (s: TimelessStat | undefined) => {
    if (!s) return;
    onpick(s);
    q = '';
    open = false;
  };
  const move = (by: number) => {
    open = true;
    active = Math.min(Math.max(active + by, 0), shown.length - 1);
    list?.querySelector(`#${uid}-o${active}`)?.scrollIntoView({ block: 'nearest' });
  };
  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      move(e.key === 'ArrowDown' ? 1 : -1);
    } else if (e.key === 'Enter' && open) {
      e.preventDefault();
      pick(shown[active]);
    } else if (e.key === 'Escape' && open) {
      e.preventDefault();
      e.stopPropagation();
      open = false;
    }
  }
</script>

<div class={['relative', extra]}>
  <input
    class="input h-8 w-full px-2.5"
    type="text"
    role="combobox"
    aria-label={label}
    aria-expanded={open}
    aria-controls="{uid}-list"
    aria-autocomplete="list"
    aria-activedescendant={open && shown.length ? `${uid}-o${active}` : undefined}
    autocomplete="off"
    {placeholder}
    bind:value={q}
    onfocus={() => (open = true)}
    oninput={() => (open = true)}
    onblur={() => (open = false)}
    {onkeydown} />
  {#if open}
    <div
      bind:this={list}
      id="{uid}-list"
      role="listbox"
      aria-label="{label} options"
      class="menu absolute top-full z-dropdown mt-1 max-h-80 w-full overflow-y-auto">
      {#each shown as s, i (i)}
        <div
          id="{uid}-o{i}"
          role="option"
          tabindex="-1"
          aria-selected={i === active}
          class={['menu-item flex h-auto min-h-7.5 cursor-pointer flex-col items-start justify-center gap-0.5 py-1', i === active && 'bg-surface-3']}
          onmousedown={(e) => {
            e.preventDefault();
            pick(s);
          }}
          onmouseenter={() => (active = i)}>
          <span>{s.label}</span>
          {#if s.descriptions.length}<span class="note-sm whitespace-pre-line">{s.descriptions.join('\n')}</span>{/if}
        </div>
      {:else}
        <div class="menu-item text-text-3">No stat matches</div>
      {/each}
      {#if found.length > MAX}<div class="menu-item note-sm">{found.length - MAX} more — type to narrow</div>{/if}
    </div>
  {/if}
</div>
