<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from '../Icon.svelte';
  import { rovingIndex } from '../keys';
  import type { MenuItem } from '../menu';
  import MenuItems from '../MenuItems.svelte';
  import NavTabs from '../NavTabs.svelte';
  import Sheet from './Sheet.svelte';
  import { moreTabs, primaryTabs, type MobileTab } from './mobile';

  type Action = { key: string; label: string; note?: string };
  type Props = {
    name: string;
    current: string;
    label: string;
    dirty?: boolean;
    canUndo?: boolean;
    canRedo?: boolean;
    actions?: Action[];
    statsOpen?: boolean;
    moreOpen?: boolean;
    mainClass?: string;
    tabs?: MobileTab[];
    more?: MobileTab[];
    strip: Snippet<[() => void]>;
    stats: Snippet;
    children: Snippet;
    onnavigate?: (href: string) => void;
    onback?: () => void;
    onundo?: () => void;
    onredo?: () => void;
    onsave?: () => void;
    onaction?: (key: string) => void;
  };
  let {
    name,
    current,
    label,
    dirty = false,
    canUndo = true,
    canRedo = false,
    actions = [],
    statsOpen = $bindable(false),
    moreOpen = $bindable(false),
    mainClass = '',
    tabs = primaryTabs,
    more = moreTabs,
    strip,
    stats,
    children,
    onnavigate,
    onback,
    onundo,
    onredo,
    onsave,
    onaction
  }: Props = $props();

  const inMore = $derived(more.some((t) => t.href === current));
  const actionKey = (e: KeyboardEvent) => {
    const list = [...(e.currentTarget as HTMLElement).querySelectorAll<HTMLElement>('[role="menuitem"]:not(:disabled)')];
    const to = rovingIndex(e, list.indexOf(document.activeElement as HTMLElement), list.length, { orientation: 'vertical' });
    if (to === undefined) return;
    e.preventDefault();
    list[to].focus();
  };
  const go = (href: string) => (e: Event) => {
    e.preventDefault();
    moreOpen = false;
    onnavigate?.(href);
  };
</script>

<div
  class="box-border flex h-dvh w-full min-w-0 flex-col overflow-hidden bg-ground pt-[env(safe-area-inset-top)] pr-[env(safe-area-inset-right)] pl-[env(safe-area-inset-left)] font-sans text-text"
  data-mobile-frame>
  <header class="flex h-11 shrink-0 items-center gap-1 border-b border-line bg-bar px-2">
    <button type="button" class="btn btn-ghost h-10 shrink-0 px-2 text-14" aria-label="Back to builds" onclick={onback}>‹</button>
    <span class="min-w-0 flex-1 truncate text-14 font-semibold">{name}</span>
    <button type="button" class="btn btn-ghost size-10 shrink-0 p-0" aria-label="Undo" disabled={!canUndo} onclick={onundo}>↶</button>
    <button type="button" class="btn btn-ghost size-10 shrink-0 p-0" aria-label="Redo" disabled={!canRedo} onclick={onredo}>↷</button>
    <button
      type="button"
      class={['btn size-10 shrink-0 p-0', dirty ? 'btn-primary' : 'btn-ghost']}
      aria-label={dirty ? 'Save (unsaved changes)' : 'Saved'}
      onclick={onsave}>
      <Icon name="save" />
    </button>
  </header>
  {@render strip(() => (statsOpen = true))}
  <main class={['relative box-border flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto', mainClass]} aria-label={label}>
    {@render children()}
  </main>
  <NavTabs variant="bar" {tabs} {current} label="Pages" onnavigate={(t: MobileTab, e: Event) => go(t.href)(e)}>
    {#snippet trailing()}<button
        type="button"
        class={[
          'flex h-14 min-w-0 flex-1 items-center justify-center border-0 bg-transparent font-sans text-12',
          inMore ? 'font-semibold text-accent' : 'text-text-2'
        ]}
        aria-haspopup="dialog"
        aria-expanded={moreOpen}
        onclick={() => (moreOpen = true)}>More</button
      >{/snippet}
  </NavTabs>
</div>

<Sheet bind:open={statsOpen} title="Stats">
  <div class="flex min-h-0 flex-1 flex-col">
    {@render stats()}
  </div>
</Sheet>

<Sheet bind:open={moreOpen} title="More">
  <nav aria-label="More pages" class="grid grid-cols-2 gap-2 p-3">
    {#each more as t (t.href)}
      <a
        href={t.href}
        class={['btn h-12 justify-center text-14 no-underline', t.href === current && 'btn-primary']}
        aria-current={t.href === current ? 'page' : undefined}
        onclick={go(t.href)}>{t.label}</a>
    {/each}
  </nav>
  {#if actions.length}
    <h4 class="caps px-4 pt-2 pb-1">Build</h4>
    <div role="menu" aria-label="Build actions" tabindex="-1" class="flex flex-col pb-3" onkeydown={actionKey}>
      <MenuItems
        items={actions.map((a) => ({ label: a.label, note: a.note, onselect: () => onaction?.(a.key) }))}
        itemClass="h-12 w-full rounded-none border-b border-line px-4 text-14"
        onpick={(it: MenuItem) => {
          moreOpen = false;
          it.onselect?.();
        }} />
    </div>
  {/if}
</Sheet>
