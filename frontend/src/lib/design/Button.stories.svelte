<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect } from 'storybook/test';
  import Button from './Button.svelte';
  import Icon from './Icon.svelte';

  const { Story } = defineMeta({
    title: 'Spec/Button',
    component: Button,
    argTypes: {
      size: { control: 'inline-radio', options: ['2xs', 'xs', 'sm', 'default', 'md', 'lg'] },
      variant: { control: 'inline-radio', options: ['default', 'primary', 'ghost', 'danger', 'danger-solid', 'dashed', 'close', 'bare'] },
      icon: { control: 'boolean' },
      disabled: { control: 'boolean' }
    },
    args: { size: 'default', variant: 'default', icon: false, disabled: false }
  });
</script>

<script lang="ts">
  import Select from './Select.svelte';
</script>

<Story name="Playground">Import / Export</Story>

<Story name="Variants" asChild>
  <div class="flex items-center gap-2">
    <Button>Default</Button>
    <Button variant="primary">Primary</Button>
    <Button variant="ghost">Ghost</Button>
    <Button variant="danger">Delete…</Button>
    <Button variant="danger-solid">Delete</Button>
    <Button variant="dashed">+ Add group</Button>
    <Button disabled>Disabled</Button>
  </div>
</Story>

<Story
  name="Links and standalone variants"
  asChild
  play={async ({ canvasElement }) => {
    const root = canvasElement as HTMLElement;
    const link = root.querySelector('a[href="#import"]')!;
    await expect(link.className.split(' ')).toEqual(expect.arrayContaining(['btn', 'btn-md', 'btn-primary']));
    for (const name of ['Close', 'Bare']) {
      const el = root.querySelector(`[data-standalone="${name}"]`)!;
      await expect(el.tagName, name).toBe('BUTTON');
      await expect(el.classList.contains('btn'), name).toBe(false);
    }
  }}>
  <div class="flex items-center gap-2">
    <Button href="#import" size="md" variant="primary">Import build</Button>
    <Button variant="close" data-standalone="Close" aria-label="Close">×</Button>
    <Button variant="bare" data-standalone="Bare">Bare</Button>
  </div>
</Story>

<Story
  name="Hover states"
  asChild
  play={async ({ canvasElement }) => {
    const look = (el: Element) => {
      const s = getComputedStyle(el);
      return [s.backgroundColor, s.borderColor, s.color, s.textDecorationLine].join(' ');
    };
    const userEvent = await import('@vitest/browser/context').then((m) => m.userEvent).catch(() => undefined);
    if (!userEvent) return;
    for (const el of (canvasElement as HTMLElement).querySelectorAll<HTMLElement>('[data-hover-case]')) {
      await userEvent.unhover(el);
      const before = look(el);
      await userEvent.hover(el);
      if (el.matches(':disabled')) await expect(look(el), el.dataset.hoverCase).toBe(before);
      else await expect(look(el), el.dataset.hoverCase).not.toBe(before);
      await userEvent.unhover(el);
    }
  }}>
  <div class="flex flex-wrap items-center gap-2">
    <Button data-hover-case="default">Default</Button>
    <Button data-hover-case="primary" variant="primary">Primary</Button>
    <Button data-hover-case="ghost" variant="ghost">Ghost</Button>
    <Button data-hover-case="danger" variant="danger">Delete…</Button>
    <Button data-hover-case="danger-solid" variant="danger-solid">Delete</Button>
    <Button data-hover-case="dashed" variant="dashed">+ Add group</Button>
    <Button data-hover-case="icon" icon aria-label="Settings"><Icon name="gear" /></Button>
    <Button data-hover-case="icon xs" icon size="xs" aria-label="Info"><Icon name="info" /></Button>
    <Button data-hover-case="disabled" disabled>Disabled</Button>
    <Button data-hover-case="close" variant="close" aria-label="Close">×</Button>
    <Button data-hover-case="bare" variant="bare">Bare</Button>
    <button data-hover-case="chip" class="chip">Chip</button>
    <button data-hover-case="chip pressed" class="chip" aria-pressed="true">Pressed chip</button>
    <div class="seg"><button data-hover-case="seg" class="seg-btn">Seg</button><button class="seg-btn" aria-pressed="true">On</button></div>
    <button data-hover-case="choice card" class="choice-card p-2">Card</button>
    <button data-hover-case="cell link" class="cell-link">Link</button>
    <button data-hover-case="menu item" class="menu-item">Menu item</button>
    <a data-hover-case="tab" class="tab h-9" href="#tab">Tab</a>
    <Select data-hover-case="select" class="w-30" aria-label="Select"><option>Select</option></Select>
  </div>
</Story>

<Story name="Sizes" asChild>
  <div class="flex items-center gap-2">
    <Button size="2xs">2xs 24</Button>
    <Button size="xs">xs 26</Button>
    <Button size="sm">sm 30</Button>
    <Button>default 32</Button>
    <Button size="md">md 34</Button>
    <Button size="lg">lg 36</Button>
    <Button icon size="md" aria-label="Collapse the sidebar">‹</Button>
  </div>
</Story>

<Story name="Toolbar" asChild>
  <div class="flex flex-wrap items-center gap-2">
    <Button size="lg" variant="primary">Equip</Button>
    <Button size="lg">Compare</Button>
    <Button size="lg" variant="ghost" class="px-3">Cancel</Button>
    <Button size="lg" disabled>Disabled</Button>
    <Button size="lg" icon aria-label="Settings" title="Settings"><Icon name="gear" /></Button>
    <Button size="lg" icon aria-label="About go-pob" title="About"><Icon name="info" /></Button>
  </div>
</Story>
