<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import OptionRow from './OptionRow.svelte';
  import NumberInput from './NumberInput.svelte';

  const { Story } = defineMeta({ title: 'Spec/OptionRow', component: OptionRow });
</script>

<Story name="States" asChild>
  <div class="flex w-110 flex-col rounded-md border border-line bg-surface-1">
    <OptionRow label="# of Power Charges (if not maximum):" modified>
      {#snippet control()}<NumberInput size="xs" class="w-16 font-mono" value={3} min={0} />{/snippet}
    </OptionRow>
    <OptionRow label="Enemy Level:">
      {#snippet control()}<NumberInput size="xs" class="w-16 font-mono" placeholder="84" min={1} max={100} />{/snippet}
    </OptionRow>
    <OptionRow label="Do you use Ghost Shrouds?" inactive>
      {#snippet control()}<input class="check size-3.75" type="checkbox" />{/snippet}
    </OptionRow>
  </div>
</Story>
