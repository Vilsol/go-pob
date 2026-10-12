import { existsSync, readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { resolve } from 'node:path';
import { createServer } from 'vite';
import { MOD_SAMPLES, verdictOf, configFixture, calcsFixture, compareFixture } from './design_fixtures/pages.js';
import { treeFixture } from './design_fixtures/tree.js';

const ROOT = resolve(new URL('..', import.meta.url).pathname);
const WASM = resolve(ROOT, 'static/go-pob.wasm');
const CORPUS = resolve(ROOT, '../testdata/corpus-3.29');
const OUT = resolve(ROOT, 'src/lib/design/fixtures');

const BUILDS = ['0243-necromancer-raise-spectre.xml', '0041-pathfinder-vaal-fireball.xml', '0029-inquisitor-spark-of-the-nova.xml'];
const COMPARE_WITH = {
  '0243-necromancer-raise-spectre.xml': '0008-necromancer-vaal-summon-skeletons.xml',
  '0041-pathfinder-vaal-fireball.xml': '0313-pathfinder-vaal-caustic-arrow.xml',
  '0029-inquisitor-spark-of-the-nova.xml': '0013-inquisitor-divine-ire.xml'
};

const NOTES_FROM = ['0029-inquisitor-spark-of-the-nova.xml', '1040-elementalist-summon-flame-golem-of-hordes.xml', '1147-necromancer-vaal-absolution.xml'];

const PASTE_TEXT = `Item Class: Body Armours
Rarity: Rare
Dusk Shroud
Carnal Armour
--------
+94 to maximum Life
+42% to Cold Resistance`;
const STASH_IMPACTS = 4;
const IMPACT_LINES = 4;

const say = (msg) => process.stdout.write(`${msg}\n`);
const refuse = (msg) => {
  console.error(`REFUSING: ${msg}`);
  process.exit(2);
};

const args = process.argv.slice(2);
let check = false;
for (const a of args) {
  if (a === '--check') check = true;
  else refuse(`unknown argument ${a}`);
}

if (!existsSync(WASM)) refuse(`the wasm module is missing — run \`mise run build-go\`\n  missing: ${WASM}`);
for (const b of [...BUILDS, ...Object.values(COMPARE_WITH), ...NOTES_FROM]) {
  if (!existsSync(`${CORPUS}/${b}`)) refuse(`a corpus build is missing — the corpus is git-LFS, run \`git lfs pull\`\n  missing: ${CORPUS}/${b}`);
}
const dirty = execFileSync('git', ['status', '--porcelain', '--', '../calculator', '../wasm', '../data', '../mod', '../moddb', '../builds', '../pob'], {
  cwd: ROOT,
  encoding: 'utf8'
}).trim();
if (dirty) refuse(`the Go tree is dirty, so the wasm may not match any commit:\n${dirty}`);

const server = await createServer({
  root: ROOT,
  configFile: resolve(ROOT, 'vite.config.js'),
  appType: 'custom',
  logLevel: 'error',
  server: { middlewareMode: true, hmr: false, watch: null },
  resolve: {
    alias: {
      '@zenfs/core': resolve(ROOT, 'harness/design_fixtures/zenfs_node.js'),
      '@zenfs/dom': resolve(ROOT, 'harness/design_fixtures/zenfs_dom.js')
    }
  }
});

globalThis.addEventListener ??= () => {};

const lines = (l) => (l ?? []).map((m) => ({ text: m.Line, kind: m.Kind.toLowerCase() }));

async function itemSummary(worker, text) {
  const { item, error } = await worker.ParseItem(text);
  if (!item) return { error };
  return {
    name: item.Title || item.Name,
    base: item.BaseName || item.Type,
    rarity: item.Rarity.toLowerCase(),
    sockets: item.Sockets,
    implicits: lines(item.Implicits),
    explicits: lines(item.Explicits),
    flask: item.Flask,
    fits: item.FitSlots ?? [],
    tooltip: {
      influence: [...(item.Tooltip.Influence ?? [])],
      title: [...(item.Tooltip.Title ?? [])],
      lines: (item.Tooltip.Lines ?? []).map((l) => ({ text: l.Text, sep: l.Separator }))
    }
  };
}

const impactOf = (r) => {
  const b = r?.blocks?.[0];
  if (!b) return undefined;
  const sections = b.sections.map((s) => ({ actor: s.actor, lines: s.lines.slice(0, IMPACT_LINES) }));
  return { header: b.header, slot: b.slot, sections };
};

async function itemsFixture(worker, m) {
  const raw = worker.GetItems();
  const items = {};
  for (const it of raw.Items ?? []) {
    const s = await itemSummary(worker, it.Text);
    if (s.error) continue;
    items[it.ID] = s;
  }
  const texts = Object.fromEntries((raw.Items ?? []).map((it) => [it.ID, it.Text]));
  const sets = (raw.ItemSets ?? []).map((s) => ({
    id: s.ID,
    title: s.Title || 'Default',
    swap: !!s.UseSecondWeaponSet,
    slots: (s.Slots ?? []).filter((x) => x.ItemID > 0 && items[x.ItemID]).map((x) => ({ slot: x.Name, item: x.ItemID, active: !!x.Active }))
  }));

  await worker.PrepareStatCompare();
  const impact = {};
  for (const e of await worker.GetEquippedItems()) {
    const r = e.Flask ? await worker.FlaskToggleCompare(e.Text) : await worker.ItemUnequipCompare(e.SlotName, e.SlotName);
    const d = impactOf(r);
    if (d) impact[e.SlotName] = d;
  }

  const spec = worker.currentBuild.Tree.Specs?.[worker.currentBuild.Tree.ActiveSpec - 1] ?? worker.currentBuild.Tree.Specs?.[0];
  const jewels = (spec?.Sockets ?? []).map((s) => ({ node: s.NodeID, item: items[s.ItemID] ? s.ItemID : 0 }));

  // ItemListControl.lua:242-258
  const tree = worker.currentBuild.Tree;
  const ctx = {
    sets,
    activeSet: String(raw.ActiveItemSet),
    specs: (tree.Specs ?? []).map((sp) => ({
      title: sp.Title ?? 'Default',
      sockets: (sp.Sockets ?? []).map((x) => ({ node: x.NodeID, item: x.ItemID })),
      alloc: new Set((sp.NodesAttr.match(/\d+/g) ?? []).map(Number))
    })),
    activeSpec: Math.max(0, (tree.ActiveSpec ?? 1) - 1)
  };
  const stash = [];
  let compared = 0;
  for (const id of Object.keys(items).map(Number)) {
    const usage = m.usage.itemUsage(id, { jewel: /Jewel$/.test(items[id].base), abyss: /Abyss/.test(items[id].base) }, ctx);
    if (usage === '') continue;
    const row = { item: id, used: usage };
    if (items[id].fits.length && compared++ < STASH_IMPACTS) row.impact = impactOf(await worker.ItemCompare(texts[id]));
    stash.push(row);
  }
  const paste = { text: PASTE_TEXT, item: await itemSummary(worker, PASTE_TEXT), impact: impactOf(await worker.ItemCompare(PASTE_TEXT)) };
  return { activeSet: String(raw.ActiveItemSet), items, sets, impact, jewels, stash, paste };
}

const ATTR = { STR: 'str', DEX: 'dex', INT: 'int', NONE: 'white' };

function skillsFixture(build, gemById, socketsBySlot) {
  const skills = build.Skills;
  return {
    activeSet: build.ActiveSkillSetIndex(),
    sets: (skills?.SkillSets ?? []).map((set, i) => ({
      title: set.Title || `Skill set ${i + 1}`,
      groups: (set.Skills ?? []).map((g) => ({
        label: g.DisplayLabel || g.Label || '<No active skills>',
        slot: g.Slot || '',
        enabled: g.Enabled,
        inFull: g.IncludeInFullDPS ?? false,
        count: g.GroupCount ?? 1,
        sockets: socketsBySlot[g.Slot] ?? '',
        gems: (g.Gems ?? []).map((m) => {
          const def = gemById.get(m.GemID);
          return {
            name: def?.name || m.NameSpec || m.SkillID,
            level: m.Level,
            quality: m.Quality,
            enabled: m.Enabled,
            support: def?.support ?? false,
            attr: def?.attr ?? 'white',
            count: m.Count ?? 1
          };
        })
      }))
    }))
  };
}

let exitCode = 0;
try {
  const { workerInstance } = await server.ssrLoadModule('/src/lib/go/sync_worker.ts');
  const types = await server.ssrLoadModule('/src/lib/types/index.js');
  const { buildSidebar } = await server.ssrLoadModule('/src/lib/display/sidebar.ts');
  const { estimatePlayerProgress } = await server.ssrLoadModule('/src/lib/display/points.ts');
  const { cacheStats } = await server.ssrLoadModule('/harness/design_fixtures/zenfs_node.js');
  const m = {
    cfg: await server.ssrLoadModule('/src/lib/display/configurations.ts'),
    custom: await server.ssrLoadModule('/src/lib/display/custom_mods.ts'),
    sections: await server.ssrLoadModule('/src/lib/calcs/calc_sections.ts'),
    calcs: await server.ssrLoadModule('/src/lib/display/calcs.ts'),
    stats: await server.ssrLoadModule('/src/lib/display/stats.ts'),
    fmtStat: await server.ssrLoadModule('/src/lib/display/format_stat.ts'),
    geo: await server.ssrLoadModule('/src/lib/skill_tree/geometry.ts'),
    paths: await server.ssrLoadModule('/src/lib/components/skill-tree/paths.ts'),
    usage: await server.ssrLoadModule('/src/lib/design/item_usage.ts')
  };

  const quiet = console.log;
  console.log = () => {};

  let last;
  const wasm = readFileSync(WASM);
  await workerInstance.boot(wasm.buffer.slice(wasm.byteOffset, wasm.byteOffset + wasm.byteLength), (out) => (last = out), { set: () => {}, touch: () => {} });
  await workerInstance.loadData(async () => {});
  console.log = quiet;
  say(`game data: ${cacheStats.hits} cached, ${cacheStats.misses} fetched`);

  mkdirSync(OUT, { recursive: true });
  const gemList = workerInstance
    .GetSkillGems()
    .map((g) => ({
      id: g.ID,
      name: g.Base.Name,
      attr: ATTR[g.GemType] ?? 'white',
      support: g.Support,
      searchName: g.Name,
      tags: g.Tags ?? [],
      levelReq: g.LevelRequirement
    }))
    .filter((g) => g.name)
    .sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
  const gemById = new Map(gemList.map((g) => [g.id, g]));
  const outputs = [
    [`${OUT}/gems.json`, JSON.stringify(gemList.map((g) => [g.name, g.attr, g.support ? 1 : 0, g.searchName, g.tags.join(' '), g.levelReq])) + '\n']
  ];
  const sampleVerdicts = workerInstance.ParseModLines(MOD_SAMPLES).map((l, i) => {
    const v = verdictOf(m, l);
    return [MOD_SAMPLES[i], v.k, v.parsed];
  });
  outputs.push([`${OUT}/mods.json`, JSON.stringify(sampleVerdicts, null, 2) + '\n']);
  const bi = workerInstance.BuildInfo();
  const list = (o) =>
    Object.entries(o ?? {})
      .filter(([k]) => /^\d+$/.test(k))
      .map(([, v]) => v);
  const about = {
    version: (bi.Main?.Version || '(devel)').split('-')[0],
    go: bi.GoVersion,
    gameData: workerInstance.GameDataVersion(),
    tree: workerInstance.GetTreeVersion(),
    settings: list(bi.Settings)
      .filter((x) => !x.Key.startsWith('vcs'))
      .map((x) => [x.Key, x.Value]),
    deps: list(bi.Deps).map((d) => [d.Path, d.Version])
  };
  outputs.push([`${OUT}/about.json`, JSON.stringify(about, null, 2) + '\n']);
  const notes = NOTES_FROM.map((file) => {
    const b = types.builds.ParseBuildStr(readFileSync(`${CORPUS}/${file}`, 'utf8')).unwrap();
    if (!b?.Notes?.trim()) refuse(`${file} has no notes`);
    return { source: file.replace(/\.xml$/, ''), text: b.Notes };
  });
  outputs.push([`${OUT}/notes.json`, JSON.stringify(notes, null, 2) + '\n']);
  for (const file of BUILDS) {
    const xml = readFileSync(`${CORPUS}/${file}`, 'utf8');
    const build = types.builds.ParseBuildStr(xml).unwrap();
    if (!build) refuse(`ParseBuildStr returned nothing for ${file}`);

    last = undefined;
    console.log = () => {};
    workerInstance.currentBuild = build;
    await workerInstance.Tick('design-fixtures');
    const tick = await workerInstance.GetSidebarWarnings();
    console.log = quiet;
    const out = last;
    if (!out?.Output) refuse(`the calc produced no outputs for ${file}`);
    if (!tick) refuse(`the sidebar warnings did not arrive for ${file}`);

    const info = workerInstance.GetBuildInfo();
    const skillSet = build.ActiveSkillSetIndex();
    const progress = out.PointsUsed ? estimatePlayerProgress(out.PointsUsed, out.Output?.ExtraPoints) : undefined;
    const fixture = {
      source: `testdata/corpus-3.29/${file}`,
      name: file.replace(/\.xml$/, ''),
      className: info.className,
      ascendClassName: info.ascendClassName,
      level: info.level,
      mainSocketGroup: info.mainSocketGroup - 1,
      socketGroups: workerInstance.GetSocketGroupSummaries(skillSet).map((g) => g.label),
      mainSkill: out.MainSkill,
      progress,
      sidebar: buildSidebar(out, tick),
      shareCode: workerInstance.ExportCode()
    };
    console.log = () => {};
    fixture.items = await itemsFixture(workerInstance, m);
    fixture.tree = await treeFixture(m, workerInstance, fixture.items);
    console.log = quiet;
    const set = fixture.items.sets.find((s) => s.id === fixture.items.activeSet) ?? fixture.items.sets[0];
    const socketsBySlot = Object.fromEntries((set?.slots ?? []).map((x) => [x.slot, fixture.items.items[x.item].sockets]));
    fixture.skills = skillsFixture(build, gemById, socketsBySlot);

    const getLast = () => last;
    console.log = () => {};
    fixture.config = configFixture(m, workerInstance, out);
    fixture.calcs = await calcsFixture(m, workerInstance, getLast);
    const bFile = COMPARE_WITH[file];
    const bBuild = types.builds.ParseBuildStr(readFileSync(`${CORPUS}/${bFile}`, 'utf8')).unwrap();
    if (!bBuild) refuse(`ParseBuildStr returned nothing for ${bFile}`);
    fixture.compare = await compareFixture(
      m,
      workerInstance,
      getLast,
      {
        file,
        build,
        items: fixture.items,
        inputs: fixture.config.inputs,
        texts: Object.fromEntries((workerInstance.GetItems().Items ?? []).map((it) => [it.ID, it.Text]))
      },
      { file: bFile, build: bBuild },
      { itemsOf: (w) => itemsFixture(w, m), skillsOf: (b) => skillsFixture(b, gemById, {}) }
    );
    console.log = quiet;
    const text = JSON.stringify(fixture, (k, v) => (k.startsWith('__') ? undefined : v), 2) + '\n';
    const path = `${OUT}/${fixture.name}.json`;
    const lines = fixture.sidebar.sections.reduce((n, s) => n + s.groups.flat().length, 0);
    if (lines === 0) refuse(`${file}: the sidebar came out empty`);
    if (Object.keys(fixture.items.items).length === 0) refuse(`${file}: no item parsed`);
    if (!fixture.skills.sets.some((s) => s.groups.length)) refuse(`${file}: no socket group`);
    if (!fixture.calcs.sections.some((s) => s.blocks?.length)) refuse(`${file}: no calcs section`);
    if (!fixture.compare.b.stats.length) refuse(`${file}: build B has no stats`);
    outputs.push([path, text, `${lines} stat lines, ${fixture.sidebar.warnings.length} warnings, ${Object.keys(fixture.items.items).length} items`]);
  }
  for (const [path, text, what] of outputs) {
    if (check) {
      const same = existsSync(path) && readFileSync(path, 'utf8') === text;
      if (!same) exitCode = 1;
      say(`${same ? 'same' : 'DIFFERS'}  ${path}`);
    } else {
      writeFileSync(path, text);
      say(`wrote ${path}${what ? ` (${what})` : ''}`);
    }
  }
} finally {
  await server.close();
}
process.exit(exitCode);
