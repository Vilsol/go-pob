<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Icon, { type IconName } from './Icon.svelte';

  const ALL: IconName[] = [
    'gear',
    'info',
    'save',
    'import-export',
    'grip',
    'more',
    'folder',
    'folder-open',
    'chevron-up',
    'chevron-down',
    'chevron-right',
    'chevron-left',
    'close',
    'copy',
    'check',
    'warn',
    'star',
    'refresh',
    'undo',
    'redo'
  ];

  const { Story } = defineMeta({
    title: 'Spec/Icon',
    component: Icon,
    argTypes: { name: { control: 'inline-radio', options: ['gear', 'info'] } },
    args: { name: 'gear' }
  });
</script>

<Story name="Gear" />

<Story name="Info" args={{ name: 'info' }} />

<Story name="All" asChild>
  <div class="flex flex-wrap items-center gap-3 text-text-2">
    {#each ALL as n (n)}<span class="flex flex-col items-center gap-1 text-11" title={n}><Icon name={n} />{n}</span>{/each}
    <span class="flex items-center gap-1 text-11"><Icon name="close" size={12} />12</span>
    <span class="flex items-center gap-1 text-11"><Icon name="close" size={20} />20</span>
  </div>
</Story>
