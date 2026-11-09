<script lang="ts">
  // a ring that turns; with motion reduced it stands still
  let { class: extra = '' }: { class?: string } = $props();
</script>

<span class={['inline-block size-3.5 shrink-0 rounded-full border-2 border-line-strong border-t-accent motion-safe:animate-spin', extra]} aria-hidden="true"
></span>
