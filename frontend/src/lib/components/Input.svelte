<script lang="ts">
  export let prefix: string | undefined = undefined;
  export let value = '';
  export let fullWidth = false;
  export let max: number | undefined = undefined;
  export let placeholder: string | undefined = undefined;
  export let disabled = false;

  let inputElement: HTMLInputElement;
</script>

<div
  class="input-wrapper flex flex-row items-center"
  class:disabled
  class:min-w-full={fullWidth}
  on:click={() => inputElement.focus()}>
  {#if prefix}
    <span class="mx-1 select-none">{prefix}</span>
  {/if}
  <input
    bind:this={inputElement}
    type="text"
    bind:value
    class="input flex-1"
    maxlength={max}
    {placeholder}
    {disabled} />
</div>
