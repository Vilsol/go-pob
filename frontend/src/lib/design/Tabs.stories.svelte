<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Tabs from './Tabs.svelte';
  import { panelId, tabId } from './tab_ids';
  import IssueRow from './IssueRow.svelte';
  import LogList from './LogList.svelte';

  const { Story } = defineMeta({ title: 'Spec/Tabs', component: Tabs });

  const tabs = [
    { id: 'issues', label: 'Build issues', badge: { text: '3', tone: 'warn' as const } },
    { id: 'logs', label: 'Logs', badge: { text: '1 error', tone: 'error' as const } }
  ];
  const lines = [
    { time: '12:04:01', level: 'info' as const, src: 'worker', msg: 'Calculation worker started (wasm 21.3 MB)' },
    { time: '12:11:15', level: 'error' as const, src: 'save', msg: 'Save failed: browser storage quota exceeded (retrying after cleanup)' }
  ];
</script>

<script lang="ts">
  let segSel = $state('groups');
</script>

<Story
  name="Issues and log"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const issues = c.getByRole('tab', { name: /Build issues/ });
    const logs = c.getByRole('tab', { name: /Logs/ });
    await expect(issues).toHaveAttribute('aria-selected', 'true');
    await expect(c.getByText('Chaos resistance is negative (−12%)')).toBeInTheDocument();
    await userEvent.click(logs);
    await expect(logs).toHaveAttribute('aria-selected', 'true');
    await expect(c.getByRole('list', { name: 'Application log' })).toBeInTheDocument();
    await expect(c.getByRole('tabpanel')).toHaveAccessibleName(/Logs/);
    await userEvent.keyboard('{ArrowLeft}');
    await expect(issues).toHaveAttribute('aria-selected', 'true');
    await expect(issues).toHaveFocus();
  }}>
  <div class="panel w-110 overflow-hidden">
    <Tabs {tabs} label="Panels">
      {#snippet panel(id)}
        {#if id === 'issues'}
          <IssueRow text="Chaos resistance is negative (−12%)" kind="Resistance" where="Resistances · sidebar" />
          <IssueRow text="Blade Vortex costs 42 mana; 38 unreserved" kind="Cost" where="Main skill · Blade Vortex" />
        {:else}
          <LogList {lines} />
        {/if}
      {/snippet}
    </Tabs>
  </div>
</Story>

<Story
  name="Segmented, panel elsewhere"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const b = c.getByRole('tab', { name: 'Selected group' });
    await userEvent.click(b);
    await expect(b).toHaveAttribute('aria-selected', 'true');
    await expect(c.getByRole('tabpanel')).toHaveAccessibleName('Selected group');
    await expect(b).toHaveAttribute('aria-controls', c.getByRole('tabpanel').id);
  }}>
  <div class="panel flex w-90 flex-col gap-2 p-3">
    <Tabs
      id="seg-demo"
      variant="segmented"
      fill
      label="Skills view"
      tabs={[
        { id: 'groups', label: 'Socket groups' },
        { id: 'group', label: 'Selected group' }
      ]}
      bind:selected={segSel} />
    <div role="tabpanel" id={panelId('seg-demo')} aria-labelledby={tabId('seg-demo', segSel)} class="note">{segSel}</div>
  </div>
</Story>
