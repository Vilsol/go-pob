<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import ListOption from './ListOption.svelte';
  import ModCell from './ModCell.svelte';

  const { Story } = defineMeta({ title: 'Spec/ListOption', component: ListOption });

  const groups = [
    {
      title: 'Default',
      cells: [
        { text: '80% increased global Accuracy Rating', k: 'ok' as const },
        { text: 'Crit Immune', k: 'bad' as const },
        { text: '# boss uptime assumptions', k: 'comment' as const }
      ]
    },
    {
      title: 'Spectre buffs',
      cells: [
        { text: 'Minions deal 15% more Damage', k: 'ok' as const },
        { text: '+1 to Level of all Minion Skill Gems', k: 'ok' as const }
      ]
    },
    { title: 'Uber boss what-ifs', cells: [{ text: 'Nearby Enemies take 10% increased Physical Damage', k: 'ok' as const }] }
  ];
</script>

<script lang="ts">
  let sel = $state(0);
  let enabled = $state([true, true, false]);
</script>

<Story
  name="Modifier groups"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getByRole('button', { name: 'Default' })).toHaveAttribute('aria-current', 'true');
    const cell = c.getByText('Crit Immune').closest('.mod-cell');
    await expect(cell).toHaveAttribute('data-off', 'false');
    await userEvent.click(c.getByRole('checkbox', { name: 'Enable Default' }));
    await expect(cell).toHaveAttribute('data-off', 'true');
    await userEvent.click(c.getByRole('button', { name: 'Spectre buffs' }));
    await expect(c.getByRole('button', { name: 'Spectre buffs' })).toHaveAttribute('aria-current', 'true');
    await expect(c.getByRole('button', { name: 'Default' })).not.toHaveAttribute('aria-current');
    await expect(c.getByText('Minions deal 15% more Damage')).toBeInTheDocument();
  }}>
  <div class="panel flex w-150 gap-2 p-2.5">
    <div role="group" aria-label="Mod groups" class="flex w-60 flex-col gap-1">
      {#each groups as g, i (g.title)}
        <ListOption label={g.title} count={g.cells.length} selected={sel === i} bind:enabled={enabled[i]} onselect={() => (sel = i)} />
      {/each}
    </div>
    <div class="flex flex-wrap content-start items-center gap-1.5">
      {#each groups[sel].cells as cell, i (cell.text)}
        <ModCell text={cell.text} k={cell.k} off={!enabled[sel]} hover={sel === 0 && i === 0} />
      {/each}
    </div>
  </div>
</Story>
