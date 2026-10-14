<script lang="ts">
  import { tick, type Snippet } from 'svelte';
  import { tone as toneClass, type Tone } from './tone';

  // ItemTools.lua:380-409
  type Item = {
    label: string;
    tone?: Tone;
    separator?: boolean;
    caption?: string;
    current?: boolean;
    disabled?: boolean;
    onselect?: () => void;
    onhover?: (el: HTMLElement | null) => void;
    wiki?: string;
  };
  type TriggerAttrs = {
    'aria-haspopup': 'menu';
    'aria-expanded': boolean;
    'aria-controls': string;
    onclick: () => void;
    onkeydown: (e: KeyboardEvent) => void;
  };
  type Props = {
    label: string;
    items: Item[];
    open?: boolean;
    class?: string;
    itemClass?: string;
    trigger: Snippet<[TriggerAttrs]>;
    head?: Snippet;
    row?: Snippet<[Item, number]>;
  };
  let { label, items, open = $bindable(false), class: extra = 'top-full mt-1 w-57.5', itemClass = '', trigger, head, row }: Props = $props();

  const uid = $props.id();
  let root: HTMLElement;
  let menu = $state<HTMLElement>();

  const entries = () => [...(menu?.querySelectorAll<HTMLElement>('[role="menuitem"]:not(:disabled)') ?? [])];

  async function show(at: 'first' | 'last' = 'first') {
    open = true;
    await tick();
    const e = entries();
    (at === 'first' ? e[0] : e[e.length - 1])?.focus();
  }

  function close(refocus = true) {
    open = false;
    if (refocus) root.querySelector<HTMLElement>('[aria-haspopup="menu"]')?.focus();
  }

  function onMenuKey(e: KeyboardEvent) {
    const list = entries();
    const i = list.indexOf(document.activeElement as HTMLElement);
    const to = { ArrowDown: (i + 1) % list.length, ArrowUp: (i - 1 + list.length) % list.length, Home: 0, End: list.length - 1 }[e.key];
    if (to !== undefined) {
      e.preventDefault();
      list[to].focus();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      e.stopPropagation();
      close();
    } else if (e.key === 'Tab') close(false);
  }

  const outside = (e: PointerEvent) => {
    if (open && !root.contains(e.target as Node)) close(false);
  };

  const placed = (panel: HTMLElement) => {
    const pad = 8;
    const host = panel.parentElement!;
    const r0 = host.getBoundingClientRect();
    const p0 = panel.getBoundingClientRect();
    const dx = p0.left - r0.left;
    const gap = Math.max(0, p0.top - r0.bottom);
    Object.assign(panel.style, { position: 'fixed', right: 'auto', bottom: 'auto', margin: '0' });
    const place = () => {
      const r = host.getBoundingClientRect();
      panel.style.maxHeight = '';
      const natural = panel.offsetHeight;
      const below = innerHeight - r.bottom - gap - pad;
      const above = r.top - gap - pad;
      const down = below >= natural || below >= above;
      const h = Math.min(natural, Math.max(0, down ? below : above));
      if (h < natural) Object.assign(panel.style, { maxHeight: `${h}px`, overflowY: 'auto' });
      const x = Math.min(r.left + dx, innerWidth - pad - panel.offsetWidth);
      panel.style.left = `${Math.max(pad, x)}px`;
      panel.style.top = `${down ? r.bottom + gap : r.top - gap - h}px`;
    };
    place();
    const moved = (e: Event) => {
      if (!panel.contains(e.target as Node)) place();
    };
    addEventListener('scroll', moved, true);
    addEventListener('resize', place);
    return {
      destroy: () => {
        removeEventListener('scroll', moved, true);
        removeEventListener('resize', place);
      }
    };
  };

  const attrs: TriggerAttrs = {
    'aria-haspopup': 'menu',
    get 'aria-expanded'() {
      return open;
    },
    'aria-controls': `${uid}-menu`,
    onclick: () => (open ? close(false) : show()),
    onkeydown: (e) => {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        e.preventDefault();
        void show(e.key === 'ArrowUp' ? 'last' : 'first');
      }
    }
  };
</script>

<svelte:window onpointerdown={outside} />

<div bind:this={root} class="relative inline-block">
  {@render trigger(attrs)}
  {#if open}
    <div bind:this={menu} id="{uid}-menu" role="menu" aria-label={label} tabindex="-1" class={['menu', extra]} onkeydown={onMenuKey} use:placed>
      {@render head?.()}
      {#each items as it, i (i)}
        {#if it.separator}<div role="separator" class="my-1 border-t border-line"></div>{/if}
        {#if it.caption}<span class="px-2 pt-1.5 pb-0.5 text-11 tracking-caps-tight text-text-3">{it.caption}</span>{/if}
        <button
          role="menuitem"
          tabindex="-1"
          class={['menu-item', toneClass(it.tone), itemClass]}
          aria-current={it.current ? 'true' : undefined}
          data-wiki={it.wiki}
          disabled={it.disabled}
          onmouseenter={it.onhover && ((e) => it.onhover?.(e.currentTarget))}
          onmouseleave={it.onhover && (() => it.onhover?.(null))}
          onfocus={it.onhover && ((e) => it.onhover?.(e.currentTarget))}
          onblur={it.onhover && (() => it.onhover?.(null))}
          onclick={() => {
            it.onselect?.();
            close();
          }}
          >{#if row}{@render row(it, i)}{:else}{it.label}{/if}</button>
      {/each}
    </div>
  {/if}
</div>
