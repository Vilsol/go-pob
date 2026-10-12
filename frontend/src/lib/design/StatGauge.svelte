<script lang="ts">
  import { tone as toneClass, type Tone } from './tone';

  type Props = { pct?: number; cap?: number; tone?: Tone; res?: number; seg?: number; warn?: boolean };
  let { pct, cap, tone, res, seg, warn = false }: Props = $props();

  const over = $derived(warn && pct != null && pct > 100);
  const fill = $derived(
    over
      ? '[--fill:color-mix(in_srgb,var(--color-worse)_24%,transparent)]'
      : tone
        ? '[--fill:color-mix(in_srgb,var(--tone)_22%,transparent)]'
        : '[--fill:color-mix(in_srgb,var(--color-accent)_16%,transparent)]'
  );
</script>

<div
  data-gauge
  class={['absolute inset-0', toneClass(tone), fill]}
  style:--pct={pct == null ? null : Math.min(pct, 100) + '%'}
  style:--cap={(cap ?? 0) + '%'}
  style:--res={(res ?? 0) + '%'}
  style:--seg={seg ?? 1}>
  {#if pct != null}
    <div data-fill class="absolute top-px bottom-px left-0 w-(--pct) rounded-[2px] bg-(--fill)" aria-hidden="true"></div>
  {/if}
  {#if res != null}
    <div data-res class="absolute top-px right-0 bottom-px w-(--res) border-l border-line-bright bg-[rgba(141,138,131,0.14)]" title="{res}% reserved"></div>
  {/if}
  {#if seg != null}
    <div
      class="absolute inset-x-0 top-px bottom-px [background:repeating-linear-gradient(90deg,_transparent_0,_transparent_calc(100%_/_var(--seg)_-_2px),_var(--color-surface-1)_calc(100%_/_var(--seg)_-_2px),_var(--color-surface-1)_calc(100%_/_var(--seg)))]"
      aria-hidden="true">
    </div>
  {/if}
  {#if cap != null && cap < 100}
    <div
      data-cap-zone
      class="absolute top-px right-0 bottom-px left-(--cap) [background:repeating-linear-gradient(135deg,_rgba(141,_138,_131,_0.22)_0_3px,_transparent_3px_6px)]"
      aria-hidden="true">
    </div>
    <div data-cap-tick class="absolute inset-y-0 left-[calc(var(--cap)-1px)] w-0.5 bg-text-3" aria-hidden="true"></div>
  {/if}
</div>
