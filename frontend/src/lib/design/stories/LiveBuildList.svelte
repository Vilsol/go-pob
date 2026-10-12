<script lang="ts">
  import { onMount } from 'svelte';
  import BuildListPage from '../BuildListPage.svelte';
  import SettingsDialog from '../SettingsDialog.svelte';
  import AboutDialog from '../AboutDialog.svelte';
  import { buildList, settingsSections, settingsSample, type Setting } from '../fixtures/app';
  import { buildId, type ColKey } from '../builds';
  import { aboutInfo } from '../fixtures';

  type Props = { folder?: string; about?: boolean; settings?: boolean; step?: number; app?: boolean };
  let { folder: start = '', about = false, settings = false, step = 120, app = false }: Props = $props();
  let actions = $state<string[]>([]);
  const log = (a: string) => () => actions.push(a);
  const appProps = $derived(
    app
      ? {
          columns: ['name', 'cls', 'level', 'edited'] as ColKey[],
          onnew: log('new'),
          onimport: log('import'),
          onnewfolder: log('new folder'),
          oncopy: (id: string) => actions.push(`copy ${id}`),
          onrename: (id: string) => actions.push(`rename ${id}`),
          ondelete: (id: string) => actions.push(`delete ${id}`),
          onopen: (id: string) => actions.push(`open ${id}`)
        }
      : {}
  );

  const read = () => {
    const m = /^#folder=(.*)$/.exec(location.hash);
    return m ? decodeURIComponent(m[1]) : '';
  };
  const ancestors = (f: string) => f.split('/').map((_, i, a) => a.slice(0, i + 1).join('/'));

  let s = $state({
    builds: structuredClone(buildList.builds),
    folder: start,
    open: Object.fromEntries(
      ['League starters', 'League starters/Necromancer', 'Bossers', 'Bossers/Uber', ...(start ? ancestors(start) : [])].map((f) => [f, true])
    ) as Record<string, boolean>,
    fresh: {} as Record<string, boolean>,
    recalc: null as { done: number; total: number; name: string } | null,
    about,
    settings,
    values: {} as Record<string, Setting['value']>
  });

  onMount(() => {
    history.replaceState(null, '', `#folder=${encodeURIComponent(s.folder)}`);
    const pop = () => (s.folder = read());
    addEventListener('popstate', pop);
    return () => removeEventListener('popstate', pop);
  });

  const go = (f: string) => {
    for (const a of f ? ancestors(f) : []) s.open[a] = true;
    if (f === s.folder) return;
    s.folder = f;
    history.pushState(null, '', `#folder=${encodeURIComponent(f)}`);
  };

  const recalc = () => {
    const todo = s.builds.filter((b) => b.version !== buildList.gameVersion && !s.fresh[b.name]).map((b) => b.name);
    let i = 0;
    s.recalc = { done: 0, total: todo.length, name: todo[0] };
    const tick = () => {
      s.fresh[todo[i++]] = true;
      s.recalc = i < todo.length ? { done: i, total: todo.length, name: todo[i] } : null;
      if (i < todo.length) setTimeout(tick, step);
    };
    setTimeout(tick, step);
  };
</script>

<BuildListPage
  builds={s.builds}
  folders={buildList.folders}
  classes={buildList.classes}
  gameVersion={buildList.gameVersion}
  folder={s.folder}
  open={s.open}
  fresh={s.fresh}
  recalc={s.recalc}
  onfolder={go}
  ontoggle={(f: string) => (s.open[f] = !s.open[f])}
  onmove={(id: string, dest: string) => {
    const b = s.builds.find((x) => buildId(x) === id);
    if (b) b.folder = dest;
  }}
  onrecalc={recalc}
  onabout={() => (s.about = true)}
  onsettings={() => (s.settings = true)}
  {...appProps}>
  {#snippet dialogs()}
    <AboutDialog bind:open={s.about} info={aboutInfo} />
    <SettingsDialog
      bind:open={s.settings}
      sections={settingsSections}
      values={s.values}
      sample={settingsSample}
      onchange={(id: string, v: Setting['value']) => (s.values[id] = v)}
      onaction={(id: string) => actions.push(`setting ${id}`)}
      onreset={(sec: string) => {
        for (const r of settingsSections.find((x) => x.id === sec)?.rows ?? []) delete s.values[r.id];
      }} />
  {/snippet}
</BuildListPage>
{#if app}<output aria-label="Actions taken" class="sr-only">{actions.join('; ')}</output>{/if}
