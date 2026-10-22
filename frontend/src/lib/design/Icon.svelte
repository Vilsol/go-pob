<script lang="ts" module>
  // stroked 16x16 paths drawn with currentColor: [path, stroke width]
  const STROKES = {
    'import-export': ['M5 2v11.5M2.5 11 5 13.5 7.5 11 M11 14V2.5M8.5 5 11 2.5 13.5 5', 1.4],
    folder: ['M1.5 3.5h4.5l1.5 1.5h7v7.5h-13z', 1.3],
    'folder-open': ['M1.5 12.5v-9h4.5l1.5 1.5h5.5v2 M1.5 12.5l2-5.5h11l-2 5.5z', 1.3],
    'chevron-up': ['M4 10l4-4 4 4', 1.5],
    'chevron-down': ['M4 6l4 4 4-4', 1.5],
    'chevron-right': ['M6 4l4 4-4 4', 1.5],
    'chevron-left': ['M10 4 6 8l4 4', 1.5],
    close: ['M4 4l8 8M12 4l-8 8', 1.5],
    copy: ['M5.5 5.5h8v8h-8z M10.5 5.5v-3h-8v8h3', 1.3],
    check: ['M3 8.5l3.2 3L13 4.5', 1.6],
    warn: ['M8 1.8 14.7 13.5H1.3z M8 6.2v3.6 M8 11.6v.1', 1.4],
    star: ['M8 1.8l1.9 4 4.3.5-3.2 3 .9 4.3L8 11.4l-3.9 2.2.9-4.3-3.2-3 4.3-.5z', 1.3],
    refresh: ['M13 3.5v3h-3 M12.6 6.5A5 5 0 1 0 13 9.5', 1.5],
    undo: ['M5 3 2 6l3 3 M2 6h8a4 4 0 0 1 0 8H7', 1.6],
    redo: ['m11 3 3 3-3 3 M14 6H6a4 4 0 0 0 0 8h3', 1.6],
    tree: [
      'M8 4.5v3.5M8 8 5 11.2M8 8l3 3.2 M6.5 3a1.5 1.5 0 1 0 3 0a1.5 1.5 0 1 0-3 0 M2.5 12.5a1.5 1.5 0 1 0 3 0a1.5 1.5 0 1 0-3 0 M10.5 12.5a1.5 1.5 0 1 0 3 0a1.5 1.5 0 1 0-3 0',
      1.3
    ],
    armour: ['M8 1.8 13 3.5v4c0 3.2-2.2 5.4-5 6.7-2.8-1.3-5-3.5-5-6.7v-4z', 1.3],
    gem: ['M4.5 2.5h7l3 3.5L8 14 1.5 6z M1.5 6h13 M6 2.5 5 6l3 8 3-8-1-3.5', 1.2],
    sliders: ['M2.5 4h11M2.5 8h11M2.5 12h11 M6 2.5v3 M10.5 6.5v3 M5 10.5v3', 1.3],
    layers: ['M8 2 14 5 8 8 2 5z M2 8l6 3 6-3 M2 11l6 3 6-3', 1.3],
    party: [
      'M6 7a2.2 2.2 0 1 0 0-4.4A2.2 2.2 0 0 0 6 7z M1.8 13.5c.4-2.6 2.1-4 4.2-4s3.8 1.4 4.2 4 M11 7.2a1.8 1.8 0 1 0 0-3.6 M12 9.6c1.3.4 2.1 1.6 2.3 3.4',
      1.3
    ],
    calc: ['M3 2h10v12H3z M5 4.5h6 M5.5 8h1M9.5 8h1M5.5 11h1M9.5 11h1', 1.3],
    notes: ['M3.5 2h9v12h-9z M5.5 5h5M5.5 8h5M5.5 11h3', 1.3]
  } as const;
  export type IconName = keyof typeof STROKES | 'gear' | 'save' | 'grip' | 'more' | 'info';
</script>

<script lang="ts">
  let { name, size = 16, class: extra = '' }: { name: IconName; size?: number; class?: string } = $props();
  const stroke = $derived(name in STROKES ? STROKES[name as keyof typeof STROKES] : undefined);
</script>

<svg width={size} height={size} class={['shrink-0', extra]} style:width="{size / 16}rem" style:height="{size / 16}rem" viewBox="0 0 16 16" aria-hidden="true">
  {#if name === 'gear'}
    <path
      d="M13.40 6.51 L15.12 6.92 L15.12 9.08 L13.40 9.49 L12.87 10.76 L13.80 12.27 L12.27 13.80 L10.76 12.87 L9.49 13.40 L9.08 15.12 L6.92 15.12 L6.51 13.40 L5.24 12.87 L3.73 13.80 L2.20 12.27 L3.13 10.76 L2.60 9.49 L0.88 9.08 L0.88 6.92 L2.60 6.51 L3.13 5.24 L2.20 3.73 L3.73 2.20 L5.24 3.13 L6.51 2.60 L6.92 0.88 L9.08 0.88 L9.49 2.60 L10.76 3.13 L12.27 2.20 L13.80 3.73 L12.87 5.24 Z"
      fill="none"
      stroke="currentColor"
      stroke-width="1.3"
      stroke-linejoin="round"></path>
    <circle cx="8" cy="8" r="2.3" fill="none" stroke="currentColor" stroke-width="1.3"></circle>
  {:else if name === 'save'}
    <g fill="none" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round">
      <path d="M2.5 2.5h8.5l2.5 2.5v8.5h-11z"></path>
      <path d="M5 2.5v3.5h5.5V2.5"></path>
      <path d="M4.5 13.5V9.5h7v4"></path>
    </g>
  {:else if name === 'grip'}
    <g fill="currentColor">
      {#each [4, 8, 12] as y (y)}<circle cx="6" cy={y} r="1.1"></circle><circle cx="10" cy={y} r="1.1"></circle>{/each}
    </g>
  {:else if name === 'more'}
    <g fill="currentColor">
      {#each [3.5, 8, 12.5] as x (x)}<circle cx={x} cy="8" r="1.25"></circle>{/each}
    </g>
  {:else if name === 'info'}
    <circle cx="8" cy="8" r="6.3" fill="none" stroke="currentColor" stroke-width="1.4"></circle>
    <path d="M8 7.2v4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"></path>
    <circle cx="8" cy="4.9" r="0.95" fill="currentColor"></circle>
  {:else if stroke}
    <path d={stroke[0]} fill="none" stroke="currentColor" stroke-width={stroke[1]} stroke-linecap="round" stroke-linejoin="round"></path>
  {/if}
</svg>
