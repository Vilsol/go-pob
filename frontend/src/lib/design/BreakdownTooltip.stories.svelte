<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import BreakdownTooltip from './BreakdownTooltip.svelte';

  const { Story } = defineMeta({
    title: 'Spec/BreakdownTooltip',
    component: BreakdownTooltip,
    args: {
      title: 'Total Life',
      value: '5,120',
      tone: 'life',
      formula: '(1,108 base + 1,342 flat) × (1 + 109% inc)',
      groups: [
        { title: 'ITEMS', rows: [{ src: 'Stygian Coil', val: '+98', tone: 'rare' }] },
        { title: 'PASSIVES', rows: [{ src: 'Heart and Soul', val: '+8%', tone: 'text' }] }
      ],
      footer: 'Click for the full breakdown on the Calcs page'
    }
  });
</script>

<Story name="Sidebar stat" />

<Story
  name="Resistance"
  args={{
    title: 'Fire Resistance',
    value: '80%',
    tone: 'fire',
    formula: '117% total, capped at 80% max (+37% over)',
    groups: [
      {
        title: 'ITEMS',
        rows: [
          { src: 'Morbid Bulwark', val: '+31%', tone: 'rare' },
          { src: 'Death Loop', val: '+14%', tone: 'rare' }
        ]
      },
      { title: 'PASSIVES', rows: [{ src: 'Practical Application', val: '+10%', tone: 'text' }] },
      { title: 'BUFFS', rows: [{ src: 'Purity of Elements (party)', val: '+43%', tone: 'text' }] }
    ]
  }} />

<Story
  name="Compact"
  args={{
    compact: true,
    title: 'Crit Chance',
    value: '31.00%',
    tone: undefined,
    formula: '6.00% base × (1 + 417%) = 31.02%',
    groups: [
      { title: 'ITEMS', rows: [{ src: 'Storm Grip', val: '+28%', tone: 'rare' }] },
      { title: 'PASSIVES', rows: [{ src: 'Annihilation', val: '+50%', tone: 'text' }] },
      { title: 'GEMS', rows: [{ src: 'Blade Vortex', val: '6.00%', tone: 'gem' }] }
    ],
    footer: undefined
  }} />
