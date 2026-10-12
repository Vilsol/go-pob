<script lang="ts">
  type Tab = { label: string; href: string; key?: string };
  type Props = { tabs: Tab[]; current: string; label?: string; class?: string; onnavigate?: (tab: Tab, e: Event) => void };
  let { tabs, current, label, class: extra = 'h-10 border-b border-line', onnavigate }: Props = $props();

  function onkeydown(e: KeyboardEvent) {
    if (e.ctrlKey || e.metaKey || e.repeat || !onnavigate) return;
    const digit = /^(?:Digit|Numpad)([0-9])$/.exec(e.code)?.[1];
    if (!digit) return;
    const t = e.target as HTMLElement | null;
    const typing = !!t && (t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName));
    if (typing && !e.altKey) return;
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
    const to = { ArrowRight: i + 1, ArrowLeft: i - 1, Home: 0, End: links.length - 1 }[e.key];
    if (to === undefined) return;
    e.preventDefault();
    links[(to + links.length) % links.length].focus();
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
