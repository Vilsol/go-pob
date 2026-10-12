<script lang="ts">
  let { title }: { title: string } = $props();
</script>

<div role="separator" aria-label={title} class="ruler">{title}</div>
