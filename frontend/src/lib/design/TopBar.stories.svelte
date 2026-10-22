<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fn, userEvent, within, waitFor } from 'storybook/test';
  import TopBar from './TopBar.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import { spectreBuild, fireballBuild, novaBuild } from './fixtures';
  import { expectTopBarFits, expectPanelBelow, expectFloatingLabel, expectIconButton, expectSelectsFloating } from './stories/layout_checks';

  const loadoutLabel = async (root: HTMLElement) => {
    const c = within(root);
    const label = c.getByText('Loadout');
    await expect(label).toBeVisible();
    await expectFloatingLabel(label, c.getByRole('combobox', { name: 'Loadout' }));
  };
  const noPoints = async (root: HTMLElement) => {
    const bar = root.querySelector('header')!;
    for (const name of [/Passives/, /\bAsc\b/]) await expect(within(bar).queryByText(name)).toBeNull();
    await expect(bar.textContent).not.toMatch(/\d+ \/ \d+/);
  };

  const saveAsSpy = fn();

  const { Story } = defineMeta({
    title: 'Shell/TopBar',
    component: TopBar,
    parameters: { layout: 'fullscreen' },
    globals: { viewport: { value: 'target', isRotated: false } }
  });
</script>

<Story
  name="Spectre Necromancer"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(window.innerWidth).toBeGreaterThan(1800);
    await expect(c.getByText('Import / Export')).toBeVisible();
    await expect(c.getByText('Save')).toBeVisible();
    await expect(c.queryByRole('button', { name: 'Save As' })).toBeNull();
    await loadoutLabel(canvasElement);
    await noPoints(canvasElement);
    await expectSelectsFloating(canvasElement);
  }}><LiveTopBar fx={spectreBuild} /></Story>

<Story name="Vaal Fireball" asChild><LiveTopBar fx={fireballBuild} /></Story>

<Story name="Spark of the Nova (issues)" asChild><LiveTopBar fx={novaBuild} /></Story>

<Story name="Saved" asChild><LiveTopBar fx={spectreBuild} dirty={false} /></Story>

<Story
  name="Save As (saved build)"
  asChild
  globals={{ viewport: { value: 'min', isRotated: false } }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    await expectTopBarFits(canvasElement);
    const saveAs = c.getByRole('button', { name: 'Save As' });
    await expect(saveAs).toHaveAttribute('aria-haspopup', 'dialog');
    saveAsSpy.mockClear();
    await userEvent.click(saveAs);
    await expect(saveAsSpy).toHaveBeenCalledTimes(1);
  }}><LiveTopBar fx={spectreBuild} dirty={false} onsaveas={saveAsSpy} /></Story>

<Story
  name="Issues panel open"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expectPanelBelow(canvasElement, canvasElement.querySelector<HTMLElement>('[role="dialog"]')!);
  }}><LiveTopBar fx={novaBuild} panel="issues" /></Story>

<Story
  name="Layout 1280"
  asChild
  globals={{ viewport: { value: 'min', isRotated: false } }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => expectTopBarFits(canvasElement)}><LiveTopBar fx={novaBuild} /></Story>

<Story
  name="Layout 960"
  asChild
  globals={{ viewport: { value: 'half', isRotated: false } }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expect(window.innerWidth).toBeLessThanOrEqual(960);
    await expect(within(canvasElement).getByRole('button', { name: 'Save As' })).toBeVisible();
    await loadoutLabel(canvasElement);
    await expectTopBarFits(canvasElement);
  }}><LiveTopBar fx={spectreBuild} dirty={false} onsaveas={saveAsSpy} /></Story>

<Story
  name="Layout 1600"
  asChild
  globals={{ viewport: { value: 'mid', isRotated: false } }}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => expectTopBarFits(canvasElement)}><LiveTopBar fx={novaBuild} /></Story>

<Story name="Layout 1920" asChild play={async ({ canvasElement }: { canvasElement: HTMLElement }) => expectTopBarFits(canvasElement)}
  ><LiveTopBar fx={novaBuild} /></Story>

<Story
  name="Issues panel, keyboard"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const dlg = c.getByRole('dialog', { name: 'Build issues' });
    await expect(within(dlg).getAllByRole('listitem')).toHaveLength(novaBuild.sidebar.warnings.length);
    await expect(within(dlg).getByText('Your unreserved Mana is negative')).toBeVisible();
    await userEvent.click(within(dlg).getByRole('tab', { name: /Logs/ }));
    await expect(c.getByRole('dialog', { name: 'Logs' })).toBeVisible();
    await expect(c.getByRole('list', { name: 'Application log' })).toBeVisible();
    await userEvent.click(c.getByRole('button', { name: 'Error' }));
    await expect(within(c.getByRole('list', { name: 'Application log' })).getAllByRole('listitem')).toHaveLength(1);
    await userEvent.click(c.getByRole('button', { name: 'Copy for bug reports' }));
    await expect(c.getByRole('button', { name: 'Copied' })).toBeVisible();
    await userEvent.keyboard('{Escape}');
    await expect(c.queryByRole('dialog')).toBeNull();
    await expect(c.getByRole('button', { name: /^Logs/ })).toHaveFocus();
  }}><LiveTopBar fx={novaBuild} panel="issues" /></Story>

<Story
  name="History open"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const menu = c.getByRole('menu', { name: 'Undo history' });
    const entries = within(menu).getAllByRole('menuitem');
    await expect(entries[0].querySelector('svg')).not.toBeNull();
    await userEvent.hover(entries[0]);
    const tip = await within(document.body).findByRole('tooltip', { name: 'Stat change' });
    await expect(tip).toHaveTextContent('Life');
    await expect(tip).toHaveTextContent('+212 (+4.1%)');
    await expect(within(menu).getByText('Allocated Heart and Soul (+2 path nodes)')).toBeVisible();

    // the keyboard reaches each entry, shows its change and jumps there
    entries[3].focus();
    await waitFor(() => expect(within(document.body).getByRole('tooltip', { name: 'Stat change' })).toHaveTextContent('Spectre DPS'));
    await userEvent.keyboard('{ArrowDown}');
    await expect(entries[4]).toHaveFocus();
    await waitFor(() => expect(within(document.body).queryByRole('tooltip', { name: 'Stat change' })).toBeNull());
    await userEvent.keyboard('{ArrowUp}{Enter}');
    await waitFor(() => expect(c.queryByRole('menu')).toBeNull());
    await expect(c.getByRole('button', { name: 'Undo: Switched item set → Endgame (Ctrl+Z)' })).toBeEnabled();
  }}><LiveTopBar fx={spectreBuild} historyOpen /></Story>

<Story
  name="Compact 1280"
  asChild
  globals={{ viewport: { value: 'min', isRotated: false } }}
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(window.innerWidth).toBeLessThanOrEqual(1280);
    await loadoutLabel(canvasElement);
    await noPoints(canvasElement);
    await expectIconButton(c.getByRole('link', { name: 'Import / Export' }));
    await expectIconButton(c.getByRole('button', { name: 'Save (unsaved changes)' }));
    await expect(c.getByRole('link', { name: 'Import / Export' })).toHaveAttribute('title', 'Import / Export');
  }}><LiveTopBar fx={fireballBuild} /></Story>

<Story
  name="Interactive"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('link', { name: 'Items 3' }));
    await expect(c.getByRole('link', { name: 'Items 3' })).toHaveAttribute('aria-current', 'page');
    await userEvent.keyboard('5');
    await expect(c.getByRole('link', { name: 'Config 5' })).toHaveAttribute('aria-current', 'page');

    const save = c.getByRole('button', { name: 'Save (unsaved changes)' });
    await userEvent.click(save);
    await expect(c.getByRole('button', { name: 'Save (no unsaved changes)' })).toBeInTheDocument();
    await expect(c.getByRole('button', { name: 'Nothing to redo' })).toBeDisabled();
    await userEvent.click(c.getByRole('button', { name: 'Undo: Allocated Heart and Soul (+2 path nodes) (Ctrl+Z)' }));
    await expect(c.getByRole('button', { name: 'Undo: Activated Flask 2 (Quicksilver Flask) (Ctrl+Z)' })).toBeEnabled();
    await expect(c.getByRole('button', { name: 'Redo: Allocated Heart and Soul (+2 path nodes) (Ctrl+Y / Ctrl+Shift+Z)' })).toBeEnabled();
    await expect(c.getByRole('button', { name: 'Save (unsaved changes)' })).toBeInTheDocument();

    await userEvent.click(c.getByRole('button', { name: /^Undo history/ }));
    const menu = c.getByRole('menu', { name: 'Undo history' });
    await expect(within(menu).getByRole('menuitem', { current: true })).toHaveTextContent('Activated Flask 2 (Quicksilver Flask)');
    await userEvent.click(within(menu).getByRole('menuitem', { name: /Switched item set/ }));
    await waitFor(() => expect(c.queryByRole('menu')).toBeNull());
    await expect(c.getByRole('button', { name: 'Undo: Switched item set → Endgame (Ctrl+Z)' })).toBeEnabled();
    await expect(c.getByRole('button', { name: 'Redo: Set the main active skill to Raise Spectre (Ctrl+Y / Ctrl+Shift+Z)' })).toBeEnabled();

    await userEvent.pointer({ keys: '[MouseRight]', target: c.getByRole('button', { name: /^Undo: / }) });
    await expect(c.getByRole('menu', { name: 'Undo history' })).toBeVisible();
    await userEvent.keyboard('{Escape}');

    await userEvent.keyboard('{Control>}s{/Control}');
    await expect(c.getByRole('button', { name: 'Save (no unsaved changes)' })).toBeInTheDocument();
  }}><LiveTopBar fx={spectreBuild} /></Story>
