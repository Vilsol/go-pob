<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Toolbar from './Toolbar.svelte';

  const { Story } = defineMeta({ title: 'Spec/Toolbar', component: Toolbar });
</script>

<script lang="ts">
  import Input from './Input.svelte';
  import Button from './Button.svelte';
</script>

<Story
  name="Arrow keys"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const bar = within(c.getByRole('toolbar', { name: 'Formatting' }));
    bar.getByRole('button', { name: 'Bold' }).focus();
    await userEvent.keyboard('{ArrowRight}');
    await expect(bar.getByRole('button', { name: 'Italic' })).toHaveFocus();
    await userEvent.keyboard('{ArrowRight}');
    await expect(bar.getByRole('button', { name: 'Clear' }), 'a disabled button is skipped').toHaveFocus();
    await userEvent.keyboard('{Home}');
    await expect(bar.getByRole('button', { name: 'Bold' })).toHaveFocus();
    await userEvent.keyboard('{End}');
    await expect(bar.getByRole('button', { name: 'Clear' })).toHaveFocus();
    const box = bar.getByRole('textbox', { name: 'Colour' });
    box.focus();
    await userEvent.keyboard('{ArrowLeft}');
    await expect(box, 'arrows stay in a text box').toHaveFocus();
  }}>
  <div class="flex w-120 flex-col gap-3">
    <Toolbar label="Formatting" class="panel gap-1 p-1.5">
      <Button size="sm">Bold</Button>
      <Button size="sm">Italic</Button>
      <Button size="sm" disabled>Underline</Button>
      <Input size="sm" class="w-24" aria-label="Colour" />
      <Button size="sm">Clear</Button>
    </Toolbar>
    <Toolbar edge="top" surface="bar" class="gap-2 px-3 py-2"><span class="note">An unlabelled strip is only styling</span></Toolbar>
  </div>
</Story>
