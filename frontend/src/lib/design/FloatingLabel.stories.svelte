<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, within } from 'storybook/test';
  import FloatingLabel from './FloatingLabel.svelte';
  import Select from './Select.svelte';

  const { Story } = defineMeta({ title: 'Spec/FloatingLabel', component: FloatingLabel });
</script>

<Story
  name="Toolbar selects"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getByRole('combobox', { name: 'Ascendancy' })).toHaveValue('Necromancer');
    await expect(c.getByRole('combobox', { name: 'Tree' })).toHaveValue('3.29');
    for (const l of canvasElement.querySelectorAll('[data-floating-label]')) await expect(getComputedStyle(l).userSelect).toBe('none');
  }}>
  <div class="flex gap-2.5">
    <FloatingLabel label="Ascendancy"><Select size="lg" class="h-9.5 w-45" options={['Necromancer', 'Elementalist', 'Occultist']} /></FloatingLabel>
    <FloatingLabel label="Tree"><Select size="lg" class="h-9.5 w-25" options={['3.29', '3.28']} /></FloatingLabel>
  </div>
</Story>
