<script lang="ts">
  import Dialog from '$lib/design/Dialog.svelte';
  import Field from '$lib/design/Field.svelte';
  import Input from '$lib/design/Input.svelte';
  import Button from '$lib/design/Button.svelte';

  type Props = { open?: boolean; title: string; label: string; value?: string; confirm: string; onconfirm: (value: string) => void };
  let { open = $bindable(false), title, label, value = '', confirm, onconfirm }: Props = $props();

  let text = $state('');
  $effect.pre(() => {
    if (open) text = value;
  });

  const submit = () => {
    if (!text) return;
    open = false;
    onconfirm(text);
  };
</script>

<Dialog bind:open {title}>
  <Field {label}>
    <Input
      bind:value={text}
      onkeydown={(e: KeyboardEvent) => {
        if (e.key === 'Enter') {
          e.preventDefault();
          submit();
        }
      }} />
  </Field>
  {#snippet actions()}
    <Button size="md" variant="ghost" class="px-3" onclick={() => (open = false)}>Cancel</Button>
    <Button size="md" variant="primary" class="px-4" disabled={!text} onclick={submit}>{confirm}</Button>
  {/snippet}
</Dialog>
