<script lang="ts">
  import type { Snippet } from 'svelte';
  import { isTypingTarget, rovingIndex } from './keys';
  type Tab = { label: string; href: string; key?: string };
  type Props = {
    tabs: Tab[];
    current: string;
    label?: string;
    /** underline: the page tabs; bar: the phone's bottom bar */
    variant?: 'underline' | 'bar';
    class?: string;
    /** extra classes on each underline tab, and on its shortcut key */
    tabClass?: string;
    keyClass?: string;
    /** after the tabs, inside the bar (the phone's More button) */
    trailing?: Snippet;
    onnavigate?: (tab: Tab, e: Event) => void;
  };
  let { tabs, current, label, variant = 'underline', class: extra, tabClass, keyClass, trailing, onnavigate }: Props = $props();
  const navClass = $derived(
    extra ?? (variant === 'bar' ? 'flex shrink-0 border-t border-line bg-bar pb-[env(safe-area-inset-bottom)]' : 'h-10 border-b border-line')
  );

  function onkeydown(e: KeyboardEvent) {
    if (e.ctrlKey || e.metaKey || e.repeat || !onnavigate) return;
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(e.code)?.[1];
    if (!digit) return;
    if (isTypingTarget(e.target) && !e.altKey) return;
    const tab = tabs.find((x) => x.key === digit);
    if (!tab) return;
    e.preventDefault();
    onnavigate(tab, e);
  }

  let nav: HTMLElement;
  function onnavkey(e: KeyboardEvent) {
    const links = [...nav.querySelectorAll<HTMLAnchorElement>(':scope > a')];
    const i = links.indexOf(document.activeElement as HTMLAnchorElement);
    if (i < 0) return;
    const to = rovingIndex(e, i, links.length);
    if (to === undefined) return;
    e.preventDefault();
    links[to].focus();
  }
</script>

<svelte:window {onkeydown} />

<nav bind:this={nav} class={[variant === 'underline' && 'tabs', navClass]} aria-label={label}>
  {#each tabs as t (t.href)}
    <a
      href={t.href}
      class={variant === 'bar'
        ? ['flex h-14 min-w-0 flex-1 items-center justify-center text-12 no-underline', t.href === current ? 'font-semibold text-accent' : 'text-text-2']
        : ['tab', tabClass]}
      aria-current={t.href === current ? 'page' : undefined}
      onclick={(e) => onnavigate?.(t, e)}
      onkeydown={onnavkey}>
      {t.label}{#if t.key}<span class={['tab-key', keyClass]}>{t.key}</span>{/if}
    </a>
  {/each}
  {@render trailing?.()}
</nav>
