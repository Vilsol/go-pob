<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from '../Icon.svelte';
  import Sheet from './Sheet.svelte';
  import { moreTabs, primaryTabs } from './mobile';

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

  const inMore = $derived(moreTabs.some((t) => t.href === current));
  const go = (href: string) => (e: Event) => {
    e.preventDefault();
    moreOpen = false;
    onnavigate?.(href);
  };
</script>

<div class="box-border flex h-dvh w-full min-w-0 flex-col overflow-hidden bg-ground font-sans text-text" data-mobile-frame>
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
  <nav class="flex shrink-0 border-t border-line bg-bar pb-[env(safe-area-inset-bottom)]" aria-label="Pages">
    {#each primaryTabs as t (t.href)}
      <a
        href={t.href}
        class={['flex h-14 min-w-0 flex-1 items-center justify-center text-12 no-underline', t.href === current ? 'font-semibold text-accent' : 'text-text-2']}
        aria-current={t.href === current ? 'page' : undefined}
        onclick={go(t.href)}>{t.label}</a>
    {/each}
    <button
      type="button"
      class={[
        'flex h-14 min-w-0 flex-1 items-center justify-center border-0 bg-transparent font-sans text-12',
        inMore ? 'font-semibold text-accent' : 'text-text-2'
      ]}
      aria-haspopup="dialog"
      aria-expanded={moreOpen}
      onclick={() => (moreOpen = true)}>More</button>
  </nav>
</div>

<Sheet bind:open={statsOpen} title="Stats">
  {@render stats()}
</Sheet>

<Sheet bind:open={moreOpen} title="More">
  <nav aria-label="More pages" class="grid grid-cols-2 gap-2 p-3">
    {#each moreTabs as t (t.href)}
      <a
        href={t.href}
        class={['btn h-12 justify-center text-14 no-underline', t.href === current && 'btn-primary']}
        aria-current={t.href === current ? 'page' : undefined}
        onclick={go(t.href)}>{t.label}</a>
    {/each}
  </nav>
  {#if actions.length}
    <h4 class="caps px-4 pt-2 pb-1">Build</h4>
    <ul class="m-0 flex list-none flex-col p-0 pb-3" aria-label="Build actions">
      {#each actions as a (a.key)}
        <li>
          <button
            type="button"
            class="flex h-12 w-full items-center gap-2 border-0 border-b border-line bg-transparent px-4 text-left font-sans text-14 text-text"
            onclick={() => {
              moreOpen = false;
              onaction?.(a.key);
            }}>
            <span class="flex-1">{a.label}</span>
            {#if a.note}<span class="note">{a.note}</span>{/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</Sheet>
