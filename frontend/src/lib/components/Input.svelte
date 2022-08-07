<script lang="ts">
  export let prefix: string | undefined;
  export let value: number;
  export let fullWidth = false;
  export let max: number | undefined;
  export let placeholder: string | undefined;
  export let disabled = false;

  let inputElement: HTMLInputElement;
</script>

<div
  class="input-wrapper flex flex-row items-center" class:disabled={disabled}
  class:min-w-full={fullWidth}
  on:click={() => inputElement.focus()}>
  {#if prefix}
    <span class="mx-1 select-none">{prefix}</span>
  {/if}
  <input bind:this={inputElement} type="text" bind:value class="input flex-1" maxlength={max} placeholder={placeholder} disabled={disabled} />
</div>
