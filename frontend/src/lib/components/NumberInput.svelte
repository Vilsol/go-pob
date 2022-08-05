<script lang="ts">
  export let prefix: string | undefined;
  export let min: number | undefined;
  export let max: number | undefined;
  export let value: number;

  let inputElement: HTMLInputElement;
</script>

<div class="flex flex-row">
  <div class="input-wrapper flex flex-row items-center" on:click={() => inputElement.focus()}>
    {#if prefix}
      <span class="mx-1 select-none">{prefix}</span>
    {/if}
    <input bind:this={inputElement} type="number" {min} {max} bind:value class="input" />
  </div>
  <button class="container font-bold" on:click={() => (value = value + 1 || min || 0)} disabled={value >= max}>
    &plus;
  </button>
  <button class="container font-bold" on:click={() => (value = value - 1 || max || 0)} disabled={value <= min}>
    &minus;
  </button>
</div>
