<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import PowerReport from './PowerReport.svelte';
  import { showPowerRows, type PowerFilter, type PowerRow, type PowerSort, type ShownPowerRow } from './tree';

  const { Story } = defineMeta({ title: 'Spec/PowerReport', component: PowerReport });
  const ROWS: PowerRow[] = [
    { ID: 1, Name: 'Constitution', Type: 'Notable', Power: 412, PathPower: 137, PathDist: 3, Allocated: false, Stats: ['+30 to maximum Life'] },
    { ID: 2, Name: 'Heart of Oak', Type: 'Notable', Power: 380, PathPower: 190, PathDist: 2, Allocated: false },
    { ID: 3, Name: 'Life', Type: 'Normal', Power: 64, PathPower: 64, PathDist: 1, Allocated: false },
    { ID: 4, Name: 'Iron Reflexes', Type: 'Keystone', Power: -120, PathPower: -40, PathDist: 3, Allocated: false }
  ];
</script>

<script lang="ts">
  let stat = $state('Life');
  let depth = $state(0);
  let custom = $state('0');
  let filter = $state<PowerFilter>('unallocated');
  let masteries = $state(true);
  let sort = $state<PowerSort>('power');
  let focus = $state<number>();
  const rows = $derived(showPowerRows(ROWS, undefined, filter, masteries, (_, v) => v.toFixed(1), sort));
</script>

<Story
  name="Report"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const table = within(c.getByRole('table', { name: 'Node power' }));
    await expect(table.getByRole('columnheader', { name: 'Life' })).toHaveAttribute('aria-sort', 'descending');
    await userEvent.click(table.getByRole('button', { name: 'Points' }));
    await expect(table.getByRole('columnheader', { name: 'Points' })).toHaveAttribute('aria-sort', 'ascending');
    await expect(table.getAllByRole('row')[1]).toHaveTextContent('Life');
    await userEvent.click(table.getAllByRole('row')[2]);
    await expect(table.getAllByRole('row')[2]).toHaveClass('bg-surface-3');
  }}>
  <div class="flex h-120 w-115 flex-col border border-line bg-surface-1">
    <PowerReport
      stats={[
        { Stat: 'Life', Label: 'Life' },
        { Stat: 'FullDPS', Label: 'Full DPS' }
      ]}
      bind:stat
      bind:depth
      bind:custom
      bind:filter
      bind:masteries
      bind:sort
      statLabel={stat === 'Life' ? 'Life' : 'Full DPS'}
      heat={false}
      started
      {rows}
      progress={1}
      {focus}
      onheat={() => {}}
      onstart={() => {}}
      onstop={() => {}}
      onclose={() => {}}
      onhover={() => {}}
      onpick={(r: ShownPowerRow) => (focus = r.ID)} />
  </div>
</Story>

<Story name="Building" asChild>
  <div class="flex w-115 flex-col border border-line bg-surface-1">
    <PowerReport
      stats={[{ Stat: 'Life', Label: 'Life' }]}
      stat="Life"
      depth={0}
      custom="0"
      filter="unallocated"
      masteries
      sort="power"
      statLabel="Life"
      heat={false}
      started
      rows={undefined}
      progress={0.42}
      onheat={() => {}}
      onstart={() => {}}
      onstop={() => {}}
      onclose={() => {}}
      onhover={() => {}}
      onpick={() => {}} />
  </div>
</Story>
