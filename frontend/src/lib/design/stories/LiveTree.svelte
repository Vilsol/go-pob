<script lang="ts">
  import TreePage from '../TreePage.svelte';
  import TreeView from '../TreeView.svelte';
  import type { BuildFixture, TreeNodeFx } from '../fixtures';
  import type { CanvasApi } from '../tree';

  type Props = { fx: BuildFixture; q?: string; zoom?: number };
  let { fx, q = '', zoom = 1.5 }: Props = $props();

  let tree = $state(structuredClone(fx.tree));
  let s = $state({
    alloc: Object.fromEntries(fx.tree.nodes.filter((n) => n.a).map((n) => [n.id, true])) as Record<number, boolean>,
    q,
    mastery: null as number | null,
    log: '',
    className: fx.className,
    ascendancy: fx.ascendClassName,
    level: fx.level
  });
  let bw = $state(0);
  let bh = $state(0);
  const vw = $derived(bw * zoom);
  const vh = $derived(vw / tree.aspect);
  const ox = $derived((bw - vw) / 2);
  const oy = $derived((bh - vh) / 2);
  const byId = $derived(new Map(tree.nodes.map((n) => [n.id, n])));
  const project = (id: number) => {
    const n = byId.get(id);
    return n ? { x: ox + n.x * vw, y: oy + n.y * vh } : undefined;
  };

  const pick = (n: TreeNodeFx) => {
    if (s.alloc[n.id]) {
      for (const id of [n.id, ...(n.a ? (n.loseIds ?? []) : [])]) delete s.alloc[id];
      return;
    }
    for (let x: TreeNodeFx | undefined = n; x && !s.alloc[x.id]; x = x.next !== undefined ? byId.get(x.next) : undefined) s.alloc[x.id] = true;
  };
</script>

<TreePage
  {tree}
  className={s.className}
  ascendancy={s.ascendancy}
  level={s.level}
  progress={fx.progress}
  alloc={s.alloc}
  bind:q={s.q}
  bind:mastery={s.mastery}
  {project}
  onpick={pick}
  onmastery={(id: number, effect: number) => {
    const m = tree.nodes.find((n) => n.id === id);
    if (m) m.effect = effect;
  }}
  onmenuaction={(action: string, n: TreeNodeFx) => (s.log = `${action} ${n.name}`)}
  ontimeless={() => (s.log = 'timeless')}
  onpathmode={(m: string) => (s.log = `path ${m}`)}
  onclass={(c: string) => {
    s.className = c;
    s.log = `class ${c}`;
  }}
  onascendancy={(a: string) => {
    s.ascendancy = a;
    s.log = `ascendancy ${a}`;
  }}
  onlevel={(l: number) => {
    s.level = l;
    s.log = `level ${l}`;
  }}>
  {#snippet canvas(api: CanvasApi)}
    <div class="absolute inset-0" bind:clientWidth={bw} bind:clientHeight={bh} data-canvas-placeholder>
      <div class="absolute top-(--oy) left-(--ox) w-(--vw)" style:--ox="{ox}px" style:--oy="{oy}px" style:--vw="{vw}px">
        <TreeView
          {tree}
          label="Canvas placeholder"
          alloc={s.alloc}
          hovered={api.hovered}
          describedBy={api.describedBy}
          path={api.path}
          lose={api.lose}
          matches={api.matches}
          active={(n: TreeNodeFx) => n.k !== 'c'}
          onenter={api.enter}
          onleave={api.leave}
          onpick={api.pick}
          onmenu={api.menu} />
      </div>
      <span
        class="pointer-events-none absolute top-2.5 left-1/2 z-above -translate-x-1/2 rounded border border-dashed border-line-strong bg-surface-1/90 px-2 py-1 text-11 text-text-2"
        >Canvas placeholder: the WebGL tree draws here; these DOM nodes stand in for its hit-testing</span>
    </div>
  {/snippet}
</TreePage>
<span class="sr-only" data-log>{s.log}</span>
