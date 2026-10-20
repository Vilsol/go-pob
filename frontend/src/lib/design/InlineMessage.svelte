<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { ClassValue } from 'svelte/elements';

  type Props = {
    kind?: 'info' | 'success' | 'warn' | 'error';
    size?: 'sm' | 'md';
    tag?: 'p' | 'div' | 'span';
    id?: string;
    class?: ClassValue;
    children: Snippet;
  };
  let { kind = 'info', size = 'sm', tag = 'div', id, class: extra = '', children }: Props = $props();
  // error is red; worse/better stay stat directions, so success and warn borrow them only as colours
  const kinds = { info: 'text-text-2', success: 'text-better', warn: 'text-worse', error: 'text-error' };
  const sizes = { sm: 'text-12', md: 'text-13' };
</script>

<svelte:element this={tag} {id} role={kind === 'error' ? 'alert' : 'status'} class={['m-0', sizes[size], kinds[kind], extra]}
  >{@render children()}</svelte:element>
