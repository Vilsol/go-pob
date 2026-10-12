<script lang="ts">
  import StatGauge from './StatGauge.svelte';
  import { tone as toneClass, type Tone } from './tone';

  type Props = {
    label: string;
    value: string;
    tone?: Tone;
    gauge?: { pct: number; cap?: number; res?: number; seg?: number; warn?: boolean };
    alt?: boolean;
  };
  let { label, value, tone, gauge, alt = false }: Props = $props();
</script>

<div class={['stat-row', toneClass(tone), alt && 'bg-bar']}>
  {#if gauge}<StatGauge {...gauge} {tone} />{/if}
  <span class="stat-label">{label}</span><span class="stat-value">{value}</span>
</div>
