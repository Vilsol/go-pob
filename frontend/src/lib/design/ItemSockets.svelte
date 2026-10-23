<script lang="ts">
  import Socket from './Socket.svelte';
  import GemIcon from './GemIcon.svelte';
  import { socketLine } from './items';
  import { socketGrid, type SocketedGem } from './item_sockets';

  /* sizes in px of the board's design, drawn in rem; gems[i] sits in socket i */
  type Props = {
    line: string;
    w: number;
    h: number;
    cell: number;
    cols: number;
    gems?: (SocketedGem | undefined)[];
    ongem?: (gem: SocketedGem | null, el: HTMLElement | null) => void;
  };
  let { line, w, h, cell, cols, gems = [], ongem }: Props = $props();

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

<span class="pointer-events-none absolute inset-0" data-item-sockets={line}>
  {#each parsed.colours as c, i (i)}
    {@const at = centre(i)}
    {@const g = gems[i]}
    <span data-socket={i} class="absolute -translate-1/2" style:left={rem(at.x)} style:top={rem(at.y)}>
      <span class="block" style:transform="scale({size / 24})"
        ><Socket socket={c}
          >{#if g}<span
              role="img"
              aria-label={g.name}
              data-socket-gem={g.name}
              class="pointer-events-auto flex"
              onmouseenter={(ev) => ongem?.(g, ev.currentTarget)}
              onmouseleave={() => ongem?.(null, null)}><GemIcon name={g.name} size={18} decorative /></span
            >{/if}</Socket
        ></span>
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
