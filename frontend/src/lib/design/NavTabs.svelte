<script lang="ts">
  import { isTypingTarget, rovingIndex } from './keys';
  type Tab = { label: string; href: string; key?: string };
  type Props = { tabs: Tab[]; current: string; label?: string; class?: string; onnavigate?: (tab: Tab, e: Event) => void };
  let { tabs, current, label, class: extra = 'h-10 border-b border-line', onnavigate }: Props = $props();

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
    const links = [...nav.querySelectorAll<HTMLAnchorElement>('a.tab')];
    const i = links.indexOf(document.activeElement as HTMLAnchorElement);
    if (i < 0) return;
    const to = rovingIndex(e, i, links.length);
    if (to === undefined) return;
    e.preventDefault();
    links[to].focus();
  }
</script>

<svelte:window {onkeydown} />

<nav bind:this={nav} class={['tabs', extra]} aria-label={label}>
  {#each tabs as t (t.href)}
    <a href={t.href} class="tab" aria-current={t.href === current ? 'page' : undefined} onclick={(e) => onnavigate?.(t, e)} onkeydown={onnavkey}>
      {t.label}{#if t.key}<span class="tab-key">{t.key}</span>{/if}
    </a>
  {/each}
</nav>
