<script lang="ts">
  import { resource } from 'runed';
  import { buildRevision, currentBuild, outputs } from '$lib/global.js';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import type { ConfigValue } from '$lib/display/configurations';
  import { cellsToText, modSummary, textToCells, type TextCells } from '$lib/display/custom_mods';
  import type { calculator, exposition, pob } from '$lib/types';
  import ConfigPage from '$lib/design/ConfigPage.svelte';
  import Dialog from '$lib/design/Dialog.svelte';
  import Button from '$lib/design/Button.svelte';
  import type { ConfigFilter, CustomOps, Inputs } from '$lib/design/config';
  import type { CustomGroup } from '$lib/design/fixtures';
  import type { SetListOp } from '$lib/design/set_list';

  // ConfigTab.lua:15-86

  const config = resource(
    () => [$buildRevision, $currentBuild === undefined] as const,
    async () => {
      const b = $currentBuild;
      if (!b || !syncWrap) return undefined;
      const [inputs, placeholders, cfg] = await Promise.all([syncWrap.GetAllConfigOptions(), syncWrap.GetConfigPlaceholders(), syncWrap.GetConfigSets()]);
      // ConfigSetListControl.lua:73
      const sets = cfg.sets.map((s) => s.title || 'Default');
      return { inputs: inputs as Inputs, placeholders, sets, setIds: cfg.sets.map((s) => s.id), activeSet: cfg.active };
    }
  );
  const inputs = $derived(config.current?.inputs ?? {});

  const relevance = $derived($outputs?.ConfigRelevance ?? ({} as calculator.ConfigRelevance));

  let filter = $state<ConfigFilter>({ q: '', showAll: false, modOnly: false });

  // ConfigOptions.lua:1457-1589, 2246-2286
  const refreshPlaceholders = async () => {
    const placeholders = await syncWrap?.GetConfigPlaceholders();
    if (placeholders && config.current) config.mutate({ ...config.current, placeholders });
  };
  const onset = (key: string, value: ConfigValue) => {
    if (inputs[key] === value) return;
    if (config.current) config.mutate({ ...config.current, inputs: { ...inputs, [key]: value } });
    syncWrap?.SetConfigOption(key, value).then(refreshPlaceholders).catch(logError);
  };
  const onunset = (key: string) => {
    if (!config.current || inputs[key] === undefined) return;
    const rest = { ...inputs };
    delete rest[key];
    config.mutate({ ...config.current, inputs: rest });
    syncWrap
      ?.SetConfigOption(key, null as unknown as number)
      .then(refreshPlaceholders)
      .catch(logError);
  };

  let confirming = $state(false);
  const reset = async () => {
    confirming = false;
    await syncWrap?.ResetConfigOptions();
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
    if (!syncWrap || targets.length === 0) return;
    const verdicts = await syncWrap.ParseModLines(targets.map((c) => c.text));
    targets.forEach((c, i) => {
      c.line = verdicts[i];
    });
  };

  const customBlocks = resource(
    () => [$buildRevision, $currentBuild === undefined] as const,
    async () => {
      if (!$currentBuild || !syncWrap) return;
      groups = (await syncWrap.GetCustomModifierBlocks()).map(toGroup);
      selected = Math.min(selected, Math.max(0, groups.length - 1));
      await parseAll(groups.flatMap((g) => g.cells));
    }
  );

  // ConfigTab.lua:152-155
  const onconfigset = async (i: number) => {
    const id = config.current?.setIds[i];
    if (id === undefined || !syncWrap) return;
    await syncWrap.SetActiveConfigSet(id);
    await Promise.all([config.refetch(), customBlocks.refetch()]);
  };

  // ConfigSetListControl.lua:13-110
  const onconfigsets = async (op: SetListOp) => {
    const ids = config.current?.setIds;
    if (!ids || !syncWrap) return;
    if (op.op === 'new') await syncWrap.EditConfigSets(op);
    else if (op.op === 'move') await syncWrap.EditConfigSets(op);
    else if (op.op === 'delete') await syncWrap.EditConfigSets({ op: 'delete', id: ids[op.index] });
    else await syncWrap.EditConfigSets({ op: op.op, id: ids[op.index], title: op.title });
    await Promise.all([config.refetch(), customBlocks.refetch()]);
  };

  const write = () => {
    const blocks: pob.CustomModifierBlock[] = groups.map((g) => ({
      Title: g.title,
      Enabled: g.enabled ? 'true' : 'false',
      Text: cellsToText({ ...g.shape, cells: g.cells.map((c) => c.text) })
    }));
    syncWrap?.SetCustomModifierBlocks(blocks).catch(logError);
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
    togglewhatif: async (g) => syncWrap?.CustomModGroupCompare(g),
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
      void syncWrap?.ParseModLines([text]).then(([verdict]) => {
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
      list: async (search) => (await syncWrap?.ModBrowserList(search)) ?? [],
      supported: async (line) => (await syncWrap?.ModBrowserSupported(line)) ?? false,
      onadd: (g, line) => {
        const cells = groups[g].cells;
        const last = cells[cells.length - 1];
        const cell = last && last.text === '' && last.draft === undefined ? last : cells[cells.push({ text: '' }) - 1];
        cell.text = line;
        void syncWrap?.ParseModLines([line]).then(([verdict]) => {
          if (cell.text === line) cell.line = verdict;
        });
        write();
      }
    }
  };
</script>

{#if config.current}
  <ConfigPage
    {inputs}
    placeholders={config.current.placeholders}
    {relevance}
    sets={config.current.sets.length ? config.current.sets : ['Default']}
    activeSet={config.current.activeSet}
    {custom}
    customSelected={selected}
    customOps={ops}
    bind:filter
    {onset}
    {onunset}
    onconfigset={(i: number) => void onconfigset(i).catch(logError)}
    onconfigsets={(op: SetListOp) => void onconfigsets(op).catch(logError)}
    whatif={async (key: string, value: ConfigValue) => syncWrap?.ConfigCompare(key, value)}
    output={$outputs?.Output}
    revision={$outputs}
    onreset={() => (confirming = true)} />
{:else}
  <p class="note" role="status">Loading the configuration…</p>
{/if}

<Dialog bind:open={confirming} title="Reset to defaults?">
  <p class="m-0 text-13 text-text-2">Every config option goes back to its default. Custom modifiers are kept.</p>
  {#snippet actions()}
    <Button onclick={() => (confirming = false)}>Cancel</Button>
    <Button variant="danger" onclick={reset}>Confirm reset</Button>
  {/snippet}
</Dialog>
