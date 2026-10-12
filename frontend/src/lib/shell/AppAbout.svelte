<script lang="ts">
  import AboutDialog from '$lib/design/AboutDialog.svelte';
  import type { AboutInfo } from '$lib/design/fixtures';
  import { syncWrap } from '$lib/go/worker';
  import { logError } from '$lib/utils';
  import { base } from '$app/paths';
  import { devMode } from '$lib/global.js';
  import { parseChangelog, parseHelp } from '$lib/design/help';

  // Build.lua:311-315
  type Props = { open?: boolean; tab?: string; helpSection?: string };
  let { open = $bindable(false), tab = $bindable('about'), helpSection }: Props = $props();

  // Main.lua:1359-1397
  let helpText = $state<string>();
  $effect(() => {
    if (!open || helpText !== undefined) return;
    fetch(`${base}/help.txt`)
      .then((r) => (r.ok ? r.text() : ''))
      .then((t) => (helpText = t))
      .catch(logError);
  });
  const help = $derived(helpText ? parseHelp(helpText, $devMode) : undefined);
  // Main.lua:1415-1423
  const section = $derived(
    helpSection === undefined ? undefined : (help?.find((s) => s.title.toLowerCase() === helpSection.toLowerCase()) ?? help?.[0])?.title
  );

  // Main.lua:1338-1358
  let changelogText = $state<string>();
  $effect(() => {
    if (!open || changelogText !== undefined) return;
    fetch(`${base}/changelog.txt`)
      .then((r) => (r.ok ? r.text() : ''))
      .then((t) => (changelogText = t))
      .catch(logError);
  });
  const changelog = $derived(changelogText ? parseChangelog(changelogText) : undefined);

  const list = <T,>(o: T[] | Record<string, T> | undefined): T[] =>
    Array.isArray(o) ? o : Object.entries(o ?? {}).flatMap(([k, v]) => (/^\d+$/.test(k) ? [v] : []));

  let info = $state<AboutInfo>({ version: '', go: '', gameData: '', tree: '', settings: [], deps: [] });

  $effect(() => {
    if (!open) return;
    Promise.all([syncWrap.BuildInfo(), syncWrap.GameDataVersion(), syncWrap.GetTreeVersion()])
      .then(([bi, gameData, tree]) => {
        info = {
          version: bi?.Main?.Version || '(devel)',
          go: bi?.GoVersion ?? '',
          gameData,
          tree,
          settings: list(bi?.Settings).map((s) => [s.Key, s.Value] as [string, string]),
          deps: list(bi?.Deps).flatMap((d) => (d ? [[d.Path, d.Version] as [string, string]] : []))
        };
      })
      .catch(logError);
  });
</script>

<AboutDialog
  bind:open
  bind:tab
  {info}
  {help}
  helpSection={section}
  {changelog}
  oncopy={(text: string) => void navigator.clipboard?.writeText(text).catch(logError)} />
