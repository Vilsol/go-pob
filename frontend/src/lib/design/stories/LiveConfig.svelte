<script lang="ts">
  import ConfigPage from '../ConfigPage.svelte';
  import type { CustomOps } from '../config';
  import { modVerdicts, type BuildFixture, type CustomCell } from '../fixtures';
  import type { ConfigValue } from '$lib/display/configurations';

  type Props = { fx: BuildFixture; showAll?: boolean; modOnly?: boolean };
  let { fx, showAll = false, modOnly = false }: Props = $props();

  const verdict = (text: string): CustomCell => ({ text, ...(modVerdicts[text.trim()] ?? { k: 'empty', parsed: '' }) });

  let s = $state({
    inputs: { ...fx.config.inputs } as Record<string, ConfigValue | undefined>,
    filter: { q: '', showAll, modOnly },
    set: fx.config.activeSet,
    custom: structuredClone(fx.config.custom.length ? fx.config.custom : [{ title: 'Default', enabled: true, cells: [] }]),
    sel: 0
  });
  const cells = (g: number) => s.custom[g].cells;
  const ops: CustomOps = {
    onselect: (g) => (s.sel = g),
    ontoggle: (g, on) => (s.custom[g].enabled = on),
    onrename: (g, title) => (s.custom[g].title = title),
    onaddgroup: () => {
      s.custom.push({ title: `Group ${s.custom.length + 1}`, enabled: true, cells: [] });
      s.sel = s.custom.length - 1;
    },
    ondeletegroup: (g) => {
      s.custom.splice(g, 1);
      s.sel = Math.max(0, g - 1);
    },
    oncell: (g, c, text) => (cells(g)[c] = verdict(text)),
    oncommit: (g, c, text) => (cells(g)[c] = verdict(text)),
    onmove: (g, c, by) => {
      const to = c + by;
      if (to < 0 || to >= cells(g).length) return;
      const [x] = cells(g).splice(c, 1);
      cells(g).splice(to, 0, x);
    },
    onremove: (g, c) => cells(g).splice(c, 1),
    onaddcell: (g) => cells(g).push({ text: '', k: 'empty', parsed: '' })
  };
</script>

<ConfigPage
  inputs={s.inputs}
  placeholders={fx.config.placeholders}
  relevance={fx.config.relevance}
  sets={fx.config.sets}
  activeSet={s.set}
  custom={s.custom}
  customSelected={s.sel}
  bind:filter={s.filter}
  onset={(key: string, value: ConfigValue) => {
    s.inputs[key] = value;
  }}
  onreset={() => {
    s.inputs = {};
  }}
  onconfigset={(i: number) => {
    s.set = i;
  }}
  customOps={ops} />
