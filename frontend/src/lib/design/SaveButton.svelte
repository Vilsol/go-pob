<script lang="ts">
  import Button from './Button.svelte';
  import Icon from './Icon.svelte';

  let { dirty, onsave, class: extra = '' }: { dirty: boolean; onsave?: () => void; class?: string } = $props();
</script>

<Button
  variant={dirty ? 'primary' : 'default'}
  class={extra}
  aria-label={dirty ? 'Save (unsaved changes)' : 'Save (no unsaved changes)'}
  title={dirty ? 'Unsaved changes — Save (Ctrl+S)' : 'All changes saved (Ctrl+S)'}
  onclick={onsave}><span class="save-icon hidden"><Icon name="save" /></span><span class="save-text">Save</span></Button>
