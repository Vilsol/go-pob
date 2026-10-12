<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import ImportExportPage from './ImportExportPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveImportExport from './stories/LiveImportExport.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { expectPageFits, expectDarkCheckboxes, expectSingleLine, expectSelectsFloating } from './stories/layout_checks';
  import { UNAVAILABLE as APP_UNAVAILABLE } from '$lib/shell/import_export';

  const { Story } = defineMeta({
    title: 'Pages/Import and export',
    component: ImportExportPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    return { c, main: c.getByRole('main', { name: 'Import / Export' }) };
  };
  const log = (el: HTMLElement) => within(el).getByRole('status', { name: 'Import log' });
  const radiosDrawn = async (root: HTMLElement) => {
    for (const r of root.querySelectorAll<HTMLInputElement>('input[type="radio"]')) {
      await expect(getComputedStyle(r).appearance).toBe('none');
      await expect(r.getBoundingClientRect().width).toBeGreaterThan(0);
    }
  };

  const shows = async (el: HTMLElement, fx: BuildFixture) => {
    const { c, main } = page(el);
    await expectPageFits(main);
    await expect(c.getByRole('textbox', { name: 'Build code' })).toHaveValue(fx.shareCode);
    for (const name of ['Build sharing', 'Build import', 'Character import']) await expect(c.getByRole('region', { name })).toBeVisible();
    const share = c.getByRole('region', { name: 'Build sharing' }).getBoundingClientRect();
    const imp = c.getByRole('region', { name: 'Build import' }).getBoundingClientRect();
    const chars = c.getByRole('region', { name: 'Character import' }).getBoundingClientRect();
    await expect(imp.left).toBeGreaterThanOrEqual(share.right);
    await expect(chars.top).toBeGreaterThanOrEqual(Math.max(share.bottom, imp.bottom));
    await expect(Math.round(chars.width)).toBe(Math.round(imp.right - share.left));
    await expectDarkCheckboxes(main);
    await radiosDrawn(main);
    await expectSingleLine([...main.querySelectorAll<HTMLElement>('.btn')]);
    await expectSelectsFloating(el);
  };
</script>

{#snippet frame(fx: BuildFixture, opts: { importText?: string; fetched?: boolean; unavailable?: typeof APP_UNAVAILABLE; mode?: 'replace' } = {})}
  <PageFrame label="Import / Export" mainClass="scroll grid grid-cols-2 content-start gap-4 py-4">
    {#snippet topbar()}<LiveTopBar {fx} current="#import" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveImportExport {fx} {...opts} />
  </PageFrame>
{/snippet}

<Story
  name="Spectre · target"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, spectreBuild)}>
  {@render frame(spectreBuild, { fetched: true })}
</Story>

<Story
  name="Fireball · minimum"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, fireballBuild)}>
  {@render frame(fireballBuild, { fetched: true })}
</Story>

<Story name="Nova · mid" asChild globals={vp('mid')} play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, novaBuild)}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Import a code"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const box = within(c.getByRole('region', { name: 'Build import' }));
    const field = box.getByRole('textbox', { name: /To import a build/ });
    const go = box.getByRole('button', { name: 'Import' });
    await expect(field).toHaveAccessibleDescription(/^Paste a code or a link/);
    await expect(go).toBeDisabled();
    await userEvent.click(field);
    await userEvent.paste(fireballBuild.shareCode);
    await waitFor(() => expect(field).toHaveAccessibleDescription('Code is valid'));
    await expect(go).toBeEnabled();
    await userEvent.click(box.getByRole('radio', { name: /Import to this build/ }));
    await userEvent.click(go);
    await expect(log(canvasElement)).toHaveTextContent(`import replace: ${fireballBuild.shareCode.slice(0, 24)}`);
    await userEvent.clear(field);
    await userEvent.paste(fireballBuild.shareCode.slice(0, 200));
    await waitFor(() => expect(field).toHaveAccessibleDescription('Invalid input'));
    await expect(go).toBeDisabled();
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Import a link"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const box = within(c.getByRole('region', { name: 'Build import' }));
    const field = box.getByRole('textbox', { name: /To import a build/ });
    await userEvent.type(field, 'https://pobb.in/aBcD1234');
    await expect(field).toHaveAccessibleDescription('URL is valid (pobb.in)');
    await userEvent.clear(field);
    await userEvent.type(field, 'https://poe.ninja/poe1/pob/1a2b');
    await expect(field).toHaveAccessibleDescription('URL is valid (PoeNinja)');
    await userEvent.click(box.getByRole('radio', { name: /Import as a comparison/ }));
    await userEvent.click(box.getByRole('button', { name: 'Import' }));
    await expect(log(canvasElement)).toHaveTextContent('import compare: https://poe.ninja/poe1/p');
    await userEvent.clear(field);
    await userEvent.type(field, 'https://example.com/x');
    await waitFor(() => expect(field).toHaveAccessibleDescription('Invalid input'));
  }}>
  {@render frame(novaBuild)}
</Story>

<!-- ImportTab.lua:609-628, 701-705 -->
<Story
  name="Import a character JSON"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    const box = within(c.getByRole('region', { name: 'Build import' }));
    const field = box.getByRole('textbox', { name: /To import a build/ });
    const go = box.getByRole('button', { name: 'Import' });
    await userEvent.click(field);
    await userEvent.paste('{"character":{"equipment":[]}}');
    await expect(field).toHaveAccessibleDescription('Invalid JSON format (equipment or passives missing)');
    await expect(go).toBeDisabled();
    await userEvent.clear(field);
    await userEvent.paste('{"character":{"level":90,"equipment":[],"passives":{"hashes":[]}}}');
    await expect(field).toHaveAccessibleDescription('JSON is valid');
    await userEvent.click(go);
    await expect(log(canvasElement)).toHaveTextContent('import new: {"character":{"level":90');
    await expect(box.getByRole('status')).toHaveTextContent('Items and skills successfully imported.');
    await expectPageFits(main);
  }}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Share"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const box = within(c.getByRole('region', { name: 'Build sharing' }));
    await expect(box.queryByRole('textbox', { name: 'Share link' })).toBeNull();
    await userEvent.selectOptions(box.getByRole('combobox', { name: 'Share via' }), 'pobb.in');
    await userEvent.click(box.getByRole('button', { name: 'Share' }));
    await expect(box.getByRole('textbox', { name: 'Share link' })).toHaveValue(`https://pobb.in/${spectreBuild.name}`);
    await userEvent.click(box.getByRole('button', { name: 'Copy code' }));
    await expect(log(canvasElement)).toHaveTextContent('copied');
    await userEvent.click(box.getByRole('checkbox', { name: /Export support/ }));
    await expect(box.getByRole('checkbox', { name: /Export support/ })).toBeChecked();
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Character import"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    const box = within(c.getByRole('region', { name: 'Character import' }));
    const fetch = box.getByRole('button', { name: 'Fetch characters' });
    await expect(fetch).toBeDisabled();
    await expect(box.getByText('Not fetched')).toBeVisible();
    await userEvent.type(box.getByRole('textbox', { name: 'Account name' }), 'Exile#1234');
    await userEvent.click(fetch);
    await expect(box.getByText('Fetched 4 characters')).toBeVisible();
    const chars = () => within(box.getByRole('group', { name: 'Characters' })).getAllByRole('button');
    await expect(chars()).toHaveLength(3);
    await expect(chars()[0]).toHaveAttribute('aria-pressed', 'true');
    await userEvent.click(chars()[2]);
    await expect(chars()[2]).toHaveAttribute('aria-pressed', 'true');
    await userEvent.click(box.getByRole('button', { name: 'Import tree' }));
    await expect(log(canvasElement)).toHaveTextContent('tree of MineShadow');
    await userEvent.selectOptions(box.getByRole('combobox', { name: 'League' }), 'Hardcore');
    await expect(chars()).toHaveLength(1);
    await userEvent.click(box.getByRole('button', { name: 'Import items and skills' }));
    await expect(log(canvasElement)).toHaveTextContent('items of LSJugg');
    await expectPageFits(main);
    await expectDarkCheckboxes(main);
    await expectSelectsFloating(canvasElement);
  }}>
  {@render frame(fireballBuild)}
</Story>

<Story
  name="In the app"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    const share = within(c.getByRole('region', { name: 'Build sharing' }));
    await expect(share.getByRole('button', { name: 'Copy code' })).toBeEnabled();
    for (const el of [share.getByRole('button', { name: 'Share' }), share.getByRole('combobox', { name: 'Share via' })]) {
      await expect(el).toBeDisabled();
      await expect(el).toHaveAttribute('title', APP_UNAVAILABLE.share);
    }
    const exportSupport = share.getByRole('checkbox', { name: /Export support/ });
    await expect(exportSupport).toBeEnabled();
    await userEvent.click(exportSupport);
    await expect(log(canvasElement)).toHaveTextContent('export support: true');
    const box = within(c.getByRole('region', { name: 'Build import' }));
    await expect(box.getByRole('radio', { name: /Import to this build/ })).toBeChecked();
    await expect(box.getByRole('radio', { name: /Import as a new build/ })).toBeEnabled();
    await expect(box.getByRole('radio', { name: /Import as a comparison/ })).toBeEnabled();
    const field = box.getByRole('textbox', { name: /To import a build/ });
    await userEvent.type(field, 'https://pobb.in/aBcD1234');
    await expect(field).toHaveAccessibleDescription('URL is valid (pobb.in)');
    await expect(box.getByRole('button', { name: 'Import' })).toBeEnabled();
    await userEvent.clear(field);
    await userEvent.paste(fireballBuild.shareCode);
    await waitFor(() => expect(field).toHaveAccessibleDescription('Code is valid'));
    await userEvent.click(box.getByRole('button', { name: 'Import' }));
    await expect(log(canvasElement)).toHaveTextContent(`import replace: ${fireballBuild.shareCode.slice(0, 24)}`);
    const chars = within(c.getByRole('region', { name: 'Character import' }));
    await expect(chars.getByText('Not available')).toBeVisible();
    await expect(chars.getByText(APP_UNAVAILABLE.characters)).toBeVisible();
    await expect(chars.getByRole('button', { name: 'Fetch characters' })).toBeDisabled();
    await expect(chars.getByRole('textbox', { name: 'Account name' })).toBeDisabled();
    await expectPageFits(main);
    await expectDarkCheckboxes(main);
    await radiosDrawn(main);
  }}>
  {@render frame(spectreBuild, { unavailable: APP_UNAVAILABLE, mode: 'replace' })}
</Story>
