<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within, waitFor } from 'storybook/test';
  import SkillsPage from '../SkillsPage.svelte';
  import LiveMobile from '../stories/LiveMobile.svelte';
  import LivePhoneSkills from '../stories/LivePhoneSkills.svelte';
  import { spectreBuild, fireballBuild } from '../fixtures';
  import { phone, expectPhoneFits } from '../stories/mobile_checks';

  const { Story } = defineMeta({
    title: 'Mobile/Skills',
    component: SkillsPage,
    parameters: {
      layout: 'fullscreen',
      docs: {
        description: {
          component:
            'Separate draft: the desktop shows the group list and the group editor side by side, which a phone cannot; here they are two full-width panes you swipe between or pick with connected tabs, and tapping a group slides to its editor. In the editor the drag column is gone and each gem’s socket bubble is its on/off switch (an empty middle = disabled).'
        }
      }
    },
    globals: phone
  });

  const inView = (el: HTMLElement) => {
    const r = el.getBoundingClientRect();
    return r.left >= -1 && r.right <= 391;
  };
</script>

<Story
  name="Phone"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Skills');
    const c = within(canvasElement);
    const list = c.getByRole('region', { name: 'Socket groups' });
    await expect(inView(list)).toBe(true);
    await userEvent.click(
      within(list)
        .getAllByRole('button')
        .find((b) => b.hasAttribute('aria-current'))!
    );
    await waitFor(() => expect(inView(c.getByRole('region', { name: 'Selected group' }))).toBe(true));
    await expect(c.getByRole('tab', { name: 'Selected group' })).toHaveAttribute('aria-selected', 'true');
  }}>
  <LiveMobile fx={spectreBuild} current="#skills" label="Skills" mainClass="overflow-hidden"><LivePhoneSkills fx={spectreBuild} /></LiveMobile>
</Story>

<Story
  name="Group editor"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPhoneFits(canvasElement, 'Skills');
    const ed = within(canvasElement).getByRole('region', { name: 'Selected group' });
    await expect(within(ed).queryAllByRole('checkbox', { name: / enabled$/ })).toEqual([]);
    const on = within(ed).getAllByRole('button', { name: / enabled$/ })[0];
    await expect(on).toHaveAttribute('aria-pressed', 'true');
    await expect(on.querySelector('.socket-gem')).not.toBeNull();
    await userEvent.click(on);
    await waitFor(() => expect(on).toHaveAttribute('aria-pressed', 'false'));
    await expect(on.querySelector('.socket-gem')).toBeNull();
    const grip = ed.querySelector('.gem-row > span:first-child')!;
    await expect(getComputedStyle(grip).display).toBe('none');
  }}>
  <LiveMobile fx={fireballBuild} current="#skills" label="Skills" mainClass="overflow-hidden"><LivePhoneSkills fx={fireballBuild} pane={1} /></LiveMobile>
</Story>
