<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, fn, userEvent, within } from 'storybook/test';
  import NumberInput from './NumberInput.svelte';
  import Field from './Field.svelte';
  import OptionRow from './OptionRow.svelte';

  const { Story } = defineMeta({
    title: 'Spec/NumberInput',
    component: NumberInput,
    argTypes: { size: { control: 'inline-radio', options: ['xs', 'default', 'lg'] } },
    args: { value: 94, min: 1, max: 100, step: 1, label: 'Level', class: 'w-22 font-mono', onchange: fn() }
  });

  type Ctx = { canvasElement: HTMLElement; args: { onchange?: (value: number) => void } };
  const spin = (root: HTMLElement) => within(root).getByRole('spinbutton');
  const stepper = (root: HTMLElement, label: 'Increase' | 'Decrease') => root.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;

  const hoverRule = (needle: string) => {
    const walk = (rules: CSSRuleList, inside: boolean): boolean =>
      [...rules].some((r) => {
        const sel = r instanceof CSSStyleRule ? r.selectorText : '';
        const here = inside || sel.includes(needle);
        if (here && sel.includes(':hover')) return true;
        return 'cssRules' in r && walk((r as CSSGroupingRule).cssRules, here);
      });
    return [...document.styleSheets].some((s) => {
      try {
        return walk(s.cssRules, false);
      } catch {
        return false;
      }
    });
  };

  async function scrub(el: HTMLElement, dx: number, mods: { shiftKey?: boolean; altKey?: boolean } = {}) {
    const r = el.getBoundingClientRect();
    const x = r.left + r.width / 2;
    const y = r.top + r.height / 2;
    const at = (cx: number) => ({ clientX: cx, clientY: y, pointerId: 1, button: 0, bubbles: true, isPrimary: true, ...mods });
    el.dispatchEvent(new PointerEvent('pointerdown', at(x)));
    const n = Math.abs(dx) / 2;
    for (let i = 1; i <= n; i++) el.dispatchEvent(new PointerEvent('pointermove', at(x + Math.sign(dx) * i * 2)));
    el.dispatchEvent(new PointerEvent('pointerup', at(x + dx)));
    await new Promise((done) => requestAnimationFrame(done));
  }
</script>

<Story
  name="Default"
  play={async ({ canvasElement, args }: Ctx) => {
    const el = spin(canvasElement);
    await expect(el).toHaveAccessibleName('Level');
    await expect(el).toHaveAttribute('type', 'text');
    await expect(el).toHaveAttribute('inputmode', 'decimal');
    await expect(el).toHaveAttribute('aria-valuenow', '94');
    await expect(el).toHaveAttribute('aria-valuemin', '1');
    await expect(el).toHaveAttribute('aria-valuemax', '100');
    await expect(getComputedStyle(el).cursor).toBe('ew-resize');
    const box = el.parentElement!;
    const inner = box.clientHeight;
    const col = stepper(canvasElement, 'Increase').parentElement!;
    await expect(Math.abs(col.getBoundingClientRect().height - inner)).toBeLessThanOrEqual(1);
    await expect(box.getBoundingClientRect().height).toBe(32);
    await expect(hoverRule('.number-input-steppers')).toBe(true);

    el.focus();
    await userEvent.keyboard('{ArrowUp}');
    await expect(el).toHaveValue('95');
    await expect(args.onchange).toHaveBeenLastCalledWith(95);
    await userEvent.keyboard('{ArrowDown}{ArrowDown}');
    await expect(el).toHaveValue('93');
    await userEvent.keyboard('{Shift>}{ArrowDown}{/Shift}');
    await expect(el).toHaveValue('83');
    await userEvent.keyboard('{PageUp}');
    await expect(el).toHaveValue('93');
    await userEvent.keyboard('{PageDown}');
    await expect(el).toHaveValue('83');
    await userEvent.keyboard('{Home}');
    await expect(el).toHaveValue('1');
    await userEvent.keyboard('{End}');
    await expect(el).toHaveValue('100');
    await expect(el).toHaveAttribute('aria-valuenow', '100');

    await userEvent.clear(el);
    await userEvent.type(el, '42{Enter}');
    await expect(el).toHaveValue('42');
    await expect(args.onchange).toHaveBeenLastCalledWith(42);
    await userEvent.clear(el);
    await userEvent.type(el, '7{Escape}');
    await expect(el).toHaveValue('42');
    await expect(args.onchange).toHaveBeenLastCalledWith(42);
    await userEvent.clear(el);
    await userEvent.type(el, '60');
    el.blur();
    await expect(el).toHaveValue('60');
    await expect(el).toHaveAttribute('aria-valuenow', '60');

    await userEvent.click(stepper(canvasElement, 'Increase'));
    await expect(el).toHaveValue('61');
  }} />

<Story
  name="Drag to scrub"
  play={async ({ canvasElement, args }: Ctx) => {
    const el = spin(canvasElement);
    await expect(el).toHaveValue('94');
    await scrub(el, -20);
    await expect(el).toHaveValue('89');
    await expect(args.onchange).toHaveBeenCalledTimes(1);
    await expect(args.onchange).toHaveBeenLastCalledWith(89);
    await expect(el).not.toHaveFocus();
    await scrub(el, 8, { shiftKey: true });
    await expect(el).toHaveValue('100');
    await scrub(el, 0);
    await expect(el).toHaveFocus();
    await expect(el).toHaveValue('100');
  }} />

<Story
  name="Min / max clamp"
  args={{ value: 3, min: 0, max: 5, label: 'Power Charges', class: 'w-16 font-mono' }}
  play={async ({ canvasElement, args }: Ctx) => {
    const el = spin(canvasElement);
    el.focus();
    await userEvent.keyboard('{Shift>}{ArrowUp}{/Shift}');
    await expect(el).toHaveValue('5');
    await userEvent.keyboard('{ArrowUp}');
    await expect(el).toHaveValue('5');
    await userEvent.clear(el);
    await userEvent.type(el, '-4{Enter}');
    await expect(el).toHaveValue('0');
    await expect(args.onchange).toHaveBeenLastCalledWith(0);
    await scrub(el, -40);
    await expect(el).toHaveValue('0');
  }} />

<Story
  name="Fractional step"
  args={{ value: 1.5, min: 0, max: 10, step: 0.1, label: 'Multiplier', format: (v: number) => `${v.toFixed(1)}×` }}
  play={async ({ canvasElement, args }: Ctx) => {
    const el = spin(canvasElement);
    await expect(el).toHaveValue('1.5×');
    await expect(el).toHaveAttribute('aria-valuetext', '1.5×');
    el.focus();
    await userEvent.keyboard('{ArrowUp}');
    await expect(el).toHaveAttribute('aria-valuenow', '1.6');
    await userEvent.keyboard('{Alt>}{ArrowUp}{/Alt}');
    await expect(el).toHaveAttribute('aria-valuenow', '1.61');
    el.blur();
    await scrub(el, 12, { altKey: true });
    await expect(el).toHaveAttribute('aria-valuenow', '1.64');
    await expect(args.onchange).toHaveBeenLastCalledWith(1.64);
  }} />

<Story
  name="Disabled"
  args={{ disabled: true }}
  play={async ({ canvasElement, args }: Ctx) => {
    const el = spin(canvasElement);
    await expect(el).toBeDisabled();
    await expect(stepper(canvasElement, 'Increase')).toBeDisabled();
    await scrub(el, 40);
    await expect(el).toHaveValue('94');
    await expect(args.onchange).not.toHaveBeenCalled();
  }} />

<Story
  name="Pointer capture refused"
  play={async ({ canvasElement, args }: Ctx) => {
    // eslint-disable-next-line @typescript-eslint/unbound-method
    const orig = Element.prototype.setPointerCapture;
    Element.prototype.setPointerCapture = () => {
      throw new DOMException('Invalid pointer id', 'NotFoundError');
    };
    try {
      const input = spin(canvasElement);
      await userEvent.tripleClick(input);
      await expect(input).toHaveFocus();
      await userEvent.keyboard('12{Enter}');
      await expect(input).toHaveValue('12');
      await expect(args.onchange).toHaveBeenLastCalledWith(12);
    } finally {
      Element.prototype.setPointerCapture = orig;
    }
  }} />

<Story name="Sizes" asChild>
  <div class="flex items-center gap-2">
    <NumberInput size="xs" class="w-16 font-mono" value={50} label="xs" />
    <NumberInput class="w-16 font-mono" value={94} label="default" />
    <NumberInput size="lg" class="w-22 bg-surface-1 font-mono" value={94} label="lg" />
  </div>
</Story>

<Story
  name="In a Field"
  asChild
  play={async ({ canvasElement }: Ctx) => {
    const el = within(canvasElement).getByRole('spinbutton', { name: 'Level' });
    await expect(el.parentElement!.getBoundingClientRect().height).toBe(36);
  }}>
  <Field label="Level" class="w-40 gap-1.5 text-13"><NumberInput size="lg" class="w-22 bg-surface-1 font-mono" value={94} min={1} max={100} /></Field>
</Story>

<Story
  name="In an OptionRow"
  asChild
  play={async ({ canvasElement }: Ctx) => {
    const el = within(canvasElement).getByRole('spinbutton', { name: '# of Power Charges (if not maximum): (changed)' });
    await expect(el.parentElement!.getBoundingClientRect().height).toBe(26);
  }}>
  <div class="flex w-110 flex-col rounded-md border border-line bg-surface-1">
    <OptionRow label="# of Power Charges (if not maximum):" modified>
      {#snippet control()}<NumberInput size="xs" class="w-16 font-mono" value={3} min={0} />{/snippet}
    </OptionRow>
  </div>
</Story>
