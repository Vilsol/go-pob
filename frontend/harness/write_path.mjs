import { readFile } from 'node:fs/promises';
import { existsSync } from 'node:fs';
import { MessageChannel } from 'node:worker_threads';
import * as Comlink from 'comlink';
import { execFileSync } from 'node:child_process';

const HERE = new URL('.', import.meta.url).pathname;
const WASM = `${HERE}../static/go-pob.wasm`;
const TYPES = `${HERE}../src/lib/types/index.js`;
const CORPUS = `${HERE}../../testdata/corpus-3.29`;

const refuse = (msg) => {
  console.error(`REFUSING: ${msg}`);
  process.exit(2);
};

for (const [path, what] of [
  [WASM, 'the wasm module — run `mise run build-go` (or use `task write-path`)'],
  [TYPES, 'the generated bindings — run `mise run generate`'],
  [CORPUS, 'the 3.29 corpus — it is git-LFS, run `git lfs pull`']
]) {
  if (!existsSync(path)) refuse(`${what}\n  missing: ${path}`);
}

let goroot;
try {
  goroot = execFileSync('go', ['env', 'GOROOT'], { encoding: 'utf8' }).trim();
} catch (e) {
  refuse(`could not run \`go env GOROOT\` to locate wasm_exec.js: ${String(e.message)}`);
}
const execPath = `${goroot}/lib/wasm/wasm_exec.js`;
if (!existsSync(execPath)) refuse(`wasm_exec.js not found at ${execPath} (Go ${goroot})`);
await import(`file://${execPath}`);

globalThis.performance ??= (await import('node:perf_hooks')).performance;
globalThis.crypto ??= (await import('node:crypto')).webcrypto;

const wasm = await readFile(WASM);
const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(wasm, go.importObject);
void go.run(instance);
await new Promise((r) => setTimeout(r, 250));

const registered = Object.keys(globalThis.go?.['go-pob'] ?? {});
if (registered.length === 0) refuse('the wasm registered no namespaces at all');

const mod = await import(TYPES);
try {
  mod.initializeCrystalline();
} catch (e) {
  refuse(
    `initializeCrystalline failed — go-pob.wasm is probably STALE relative to the bindings.\n` +
      `  it registered: ${registered.join(', ')}\n` +
      `  run \`mise run build-go\`\n  original: ${String(e.message)}`
  );
}

const { builds, pob } = mod;

let failures = 0;
let checked = 0;

const check = (name, ok, detail) => {
  checked++;
  if (ok) {
    console.log(`  PASS  ${name}${detail ? `  (${detail})` : ''}`);
  } else {
    failures++;
    console.log(`  FAIL  ${name}  ${detail ?? ''}`);
  }
};

console.log('\n=== round-trip stability (no mutation) ===');

const buildFiles = ['0001-necromancer-unknown.xml', '0008-necromancer-vaal-summon-skeletons.xml', '0028-occultist-forbidden-rite-of-soul-sacrifice.xml'];

for (const file of buildFiles) {
  const xml = await readFile(`${CORPUS}/${file}`, 'utf8');
  const parsed = builds.ParseBuildStr(xml);
  const build = parsed.unwrap ? parsed.unwrap() : parsed;
  const out = builds.SerializeBuild(build);
  const bytes = out.unwrap ? out.unwrap() : out;
  const serialized = new TextDecoder().decode(bytes);

  const attrs = ['defaultGemLevel', 'defaultGemQuality', 'showSupportGemTypes', 'sortGemsByDPS', 'sortGemsByDPSField'];
  for (const a of attrs) {
    const before = xml.match(new RegExp(`${a}="([^"]*)"`))?.[1];
    const after = serialized.match(new RegExp(`${a}="([^"]*)"`))?.[1];
    check(`${file} ${a}`, before === after, `before=${JSON.stringify(before)} after=${JSON.stringify(after)}`);
  }
}

console.log('\n=== mutations reach Go, through real Comlink ===');

const xml = await readFile(`${CORPUS}/0001-necromancer-unknown.xml`, 'utf8');
const p = builds.ParseBuildStr(xml);
const build = p.unwrap ? p.unwrap() : p;

const { port1, port2 } = new MessageChannel();
port1.unref();
port2.unref();
Comlink.expose({ currentBuild: Comlink.proxy(build) }, port2);
const remote = Comlink.wrap(port1);
const cb = remote.currentBuild;

const settle = () => new Promise((r) => setTimeout(r, 150));

await cb.SetNotes('HARNESS-CONTROL');
await settle();
const controlLanded = build.Notes === 'HARNESS-CONTROL';
check('CONTROL: SetNotes reaches Go', controlLanded, `inGo=${JSON.stringify(build.Notes)}`);
if (!controlLanded) refuse('the control write did not land — the harness itself is broken, not the code under test');

const cases = [
  {
    name: 'SetRichNotes -> Notes + RichNotes (hash of the Notes written)',
    apply: () => cb.SetRichNotes('^xB97123fire\nplain', '{"type":"doc","content":[]}'),
    read: () => `${build.Notes}|${build.RichNotesStatus()}|${build.RichNotesDoc()}`,
    want: '^xB97123fire\nplain|match|{"type":"doc","content":[]}'
  },
  {
    name: 'SetSkillGroupName -> Skills.SkillSets[0].Skills[0].Label',
    apply: () => cb.SetSkillGroupName(0, 0, 'HARNESS-LABEL'),
    read: () => build.Skills?.SkillSets?.[0]?.Skills?.[0]?.Label,
    want: 'HARNESS-LABEL'
  },
  {
    name: 'slice-element write -> Skills.SkillSets[0].Skills[0].Label',
    apply: async () => {
      cb.Skills.SkillSets[0].Skills[0].Label = 'HARNESS-DIRECT';
    },
    read: () => build.Skills?.SkillSets?.[0]?.Skills?.[0]?.Label,
    want: 'HARNESS-DIRECT'
  },
  {
    name: 'SetDefaultGemLevel -> Skills.DefaultGemLevel',
    apply: () => cb.SetDefaultGemLevel(17),
    read: () => build.Skills?.DefaultGemLevel,
    want: '17'
  },
  {
    name: 'slice-element write -> Skills.SkillSets[0].Skills[0].Slot',
    apply: async () => {
      cb.Skills.SkillSets[0].Skills[0].Slot = 'Weapon 1';
    },
    read: () => build.Skills?.SkillSets?.[0]?.Skills?.[0]?.Slot,
    want: 'Weapon 1'
  },
  {
    name: 'slice-element write -> Skills.SkillSets[0].Skills[0].Enabled',
    apply: async () => {
      cb.Skills.SkillSets[0].Skills[0].Enabled = false;
    },
    read: () => build.Skills?.SkillSets?.[0]?.Skills?.[0]?.Enabled,
    want: false
  },
  {
    name: 'slice-element write -> Skills.SkillSets[0].Skills[0].IncludeInFullDPS',
    apply: async () => {
      cb.Skills.SkillSets[0].Skills[0].IncludeInFullDPS = true;
    },
    read: () => build.Skills?.SkillSets?.[0]?.Skills?.[0]?.IncludeInFullDPS,
    want: true
  }
];

const MS_GROUP = (build.Skills?.SkillSets?.[0]?.Skills ?? []).findIndex((g) => (g.Gems?.length ?? 0) > 0);
if (MS_GROUP < 0) refuse('0001 has no socket group with a gem to address');
const msGroup = () => build.Skills?.SkillSets?.[0]?.Skills?.[MS_GROUP];
const msGem = () => msGroup()?.Gems?.[0];
cases.push(
  { name: 'SetMainActiveSkill -> Skill.MainActiveSkill', apply: () => cb.SetMainActiveSkill(0, MS_GROUP, 2), read: () => msGroup()?.MainActiveSkill, want: 2 },
  { name: 'SetGemSkillPart -> Gem.SkillPart', apply: () => cb.SetGemSkillPart(0, MS_GROUP, 0, 2), read: () => msGem()?.SkillPart, want: 2 },
  {
    name: 'SetGemSkillStageCount -> Gem.SkillStageCount',
    apply: () => cb.SetGemSkillStageCount(0, MS_GROUP, 0, 3),
    read: () => msGem()?.SkillStageCount,
    want: 3
  },
  { name: 'SetGemSkillMineCount -> Gem.SkillMineCount', apply: () => cb.SetGemSkillMineCount(0, MS_GROUP, 0, 4), read: () => msGem()?.SkillMineCount, want: 4 },
  {
    name: 'SetGemSkillMinion -> Gem.SkillMinion',
    apply: () => cb.SetGemSkillMinion(0, MS_GROUP, 0, 'RaisedZombie'),
    read: () => msGem()?.SkillMinion,
    want: 'RaisedZombie'
  },
  {
    name: 'SetGemSkillMinionItemSet -> Gem.SkillMinionItemSet',
    apply: () => cb.SetGemSkillMinionItemSet(0, MS_GROUP, 0, 2),
    read: () => msGem()?.SkillMinionItemSet,
    want: 2
  },
  {
    name: 'SetGemSkillMinionSkill -> Gem.SkillMinionSkill',
    apply: () => cb.SetGemSkillMinionSkill(0, MS_GROUP, 0, 2),
    read: () => msGem()?.SkillMinionSkill,
    want: 2
  }
);

cases.push(
  {
    name: 'SetCustomModifierBlocks -> Config.CustomModifierBlocks',
    apply: () => cb.SetCustomModifierBlocks([{ Title: 'HARNESS', Enabled: 'false', Text: '+1 to Strength\nHARNESS-LINE' }]),
    read: () => JSON.stringify(build.GetCustomModifierBlocks().map((b) => [b.Title, b.Enabled, b.Text])),
    want: JSON.stringify([['HARNESS', 'false', '+1 to Strength\nHARNESS-LINE']])
  },
  {
    name: 'SetConfigOption -> Config.Inputs (control for the reset below)',
    apply: () => cb.SetConfigOption({ Name: 'conditionOnslaught', Boolean: true }),
    read: () => build.GetBooleanOption('conditionOnslaught'),
    want: true
  },
  {
    name: 'ResetConfigOptions -> the option input is gone',
    apply: () => cb.ResetConfigOptions(),
    read: () => build.GetBooleanOption('conditionOnslaught'),
    want: false
  }
);

const freshItems = () => {
  const c = mod.calculator.NewCalculator(build);
  try {
    return c.PoB.Items;
  } finally {
    c.release();
  }
};
const activeSlot = (name) => {
  const items = freshItems();
  const set = items.ItemSets.find((s) => s.ID === String(items.ActiveItemSet)) ?? items.ItemSets[0];
  return set?.Slots?.find((s) => s.Name === name);
};
const ITEM_TEXT = 'Rarity: Rare\nHarness Crown\nHubris Circlet\n+10 to Strength & <more>';
let itemId = 0;
cases.push(
  {
    name: 'AddItem -> Items.Items[].Text (escaped inner XML)',
    apply: async () => {
      itemId = await cb.AddItem(ITEM_TEXT);
    },
    read: () => freshItems().Items.find((i) => i.ID === itemId)?.Text,
    want: 'Rarity: Rare\nHarness Crown\nHubris Circlet\n+10 to Strength &amp; &lt;more&gt;'
  },
  {
    name: 'SetSlotItem -> active ItemSet Slot "Helmet".ItemID',
    apply: () => cb.SetSlotItem('Helmet', itemId),
    read: () => activeSlot('Helmet')?.ItemID === itemId && itemId > 0,
    want: true
  },
  {
    name: 'SetFlaskActive -> active ItemSet Slot "Flask 3".Active',
    apply: () => cb.SetFlaskActive('Flask 3', true),
    read: () => activeSlot('Flask 3')?.Active,
    want: true
  },
  { name: 'SetSlotItem 0 -> active ItemSet Slot "Gloves" emptied', apply: () => cb.SetSlotItem('Gloves', 0), read: () => activeSlot('Gloves')?.ItemID, want: 0 }
);

for (const c of cases) {
  await c.apply();
  await settle();
  const got = c.read();
  check(c.name, got === c.want, `inGo=${JSON.stringify(got)} want=${JSON.stringify(c.want)}`);
}

{
  const saved = new TextDecoder().decode(builds.SerializeBuild(build).unwrap());
  const reparsed = builds.ParseBuildStr(saved).unwrap();
  const c = mod.calculator.NewCalculator(reparsed);
  const text = c.PoB.Items.Items.find((i) => i.ID === itemId)?.Text;
  c.release();
  check(
    'an added and equipped item survives a save',
    saved.includes(`<Slot itemId="${itemId}" name="Helmet"`) && text === 'Rarity: Rare\nHarness Crown\nHubris Circlet\n+10 to Strength &amp; &lt;more&gt;',
    `text=${JSON.stringify(text)}`
  );
}

{
  const saved = new TextDecoder().decode(builds.SerializeBuild(build).unwrap());
  const gemXml = saved.match(/<Skill [^>]*mainActiveSkill="2"[^>]*>\s*<Gem [^>]*>/)?.[0] ?? '';
  for (const attr of [
    'skillPart="2"',
    'skillStageCount="3"',
    'skillMineCount="4"',
    'skillMinion="RaisedZombie"',
    'skillMinionItemSet="2"',
    'skillMinionSkill="2"'
  ]) {
    check(
      `main-skill write saved as ${attr}`,
      gemXml.includes(attr),
      gemXml ? gemXml.slice(0, 200) : 'no <Skill mainActiveSkill="2"> with a <Gem> in the saved XML'
    );
  }
}

console.log('\n=== worker-side flat summaries (B2) ===');

{
  const groups = build.Skills?.SkillSets?.[0]?.Skills;
  check('a local wrapper exposes the socket groups', Array.isArray(groups) || typeof groups?.length === 'number', `len=${groups?.length}`);

  const rows = Array.from(groups ?? []).map((g) => ({
    label: g.DisplayLabel || g.Label || '<No active skills>',
    enabled: g.Enabled,
    fullDPS: g.IncludeInFullDPS ?? false
  }));

  check('every group maps to a flat row', rows.length === groups?.length, `rows=${rows.length} groups=${groups?.length}`);
  check(
    'every row has a non-empty label',
    rows.every((r) => typeof r.label === 'string' && r.label.length > 0),
    JSON.stringify(rows.map((r) => r.label).slice(0, 3))
  );
  check(
    'every row has a boolean enabled',
    rows.every((r) => typeof r.enabled === 'boolean')
  );
  let cloneRefused = false;
  try {
    structuredClone(groups);
  } catch {
    cloneRefused = true;
  }
  check('...and the same value does NOT cross a structuredClone', cloneRefused, 'this is why the summary is computed worker-side');
}

console.log('\n=== bind.Plain() subtrees cross as data, and Skills deliberately does not ===');

{
  for (const field of ['Build', 'Tree', 'Calcs', 'Items', 'TreeView', 'Config']) {
    const v = build[field];
    check(`${field} is present`, v !== undefined && v !== null, `typeof=${typeof v}`);

    let cloned = false;
    let why = '';
    try {
      structuredClone(v);
      cloned = true;
    } catch (e) {
      why = String(e?.message ?? e);
    }
    check(`${field} crosses a structuredClone (bind.Plain)`, cloned, cloned ? 'plain data' : `refused: ${why}`);
  }

  let skillsCloned = false;
  try {
    structuredClone(build.Skills);
    skillsCloned = true;
  } catch {
    /* empty */
  }
  check(
    'CONTROL: Skills does NOT cross, because it is still LIVE and still writable',
    !skillsCloned,
    'if this fails, the four socket-group writes above have lost their write path'
  );
}

console.log(`\n${checked} checks, ${failures} failed`);
if (checked === 0) refuse('zero checks ran — a harness that checks nothing must not exit 0');
process.exit(failures === 0 ? 0 : 1);
