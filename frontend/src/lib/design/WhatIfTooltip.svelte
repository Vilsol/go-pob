<script lang="ts">
  import type { Snippet } from 'svelte';
  import { tone as toneClass } from './tone';

  type Line = { label: string; text: string; dir: 'up' | 'down'; perPoint?: string; color?: string };
  type Section = { title?: string; lines: Line[]; header?: string; actor?: string; readyAt?: number; calcMs?: number };
  type Props = {
    title: string;
    kind?: string;
    mods?: string[];
    modsTitle?: string;
    info?: string[];
    footer?: string;
    action: string;
    sections: Section[];
    id?: string;
    node?: number;
    pending?: boolean;
    empty?: string;
    extra?: Snippet;
    tail?: Snippet;
    translucent?: boolean;
  };
  let {
    title,
    kind,
    mods = [],
    modsTitle,
    info = [],
    footer,
    action,
    sections,
    id,
    node,
    pending = false,
    empty,
    extra,
    tail,
    translucent = false
  }: Props = $props();
  // Tooltip.lua:95
  const lines = (entries: string[]) => entries.flatMap((e) => e.split('\n'));
</script>

<div
  role="tooltip"
  {id}
  class={[
    'tooltip flex w-max max-w-[min(640px,90vw)] min-w-90 flex-col gap-2 rounded-ovl px-4 py-3 text-13 shadow-[0_8px_24px_rgba(0,0,0,0.5)]',
    translucent && 'bg-surface-2/88'
  ]}>
  <div class="flex items-baseline justify-between">
    <span class="text-15 font-semibold">{title}</span>{#if kind}<span class="note">{kind}</span>{/if}
  </div>
  {#if info.length}
    <div class="flex flex-col gap-0.5" aria-label="Details">
      {#each lines(info) as l, k (k)}<span>{l}</span>{/each}
    </div>
  {/if}
  {#if mods.length}
    {#if modsTitle}<div class="text-12 text-text-2">{modsTitle}</div>{/if}
    <div class="flex flex-col gap-0.5 text-magic">
      {#each lines(mods) as m, k (k)}<span>{m}</span>{/each}
    </div>
  {/if}
  {@render extra?.()}
  {#if action || sections.length || pending || empty}<div
      class="flex flex-col gap-1 border-t border-line pt-2.5 font-mono"
      role="group"
      aria-label="Stat differences"
      aria-busy={node !== undefined ? pending : undefined}
      data-node={node}>
      <div class="font-sans text-12 text-text-2">{action}</div>
      {#each sections as s, i (i)}
        <div class="contents" data-header={s.header} data-actor={s.actor} data-ready-at={s.readyAt} data-calc-ms={s.calcMs}>
          {#if s.title}<div class="subhead font-sans">{s.title}</div>{/if}
          {#each s.lines as l, j (j)}
            <div class={['kv', j % 2 && 'bg-bar']} data-line>
              <span
                class={l.color &&
                  "relative pl-2 before:absolute before:top-0.75 before:bottom-0.75 before:left-0 before:w-0.75 before:rounded-[2px] before:bg-(--tone) before:content-['']"}
                style:--tone={l.color}>{l.label}</span
              ><span class={['toned', toneClass(l.dir)]}
                >{l.perPoint ? l.text + ' ' : l.text}{#if l.perPoint}<span class="text-11 text-text-3">{l.perPoint}</span>{/if}</span>
            </div>
          {/each}
        </div>
      {/each}
      {#if pending}
        <div class="font-sans text-12 text-text-3">Calculating…</div>
      {:else if empty}
        <div class="font-sans text-12 text-text-2">{empty}</div>
      {/if}
    </div>{/if}
  {#if footer}<div class="text-12 whitespace-pre-line text-text-2">{footer}</div>{/if}
  {@render tail?.()}
</div>
