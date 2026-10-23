<script lang="ts">
  import { socketLetters, socketStyle, type SocketStyle } from '$lib/display/appearance';
  import type { Snippet } from 'svelte';
  import { tone as toneClass, type Tone } from './tone';

  /* children: what sits in the socket in place of the gem dot */
  type Props = { socket: Tone; gem?: Tone; letters?: boolean; ornament?: SocketStyle; children?: Snippet };
  let { socket, gem, letters, ornament, children }: Props = $props();
  const uid = $props.id();

  const COLOURS: Partial<Record<Tone, string>> = { R: 'Red', G: 'Green', B: 'Blue', W: 'White', A: 'Abyssal' };
  const colour = $derived(COLOURS[socket]);
  const showLetter = $derived(!!colour && (letters ?? $socketLetters));
  const style = $derived(colour ? (ornament ?? $socketStyle) : 'None');

  // one spike pointing up from the rim, rotated into place
  const spike = (w: number, len: number) => `M${-w},-11 L0,${-11 - len} L${w},-11 Z`;
  const turns = (n: number, from = 0) => Array.from({ length: n }, (_, i) => from + (360 / n) * i);
</script>

<span
  class={['socket', socket === 'A' && 'socket-abyss', toneClass(socket)]}
  role={colour ? (children ? 'group' : 'img') : undefined}
  aria-label={colour ? `${colour} socket` : undefined}>
  {#if socket === 'A'}
    <!-- a ring of rough stone around a dark pit -->
    <svg aria-hidden="true" class="socket-stone" viewBox="-12 -12 24 24">
      <defs>
        <filter id="{uid}-rock" x="-25%" y="-25%" width="150%" height="150%">
          <feTurbulence type="fractalNoise" baseFrequency="0.5" numOctaves="2" seed="7" result="noise" />
          <feDisplacementMap in="SourceGraphic" in2="noise" scale="1.4" xChannelSelector="R" yChannelSelector="G" result="rough" />
          <feDiffuseLighting in="noise" surfaceScale="1.6" lighting-color="#fff" result="light"
            ><feDistantLight azimuth="225" elevation="50" /></feDiffuseLighting>
          <feComposite in="light" in2="rough" operator="arithmetic" k1="1.3" result="lit" />
          <feComposite in="lit" in2="rough" operator="in" />
        </filter>
        <radialGradient id="{uid}-pit"
          ><stop offset="0" stop-color="#020203" /><stop offset="0.7" stop-color="#0d0b10" /><stop offset="1" stop-color="#25222a" /></radialGradient>
      </defs>
      <circle class="stone" r="9" stroke-width="5.6" filter="url(#{uid}-rock)" />
      <circle r="6.6" fill="url(#{uid}-pit)" />
    </svg>
  {/if}
  {#if style !== 'None'}
    <svg aria-hidden="true" class="socket-ornament" viewBox="-20 -20 40 40" data-ornament={style === 'Small' ? 'small' : socket}>
      {#if style === 'Small'}
        <path d={spike(2.5, 5)} />
      {:else if socket === 'G'}
        {#each turns(4, 45) as r (r)}<path d={spike(3.5, 5.5)} transform="rotate({r})" />{/each}
      {:else if socket === 'R'}
        {#each turns(8) as r (r)}<path d={spike(2, 7.5)} transform="rotate({r})" />{/each}
      {:else if socket === 'A'}
        <circle class="swirl" r="14" />
        <circle class="swirl rubble" r="16.5" />
      {:else if socket === 'B'}
        {#each turns(3, 20) as r (r)}<path class="swirl" d="M0,-12.5 C7,-14.5 13,-11 15.5,-4" transform="rotate({r})" />{/each}
      {:else}
        <circle class="swirl" r="15" />
        {#each turns(4) as r (r)}<path d="M0,-12 L2.5,-15 L0,-18 L-2.5,-15 Z" transform="rotate({r})" />{/each}
      {/if}
    </svg>
  {/if}
  {#if children}<span class="contents">{@render children()}</span>{:else if gem}<span aria-hidden="true" class={['socket-gem', toneClass(gem)]}></span>{/if}
  {#if showLetter}<span aria-hidden="true" class="socket-letter">{socket}</span>{/if}
</span>

<style>
  .socket-abyss {
    background: none;
  }
  .socket-stone {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    overflow: visible;
    pointer-events: none;
  }
  .socket-stone .stone {
    fill: none;
    stroke: var(--tone);
  }
  .socket-ornament {
    position: absolute;
    inset: -11px;
    width: calc(100% + 22px);
    height: calc(100% + 22px);
    overflow: visible;
    pointer-events: none;
    z-index: -1;
    fill: var(--color-text-2);
    stroke: var(--color-ground);
    stroke-width: 0.75;
    stroke-linejoin: round;
  }
  .socket-ornament .swirl {
    fill: none;
    stroke: var(--color-text-2);
    stroke-width: 2;
    stroke-linecap: round;
  }
  .socket-ornament .rubble {
    stroke-width: 1.5;
    stroke-dasharray: 2.5 1.8;
  }
  .socket-letter {
    position: absolute;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    font-weight: 700;
    line-height: 1;
    color: var(--color-text);
    text-shadow:
      0 0 2px var(--color-ground),
      0 0 1px var(--color-ground),
      0 0 1px var(--color-ground);
  }
</style>
