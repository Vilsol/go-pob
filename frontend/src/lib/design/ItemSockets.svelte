<script lang="ts">
  import Socket from './Socket.svelte';
  import GemIcon from './GemIcon.svelte';
  import ItemIcon from './ItemIcon.svelte';
  import type { ItemSummary } from './fixtures';
  import { socketLine } from './items';
  import { socketGrid, type SocketedGem } from './item_sockets';
  import { rovingIndex } from './keys';

  /* sizes in px of the board's design, drawn in rem; gems[i] or jewels[i] sits in socket i */
  type Props = {
    line: string;
    w: number;
    h: number;
    cell: number;
    cols: number;
    gems?: (SocketedGem | undefined)[];
    ongem?: (gem: SocketedGem | null, el: HTMLElement | null) => void;
    jewels?: (ItemSummary | undefined)[];
    onjewel?: (socket: number | null, el: HTMLElement | null) => void;
    onopenjewel?: (socket: number) => void;
    /* the socket whose jewel tooltip `jewelTipId` is showing */
    shownJewel?: number;
    jewelTipId?: string;
    /* a gem lost the pointer or focus to `to` */
    onleave?: (to: EventTarget | null) => void;
    onescape?: () => void;
    onopen?: () => void;
    /* the gem whose tooltip `tipId` is showing */
    shown?: SocketedGem;
    tipId?: string;
  };
  let {
    line,
    w,
    h,
    cell,
    cols,
    gems = [],
    ongem,
    jewels = [],
    onjewel,
    onopenjewel,
    shownJewel,
    jewelTipId,
    onleave,
    onescape,
    onopen,
    shown,
    tipId
  }: Props = $props();
  let root = $state<HTMLElement>();
  let active = $state(0);
  const filled = $derived(Array.from({ length: Math.max(gems.length, jewels.length) }, (_, i) => i).filter((i) => gems[i] || jewels[i]));
  const tabStop = $derived(filled.includes(active) ? active : filled[0]);
  const focusAt = (i: number) => root?.querySelector<HTMLElement>(`[data-socket="${i}"] :is([data-socket-gem], [data-socket-jewel])`)?.focus();

  export function focusGem(last = false) {
    const i = last ? filled.at(-1) : filled[0];
    if (i !== undefined) focusAt(i);
  }
  // a touch has no hover: the first tap on a jewel shows its tooltip, the second opens it
  let tap = { touch: false, shown: false };
  const onkey = (ev: KeyboardEvent, i: number) => {
    if (ev.key === 'Escape') {
      ev.preventDefault();
      onescape?.();
      return;
    }
    if (ev.key === 'Enter' || ev.key === ' ') {
      ev.preventDefault();
      if (jewels[i]) onopenjewel?.(i);
      else onopen?.();
      return;
    }
    const at = filled.indexOf(i);
    const to = rovingIndex(ev, at, filled.length) ?? rovingIndex(ev, at, filled.length, { orientation: 'vertical' });
    if (to === undefined) return;
    ev.preventDefault();
    focusAt(filled[to]);
  };

  const parsed = $derived(socketLine(line));
  const grid = $derived(socketGrid(parsed.colours.length, cols));
  // the game's item popup, per 47.4645px inventory cell: 35px sockets, 38x15 links centred between two sockets
  const CELL = 47.4645;
  const size = $derived((cell * 35) / CELL);
  const centre = (i: number) => {
    const p = grid.at[i];
    return { x: w / 2 + (p.col - (grid.cols - 1) / 2) * cell, y: h / 2 + (p.row - (grid.rows - 1) / 2) * cell };
  };
  const rem = (px: number) => `${px / 16}rem`;
  const link = (i: number) => {
    const a = centre(i);
    const b = centre(i + 1);
    const across = a.y === b.y;
    const long = (cell * 38) / CELL;
    const thick = (cell * 15) / CELL;
    return { x: (a.x + b.x) / 2, y: (a.y + b.y) / 2, w: across ? long : thick, h: across ? thick : long, across };
  };
</script>

<span class="pointer-events-none absolute inset-0" data-item-sockets={line} bind:this={root}>
  {#each parsed.colours as c, i (i)}
    {@const at = centre(i)}
    {@const g = gems[i]}
    {@const j = jewels[i]}
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
          >{:else if j}<Socket socket={c}
            ><span
              role="button"
              aria-label={j.name}
              data-socket-jewel={j.name}
              tabindex={i === tabStop ? 0 : -1}
              aria-describedby={shownJewel === i ? jewelTipId : undefined}
              class="pointer-events-auto relative flex overflow-hidden rounded-full"
              onpointerdown={(ev) => (tap = { touch: ev.pointerType === 'touch', shown: shownJewel === i })}
              onmouseenter={(ev) => onjewel?.(i, ev.currentTarget)}
              onmouseleave={(ev) => {
                onjewel?.(null, null);
                onleave?.(ev.relatedTarget);
              }}
              onfocus={(ev) => {
                active = i;
                onjewel?.(i, ev.currentTarget);
              }}
              onblur={(ev) => {
                onjewel?.(null, null);
                onleave?.(ev.relatedTarget);
              }}
              onclick={(ev) => {
                const first = tap.touch && !tap.shown;
                tap = { touch: false, shown: false };
                if (first) onjewel?.(i, ev.currentTarget);
                else onopenjewel?.(i);
              }}
              onkeydown={(ev) => onkey(ev, i)}><ItemIcon name={j.name} base={j.base} rarity={j.rarity} size={14} decorative /></span
            ></Socket
          >{:else}<Socket socket={c} />{/if}</span>
    </span>
  {/each}
  {#each parsed.links as linked, i (i)}
    {#if linked}
      {@const l = link(i)}
      <svg
        aria-hidden="true"
        data-socket-link={i}
        class="socket-bar absolute -translate-1/2"
        viewBox={l.across ? '-19 -7.5 38 15' : '-7.5 -19 15 38'}
        style:left={rem(l.x)}
        style:top={rem(l.y)}
        style:width={rem(l.w)}
        style:height={rem(l.h)}
        ><g transform={l.across ? undefined : 'rotate(90)'}
          ><rect x="-11.6" y="-1.2" width="23.2" height="2.4" rx="0.6" /><path d="M-7.5,0 L0,-3.1 L7.5,0 L0,3.1 Z" /><path
            class="glint"
            d="M-5.5,0 L0,-1.9 L5.5,0" /></g
        ></svg>
    {/if}
  {/each}
</span>

<style>
  /* the bar spans the gap between the two sockets' gold rims */
  .socket-bar {
    z-index: 2;
    overflow: visible;
    fill: var(--color-sock-link);
    stroke: #1e140a;
    stroke-width: 0.6;
    stroke-linejoin: round;
  }
  .socket-bar .glint {
    fill: none;
    stroke: var(--color-sock-rim);
    stroke-width: 0.5;
    stroke-linecap: round;
  }
</style>
