<script lang="ts">
  export let prefix: string | undefined = undefined;
  export let min: number | undefined = undefined;
  export let max: number | undefined = undefined;
  export let value: number | undefined = undefined;
  export let placeholder = '';

  let inputElement: HTMLInputElement;
</script>

<div class="flex flex-row">
  <div class="input-wrapper flex flex-row items-center" on:click={() => inputElement.focus()}>
    {#if prefix}
      <span class="mx-1 select-none">{prefix}</span>
    {/if}
    <input bind:this={inputElement} type="number" {min} {max} bind:value class="input" {placeholder} />
  </div>
  <button class="container font-bold" on:click={() => (value = value + 1 || min || 0)} disabled={value >= max}>
    &plus;
  </button>
  <button class="container font-bold" on:click={() => (value = value - 1 || max || 0)} disabled={value <= min}>
    &minus;
  </button>
</div>
