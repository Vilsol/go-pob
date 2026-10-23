<script lang="ts">
  import Socket from './Socket.svelte';
  import GemIcon from './GemIcon.svelte';
  import { socketLine } from './items';
  import { socketGrid, type SocketedGem } from './item_sockets';
  import { rovingIndex } from './keys';

  /* sizes in px of the board's design, drawn in rem; gems[i] sits in socket i */
  type Props = {
    line: string;
    w: number;
    h: number;
    cell: number;
    cols: number;
    gems?: (SocketedGem | undefined)[];
    ongem?: (gem: SocketedGem | null, el: HTMLElement | null) => void;
    /* a gem lost the pointer or focus to `to` */
    onleave?: (to: EventTarget | null) => void;
    onescape?: () => void;
    onopen?: () => void;
    /* the gem whose tooltip `tipId` is showing */
    shown?: SocketedGem;
    tipId?: string;
  };
  let { line, w, h, cell, cols, gems = [], ongem, onleave, onescape, onopen, shown, tipId }: Props = $props();
  let root = $state<HTMLElement>();
  let active = $state(0);
  const filled = $derived(gems.flatMap((g, i) => (g ? [i] : [])));
  const tabStop = $derived(filled.includes(active) ? active : filled[0]);

  export function focusGem(last = false) {
    const i = last ? filled.at(-1) : filled[0];
    if (i !== undefined) root?.querySelector<HTMLElement>(`[data-socket="${i}"] [data-socket-gem]`)?.focus();
  }
  const onkey = (ev: KeyboardEvent, i: number) => {
    if (ev.key === 'Escape') {
      ev.preventDefault();
      onescape?.();
      return;
    }
    if (ev.key === 'Enter' || ev.key === ' ') {
      ev.preventDefault();
      onopen?.();
      return;
    }
    const at = filled.indexOf(i);
    const to = rovingIndex(ev, at, filled.length) ?? rovingIndex(ev, at, filled.length, { orientation: 'vertical' });
    if (to === undefined) return;
    ev.preventDefault();
    root?.querySelector<HTMLElement>(`[data-socket="${filled[to]}"] [data-socket-gem]`)?.focus();
  };

  const parsed = $derived(socketLine(line));
  const grid = $derived(socketGrid(parsed.colours.length, cols));
  const size = $derived(Math.min(cell * 0.6, 44));
  const centre = (i: number) => {
    const p = grid.at[i];
    return { x: w / 2 + (p.col - (grid.cols - 1) / 2) * cell, y: h / 2 + (p.row - (grid.rows - 1) / 2) * cell };
  };
  const rem = (px: number) => `${px / 16}rem`;
  const link = (i: number) => {
    const a = centre(i);
    const b = centre(i + 1);
    const across = a.y === b.y;
    const long = Math.abs(across ? b.x - a.x : b.y - a.y) - size * 0.7;
    const thick = size * 0.3;
    return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2, w: across ? long : thick, h: across ? thick : long };
  };
</script>

<span class="pointer-events-none absolute inset-0" data-item-sockets={line} bind:this={root}>
  {#each parsed.colours as c, i (i)}
    {@const at = centre(i)}
    {@const g = gems[i]}
    <span data-socket={i} class="absolute -translate-1/2" style:left={rem(at.x)} style:top={rem(at.y)}>
      <span class="block" style:transform="scale({size / 24})"
        >{#if g}<Socket socket={c}
            ><span
              role="button"
              aria-label={g.name}
              data-socket-gem={g.name}
              tabindex={i === tabStop ? 0 : -1}
              aria-describedby={shown?.group === g.group && shown.gem === g.gem ? tipId : undefined}
              class="pointer-events-auto flex rounded-full"
              onmouseenter={(ev) => ongem?.(g, ev.currentTarget)}
              onmouseleave={(ev) => {
                ongem?.(null, null);
                onleave?.(ev.relatedTarget);
              }}
              onfocus={(ev) => {
                active = i;
                ongem?.(g, ev.currentTarget);
              }}
              onblur={(ev) => {
                ongem?.(null, null);
                onleave?.(ev.relatedTarget);
              }}
              onclick={() => onopen?.()}
              onkeydown={(ev) => onkey(ev, i)}><GemIcon name={g.name} size={18} decorative /></span
            ></Socket
          >{:else}<Socket socket={c} />{/if}</span>
    </span>
  {/each}
  {#each parsed.links as linked, i (i)}
    {#if linked}
      {@const l = link(i)}
      <span
        data-socket-link={i}
        class="socket-bar absolute -translate-1/2"
        style:left={rem(l.x)}
        style:top={rem(l.y)}
        style:width={rem(l.w)}
        style:height={rem(l.h)}></span>
    {/if}
  {/each}
</span>

<style>
  .socket-bar {
    z-index: 2;
    border-radius: 2px;
    background: var(--color-text-2);
    box-shadow: 0 0 0 1px var(--color-ground);
  }
</style>
