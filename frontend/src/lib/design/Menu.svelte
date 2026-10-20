<script lang="ts">
  import { float, pressedOutside, raise } from './floating';
  import { rovingIndex } from './keys';
  import { tick, type Snippet } from 'svelte';
  import MenuItems from './MenuItems.svelte';
  import type { MenuItem } from './menu';

  type Item = MenuItem;
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
    const to = rovingIndex(e, i, list.length, { orientation: 'vertical' });
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
    if (open && pressedOutside(e, root)) close(false);
  };

  // the class places the panel first; its offset from the trigger is kept once it is fixed
  const placed = (panel: HTMLElement) => {
    const host = panel.parentElement!;
    const r0 = host.getBoundingClientRect();
    const p0 = panel.getBoundingClientRect();
    const along = p0.left - r0.left;
    const gap = Math.max(0, p0.top - r0.bottom);
    Object.assign(panel.style, { position: 'fixed', right: 'auto', bottom: 'auto', margin: '0' });
    raise(panel);
    return { destroy: float(panel, host, { gap, along, fit: true }) };
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
      <MenuItems
        {items}
        {itemClass}
        {row}
        onpick={(it: MenuItem) => {
          it.onselect?.();
          close();
        }} />
    </div>
  {/if}
</div>
