<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import StatRow from './StatRow.svelte';

  const { Story } = defineMeta({
    title: 'Spec/StatRow',
    component: StatRow,
    argTypes: {
      tone: { control: 'select', options: ['', 'life', 'mana', 'es', 'fire', 'cold', 'light', 'chaos', 'warn', 'muted'] }
    },
    args: { label: 'Total Life', value: '5,120', tone: 'life' },
    decorators: []
  });
</script>

<Story name="Plain" />

<Story name="Gauge with cap" args={{ label: 'Evade Chance', value: '52%', tone: undefined, gauge: { pct: 52, cap: 95 } }} />

<Story name="Over the limit" args={{ label: 'Mana Cost', value: '42', tone: 'warn', gauge: { pct: (42 / 38) * 100, warn: true } }} />

<Story name="Gauge rows" asChild>
  <div class="panel flex w-90 flex-col px-3 py-2.5">
    <StatRow label="Crit Chance" value="31.00%" gauge={{ pct: 31 }} />
    <StatRow label="Block Chance" value="28%" gauge={{ pct: 28, cap: 75 }} />
    <StatRow label="Fire Resistance" value="80% · max 80" tone="fire" gauge={{ pct: 80, cap: 80 }} />
    <StatRow label="Unreserved Mana" value="38 (4%)" tone="mana" gauge={{ pct: 4, res: 96 }} />
    <StatRow label="Mana Cost" value="42" tone="warn" gauge={{ pct: (42 / 38) * 100, warn: true }} />
    <StatRow label="Power Charges" value="3 / 3" gauge={{ pct: 100, seg: 3 }} />
  </div>
</Story>

<Story name="Sidebar group" asChild>
  <div class="flex w-80 flex-col bg-surface-1 p-2">
    <StatRow label="Total Life" value="5,120" tone="life" />
    <StatRow label="Unreserved Mana" value="38 (4%)" tone="mana" gauge={{ pct: 4, res: 96 }} alt />
    <StatRow label="Evade Chance" value="52%" gauge={{ pct: 52, cap: 95 }} />
    <StatRow label="Block Chance" value="28%" gauge={{ pct: 28, cap: 75 }} alt />
    <StatRow label="Mana Cost" value="42" tone="warn" gauge={{ pct: (42 / 38) * 100, warn: true }} />
    <StatRow label="Chaos Max Hit" value="4,410" tone="chaos" alt />
  </div>
</Story>
