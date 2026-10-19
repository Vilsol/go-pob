<script lang="ts">
  import { tick } from 'svelte';
  import type { Snippet } from 'svelte';
  import Tabs from '../Tabs.svelte';
  import { panelId, tabId } from '../tab_ids';

  // phone Skills: the socket group list and the group editor as two swipe panes; a tapped group slides to its editor
  // off: the page as it is, for the desktop
  type Props = { on?: boolean; pane?: 0 | 1; children: Snippet };
  let { on = true, pane: pane0 = 0, children }: Props = $props();

  const panes = [
    { id: 'groups', label: 'Socket groups' },
    { id: 'group', label: 'Selected group' }
  ];
  const base = $props.id();
  let strip = $state<HTMLElement>();
  let pane = $state<0 | 1>(pane0);
  const show = (p: 0 | 1, smooth = true) => {
    pane = p;
    strip?.scrollTo({ left: p * strip.clientWidth, behavior: smooth ? 'smooth' : 'instant' });
  };
  // a tapped socket group slides to its editor
  const onGroupTap = (el: HTMLElement) => {
    const h = async (e: MouseEvent) => {
      if (!(e.target as HTMLElement).closest('[aria-label="Socket groups"] button[aria-current]')) return;
      await tick();
      show(1);
    };
    el.addEventListener('click', h);
    return { destroy: () => el.removeEventListener('click', h) };
  };
  $effect(() => {
    if (strip && pane0) show(pane0, false);
  });
</script>

{#if !on}
  {@render children()}
{:else}
  <div class="shrink-0 border-b border-line px-3 py-2">
    <Tabs id={base} variant="segmented" fill label="Skills view" tabs={panes} selected={panes[pane].id} onchange={(id) => show(id === 'group' ? 1 : 0)} />
  </div>
  <div
    bind:this={strip}
    role="tabpanel"
    id={panelId(base)}
    aria-labelledby={tabId(base, panes[pane].id)}
    class="flex min-h-0 flex-1 snap-x snap-mandatory overflow-x-auto overflow-y-hidden [scrollbar-width:none] [&>*]:box-border [&>*]:h-full [&>*]:max-h-none! [&>*]:w-full! [&>*]:shrink-0! [&>*]:basis-full! [&>*]:snap-start [&>*]:p-3 [&>section]:rounded-none [&>section]:border-0"
    onscroll={() => {
      if (strip) pane = Math.round(strip.scrollLeft / strip.clientWidth) as 0 | 1;
    }}
    use:onGroupTap>
    {@render children()}
  </div>
{/if}
