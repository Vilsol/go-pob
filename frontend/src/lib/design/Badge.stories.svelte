<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Badge from './Badge.svelte';

  const { Story } = defineMeta({ title: 'Spec/Badge', component: Badge });
</script>

<Story name="Variants" asChild>
  <div class="flex flex-col gap-2">
    <div class="flex items-center gap-2">
      <Badge>soft</Badge><Badge tone="accent">current</Badge><Badge tone="accent">★ MAIN</Badge><Badge tone="warn">⚠ 2 warnings</Badge><Badge tone="up"
        >better</Badge>
    </div>
    <div class="flex items-center gap-2">
      <Badge variant="solid" size="sm" mono>3</Badge><Badge variant="solid" size="sm" mono tone="warn">12</Badge><Badge variant="solid" shape="tag">tag</Badge>
    </div>
    <div class="flex items-center gap-2"><Badge variant="outline">Notable</Badge><Badge variant="outline">Keystone</Badge></div>
  </div>
</Story>
