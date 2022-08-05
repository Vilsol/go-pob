<script lang="ts">
  export let prefix: string | undefined;
  export let value: number;
  export let fullWidth = false;

  let inputElement: HTMLInputElement;
</script>

<div
  class="input-wrapper flex flex-row items-center"
  class:min-w-full={fullWidth}
  on:click={() => inputElement.focus()}>
  {#if prefix}
    <span class="mx-1 select-none">{prefix}</span>
  {/if}
  <input bind:this={inputElement} type="text" bind:value class="input flex-1" />
</div>
