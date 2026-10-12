<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Ruler from './Ruler.svelte';

  const { Story } = defineMeta({ title: 'Spec/Ruler', component: Ruler, args: { title: 'DEFENCE' } });
</script>

<Story name="Defence" />

<Story name="Offence" args={{ title: 'OFFENCE' }} />
