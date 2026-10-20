<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import Sidebar from './Sidebar.svelte';
  import LiveSidebar from './stories/LiveSidebar.svelte';
  import { spectreBuild, fireballBuild, novaBuild, type BuildFixture } from './fixtures';
  import { expectInset, expectCollapsed, expectSelectsFloating, expectGauge, expectCapped } from './stories/layout_checks';
  import { tone, type Tone } from './tone';
  import { VALUE_COLOR } from '$lib/display/format_stat';

  const { Story } = defineMeta({
    title: 'Shell/Sidebar',
    component: Sidebar,
    parameters: { layout: 'fullscreen' },
    globals: { viewport: { value: 'target', isRotated: false } }
  });

  const manyPins = [...new Set(spectreBuild.sidebar.sections[1].groups.flat().map((l) => l.label))].slice(0, 12).map((l) => `Player|${l}|1`);

  type Want = { tone?: Tone; cap?: string; pool?: string; cost?: string; free?: boolean };
  const GAUGED: Record<string, Want> = {
    'Fire Resistance': { tone: 'fire', cap: 'Fire Resist' },
    'Cold Resistance': { tone: 'cold', cap: 'Cold Resist' },
    'Lightning Resistance': { tone: 'light', cap: 'Lightning Resist' },
    'Chaos Resistance': { tone: 'chaos', cap: 'Chaos Resist' },
    'Block Chance': { cap: 'Block Chance' },
    'Spell Block Chance': { cap: 'Spell Block Chance' },
    'Spell Suppression Chance': { cap: 'Suppression Ch.' },
    'Evade Chance': { cap: 'Evade Chance' },
    'Melee Evade Chance': { cap: 'Evade Chance' },
    'Projectile Evade Chance': { cap: 'Evade Chance' },
    'Phys. Damage Reduction': { cap: 'Phys. Dmg. Reduct' },
    'Attack Dodge Chance': { free: true },
    'Spell Dodge Chance': { free: true },
    'Crit Chance': {},
    'Effective Crit Chance': {},
    'Hit Chance': {},
    'Total Explode Chance': {},
    'Unreserved Life': { tone: 'life', pool: 'Life' },
    'Unreserved Mana': { tone: 'mana', pool: 'Mana' },
    'Life Cost': { tone: 'life', cost: 'Life' },
    'Mana Cost': { tone: 'mana', cost: 'Mana' },
    'Energy Shield Cost': { tone: 'es', cost: 'Energy Shield' }
  };
  const num = (s: string) => parseFloat(s.replace(/,/g, '').replace('−', '-'));
  const rgb = (hex: string) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`;
  const calcsCap = (fx: BuildFixture, label: string) => {
    for (const s of fx.calcs.sections.filter((x) => x.applicable))
      for (const b of s.blocks ?? []) for (const r of b.rows) if (r.label === label) return r.gauge?.cap;
    throw new Error(`no Calcs row ${label}`);
  };

  const gaugesShown = async (el: HTMLElement, fx: BuildFixture) => {
    const heads = [...el.querySelectorAll<HTMLButtonElement>('aside button.stat-group-head[aria-controls]')];
    await expect(heads).toHaveLength(fx.sidebar.sections.length);
    let gauged = 0;
    for (const [i, head] of heads.entries()) {
      const lines = fx.sidebar.sections[i].groups.flat();
      const rows = [...document.getElementById(head.getAttribute('aria-controls')!)!.querySelectorAll<HTMLElement>('.stat-row')];
      const text = (r: HTMLElement) => r.querySelector('.stat-label')!.textContent!;
      const value = (r: HTMLElement) => r.querySelector('.stat-value')!.textContent!;
      const find = (label: string, pc: boolean) => rows.find((r) => text(r) === label && value(r).includes('%') === pc);
      for (const row of rows) {
        const label = text(row);
        const line = lines.find((l) => l.label === label)!;
        if (line.labelColor !== VALUE_COLOR)
          await expect(getComputedStyle(row, '::before').backgroundColor, `${label}: tick colour`).toBe(rgb(line.labelColor));
        const want = GAUGED[label];
        if (!want) {
          await expect(row.querySelector('[data-gauge]'), `${label}: no gauge`).toBeNull();
          continue;
        }
        gauged++;
        const v = num(value(row));
        const toneClass = want.tone && tone(want.tone);
        if (want.pool) {
          const share = num(value(find(label, true)!));
          await expectGauge(row, { pct: share, toneClass, res: Math.max(0, Math.min(100, 100 - share)), slack: 1 });
        } else if (want.cost) {
          const pc = value(row).includes('%');
          const pool = find(`Unreserved ${want.cost}`, pc) ?? (pc ? undefined : rows.find((r) => [`Total ${want.cost}`, want.cost].includes(text(r))));
          await expectGauge(row, { pct: (v / (pool ? num(value(pool)) : 100)) * 100, toneClass, slack: 1 });
        } else {
          await expectGauge(row, { pct: v, toneClass, slack: 1 });
        }
        if (want.free) continue;
        const cap = want.cap ? calcsCap(fx, want.cap) : undefined;
        if (cap !== undefined && cap < 100) await expectCapped(row, cap);
        else await expect(row.querySelector('[data-cap-zone]'), `${label}: no cap`).toBeNull();
      }
    }
    await expect(gauged, 'gauged rows').toBeGreaterThanOrEqual(8);
  };

  const tabTo = async (name: RegExp) => {
    for (let i = 0; i < 40 && !name.test(document.activeElement?.textContent ?? ''); i++) await userEvent.tab();
    return document.activeElement as HTMLElement;
  };
</script>

<Story
  name="Spectre Necromancer"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await expect(c.getByRole('combobox', { name: 'Main skill' })).toHaveDisplayValue('Raise Spectre');
    // Build.lua:468
    await expect(c.queryByRole('button', { name: 'Compare' })).toBeNull();
    await expect(c.getByRole('combobox', { name: 'Minion' })).toHaveDisplayValue('Forged Frostbearer');
    await expect(c.getByRole('combobox', { name: 'Minion skill' })).toBeEnabled();
    await expect(c.getByRole('button', { name: 'Minion' })).toHaveAttribute('aria-expanded', 'true');
    await expectSelectsFloating(canvasElement);
    await gaugesShown(canvasElement, spectreBuild);
  }}>
  <div class="h-[1028px]"><LiveSidebar fx={spectreBuild} /></div>
</Story>

<Story
  name="Group spacing"
  asChild
  play={async ({ canvasElement }) => {
    const gaps = (els: Element[]) => els.slice(1).map((el, i) => el.getBoundingClientRect().top - els[i].getBoundingClientRect().bottom);
    const stats = (canvasElement as HTMLElement).querySelector('[aria-label="Build stats"]')!;
    const sections = [...stats.querySelectorAll(':scope > div > .stat-group-head')].map((h) => h.parentElement!);
    await expect(sections.length).toBeGreaterThan(1);
    for (const g of gaps(sections)) await expect(g).toBeGreaterThanOrEqual(16);
    const groups = [...sections.at(-1)!.querySelectorAll('[data-stat-group]')];
    await expect(groups.length).toBeGreaterThan(1);
    for (const g of gaps(groups)) await expect(g).toBeGreaterThanOrEqual(12);
  }}>
  <div class="h-[1028px]"><LiveSidebar fx={spectreBuild} /></div>
</Story>

<Story
  name="Vaal Fireball (several actives)"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const active = c.getByRole('combobox', { name: 'Active skill' });
    await expect(active).toHaveDisplayValue('Vaal Fireball');
    await userEvent.selectOptions(active, 'Fireball');
    await expect(active).toHaveDisplayValue('Fireball');
    await expect(c.getByRole('combobox', { name: 'Skill part' })).toHaveDisplayValue('Explosion');
    await expectSelectsFloating(canvasElement);
    await gaugesShown(canvasElement, fireballBuild);
    const byskill = c.getByRole('button', { name: /Full DPS by skill/ });
    await expect(c.getByRole('list', { name: 'Full DPS by skill' })).toBeVisible();
    await userEvent.click(byskill);
    await expect(byskill).toHaveAttribute('aria-expanded', 'false');
    await expect(c.queryByRole('list', { name: 'Full DPS by skill' })).toBeNull();
  }}>
  <div class="h-[1028px]"><LiveSidebar fx={fireballBuild} /></div>
</Story>

<Story
  name="Narrowest width"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    const handle = c.getByRole('separator', { name: 'Resize the sidebar' });
    handle.focus();
    for (let i = 0; i < 10; i++) await userEvent.keyboard('{ArrowLeft}');
    await expect(handle).toHaveAttribute('aria-valuenow', '240');
    localStorage.removeItem('go-pob:sidebar-width');
    await expect(c.getByRole('list', { name: 'Full DPS by skill' })).toBeVisible();
    const rows: [HTMLElement, HTMLElement][] = [
      ...[...canvasElement.querySelectorAll<HTMLElement>('.stat-row')].map((r): [HTMLElement, HTMLElement] => [
        r.querySelector<HTMLElement>('.stat-label')!,
        r.querySelector<HTMLElement>('.stat-value')!
      ]),
      ...[...canvasElement.querySelectorAll<HTMLElement>('[aria-label="Full DPS by skill"] li > span:first-child')].map((r): [HTMLElement, HTMLElement] => [
        r.firstElementChild as HTMLElement,
        r.lastElementChild as HTMLElement
      ])
    ];
    await expect(rows.length).toBeGreaterThan(20);
    for (const [label, value] of rows) {
      const l = label.getBoundingClientRect();
      const v = value.getBoundingClientRect();
      const row = value.parentElement!.getBoundingClientRect();
      const name = `${label.textContent} | ${value.textContent}`;
      await expect(Math.abs(l.top - v.top), `${name}: the value stays on the label's line`).toBeLessThanOrEqual(2);
      await expect(value.scrollWidth, `${name}: the value is cut`).toBeLessThanOrEqual(value.clientWidth);
      await expect(v.right, `${name}: the value runs past its row`).toBeLessThanOrEqual(row.right + 0.5);
      await expect(l.left, `${name}: the label starts before the value`).toBeLessThan(v.left);
    }
    const cut = rows.filter(([label]) => label.scrollWidth > label.clientWidth);
    await expect(cut.length, 'at 240px some labels run under their value').toBeGreaterThan(0);
    for (const [, value] of cut) await expect(getComputedStyle(value, '::before').backgroundImage).toContain('gradient');
  }}>
  <div class="h-[1028px]"><LiveSidebar fx={fireballBuild} /></div>
</Story>

<Story
  name="Collapse a stat group"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const heads = [...canvasElement.querySelectorAll<HTMLButtonElement>('button.stat-group-head[aria-controls]')];
    await expect(heads.length).toBeGreaterThan(0);
    for (const head of heads) {
      const body = document.getElementById(head.getAttribute('aria-controls')!)!;
      const open = body.getBoundingClientRect().height;
      await expect(open).toBeGreaterThan(0);
      await userEvent.click(head);
      await expect(head).toHaveAttribute('aria-expanded', 'false');
      await expectCollapsed(body);
      await userEvent.click(head);
      await expect(body.getBoundingClientRect().height).toBe(open);
    }
  }}>
  <div class="h-[1028px]"><LiveSidebar fx={spectreBuild} /></div>
</Story>

<Story
  name="Spark of the Nova (status)"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    const statuses = c.getAllByRole('status');
    await expect(statuses).toHaveLength(novaBuild.sidebar.warnings.filter((w) => w.kind === 'cost' || w.kind === 'skill').length);
    for (const st of statuses) {
      await expect(getComputedStyle(st).display).toBe('block');
      await expect(parseFloat(getComputedStyle(st).paddingLeft)).toBeGreaterThanOrEqual(8);
      await expect(parseFloat(getComputedStyle(st).paddingRight)).toBeGreaterThanOrEqual(8);
    }
    const aside = canvasElement.querySelector<HTMLElement>('aside')!;
    const top = aside.firstElementChild as HTMLElement;
    await expectInset(top, [...top.children] as HTMLElement[], 12, ['left', 'right', 'top']);
    const last = (top.lastElementChild as HTMLElement).getBoundingClientRect();
    await expect(top.getBoundingClientRect().bottom - last.bottom).toBeGreaterThanOrEqual(12);
    await gaugesShown(canvasElement, novaBuild);
  }}>
  <div class="h-[1028px]"><LiveSidebar fx={novaBuild} /></div>
</Story>

<Story
  name="Rail with pins"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const rail = canvasElement.querySelector<HTMLElement>('aside')!;
    await expectInset(rail, [...rail.querySelectorAll<HTMLElement>('button, span:not(.sr-only)')], 8, ['left', 'right']);
  }}>
  <div class="h-180"><LiveSidebar fx={spectreBuild} collapsed pinned={['Player|Total Life|1', 'Player|Energy Shield|1', 'Minion|Hit DPS|1']} /></div>
</Story>

<Story
  name="Rail with many pins"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const rail = canvasElement.querySelector<HTMLElement>('aside')!;
    const pins = rail.querySelector<HTMLElement>('[data-rail-pins]')!;
    await expect(pins.children).toHaveLength(12);
    await expect(rail.getBoundingClientRect().bottom).toBeLessThanOrEqual(canvasElement.querySelector('div')!.getBoundingClientRect().bottom + 1);
    await expect(pins.scrollHeight).toBeGreaterThan(pins.clientHeight);
    await expectInset(
      rail,
      [...rail.querySelectorAll<HTMLElement>('button, span:not(.sr-only)')].filter(
        (x) => x.getBoundingClientRect().bottom <= rail.getBoundingClientRect().bottom
      ),
      8,
      ['left', 'right']
    );
  }}>
  <div class="h-120">
    <LiveSidebar fx={spectreBuild} collapsed pinned={manyPins} />
  </div>
</Story>

<Story
  name="Minion item sets"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    const select = c.getByRole('combobox', { name: 'Minion' });
    await expect(select).toHaveDisplayValue('AG');
    await userEvent.hover(select);
    await expect(c.getByRole('tooltip')).toHaveTextContent("Helmet: Ylfeban's Trickery, Hubris Circlet");
    await expect(select).toHaveAccessibleDescription(/Dying Breath/);
    await userEvent.unhover(select);
  }}>
  <div class="h-180">
    <LiveSidebar
      fx={{
        ...spectreBuild,
        socketGroups: ['Animate Guardian'],
        mainSocketGroup: 0,
        mainSkill: {
          ...spectreBuild.mainSkill,
          Skills: ['Animate Guardian'],
          SkillEnabled: false,
          MainActiveSkill: 1,
          MinionLibraryShown: false,
          MinionSkillShown: false,
          Minion: 2,
          Minions: [
            {
              Label: 'Default Item Set',
              MinionID: '',
              ItemSetID: 1,
              Tooltip: ['^7Weapon 1: ^xFFFF77New Item, Void Sceptre', '^7Amulet: ^xAF6025Astramentis, Onyx Amulet']
            },
            {
              Label: 'AG',
              MinionID: '',
              ItemSetID: 2,
              Tooltip: [
                '^7Weapon 1: ^xAF6025Dying Breath, Iron Staff',
                "^7Helmet: ^xAF6025Ylfeban's Trickery, Hubris Circlet",
                "^7Body Armour: ^xAF6025Zahndethus' Cassock, Sage's Robe"
              ]
            }
          ]
        }
      } as BuildFixture} />
  </div>
</Story>

<Story name="Minimum screen" asChild globals={{ viewport: { value: 'min', isRotated: false } }}>
  <div class="h-180"><LiveSidebar fx={fireballBuild} /></div>
</Story>

<Story
  name="Interactive"
  asChild
  play={async ({ canvasElement }) => {
    const c = within(canvasElement);
    await userEvent.click(c.getByRole('button', { name: 'Pin Strength to the collapsed rail' }));
    await expect(c.getByRole('button', { name: 'Unpin Strength' })).toHaveAttribute('aria-pressed', 'true');

    await userEvent.click(c.getByRole('button', { name: 'Collapse the sidebar to a rail' }));
    const rail = c.getByRole('complementary', { name: 'Main skill and stats (collapsed)' });
    await expect(within(rail).getByText('Strength: 227')).toBeInTheDocument();
    await expect(c.getByRole('button', { name: 'Expand the sidebar' })).toHaveFocus();
    await userEvent.keyboard('{Enter}');
    await expect(c.getByRole('complementary', { name: 'Main skill and stats' })).toBeVisible();
    await expect(c.getByRole('button', { name: 'Collapse the sidebar to a rail' })).toHaveFocus();
    await expect(c.getByRole('button', { name: 'Unpin Strength' })).toBeInTheDocument();

    const row = await tabTo(/^Total Life/);
    await expect(row).toHaveFocus();
    await expect(c.getByRole('tooltip')).toBeVisible();
    await expect(row).toHaveAccessibleDescription(/Heart and Soul/);
    await userEvent.tab();
    await userEvent.tab();
    await expect(row).not.toHaveAttribute('aria-describedby');
  }}>
  <div class="h-[1028px]"><LiveSidebar fx={spectreBuild} /></div>
</Story>
