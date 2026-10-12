<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Field from './Field.svelte';
  import Input from './Input.svelte';
  import NumberInput from './NumberInput.svelte';
  import Select from './Select.svelte';
  import Checkbox from './Checkbox.svelte';

  const { Story } = defineMeta({ title: 'Spec/Field', component: Field });
</script>

<script lang="ts">
  let level = $state<number | null>(94);
  let boss = $state(true);
</script>

<Story
  name="Form"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const lvl = c.getByRole('spinbutton', { name: 'Level' });
    await userEvent.clear(lvl);
    await userEvent.type(lvl, '100{Enter}');
    await expect(lvl).toHaveValue('100');
    await expect(lvl).toHaveAttribute('aria-valuenow', '100');
    await expect(c.getByRole('combobox', { name: 'Main skill' })).toHaveValue('Raise Spectre');
    const box = c.getByRole('checkbox', { name: 'Enemy is a Boss' });
    await userEvent.click(box);
    await expect(box).not.toBeChecked();
  }}>
  <div class="flex w-110 flex-col gap-3.5">
    <Field label="Main skill" class="gap-1.5 text-13">
      <Select size="lg" class="bg-surface-1" options={['Raise Spectre', 'Blade Vortex', 'Stormblast Mine']} />
    </Field>
    <div class="flex gap-3">
      <Field label="Level" class="gap-1.5 text-13"><NumberInput size="lg" class="w-22 bg-surface-1 font-mono" bind:value={level} min={1} max={100} /></Field>
      <Field label="Search" class="grow gap-1.5 text-13">
        <Input size="lg" class="bg-surface-1" type="search" placeholder="Search nodes, stats, gems" />
      </Field>
    </div>
    <Checkbox bind:checked={boss}>Enemy is a Boss</Checkbox>
  </div>
</Story>

<Story name="Inline" asChild>
  <Field label="Level" inline><NumberInput size="xs" class="w-16 font-mono" value={94} min={1} max={100} /></Field>
</Story>
