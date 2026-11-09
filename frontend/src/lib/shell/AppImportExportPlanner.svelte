<script lang="ts">
  // ImportTab.lua:354-476 (PoB-PoE2): Export to in-game build planner; the browser saves the .build file where the
  // viewer picks, so the save path, Open Folder and the overwrite prompt are left to it
  import { poe2Build } from '$lib/global.js';
  import { poe2Worker } from '$lib/go/engine_poe2.svelte';
  import { logError } from '$lib/utils';
  import Button from '$lib/design/Button.svelte';
  import Checkbox from '$lib/design/Checkbox.svelte';
  import Field from '$lib/design/Field.svelte';
  import InlineMessage from '$lib/design/InlineMessage.svelte';
  import Input from '$lib/design/Input.svelte';
  import SectionHeading from '$lib/design/SectionHeading.svelte';
  import Select from '$lib/design/Select.svelte';
  import Textarea from '$lib/design/Textarea.svelte';
  import type { exposition } from '$lib/types2';
  import { BUILD_PLANNER_NAME, BUILD_PLANNER_AUTHOR, saveBuildFile } from './import_export';

  let buildName = $state('');
  let author = $state('');
  let description = $state('');
  let useGeneratedItemText = $state(true);
  let sets = $state<exposition.BuildPlannerSets>();
  let spec = $state('1');
  let skillSet = $state('1');
  let itemSet = $state('1');
  let error = $state<string>();
  let saved = $state<string>();

  // ImportTab.lua:494-531: every rebuild of the lists selects the first of each
  $effect(() => {
    void $poe2Build;
    poe2Worker()
      .then((w) => w.BuildPlannerSets())
      .then((s) => {
        sets = s;
        spec = String(s.Trees?.[0]?.Key ?? 1);
        skillSet = String(s.SkillSets?.[0]?.Key ?? 1);
        itemSet = String(s.ItemSets?.[0]?.Key ?? 1);
      })
      .catch(logError);
  });
  const options = (list?: exposition.BuildPlannerSet[]) => (list ?? []).map((s) => ({ value: String(s.Key), label: s.Label }));

  // ImportTab.lua:413-435, 484-491
  const onexport = () => {
    error = undefined;
    saved = undefined;
    const meta = {
      Name: buildName || BUILD_PLANNER_NAME,
      Author: author || BUILD_PLANNER_AUTHOR,
      Description: description,
      UseGeneratedItemText: useGeneratedItemText
    };
    poe2Worker()
      .then((w) => w.BuildPlannerExport(meta, Number(spec), Number(skillSet), Number(itemSet)))
      .then((file) => {
        saveBuildFile(file.FileName, file.JSON);
        saved = file.FileName;
      })
      .catch((err: unknown) => (error = err instanceof Error ? err.message : String(err)));
  };
</script>

<section aria-label="Build planner export" class="panel flex flex-col gap-2.5 p-3.5">
  <SectionHeading size="lg">EXPORT TO IN-GAME BUILD PLANNER</SectionHeading>
  <p class="m-0 text-13">Save this build as a .build file the in-game build planner can load.</p>
  <p class="note m-0">Each file holds one passive tree, one skill set and one item set.</p>
  <div class="flex flex-wrap gap-2.5">
    <Field label="Build name"><Input size="md" class="w-55" bind:value={buildName} placeholder={BUILD_PLANNER_NAME} /></Field>
    <Field label="Author name"><Input size="md" class="w-55" bind:value={author} placeholder={BUILD_PLANNER_AUTHOR} /></Field>
  </div>
  <div class="flex flex-wrap gap-2.5">
    <Field label="Tree"><Select size="md" bind:value={spec} options={options(sets?.Trees)} title="Which passive tree to export" /></Field>
    <Field label="Skill"><Select size="md" bind:value={skillSet} options={options(sets?.SkillSets)} title="Which skill set to export" /></Field>
    <Field label="Item"><Select size="md" bind:value={itemSet} options={options(sets?.ItemSets)} title="Which item set to export" /></Field>
  </div>
  <Checkbox
    compact
    bind:checked={useGeneratedItemText}
    title="For items without a note, generate a note that contains the current mods, item name and item base">Generated text for empty item notes</Checkbox>
  <Field label="Description"><Textarea class="h-16 p-2 text-13" bind:value={description} /></Field>
  <Button
    size="md"
    variant="primary"
    class="self-start"
    disabled={!sets}
    title="Writes a single file containing the three sets picked above."
    onclick={onexport}>Export Selected Sets</Button>
  {#if error}<InlineMessage kind="error">Couldn't save the build file: {error}</InlineMessage>{/if}
  {#if saved}<InlineMessage kind="success">Build file exported: {saved}</InlineMessage>{/if}
</section>
