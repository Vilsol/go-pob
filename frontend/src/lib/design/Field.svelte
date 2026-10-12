<script lang="ts">
  import type { Snippet } from 'svelte';

  type Props = { label: string; inline?: boolean; class?: string; children: Snippet };
  let { label, inline = false, class: extra = '', children }: Props = $props();
</script>

<label class={['field', inline && 'field-inline', extra]}>{label}{@render children()}</label>
