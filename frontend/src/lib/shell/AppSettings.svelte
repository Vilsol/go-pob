<script lang="ts">
  import SettingsDialog from '$lib/design/SettingsDialog.svelte';
  import type { Setting, SettingsSection } from '$lib/design/fixtures/app';
  import { backendLoaded, devMode, fontScaling, nodePowerTheme, outputs, zoomSensitivity } from '$lib/global.js';
  import { ITEMS_LAYOUTS, THEMES, itemsLayout, showAnimations, theme, type ItemsLayout, type Theme } from '$lib/display/appearance';
  import { allAffixSliders } from '$lib/design/items';
  import { addToast } from '$lib/design/toast';
  import { checkForUpdate } from '$lib/pwa';
  import { zipStore } from './zip';
  import { NODE_POWER_THEMES, type NodePowerTheme } from '$lib/skill_tree/heat';
  import { numberFormat, type NumberFormat } from '$lib/display/number_format';
  import { showTitlebarName } from '$lib/display/window_title';
  import {
    defaultCharLevel,
    defaultGemQuality,
    defaultItemAffixQuality,
    migrateEldritchImplicits,
    notSupportedModTooltips,
    showAllItemAffixes,
    showFlavourText,
    showWarnings,
    slotOnlyTooltips
  } from '$lib/display/item_options';
  import { notSupportedNote } from '$lib/design/pob_text';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import { colorHighlight, colorNegative, colorPositive, edgeSearchHighlight, hexColour } from '$lib/skill_tree/options';
  import { get } from 'svelte/store';
  import { untrack } from 'svelte';

  type Props = { open?: boolean };
  let { open = $bindable(false) }: Props = $props();

  const sections: SettingsSection[] = [
    {
      id: 'general',
      label: 'General',
      rows: [
        {
          id: 'fontScaling',
          label: 'Font scaling',
          kind: 'range',
          value: 12,
          hint: 'Base text size for the whole app',
          min: 6,
          max: 18,
          step: 0.5,
          unit: 'pt',
          onRelease: true
        },
        {
          id: 'theme',
          label: 'Theme',
          kind: 'select',
          value: 'Dark',
          hint: 'Light and high-contrast variants; the passive tree keeps its own colours',
          opts: [...THEMES]
        },
        // Main.lua:1027-1029
        { id: 'animations', label: 'Show animations', kind: 'check', value: true, hint: 'Off also honours the system “reduce motion” setting' },
        { id: 'titleBuild', label: 'Show build name in window title', kind: 'check', value: true, hint: 'The tab title becomes “<build> (<class>) - go-pob”' },
        { id: 'devMode', label: 'Dev mode', kind: 'check', value: false, hint: 'Passive tooltips show the node id' },
        // Main.lua:1085-1096
        {
          id: 'defLevel',
          label: 'Default character level',
          kind: 'number',
          value: 1,
          hint: 'A new build’s level; above 1 it is set by hand',
          min: 1,
          max: 100
        },
        { id: 'gemQuality', label: 'Default gem quality', kind: 'number', value: 0, hint: 'A new build’s default gem quality', min: 0, max: 23 },
        // Main.lua:1098-1106
        {
          id: 'affixQuality',
          label: 'Default item affix quality',
          kind: 'range',
          value: 0.5,
          hint: 'Where in its range a new affix rolls (0 = min, 1 = max)',
          min: 0,
          max: 1,
          step: 0.05,
          unit: ''
        },
        // Main.lua:1109-1113
        { id: 'showWarnings', label: 'Show build warnings', kind: 'check', value: true, hint: 'The warnings in the sidebar and the top bar' }
      ]
    },
    {
      // Main.lua:1056-1079
      id: 'numbers',
      label: 'Numbers',
      rows: [
        { id: 'thousands', label: 'Show thousands separators', kind: 'check', value: true },
        { id: 'thousandsSep', label: 'Thousands separator', kind: 'select', value: ',', opts: [',', '.', 'space', '’'] },
        { id: 'decimalSep', label: 'Decimal separator', kind: 'select', value: '.', opts: ['.', ','] },
        { id: 'compact', label: 'Compact large numbers (e.g. 12.3K)', kind: 'check', value: false }
      ]
    },
    {
      id: 'tree',
      label: 'Passive tree',
      rows: [
        {
          id: 'zoomSensitivity',
          label: 'Zoom sensitivity',
          kind: 'range',
          value: 5,
          hint: 'Mouse wheel and the zoom buttons',
          min: 1,
          max: 10,
          step: 0.1,
          unit: ''
        },
        // Main.lua:957-961
        {
          id: 'nodePowerTheme',
          label: 'Node Power colours',
          kind: 'select',
          value: 'RED/BLUE',
          opts: NODE_POWER_THEMES,
          hint: 'The heat map of Show Node Power'
        },
        // Main.lua:969-1013
        { id: 'colorPositive', label: 'Hex colour for positive values', kind: 'color', value: '#6cb4ff', hint: 'Stat differences and breakdowns' },
        { id: 'colorNegative', label: 'Hex colour for negative values', kind: 'color', value: '#ff9b52', hint: 'Stat differences and breakdowns' },
        { id: 'colorHighlight', label: 'Hex colour for highlight nodes', kind: 'color', value: '#ff0000', hint: 'The passive tree search' },
        { id: 'edgeSearchHighlight', label: 'Show search circles at viewport edge', kind: 'check', value: true }
      ]
    },
    {
      id: 'items',
      label: 'Items',
      rows: [
        {
          id: 'itemsLayout',
          label: 'Default items view',
          kind: 'select',
          value: 'Board',
          opts: ['Board', 'List'],
          hint: 'How the Items page lays out the equipped slots when opened'
        },
        {
          id: 'slotTooltips',
          label: 'Show tooltips only for affected slots',
          kind: 'check',
          value: true,
          hint: 'An item hovered in a slot’s menu compares against that slot only (Main.lua:1115)'
        },
        // Main.lua:1020-1024, 1120-1131
        {
          id: 'flavourText',
          label: 'Styled tooltips with flavour text',
          kind: 'check',
          value: true,
          hint: 'Item tooltips show the flavour text and the influence labels'
        },
        {
          id: 'migrateEldritch',
          label: 'Copy Eldritch implicits onto display item',
          kind: 'check',
          value: true,
          hint: 'A pasted or opened item without influence takes the equipped item’s Eldritch implicits'
        },
        {
          id: 'notSupported',
          label: 'Show tooltip for unsupported mods',
          kind: 'check',
          value: true,
          hint: 'Show “(Not supported in PoB yet)” next to unsupported mods'
        },
        // Main.lua:121, 1032-1035
        {
          id: 'affixSliders',
          label: 'Show all item affix sliders',
          kind: 'check',
          value: true,
          hint: 'The item editor shows a roll slider on every ranged line; off, a menu picks the line one slider rolls'
        }
      ]
    },
    {
      id: 'data',
      label: 'Data and storage',
      rows: [
        { id: 'dataVersion', label: 'Game data', kind: 'info', value: '…', hint: 'Loaded at start-up' },
        { id: 'checkData', label: 'Check for newer game data', kind: 'button', value: 'Check now', hint: 'A newer version of the app brings newer game data' },
        { id: 'cache', label: 'Cached data', kind: 'info', value: '…', hint: 'Everything this site keeps in the browser: game data, the offline copy, builds' },
        {
          id: 'clearCache',
          label: 'Clear cached data',
          kind: 'button',
          value: 'Clear…',
          hint: 'Builds are not affected; game data downloads again',
          danger: true,
          confirm: 'Clear the cached game data? Builds are kept.'
        },
        { id: 'exportAll', label: 'Export all builds', kind: 'button', value: 'Export .zip', hint: 'Builds live only in this browser' },
        {
          id: 'resetAll',
          label: 'Reset all settings',
          kind: 'button',
          value: 'Reset…',
          hint: 'Every section back to its defaults; builds and cached data are not affected',
          danger: true,
          confirm: 'Put every setting back to its default?'
        }
      ]
    }
  ];

  const list = <T,>(o: T[] | Record<string, T> | undefined): T[] =>
    Array.isArray(o) ? o : Object.entries(o ?? {}).flatMap(([k, v]) => (/^\d+$/.test(k) ? [v] : []));
  let dataVersion = $state('…');
  let cacheSize = $state('…');
  const measure = () =>
    navigator.storage
      ?.estimate?.()
      .then((e) => (cacheSize = `${((e.usage ?? 0) / 1048576).toFixed(1)} MB`))
      .catch(logError);
  $effect(() => {
    if (!open) return;
    Promise.all([syncWrap?.GameDataVersion(), syncWrap?.BuildInfo()])
      .then(([v, bi]) => {
        const mod = list(bi?.Deps).find((d) => d?.Path === 'github.com/Vilsol/go-pob-data');
        dataVersion = [v, mod && `go-pob-data ${mod.Version}`].filter(Boolean).join(' · ');
      })
      .catch(logError);
    void measure();
  });
  const download = (name: string, data: Uint8Array<ArrayBuffer>) => {
    const a = document.createElement('a');
    a.href = URL.createObjectURL(new Blob([data], { type: 'application/zip' }));
    a.download = name;
    a.click();
    URL.revokeObjectURL(a.href);
  };

  const values = $derived({
    fontScaling: $fontScaling,
    theme: $theme,
    animations: $showAnimations,
    affixSliders: $showAllItemAffixes,
    itemsLayout: $itemsLayout === 'list' ? 'List' : 'Board',
    dataVersion,
    cache: cacheSize,
    devMode: $devMode,
    titleBuild: $showTitlebarName,
    zoomSensitivity: $zoomSensitivity,
    nodePowerTheme: $nodePowerTheme,
    thousands: $numberFormat.showThousandsSeparators,
    compact: $numberFormat.useCompactValues,
    thousandsSep: $numberFormat.thousandsSeparator === ' ' ? 'space' : $numberFormat.thousandsSeparator,
    decimalSep: $numberFormat.decimalSeparator,
    slotTooltips: $slotOnlyTooltips,
    showWarnings: $showWarnings,
    flavourText: $showFlavourText,
    migrateEldritch: $migrateEldritchImplicits,
    notSupported: $notSupportedModTooltips,
    defLevel: $defaultCharLevel,
    gemQuality: $defaultGemQuality,
    affixQuality: $defaultItemAffixQuality,
    colorPositive: hexColour($colorPositive) ?? '#6cb4ff',
    colorNegative: hexColour($colorNegative) ?? '#ff9b52',
    colorHighlight: hexColour($colorHighlight) ?? '#ff0000',
    edgeSearchHighlight: $edgeSearchHighlight
  });
  $effect(() => notSupportedNote.set($notSupportedModTooltips));
  $effect(() => allAffixSliders.set($showAllItemAffixes));
  $effect(() => {
    syncWrap?.SetShowFlavourText($showFlavourText).catch(logError);
  });
  // ItemsTab.lua:5160-5166
  $effect(() => {
    syncWrap?.SetItemModDebug($devMode).catch(logError);
  });
  $effect(() => {
    const q = $defaultItemAffixQuality;
    backendLoaded.then(() => syncWrap?.SetDefaultItemAffixQuality(q)).catch(logError);
  });

  const setNumbers = (change: Partial<NumberFormat>) => {
    numberFormat.update((f) => ({ ...f, ...change }));
    outputs.set(get(outputs));
  };
  // Main.lua:1191-1225
  let saved: Record<string, Setting['value']> = {};
  $effect(() => {
    if (open) saved = untrack(() => ({ ...values }));
  });
  const set = (id: string, v: Setting['value']) => {
    if (id === 'fontScaling') fontScaling.set(Math.min(Math.max(Number(v) || 12, 6), 18));
    else if (id === 'theme' && (THEMES as readonly string[]).includes(String(v))) theme.set(String(v) as Theme);
    else if (id === 'animations') showAnimations.set(!!v);
    else if (id === 'affixSliders') showAllItemAffixes.set(!!v);
    else if (id === 'itemsLayout' && (ITEMS_LAYOUTS as readonly string[]).includes(String(v).toLowerCase()))
      itemsLayout.set(String(v).toLowerCase() as ItemsLayout);
    else if (id === 'devMode') devMode.set(!!v);
    else if (id === 'titleBuild') showTitlebarName.set(!!v);
    else if (id === 'zoomSensitivity') zoomSensitivity.set(Number(v));
    else if (id === 'nodePowerTheme' && (NODE_POWER_THEMES as string[]).includes(String(v))) nodePowerTheme.set(String(v) as NodePowerTheme);
    else if (id === 'thousands') setNumbers({ showThousandsSeparators: !!v });
    else if (id === 'compact') setNumbers({ useCompactValues: !!v });
    else if (id === 'thousandsSep') setNumbers({ thousandsSeparator: v === 'space' ? ' ' : String(v) });
    else if (id === 'decimalSep') setNumbers({ decimalSeparator: String(v) });
    else if (id === 'slotTooltips') slotOnlyTooltips.set(!!v);
    else if (id === 'showWarnings') showWarnings.set(!!v);
    else if (id === 'flavourText') showFlavourText.set(!!v);
    else if (id === 'migrateEldritch') migrateEldritchImplicits.set(!!v);
    else if (id === 'notSupported') notSupportedModTooltips.set(!!v);
    else if (id === 'defLevel') defaultCharLevel.set(Math.min(Math.max(Math.trunc(Number(v)) || 1, 1), 100));
    else if (id === 'gemQuality') defaultGemQuality.set(Math.min(Math.max(Math.trunc(Number(v)) || 0, 0), 23));
    // Main.lua:1100
    else if (id === 'affixQuality') defaultItemAffixQuality.set(Math.min(Math.max(Math.round(Number(v) * 100) || 0, 0), 100) / 100);
    else if (id === 'colorPositive') colorPositive.set('0x' + String(v).slice(1));
    else if (id === 'colorNegative') colorNegative.set('0x' + String(v).slice(1));
    else if (id === 'colorHighlight') colorHighlight.set('0x' + String(v).slice(1));
    else if (id === 'edgeSearchHighlight') edgeSearchHighlight.set(!!v);
  };
  const action = async (id: string) => {
    if (id === 'checkData') await checkForUpdate();
    else if (id === 'clearCache') {
      const n = (await syncWrap?.ClearDataCache()) ?? 0;
      addToast(`Cached data cleared\n${n} game data files removed; they download again on the next start.`);
      void measure();
    } else if (id === 'exportAll') {
      const files = (await syncWrap?.ExportBuilds()) ?? [];
      const zip = zipStore(files.map((f) => ({ path: /\.xml$/i.test(f.path) ? f.path : `${f.path}.xml`, data: f.data })));
      download(`go-pob-builds-${new Date().toISOString().slice(0, 10)}.zip`, zip);
      addToast(`Exported ${files.length} build${files.length === 1 ? '' : 's'}`);
    } else if (id === 'resetAll') for (const r of sections.flatMap((x) => x.rows)) set(r.id, r.value);
  };
</script>

<SettingsDialog
  bind:open
  {sections}
  {values}
  onchange={set}
  onaction={(id: string) => void action(id).catch(logError)}
  oncancel={() => {
    for (const [id, v] of Object.entries(saved)) if (values[id as keyof typeof values] !== v) set(id, v);
  }}
  onreset={(sec: string) => {
    for (const r of sections.find((s) => s.id === sec)?.rows ?? []) set(r.id, r.value);
  }} />
