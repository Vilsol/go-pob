<script lang="ts">
  import Input from '$lib/components/Input.svelte';

  interface Props {
    title: string;
    subtitle: string;
    value?: string;
    confirm: string;
    onconfirm: (value: string) => void;
    onclose: () => void;
  }

  let { title, subtitle, value, confirm, onconfirm, onclose }: Props = $props();

  let currentValue = $state(value || '');

  const clickConfirm = () => {
    onconfirm(currentValue);
    onclose();
  };
</script>

<div class="flex flex-col gap-2">
  <fieldset class="border border-white bg-neutral-900 p-2 mt-4 min-w-[15vw]">
    <legend class="container">{title}</legend>
    <div class="flex flex-row justify-center">
      <p>{subtitle}</p>
    </div>

    <Input fullWidth={true} bind:value={currentValue} />
  </fieldset>

  <div class="flex flex-row items-center justify-center gap-2">
    <button class="container" onclick={clickConfirm} disabled={!currentValue}>{confirm}</button>
    <button class="container" onclick={onclose}>Close</button>
  </div>
</div>
