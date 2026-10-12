<script lang="ts">
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';
  import { METRICS, type Metric, type MetricId } from './compare';

  // CompareTab.lua:968-985
  type Props = { value: MetricId; onchange: (m: MetricId) => void; metrics?: Metric[] };
  let { value, onchange, metrics = METRICS }: Props = $props();
</script>

<FloatingLabel label="Metric"
  ><Select
    size="xs"
    class="h-7"
    options={metrics.map((m) => ({ value: m.id, label: m.label }))}
    {value}
    onchange={(e: Event) => onchange((e.currentTarget as HTMLSelectElement).value)} /></FloatingLabel>
