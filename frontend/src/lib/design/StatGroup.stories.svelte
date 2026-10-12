<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import StatGroup from './StatGroup.svelte';
  import StatRow from './StatRow.svelte';
  import { expectCollapsed } from './stories/layout_checks';

  const { Story } = defineMeta({ title: 'Spec/StatGroup', component: StatGroup });
</script>

<Story
  name="Resistances"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    const head = c.getByRole('button', { name: /Resistances/ });
    await expect(head).toHaveAttribute('aria-expanded', 'true');
    await expect(c.getByText('Chaos Resistance')).toBeVisible();
    await userEvent.click(head);
    await expect(head).toHaveAttribute('aria-expanded', 'false');
    await expect(c.queryByText('Chaos Resistance')).not.toBeVisible();
    await expectCollapsed(document.getElementById(head.getAttribute('aria-controls')!)!);
    await userEvent.click(head);
    await expect(c.getByText('Chaos Resistance')).toBeVisible();
  }}>
  <div class="panel w-90 px-3 py-2.5">
    <StatGroup title="Resistances" warn="chaos res negative">
      <StatRow label="Fire Resistance" value="80% · max 80" tone="fire" />
      <StatRow label="Cold Resistance" value="76% · max 76" tone="cold" alt />
      <StatRow label="Lightning Resistance" value="76% · max 76" tone="light" />
      <StatRow label="Chaos Resistance" value="−12% · max 75" tone="chaos" alt />
    </StatGroup>
  </div>
</Story>
