<script lang="ts">
  import type { Snippet } from 'svelte';
  import Button from './Button.svelte';

  // Every dialog footer: secondary actions, then Cancel, then the confirming action rightmost.
  type Props = {
    confirm?: string;
    tone?: 'primary' | 'danger' | 'default';
    disabled?: boolean;
    title?: string;
    onconfirm?: () => void;
    cancel?: string | false;
    oncancel?: () => void;
    extra?: Snippet;
  };
  let { confirm, tone = 'primary', disabled, title, onconfirm, cancel = 'Cancel', oncancel, extra }: Props = $props();
  const variant = $derived(tone === 'danger' ? 'danger-solid' : tone);
</script>

{@render extra?.()}
{#if cancel !== false}<Button size="md" variant="ghost" class="px-3" onclick={oncancel}>{cancel}</Button>{/if}
{#if confirm !== undefined}<Button size="md" {variant} class="px-4" {disabled} {title} onclick={onconfirm}>{confirm}</Button>{/if}
