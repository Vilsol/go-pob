<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import TreePage from '../TreePage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LivePhoneTree from '../stories/LivePhoneTree.svelte';
  import { spectreBuild, novaBuild } from '../fixtures';
  import { phone, phoneTall, treeDelta, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Tree',
    component: TreePage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not(.map-note, .map-note *)' }] } },
      docs: {
        description: {
          component:
            'Separate draft: a tap is hover (the node tooltip and the path a double tap would take), a double tap on the same node allocates or unallocates that path with an Undo toast, a hold opens the right-click menu, and one or two fingers pan and pinch-zoom (any of which cancels the tap). The page around it only gains phone padding and a sideways-scrolling bottom bar.'
        }
      }
    },
    globals: phone
  });

  const node = (el: HTMLElement) => el.querySelector<HTMLElement>('[data-phone-canvas] button.tree-node[data-alloc="false"]')!;
</script>

<Story
  name="Phone"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Tree');
    const c = within(canvasElement);
    await expect(c.getByRole('region', { name: 'Passive tree' })).toBeVisible();
    const n = node(canvasElement);
    await userEvent.click(n);
    await waitFor(() => expect(n).toHaveAttribute('data-hover', 'true'));
    await expect(document.getElementById(n.getAttribute('aria-describedby')!)).toBeVisible();
    await expect(n).toHaveAttribute('data-alloc', 'false');
    await userEvent.click(canvasElement.querySelector<HTMLElement>('[data-phone-canvas]')!);
    await waitFor(() => expect(n).toHaveAttribute('data-hover', 'false'));
  }}>
  <LiveMobile fx={spectreBuild} current="#tree" label="Tree" delta={treeDelta(spectreBuild)}><LivePhoneTree fx={spectreBuild} /></LiveMobile>
</Story>

<Story
  name="Double tap allocates"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    const n = node(canvasElement);
    const id = n.dataset.node;
    await userEvent.dblClick(n);
    await waitFor(() => expect(n).toHaveAttribute('data-alloc', 'true'));
    const toast = await waitFor(() => c.getAllByRole('status').find((el) => /Allocated/.test(el.textContent ?? ''))!);
    await expect(toast).toHaveTextContent(/Allocated \d+/);
    await userEvent.click(within(toast).getByRole('button', { name: 'Undo' }));
    await waitFor(() => expect(canvasElement.querySelector(`[data-node="${id}"]`)).toHaveAttribute('data-alloc', 'false'));
  }}>
  <LiveMobile fx={spectreBuild} current="#tree" label="Tree"><LivePhoneTree fx={spectreBuild} /></LiveMobile>
</Story>

<Story
  name="Hold opens the node menu"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    const n = node(canvasElement);
    await userEvent.pointer({ keys: '[MouseLeft>]', target: n });
    await new Promise((r) => setTimeout(r, 600));
    await userEvent.pointer({ keys: '[/MouseLeft]', target: n });
    await waitFor(() => expect(c.getByRole('menu')).toBeVisible());
    await expect(n).toHaveAttribute('data-alloc', 'false');
  }}>
  <LiveMobile fx={spectreBuild} current="#tree" label="Tree"><LivePhoneTree fx={spectreBuild} /></LiveMobile>
</Story>

<Story name="Toolbar hidden" asChild globals={phoneTall} play={async ({ canvasElement }) => expectPhoneFits(canvasElement, 'Tree', 834)}>
  <LiveMobile fx={novaBuild} current="#tree" label="Tree"><LivePhoneTree fx={novaBuild} /></LiveMobile>
</Story>
