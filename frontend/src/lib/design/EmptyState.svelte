<script lang="ts">
  type Props = { text: string; inset?: 'none' | 'row' | 'page'; busy?: boolean; class?: string };
  let { text, inset = 'none', busy = false, class: extra = '' }: Props = $props();
  const insets = { none: '', row: 'px-4 py-3', page: 'p-6 text-center text-13' };
</script>

<p role={busy ? 'status' : undefined} class={['note m-0', insets[inset], extra]}>{text}</p>
