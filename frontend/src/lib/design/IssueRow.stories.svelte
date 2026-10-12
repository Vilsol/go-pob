<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import IssueRow from './IssueRow.svelte';

  const { Story } = defineMeta({
    title: 'Spec/IssueRow',
    component: IssueRow,
    args: { text: 'Chaos resistance is negative (−12%)', kind: 'Resistance', where: 'Resistances · sidebar' }
  });
</script>

<Story name="Resistance" />

<Story name="Parse" args={{ text: 'Custom modifier line 4 did not parse: “Crit Immune”', kind: 'Parse', where: 'Config · custom modifiers' }} />
