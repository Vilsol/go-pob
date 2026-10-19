<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import CalcsBreakdown from './CalcsBreakdown.svelte';
  import { expect } from 'storybook/test';

  const { Story } = defineMeta({ title: 'Calcs/CalcsBreakdown', component: CalcsBreakdown });
  const mod = (Value: string, SourceName: string, Flags = '', Source = 'Tree', Name = '') => ({ Value, Name, Flags, Tags: '', Source, SourceName });
</script>

<Story
  name="A pinned row"
  asChild
  play={async ({ canvasElement }: { canvasElement: HTMLElement }) => {
    await expect(canvasElement.querySelectorAll('[data-step]').length).toBe(9);
    await expect(canvasElement.querySelector('[role="separator"]')).not.toBeNull();
  }}>
  <div class="flex h-[720px] justify-end bg-ground">
    <CalcsBreakdown
      id="bd-story"
      title="Skill Hit Damage · Total Increased"
      pinned
      onunpin={() => {}}
      parts={[
        {
          head: 'Lightning:',
          text: '572 to 2,839',
          bd: {
            lines: [
              'Base damage:',
              '26 to 129 (added damage)',
              'x 2.70 (damage effectiveness)',
              '= 70.2 to 348.3',
              'Hit damage:',
              '954 to 4731 (total damage)',
              'x 0.600 (effective DPS modifier)',
              '= 572.40 to 2838.60'
            ]
          }
        },
        {
          head: 'All Types:',
          text: '343%',
          modTables: [
            {
              rows: [
                mod('100', 'Fire Mastery', 'Hit'),
                mod('60', 'Discord Artisan'),
                mod('50', 'Summon Flame Golem', '', 'Skill'),
                mod('37', 'Chimeric Gnarl, Tornado Wand', 'Spell', 'Item'),
                mod('20', "Arcanist's Dominion", 'Spell')
              ]
            },
            {
              label: 'Elemental',
              rows: [mod('24', 'Holy Dominion', '', 'Tree', 'Elemental Damage'), mod('10', 'Elemental Damage', '', 'Tree', 'Elemental Damage')]
            }
          ]
        },
        { head: 'Radius', text: '47', bd: { lines: ['3.8m (base radius)', 'x 1.24 (square root of area of effect modifier)', '= 4.7m'], radius: 47 } }
      ]} />
  </div>
</Story>
