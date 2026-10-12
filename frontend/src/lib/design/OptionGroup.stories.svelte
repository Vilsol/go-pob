<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import OptionGroup from './OptionGroup.svelte';
  import OptionRow from './OptionRow.svelte';
  import Select from './Select.svelte';
  import NumberInput from './NumberInput.svelte';

  const { Story } = defineMeta({ title: 'Spec/OptionGroup', component: OptionGroup });
</script>

<Story name="Enemy Stats" asChild>
  <OptionGroup title="Enemy Stats" count="29 options · 1 modified" more="+ 19 more (hidden: only shown when the build uses them)" class="mb-0 w-110">
    <OptionRow label="Is the enemy a Boss?" modified>
      {#snippet control()}<Select size="xs" class="max-w-37.5" options={['Uber Pinnacle Boss', 'Pinnacle Boss', 'Boss', 'None']} />{/snippet}
    </OptionRow>
    <OptionRow>
      {#snippet label()}Enemy <span class="toned tone-lightning">Lightning</span> Resistance:{/snippet}
      {#snippet control()}<NumberInput size="xs" class="w-16 font-mono" placeholder="50" min={-200} max={100} />{/snippet}
    </OptionRow>
    <OptionRow label="Enemy Base Evasion:" inactive>
      {#snippet control()}<NumberInput size="xs" class="w-16 font-mono" placeholder="0" min={0} />{/snippet}
    </OptionRow>
  </OptionGroup>
</Story>
