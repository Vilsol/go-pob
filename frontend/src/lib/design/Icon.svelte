<script lang="ts">
  let { name }: { name: 'gear' | 'info' | 'save' | 'import-export' | 'grip' } = $props();
</script>

<svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true">
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
  {:else if name === 'import-export'}
    <g fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round">
      <path d="M5 2v11.5M2.5 11 5 13.5 7.5 11"></path>
      <path d="M11 14V2.5M8.5 5 11 2.5 13.5 5"></path>
    </g>
  {:else if name === 'grip'}
    <g fill="currentColor">
      {#each [4, 8, 12] as y (y)}<circle cx="6" cy={y} r="1.1"></circle><circle cx="10" cy={y} r="1.1"></circle>{/each}
    </g>
  {:else}
    <circle cx="8" cy="8" r="6.3" fill="none" stroke="currentColor" stroke-width="1.4"></circle>
    <path d="M8 7.2v4" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"></path>
    <circle cx="8" cy="4.9" r="0.95" fill="currentColor"></circle>
  {/if}
</svg>
