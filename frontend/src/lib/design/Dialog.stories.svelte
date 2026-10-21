<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import Dialog from './Dialog.svelte';
  import Button from './Button.svelte';

  const { Story } = defineMeta({ title: 'Spec/Dialog', component: Dialog });
</script>

<script lang="ts">
  let confirm = $state(false);
  let sheet = $state(false);
  let grown = $state(false);
</script>

<Story
  name="Confirm delete"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const opener = c.getByRole('button', { name: 'Delete…' });
    await userEvent.click(opener);
    const dlg = c.getByRole('dialog', { name: 'Delete “Spectre Necro – budget”?', hidden: true });
    await waitFor(() => expect(dlg).toHaveAttribute('open'));
    await expect(dlg).toBeVisible();
    await userEvent.keyboard('{Escape}');
    await waitFor(() => expect(dlg).not.toHaveAttribute('open'));
    await expect(dlg).not.toBeVisible();
    await expect(opener).toHaveFocus();
    await userEvent.click(opener);
    await waitFor(() => expect(dlg).toHaveAttribute('open'));
    await userEvent.click(c.getByRole('button', { name: 'Cancel' }));
    await waitFor(() => expect(dlg).not.toHaveAttribute('open'));
  }}>
  <Button variant="danger" onclick={() => (confirm = true)}>Delete…</Button>
  <Dialog bind:open={confirm} title="Delete “Spectre Necro – budget”?">
    <span class="text-13 text-text-2">The build is removed from this browser. Export it first to keep a copy.</span>
    {#snippet actions()}
      <Button size="md" variant="ghost" class="px-3" onclick={() => (confirm = false)}>Cancel</Button>
      <Button size="md" variant="danger-solid" onclick={() => (confirm = false)}>Delete</Button>
    {/snippet}
  </Dialog>
</Story>

<Story
  name="Sheet"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('button', { name: 'Open sheet' }));
    const dlg = c.getByRole('dialog', { name: 'Sheet', hidden: true });
    await waitFor(() => expect(dlg).toBeVisible());
    await userEvent.click(c.getByRole('button', { name: 'Close' }));
    await waitFor(() => expect(dlg).not.toHaveAttribute('open'));
  }}>
  <Button onclick={() => (sheet = true)}>Open sheet</Button>
  <Dialog bind:open={sheet} sheet title="Sheet" note=".dialog.dialog-sheet + .sheet-head">
    <div class="note px-4.5 py-3">Settings, About and the item editor: a full-size overlay whose sections carry their own padding.</div>
  </Dialog>
</Story>

<Story
  name="Top anchor keeps its top while content grows"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const top = async (name: string) => {
      grown = false;
      const dlg = c.getByRole('dialog', { name, hidden: true });
      await waitFor(() => expect(dlg).toHaveAttribute('open'));
      const before = dlg.getBoundingClientRect().top;
      grown = true;
      await waitFor(() => expect(dlg.getBoundingClientRect().height).toBeGreaterThan(300));
      return [before, dlg.getBoundingClientRect().top];
    };
    const [a, b] = await top('Top anchored');
    await expect(Math.abs(a - b), 'a top-anchored dialog does not move as it grows').toBeLessThanOrEqual(1);
    c.getByRole('dialog', { name: 'Top anchored', hidden: true }).querySelector<HTMLButtonElement>('button')?.click();
  }}>
  <Dialog open title="Top anchored" size="md" anchor="top">
    <span class="text-13">Results arrive after the dialog opens.</span>
    {#if grown}<div class="h-80">Late preview</div>{/if}
  </Dialog>
</Story>

<Story
  name="Centred dialog re-centres as content grows"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    grown = false;
    const dlg = c.getByRole('dialog', { name: 'Centred', hidden: true });
    await waitFor(() => expect(dlg).toHaveAttribute('open'));
    const before = dlg.getBoundingClientRect().top;
    grown = true;
    await waitFor(() => expect(dlg.getBoundingClientRect().height).toBeGreaterThan(300));
    await expect(before - dlg.getBoundingClientRect().top, 'control: a centred dialog moves up as it grows').toBeGreaterThan(50);
  }}>
  <Dialog open title="Centred" size="md">
    <span class="text-13">Results arrive after the dialog opens.</span>
    {#if grown}<div class="h-80">Late preview</div>{/if}
  </Dialog>
</Story>
