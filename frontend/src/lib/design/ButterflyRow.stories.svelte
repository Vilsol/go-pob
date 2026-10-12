<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect, userEvent, within } from 'storybook/test';
  import { expectFloating } from './stories/layout_checks';
  import ButterflyRow from './ButterflyRow.svelte';
  import Ruler from './Ruler.svelte';

  const { Story } = defineMeta({
    title: 'Spec/ButterflyRow',
    component: ButterflyRow,
    args: { label: 'Fire Resistance', a: '80%', b: '76%', lead: 'A +4 pp', winner: 'a', pA: 100, pB: 95, caps: [100, 95] }
  });
</script>

<Story name="A wins (capped)" asChild>
  <div class="panel w-200 pb-1"><ButterflyRow label="Fire Resistance" a="80%" b="76%" lead="A +4 pp" winner="a" pA={100} pB={95} caps={[100, 95]} /></div>
</Story>

<Story name="Compare rows" asChild>
  <div class="panel w-200 pb-1">
    <Ruler title="DEFENCE" />
    <ButterflyRow label="Fire Resistance" a="80%" b="76%" lead="A +4 pp" winner="a" pA={100} pB={95} caps={[100, 95]} />
    <ButterflyRow label="Mana Cost (Blade Vortex)" a="42" b="31" lead="B −26.2%" winner="b" pA={100} pB={73.81} />
    <ButterflyRow label="Chaos Resistance" a="−12%" b="−12%" lead="=" pA={40} pB={40} zero={46.67} caps={[100, 100]} />
  </div>
</Story>

<Story
  name="Difference on hover"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    const c = within(canvasElement);
    await expect(c.queryByRole('tooltip')).toBeNull();
    const row = canvasElement.querySelector<HTMLElement>('.butterfly-row')!;
    await userEvent.hover(row);
    const tip = await c.findByRole('tooltip');
    await expect(tip).toHaveTextContent('B − A: +1,250 (+25.0%)');
    await expect(tip.querySelector('[data-better]')).toHaveAttribute('data-better', 'true');
    await expectFloating(row, tip);
    await userEvent.unhover(row);
    await expect(c.queryByRole('tooltip')).toBeNull();
  }}>
  <div class="panel w-200 pb-1">
    <ButterflyRow label="Total Life" a="5,000" b="6,250" lead="B +25.0%" winner="b" pA={80} pB={100} diff="+1,250 (+25.0%)" diffBetter />
  </div>
</Story>
