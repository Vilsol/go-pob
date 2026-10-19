<script lang="ts">
  import LiveMobile from './LiveMobile.svelte';
  import LivePhoneTree from './LivePhoneTree.svelte';
  import LivePhoneSkills from './LivePhoneSkills.svelte';
  import LiveItems from './LiveItems.svelte';
  import LiveCalcs from './LiveCalcs.svelte';
  import LiveConfig from './LiveConfig.svelte';
  import LiveNotes from './LiveNotes.svelte';
  import LiveParty from './LiveParty.svelte';
  import LiveCompare from './LiveCompare.svelte';
  import LiveImportExport from './LiveImportExport.svelte';
  import LiveTimeless from './LiveTimeless.svelte';
  import { notesSamples, type BuildFixture } from '../fixtures';
  import { PHONE_PAGES, moreTabs, primaryTabs } from '../mobile/mobile';

  type Props = { fx: BuildFixture; current?: string };
  let { fx, current: start = '#tree' }: Props = $props();
  let current = $state(start);

  const page = $derived(PHONE_PAGES[[...primaryTabs, ...moreTabs].find((t) => t.href === current)?.label ?? 'Tree']);
</script>

<LiveMobile {fx} current={start} label={page.label} mainClass={page.main} onnavigate={(href: string) => (current = href)}>
  {#key current}
    {#if current === '#tree'}<LivePhoneTree {fx} />
    {:else if current === '#skills'}<LivePhoneSkills {fx} />
    {:else if current === '#items'}<LiveItems {fx} layout="list" />
    {:else if current === '#calcs'}<LiveCalcs {fx} />
    {:else if current === '#config'}<LiveConfig {fx} />
    {:else if current === '#notes'}<LiveNotes text={notesSamples[1].text} />
    {:else if current === '#party'}<LiveParty {fx} />
    {:else if current === '#compare'}<LiveCompare {fx} />
    {:else if current === '#import'}<LiveImportExport {fx} />
    {:else}<LiveTimeless {fx} />
    {/if}
  {/key}
</LiveMobile>
