<script lang="ts">
  import { tick } from 'svelte';

  type Option = { value: string; label: string; title?: string };
  type Props = {
    options: Option[];
    value?: string;
    size?: 'xs' | 'default' | 'lg';
    class?: string;
    disabled?: boolean;
    title?: string;
    'aria-label'?: string;
    'aria-describedby'?: string;
    onchange?: (value: string) => boolean | void;
    onhover?: (index: number | null, el: HTMLElement | null) => void;
  };
  let { options, value = $bindable(), size = 'default', class: extra = '', disabled = false, title, onchange, onhover, ...aria }: Props = $props();

  const uid = $props.id();
  const sizes = { xs: 'input-xs', default: '', lg: 'input-lg' };
  let open = $state(false);
  let active = $state(-1);
  let root: HTMLElement;
  let list = $state<HTMLElement>();
  const chosen = $derived(options.findIndex((o) => o.value === value));

  const optionEl = (i: number) => list?.querySelector<HTMLElement>(`#${uid}-o${i}`) ?? null;
  const moveTo = async (i: number) => {
    active = i;
    await tick();
    const el = optionEl(i);
    el?.scrollIntoView?.({ block: 'nearest' });
    onhover?.(i, el);
  };
  const show = async (at = chosen) => {
    open = true;
    await tick();
    await moveTo(Math.max(0, at));
  };
  const close = () => {
    open = false;
    active = -1;
    onhover?.(null, null);
  };
  const choose = (i: number) => {
    close();
    const o = options[i];
    if (!o || o.value === value || onchange?.(o.value) === false) return;
    value = o.value;
  };
  const onkeydown = (e: KeyboardEvent) => {
    if (disabled) return;
    if (!open) {
      if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) {
        e.preventDefault();
        void show();
      }
      return;
    }
    const last = options.length - 1;
    const to = { ArrowDown: Math.min(active + 1, last), ArrowUp: Math.max(active - 1, 0), Home: 0, End: last }[e.key];
    if (to !== undefined) {
      e.preventDefault();
      void moveTo(to);
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      choose(active);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      close();
    } else if (e.key === 'Tab') close();
  };
  const outside = (e: PointerEvent) => {
    if (open && !root.contains(e.target as Node)) close();
  };
</script>

<svelte:window onpointerdown={outside} />

<div bind:this={root} class={['relative inline-block', extra]}>
  <button
    type="button"
    role="combobox"
    class={['select h-full w-full truncate text-left', sizes[size]]}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-controls="{uid}-list"
    aria-activedescendant={open && active >= 0 ? `${uid}-o${active}` : undefined}
    title={title ?? options[chosen]?.title}
    {disabled}
    {...aria}
    onclick={() => (open ? close() : show())}
    {onkeydown}>{options[chosen]?.label ?? ''}</button>
  {#if open}
    <ul
      bind:this={list}
      id="{uid}-list"
      role="listbox"
      aria-label={aria['aria-label']}
      class="menu absolute top-full z-40 mt-1 max-h-80 min-w-full overflow-y-auto">
      {#each options as o, i (i)}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <li
          id="{uid}-o{i}"
          role="option"
          aria-selected={i === chosen}
          class={['menu-item cursor-pointer whitespace-nowrap', i === active && 'bg-surface-3']}
          title={o.title}
          onmouseenter={() => void moveTo(i)}
          onmouseleave={() => {
            if (active === i) onhover?.(null, null);
          }}
          onclick={(e) => {
            e.preventDefault();
            choose(i);
          }}>
          {o.label}
        </li>
      {/each}
    </ul>
  {/if}
</div>
