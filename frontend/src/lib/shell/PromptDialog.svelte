<script lang="ts">
  import Dialog from '$lib/design/Dialog.svelte';
  import Field from '$lib/design/Field.svelte';
  import Input from '$lib/design/Input.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';

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
    <DialogActions {confirm} disabled={!text} oncancel={() => (open = false)} onconfirm={submit} />
  {/snippet}
</Dialog>
