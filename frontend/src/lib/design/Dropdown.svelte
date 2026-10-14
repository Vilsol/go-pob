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
    'aria-busy'?: boolean;
    onchange?: (value: string) => boolean | void;
    onhover?: (index: number | null, el: HTMLElement | null) => void;
    searchable?: boolean;
  };
  let {
    options,
    value = $bindable(),
    size = 'default',
    class: extra = '',
    disabled = false,
    title,
    onchange,
    onhover,
    searchable = false,
    ...aria
  }: Props = $props();

  const uid = $props.id();
  const sizes = { xs: 'input-xs', default: '', lg: 'input-lg' };
  let open = $state(false);
  let active = $state(-1);
  let root: HTMLElement;
  let list = $state<HTMLElement>();
  const chosen = $derived(options.findIndex((o) => o.value === value));
  let q = $state('');
  const shown = $derived.by(() => {
    const words = q.toLowerCase().split(/\s+/).filter(Boolean);
    return options.flatMap((o, i) => (words.every((w) => o.label.toLowerCase().includes(w)) ? [i] : []));
  });

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
    q = '';
    onhover?.(null, null);
  };
  const filter = async (to: string) => {
    q = to;
    open = true;
    await tick();
    const exact = shown.find((i) => options[i].label.toLowerCase() === q.trim().toLowerCase());
    const first = exact ?? (shown.includes(active) ? active : shown[0]);
    if (first === undefined) {
      active = -1;
      onhover?.(null, null);
    } else await moveTo(first);
  };
  const choose = (i: number) => {
    close();
    const o = options[i];
    if (!o || o.value === value || onchange?.(o.value) === false) return;
    value = o.value;
  };
  const printable = (e: KeyboardEvent) => e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey;
  const onkeydown = (e: KeyboardEvent, inInput = false) => {
    if (disabled) return;
    if (searchable && !inInput && printable(e) && (e.key !== ' ' || q)) {
      e.preventDefault();
      void filter(q + e.key);
      return;
    }
    if (searchable && !inInput && e.key === 'Backspace' && q) {
      e.preventDefault();
      void filter(q.slice(0, -1));
      return;
    }
    if (inInput && e.key === ' ') return;
    if (!open) {
      if (['ArrowDown', 'ArrowUp', 'Enter', ' '].includes(e.key)) {
        e.preventDefault();
        void show();
      }
      return;
    }
    const at = shown.indexOf(active);
    const last = shown.length - 1;
    const to = { ArrowDown: Math.min(at + 1, last), ArrowUp: Math.max(at - 1, 0), Home: 0, End: last }[e.key];
    if (to !== undefined) {
      e.preventDefault();
      if (shown[to] !== undefined) void moveTo(shown[to]);
    } else if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      if (shown.includes(active)) choose(active);
    } else if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      if (q) void filter('');
      else close();
    } else if (e.key === 'Tab') close();
  };
  const outside = (e: PointerEvent) => {
    if (open && !root.contains(e.target as Node)) close();
  };

  const placed = (panel: HTMLElement) => {
    const pad = 8;
    const gap = 4;
    const place = () => {
      const r = root.getBoundingClientRect();
      panel.style.maxHeight = '';
      const natural = panel.offsetHeight;
      const below = innerHeight - r.bottom - gap - pad;
      const above = r.top - gap - pad;
      const down = below >= natural || below >= above;
      const h = Math.min(natural, Math.max(0, down ? below : above));
      panel.style.maxHeight = `${h}px`;
      panel.style.minWidth = `${Math.min(r.width, innerWidth - 2 * pad)}px`;
      const x = Math.min(r.left, innerWidth - pad - panel.offsetWidth);
      panel.style.setProperty('--x', `${Math.max(pad, x)}px`);
      panel.style.setProperty('--y', `${down ? r.bottom + gap : r.top - gap - h}px`);
    };
    place();
    const ro = typeof ResizeObserver === 'undefined' ? undefined : new ResizeObserver(place);
    ro?.observe(panel);
    const moved = (e: Event) => {
      if (!panel.contains(e.target as Node)) place();
    };
    addEventListener('scroll', moved, true);
    addEventListener('resize', place);
    return {
      destroy: () => {
        ro?.disconnect();
        removeEventListener('scroll', moved, true);
        removeEventListener('resize', place);
      }
    };
  };
</script>

<svelte:window onpointerdown={outside} />

<div bind:this={root} class={['relative inline-block', extra]}>
  <button
    type="button"
    role="combobox"
    class={['select max-h-full min-h-full w-full truncate text-left', sizes[size]]}
    aria-haspopup="listbox"
    aria-expanded={open}
    aria-controls="{uid}-list"
    aria-activedescendant={open && active >= 0 ? `${uid}-o${active}` : undefined}
    title={title ?? options[chosen]?.title}
    {disabled}
    {...aria}
    onclick={() => (open ? close() : show())}
    onkeydown={(e) => onkeydown(e)}>{options[chosen]?.label ?? ''}</button>
  {#if open}
    <div class="menu fixed top-(--y) right-auto left-(--x) w-max max-w-[min(40rem,calc(100vw-16px))] gap-1" use:placed>
      {#if searchable}
        <input
          type="search"
          class="input input-xs w-full shrink-0"
          aria-label="Filter {aria['aria-label'] ?? ''}"
          aria-controls="{uid}-list"
          placeholder="Type to filter…"
          autocomplete="off"
          value={q}
          oninput={(e) => void filter(e.currentTarget.value)}
          onkeydown={(e) => onkeydown(e, true)} />
      {/if}
      <ul
        bind:this={list}
        id="{uid}-list"
        role="listbox"
        aria-label={aria['aria-label']}
        class="m-0 flex max-h-80 min-h-0 list-none flex-col overflow-x-hidden overflow-y-auto p-0">
        {#each shown as i (i)}
          {@const o = options[i]}
          <!-- svelte-ignore a11y_click_events_have_key_events -->
          <li
            id="{uid}-o{i}"
            role="option"
            aria-selected={i === chosen}
            class={['menu-item shrink-0 cursor-pointer aria-selected:font-semibold', i === active && 'bg-hover']}
            title={o.title ?? o.label}
            onmouseenter={() => void moveTo(i)}
            onmouseleave={() => {
              if (active === i) onhover?.(null, null);
            }}
            onclick={(e) => {
              e.preventDefault();
              choose(i);
            }}>
            <span class="min-w-0 truncate">{o.label}</span>
          </li>
        {:else}
          <li role="presentation" class="note shrink-0 px-2 py-1">No entry matches “{q}”</li>
        {/each}
      </ul>
    </div>
  {/if}
</div>
