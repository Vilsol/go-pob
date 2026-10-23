<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fn, userEvent, waitFor } from 'storybook/test';
  import ItemsBoard from './ItemsBoard.svelte';
  import type { ItemSummary } from './fixtures';

  const item = (name: string, base: string, cells: [number, number], sockets = '', rarity: ItemSummary['rarity'] = 'rare'): ItemSummary => ({
    name,
    base,
    rarity,
    sockets,
    implicits: [],
    explicits: [],
    flask: /Flask$/.test(base),
    fits: [],
    cells
  });

  // synthetic items: every size class the board has, socket lines of each shape
  const ITEMS: Record<string, ItemSummary> = {
    'Weapon 1': item('Storm Song', 'Thicket Bow', [2, 4], 'G-G-G-G-G-G'),
    'Weapon 2': item('Doom Twig', 'Imbued Wand', [1, 3], 'B-B-R'),
    Helmet: item('Grim Visor', 'Hubris Circlet', [2, 2], 'R-G B W'),
    'Body Armour': item('Carcass Jack', 'Varnished Coat', [2, 3], 'R-R-G-G B-B', 'unique'),
    Gloves: item('Rift Grip', 'Fingerless Silk Gloves', [2, 2], 'R-R-R'),
    'Ring 1': item('Vortex Loop', 'Unset Ring', [1, 1], 'W'),
    Boots: item('Abyss Tread', 'Two-Toned Boots', [2, 2], 'R-G-A'),
    Belt: item('Dire Cord', 'Leather Belt', [2, 1]),
    'Flask 1': item('Divine Life Flask', 'Divine Life Flask', [1, 2], '', 'normal')
  };
  const slots = Object.fromEntries(Object.entries(ITEMS).map(([slot, it], i) => [slot, { id: i + 1, item: it }]));
  const socketed = {
    'Weapon 1': [
      {
        group: 0,
        gems: ['Tornado Shot', 'Greater Multiple Projectiles', 'Mirage Archer', 'Elemental Damage with Attacks'].map((name, index) => ({ index, name }))
      },
      { group: 1, gems: [{ index: 0, name: 'Blood Rage' }] }
    ],
    'Body Armour': [{ group: 2, gems: ['Cyclone', 'Melee Physical Damage'].map((name, index) => ({ index, name })) }]
  };
  const onedit = fn();
  const gemtip = fn((group: number, gem: number) => Promise.resolve([{ Text: `Gem ${gem + 1} of group ${group + 1}`, Separator: false }]));

  const { Story } = defineMeta({
    title: 'Pages/Items/Board',
    component: ItemsBoard,
    args: {
      slots,
      flasks: { 'Flask 1': true },
      calcFlasks: { 'Flask 1': true },
      fitting: () => [],
      jewels: [],
      jewelOptions: [],
      onedit,
      onflask: fn(),
      onequip: fn(),
      socketed,
      gemtip
    },
    // the art falls back to the base's initials here, coloured by rarity
    parameters: { a11y: { config: { rules: [{ id: 'color-contrast', selector: '*:not([data-pob-color])' }] } } }
  });

  const tile = (root: HTMLElement, slot: string) => root.querySelector<HTMLElement>(`button[aria-label^="${slot}: "]`)!;
  const rect = (el: Element) => el.getBoundingClientRect();
  const centres = (t: HTMLElement) =>
    [...t.parentElement!.querySelectorAll('[data-socket]')].map((s) => {
      const r = rect(s);
      return { x: r.left + r.width / 2, y: r.top + r.height / 2 };
    });
</script>

<Story
  name="Art in proportion"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    for (const [slot, it] of Object.entries(ITEMS)) {
      const t = tile(canvasElement, slot);
      const box = rect(t.querySelector('[data-item-icon]')!);
      const [w, h] = it.cells!;
      await expect(Math.abs(box.width / box.height - w / h), slot).toBeLessThan(0.02);
      // centred in the slot, as large as fits with the slot's margin
      const s = rect(t);
      await expect(Math.abs(box.left + box.width / 2 - (s.left + s.width / 2)), slot).toBeLessThan(1);
      await expect(Math.min(s.width - box.width, s.height - box.height), slot).toBeLessThan(25);
      await expect(t.querySelector('[data-slot-caption]'), slot).toBeNull();
    }
  }} />

<Story
  name="Sockets on hover"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const helm = tile(canvasElement, 'Helmet');
    await userEvent.hover(helm);
    await waitFor(() => expect(helm.querySelector('[data-slot-caption]')?.textContent).toContain('Grim Visor'));
    const [a, b, c, d] = centres(helm);
    // 1 2 / 4 3
    await expect([a.y === b.y, b.x === c.x, c.y === d.y, d.x === a.x, a.x < b.x, a.y < d.y]).toEqual(Array(6).fill(true));
    await expect([...helm.parentElement!.querySelectorAll('[data-socket-link]')].map((l) => l.getAttribute('data-socket-link'))).toEqual(['0']);
    await expect(
      [...helm.parentElement!.querySelectorAll('[data-socket] [role="img"][aria-label$=" socket"]')].map((s) => s.getAttribute('aria-label'))
    ).toEqual(['Red socket', 'Green socket', 'Blue socket', 'White socket']);
    const wand = tile(canvasElement, 'Weapon 2');
    await userEvent.hover(wand);
    const col = centres(wand);
    await expect(col.map((p) => Math.round(p.x - col[0].x))).toEqual([0, 0, 0]);
    await expect(col[1].y > col[0].y && col[2].y > col[1].y).toBe(true);
    await expect(helm.parentElement!.querySelector('[data-item-sockets]')).toBeNull();
  }} />

<Story
  name="Six-link with gems"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const bow = tile(canvasElement, 'Weapon 1');
    await userEvent.hover(bow);
    await expect(bow.parentElement!.querySelectorAll('[data-socket-link]')).toHaveLength(5);
    const gems = [...bow.parentElement!.querySelectorAll('[data-socket]')].map(
      (s) => s.querySelector('[data-socket-gem]')?.getAttribute('data-socket-gem') ?? ''
    );
    await expect(gems).toEqual(['Tornado Shot', 'Greater Multiple Projectiles', 'Mirage Archer', 'Elemental Damage with Attacks', 'Blood Rage', '']);
    await userEvent.hover(bow.parentElement!.querySelector('[data-socket-gem="Blood Rage"]')!);
    await waitFor(() => expect(document.querySelector('[aria-label="Gem tooltip"]')?.textContent).toContain('Gem 1 of group 2'));
    await expect(gemtip).toHaveBeenCalledWith(1, 0);
    await userEvent.hover(tile(canvasElement, 'Body Armour'));
    await expect(document.querySelector('[aria-label="Gem tooltip"]')).toBeNull();
  }} />

<Story
  name="Keyboard gem tooltips"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const bow = tile(canvasElement, 'Weapon 1');
    const gemTip = () => document.querySelector('[aria-label="Gem tooltip"]')?.textContent;
    bow.focus();
    await waitFor(() => expect(bow.parentElement!.querySelector('[data-item-sockets]')).not.toBeNull());
    await userEvent.tab();
    await expect(document.activeElement?.getAttribute('data-socket-gem')).toBe('Tornado Shot');
    await waitFor(() => expect(gemTip()).toContain('Gem 1 of group 1'));
    await userEvent.keyboard('{ArrowRight}');
    await expect(document.activeElement?.getAttribute('data-socket-gem')).toBe('Greater Multiple Projectiles');
    await waitFor(() => expect(gemTip()).toContain('Gem 2 of group 1'));
    await userEvent.keyboard('{End}');
    await expect(document.activeElement?.getAttribute('data-socket-gem')).toBe('Blood Rage');
    await userEvent.keyboard('{Escape}');
    await expect(document.activeElement).toBe(bow);
    await expect(gemTip()).toBeUndefined();
    await userEvent.keyboard('{ArrowLeft}');
    await expect(document.activeElement?.getAttribute('data-socket-gem')).toBe('Blood Rage');
    (document.activeElement as HTMLElement).blur();
    await waitFor(() => expect(bow.parentElement!.querySelector('[data-item-sockets]')).toBeNull());
    await expect(gemTip()).toBeUndefined();
  }} />

<Story
  name="Abyssal socket"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const boots = tile(canvasElement, 'Boots');
    await userEvent.hover(boots);
    await expect(
      [...boots.parentElement!.querySelectorAll('[data-socket] [role="img"][aria-label$=" socket"]')].map((s) => s.getAttribute('aria-label'))
    ).toEqual(['Red socket', 'Green socket', 'Abyssal socket']);
    await expect(boots.parentElement!.querySelectorAll('[data-socket-link]')).toHaveLength(2);
  }} />

<Story
  name="Empty sockets"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const gloves = tile(canvasElement, 'Gloves');
    await userEvent.hover(gloves);
    await expect(gloves.parentElement!.querySelectorAll('[data-socket]')).toHaveLength(3);
    await expect(gloves.parentElement!.querySelectorAll('[data-socket-gem]')).toHaveLength(0);
  }} />

<Story
  name="Touch first tap"
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    onedit.mockClear();
    const body = tile(canvasElement, 'Body Armour');
    const tap = () => {
      body.dispatchEvent(new PointerEvent('pointerdown', { pointerType: 'touch', bubbles: true }));
      body.dispatchEvent(new MouseEvent('mouseenter'));
      body.click();
    };
    tap();
    await waitFor(() => expect(body.parentElement!.querySelector('[data-item-sockets]')).not.toBeNull());
    await expect(onedit).not.toHaveBeenCalled();
    tap();
    await expect(onedit).toHaveBeenCalledWith('Body Armour');
  }} />
