<script module lang="ts">
  import { defineMeta } from '@storybook/addon-svelte-csf';
  import { expect } from 'storybook/test';
  import Socket from './Socket.svelte';
  import { SOCKET_STYLES } from '$lib/display/appearance';

  const COLOURS = [
    ['R', 'str'],
    ['G', 'dex'],
    ['B', 'int'],
    ['W', 'white'],
    ['A', undefined]
  ] as const;

  const { Story } = defineMeta({ title: 'Spec/Socket', component: Socket, args: { socket: 'B', gem: 'int' } });
</script>

<Story name="Matching gem" />

<Story name="States" asChild>
  <div class="flex items-center gap-3">
    <Socket socket="R" gem="str" />
    <Socket socket="G" gem="dex" />
    <Socket socket="B" gem="int" />
    <Socket socket="W" gem="white" />
    <Socket socket="A" />
    <Socket socket="warn" gem="dex" />
    <Socket socket="B" gem="flask" />
    <Socket socket="B" />
  </div>
</Story>

<Story
  name="Styles"
  asChild
  play={async ({ canvasElement }) => {
    const shapes = [...canvasElement.querySelectorAll('svg[data-ornament]')].map((s) => s.getAttribute('data-ornament'));
    await expect(shapes).toEqual([...Array(10).fill('small'), 'R', 'G', 'B', 'W', 'A', 'R', 'G', 'B', 'W', 'A']);
    await expect(canvasElement.querySelectorAll('[role="img"][aria-label$=" socket"]')).toHaveLength(30);
  }}>
  <div class="grid grid-cols-[auto_repeat(10,2.5rem)] items-center gap-x-2 gap-y-4 p-4 text-12">
    {#each SOCKET_STYLES as style (style)}
      <span>{style}</span>
      {#each COLOURS as [s, g] (s)}<span class="flex justify-center"><Socket socket={s} gem={g} ornament={style} letters={false} /></span>{/each}
      {#each COLOURS as [s] (s)}<span class="flex justify-center"><Socket socket={s} ornament={style} letters={false} /></span>{/each}
    {/each}
  </div>
</Story>

<Story
  name="Letters"
  asChild
  play={async ({ canvasElement }) => {
    const letters = [...canvasElement.querySelectorAll('.socket-letter')];
    await expect(letters.map((l) => l.textContent)).toEqual(['R', 'G', 'B', 'W', 'A', 'R', 'G', 'B', 'W', 'A']);
    await expect(letters.every((l) => l.getAttribute('aria-hidden') === 'true')).toBe(true);
  }}>
  <div class="flex items-center gap-3 p-4">
    {#each COLOURS as [s, g] (s)}<Socket socket={s} gem={g} letters ornament="None" />{/each}
    {#each COLOURS as [s] (s)}<Socket socket={s} letters ornament="Ornate" />{/each}
  </div>
</Story>

<Story
  name="Abyssal"
  asChild
  play={async ({ canvasElement }) => {
    const sockets = [...canvasElement.querySelectorAll('.socket')];
    await expect(sockets.map((s) => s.getAttribute('aria-label'))).toEqual(Array(3).fill('Abyssal socket'));
    await expect(sockets.every((s) => s.classList.contains('tone-sock-a') && !s.querySelector('.socket-gem'))).toBe(true);
    await expect([...canvasElement.querySelectorAll('svg[data-ornament]')].map((s) => s.getAttribute('data-ornament'))).toEqual(['small', 'A']);
    await expect([...canvasElement.querySelectorAll('.socket-letter')].map((l) => l.textContent)).toEqual(['A', 'A', 'A']);
  }}>
  <div class="flex items-center gap-6 p-4">
    {#each SOCKET_STYLES as style (style)}<Socket socket="A" ornament={style} letters />{/each}
  </div>
</Story>
