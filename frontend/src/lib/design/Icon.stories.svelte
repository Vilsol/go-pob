<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Icon from './Icon.svelte';

  const { Story } = defineMeta({
    title: 'Spec/Icon',
    component: Icon,
    argTypes: { name: { control: 'inline-radio', options: ['gear', 'info'] } },
    args: { name: 'gear' }
  });
</script>

<Story name="Gear" />

<Story name="Info" args={{ name: 'info' }} />
