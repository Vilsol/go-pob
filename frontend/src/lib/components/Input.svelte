<script lang="ts">
  let {
    prefix = undefined,
    value = $bindable(''),
    fullWidth = false,
    max = undefined,
    placeholder = undefined,
    disabled = false,
    classes = ''
  }: {
    prefix?: string | undefined;
    value?: string;
    fullWidth?: boolean;
    max?: number | undefined;
    placeholder?: string | undefined;
    disabled?: boolean;
    classes?: string | undefined;
  } = $props();

  let inputElement = $state<HTMLInputElement>();
</script>

<button class="input-wrapper flex flex-row items-center w-full {classes}" class:disabled class:min-w-full={fullWidth} onclick={() => inputElement?.focus()}>
  {#if prefix}
    <span class="mx-1 select-none">{prefix}</span>
  {/if}
  <input bind:this={inputElement} type="text" bind:value class="input flex-1" maxlength={max} {placeholder} {disabled} />
</button>
