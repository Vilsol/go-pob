<script lang="ts">
  import Dialog from './Dialog.svelte';
  import Tabs from './Tabs.svelte';
  import type { AboutInfo } from './fixtures';
  import type { ChangeVersion, HelpSection } from './help';
  import { tick } from 'svelte';

  // Main.lua:1359-1447
  // Main.lua:1338-1358, 1434-1437
  type Props = {
    open?: boolean;
    info: AboutInfo;
    tab?: string;
    oncopy?: (text: string) => void;
    help?: HelpSection[];
    helpSection?: string;
    changelog?: ChangeVersion[];
  };
  let { open = $bindable(false), info, tab = $bindable('about'), oncopy, help, helpSection, changelog }: Props = $props();

  let helpBox = $state<HTMLElement>();
  let current = $state<string>();
  const headings = () => [...(helpBox?.querySelectorAll<HTMLElement>('[data-help-section]') ?? [])];
  const jump = async (title: string) => {
    await tick();
    const h = headings().find((e) => e.dataset.helpSection === title);
    if (helpBox && h) helpBox.scrollTop = h.offsetTop;
    current = title;
  };
  const scrolled = () => {
    const top = (helpBox?.scrollTop ?? 0) + 8;
    current = headings().findLast((h) => h.offsetTop <= top)?.dataset.helpSection ?? current;
  };
  $effect(() => {
    if (open && tab === 'help' && helpSection && help?.length) void jump(helpSection);
  });

  let copied = $state(false);
  const facts = $derived([
    ['VERSION', info.version],
    ['GAME DATA', info.gameData],
    ['PASSIVE TREE', info.tree],
    ['ENGINE', `${info.go} · wasm`]
  ]);
  const tables: Record<string, { title: string; note: string; k: string; v: string; rows: [string, string][] }> = $derived({
    data: {
      title: 'Game data',
      note: 'Where the numbers come from',
      k: 'ITEM',
      v: 'VERSION',
      rows: [
        ['Game data', info.gameData],
        ['Passive tree', info.tree]
      ]
    },
    build: { title: 'Build settings', note: 'How the WebAssembly binary was built', k: 'KEY', v: 'VALUE', rows: info.settings },
    deps: { title: 'Dependencies', note: 'Modules compiled into the WebAssembly binary', k: 'MODULE', v: 'VERSION', rows: info.deps }
  });
  const text = $derived(
    [
      `go-pob ${info.version}`,
      ...facts.map(([k, v]) => `${k}: ${v}`),
      '',
      ...info.settings.map(([k, v]) => `${k}=${v}`),
      '',
      ...info.deps.map(([k, v]) => `${k} ${v}`)
    ].join('\n')
  );
</script>

<Dialog bind:open title="About go-pob" sheet backdrop class="h-160 max-h-[calc(100vh-24px)] w-205" onclose={() => (copied = false)}>
  {#snippet head()}
    <span class="font-mono text-12 text-text-2">{info.version}</span>
    <button
      class="btn btn-sm ml-auto"
      title="Copies version, build settings and dependencies as text for a bug report"
      onclick={() => {
        oncopy?.(text);
        copied = true;
      }}>{copied ? 'Copied ✓' : 'Copy diagnostics'}</button>
  {/snippet}
  <Tabs
    label="About sections"
    bind:selected={tab}
    class="border-b border-line px-3.5"
    panelClass="flex min-h-0 flex-1 flex-col"
    tabs={[
      { id: 'about', label: 'About' },
      { id: 'data', label: 'Game data' },
      { id: 'build', label: 'Build settings' },
      { id: 'deps', label: 'Dependencies' },
      ...(changelog ? [{ id: 'history', label: 'Version history' }] : []),
      ...(help ? [{ id: 'help', label: 'Help' }] : [])
    ]}>
    {#snippet panel(id: string)}
      {#if id === 'help'}
        <div class="flex min-h-0 flex-1">
          <nav aria-label="Help sections" class="scroll flex w-52 shrink-0 flex-col gap-0.5 border-r border-line bg-inset px-2 py-3">
            {#each help ?? [] as s, i (i)}
              <button
                class={['menu-item shrink-0', s.title === current && 'bg-hover font-semibold']}
                aria-current={s.title === current ? 'true' : undefined}
                onclick={() => void jump(s.title)}>{i + 1}. {s.title}</button>
            {/each}
          </nav>
          <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
          <section bind:this={helpBox} aria-label="Help text" tabindex="0" class="scroll relative min-h-0 flex-1 px-6 pt-1 pb-6" onscroll={scrolled}>
            {#each help ?? [] as s, i (i)}
              <h3 data-help-section={s.title} class="mt-4 mb-2 border-b border-line pt-3 pb-1.5 text-16 font-semibold">{s.title} ({i + 1})</h3>
              <div class="flex max-w-[72ch] flex-col text-14 leading-[1.6]">
                {#each s.lines as l, j (j)}
                  {#if l.key !== undefined}
                    <div class={['grid grid-cols-[12rem_1fr] gap-3 py-0.5', l.dev && 'text-accent']}>
                      <span class="font-mono text-12 text-text-2">{l.key}</span><span>{l.text}</span>
                    </div>
                  {:else if l.text.trim() === ''}
                    <div class="h-3"></div>
                  {:else}
                    <p class={['m-0 whitespace-pre-wrap', l.dev && 'text-accent']}>{l.text}</p>
                  {/if}
                {/each}
              </div>
            {/each}
          </section>
        </div>
      {:else}
        <div class="scroll flex min-h-0 flex-1 flex-col gap-3 px-4.5 py-3.5">
          {#if id === 'about'}
            <p class="m-0 max-w-160 text-14 leading-[1.55]">
              A web port of Path of Building: the calculation engine is Go compiled to WebAssembly, the interface is SvelteKit, and game data is downloaded from
              go-pob-data. Everything runs in your browser; builds are stored locally.
            </p>
            <div class="grid grid-cols-2 gap-2.5">
              {#each facts as [k, v] (k)}
                <div class="flex flex-col gap-0.75 rounded-md border border-line bg-inset px-3 py-2.5">
                  <span class="text-11 tracking-caps-tight text-text-3">{k}</span><span class="font-mono text-14">{v}</span>
                </div>
              {/each}
            </div>
            <div class="flex flex-wrap gap-2.5 text-13">
              <a href="https://github.com/Vilsol/go-pob">Source on GitHub</a>
              <a href="https://github.com/Vilsol/go-pob/issues">Report a problem</a>
              <a href="https://github.com/Vilsol/go-pob-data">go-pob-data</a>
              <a href="https://github.com/PathOfBuildingCommunity/PathOfBuilding">Path of Building Community</a>
            </div>
            <p class="note m-0 max-w-160">Not affiliated with or endorsed by Grinding Gear Games.</p>
          {:else if id === 'history'}
            <div class="flex flex-col gap-0.5 text-13">
              {#each changelog ?? [] as v, i (i)}
                <h3 class="mt-3 mb-1 text-15 font-semibold first:mt-0">Version {v.version} ({v.date})</h3>
                {#each v.lines as l, j (j)}
                  <p class={['m-0 min-h-3 whitespace-pre-wrap', l.sub && 'mt-1 font-semibold']}>{l.text}</p>
                {/each}
              {/each}
            </div>
          {:else}
            {@const t = tables[id]}
            <span class="note">{t.note}</span>
            <div role="table" aria-label={t.title} class="overflow-hidden rounded-md border border-line">
              <div role="row" class="data-row data-head grid-cols-2 gap-3 bg-inset py-1.5 tracking-caps-tight">
                <span role="columnheader">{t.k}</span><span role="columnheader">{t.v}</span>
              </div>
              {#each t.rows as [k, v] (k)}
                <div role="row" class="data-row grid-cols-2 gap-3 py-1.25 font-mono text-12 hover:bg-bar">
                  <span role="cell" class="[overflow-wrap:anywhere] text-text-2">{k}</span><span role="cell" class="[overflow-wrap:anywhere]">{v || '—'}</span>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/if}
    {/snippet}
  </Tabs>
</Dialog>
