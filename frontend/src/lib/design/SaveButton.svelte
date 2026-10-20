<script lang="ts">
  import Button from './Button.svelte';
  import Icon from './Icon.svelte';

  // iconClass / textClass swap the label for the icon, e.g. at a breakpoint
  let {
    dirty,
    onsave,
    class: extra = '',
    iconClass = '',
    textClass = ''
  }: { dirty: boolean; onsave?: () => void; class?: string; iconClass?: string; textClass?: string } = $props();
</script>

<Button
  variant={dirty ? 'primary' : 'default'}
  class={extra}
  aria-label={dirty ? 'Save (unsaved changes)' : 'Save (no unsaved changes)'}
  title={dirty ? 'Unsaved changes — Save (Ctrl+S)' : 'All changes saved (Ctrl+S)'}
  onclick={onsave}><span class={['hidden', iconClass]}><Icon name="save" /></span><span class={textClass}>Save</span></Button>
