<script lang="ts">
  import ImportExportPage from '../ImportExportPage.svelte';
  import type { BuildFixture } from '../fixtures';
  import { characterImport, type Character } from '../fixtures/app';
  import { EXPORT_SITES, jsonVerdict, type ImportMode } from '../share';
  import type { ComponentProps } from 'svelte';

  type Page = ComponentProps<typeof ImportExportPage>;
  type Props = { fx: BuildFixture; importText?: string; fetched?: boolean; unavailable?: Page['unavailable']; mode?: ImportMode };
  let { fx, importText = '', fetched = false, unavailable, mode = 'new' }: Props = $props();

  let s = $state({
    link: null as string | null,
    importText,
    mode,
    account: fetched ? characterImport.account : '',
    characters: fetched ? characterImport.characters : (null as Character[] | null),
    log: [] as string[],
    imported: null as string | null
  });
</script>

<ImportExportPage
  code={fx.shareCode}
  link={s.link}
  realms={characterImport.realms}
  characters={s.characters}
  {unavailable}
  bind:mode={s.mode}
  bind:importText={s.importText}
  bind:account={s.account}
  onshare={(site: string) => (s.link = `${EXPORT_SITES.find((x) => x.label === site)?.link}${fx.name}`)}
  oncopy={() => s.log.push('copied')}
  onexportparty={(on: boolean) => s.log.push(`export support: ${on}`)}
  imported={s.imported}
  onimport={(text: string, m: ImportMode) => {
    s.log.push(`import ${m}: ${text.slice(0, 24)}`);
    if (jsonVerdict(text)?.valid) s.imported = 'Items and skills successfully imported.';
  }}
  onfetch={() => {
    s.characters = characterImport.characters;
  }}
  onimporttree={(c: Character) => s.log.push(`tree of ${c.name}`)}
  onimportitems={(c: Character) => s.log.push(`items of ${c.name}`)} />
<output class="sr-only" aria-label="Import log">{s.log.join('; ')}</output>
