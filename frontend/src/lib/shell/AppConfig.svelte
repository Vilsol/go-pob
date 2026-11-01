<script lang="ts">
  import EmptyState from '$lib/design/EmptyState.svelte';
  import { resource } from 'runed';
  import { currentEngine } from '$lib/go/engine';
  import { logError } from '$lib/utils';
  import type { ConfigValue } from '$lib/display/configurations';
  import { cellsToText, modSummary, textToCells, type TextCells } from '$lib/display/custom_mods';
  import type { exposition, pob } from '$lib/types';
  import ConfigPage from '$lib/design/ConfigPage.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import DialogActions from '$lib/design/DialogActions.svelte';
  import type { ConfigFilter, ConfigTable, CustomOps } from '$lib/design/config';
  import type { CustomGroup } from '$lib/design/fixtures';
  import type { SetListOp } from '$lib/design/set_list';

  // ConfigTab.lua:15-86

  const engine = currentEngine().config;
  const { revision, relevance, output } = engine;
  const calculated = currentEngine().calculated;
  const custom_ = engine.custom;
  const failed = currentEngine().error;

  // kept out of the resource's state, which would proxy the whole table
  let table = $state.raw<ConfigTable>();
  const config = resource(
    () => [$revision] as const,
    async () => {
      const read = engine.inputs();
      if (!read) return undefined;
      const [inputs, placeholders, cfg, t] = await Promise.all([read, engine.placeholders(), engine.sets?.read(), engine.table()]);
      if (!inputs) return undefined;
      table = t;
      // ConfigSetListControl.lua:73
      const sets = cfg?.sets.map((s) => s.title || 'Default') ?? [];
      return { inputs, placeholders: placeholders ?? {}, sets, setIds: cfg?.sets.map((s) => s.id) ?? [], activeSet: cfg?.active ?? 1 };
    }
  );
  const inputs = $derived(config.current?.inputs ?? {});

  let filter = $state<ConfigFilter>({ q: '', showAll: false, modOnly: false });

  // ConfigOptions.lua:1457-1589, 2246-2286
  const refreshPlaceholders = async () => {
    const placeholders = await engine.placeholders();
    if (placeholders && config.current) config.mutate({ ...config.current, placeholders });
  };
  const onset = (key: string, value: ConfigValue) => {
    if (inputs[key] === value) return;
    if (config.current) config.mutate({ ...config.current, inputs: { ...inputs, [key]: value } });
    engine.set(key, value)?.then(refreshPlaceholders).catch(logError);
  };
  const onunset = (key: string) => {
    if (!config.current || inputs[key] === undefined) return;
    const rest = { ...inputs };
    delete rest[key];
    config.mutate({ ...config.current, inputs: rest });
    engine.set(key, undefined)?.then(refreshPlaceholders).catch(logError);
  };

  let confirming = $state(false);
  const reset = async () => {
    confirming = false;
    await engine.reset?.();
    await config.refetch();
  };

  type Cell = { text: string; draft?: string; line?: exposition.ModLine };
  type Group = { title: string; enabled: boolean; shape: TextCells; cells: Cell[] };
  const toGroup = (b: pob.CustomModifierBlock): Group => {
    const shape = textToCells(b.Text ?? '');
    return { title: b.Title ?? 'Default', enabled: b.Enabled === undefined || b.Enabled === 'true', shape, cells: shape.cells.map((text) => ({ text })) };
  };

  let groups = $state<Group[]>([]);
  let selected = $state(0);

  const parseAll = async (targets: Cell[]) => {
    if (!custom_ || targets.length === 0) return;
    const verdicts = await custom_.parse(targets.map((c) => c.text));
    if (!verdicts) return;
    targets.forEach((c, i) => {
      c.line = verdicts[i];
    });
  };

  const customBlocks = resource(
    () => [$revision] as const,
    async () => {
      const read = custom_?.read();
      if (!read) return;
      groups = (await read).map(toGroup);
      selected = Math.min(selected, Math.max(0, groups.length - 1));
      await parseAll(groups.flatMap((g) => g.cells));
    }
  );

  // ConfigTab.lua:152-155
  const onconfigset = async (i: number) => {
    const id = config.current?.setIds[i];
    if (id === undefined || !engine.sets) return;
    await engine.sets.select(id);
    await Promise.all([config.refetch(), customBlocks.refetch()]);
  };

  // ConfigSetListControl.lua:13-110
  const onconfigsets = async (op: SetListOp) => {
    const ids = config.current?.setIds;
    const sets = engine.sets;
    if (!ids || !sets) return;
    if (op.op === 'new') await sets.edit(op);
    else if (op.op === 'move') await sets.edit(op);
    else if (op.op === 'delete') await sets.edit({ op: 'delete', id: ids[op.index] });
    else await sets.edit({ op: op.op, id: ids[op.index], title: op.title });
    await Promise.all([config.refetch(), customBlocks.refetch()]);
  };

  const write = () => {
    const blocks: pob.CustomModifierBlock[] = groups.map((g) => ({
      Title: g.title,
      Enabled: g.enabled ? 'true' : 'false',
      Text: cellsToText({ ...g.shape, cells: g.cells.map((c) => c.text) })
    }));
    custom_?.write(blocks)?.catch(logError);
  };

  const K = { parsed: 'ok', unsupported: 'bad', comment: 'comment' } as const;
  const custom = $derived<CustomGroup[]>(
    groups.map((g) => ({
      title: g.title,
      enabled: g.enabled,
      cells: g.cells.map((c) => ({
        text: c.draft ?? c.text,
        k: K[c.line?.Status as keyof typeof K] ?? 'empty',
        parsed: c.line ? modSummary(c.line) : ''
      }))
    }))
  );

  const ops: CustomOps = {
    togglewhatif: async (g) => custom_?.compare(g),
    onselect: (g) => (selected = g),
    ontoggle: (g, on) => {
      groups[g].enabled = on;
      write();
    },
    onrename: (g, title) => {
      groups[g].title = title;
      write();
    },
    onaddgroup: () => {
      groups.push({ title: `Group ${groups.length + 1}`, enabled: true, shape: textToCells(''), cells: [{ text: '' }] });
      selected = groups.length - 1;
      write();
    },
    // ConfigTab.lua:23-33
    ondeletegroup: (g) => {
      groups.splice(g, 1);
      if (groups.length === 0) groups.push(toGroup({ Title: 'Default', Enabled: 'true', Text: '' }));
      selected = Math.max(0, Math.min(g, groups.length - 1));
      write();
    },
    oncell: (g, c, text) => {
      const cell = groups[g].cells[c];
      cell.draft = text;
      void custom_?.parse([text])?.then(([verdict]) => {
        if ((cell.draft ?? cell.text) === text) cell.line = verdict;
      });
    },
    oncommit: (g, c, text) => {
      const cell = groups[g].cells[c];
      cell.draft = undefined;
      if (cell.text === text) return;
      cell.text = text;
      write();
    },
    onmove: (g, c, by) => {
      const cells = groups[g].cells;
      const to = c + by;
      if (to < 0 || to >= cells.length) return;
      const [cell] = cells.splice(c, 1);
      cells.splice(to, 0, cell);
      write();
    },
    onremove: (g, c) => {
      const cells = groups[g].cells;
      cells.splice(c, 1);
      if (cells.length === 0) cells.push({ text: '' });
      write();
    },
    onaddcell: (g) => {
      groups[g].cells.push({ text: '' });
    },
    // ConfigTab.lua:1524-1536
    modbrowser: {
      list: async (search) => (await custom_?.browser.list(search)) ?? [],
      sources: async (line) => (await custom_?.browser.sources(line)) ?? [],
      onadd: (g, line) => {
        const cells = groups[g].cells;
        const last = cells[cells.length - 1];
        const cell = last && last.text === '' && last.draft === undefined ? last : cells[cells.push({ text: '' }) - 1];
        cell.text = line;
        void custom_?.parse([line])?.then(([verdict]) => {
          if (cell.text === line) cell.line = verdict;
        });
        write();
      }
    }
  };
</script>

{#if $failed}<p class="m-0 text-14" role="alert">{$failed}</p>{/if}
{#if config.current && table}
  <ConfigPage
    {table}
    {inputs}
    placeholders={config.current.placeholders}
    relevance={$relevance}
    sets={config.current.sets.length ? config.current.sets : ['Default']}
    activeSet={config.current.activeSet}
    {custom}
    customSelected={selected}
    customOps={custom_ && ops}
    bind:filter
    {onset}
    {onunset}
    onconfigset={engine.sets && ((i: number) => void onconfigset(i).catch(logError))}
    onconfigsets={engine.sets && ((op: SetListOp) => void onconfigsets(op).catch(logError))}
    whatif={engine.compare && (async (key: string, value: ConfigValue) => engine.compare?.(key, value))}
    output={$output}
    revision={$calculated}
    onreset={engine.reset && (() => (confirming = true))} />
{:else}
  <EmptyState busy text="Loading the configuration…" />
{/if}

<Dialog bind:open={confirming} title="Reset to defaults?">
  <p class="m-0 text-13 text-text-2">Every config option goes back to its default. Custom modifiers are kept.</p>
  {#snippet actions()}
    <DialogActions confirm="Confirm reset" tone="danger" oncancel={() => (confirming = false)} onconfirm={reset} />
  {/snippet}
</Dialog>
