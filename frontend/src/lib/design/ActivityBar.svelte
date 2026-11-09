<script lang="ts">
  import { overallFraction, runningJobs } from '$lib/activity/jobs';

  // determinate while any running job knows its total, a sweep otherwise
  const fraction = $derived(overallFraction($runningJobs));
</script>

{#if $runningJobs.length}
  <div
    class="pointer-events-none fixed inset-x-0 top-0 z-toast h-0.5 overflow-hidden"
    role="progressbar"
    aria-label="Activity"
    aria-valuemin={0}
    aria-valuemax={100}
    aria-valuenow={fraction === undefined ? undefined : Math.round(fraction * 100)}
    aria-busy="true"
    data-determinate={fraction !== undefined}>
    {#if fraction === undefined}
      <div class="sweep h-full w-1/3 bg-accent"></div>
    {:else}
      <div class="h-full bg-accent transition-[width] duration-200" style:width="{fraction * 100}%"></div>
    {/if}
  </div>
{/if}

<style>
  .sweep {
    animation: sweep 1.4s ease-in-out infinite;
  }
  @keyframes sweep {
    from {
      transform: translateX(-100%);
    }
    to {
      transform: translateX(300%);
    }
  }
  /* standing still, the sweep becomes a dim full-width bar */
  @media (prefers-reduced-motion: reduce) {
    .sweep {
      width: 100%;
      opacity: 0.5;
    }
  }
  :global(:root[data-motion='off']) .sweep {
    width: 100%;
    opacity: 0.5;
  }
</style>
