<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import Toasts from './Toasts.svelte';
  import { addToast, clearToasts } from './toast';

  const { Story } = defineMeta({ title: 'Spec/Toasts', component: Toasts });
</script>

<script lang="ts">
  clearToasts();
  addToast('No update available\nYou are running the latest version.');
  addToast('Power builder\nCalculating node power: 40%');
</script>

<Story name="Two toasts" asChild>
  <div class="relative h-60 w-100">
    <Toasts />
  </div>
</Story>
