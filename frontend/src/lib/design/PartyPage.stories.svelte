<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, waitFor, within } from 'storybook/test';
  import PartyPage from './PartyPage.svelte';
  import PageFrame from './PageFrame.svelte';
  import LiveTopBar from './stories/LiveTopBar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import LiveParty from './stories/LiveParty.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { PARTY_CATEGORIES } from './party';
  import { expectPageFits, expectDarkCheckboxes, expectNotClipped, expectSelectsFloating } from './stories/layout_checks';

  const { Story } = defineMeta({
    title: 'Pages/Party',
    component: PartyPage,
    parameters: {
      layout: 'fullscreen',
      a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } }
    }
  });

  const vp = (value: 'min' | 'mid' | 'target') => ({ viewport: { value, isRotated: false } });
  const page = (el: HTMLElement) => {
    const c = within(el);
    return { c, main: c.getByRole('main', { name: 'Party' }) };
  };
  const cat = (el: HTMLElement, label: string) => within(page(el).main).getByRole('region', { name: label });
  const entries = (el: HTMLElement, label: string) => [...cat(el, label).querySelectorAll<HTMLElement>('[data-entry]')];
  const live = (el: HTMLElement, name: string) => page(el).main.querySelector<HTMLElement>(`[data-entry="${name}"]`)!.dataset.live;

  const shows = async (el: HTMLElement, fx: BuildFixture) => {
    const { c, main } = page(el);
    await expectPageFits(main);
    for (const k of PARTY_CATEGORIES) await expect(cat(el, k.label)).toBeVisible();
    const life = fx.compare.b.stats.flat().find((s) => s.key === 'Player:Life')!.v;
    await expect(within(cat(el, 'Party Member Stats')).getByText(String(Math.round(life)))).toBeVisible();
    await expect(c.getByText(`${fx.compare.b.name} (imported)`)).toBeVisible();
    const left = c.getByRole('region', { name: 'Party import' }).getBoundingClientRect();
    const right = c.getByRole('region', { name: 'Party effects' }).getBoundingClientRect();
    await expect(right.left).toBeGreaterThanOrEqual(left.right);
    await expectNotClipped([...main.querySelectorAll<HTMLElement>('[data-entry] .font-mono')]);
    await expectDarkCheckboxes(main);
    await expectSelectsFloating(el);
  };
</script>

{#snippet frame(fx: BuildFixture, opts: { advanced?: boolean; empty?: boolean } = {})}
  <PageFrame label="Party" mainClass="flex-row gap-4 py-4">
    {#snippet topbar()}<LiveTopBar {fx} current="#party" />{/snippet}
    {#snippet sidebar()}<LiveSidebar {fx} />{/snippet}
    <LiveParty {fx} {...opts} />
  </PageFrame>
{/snippet}

<Story
  name="Spectre · target"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, spectreBuild)}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Fireball · minimum"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, fireballBuild)}>
  {@render frame(fireballBuild)}
</Story>

<Story name="Nova · mid" asChild globals={vp('mid')} play={async ({ canvasElement }: { canvasElement: HTMLElement }) => shows(canvasElement, novaBuild)}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Members and disable"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    await expect(live(canvasElement, 'Temporal Chains')).toBe('true');
    await userEvent.click(c.getByRole('checkbox', { name: 'Curse support (imported) active' }));
    await expect(live(canvasElement, 'Temporal Chains')).toBe('false');
    await expect(live(canvasElement, 'Wrath')).toBe('true');
    await expect(within(main.querySelector<HTMLElement>('[data-member="curse"]')!).getByText(/^Curses, Enemy Conditions, Enemy Modifiers$/)).toBeVisible();
    await userEvent.click(c.getByRole('checkbox', { name: 'Disable Party Effects' }));
    await expect(live(canvasElement, 'Wrath')).toBe('false');
    await expect(within(c.getByRole('group', { name: 'Effect of the party' })).getByText('Party effects disabled')).toBeVisible();
    await userEvent.click(c.getByRole('checkbox', { name: 'Disable Party Effects' }));
    await expect(live(canvasElement, 'Wrath')).toBe('true');
    await expectDarkCheckboxes(main);
  }}>
  {@render frame(spectreBuild)}
</Story>

<Story
  name="Import and clear"
  asChild
  globals={vp('target')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c } = page(canvasElement);
    const imp = within(c.getByRole('region', { name: 'Party import' }));
    const dest = imp.getByRole('combobox', { name: 'Destination' });
    await userEvent.selectOptions(dest, 'Aura');
    await userEvent.click(imp.getByRole('button', { name: 'Clear' }));
    await expect(within(cat(canvasElement, 'Auras')).getByText('nothing imported')).toBeVisible();
    await expect(entries(canvasElement, 'Curses').length).toBe(2);
    const go = imp.getByRole('button', { name: 'Import' });
    await expect(go).toBeDisabled();
    const field = imp.getByRole('textbox', { name: /Enter a build code/ });
    await userEvent.type(field, 'https://pobb.in/party123');
    await expect(field).toHaveAccessibleDescription('URL is valid (pobb.in)');
    await userEvent.click(go);
    await expect(entries(canvasElement, 'Auras').map((e) => e.dataset.entry)).toEqual(['Wrath', 'Discipline']);
    await userEvent.click(imp.getByRole('checkbox', { name: 'Append' }));
    await userEvent.click(go);
    await expect(entries(canvasElement, 'Auras')).toHaveLength(2);
    await userEvent.selectOptions(dest, 'All');
    await userEvent.click(imp.getByRole('button', { name: 'Clear' }));
    await waitFor(() => expect(c.getAllByText('nothing imported')).toHaveLength(PARTY_CATEGORIES.length));
  }}>
  {@render frame(fireballBuild)}
</Story>

<Story
  name="Advanced info"
  asChild
  globals={vp('min')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    await userEvent.click(c.getByRole('checkbox', { name: 'Show Advanced Info' }));
    const auras = c.getByRole('textbox', { name: 'Auras buffer' });
    await expect((auras as HTMLTextAreaElement).value).toContain('Wrath\n74\n');
    await userEvent.click(auras);
    await userEvent.keyboard('{Control>}{End}{/Control}{Enter}Haste{Enter}40{Enter}---');
    await userEvent.click(c.getByRole('checkbox', { name: 'Show Advanced Info' }));
    await expect(entries(canvasElement, 'Auras').map((e) => e.dataset.entry)).toEqual(['Wrath', 'Discipline', 'Haste']);
    await expect(within(cat(canvasElement, 'Auras')).getByText('+40% effect')).toBeVisible();
    await expectPageFits(main);
  }}>
  {@render frame(novaBuild)}
</Story>

<Story
  name="Empty party"
  asChild
  globals={vp('mid')}
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const { c, main } = page(canvasElement);
    await expect(c.getAllByText('nothing imported')).toHaveLength(PARTY_CATEGORIES.length);
    await expect(c.queryByText('MEMBERS')).toBeNull();
    await expectPageFits(main);
  }}>
  {@render frame(spectreBuild, { empty: true })}
</Story>
