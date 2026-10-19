<script lang="ts">
  import TreePage from '../TreePage.svelte';
  import TreeView from '../TreeView.svelte';
  import type { BuildFixture, TreeNodeFx } from '../fixtures';
  import type { CanvasApi } from '../tree';
  import { gestureDriver } from '../mobile/tree_gestures';
  import UndoToast from '../mobile/UndoToast.svelte';

  type Props = { fx: BuildFixture; zoom?: number; selected?: number; onedit?: (n: TreeNodeFx) => void };
  let { fx, zoom: zoom0 = 2.5, selected: selected0, onedit }: Props = $props();

  let tree = $state(structuredClone(fx.tree));
  let s = $state({
    alloc: Object.fromEntries(fx.tree.nodes.filter((n) => n.a).map((n) => [n.id, true])) as Record<number, boolean>,
    q: '',
    mastery: null as number | null,
    log: '',
    className: fx.className,
    ascendancy: fx.ascendClassName,
    level: fx.level
  });
  let bw = $state(0);
  let bh = $state(0);
  let view = $state({ zoom: zoom0, dx: 0, dy: 0 });
  const vw = $derived(bw * view.zoom);
  const vh = $derived(vw / tree.aspect);
  const ox = $derived((bw - vw) / 2 + view.dx);
  const oy = $derived((bh - vh) / 2 + view.dy);
  const byId = $derived(new Map(tree.nodes.map((n) => [n.id, n])));
  const project = (id: number) => {
    const n = byId.get(id);
    return n ? { x: ox + n.x * vw, y: oy + n.y * vh } : undefined;
  };
  let sel = $state<TreeNodeFx | undefined>(selected0 !== undefined ? fx.tree.nodes.find((n) => n.id === selected0) : undefined);

  const pick = (n: TreeNodeFx) => {
    onedit?.(n);
    if (s.alloc[n.id]) {
      for (const id of [n.id, ...(n.a ? (n.loseIds ?? []) : [])]) delete s.alloc[id];
      return;
    }
    for (let x: TreeNodeFx | undefined = n; x && !s.alloc[x.id]; x = x.next !== undefined ? byId.get(x.next) : undefined) s.alloc[x.id] = true;
  };

  let toast = $state<{ text: string; undo: Record<number, boolean> } | undefined>();
  let toastTimer: ReturnType<typeof setTimeout> | undefined;

  const hover = (api: CanvasApi, n: TreeNodeFx | undefined) => {
    if (sel && sel !== n) api.leave(sel);
    sel = n;
    if (n) api.enter(n);
  };
  const allocate = (api: CanvasApi, n: TreeNodeFx) => {
    const before = { ...s.alloc };
    api.pick(n);
    const was = Object.keys(before).length;
    const now = Object.keys(s.alloc).length;
    if (was === now) return;
    clearTimeout(toastTimer);
    toast = { text: now > was ? `Allocated ${now - was}` : `Unallocated ${was - now}`, undo: before };
    toastTimer = setTimeout(() => (toast = undefined), 5000);
  };
  const undo = () => {
    if (!toast) return;
    s.alloc = toast.undo;
    toast = undefined;
  };
  // one driver for the canvas's life, on whichever api the snippet last rendered with
  let canvasApi!: CanvasApi;
  const driver = gestureDriver({
    gesture: (g) => {
      const n = g.node !== undefined ? byId.get(g.node) : undefined;
      if (g.kind === 'tap') hover(canvasApi, n);
      else if (g.kind === 'double' && n) allocate(canvasApi, n);
      else if (g.kind === 'hold' && n) {
        navigator.vibrate?.(15);
        hover(canvasApi, undefined);
        canvasApi.menu(n, project(n.id) ?? { x: 0, y: 0 });
      }
    },
    pan: (dx, dy) => {
      view.dx += dx;
      view.dy += dy;
    },
    pinch: (f) => (view.zoom = Math.min(8, Math.max(0.8, view.zoom * f)))
  });
  const gestures = (a: CanvasApi) => {
    canvasApi = a;
    return driver;
  };
  const nodeOf = (e: PointerEvent) => {
    const node = Number((e.target as HTMLElement).closest<HTMLElement>('[data-node]')?.dataset.node);
    return byId.has(node) ? node : undefined;
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
  onzoom={(z: 'in' | 'out' | 'fit') => {
    if (z === 'fit') view = { zoom: 1, dx: 0, dy: 0 };
    else view.zoom = Math.min(8, Math.max(0.8, view.zoom * (z === 'in' ? 1.25 : 0.8)));
  }}
  onmastery={(id: number, effect: number) => {
    const m = tree.nodes.find((n) => n.id === id);
    if (m) m.effect = effect;
  }}
  onmenuaction={(action: string, n: TreeNodeFx) => (s.log = `${action} ${n.name}`)}
  ontimeless={() => (s.log = 'timeless')}
  onpathmode={(m: string) => (s.log = `path ${m}`)}
  onclass={(c: string) => (s.className = c)}
  onascendancy={(a: string) => (s.ascendancy = a)}
  onlevel={(l: number) => (s.level = l)}>
  {#snippet canvas(api: CanvasApi)}
    {@const d = gestures(api)}
    <div
      class="absolute inset-0 touch-none select-none [-webkit-touch-callout:none]"
      bind:clientWidth={bw}
      bind:clientHeight={bh}
      role="presentation"
      data-phone-canvas
      onpointerdown={(e) => d.down(e, nodeOf(e))}
      onpointermove={(e) => d.move(e)}
      onpointerup={(e) => d.up(e)}
      onpointercancel={(e) => d.up(e)}
      oncontextmenu={(e) => e.preventDefault()}
      onclickcapture={(e) => {
        // a pointer's click is already a gesture; a keyboard click (detail 0) still reaches the node
        if (e.detail > 0) e.stopPropagation();
      }}>
      <div class="absolute top-(--oy) left-(--ox) w-(--vw)" style:--ox="{ox}px" style:--oy="{oy}px" style:--vw="{vw}px">
        <TreeView
          {tree}
          label="Canvas placeholder"
          alloc={s.alloc}
          hovered={sel?.id ?? api.hovered}
          describedBy={api.describedBy}
          path={api.path}
          lose={api.lose}
          matches={api.matches}
          active={(n: TreeNodeFx) => n.k !== 'c'}
          onpick={(n: TreeNodeFx) => allocate(api, n)} />
      </div>
    </div>
    {#if toast}<UndoToast text={toast.text} onundo={undo} />{/if}
  {/snippet}
</TreePage>
<span class="sr-only" data-log>{s.log}</span>
