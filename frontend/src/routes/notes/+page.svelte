<script lang="ts">
  import Tiptap from '$lib/components/Tiptap.svelte';

  let notesDesc = `You can use Ctrl +/- (or Ctrl+Scroll) to zoom in and out and Ctrl+0 to reset.
This field also supports different colors. Using the caret symbol (^) followed by a Hex code or a number (0-9) will set the color.
Below are some common color codes PoB uses:`

</script>

<div class="notes-tab w-full h-full">
  <div class="notes-label whitespace-pre-line">
    <p>{notesDesc}</p>
  </div>

  <div class="editor h-full">
    <!-- TODO: Implement copy/paste functionality with support for color codes. -->
    <Tiptap />
  </div>
  
</div>

<style>
  .notes-tab {
    display: flex;
    flex-direction: column;
    gap: 10px;
    background: black;
    color: white;
    padding: 10px;
  }
</style>

