import { expose, proxy } from 'comlink';
import '$lib/console_hook';
import '../../wasm_exec.js';
import { initializeCrystalline, storage, raw, config, pob, builds, calculator, exposition } from '../types';
import type { Outputs, TickCallback, TickMeta, TickWarnings } from '../custom_types';
import type { BuildChannel } from '../global.js';
import type { PartyMemberView, PartyView } from './party_view';
import { dump, releaseWrapper, type ProxiedRemote } from '../type_utils';
import { epochNow } from '../utils';
import { configRelevanceQuery, reverseConfigOptions } from '../display/configurations';
import { autoLevel, estimatePlayerProgress } from '../display/points';
import { savedStats } from '../display/saved_stats';
import type { CalcDataColProp } from '$lib/calcs/calc_sections';
import { breakdownOf, colPropsOf } from '$lib/display/calcs';
import type { SidebarCell, SidebarSection, SidebarStatRef } from '$lib/display/sidebar_breakdown';
import { addStatCompares, type StatCompareSection } from '$lib/skill_tree/stat_compare';
import type { ItemStatDiffBlock } from '$lib/components/items/item_diff';
import type { CraftChoice, DisplayEdit, DisplayInfo } from '$lib/design/items';
import type { AffixControls, AnointNotable, CorruptChoices, CrucibleChoices, CustomModChoices, EnchantChoices, ImplicitChoices } from '$lib/design/crafting';
import type { ClassChangeResult } from '$lib/skill_tree/class_change';
import type { TreeClusterNode } from '$lib/skill_tree/types';
import { NOT_A_CHANGE, OPEN_SOURCES, UNRECORDED, UndoHistory, stepLabel, unrecordedLabel, type Snapshot, type UndoView } from './undo';
import { TickCache, type TickCacheEntry } from './tick_cache';
import type { CompareCalcsInputs, CompareInfo, CompareItem, CompareJewel, CompareSetKind, CompareSets, CompareSideData } from '$lib/shell/compare_types';
import type { MainSkillField } from '$lib/shell/main_skill';
import { modRowsEqual } from '$lib/display/compare_mods';
import type { ItemDBFilterInput, ItemDBLists, ItemDBRowPlain } from '$lib/design/item_db';
import { configureStorage } from './storage_config';
import { initializeStorage, storageFs } from './storage_init';
import { PowerPool, type PowerPoolRun } from './power_pool';
import { noteCall, whenIdle } from './idle';

const storageConfigurationPromise = configureStorage();

type WarningsEnv = { out: calculator.Environment; calc: calculator.Calculator };

const WARNINGS_QUIET_MS = 300;
const NO_TIP_STATS: calculator.ItemTooltipStats = { Str: 0, Dex: 0, Int: 0, Omni: 0, OmniscienceRequirements: false, OmniAttributeRequirements: 0 };
const WARNINGS_STEP = 'pob:warnings-step';

const yieldToQueue = () =>
  new Promise<void>((resolve) => {
    const { port1, port2 } = new MessageChannel();
    port1.onmessage = () => {
      port1.close();
      resolve();
    };
    port2.postMessage(undefined);
  });

class PoBWorker {
  private _currentBuild?: pob.PathOfBuilding;

  get currentBuild(): pob.PathOfBuilding | undefined {
    return this._currentBuild;
  }

  set currentBuild(value: pob.PathOfBuilding | undefined) {
    this._currentBuild = value;
    this.dropMiscCalculator();
    this.updateStore();
  }

  private openBuild(build: pob.PathOfBuilding | undefined, source: string) {
    this.ClearCompare();
    this.currentBuild = build;
    this.lastOutputs = undefined;
    this.history.clear();
    this.forgetTicks();
    void this.Tick(source);
  }

  booted = false;

  private wasmModule?: WebAssembly.Module;

  callback?: TickCallback;
  currentBuildStore?: BuildChannel;

  private publishedBuild?: pob.PathOfBuilding;

  private updateStore() {
    if (!this.currentBuildStore) {
      return;
    }

    if (!this._currentBuild) {
      this.publishedBuild = undefined;
      this.currentBuildStore.set(undefined);
      return;
    }

    if (this._currentBuild === this.publishedBuild) {
      this.currentBuildStore.touch();
      return;
    }

    this.publishedBuild = this._currentBuild;
    this.currentBuildStore.set(proxy(this._currentBuild) as unknown as ProxiedRemote<pob.PathOfBuilding>);
  }

  boot(wasm: ArrayBuffer, callback: TickCallback, currentBuildStore: BuildChannel) {
    this.callback = callback;
    this.booted = true;
    this.currentBuildStore = currentBuildStore;

    return new Promise((resolve) => {
      const go = new Go();
      performance.mark('pob:instantiate-start');
      void WebAssembly.instantiate(wasm, go.importObject).then(async (result) => {
        this.wasmModule = result.module;
        performance.mark('pob:instantiated');
        void go.run(result.instance);
        performance.mark('pob:go-main');
        (globalThis as { __pobWasmMemory?: unknown }).__pobWasmMemory = result.instance.exports.mem;

        initializeCrystalline();
        (globalThis as { __pobLiveHeapMB?: unknown }).__pobLiveHeapMB = () => exposition.LiveHeapMB();

        config.InitLogging(false);

        await storageConfigurationPromise;

        await initializeStorage();

        performance.mark('pob:storage-ready');
        resolve(undefined);
      });
    });
  }

  private dataReady?: Promise<void>;

  loadData(updates: (data: string) => Promise<void>) {
    this.dataReady ??= this.initializeData(updates);
    return this.dataReady;
  }

  private async initializeData(updates: (data: string) => Promise<void>) {
    const start = Date.now();
    const result = await raw.InitializeAll(raw.GameDataVersion(), (stage: string) => {
      performance.mark(`pob:data:${stage}`);
      return updates(stage);
    });
    console.log('Initialization took', Date.now() - start, 'ms');
    if (!result.ok) {
      console.error(result.error);
    }
  }

  async ImportCode(code: string) {
    if (this.dataReady) await this.dataReady;
    const xml = pob.DecodeDecompress(code).unwrap();
    const build = builds.ParseBuildStr(xml).unwrap();
    this.openBuild(build, 'ImportCode');
  }

  ExportCode(): string {
    if (!this.currentBuild) {
      return '';
    }

    const serialized = this.serializeForSave().unwrap();
    if (!serialized) {
      throw new Error('failed serializing build');
    }

    return pob.CompressEncode(new TextDecoder().decode(serialized)).unwrap();
  }

  private tickSeq = 0;
  private ticksInFlight = 0;

  async Tick(source: string, restored?: string) {
    const meta: TickMeta = { seq: 0, pending: 0, source, startedAt: epochNow(), calcMs: 0, dumpMs: 0, sentAt: 0 };
    const undoTicket = this.undoTicket(source);
    const started = ++this.ticksStarted;
    this.dropMiscCalculator();
    this.ticksInFlight++;
    let payload: Outputs | undefined;
    const cached = restored !== undefined && !this.calcDetail ? this.tickCache.get(restored) : undefined;
    try {
      payload = cached?.warnings ? this.fromCache(cached) : await this.computeOutputs(source, meta);
    } finally {
      this.ticksInFlight--;
      meta.seq = ++this.tickSeq;
      if (payload) {
        this.warningsEnvSeq = meta.seq;
      }
      if (cached?.warnings) {
        this.warnings = { ...cached.warnings, seq: meta.seq };
      } else if (payload && restored !== undefined && started === this.ticksStarted) {
        this.rememberTick(restored, payload, meta.seq);
      }
      if (payload && started === this.ticksStarted) {
        this.lastOutputs = payload;
      }
      meta.pending = this.ticksInFlight;
      meta.sentAt = epochNow();
      if (this.callback) {
        this.callback(payload, meta);
      }
      this.recordUndo(undoTicket, payload, meta.seq);
      if (payload) {
        this.loadDeferred();
      }
    }
  }

  private history = new UndoHistory();
  private writeSeq = 0;
  private pendingWrite?: string;
  private openPending = false;

  private undoTicket(source: string): { kind: 'open' | 'write' | 'none'; seq: number } {
    if (OPEN_SOURCES.has(source)) {
      this.openPending = true;
      return { kind: 'open', seq: ++this.writeSeq };
    }
    if (NOT_A_CHANGE.has(source)) {
      return { kind: 'none', seq: 0 };
    }
    this.pendingWrite ??= source;
    return { kind: 'write', seq: ++this.writeSeq };
  }

  private recordUndo(ticket: { kind: 'open' | 'write' | 'none'; seq: number }, payload?: Outputs, tickSeq = 0) {
    if (ticket.kind === 'none' || ticket.seq !== this.writeSeq || !this.currentBuild) {
      return;
    }
    if (this.openPending) {
      this.openPending = false;
      this.pendingWrite = undefined;
      const saved = this.saved();
      if (saved) {
        this.history.reset(saved.snap);
        if (payload) {
          this.rememberTick(saved.xml, payload, tickSeq);
        }
      }
      return;
    }
    this.whenIdle(() => {
      if (ticket.seq !== this.writeSeq || !this.currentBuild) {
        return;
      }
      const source = this.pendingWrite ?? 'Edit';
      this.pendingWrite = undefined;
      const saved = this.saved();
      if (saved && this.history.open) {
        this.history.record(source, stepLabel(source), saved.snap, Date.now());
      }
      if (saved && payload) {
        this.rememberTick(saved.xml, payload, tickSeq);
      }
    });
  }

  private idleTimer?: ReturnType<typeof setTimeout>;
  private idleTask?: () => void;
  private idleDone?: Promise<void>;
  private idleResolve?: () => void;
  private whenIdle(task: () => void) {
    this.idleTask = task;
    clearTimeout(this.idleTimer);
    this.idleDone ??= new Promise((resolve) => (this.idleResolve = resolve));
    const check = () => {
      if (this.ticksInFlight > 0 || this.warningsInFlight) {
        this.idleTimer = setTimeout(check, 50);
        return;
      }
      const run = this.idleTask;
      const resolve = this.idleResolve;
      this.idleTask = undefined;
      this.idleDone = undefined;
      this.idleResolve = undefined;
      run?.();
      resolve?.();
    };
    this.idleTimer = setTimeout(check, 50);
  }

  private snapshot(): Snapshot | undefined {
    return this.saved()?.snap;
  }

  private saved(): { xml: string; snap: Snapshot } | undefined {
    if (!this.currentBuild) {
      return undefined;
    }
    const result = builds.SerializeBuild(this.currentBuild);
    if (!result.ok || !result.value) {
      console.error('undo: failed serializing build', result.ok ? 'empty' : dump(result.error));
      return undefined;
    }
    const xml = new TextDecoder().decode(result.value);
    return { xml, snap: this.history.table.encode(xml) };
  }

  private tickCache = new TickCache();
  private tickCacheTarget?: { seq: number; entry: TickCacheEntry };

  private rememberTick(xml: string, payload: Outputs, seq: number) {
    if (payload.Breakdown !== undefined || payload.Calcs !== undefined) {
      return;
    }
    const groups = this.activeGroups((gs) => gs.map((g) => ({ label: g.DisplayLabel, slotEnabled: g.SlotEnabled })));
    if (!groups) {
      return;
    }
    const entry: TickCacheEntry = { payload, groups };
    if (this.warnings?.seq === seq) {
      entry.warnings = this.warnings;
    }
    this.tickCache.set(xml, entry);
    this.tickCacheTarget = { seq, entry };
  }

  private fromCache(entry: TickCacheEntry): Outputs {
    this.releaseWarningsEnv();
    this.warnings = undefined;
    this.activeGroups((gs) => {
      if (gs.length !== entry.groups.length) {
        throw new Error('undo: a kept calculation does not match the restored socket groups');
      }
      gs.forEach((g, i) => {
        g.DisplayLabel = entry.groups[i].label;
        g.SlotEnabled = entry.groups[i].slotEnabled;
      });
    });
    return entry.payload;
  }

  private forgetTicks() {
    this.tickCache.clear();
    this.tickCacheTarget = undefined;
  }

  private activeGroups<T>(visit: (groups: pob.Skill[]) => T): T | undefined {
    const build = this.currentBuild;
    if (!build) {
      return undefined;
    }
    const sets = build.Skills?.SkillSets ?? [];
    const groups = sets[build.ActiveSkillSetIndex()]?.Skills ?? [];
    try {
      return visit(groups);
    } finally {
      groups.forEach(releaseWrapper);
      sets.forEach(releaseWrapper);
    }
  }

  private recordUnrecorded(): { recorded: boolean; now?: Snapshot } {
    const before = this.history.current();
    const now = this.snapshot();
    if (!before || !now) {
      return { recorded: false, now };
    }
    const pending = this.pendingWrite;
    this.pendingWrite = undefined;
    this.writeSeq++;
    const label = pending ? stepLabel(pending) : unrecordedLabel(this.history.table.decode(before), this.history.table.decode(now));
    return { recorded: this.history.record(pending ?? UNRECORDED, label, now, Date.now(), false), now };
  }

  private applySnapshot(snap: Snapshot, source: string, live?: Snapshot) {
    const build = this.currentBuild;
    if (!build) {
      return;
    }
    const xml = this.history.table.decode(snap);
    const parsed = (live ? builds.ParseBuildOver(build, this.history.table.decode(live), xml) : builds.ParseBuildStr(xml)).unwrap();
    if (!parsed) {
      throw new Error('undo: failed parsing a recorded state');
    }
    try {
      const target = build as unknown as Record<string, unknown>;
      const from = parsed as unknown as Record<string, unknown>;
      for (const [field, d] of Object.entries(Object.getOwnPropertyDescriptors(parsed))) {
        if (!d.get || !d.set) {
          continue;
        }
        const value = from[field];
        target[field] = value;
        releaseWrapper(value);
      }
    } finally {
      releaseWrapper(parsed);
    }
    this.dropMiscCalculator();
    this.publishedBuild = undefined;
    this.updateStore();
    void this.Tick(source, xml);
  }

  private restore(step: () => Snapshot | undefined, source: string, live?: Snapshot): UndoView {
    if (this.currentBuild && this.history.open) {
      const snap = step();
      if (snap) {
        this.applySnapshot(snap, source, live);
      }
    }
    return this.history.view();
  }

  GetUndoHistory(): UndoView {
    return this.history.view();
  }

  async GetUndoHistoryWhenRecorded(): Promise<UndoView> {
    await this.idleDone;
    return this.history.view();
  }

  Undo(): UndoView {
    const live = this.currentBuild && this.history.open ? this.recordUnrecorded().now : undefined;
    return this.restore(() => this.history.undo(), 'Undo', live);
  }

  Redo(): UndoView {
    const unrecorded = this.currentBuild && this.history.open ? this.recordUnrecorded() : undefined;
    if (unrecorded?.recorded) {
      return this.history.view();
    }
    return this.restore(() => this.history.redo(), 'Redo', unrecorded?.now);
  }

  UndoTo(i: number): UndoView {
    const unrecorded = this.currentBuild && this.history.open ? this.recordUnrecorded() : undefined;
    if (unrecorded?.recorded) {
      return this.history.view();
    }
    return this.restore(() => this.history.goTo(i), 'UndoTo', unrecorded?.now);
  }

  private deferredLoaded?: Promise<void>;

  private loadDeferred() {
    this.deferredLoaded ??= raw.LoadDeferred().then((result) => {
      performance.mark('pob:data-deferred-loaded');
      if (!result.ok) {
        console.error(result.error);
      }
      this.warmItemDB();
    });
  }

  private async computeOutputs(source: string, meta: TickMeta): Promise<Outputs | undefined> {
    const started = this.ticksStarted;
    this.releaseWarningsEnv();
    this.warnings = undefined;
    if (!this.currentBuild) {
      return;
    }

    const calc = calculator.NewCalculator(this.currentBuild);
    if (!calc) {
      return;
    }

    console.log('TICK from', source);
    const calcStart = performance.now();
    const out = await calc.BuildOutput('MAIN');
    meta.calcMs = performance.now() - calcStart;

    const player = out?.Player;
    const mainSkill = player?.MainSkill;
    const minion = out?.Minion;
    const minionActor = minion?.Actor;
    const minionSkill = minionActor?.MainSkill;
    let keepEnv = false;
    try {
      if (!out || !player || !mainSkill) {
        return;
      }

      if (out.DebugErrors?.length) {
        console.log('Errors from last tick:');
        console.log(out.DebugErrors);
      }

      const dumpStart = performance.now();
      const payload: Outputs = {
        Output: player.Output,
        OutputTable: player.OutputTable,
        OutputStrings: player.OutputStrings,
        SkillFlags: mainSkill.SkillFlags,
        SkillDPS: dump(player.SkillDPS ?? [], true),
        MainSkill: dump(out.MainSkillControls(), true),
        PointsUsed: out.CountAllocNodes()
      };
      // Build.lua:1941-1976
      this.itemTipStats = out.ItemTooltipStats();
      // Build.lua:1849-1867
      if (minionActor && minionSkill) {
        payload.Minion = { Output: minionActor.Output, OutputTable: minionActor.OutputTable, SkillFlags: minionSkill.SkillFlags };
      }
      try {
        payload.ConfigRelevance = out.ConfigRelevance(configRelevanceQuery);
      } catch (e) {
        console.error('ConfigRelevance failed', e);
      }
      if (this.calcDetail) {
        payload.CalcsTab = await this.calcsTabOutputs();
      }
      // Build.lua:892-896
      const auto = this.fresh((b) => ({ on: b.Build.CharacterLevelAutoMode, level: b.Build.Level }));
      const level =
        auto && payload.PointsUsed ? autoLevel(estimatePlayerProgress(payload.PointsUsed, player.Output?.ExtraPoints), auto.on, auto.level) : undefined;
      if (level !== undefined && started === this.ticksStarted) {
        this.currentBuild.SetLevel(level);
        void this.Tick('EstimatePlayerProgress');
      }
      meta.dumpMs = performance.now() - dumpStart;
      if (started === this.ticksStarted && !this.misc) {
        this.tickMisc = exposition.MiscCalculatorFor(this.currentBuild, calc) ?? undefined;
      }
      this.releaseWarningsEnv();
      this.warningsEnv = { out, calc };
      keepEnv = true;
      return payload;
    } finally {
      releaseWrapper(minionSkill);
      releaseWrapper(minionActor);
      releaseWrapper(minion);
      releaseWrapper(mainSkill);
      releaseWrapper(player);
      if (!keepEnv) {
        releaseWrapper(out);
        calc.release();
      }
    }
  }

  /* Calcs.lua:432-468 */
  private warningsEnv?: WarningsEnv;
  private warningsEnvSeq = 0;
  private warnings?: TickWarnings;
  private warningsInFlight?: { seq: number; result: Promise<TickWarnings | undefined> };

  private releaseWarningsEnv() {
    if (this.warningsEnv) {
      releaseWrapper(this.warningsEnv.out);
      this.warningsEnv.calc.release();
      this.warningsEnv = undefined;
    }
  }

  /* Build.lua:1778-1831 */
  GetSidebarWarnings(): Promise<TickWarnings | undefined> {
    const seq = this.warningsEnvSeq;
    if (this.warnings?.seq === seq) {
      return Promise.resolve(this.warnings);
    }
    if (this.warningsInFlight?.seq === seq) {
      return this.warningsInFlight.result;
    }
    const env = this.warningsEnv;
    if (!env) {
      return Promise.resolve(undefined);
    }
    const result = this.runSidebarWarnings(env, seq).finally(() => {
      if (this.warningsInFlight?.result === result) {
        this.warningsInFlight = undefined;
      }
    });
    this.warningsInFlight = { seq, result };
    return result;
  }

  private async runSidebarWarnings(env: WarningsEnv, seq: number): Promise<TickWarnings | undefined> {
    await new Promise((resolve) => setTimeout(resolve, WARNINGS_QUIET_MS));
    if (this.warningsEnv !== env || this.warningsEnvSeq !== seq) {
      return undefined;
    }
    const start = performance.now();
    const run = env.out.NewSidebarWarningsRun();
    if (!run) {
      return undefined;
    }
    if (performance.getEntriesByName(WARNINGS_STEP).length > 500) {
      performance.clearMeasures(WARNINGS_STEP);
    }
    try {
      for (;;) {
        const stepStart = performance.now();
        const more = run.Step();
        performance.measure(WARNINGS_STEP, { start: stepStart, end: performance.now() });
        if (!more) {
          break;
        }
        await yieldToQueue();
        if (this.warningsEnv !== env || this.warningsEnvSeq !== seq) {
          return undefined;
        }
      }
      const warnings = dump(run.Result(), true);
      this.releaseWarningsEnv();
      if (!warnings) {
        return undefined;
      }
      this.warnings = { seq, warnings, costWarning: warnings.CostWarning ?? [], ms: performance.now() - start };
      if (this.tickCacheTarget?.seq === seq) {
        this.tickCacheTarget.entry.warnings = this.warnings;
      }
      return this.warnings;
    } finally {
      run.release();
    }
  }

  SetConfigOption(key: string, value: boolean | number | string) {
    if (!this.currentBuild || !this.currentBuild.Config.Inputs) {
      return;
    }

    let remove;
    const v = reverseConfigOptions[key];
    switch (v.type) {
      case 'list':
        remove = value === v.list[(v.defaultIndex ?? 1) - 1].value;
        break;
      case 'check':
        if (v.defaultState !== undefined) {
          remove = value === v.defaultState;
        } else {
          remove = value === false;
        }
        break;
      case 'text':
        remove = value === '';
        break;
      default:
        remove = value === null;
        break;
    }

    if (remove) {
      this.currentBuild.RemoveConfigOption(key);
      this.updateStore();
      void this.Tick('SetConfigOption: remove');
      return;
    }

    const newValue: pob.Input =
      typeof value === 'boolean' ? { Name: key, Boolean: value } : typeof value === 'string' ? { Name: key, String: value } : { Name: key, Number: value };

    this.currentBuild.SetConfigOption(newValue);
    this.updateStore();
    void this.Tick('SetConfigOption: change');
  }

  GetConfigOption(name: string): boolean | number | string | undefined {
    const inputs = this.fresh((b) => b.Config.Inputs);
    if (!inputs) {
      return undefined;
    }

    const input = inputs.find((i) => i.Name === name);
    if (!input) {
      return undefined;
    }

    if (input.String != null) {
      return input.String;
    }

    if (input.Number != null) {
      return input.Number;
    }

    if (input.Boolean != null) {
      return input.Boolean;
    }

    return undefined;
  }

  GetAllConfigOptions(): Record<string, boolean | number | string> {
    const inputs = this.fresh((b) => b.Config.Inputs);
    if (!inputs) {
      return {};
    }

    return inputs.reduce((out, input) => {
      let val: unknown;

      if (input.String != null) {
        val = input.String;
      }

      if (input.Number != null) {
        val = input.Number;
      }

      if (input.Boolean != null) {
        val = input.Boolean;
      }

      return {
        ...out,
        [input.Name]: val
      };
    }, {});
  }

  private fresh<T>(read: (build: pob.PathOfBuilding) => T): T | undefined {
    if (!this.currentBuild) {
      return undefined;
    }
    const calc = calculator.NewCalculator(this.currentBuild);
    const build = calc?.PoB;
    try {
      return build ? read(build) : undefined;
    } finally {
      releaseWrapper(build);
      calc?.release();
    }
  }

  GetBuildInfo() {
    const activeSpec = (b: pob.PathOfBuilding) => b.Tree.Specs?.[b.Tree.ActiveSpec - 1] ?? b.Tree.Specs?.[0];
    return this.fresh((b) => ({
      level: b.Build.Level,
      className: b.Build.ClassName,
      ascendClassName: b.Build.AscendClassName,
      mainSocketGroup: b.Build.MainSocketGroup,
      secondaryAscendClassId: activeSpec(b)?.SecondaryAscendClassID ?? 0,
      recordedTreeVersion: activeSpec(b)?.TreeVersion
    }));
  }

  GetTreeVersion(): string {
    return exposition.CalcTreeVersion();
  }

  GameDataVersion(): string {
    return raw.GameDataVersion();
  }

  GetPassiveNodes(): number[] {
    return this.fresh((b) => [...(b.Build.PassiveNodes ?? [])]) ?? [];
  }

  SetMainSocketGroup(mainSocketGroup: number) {
    this.currentBuild?.SetMainSocketGroup(mainSocketGroup);
    void this.Tick('SetMainSocketGroup');
  }

  DefaultGem(gemID: string, nameSpec: string, skillID: string, level: number, quality: number): pob.Gem {
    return exposition.DefaultGem(gemID, nameSpec, skillID, level, quality);
  }
  /* SkillsTab.lua:1007-1012 */
  GemError(gemID: string, skillID: string, nameSpec: string): string {
    return exposition.GemError(gemID, skillID, nameSpec);
  }

  GetSkillGems(): exposition.SkillGem[] {
    return exposition.GetSkillGems()!;
  }

  async GetTree(version: string): Promise<string> {
    const rawData = await exposition.GetRawTree(version);
    if (!rawData) {
      throw new Error('Failed loading tree');
    }
    return new TextDecoder().decode(rawData);
  }

  SetClass(value: string) {
    this.currentBuild?.SetClass(value);
    void this.Tick('SetClass');
  }

  SetAscendancy(value: string) {
    this.currentBuild?.SetAscendancy(value);
    void this.Tick('SetAscendancy');
  }

  SetLevel(value: number) {
    // Build.lua:218-224
    this.currentBuild?.SetCharacterLevelAutoMode(false);
    this.currentBuild?.SetLevel(value);
    void this.Tick('SetLevel');
  }

  /* Build.lua:211-215 */
  SetCharacterLevelAutoMode(auto: boolean) {
    this.currentBuild?.SetCharacterLevelAutoMode(auto);
    void this.Tick('SetCharacterLevelAutoMode');
  }

  AllocateNodes(nodeIds: number[]) {
    this.currentBuild?.AllocateNodes(nodeIds);
    void this.Tick('AllocateNode');
  }

  DeallocateNodes(nodeIds: number[]) {
    this.currentBuild?.DeallocateNodes(nodeIds);
    void this.Tick('DeallocateNodes');
  }

  CalculateAllocationPaths(version: string, activeNodes: number[], rootNodes: number[]) {
    return exposition.CalculateAllocationPaths(version, activeNodes, rootNodes);
  }

  CalculatePrunableNodes(version: string, activeNodes: number[], rootNodes: number[]) {
    return exposition.CalculatePrunableNodes(version, activeNodes, rootNodes);
  }

  CalculateDependentNodes(version: string, activeNodes: number[], target: number, rootNodes: number[]) {
    return exposition.CalculateDependentNodes(version, activeNodes, target, rootNodes);
  }

  /* PassiveSpec.lua:1438-1592 */
  CalculateDependentNodesWith(
    version: string,
    activeNodes: number[],
    target: number,
    rootNodes: number[],
    subgraph: Parameters<typeof exposition.CalculateDependentNodesWith>[4]
  ) {
    return exposition.CalculateDependentNodesWith(version, activeNodes, target, rootNodes, subgraph);
  }

  /* src/Modules/Calcs.lua:123-148 */
  private misc?: Promise<{ calc: exposition.MiscCalculator; base: exposition.StatOutputs } | undefined>;
  private tickMisc?: exposition.MiscCalculator;
  private ticksStarted = 0;
  private miscCalls = 0;
  private miscReleases: exposition.MiscCalculator[] = [];

  private dropMiscCalculator() {
    this.tickMisc?.release();
    this.tickMisc = undefined;
    const old = this.misc;
    this.misc = undefined;
    void old?.then((m) => {
      if (m) {
        this.miscReleases.push(m.calc);
        this.releaseIdleMisc();
      }
    });
  }

  private releaseIdleMisc() {
    if (this.miscCalls === 0) {
      this.miscReleases.splice(0).forEach((c) => c.release());
    }
  }

  private miscCalculator() {
    if (!this.misc && this.tickMisc) {
      const calc = this.tickMisc;
      this.tickMisc = undefined;
      this.misc = Promise.resolve({ calc, base: calc.Base() });
    }
    if (!this.misc && this.currentBuild) {
      this.misc = exposition.GetMiscCalculator(this.currentBuild).then((calc) => (calc ? { calc, base: calc.Base() } : undefined));
    }
    return this.misc ?? Promise.resolve(undefined);
  }

  async PrepareStatCompare() {
    await this.miscCalculator();
  }

  async GetGrantedPassives(): Promise<number[]> {
    return [...((await this.miscCalculator())?.base.GrantedPassives ?? [])];
  }

  async StatCompareNodes(
    addNodes: number[],
    removeNodes: number[],
    nodeCount?: number
  ): Promise<{ sections: StatCompareSection[]; calcMs: number } | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) {
      return undefined;
    }
    this.miscCalls++;
    try {
      const start = performance.now();
      const out = await m.calc.CalcNodes(addNodes, removeNodes);
      const calcMs = performance.now() - start;
      if (token !== this.misc) {
        return undefined;
      }
      const flags = (m.base.SkillFlags ?? {}) as Record<string, boolean>;
      const sections = addStatCompares(
        flags,
        { Output: m.base.Output ?? {}, Minion: m.base.Minion },
        { Output: out.Output ?? {}, Minion: out.Minion },
        nodeCount
      );
      return { sections, calcMs };
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  BuildInfo() {
    return dump(pob.BuildInfo);
  }

  /* CalcsTab.lua:452-455, CalcSectionControl.lua:179, CalcsTab.lua:390, CalcSetup.lua:1483-1485, CalcsTab.lua:40-119 */
  private async calcsTabOutputs(build = this.currentBuild, keepEnv: boolean | ((env: calculator.Environment) => void) = true): Promise<Outputs | undefined> {
    if (!build) return undefined;
    const calc = calculator.NewCalculator(build);
    if (!calc) return undefined;
    calc.AlignCalcsSkillToMain = false;
    const env = await calc.BuildOutput('CALCS');
    const fresh = calc.PoB;
    const showMinion = fresh?.CalcsInputBool('showMinion') ?? false;
    releaseWrapper(fresh);
    const player = env?.Player;
    const minionActor = env?.Minion?.Actor;
    const actor = showMinion && minionActor?.MainSkill ? minionActor : player;
    const skill = actor?.MainSkill;
    const breakdown = actor?.Breakdown;
    try {
      if (!env || !actor || !skill) return undefined;
      return {
        Output: actor.Output,
        OutputTable: actor.OutputTable,
        OutputStrings: actor.OutputStrings,
        SkillFlags: skill.SkillFlags,
        Breakdown: breakdown?.GetData() || {},
        Calcs: dump(env.CalcProps || {}, true),
        MainSkill: dump(env.CalcsSkillControls(), true)
      };
    } finally {
      releaseWrapper(breakdown);
      releaseWrapper(skill);
      if (minionActor !== actor) releaseWrapper(minionActor);
      releaseWrapper(actor);
      if (player !== actor) releaseWrapper(player);
      // CalcBreakdownControl.lua:271-498
      if (typeof keepEnv === 'function') {
        if (env) keepEnv(env);
      } else if (keepEnv) {
        releaseWrapper(this.calcsEnv);
        this.calcsEnv = env;
      } else releaseWrapper(env);
      calc.release();
    }
  }

  /* CalcsTab.lua:28-30, 87-91, 120-125 */
  GetCalcsTabInputs(): { buffMode: string; showMinion: boolean; skillNumber: number } {
    return (
      this.fresh((b) => ({
        buffMode: b.CalcsInputString('misc_buffMode', 'EFFECTIVE'),
        showMinion: b.CalcsInputBool('showMinion'),
        skillNumber: b.CalcsInputNumber('skill_number', 1)
      })) ?? { buffMode: 'EFFECTIVE', showMinion: false, skillNumber: 1 }
    );
  }

  /* CalcsTab.lua:40-44 */
  SetCalcsSkillNumber(skillNumber: number) {
    this.currentBuild?.SetCalcsInputNumber('skill_number', skillNumber);
    this.updateStore();
    void this.Tick('SetCalcsSkillNumber');
  }

  /* CalcsTab.lua:120-125 */
  SetCalcsBuffMode(buffMode: string) {
    this.currentBuild?.SetCalcsInputString('misc_buffMode', buffMode);
    this.updateStore();
    void this.Tick('SetCalcsBuffMode');
  }

  /* CalcsTab.lua:87-91 */
  SetCalcsShowMinion(show: boolean) {
    this.currentBuild?.SetCalcsInputBool('showMinion', show);
    this.updateStore();
    void this.Tick('SetCalcsShowMinion');
  }

  calcDetail = false;

  setCalcTabElements(elements: Record<string, CalcDataColProp>) {
    this.calcDetail = true;
    exposition.SetCalcTabElements({ Elements: colPropsOf(elements) });
    void this.Tick('setCalcTabElements');
  }

  clearCalcTabElements() {
    this.calcDetail = false;
    exposition.SetCalcTabElements({ Elements: {} });
  }

  async ListBuilds(dir: string) {
    const b = (await storage.ListBuilds(dir)).unwrapOr(undefined);
    return dump(b);
  }

  async NewFolder(name: string) {
    (await storage.NewFolder(name)).unwrap();
  }

  async Copy(from: string, to: string) {
    const build = (await storage.GetBuild(from)).unwrap();
    if (!build) {
      return;
    }

    (await storage.SetBuild(to, build)).unwrap();
  }

  async Rename(from: string, to: string) {
    await this.Copy(from, to);
    (await storage.DeleteBuild(from)).unwrap();
  }

  async Delete(path: string) {
    (await storage.DeleteBuild(path)).unwrap();
  }

  /* Build.lua:79-84, SkillsTab.lua:92 */
  async NewBuild(defaults?: { level: number; gemQuality: number }) {
    if (this.dataReady) await this.dataReady;
    const build = builds.EmptyBuild();
    if (defaults) build?.ApplyNewBuildDefaults(defaults.level, defaults.gemQuality);
    this.openBuild(build, 'NewBuild');
  }

  /* Build.lua:1014-1061 */
  private lastOutputs?: Outputs;

  private serializeForSave() {
    const out = this.lastOutputs;
    if (!out) {
      return builds.SerializeBuild(this.currentBuild);
    }
    const s = savedStats(out);
    const num = (v: string) => (v === 'inf' ? Infinity : v === '-inf' ? -Infinity : Number(v));
    return builds.SerializeBuildWithStats(
      this.currentBuild,
      s.player.map((p) => ({ Stat: p.stat, Value: num(p.value) })),
      s.fullDPS.map((f) => ({ Stat: f.stat, Value: f.value, SkillPart: f.skillPart, Source: f.source })),
      s.minion.map((p) => ({ Stat: p.stat, Value: num(p.value) }))
    );
  }

  /* Build.lua:2069-2072 */
  async SaveBuildAs(path: string) {
    const result = this.serializeForSave();

    if (!result.ok) {
      console.error(dump(result.error));
      throw new Error('failed serializing build');
    }

    if (!result.value) {
      throw new Error('failed serializing build');
    }

    const build = new TextDecoder().decode(result.value);

    (await storage.SetBuild(path, build)).unwrap();
  }

  async OpenBuild(path: string) {
    if (this.dataReady) await this.dataReady;
    const result = await storage.GetBuild(path);

    if (!result.ok) {
      console.error(dump(result.error));
      return;
    }

    this.openBuild(builds.ParseBuildStr(result.value).unwrap(), 'OpenBuild');
  }

  ClearBuild() {
    this.ClearCompare();
    this.currentBuild = undefined;
    this.history.clear();
    this.forgetTicks();
  }

  SetNotes(notes: string) {
    this.currentBuild?.SetNotes(notes);
    this.updateStore();
  }

  ParseModLines(lines: string[]): exposition.ModLine[] {
    return exposition.ParseModLines(lines) ?? [];
  }

  GetCustomModifierBlocks(): pob.CustomModifierBlock[] {
    return this.currentBuild?.GetCustomModifierBlocks() ?? [];
  }

  SetCustomModifierBlocks(blocks: pob.CustomModifierBlock[]) {
    if (!this.currentBuild) {
      return;
    }
    this.currentBuild.SetCustomModifierBlocks(blocks);
    this.updateStore();
    void this.Tick('SetCustomModifierBlocks');
  }

  ResetConfigOptions() {
    if (!this.currentBuild) {
      return;
    }
    this.currentBuild.ResetConfigOptions();
    this.updateStore();
    void this.Tick('ResetConfigOptions');
  }

  GetConfigSets(): { sets: { id: number; title?: string }[]; active: number } {
    const sets = this.fresh((b) => (b.Config.ConfigSets ?? []).map((s) => ({ id: s.ID, title: s.Title ?? undefined }))) ?? [];
    return { sets, active: this.currentBuild?.ActiveConfigSetIndex() ?? 0 };
  }

  /* ConfigTab.lua:152-155 */
  SetActiveConfigSet(configSetId: number) {
    if (!this.currentBuild) {
      return;
    }
    this.currentBuild.SetActiveConfigSet(configSetId);
    this.updateStore();
    void this.Tick('SetActiveConfigSet');
  }

  /* ConfigOptions.lua:1457-1589, 2246-2286 */
  GetConfigPlaceholders(): Record<string, number | string> {
    const placeholders = this.fresh((b) => b.Config.Placeholders) ?? [];
    const out: Record<string, number | string> = {};
    for (const p of placeholders) {
      const value = p.Number ?? p.String;
      if (value !== undefined) {
        out[p.Name] = value;
      }
    }
    return this.currentBuild ? { ...out, ...(exposition.ConfigPlaceholders(this.currentBuild) ?? {}) } : out;
  }

  GetItems() {
    if (!this.currentBuild) {
      return;
    }

    return this.fresh((b) => dump(b.Items));
  }

  GetSocketGroupSummaries(skillSet: number) {
    const sets = this.currentBuild?.Skills?.SkillSets;
    const groups = sets?.[skillSet]?.Skills;
    if (!groups) {
      return [];
    }

    return groups.map((g) => ({
      label: g.DisplayLabel || g.Label || '<No active skills>',
      enabled: g.Enabled,
      fullDPS: g.IncludeInFullDPS ?? false
    }));
  }

  /* SkillsTab.lua:95-98 */
  SetActiveSkillSet(skillSetId: number) {
    if (!this.currentBuild) {
      return;
    }
    this.currentBuild.SetActiveSkillSet(skillSetId);
    this.updateStore();
    void this.Tick('SetActiveSkillSet');
  }

  /* SkillSetListControl.lua:48-50, SkillsTab.lua:1425-1435 */
  NewSkillSet(title: string) {
    const id = this.currentBuild?.NewSkillSet(title);
    this.updateStore();
    void this.Tick('NewSkillSet');
    return id;
  }

  /* SkillSetListControl.lua:14-32 */
  CopySkillSet(skillSetId: number, title: string) {
    const id = this.currentBuild?.CopySkillSet(skillSetId, title);
    this.updateStore();
    void this.Tick('CopySkillSet');
    return id;
  }

  /* SkillSetListControl.lua:53-79 */
  RenameSkillSet(skillSetId: number, title: string) {
    this.currentBuild?.RenameSkillSet(skillSetId, title);
    this.updateStore();
    void this.Tick('RenameSkillSet');
  }

  /* SkillSetListControl.lua:99-110 */
  DeleteSkillSet(skillSetId: number) {
    this.currentBuild?.DeleteSkillSet(skillSetId);
    this.updateStore();
    void this.Tick('DeleteSkillSet');
  }

  /* ListControl.lua:428-436, SkillListControl.lua:123-142 */
  MoveSocketGroup(from: number, to: number) {
    if (!this.currentBuild) {
      return;
    }
    this.currentBuild.MoveSocketGroup(from, to);
    this.updateStore();
    void this.Tick('MoveSocketGroup');
  }

  private itemTipStats: calculator.ItemTooltipStats = NO_TIP_STATS;

  /* ItemsTab.lua:2635-2653 */
  async ParseItem(text: string): Promise<{ item?: exposition.ParsedItem; error?: string }> {
    const result = await exposition.ParseItem(this.currentBuild, text, this.itemTipStats);
    return result.ok ? { item: result.value } : { error: String(result.error) };
  }

  async GetEquippedItems(): Promise<exposition.EquippedSlot[]> {
    return this.currentBuild ? ((await exposition.EquippedItems(this.currentBuild)) ?? []) : [];
  }

  private async itemWhatIf<T>(
    run: (calc: exposition.MiscCalculator) => Promise<T>,
    toBlocks: (value: T) => { header: string; output: exposition.StatOutputs; slot?: string; lines?: string[] }[]
  ): Promise<{ blocks: ItemStatDiffBlock[]; calcMs: number } | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) {
      return undefined;
    }
    this.miscCalls++;
    try {
      const start = performance.now();
      const value = await run(m.calc);
      const calcMs = performance.now() - start;
      if (token !== this.misc) {
        return undefined;
      }
      const flags = (m.base.SkillFlags ?? {}) as Record<string, boolean>;
      const blocks = toBlocks(value).map(({ header, output, slot, lines }) => ({
        header,
        slot,
        ...(lines?.length ? { lines } : {}),
        sections: addStatCompares(flags, { Output: m.base.Output ?? {}, Minion: m.base.Minion }, { Output: output.Output ?? {}, Minion: output.Minion })
      }));
      return { blocks, calcMs };
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* ItemsTab.lua:5003-5158 */
  async ItemCompare(text: string) {
    const parsed = await exposition.ParseItem(this.currentBuild, text, NO_TIP_STATS);
    if (!parsed.ok) {
      return undefined;
    }
    if (parsed.value.Flask) {
      return this.itemWhatIf(
        async (calc) => (await calc.CalcToggleFlask(text)).unwrap(),
        (f) => [
          {
            header: `${f.Active ? 'Deactivating' : 'Activating'} this ${f.Tincture ? 'tincture' : 'flask'} will give you:`,
            output: f.Output,
            lines: [...(f.Effective ?? [])]
          }
        ]
      );
    }
    return this.itemWhatIf(
      async (calc) => (await calc.CompareItem(text)).unwrap() ?? [],
      (slots) =>
        slots.map((s) => ({
          header: `Equipping this item in ${s.Label} will give you:${s.Replacing ? `\n(replacing ${s.Replacing.Name})` : ''}`,
          output: s.Output,
          slot: s.SlotName
        }))
    );
  }

  /* ItemsTab.lua:5061-5078 */
  async ItemUnequipCompare(slotName: string, label: string) {
    return this.itemWhatIf(
      (calc) => calc.CalcUnequip(slotName),
      (output) => [{ header: `Removing this item from ${label} will give you:`, output }]
    );
  }

  async FlaskToggleCompare(text: string) {
    return this.itemWhatIf(
      async (calc) => (await calc.CalcToggleFlask(text)).unwrap(),
      (f) => [
        {
          header: `${f.Active ? 'Deactivating' : 'Activating'} this ${f.Tincture ? 'tincture' : 'flask'} will give you:`,
          output: f.Output,
          lines: [...(f.Effective ?? [])]
        }
      ]
    );
  }

  /* ItemSlotControl.lua:12-19 */
  async EquipItem(text: string, slotName: string): Promise<string | undefined> {
    if (!this.currentBuild) {
      return 'no build';
    }
    const result = await exposition.EquipItem(this.currentBuild, text, slotName);
    if (!result.ok) {
      return String(result.error);
    }
    this.updateStore();
    void this.Tick('EquipItem');
    return undefined;
  }

  /* ItemsTab.lua:1679-1703 */
  async SaveItem(text: string): Promise<{ slotName?: string; error?: string }> {
    const m = await this.miscCalculator();
    if (!m) {
      return { error: 'no build' };
    }
    const result = await m.calc.AddItem(text);
    if (!result.ok) {
      return { error: String(result.error) };
    }
    this.updateStore();
    void this.Tick('SaveItem');
    return { slotName: result.value.SlotName || undefined };
  }

  /* ItemsTab.lua:2626-2629, 1705-1707 */
  async ReplaceItem(id: number, text: string): Promise<string | undefined> {
    const m = await this.miscCalculator();
    if (!m) {
      return 'no build';
    }
    const result = await m.calc.ReplaceItem(id, text);
    if (!result.ok) {
      return String(result.error);
    }
    this.updateStore();
    void this.Tick('ReplaceItem');
    return undefined;
  }

  UnequipSlot(slotName: string) {
    this.currentBuild?.SetSlotItem(slotName, 0);
    this.updateStore();
    void this.Tick('UnequipSlot');
  }

  /* ItemsTab.lua:99-102 */
  async SetActiveItemSet(itemSetId: number) {
    if (!this.currentBuild) {
      return;
    }
    await exposition.SetActiveItemSet(this.currentBuild, itemSetId);
    this.updateStore();
    void this.Tick('SetActiveItemSet');
  }

  /* ItemSlotControl.lua:82-93 */
  async GetSlotItemLists(): Promise<Record<string, number[]>> {
    if (!this.currentBuild) {
      return {};
    }
    const lists = (await exposition.SlotItemLists(this.currentBuild)) ?? [];
    return Object.fromEntries(lists.map((s) => [s.SlotName, [...(s.ItemIDs ?? [])]]));
  }

  /* ItemSlotControl.lua:12-19 */
  async EquipItemID(slotName: string, itemId: number): Promise<string | undefined> {
    if (!this.currentBuild) {
      return 'no build';
    }
    const result = await exposition.EquipItemID(this.currentBuild, slotName, itemId);
    if (!result.ok) {
      return String(result.error);
    }
    if (result.value) {
      this.updateStore();
      void this.Tick('EquipItemID');
    }
    return undefined;
  }

  /* ItemSlotControl.lua:33-38 */
  SetFlaskActive(slotName: string, active: boolean) {
    this.currentBuild?.SetFlaskActive(slotName, active);
    this.updateStore();
    void this.Tick('SetFlaskActive');
  }

  async GetTreeSpecs(): Promise<{ active: number; specs: calculator.TreeSpecRow[] } | undefined> {
    if (!this.currentBuild) {
      return undefined;
    }
    const list = await exposition.GetTreeSpecs(this.currentBuild);
    return { active: list.Active, specs: [...(list.Specs ?? [])] };
  }

  /* PassiveSpecListControl.lua:74 */
  async GetTreeSpecsPoints(): Promise<calculator.TreeSpecPoints[]> {
    return this.currentBuild ? [...((await exposition.GetTreeSpecsPoints(this.currentBuild)) ?? [])] : [];
  }

  /* TreeTab.lua:39-46 */
  async SetActiveTreeSpec(specId: number) {
    if (!this.currentBuild) {
      return;
    }
    await exposition.SetActiveTreeSpec(this.currentBuild, specId);
    this.updateStore();
    void this.Tick('SetActiveTreeSpec');
  }

  /* PassiveSpecListControl.lua:36-42 */
  async NewTreeSpec(title: string) {
    if (!this.currentBuild) {
      return;
    }
    await exposition.NewTreeSpec(this.currentBuild, title);
    this.updateStore();
    void this.Tick('NewTreeSpec');
  }

  /* PassiveSpecListControl.lua:13-20 */
  CopyTreeSpec(specId: number, title: string) {
    this.currentBuild?.CopySpec(specId, title);
    this.updateStore();
    void this.Tick('CopyTreeSpec');
  }

  /* PassiveSpecListControl.lua:46-70 */
  RenameTreeSpec(specId: number, title: string) {
    this.currentBuild?.RenameSpec(specId, title);
    this.updateStore();
    void this.Tick('RenameTreeSpec');
  }

  // TreeTab.lua:841-991

  GetTreeTattoos(): { tattoos: Record<number, string>; count: number } {
    const tattoos: Record<number, string> = {};
    const overrides = this.fresh((b) => b.Tree.Specs?.[b.ActiveSpecIndex()]?.Overrides ?? []) ?? [];
    for (const o of overrides) {
      tattoos[o.NodeID] = o.DN;
    }
    return { tattoos, count: this.currentBuild?.TattooCount() ?? 0 };
  }

  RemoveTattoo(nodeId: number) {
    if (this.currentBuild?.RemoveTattoo(nodeId)) {
      this.updateStore();
      void this.Tick('RemoveTattoo');
    }
  }

  // CalcsTab.lua:489-733, TreeTab.lua:1052-1159

  private powerSeq = 0;
  private powerDone = 0;

  GetPowerStats(): calculator.PowerStat[] {
    return [...(exposition.PowerStats() ?? [])];
  }

  async BuildPowerReport(
    stat: string,
    maxDepth: number
  ): Promise<{ rows: calculator.PowerReportEntry[]; ascendancy?: calculator.PowerReportEntry[]; ms: number } | undefined> {
    const seq = ++this.powerSeq;
    this.powerDone = 0;
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc || seq !== this.powerSeq) {
      return undefined;
    }
    const builder = exposition.NewPowerBuilder(m.calc, stat, maxDepth);
    if (!builder) {
      return undefined;
    }
    this.miscCalls++;
    const start = performance.now();
    try {
      for (let done = 0; done < 1; ) {
        if (token !== this.misc || seq !== this.powerSeq) {
          return undefined;
        }
        done = builder.Step(40);
        this.powerDone = done;
        await yieldToQueue();
      }
      if (token !== this.misc || seq !== this.powerSeq) {
        return undefined;
      }
      return { rows: [...(builder.Report() ?? [])], ascendancy: [...(exposition.PowerAscendancy(builder) ?? [])], ms: performance.now() - start };
    } finally {
      builder.release();
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  PowerReportProgress(): number {
    return this.powerDone;
  }

  CancelPowerReport() {
    this.powerSeq++;
    this.powerPool?.cancel();
  }

  // TreeTab.lua:992-1036

  GetMasterySelections(): Record<number, number> {
    return this.currentBuild ? { ...(exposition.GetMasterySelections(this.currentBuild) ?? {}) } : {};
  }

  async GetMasteryOptions(nodeId: number): Promise<calculator.MasteryOption[]> {
    return this.currentBuild ? [...((await exposition.GetMasteryOptions(this.currentBuild, nodeId)) ?? [])] : [];
  }

  async SelectMasteryEffect(nodeId: number, effectId: number, path: number[]): Promise<string | undefined> {
    if (!this.currentBuild) {
      return 'no build';
    }
    const result = await exposition.SelectMasteryEffect(this.currentBuild, nodeId, effectId, [...path]);
    if (!result.ok) {
      return String(result.error);
    }
    this.updateStore();
    void this.Tick('SelectMasteryEffect');
    return undefined;
  }

  async GetTreeSocketedJewels(): Promise<exposition.TreeJewel[]> {
    return this.currentBuild ? [...((await exposition.TreeSocketedJewels(this.currentBuild)) ?? [])] : [];
  }

  /* ItemSlotControl.lua:85-93 */
  async GetTreeSocketChoices(nodeId: number): Promise<exposition.TreeJewel[]> {
    return this.currentBuild ? [...((await exposition.TreeSocketChoices(this.currentBuild, nodeId)) ?? [])] : [];
  }

  /* ItemSlotControl.lua:12-19 */
  async SocketTreeJewel(nodeId: number, itemId: number): Promise<string | undefined> {
    if (!this.currentBuild) {
      return 'no build';
    }
    const result = await exposition.SocketTreeJewel(this.currentBuild, nodeId, itemId);
    if (!result.ok) {
      return String(result.error);
    }
    if (result.value) {
      this.updateStore();
      void this.Tick('SocketTreeJewel');
    }
    return undefined;
  }

  /* PassiveSpecListControl.lua:95-111 */
  async DeleteTreeSpec(specId: number) {
    if (!this.currentBuild) {
      return;
    }
    await exposition.DeleteTreeSpec(this.currentBuild, specId);
    this.updateStore();
    void this.Tick('DeleteTreeSpec');
  }

  private compareB?: { build: pob.PathOfBuilding; name: string; seq: number; side: CompareSideData };
  private compareList: NonNullable<typeof this.compareB>[] = [];
  private compareSeq = 0;
  private compareMisc?: Promise<{ calc: exposition.MiscCalculator; base: exposition.StatOutputs } | undefined>;
  private comparePowerSeq = 0;

  async LoadCompareCode(code: string, name = 'Imported build'): Promise<CompareInfo> {
    if (this.dataReady) await this.dataReady;
    const xml = pob.DecodeDecompress(code).unwrap();
    return await this.loadCompare(builds.ParseBuildStr(xml).unwrap(), name);
  }

  async LoadCompareSaved(path: string): Promise<CompareInfo> {
    if (this.dataReady) await this.dataReady;
    const xml = (await storage.GetBuild(path)).unwrap();
    return await this.loadCompare(
      builds.ParseBuildStr(xml).unwrap(),
      path
        .split('/')
        .at(-1)
        ?.replace(/\.xml$/, '') ?? path
    );
  }

  private async loadCompare(build: pob.PathOfBuilding | undefined, name: string): Promise<CompareInfo> {
    if (!build) {
      throw new Error('the build did not parse');
    }
    const side = await this.compareSide(build, true);
    if (!side?.outputs) {
      releaseWrapper(build);
      throw new Error('the build did not calculate');
    }
    // CompareTab.lua:1240-1253
    this.dropCompareState();
    this.compareB = { build, name, seq: ++this.compareSeq, side };
    this.compareList.push(this.compareB);
    return this.GetCompareInfo();
  }

  private dropCompareState() {
    this.comparePowerSeq++;
    const misc = this.compareMisc;
    this.compareMisc = undefined;
    void misc?.then((m) => m?.calc.release());
  }

  /* CompareTab.lua:229-234 */
  SetActiveCompare(index: number): CompareInfo {
    const entry = this.compareList[index];
    if (entry && entry !== this.compareB) {
      this.dropCompareState();
      this.compareB = entry;
    }
    return this.GetCompareInfo();
  }

  /* CompareTab.lua:1268-1276 */
  RemoveCompare(): CompareInfo {
    const i = this.compareB ? this.compareList.indexOf(this.compareB) : -1;
    if (i < 0) return this.GetCompareInfo();
    const [old] = this.compareList.splice(i, 1);
    this.dropCompareState();
    this.compareB = this.compareList[Math.min(i, this.compareList.length - 1)];
    releaseWrapper(old.build);
    return this.GetCompareInfo();
  }

  ClearCompare() {
    const list = this.compareList;
    this.compareList = [];
    this.compareB = undefined;
    this.dropCompareState();
    for (const e of list) releaseWrapper(e.build);
  }

  GetCompareInfo(): CompareInfo {
    const names = this.compareList.map((e) => e.name);
    return this.compareB
      ? { name: this.compareB.name, seq: this.compareB.seq, names, active: this.compareList.indexOf(this.compareB) }
      : { name: '', seq: 0, names, active: -1 };
  }

  async GetCompareSides(): Promise<{ a: CompareSideData; b: CompareSideData; bName: string; seq: number } | undefined> {
    const b = this.compareB;
    if (!b || !this.currentBuild) return undefined;
    const calc = calculator.NewCalculator(this.currentBuild);
    const fresh = calc?.PoB;
    try {
      const a = fresh ? await this.compareSide(fresh, false) : undefined;
      return a ? { a, b: b.side, bName: b.name, seq: b.seq } : undefined;
    } finally {
      releaseWrapper(fresh);
      calc?.release();
    }
  }

  private async compareSide(build: pob.PathOfBuilding, withOutputs: boolean): Promise<CompareSideData | undefined> {
    const info = build.Build;
    const cfg = build.Config;
    const skills = build.Skills;
    const value = (i: pob.Input) => i.String ?? i.Number ?? i.Boolean ?? undefined;
    const inputs: Record<string, boolean | number | string> = {};
    for (const i of cfg.Inputs ?? []) {
      const v = value(i);
      if (v !== undefined && v !== null) inputs[i.Name] = v;
    }
    const side: CompareSideData = {
      className: info.ClassName,
      ascendClassName: info.AscendClassName,
      level: info.Level,
      mainSocketGroup: info.MainSocketGroup,
      nodes: [...(info.PassiveNodes ?? [])],
      configSet: (cfg.ConfigSets ?? [])[build.ActiveConfigSetIndex()]?.Title || 'Default',
      inputs,
      skills: {
        activeSet: build.ActiveSkillSetIndex(),
        sets: (skills?.SkillSets ?? []).map((set, i) => ({
          title: set.Title || `Skill set ${i + 1}`,
          groups: (set.Skills ?? []).map((g) => ({
            label: g.DisplayLabel || g.Label || '<No active skills>',
            slot: g.Slot || '',
            enabled: g.Enabled,
            inFull: g.IncludeInFullDPS ?? false,
            count: g.GroupCount ?? 1,
            imbued: g.ImbuedSupport,
            gems: (g.Gems ?? []).map((m) => ({
              gemId: m.GemID,
              nameSpec: m.NameSpec,
              skillId: m.SkillID,
              level: m.Level,
              quality: m.Quality,
              enabled: m.Enabled,
              count: m.Count ?? 1
            }))
          }))
        }))
      }
    };
    if (withOutputs) {
      const calc = calculator.NewCalculator(build);
      const out = await calc?.BuildOutput('MAIN');
      const player = out?.Player;
      const mainSkill = player?.MainSkill;
      const minionActor = out?.Minion?.Actor;
      const minionSkill = minionActor?.MainSkill;
      try {
        if (!out || !player) return undefined;
        side.outputs = {
          Output: player.Output,
          SkillFlags: mainSkill?.SkillFlags,
          PointsUsed: out.CountAllocNodes(),
          MainSkill: dump(out.MainSkillControls(), true)
        };
        // CompareTab.lua:2113-2114
        try {
          side.outputs.ConfigRelevance = out.ConfigRelevance(configRelevanceQuery);
        } catch (e) {
          console.error('ConfigRelevance failed', e);
        }
        if (minionActor && minionSkill) {
          side.outputs.Minion = { Output: minionActor.Output, SkillFlags: minionSkill.SkillFlags };
        }
      } finally {
        releaseWrapper(minionSkill);
        releaseWrapper(minionActor);
        releaseWrapper(mainSkill);
        releaseWrapper(player);
        releaseWrapper(out);
        calc?.release();
      }
    }
    return side;
  }

  async GetCompareItems(): Promise<{ a: CompareItem[]; b: CompareItem[] } | undefined> {
    const b = this.compareB;
    if (!b || !this.currentBuild) return undefined;
    const of = async (build: pob.PathOfBuilding): Promise<CompareItem[]> =>
      Promise.all(
        ((await exposition.EquippedItems(build)) ?? []).map(async (e) => {
          const parsed = await exposition.ParseItem(build, e.Text, NO_TIP_STATS);
          const it = parsed.ok ? parsed.value : undefined;
          return { slot: e.SlotName, name: e.Name, rarity: e.Rarity, text: e.Text, line: it?.Explicits?.[0]?.Line ?? it?.BaseName ?? '' };
        })
      );
    return { a: await of(this.currentBuild), b: await of(b.build) };
  }

  private compareMiscCalculator() {
    const b = this.compareB;
    if (!this.compareMisc && b) {
      this.compareMisc = exposition.GetMiscCalculator(b.build).then((calc) => (calc ? { calc, base: calc.Base() } : undefined));
    }
    return this.compareMisc ?? Promise.resolve(undefined);
  }

  async CompareItemInto(receiver: 'a' | 'b', text: string, slot: string): Promise<StatCompareSection[] | undefined> {
    if (receiver === 'a') {
      const r = await this.ItemCompare(text);
      return r?.blocks.find((x) => x.slot === slot)?.sections;
    }
    const seq = this.compareB?.seq;
    const m = await this.compareMiscCalculator();
    if (!m || seq !== this.compareB?.seq) return undefined;
    const slots = (await m.calc.CompareItem(text)).unwrapOr(undefined) ?? [];
    const s = slots.find((x) => x.SlotName === slot);
    if (!s) return undefined;
    const flags = (m.base.SkillFlags ?? {}) as Record<string, boolean>;
    return addStatCompares(flags, { Output: m.base.Output ?? {}, Minion: m.base.Minion }, { Output: s.Output.Output ?? {}, Minion: s.Output.Minion });
  }

  async BuildComparePower(
    receiver: 'a' | 'b',
    stats: string[],
    categories: calculator.ComparePowerCategories,
    configInputs: pob.Input[]
  ): Promise<calculator.ComparePowerEntry[] | undefined> {
    const b = this.compareB;
    const a = this.currentBuild;
    if (!b || !a) return undefined;
    const seq = ++this.comparePowerSeq;
    const tick = this.tickSeq;
    const [primary, other] = receiver === 'a' ? [a, b.build] : [b.build, a];
    const builder = calculator.NewComparePowerBuilder(primary, other, stats, categories, configInputs);
    if (!builder) return undefined;
    const stale = () => seq !== this.comparePowerSeq || this.compareB !== b || this.currentBuild !== a || this.tickSeq !== tick || this.ticksInFlight > 0;
    try {
      for (let done = 0; done < 1; ) {
        if (stale()) return undefined;
        done = builder.StepMs(40);
        await yieldToQueue();
      }
      return stale() ? undefined : [...(builder.Report() ?? [])];
    } finally {
      builder.release();
    }
  }

  CopyCompareConfig() {
    const b = this.compareB;
    if (!b || !this.currentBuild) return;
    for (const input of b.build.Config.Inputs ?? []) {
      this.currentBuild.SetConfigOption(input);
    }
    this.updateStore();
    void this.Tick('CopyCompareConfig');
  }

  UseCompareConfig(key: string) {
    const b = this.compareB;
    if (!b || !this.currentBuild) return;
    const input = (b.build.Config.Inputs ?? []).find((i) => i.Name === key);
    if (input) {
      this.currentBuild.SetConfigOption(input);
    } else {
      this.currentBuild.RemoveConfigOption(key);
    }
    this.updateStore();
    void this.Tick('UseCompareConfig');
  }

  /* Build.lua:255-278 */
  private classChanged(c: calculator.ClassChange | undefined, label: string): ClassChangeResult {
    const out = { changed: !!c?.Changed, needsConfirm: !!c?.NeedsConfirm, className: c?.ClassName ?? '', handled: !!c?.Handled };
    if (out.changed) {
      this.updateStore();
      void this.Tick(label);
    }
    return out;
  }

  /* Build.lua:255-278 */
  async ChangeTreeClass(className: string, decision: 0 | 1 | 2 = 0): Promise<ClassChangeResult> {
    if (!this.currentBuild) return { changed: false, needsConfirm: false, className: '', handled: false };
    return this.classChanged(await exposition.ChangeTreeClass(this.currentBuild, className, decision), 'ChangeTreeClass');
  }

  /* Build.lua:279-284 */
  async ChangeTreeAscendancy(ascendancy: string) {
    if (!this.currentBuild) return;
    await exposition.ChangeTreeAscendancy(this.currentBuild, ascendancy);
    this.updateStore();
    void this.Tick('ChangeTreeAscendancy');
  }

  /* PassiveMasteryControl.lua:37-43, PassiveTreeView.lua:1671-1694 */
  async StatCompareMastery(
    nodeId: number,
    effect: number,
    addNodes: number[],
    nodeCount?: number
  ): Promise<{ sections: StatCompareSection[]; calcMs: number } | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) {
      return undefined;
    }
    this.miscCalls++;
    try {
      const start = performance.now();
      const out = await exposition.CalcMasteryEffect(m.calc, nodeId, effect, addNodes);
      const calcMs = performance.now() - start;
      if (token !== this.misc) {
        return undefined;
      }
      const flags = (m.base.SkillFlags ?? {}) as Record<string, boolean>;
      const sections = addStatCompares(
        flags,
        { Output: m.base.Output ?? {}, Minion: m.base.Minion },
        { Output: out.Output ?? {}, Minion: out.Minion },
        nodeCount
      );
      return { sections, calcMs };
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* TreeTab.lua:63-68 */
  async StatCompareTreeSpec(specId: number): Promise<{ sections: StatCompareSection[]; calcMs: number } | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) {
      return undefined;
    }
    this.miscCalls++;
    try {
      const start = performance.now();
      const out = await exposition.CalcTreeSpec(m.calc, specId);
      const calcMs = performance.now() - start;
      if (token !== this.misc) {
        return undefined;
      }
      const flags = (m.base.SkillFlags ?? {}) as Record<string, boolean>;
      const sections = addStatCompares(flags, { Output: m.base.Output ?? {}, Minion: m.base.Minion }, { Output: out.Output ?? {}, Minion: out.Minion });
      return { sections, calcMs };
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* PassiveSpecListControl.lua:82-87 */
  async MoveTreeSpec(from: number, to: number) {
    if (!this.currentBuild) return;
    await exposition.MoveTreeSpec(this.currentBuild, from, to);
    this.updateStore();
    void this.Tick('MoveTreeSpec');
  }

  /* TreeTab.lua:812-839 */
  async ExportTreeLink(): Promise<string> {
    return this.currentBuild ? ((await exposition.ExportTreeLink(this.currentBuild)) ?? '') : '';
  }

  /* TreeTab.lua:682-810 */
  async ImportTreeLink(title: string, link: string): Promise<string> {
    if (!this.currentBuild) return '';
    const msg = (await exposition.ImportTreeLink(this.currentBuild, title, link)) ?? '';
    if (!msg) {
      this.updateStore();
      void this.Tick('ImportTreeLink');
    }
    return msg;
  }

  /* TreeTab.lua:157-164 */
  GetTreeVersionOptions(): { value: string; label: string }[] {
    return (exposition.GetTreeVersionOptions() ?? []).map((o) => ({ value: o.Value, label: o.Label }));
  }

  /* TreeTab.lua:588-612 */
  async ConvertTreeSpec(version: string, replace: boolean): Promise<string> {
    if (!this.currentBuild) return '';
    const msg = (await exposition.ConvertTreeSpec(this.currentBuild, version, replace)) ?? '';
    this.updateStore();
    void this.Tick('ConvertTreeSpec');
    return msg;
  }

  /* TreeTab.lua:614-627 */
  async ConvertAllTreeSpecs(version: string): Promise<string> {
    if (!this.currentBuild) return '';
    const msg = (await exposition.ConvertAllTreeSpecs(this.currentBuild, version)) ?? '';
    this.updateStore();
    void this.Tick('ConvertAllTreeSpecs');
    return msg;
  }

  /* PassiveTreeView.lua:1157-1246 */
  async GetTreeJewelRings(compared: number[] = []): Promise<{ nodeId: number; outer: number; inner: number; art: string }[]> {
    if (!this.currentBuild) return [];
    return ((await exposition.GetTreeJewelRings(this.currentBuild, compared)) ?? []).map((r) => ({
      nodeId: r.NodeID,
      outer: r.Outer,
      inner: r.Inner,
      art: r.Art
    }));
  }

  /* PassiveTreeView.lua:977-1003, 1213-1234 */
  async GetTreeSocketRadii(nodeId: number): Promise<{ inner: number; outer: number; col: string; nodes: number[] }[]> {
    if (!this.currentBuild) return [];
    return ((await exposition.GetTreeSocketRadii(this.currentBuild, nodeId)) ?? []).map((r) => ({
      inner: r.Inner,
      outer: r.Outer,
      col: r.Col,
      nodes: [...(r.Nodes ?? [])]
    }));
  }

  /* PassiveTreeView.lua:840-846 */
  async GetTreeJewelOverlays(): Promise<{ nodeId: number; art: string }[]> {
    if (!this.currentBuild) return [];
    return ((await exposition.GetTreeJewelOverlays(this.currentBuild)) ?? []).map((o) => ({ nodeId: o.NodeID, art: o.Art }));
  }

  /* TreeTab.lua:133-139, PassiveSpec.lua:756-765 */
  async ResetTreeNodes() {
    if (!this.currentBuild) return;
    await exposition.ResetTreeNodes(this.currentBuild);
    this.updateStore();
    void this.Tick('ResetTreeNodes');
  }

  /* TreeTab.lua:140-149 */
  async RemoveAllTreeTattoos() {
    if (!this.currentBuild) return;
    await exposition.RemoveAllTreeTattoos(this.currentBuild);
    this.updateStore();
    void this.Tick('RemoveAllTreeTattoos');
  }

  /* TreeTab.lua:946-973 */
  async GetTattooCountLines(): Promise<{ line: string; count: number }[]> {
    if (!this.currentBuild) return [];
    return ((await exposition.GetTattooCountLines(this.currentBuild)) ?? []).map((l) => ({ line: l.Line, count: l.Count }));
  }

  /* Build.lua:753-878 */
  async GetLoadouts(): Promise<{ loadouts: string[]; selected: number }> {
    if (!this.currentBuild) return { loadouts: [], selected: -1 };
    const l = await exposition.GetLoadouts(this.currentBuild);
    return { loadouts: [...(l?.Loadouts ?? [])], selected: l?.Selected ?? -1 };
  }

  /* Build.lua:355-414 */
  async SelectLoadout(name: string): Promise<boolean> {
    if (!this.currentBuild) return false;
    const ok = !!(await exposition.SelectLoadout(this.currentBuild, name));
    if (ok) {
      this.updateStore();
      void this.Tick('SelectLoadout');
    }
    return ok;
  }

  /* Build.lua:316-353 */
  async NewLoadout(name: string) {
    if (!this.currentBuild) return;
    await exposition.NewLoadout(this.currentBuild, name);
    this.updateStore();
    void this.Tick('NewLoadout');
  }

  /* Build.lua:211-217 */
  GetCharacterLevelAutoMode(): boolean {
    return this.fresh((b) => !!(b.Build as { CharacterLevelAutoMode?: boolean }).CharacterLevelAutoMode) ?? false;
  }

  /* PassiveTreeView.lua:390-497 */
  async ClickTreeAscendancyNode(nodeId: number, decision: 0 | 1 | 2 = 0): Promise<ClassChangeResult> {
    if (!this.currentBuild) return { changed: false, needsConfirm: false, className: '', handled: false };
    return this.classChanged(await exposition.ClickTreeAscendancyNode(this.currentBuild, nodeId, decision), 'ClickTreeAscendancyNode');
  }

  /* ItemsTab.lua:224-263 */
  async SetWeaponSet(second: boolean) {
    if (!this.currentBuild) return;
    if (await exposition.SetWeaponSet(this.currentBuild, second)) {
      this.updateStore();
      void this.Tick('SetWeaponSet');
    }
  }

  async ItemSetOp(
    op:
      | { op: 'new'; title: string }
      | { op: 'copy' | 'rename'; id: number; title: string }
      | { op: 'delete'; id: number }
      | { op: 'move'; from: number; to: number }
  ) {
    const b = this.currentBuild;
    if (!b) return;
    let source: string;
    switch (op.op) {
      case 'new':
        await exposition.NewItemSet(b, op.title);
        source = 'NewItemSet';
        break;
      case 'copy':
        await exposition.CopyItemSet(b, op.id, op.title);
        source = 'CopyItemSet';
        break;
      case 'rename':
        await exposition.RenameItemSet(b, op.id, op.title);
        source = 'RenameItemSet';
        break;
      case 'delete':
        await exposition.DeleteItemSet(b, op.id);
        source = 'DeleteItemSet';
        break;
      case 'move':
        await exposition.MoveItemSet(b, op.from + 1, op.to + 1);
        source = 'MoveItemSet';
        break;
    }
    this.updateStore();
    void this.Tick(source);
  }

  /* ItemListControl.lua:331-369 */
  async DeleteItemPrompt(itemId: number): Promise<string> {
    return this.currentBuild ? await exposition.DeleteItemPrompt(this.currentBuild, itemId) : '';
  }

  /* ItemsTab.lua:1768-1815 */
  async DeleteItem(itemId: number) {
    if (this.currentBuild && (await exposition.DeleteItem(this.currentBuild, itemId))) {
      this.updateStore();
      void this.Tick('DeleteItem');
    }
  }

  /* ItemListControl.lua:23-42 */
  async DeleteUnusedItems() {
    if (this.currentBuild && (await exposition.DeleteUnusedItems(this.currentBuild))) {
      this.updateStore();
      void this.Tick('DeleteUnusedItems');
    }
  }

  /* ItemListControl.lua:46-65 */
  async DeleteAllItems() {
    if (!this.currentBuild) return;
    await exposition.DeleteAllItems(this.currentBuild);
    this.updateStore();
    void this.Tick('DeleteAllItems');
  }

  /* ItemsTab.lua:1731-1765 */
  async SortItems() {
    if (!this.currentBuild) return;
    await exposition.SortItems(this.currentBuild);
    this.updateStore();
    void this.Tick('SortItems');
  }

  /* ItemListControl.lua:290-314 */
  async ToggleEquipItem(itemId: number, second: boolean) {
    if (this.currentBuild && (await exposition.ToggleEquipItem(this.currentBuild, itemId, second))) {
      this.updateStore();
      void this.Tick('ToggleEquipItem');
    }
  }

  /* ItemSlotControl.lua:98-111, ItemsTab.lua:208-222, 1652-1670 */
  async GetItemsTabSlots(): Promise<{ abyssal: string[]; sockets: { node: number; item: number; choices: number[] }[] }> {
    if (!this.currentBuild) return { abyssal: [], sockets: [] };
    const l = await exposition.ItemsTabSlots(this.currentBuild);
    return {
      abyssal: [...(l.Abyssal ?? [])],
      sockets: (l.Sockets ?? []).map((s) => ({ node: s.NodeID, item: s.ItemID, choices: [...(s.Choices ?? [])] }))
    };
  }

  /* ItemsTab.lua:4580-4592 */
  async SocketJewelTooltip(nodeId: number): Promise<calculator.ItemTooltip | undefined> {
    const m = await this.miscCalculator();
    if (!m) return undefined;
    const [tip, ok] = m.calc.SocketJewelTooltip(nodeId, this.itemTipStats);
    return ok ? tip : undefined;
  }

  /* ItemsTab.lua:2408-2422 */
  async AddedModCompare(text: string, lines: string[], modType: string): Promise<ItemStatDiffBlock | undefined> {
    const m = await this.miscCalculator();
    if (!m) return undefined;
    const [r, ok] = m.calc.CompareAddedMod(text, lines, modType);
    if (!ok) return undefined;
    const flags = (r.Added.SkillFlags ?? {}) as Record<string, boolean>;
    return {
      header: '\nAdding this mod will give: ',
      sections: addStatCompares(flags, { Output: r.Base.Output ?? {}, Minion: r.Base.Minion }, { Output: r.Added.Output ?? {}, Minion: r.Added.Minion })
    };
  }

  /* ItemsTab.lua:2503-2525 */
  async ReplacedImplicitsCompare(text: string, lines: string[]): Promise<ItemStatDiffBlock | undefined> {
    const m = await this.miscCalculator();
    if (!m) return undefined;
    const [r, ok] = m.calc.CompareReplacedImplicits(text, lines);
    if (!ok) return undefined;
    const flags = (r.Added.SkillFlags ?? {}) as Record<string, boolean>;
    return {
      header: '\nAdding this mod will give: ',
      sections: addStatCompares(flags, { Output: r.Base.Output ?? {}, Minion: r.Base.Minion }, { Output: r.Added.Output ?? {}, Minion: r.Added.Minion })
    };
  }

  /* ItemsTab.lua:378-715 */
  async GetDisplayItem(text: string): Promise<DisplayInfo | undefined> {
    const [d, ok] = await exposition.DisplayItem(text);
    if (!ok) return undefined;
    return {
      quality: d.Quality,
      hasQuality: d.HasQuality,
      variants: [...(d.Variants ?? [])],
      variant: d.Variant,
      altVariants: [...(d.AltVariants ?? [])],
      sockets: d.Sockets,
      socketLimit: d.SocketLimit,
      defaultColor: d.DefaultColor,
      canInfluence: d.CanInfluence,
      influence: [...(d.Influence ?? [])],
      showsCatalyst: d.ShowsCatalyst,
      catalyst: d.Catalyst,
      catalystQuality: d.CatalystQuality,
      mods: (d.Mods ?? []).map((m) => ({
        line: m.Line,
        kind: m.Kind.toLowerCase(),
        hasRange: m.HasRange,
        range: m.Range,
        disabled: m.Disabled,
        removable: m.Removable,
        removal: m.Removal
      }))
    };
  }

  async EditItemText(text: string, edit: DisplayEdit): Promise<{ text?: string; error?: string }> {
    const r = await exposition.EditItem(text, {
      Kind: edit.kind,
      Index: edit.index ?? 0,
      Value: edit.value ?? 0,
      Second: edit.second ?? 0,
      Float: edit.float ?? 0,
      Text: edit.text ?? ''
    });
    return r.ok ? { text: r.value } : { error: String(r.error) };
  }

  /* Data.lua:1224-1254 */
  async GetCraftBases(): Promise<{ type: string; bases: string[] }[]> {
    return ((await exposition.CraftBaseGroups()) ?? []).map((g) => ({ type: g.Type, bases: [...(g.Bases ?? [])] }));
  }

  /* ItemsTab.lua:2521-2565 */
  async CraftItemText(c: CraftChoice): Promise<string | undefined> {
    const [text, ok] = await exposition.CraftItemText(c.rarity, c.title, c.base);
    return ok ? text : undefined;
  }

  /* SkillsTab.lua:500-508 */
  GetGemDefaultLevelMode(): string {
    return this.currentBuild ? exposition.GemDefaultLevelMode(this.currentBuild) : 'normalMaximum';
  }

  /* SkillsTab.lua:1102-1130 */
  GemDefaultLevel(gemID: string): number {
    return this.currentBuild ? exposition.GemDefaultLevel(this.currentBuild, gemID) : 0;
  }

  private async gemWhatIf(socketGroup: number, gemIndex: number, gem: pob.Gem): Promise<StatCompareSection[] | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) {
      return undefined;
    }
    this.miscCalls++;
    try {
      const out = await exposition.CalcGemStats(m.calc, socketGroup, gemIndex, gem);
      if (token !== this.misc) {
        return undefined;
      }
      const flags = (m.base.SkillFlags ?? {}) as Record<string, boolean>;
      return addStatCompares(flags, { Output: m.base.Output ?? {}, Minion: m.base.Minion }, { Output: out.Output ?? {}, Minion: out.Minion });
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* GemSelectControl.lua:582-603 */
  async GemSelectCompare(socketGroup: number, gemID: string, nameSpec: string): Promise<StatCompareSection[] | undefined> {
    const build = this.currentBuild;
    if (!build) return undefined;
    const count = this.fresh((b) => b.Skills.SkillSets[b.ActiveSkillSetIndex()]?.Skills?.[socketGroup]?.Gems?.length);
    if (count === undefined) return undefined;
    const level = exposition.GemDefaultLevel(build, gemID) || 1;
    const quality = this.fresh((b) => b.Skills.DefaultGemQuality) ?? 0;
    return this.gemWhatIf(socketGroup, count, exposition.DefaultGem(gemID, nameSpec, gemID, level, quality));
  }

  /* SkillsTab.lua:911-918 */
  async GemQualityCompare(socketGroup: number, gemIndex: number): Promise<StatCompareSection[] | undefined> {
    const build = this.currentBuild;
    if (!build) return undefined;
    const gems = this.fresh((b) => b.Skills.SkillSets[b.ActiveSkillSetIndex()]?.Skills?.[socketGroup]?.Gems) ?? [];
    const gem = gems[gemIndex];
    if (!gem) return undefined;
    return this.gemWhatIf(socketGroup, gemIndex, { ...gem, Quality: 20 });
  }

  /* Build.lua:1453-1482 */
  GetSpectreLibrary(): { library: calculator.SpectreRow[]; build: calculator.SpectreRow[] } {
    const ids = this.currentBuild?.SpectreIDs() ?? [];
    return { library: exposition.SpectreLibrary() ?? [], build: exposition.SpectreRows(ids) ?? [] };
  }

  /* Build.lua:1469-1474 */
  SetSpectres(ids: string[]) {
    if (!this.currentBuild) return;
    this.currentBuild.SetSpectres(ids);
    this.updateStore();
    void this.Tick('SetSpectres');
  }

  /* ConfigTab.lua:721-766 */
  async ConfigCompare(key: string, value: boolean | number | string): Promise<StatCompareSection[] | undefined> {
    const v = reverseConfigOptions[key];
    if (!v) return undefined;
    const remove =
      v.type === 'list'
        ? value === v.list[(v.defaultIndex ?? 1) - 1].value
        : v.type === 'check'
          ? value === (v.defaultState ?? false)
          : v.type === 'text'
            ? value === ''
            : value === null;
    const input: pob.Input =
      typeof value === 'boolean' ? { Name: key, Boolean: value } : typeof value === 'string' ? { Name: key, String: value } : { Name: key, Number: value };
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) return undefined;
    this.miscCalls++;
    try {
      const out = await exposition.CalcConfigStats(m.calc, input, remove);
      if (token !== this.misc) return undefined;
      const flags = (m.base.SkillFlags ?? {}) as Record<string, boolean>;
      return addStatCompares(flags, { Output: m.base.Output ?? {}, Minion: m.base.Minion }, { Output: out.Output ?? {}, Minion: out.Minion });
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* ConfigSetListControl.lua:13-110 */
  EditConfigSets(
    op:
      | { op: 'new'; title: string }
      | { op: 'copy' | 'rename'; id: number; title: string }
      | { op: 'delete'; id: number }
      | { op: 'move'; from: number; to: number }
  ) {
    const b = this.currentBuild;
    if (!b) return;
    switch (op.op) {
      case 'new':
        b.NewConfigSet(op.title);
        break;
      case 'copy':
        b.CopyConfigSet(op.id, op.title);
        break;
      case 'rename':
        b.RenameConfigSet(op.id, op.title);
        break;
      case 'delete':
        b.DeleteConfigSet(op.id);
        break;
      case 'move':
        b.MoveConfigSet(op.from, op.to);
        break;
    }
    this.updateStore();
    void this.Tick(`EditConfigSets: ${op.op}`);
  }

  /* SkillsTab.lua:185-198 */
  async GetSocketedInItemNames(slotNames: string[]): Promise<Record<string, string>> {
    return this.currentBuild ? ((await exposition.SocketedInItemNames(this.currentBuild, slotNames)) ?? {}) : {};
  }

  /* CalcBreakdownControl.lua:400-403 */
  async ItemSourceTooltip(name: string): Promise<calculator.ItemTooltip | undefined> {
    if (!this.currentBuild) return undefined;
    const [tip, ok] = await exposition.ItemSourceTooltip(this.currentBuild, name, this.itemTipStats);
    return ok ? tip : undefined;
  }

  /* Build.lua:99-106, 1325-1350 */
  GetVersionConversion(): string | undefined {
    return this.currentBuild?.NeedsVersionConversion() ? exposition.CurrentGameVersionLabel() : undefined;
  }

  /* Build.lua:1340-1344 */
  ConvertBuildVersion() {
    this.currentBuild?.ConvertToLiveVersion();
    this.updateStore();
  }

  /* TreeTab.lua:69-97 */
  TreeSpecRespecCost(index: number): calculator.TreeSpecRespec {
    return this.currentBuild ? exposition.TreeSpecRespecCost(this.currentBuild, index) : { Gold: 0, Passives: 0, Ascendancy: 0 };
  }

  /* TreeTab.lua:260-265 */
  GetNodePowerStats(): calculator.PowerStat[] {
    return [...(exposition.NodePowerStats() ?? [])];
  }

  private calcsEnv?: calculator.Environment;

  CalcsModSection(sd: calculator.ModSectionData): calculator.ModSectionRow[] {
    const showMinion = this.fresh((b) => b.CalcsInputBool('showMinion')) ?? false;
    return [...(this.calcsEnv?.CalcsModSection(sd, showMinion) ?? [])];
  }

  CalcsBreakdownModList(name: string): calculator.ModSectionRow[] {
    const showMinion = this.fresh((b) => b.CalcsInputBool('showMinion')) ?? false;
    return [...(this.calcsEnv?.CalcsBreakdownModList(name, showMinion) ?? [])];
  }

  /* CalcSetup.lua:1593, 1684 */
  private async gemDisplay(socketGroup: number, gemIndex: number): Promise<calculator.GemDisplay | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) return undefined;
    this.miscCalls++;
    try {
      return (exposition.MiscGemDisplays(m.calc) ?? []).find((d) => d.Group === socketGroup && d.Gem === gemIndex)?.Display;
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  private gemTooltipLines(gem: pob.Gem, display: calculator.GemDisplay | undefined): calculator.TooltipLine[] {
    const none: calculator.GemDisplay = { Level: 0, Quality: 0, SocketQuality: 0, ItemQuality: 0, SupportQuality: 0, GlobalQuality: 0 };
    return [...(exposition.GemTooltip(this.currentBuild, gem, display ?? none, display !== undefined, this.itemTipStats) ?? [])];
  }

  /* GemSelectControl.lua:629-646 */
  async GemTooltip(socketGroup: number, gemIndex: number): Promise<calculator.TooltipLine[]> {
    const gem = this.fresh((b) => b.Skills.SkillSets[b.ActiveSkillSetIndex()]?.Skills?.[socketGroup]?.Gems?.[gemIndex]);
    if (!gem || !this.currentBuild) return [];
    return this.gemTooltipLines(gem, await this.gemDisplay(socketGroup, gemIndex));
  }

  /* GemSelectControl.lua:587-601 */
  GemListTooltip(gemID: string, nameSpec: string): calculator.TooltipLine[] {
    const build = this.currentBuild;
    if (!build) return [];
    const level = exposition.GemDefaultLevel(build, gemID) || 1;
    const quality = this.fresh((b) => b.Skills.DefaultGemQuality) ?? 0;
    return this.gemTooltipLines(exposition.DefaultGem(gemID, nameSpec, gemID, level, quality), undefined);
  }

  /* SkillsTab.lua:1283-1368 */
  async SocketGroupTooltip(socketGroup: number): Promise<calculator.TooltipLine[]> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) return [];
    this.miscCalls++;
    try {
      return [...(exposition.MiscSocketGroupTooltip(m.calc, socketGroup) ?? [])];
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* GemSelectControl.lua:343, 419 */
  private gemSortDPS(field: string, out: { Output: Record<string, number>; Minion?: Record<string, number> }): number {
    if (field === 'FullDPS' && out.Output.FullDPS !== undefined) return out.Output.FullDPS;
    if (out.Minion && out.Minion.CombinedDPS !== undefined) return out.Minion.CombinedDPS;
    return out.Output[field] ?? 0;
  }

  /* GemSelectControl.lua:71-86 */
  private pickerGems(gems: { id: string; name: string }[]): pob.Gem[] {
    const build = this.currentBuild;
    const quality = this.fresh((b) => b.Skills.DefaultGemQuality) ?? 0;
    return gems.map((g) => exposition.DefaultGem(g.id, g.name, g.id, (build && exposition.GemDefaultLevel(build, g.id)) || 1, quality));
  }

  /* GemSelectControl.lua:294-359 */
  async GemSortCache(socketGroup: number, gems: { id: string; name: string }[]): Promise<{ canSupport: boolean[]; pending: boolean[] } | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) return undefined;
    const rows = exposition.MiscGemSortCache(m.calc, socketGroup, this.pickerGems(gems)) ?? [];
    return { canSupport: rows.map((r) => r.CanSupport), pending: rows.map((r) => r.Pending) };
  }

  /* GemSelectControl.lua:404-441 */
  async GemSortDPS(socketGroup: number, gems: { id: string; name: string }[], field: string): Promise<{ base: number; dps: number[] } | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    const count = this.fresh((b) => b.Skills.SkillSets[b.ActiveSkillSetIndex()]?.Skills?.[socketGroup]?.Gems?.length);
    if (!m || token !== this.misc || count === undefined) return undefined;
    this.miscCalls++;
    try {
      const dps: number[] = [];
      for (const gem of this.pickerGems(gems)) {
        const out = await exposition.CalcGemStats(m.calc, socketGroup, count, gem);
        if (token !== this.misc) return undefined;
        dps.push(this.gemSortDPS(field, { Output: out.Output ?? {}, Minion: out.Minion }));
      }
      return { base: this.gemSortDPS(field, { Output: m.base.Output ?? {}, Minion: m.base.Minion }), dps };
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* ConfigTab.lua:53-68 */
  async CustomModGroupCompare(group: number): Promise<StatCompareSection[] | undefined> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) return undefined;
    this.miscCalls++;
    try {
      const out = await exposition.CalcCustomModGroupStats(m.calc, group);
      if (token !== this.misc) return undefined;
      const flags = (m.base.SkillFlags ?? {}) as Record<string, boolean>;
      return addStatCompares(flags, { Output: m.base.Output ?? {}, Minion: m.base.Minion }, { Output: out.Output ?? {}, Minion: out.Minion });
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* SkillsTab.lua:868-909 */
  GemQualityTooltip(socketGroup: number, gemIndex: number): calculator.TooltipLine[] {
    const gem = this.fresh((b) => b.Skills.SkillSets[b.ActiveSkillSetIndex()]?.Skills?.[socketGroup]?.Gems?.[gemIndex]);
    return gem ? [...(exposition.GemQualityTooltip(gem) ?? [])] : [];
  }

  /* ItemsTab.lua:509-553, 2875-2890 */
  async GetDisplayItemCraft(
    text: string
  ): Promise<Pick<DisplayInfo, 'enchants' | 'anoints' | 'anointed' | 'corruptible' | 'addImplicit' | 'addCustom' | 'addCrucible' | 'cluster'> | undefined> {
    const [d, ok] = await exposition.DisplayItem(text);
    const c = d.Cluster;
    return ok
      ? {
          enchants: d.Enchants,
          anoints: d.Anoints,
          anointed: [...(d.Anointed ?? [])],
          corruptible: d.Corruptible,
          addImplicit: d.AddImplicit,
          addCustom: d.AddCustom,
          addCrucible: d.AddCrucible,
          cluster: c && {
            skills: (c.Skills ?? []).map((x) => ({ id: x.ID, label: x.Label })),
            skill: c.Skill,
            nodeCount: c.NodeCount,
            min: c.Min,
            max: c.Max
          }
        }
      : undefined;
  }

  /* ItemsTab.lua:2665-2706 */
  async GetEnchantChoices(text: string): Promise<EnchantChoices | undefined> {
    const [c, ok] = await exposition.ItemEnchantChoices(this.currentBuild, text);
    if (!ok) return undefined;
    return {
      haveSkills: c.HaveSkills,
      skills: (c.Skills ?? []).map((s) => ({
        skill: s.Skill,
        sources: (s.Sources ?? []).map((x) => ({ name: x.Name, label: x.Label, lines: [...(x.Lines ?? [])] }))
      })),
      used: [...(c.Used ?? [])]
    };
  }

  private anointNotables?: AnointNotable[];

  async GetAnointNotables(): Promise<AnointNotable[]> {
    this.anointNotables ??= ((await exposition.AnointNotables()) ?? []).map((n) => ({
      id: n.ID,
      name: n.Name,
      stats: [...(n.Stats ?? [])],
      reminder: [...(n.ReminderText ?? [])],
      recipe: [...(n.Recipe ?? [])],
      unsupported: [...(n.Unsupported ?? [])]
    }));
    return this.anointNotables;
  }

  /* ItemsTab.lua:2754-2779 */
  async EnchantSortValues(text: string, slot: number, lines: string[], stat: string): Promise<number[]> {
    const m = await this.miscCalculator();
    return m ? [...(m.calc.EnchantSortValues(text, slot, lines, stat) ?? [])] : lines.map(() => 0);
  }

  /* NotableDBControl.lua:127-145 */
  async AnointPowers(text: string, slot: number, ids: number[], stat: string): Promise<number[]> {
    const m = await this.miscCalculator();
    return m ? [...(m.calc.AnointPowers(text, slot, ids, stat) ?? [])] : ids.map(() => 0);
  }

  /* ItemsTab.lua:2925-2963 */
  async AnointCompare(text: string, slot: number, id: number): Promise<{ note?: string; block?: ItemStatDiffBlock } | undefined> {
    const m = await this.miscCalculator();
    if (!m) return undefined;
    const [r, ok] = m.calc.CompareAnoint(text, slot, id);
    if (!ok) return undefined;
    if (r.Note) return { note: r.Note };
    const flags = (r.New.SkillFlags ?? {}) as Record<string, boolean>;
    const sections = addStatCompares(flags, { Output: r.Base.Output ?? {}, Minion: r.Base.Minion }, { Output: r.New.Output ?? {}, Minion: r.New.Minion });
    if (id && !sections.some((s) => s.lines.length)) return { note: r.Header.replace(' will give you: ', ' changes nothing.') };
    return { block: { header: r.Header.replace(/^\^7/, ''), sections } };
  }

  /* ItemsTab.lua:3018-3212 */
  async GetCorruptChoices(text: string): Promise<CorruptChoices | undefined> {
    const [c, ok] = await exposition.ItemCorruptChoices(text);
    if (!ok) return undefined;
    const mods = (l?: calculator.CorruptMod[]) =>
      (l ?? []).map((m) => ({ id: m.ID, label: m.Label, lines: [...(m.Lines ?? [])], group: m.Group, modTags: [...(m.ModTags ?? [])] }));
    return {
      unique: c.Unique,
      implicitsCannotBeChanged: c.ImplicitsCannotBeChanged,
      maxImplicits: c.MaxImplicits,
      glimpse: mods(c.Glimpse),
      corrupted: mods(c.Corrupted),
      scourgeUpside: mods(c.ScourgeUpside),
      scourgeDownside: mods(c.ScourgeDownside),
      rolls: (c.Rolls ?? []).map((r) => ({ index: r.Index, line: r.Line, value: r.Value }))
    };
  }

  /* ItemsTab.lua:3182-3186 */
  async CorruptedRangeLine(text: string, index: number, value: number): Promise<string> {
    return await exposition.CorruptedRangeLine(text, index, value);
  }

  /* ItemsTab.lua:3136-3175 */
  async CorruptItemText(
    text: string,
    source: string,
    ids: string[],
    downside: string[],
    rolls?: { index: number; value: number }[]
  ): Promise<string | undefined> {
    const [t, ok] = await exposition.CorruptItemText(
      text,
      source,
      ids,
      downside,
      !!rolls,
      (rolls ?? []).map((r) => r.index),
      (rolls ?? []).map((r) => r.value)
    );
    return ok ? t : undefined;
  }

  /* ItemsTab.lua:3795-3932 */
  async GetImplicitChoices(text: string): Promise<ImplicitChoices | undefined> {
    const [c, ok] = await exposition.ItemImplicitChoices(text);
    if (!ok) return undefined;
    const groups: ImplicitChoices['groups'] = {};
    for (const [id, list] of Object.entries(c.Groups ?? {})) {
      groups[id] = (list ?? []).map((g) => ({
        label: g.Label,
        mods: (g.Mods ?? []).map((m) => ({ id: m.ID, label: m.Label, lines: [...(m.Lines ?? [])], modTags: [...(m.ModTags ?? [])] }))
      }));
    }
    return { sources: (c.Sources ?? []).map((s) => ({ id: s.ID, label: s.Label })), groups };
  }

  /* ItemsTab.lua:4041-4056 */
  async ImplicitText(text: string, source: string, modId: string, custom: string): Promise<string | undefined> {
    const [t, ok] = await exposition.ImplicitText(text, source, modId, custom);
    return ok ? t : undefined;
  }

  /* ItemsTab.lua:3957-3970 */
  async ImplicitSortValues(text: string, source: string, ids: string[], stat: string): Promise<number[]> {
    const m = await this.miscCalculator();
    return m ? [...(m.calc.ImplicitSortValues(text, source, ids, stat) ?? [])] : ids.map(() => 0);
  }

  /* ItemsTab.lua:3409-3589 */
  async GetCustomModChoices(text: string): Promise<CustomModChoices | undefined> {
    const [c, ok] = await exposition.ItemCustomModChoices(text);
    if (!ok) return undefined;
    const mods: CustomModChoices['mods'] = {};
    for (const [id, list] of Object.entries(c.Mods ?? {})) {
      mods[id] = (list ?? []).map((m) => ({ id: m.ID, label: m.Label, lines: [...(m.Lines ?? [])], modTags: [...(m.ModTags ?? [])] }));
    }
    return { sources: (c.Sources ?? []).map((s) => ({ id: s.ID, label: s.Label })), mods };
  }

  /* ItemsTab.lua:3591-3609 */
  async CustomModText(text: string, source: string, modId: string, custom: string): Promise<string | undefined> {
    const [t, ok] = await exposition.CustomModText(text, source, modId, custom);
    return ok ? t : undefined;
  }

  /* ItemsTab.lua:3356-3371 */
  async CustomModSortValues(text: string, source: string, ids: string[], stat: string): Promise<number[]> {
    const m = await this.miscCalculator();
    return m ? [...(m.calc.CustomModSortValues(text, source, ids, stat) ?? [])] : ids.map(() => 0);
  }

  /* ItemsTab.lua:2108-2332 */
  async GetAffixControls(text: string): Promise<AffixControls | undefined> {
    const [c, ok] = await exposition.ItemAffixControls(text);
    if (!ok) return undefined;
    return {
      sorting: c.Sorting,
      controls: (c.Controls ?? []).map((x) => ({
        type: x.Type,
        series: (x.Series ?? []).map((s) => ({
          label: s.Label,
          tiers: (s.ModList ?? []).length,
          haveRange: s.HaveRange,
          tooltip: [...(s.Tooltip ?? [])],
          mid: [...(s.Mid ?? [])]
        })),
        sel: x.Sel,
        sliderShown: x.SliderShown,
        sliderVal: x.SliderVal
      }))
    };
  }

  /* ItemsTab.lua:796-811, 931-939 */
  async AffixEditText(text: string, slot: number, series: number, slider: number): Promise<string | undefined> {
    const [t, ok] = await exposition.AffixEditText(text, slot, series, slider);
    return ok ? t : undefined;
  }

  /* ItemsTab.lua:943-968 */
  async AffixRollTooltip(text: string, slot: number, slider: number): Promise<string[]> {
    return [...((await exposition.AffixRollTooltip(text, slot, slider)) ?? [])];
  }

  /* ItemsTab.lua:2219-2297 */
  async AffixPowers(text: string, slot: number, stat: string): Promise<number[]> {
    const m = await this.miscCalculator();
    return m ? [...(m.calc.AffixPowers(text, slot, stat) ?? [])] : [];
  }

  /* ItemsTab.lua:3664-3758 */
  async GetCrucibleChoices(text: string): Promise<CrucibleChoices | undefined> {
    const [c, ok] = await exposition.ItemCrucibleChoices(text);
    if (!ok) return undefined;
    return {
      nodes: (c.Nodes ?? []).map((n) => (n ?? []).map((e) => ({ id: e.ID, label: e.Label, lines: [...(e.Lines ?? [])] }))),
      selected: [...(c.Selected ?? [])]
    };
  }

  /* ItemsTab.lua:3710-3730 */
  async CrucibleText(text: string, ids: string[]): Promise<string | undefined> {
    const [t, ok] = await exposition.CrucibleText(text, ids);
    return ok ? t : undefined;
  }

  /* ItemsTab.lua:3064-3088 */
  async CorruptSortValues(text: string, source: string, modType: string, ids: string[], stat: string): Promise<number[]> {
    const m = await this.miscCalculator();
    return m ? [...(m.calc.CorruptSortValues(text, source, modType, ids, stat) ?? [])] : ids.map(() => 0);
  }

  /* ItemDBControl.lua:27-56, 61-79, 205-229 */
  async GetItemDBLists(kind: string): Promise<ItemDBLists> {
    const i = await exposition.ItemDBLists(kind);
    return {
      slots: [...(i.Slots ?? [])],
      types: [...(i.Types ?? [])],
      leagues: [...(i.Leagues ?? [])],
      requirements: [...(i.Requirements ?? [])],
      obtainable: [...(i.Obtainable ?? [])],
      searchModes: [...(i.SearchModes ?? [])],
      sortStats: (i.SortStats ?? []).map((s) => ({ stat: s.Stat, label: s.Label }))
    };
  }

  /* ItemDBControl.lua:231-285 */
  async GetItemDBRows(kind: string, filter: ItemDBFilterInput, stat: string): Promise<ItemDBRowPlain[]> {
    const m = await this.miscCalculator();
    if (!m || !this.currentBuild) return [];
    const level = this.fresh((b) => b.Build.Level) ?? 1;
    const f = {
      Slot: filter.slot,
      Type: filter.type,
      League: filter.league,
      Requirement: filter.requirement,
      Obtainable: filter.obtainable,
      Search: filter.search,
      SearchMode: filter.searchMode
    };
    const t = this.itemTipStats;
    return (m.calc.ItemDBRows(kind, f, level, t.Str, t.Dex, t.Int, stat) ?? []).map((r) => ({
      name: r.Name,
      title: r.Title,
      base: r.BaseName,
      rarity: r.Rarity,
      type: r.Type,
      slot: r.PrimarySlot,
      level: r.Level,
      league: r.League,
      source: r.Source,
      power: r.Power
    }));
  }

  async GetItemDBTooltip(kind: string, name: string): Promise<calculator.ItemTooltip | undefined> {
    const [tip, ok] = await exposition.ItemDBTooltip(this.currentBuild, kind, name, this.itemTipStats);
    return ok ? tip : undefined;
  }

  async GetItemDBText(kind: string, name: string): Promise<string | undefined> {
    const [t, ok] = await exposition.ItemDBText(kind, name);
    return ok ? t : undefined;
  }

  /* ItemDBControl.lua:371-373 */
  async GetItemDBRaw(kind: string, name: string): Promise<string | undefined> {
    const [t, ok] = await exposition.ItemDBRaw(kind, name);
    return ok ? t : undefined;
  }

  /* ItemsTab.lua:1034-1070, ItemsTab.lua:378-452, 1887-1938 */
  async GetDisplayItemExtras(text: string): Promise<Pick<DisplayInfo, 'foulborn' | 'variantGroups'> | undefined> {
    const [d, ok] = await exposition.DisplayItem(text);
    if (!ok) return undefined;
    const vg = d.VariantGroups;
    return {
      foulborn: (d.Mods ?? []).map((m) => ({ mutable: m.Mutable, mutated: m.Mutated })),
      variantGroups: vg && {
        versions: [...(vg.Versions ?? [])],
        version: vg.Version,
        groups: (vg.Groups ?? []).map((g) => ({
          group: g.Group,
          options: (g.Options ?? []).map((o) => ({ variant: o.Variant, label: o.Label })),
          selected: g.Selected,
          enabled: g.Enabled
        }))
      }
    };
  }

  /* ItemsTab.lua:166-174 */
  async GetConditionalSlots(): Promise<string[]> {
    const m = await this.miscCalculator();
    return m ? [...(m.calc.ConditionalSlots() ?? [])] : [];
  }

  /* ItemSlotControl.lua:114-130 */
  async DropItemOnSlot(slotName: string, id: number, kind: string, name: string): Promise<boolean> {
    if (!this.currentBuild || !(await exposition.DropItemOnSlot(this.currentBuild, slotName, id, kind, name))) return false;
    this.updateStore();
    void this.Tick('DropItemOnSlot');
    return true;
  }

  /* ItemsTab.lua:1613-1642 */
  async EquipItemInSet(setId: number, id: number, kind: string, name: string, second: boolean): Promise<boolean> {
    if (!this.currentBuild || !(await exposition.EquipItemInSet(this.currentBuild, setId, id, kind, name, second))) return false;
    this.updateStore();
    void this.Tick('EquipItemInSet');
    return true;
  }

  /* ItemListControl.lua:77-183 */
  async GetItemListFilter(filter: string): Promise<{ filters: string[]; items: number[] }> {
    if (!this.currentBuild) return { filters: [], items: [] };
    const [filters, items] = await Promise.all([exposition.ItemListLoadoutFilters(this.currentBuild), exposition.ItemListFiltered(this.currentBuild, filter)]);
    return { filters: [...(filters ?? [])], items: [...(items ?? [])] };
  }

  async GetSharedItemSets(): Promise<{ title: string; tip: string[] }[]> {
    return ((await exposition.SharedItemSets()) ?? []).map((s) => ({ title: s.Title, tip: [...(s.Tip ?? [])] }));
  }

  /* SharedItemSetListControl.lua:76-93 */
  async ShareItemSet(setId: number): Promise<boolean> {
    return this.currentBuild ? exposition.ShareItemSet(this.currentBuild, setId, Number.MAX_SAFE_INTEGER) : false;
  }

  /* ItemSetListControl.lua:93-106 */
  async ImportSharedItemSet(index: number): Promise<number> {
    if (!this.currentBuild) return 0;
    const id = await exposition.ImportSharedItemSet(this.currentBuild, index);
    if (id) {
      this.updateStore();
      void this.Tick('ImportSharedItemSet');
    }
    return id;
  }

  /* SharedItemSetListControl.lua:15-45, 96-104 */
  async DeleteSharedItemSet(index: number): Promise<boolean> {
    return exposition.DeleteSharedItemSet(index);
  }

  async RenameSharedItemSet(index: number, title: string): Promise<boolean> {
    return exposition.RenameSharedItemSet(index, title);
  }

  /* ItemsTab.lua:4138-4147 */
  async GetItemSetTooltips(): Promise<Record<string, string[]>> {
    const tips = this.currentBuild ? ((await exposition.ItemSetTooltips(this.currentBuild)) ?? []) : [];
    return Object.fromEntries(tips.map((t) => [t.ID, [...(t.Lines ?? [])]]));
  }

  /* ItemDBControl.lua:335-360, ItemListControl.lua:275-284 */
  async AddDBItem(kind: string, name: string, equip: boolean, second: boolean): Promise<boolean> {
    if (!this.currentBuild) return false;
    const [, ok] = await exposition.AddDBItem(this.currentBuild, kind, name, equip, second);
    if (!ok) return false;
    this.updateStore();
    void this.Tick('AddDBItem');
    return true;
  }

  /* CompareTab.lua:4655-4662 */
  async GetCompareCalcs(): Promise<{ a: Outputs; b: Outputs; inputs: { a: CompareCalcsInputs; b: CompareCalcsInputs } } | undefined> {
    const b = this.compareB;
    if (!b || !this.currentBuild) return undefined;
    // CompareCalcsHelpers.lua:170-176
    const envs: calculator.Environment[] = [];
    const outA = await this.calcsTabOutputs(this.currentBuild, (e) => (envs[0] = e));
    const outB = await this.calcsTabOutputs(b.build, (e) => (envs[1] = e));
    for (const e of this.compareCalcsEnvs) releaseWrapper(e);
    this.compareCalcsEnvs = envs;
    this.compareCalcsControls = outB?.MainSkill;
    return outA && outB ? { a: outA, b: outB, inputs: { a: this.compareCalcsInputs(this.currentBuild), b: this.compareCalcsInputs(b.build) } } : undefined;
  }

  /* PassiveSpec.lua:1885-2271 */
  async GetTreeClusterNodes(): Promise<TreeClusterNode[]> {
    if (!this.currentBuild) return [];
    return ((await exposition.GetTreeClusterNodes(this.currentBuild)) ?? []).map((n) => ({
      id: n.ID,
      type: n.Type as TreeClusterNode['type'],
      name: n.Name,
      icon: n.Icon,
      stats: [...(n.Stats ?? [])],
      group: n.Group,
      orbit: n.Orbit,
      orbitIndex: n.OrbitIndex,
      x: n.X,
      y: n.Y,
      linked: [...(n.Linked ?? [])]
    }));
  }

  /* ConfigTab.lua:1450-1489 */
  ModBrowserList(search: string): string[] {
    return [...(exposition.ModBrowserList(search) ?? [])];
  }

  /* CompareTab.lua:1331-1368 */
  CopyCompareSpec(andUse: boolean) {
    const b = this.compareB;
    if (!b || !this.currentBuild) return;
    this.currentBuild.CopyCompareSpec(b.build, andUse);
    this.updateStore();
    void this.Tick('CopyCompareSpec');
  }

  /* CompareTab.lua:1439-1459 */
  CopyCompareItem(text: string) {
    if (!this.currentBuild) return;
    this.currentBuild.AddItem(text);
    this.updateStore();
    void this.Tick('CopyCompareItem');
  }

  /* BuildList.lua:145 */
  async CopyFolder(from: string, to: string) {
    await this.NewFolder(to);
    for (const e of ((await this.ListBuilds(from)) ?? []) as { Name: string; Type: string }[]) {
      if (e.Type === 'dir') await this.CopyFolder(`${from}/${e.Name}`, `${to}/${e.Name}`);
      else await this.Copy(`${from}/${e.Name}`, `${to}/${e.Name}`);
    }
  }

  /* BuildList.lua:160 */
  async MoveFolder(from: string, to: string) {
    await this.CopyFolder(from, to);
    await this.Delete(from);
  }

  /* ConfigModBrowser.lua:243-250 */
  ModBrowserSources(line: string): string[] {
    return [...(exposition.ModBrowserSources(line) ?? [])];
  }

  /* ConfigTab.lua:1496-1505 */
  ModBrowserSupported(line: string): boolean {
    return exposition.ModBrowserSupported(line);
  }

  GetParty(): PartyView | undefined {
    const b = this.currentBuild;
    if (!b) return undefined;
    const summaries = exposition.GetPartySummaries(b) ?? {};
    const [destination, append, advanced] = b.PartyOptions();
    return {
      buffers: { ...(b.PartyBuffers() ?? {}) },
      summaries: Object.fromEntries(Object.entries(summaries).map(([k, v]) => [k, (v ?? []).map((e) => ({ group: e.Group, name: e.Name, value: e.Value }))])),
      destination,
      append,
      advanced,
      applied: b.PartyEffectsApplied(),
      disabled: b.PartyEffectsDisabled()
    };
  }

  async GetPartyImpact(stats: string[]): Promise<{ stat: string; with: number; without: number }[]> {
    const b = this.currentBuild;
    if (!b) return [];
    const read = async (build: pob.PathOfBuilding | undefined) => {
      const calc = build && calculator.NewCalculator(build);
      const env = calc ? await calc.BuildOutput('MAIN') : undefined;
      const out = { ...(env?.Player?.Output ?? {}) };
      releaseWrapper(env);
      releaseWrapper(calc);
      return out;
    };
    const without = b.WithoutPartyEffects();
    const [a, z] = [await read(b), await read(without)];
    releaseWrapper(without);
    return stats.map((stat) => ({ stat, with: a[stat] ?? 0, without: z[stat] ?? 0 }));
  }

  private partyWrite(source: string, write: (b: pob.PathOfBuilding) => void) {
    if (!this.currentBuild) return;
    write(this.currentBuild);
    this.updateStore();
    void this.Tick(source);
  }

  SetPartyBuffer(name: string, text: string) {
    this.partyWrite('SetPartyBuffer', (b) => b.SetPartyBuffer(name, text));
  }

  SetPartyOptions(destination: string, append: boolean, advanced: boolean) {
    this.partyWrite('SetPartyOptions', (b) => b.SetPartyOptions(destination, append, advanced));
  }

  /* PartyTab.lua:298-325 */
  ImportParty(code: string, destination: string, append: boolean) {
    const xml = pob.DecodeDecompress(code).unwrap();
    let err: Error | undefined;
    this.partyWrite('ImportParty', (b) => {
      try {
        b.ImportParty(xml, destination, append).unwrap();
      } catch (e) {
        err = e as Error;
      }
    });
    if (err) throw err;
  }

  ClearParty(destination: string) {
    this.partyWrite('ClearParty', (b) => b.ClearParty(destination));
  }

  DisablePartyEffects() {
    this.partyWrite('DisablePartyEffects', (b) => b.DisablePartyEffects());
  }

  RebuildParty() {
    this.partyWrite('RebuildParty', (b) => b.RebuildParty());
  }

  /* GemSelectControl.lua:620-627 */
  async GemSupportPairs(socketGroup: number): Promise<[number, number][]> {
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) return [];
    this.miscCalls++;
    try {
      return (exposition.MiscGemSupportPairs(m.calc, socketGroup) ?? []).map((p) => [p.Support, p.Active] as [number, number]);
    } finally {
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  /* ItemsTab.lua:1481-1484 */
  StatDifferencesShown(): boolean {
    return this.currentBuild?.StatDifferencesShown() ?? true;
  }

  ToggleStatDifferences(): boolean {
    const b = this.currentBuild;
    if (!b) return true;
    const on = !b.StatDifferencesShown();
    b.SetShowStatDifferences(on);
    this.updateStore();
    void this.Tick('ToggleStatDifferences');
    return on;
  }

  /* CalcsTab.lua:174-189 */
  GetCalcsSectionsCollapsed(keys: [string, string][]): (boolean | null)[] {
    const b = this.currentBuild;
    return keys.map(([id, sub]) => {
      const [collapsed, saved] = b?.CalcsSectionCollapsed(id, sub) ?? [false, false];
      return saved ? collapsed : null;
    });
  }

  /* CalcSectionControl.lua:41-44 */
  SetCalcsSectionCollapsed(id: string, subsection: string, collapsed: boolean) {
    this.currentBuild?.SetCalcsSectionCollapsed(id, subsection, collapsed);
    this.updateStore();
  }

  async GetSharedItems(): Promise<string[]> {
    return [...((await exposition.SharedItems()) ?? [])];
  }

  /* SharedItemListControl.lua:44-53 */
  async ShareItem(id: number, kind: string, name: string, index: number): Promise<boolean> {
    return this.currentBuild ? exposition.ShareItem(this.currentBuild, id, kind, name, index) : false;
  }

  /* SharedItemListControl.lua:66-72 */
  async DeleteSharedItem(index: number): Promise<boolean> {
    return exposition.DeleteSharedItem(index);
  }

  /* ItemListControl.lua:275-284 */
  async AddSharedItem(index: number, slot: string, before = 0): Promise<boolean> {
    const id = this.currentBuild ? await exposition.AddSharedItem(this.currentBuild, index, slot) : 0;
    if (!id) return false;
    if (before) this.currentBuild?.MoveItem(id, before);
    this.updateStore();
    void this.Tick('AddSharedItem');
    return true;
  }

  /* ItemListControl.lua:275-284 */
  async AddDBItemAt(kind: string, name: string, before: number): Promise<boolean> {
    if (!this.currentBuild) return false;
    const [id, ok] = await exposition.AddDBItem(this.currentBuild, kind, name, false, false);
    if (!ok) return false;
    if (before) this.currentBuild.MoveItem(id, before);
    this.updateStore();
    void this.Tick('AddDBItem');
    return true;
  }

  /* ListControl.lua:428-436, ItemListControl.lua:286-288 */
  MoveItem(id: number, before: number) {
    this.currentBuild?.MoveItem(id, before);
    this.updateStore();
    void this.Tick('MoveItem');
  }

  /* ItemsTab.lua:1817-1869 */
  async NormaliseDisplayItem(text: string, copyEldritchImplicits: boolean): Promise<string> {
    return this.currentBuild ? exposition.NormaliseDisplayItem(this.currentBuild, text, copyEldritchImplicits) : text;
  }

  /* CompareTab.lua:290-389, 2199-2230 */
  GetCompareSets(): CompareSets | undefined {
    const e = this.compareB;
    if (!e) return undefined;
    const b = e.build;
    const title = (t: string | undefined | null) => t || 'Default';
    const itemSets = b.Items.ItemSets ?? [];
    const groups = (b.Skills.SkillSets ?? [])[b.ActiveSkillSetIndex()]?.Skills ?? [];
    const c = e.side.outputs?.MainSkill;
    return {
      tree: { labels: (b.Tree.Specs ?? []).map((s) => title(s.Title)), active: b.ActiveSpecIndex() },
      skill: { labels: (b.Skills.SkillSets ?? []).map((s) => title(s.Title)), active: b.ActiveSkillSetIndex() },
      item: {
        labels: itemSets.map((s) => title(s.Title)),
        active: Math.max(
          0,
          itemSets.findIndex((s) => Number(s.ID) === b.Items.ActiveItemSet)
        )
      },
      config: { labels: (b.Config.ConfigSets ?? []).map((s) => title(s.Title)), active: b.ActiveConfigSetIndex() },
      group: { labels: groups.map((g) => g.DisplayLabel || g.Label || '<No active skills>'), active: b.Build.MainSocketGroup - 1 },
      skills: { labels: [...(c?.Skills ?? [])], active: (c?.MainActiveSkill ?? 1) - 1, enabled: !!c?.SkillEnabled },
      parts: c?.PartShown ? { labels: [...(c.Parts ?? [])], active: c.SkillPart - 1 } : undefined,
      stage: c?.StageShown ? c.StageCount : undefined,
      mine: c?.MineShown ? c.MineCount : undefined,
      minion: c?.MinionShown ? { labels: (c.Minions ?? []).map((o) => o.Label), active: c.Minion - 1, enabled: c.MinionEnabled } : undefined,
      minionSkill: c?.MinionSkillShown ? { labels: [...(c.MinionSkills ?? [])], active: c.MinionSkill - 1, enabled: c.MinionSkillEnabled } : undefined
    };
  }

  /* CompareTab.lua:293-389 */
  private async recalcCompare(e: NonNullable<typeof this.compareB>): Promise<CompareInfo> {
    const side = await this.compareSide(e.build, true);
    if (side?.outputs) {
      this.dropCompareState();
      e.side = side;
      e.seq = ++this.compareSeq;
    }
    return this.GetCompareInfo();
  }

  async SetCompareSet(kind: CompareSetKind, index: number): Promise<CompareInfo> {
    const e = this.compareB;
    const b = e?.build;
    const c = e?.side.outputs?.MainSkill;
    if (!e || !b) return this.GetCompareInfo();
    if (kind === 'tree') b.SetActiveSpec(index + 1);
    else if (kind === 'skill') b.SetActiveSkillSet(b.Skills.SkillSets?.[index]?.ID ?? 1);
    else if (kind === 'item') await exposition.SetActiveItemSet(b, Number(b.Items.ItemSets?.[index]?.ID ?? 1));
    else if (kind === 'config') b.SetActiveConfigSet(b.Config.ConfigSets?.[index]?.ID ?? 1);
    else if (kind === 'group') b.SetMainSocketGroup(index + 1);
    else if (kind === 'skills' && c) b.SetMainActiveSkill(c.SkillSet, c.SocketGroup, index + 1);
    else if (kind === 'parts' && c && c.GemIndex >= 0) b.SetGemSkillPart(c.SkillSet, c.SocketGroup, c.GemIndex, index + 1);
    else if (kind === 'minion' && c && c.GemIndex >= 0) {
      // CompareTab.lua:438-458
      const o = c.Minions?.[index];
      if (o?.ItemSetID) b.SetGemSkillMinionItemSet(c.SkillSet, c.SocketGroup, c.GemIndex, o.ItemSetID);
      else if (o?.MinionID) b.SetGemSkillMinion(c.SkillSet, c.SocketGroup, c.GemIndex, o.MinionID);
    } else if (kind === 'minionSkill' && c && c.GemIndex >= 0) b.SetGemSkillMinionSkill(c.SkillSet, c.SocketGroup, c.GemIndex, index + 1);
    return this.recalcCompare(e);
  }

  /* CompareTab.lua:395-435 */
  async SetCompareCount(kind: 'stage' | 'mine', value: number | undefined): Promise<CompareInfo> {
    const e = this.compareB;
    const c = e?.side.outputs?.MainSkill;
    if (!e || !c || c.GemIndex < 0) return this.GetCompareInfo();
    if (kind === 'stage') e.build.SetGemSkillStageCount(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? 0);
    else e.build.SetGemSkillMineCount(c.SkillSet, c.SocketGroup, c.GemIndex, value ?? -1);
    return this.recalcCompare(e);
  }

  /* Main.lua:1020-1024 */
  async SetShowFlavourText(on: boolean) {
    await exposition.SetShowFlavourText(on);
  }

  private timeless?: exposition.TimelessSearch;
  private timelessSeq = 0;
  private timelessDone = 0;

  /* TreeTab.lua:1187-1282, 2328-2395 */
  GetTimelessOptions() {
    return {
      types: [...(exposition.TimelessJewelTypes() ?? [])],
      conquerors: Object.fromEntries([1, 2, 3, 4, 5, 6].map((t) => [t, [...(exposition.TimelessConquerors(t) ?? [])]])) as Record<number, string[]>,
      devotion: [...(exposition.TimelessDevotionVariants() ?? [])],
      tradeTypes: [...(exposition.TimelessTradeTypes() ?? [])],
      realms: [...(exposition.TimelessRealms() ?? [])]
    };
  }

  /* TreeTab.lua:1295-1336 */
  GetTimelessLists(jewelType: number): { sockets: calculator.TimelessSocket[]; mods: calculator.TimelessMod[] } {
    if (!this.currentBuild) return { sockets: [], mods: [] };
    return { sockets: [...(exposition.TimelessSockets(this.currentBuild) ?? [])], mods: [...(exposition.TimelessMods(jewelType) ?? [])] };
  }

  /* TreeTab.lua:1558-1588 */
  async GetTimelessProtectable(socketId: number): Promise<calculator.TimelessMod[]> {
    const m = await this.miscCalculator();
    return m ? [...(exposition.TimelessProtectableNodes(m.calc, socketId) ?? [])] : [];
  }

  /* Build.lua:975-993 */
  GetTimelessData(): pob.TimelessData | undefined {
    return this.currentBuild ? exposition.GetTimelessData(this.currentBuild) : undefined;
  }

  /* Build.lua:1062-1077 */
  SetTimelessData(d: pob.TimelessData) {
    if (!this.currentBuild) return;
    exposition.SetTimelessData(this.currentBuild, d);
  }

  /* TreeTab.lua:2794-2859 */
  async TimelessSearch(q: calculator.TimelessSearchQuery): Promise<{ count: number; desired: calculator.TimelessDesiredNode[] } | undefined> {
    const seq = ++this.timelessSeq;
    this.timelessDone = 0;
    this.timeless?.release();
    this.timeless = undefined;
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc || seq !== this.timelessSeq) return undefined;
    const search = exposition.NewTimelessSearch(m.calc, q);
    if (!search) return undefined;
    for (let done = 0; done < 1; ) {
      if (token !== this.misc || seq !== this.timelessSeq) {
        search.release();
        return undefined;
      }
      done = search.Step();
      this.timelessDone = done;
      await yieldToQueue();
    }
    this.timeless = search;
    return { count: search.Count(), desired: [...(search.Desired() ?? [])] };
  }

  TimelessSearchProgress(): number {
    return this.timelessDone;
  }

  CancelTimelessSearch() {
    this.timelessSeq++;
  }

  TimelessRows(start: number, count: number): calculator.TimelessSearchResult[] {
    return [...(this.timeless?.Rows(start, count) ?? [])];
  }

  /* TreeTab.lua:2198-2310 */
  TimelessTradeURL(q: calculator.TimelessTradeQuery): calculator.TimelessTrade | undefined {
    return this.timeless?.TradeURL(q);
  }

  /* PassiveTreeView.lua:83-111 */
  GetTreeViewState(): pob.TreeView | undefined {
    return this.currentBuild?.GetTreeView();
  }

  /* PassiveTreeView.lua:101-111 */
  SetTreeViewState(v: pob.TreeView) {
    this.currentBuild?.SetTreeView(v);
  }

  /* PassiveTreeView.lua:1607-1627 */
  GetNodeSkillTooltips(nodeId: number): calculator.TooltipLine[][] {
    if (!this.currentBuild) return [];
    return (exposition.NodeSkillTooltips(this.currentBuild, nodeId, this.itemTipStats) ?? []).map((l) => [...(l ?? [])]);
  }

  /* PassiveTreeView.lua:1727 */
  GetGoldRespecPrice(): number {
    return this.currentBuild ? exposition.GoldRespecPrice(this.currentBuild) : 0;
  }

  /* TreeTab.lua:106-128 */
  GetTreeSpecCompare(specId: number): { nodes: number[]; masteries: { node: number; effect: number; stats: string[] }[] } {
    if (!this.currentBuild) return { nodes: [], masteries: [] };
    const c = exposition.TreeSpecCompare(this.currentBuild, specId);
    return {
      nodes: [...(c.Nodes ?? [])],
      masteries: (c.Masteries ?? []).map((m) => ({ node: m.Node, effect: m.Effect, stats: [...(m.Stats ?? [])] }))
    };
  }

  /* PassiveSpec.lua:1034-1048 */
  async GetSplitPersonalityPath(): Promise<number[]> {
    if (!this.misc && !this.tickMisc) return [];
    const m = await this.miscCalculator();
    return m ? [...(m.calc.SplitPersonalityPath() ?? [])] : [];
  }

  /* PassiveSpec.lua:776-784, 1136-1172 */
  async GetLeapHeldNodes(): Promise<number[]> {
    return this.currentBuild ? [...((await exposition.LeapHeldNodes(this.currentBuild)) ?? [])] : [];
  }

  /* TreeTab.lua:2097-2106 */
  GetTimelessFallbackModes(): { label: string; stat: string }[] {
    return (exposition.TimelessFallbackModes() ?? []).map((s) => ({ label: `Sort by ${s.Label}`, stat: s.Stat }));
  }

  /* TreeTab.lua:1929-2094 */
  async GenerateTimelessFallback(jewelType: number, stat: string): Promise<string | undefined> {
    const m = await this.miscCalculator();
    if (!m) return undefined;
    const [list, ok] = m.calc.TimelessFallbackWeights(jewelType, stat);
    return ok ? list : undefined;
  }

  /* TimelessJewelListControl.lua:330-331 */
  async TimelessRowCompare(row: number, devotion1: number, devotion2: number, socketLabel: string) {
    const text = this.timeless?.Item(row, devotion1, devotion2, socketLabel);
    return text ? this.ItemCompare(text) : undefined;
  }

  /* TimelessJewelListControl.lua:335-352 */
  AddTimelessResult(row: number, devotion1: number, devotion2: number, socketLabel: string, socketID?: number): number {
    if (!this.timeless || !this.currentBuild) return 0;
    const id = this.timeless.AddItem(this.currentBuild, row, devotion1, devotion2, socketLabel);
    // TimelessJewelListControl.lua:340-348
    if (id && socketID !== undefined) this.timeless.Socket(this.currentBuild, socketID, id);
    if (id) {
      this.updateStore();
      void this.Tick('AddTimelessResult');
    }
    return id;
  }

  private static readonly WARM_IDLE_MS = 10_000;

  private warmItemDB() {
    whenIdle(PoBWorker.WARM_IDLE_MS, () =>
      Promise.all(['UNIQUE', 'RARE'].map((kind) => exposition.ItemDBLists(kind)))
        .then(() => performance.mark('pob:item-db-warmed'))
        .catch((e: unknown) => console.error(e))
    );
  }

  GetPartyMembers(): PartyMemberView[] {
    const members = this.currentBuild?.PartyMembers() ?? [];
    const out = members.map((m) => {
      const buffs = m.Buffs ?? [];
      const view = {
        name: m.Name,
        className: m.ClassName,
        ascendancy: m.Ascendancy,
        level: m.Level,
        active: m.Active,
        gives: buffs.filter((x) => x.Text.trim()).map((x) => x.Name)
      };
      buffs.forEach(releaseWrapper);
      return view;
    });
    members.forEach(releaseWrapper);
    return out;
  }

  /* PartyTab.lua:298-325 */
  ImportPartyMember(code: string, destination: string, append: boolean) {
    const xml = pob.DecodeDecompress(code).unwrap();
    let err: Error | undefined;
    this.partyWrite('ImportParty', (b) => {
      try {
        b.AddPartyMember(xml, destination, append).unwrap();
      } catch (e) {
        err = e as Error;
      }
    });
    if (err) throw err;
  }

  SetPartyMemberActive(i: number, active: boolean) {
    this.partyWrite('SetPartyMemberActive', (b) => b.SetPartyMemberActive(i, active));
  }

  RemovePartyMember(i: number) {
    this.partyWrite('RemovePartyMember', (b) => b.RemovePartyMember(i));
  }

  private livePower?: exposition.PowerBuilder;

  private powerPool?: PowerPool;
  private livePool?: PowerPoolRun;
  private liveTasks = 0;
  private powerMainDone = 0;
  private localShards = 0;

  async BuildPowerReportLive(
    stat: string,
    maxDepth: number,
    workers = 1
  ): Promise<{ rows: calculator.PowerReportEntry[]; ascendancy?: calculator.PowerReportEntry[]; ms: number } | undefined> {
    const seq = ++this.powerSeq;
    this.powerDone = 0;
    this.powerMainDone = 0;
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc || seq !== this.powerSeq) {
      return undefined;
    }
    const pooled = workers > 1 && this.wasmModule !== undefined && this.currentBuild !== undefined;
    if (pooled) this.powerPool ??= new PowerPool(() => this.wasmModule);
    const xml = pooled ? new TextDecoder().decode(builds.SerializeBuild(this.currentBuild).unwrap()) : '';
    const warm = pooled && this.powerPool!.warm(workers, xml);
    const count = pooled && (warm || maxDepth === 0) ? workers : 1;
    if (!pooled) this.powerPool?.terminate();
    else if (count === 1) this.powerPool?.cancel();
    const builder = count > 1 ? exposition.NewPowerBuilderShard(m.calc, stat, maxDepth, 0, count) : exposition.NewPowerBuilder(m.calc, stat, maxDepth);
    if (!builder) {
      return undefined;
    }
    const stale = () => token !== this.misc || seq !== this.powerSeq;
    this.miscCalls++;
    this.livePower = builder;
    this.liveTasks = count > 1 ? builder.Tasks() : 0;
    const start = performance.now();
    let pool: PowerPoolRun | undefined;
    if (count > 1 && this.powerPool) {
      pool = this.powerPool.run(seq, xml, stat, maxDepth, count);
      this.livePool = pool;
    }
    try {
      for (let done = 0; done < 1; ) {
        if (stale()) {
          return undefined;
        }
        done = builder.Step(40);
        this.powerMainDone = done;
        this.powerDone = this.poolDone(done);
        await yieldToQueue();
      }
      const shards = [builder.Shard()];
      for (const [i, p] of (pool?.shards ?? []).entries()) {
        let rows = await p.catch((err: unknown) => {
          console.warn(`power shard ${i + 1} runs here: ${String(err)}`);
          return this.localPowerShard(m.calc, stat, maxDepth, i + 1, count, stale);
        });
        if (rows === undefined && !stale()) rows = await this.localPowerShard(m.calc, stat, maxDepth, i + 1, count, stale);
        if (!rows || stale()) {
          return undefined;
        }
        shards.push(rows);
      }
      if (stale()) {
        return undefined;
      }
      if (!pool) return { rows: [...(builder.Report() ?? [])], ascendancy: [...(exposition.PowerAscendancy(builder) ?? [])], ms: performance.now() - start };
      const ascendancy = exposition.MergePowerAscendancy(shards);
      return { rows: [...(exposition.MergePowerReports(shards) ?? [])], ascendancy: [...(ascendancy ?? [])], ms: performance.now() - start };
    } finally {
      if (this.livePower === builder) {
        this.livePower = undefined;
        this.livePool = undefined;
      }
      builder.release();
      this.miscCalls--;
      this.releaseIdleMisc();
    }
  }

  private async localPowerShard(
    calc: exposition.MiscCalculator,
    stat: string,
    maxDepth: number,
    index: number,
    count: number,
    stale: () => boolean
  ): Promise<calculator.PowerShardRows | undefined> {
    const b = exposition.NewPowerBuilderShard(calc, stat, maxDepth, index, count);
    if (!b) return undefined;
    this.localShards++;
    try {
      for (let done = 0; done < 1; ) {
        if (stale()) return undefined;
        done = b.Step(40);
        await yieldToQueue();
      }
      return b.Shard();
    } finally {
      b.release();
    }
  }

  private poolDone(done: number): number {
    const pool = this.livePool;
    if (!pool) return done;
    let sum = done * this.liveTasks;
    let total = this.liveTasks;
    for (const p of pool.progress()) {
      const tasks = p.tasks || this.liveTasks;
      sum += p.done * tasks;
      total += tasks;
    }
    return total ? sum / total : 1;
  }

  PowerReportLive(withRows: boolean): { done: number; current: number[]; rows?: calculator.PowerReportEntry[] } {
    const b = this.livePower;
    if (!b) return { done: this.powerDone, current: [] };
    const pool = this.livePool;
    if (pool) this.powerDone = this.poolDone(this.powerMainDone);
    const rows = !withRows ? undefined : pool ? [b.Shard(), ...pool.partial()].flatMap((s) => s.Rows ?? []) : [...(b.Report() ?? [])];
    const current = [...new Set([b.Current(), ...(pool?.current() ?? [])])].filter((id) => id !== 0);
    return { done: this.powerDone, current, rows: rows?.filter((r) => r.Power !== 0) };
  }

  PowerPoolStats(): { workers: number; running: number; bootMs: (number | undefined)[]; memoryMB: (number | undefined)[]; localShards: number } {
    return { ...(this.powerPool?.stats() ?? { workers: 0, running: 0, bootMs: [], memoryMB: [] }), localShards: this.localShards };
  }

  // Build.lua:534-540
  async ItemPrimarySlot(id: number, kind: string, name: string): Promise<string> {
    return this.currentBuild ? await exposition.ItemPrimarySlot(this.currentBuild, id, kind, name) : '';
  }

  /* SkillsTab.lua:363-379 */
  async GetSocketGroupSourceNames(setIndex: number, groupIndex: number): Promise<calculator.GroupSourceName[]> {
    return this.currentBuild ? ((await exposition.SocketGroupSourceNames(this.currentBuild, setIndex, groupIndex)) ?? []) : [];
  }

  /* SkillsTab.lua:294-313 */
  async OptimiseSocketsUseful(slotName: string): Promise<boolean> {
    return !!this.currentBuild && (await exposition.OptimiseSocketsUseful(this.currentBuild, slotName));
  }

  /* SkillsTab.lua:242-288 */
  async OptimiseSockets(slotName: string): Promise<boolean> {
    if (!this.currentBuild || !(await exposition.OptimiseSockets(this.currentBuild, slotName))) return false;
    this.updateStore();
    void this.Tick('OptimiseSockets');
    return true;
  }

  /* Build.lua:1093-1138 */
  async GetSecondaryAscendancies(): Promise<calculator.SecondaryAscendancyDropdown | undefined> {
    return this.currentBuild ? await exposition.SecondaryAscendancies(this.currentBuild) : undefined;
  }

  /* Build.lua:285-295 */
  async ChangeSecondaryAscendancy(ascendClassId: number) {
    if (!this.currentBuild) return;
    await exposition.ChangeSecondaryAscendancy(this.currentBuild, ascendClassId);
    this.updateStore();
    void this.Tick('ChangeSecondaryAscendancy');
  }

  private compareCalcsEnvs: calculator.Environment[] = [];

  CompareModRows(side: 0 | 1, sd: calculator.ModSectionData): calculator.CompareModRow[] {
    return [...(this.compareCalcsEnvs[side]?.CompareModRows(sd, false) ?? [])];
  }

  async GetTreeSpecClusterNodes(specId: number): Promise<TreeClusterNode[]> {
    if (!this.currentBuild) return [];
    return ((await exposition.GetTreeSpecClusterNodes(this.currentBuild, specId)) ?? []).map((n) => ({
      id: n.ID,
      type: n.Type as TreeClusterNode['type'],
      name: n.Name,
      icon: n.Icon,
      stats: [...(n.Stats ?? [])],
      group: n.Group,
      orbit: n.Orbit,
      orbitIndex: n.OrbitIndex,
      x: n.X,
      y: n.Y,
      linked: [...(n.Linked ?? [])]
    }));
  }

  /* CompareTab.lua:775-831, 839-855 */
  GetComparePrimarySets():
    | { tree: { labels: string[]; active: number }; item: { labels: string[]; ids: number[]; active: number }; versions: { a: string; b: string } }
    | undefined {
    const a = this.currentBuild;
    if (!a) return undefined;
    const title = (t: string | undefined | null) => t || 'Default';
    const itemSets = a.Items.ItemSets ?? [];
    const version = (b: pob.PathOfBuilding | undefined) => (b ? ((b.Tree.Specs ?? [])[b.ActiveSpecIndex()]?.TreeVersion ?? '') : '');
    return {
      tree: { labels: (a.Tree.Specs ?? []).map((s) => title(s.Title)), active: a.ActiveSpecIndex() },
      item: {
        labels: itemSets.map((s) => title(s.Title)),
        ids: itemSets.map((s) => Number(s.ID)),
        active: Math.max(
          0,
          itemSets.findIndex((s) => Number(s.ID) === a.Items.ActiveItemSet)
        )
      },
      versions: { a: version(a), b: version(this.compareB?.build) }
    };
  }

  /* CompareTab.lua:880-887, TreeTab.lua:588-612 */
  async ConvertCompareTreeSpec(version: string, replace: boolean): Promise<{ info: CompareInfo; msg: string }> {
    const e = this.compareB;
    if (!e) return { info: this.GetCompareInfo(), msg: '' };
    const msg = (await exposition.ConvertTreeSpec(e.build, version, replace)) ?? '';
    return { info: await this.recalcCompare(e), msg };
  }

  /* CompareTab.lua:3954-3963 */
  async CompareItemTooltip(side: 'a' | 'b', name: string): Promise<calculator.ItemTooltip | undefined> {
    const build = side === 'a' ? this.currentBuild : this.compareB?.build;
    if (!build) return undefined;
    const [tip, ok] = await exposition.ItemSourceTooltip(build, name, this.itemTipStats);
    return ok ? tip : undefined;
  }

  /* CompareTab.lua:4139-4157, 4411-4427 */
  async CompareGemSupportPairs(side: 'a' | 'b', group: number): Promise<[number, number][]> {
    if (side === 'a') return this.GemSupportPairs(group);
    const m = await this.compareMiscCalculator();
    return m ? (exposition.MiscGemSupportPairs(m.calc, group) ?? []).map((p) => [p.Support, p.Active] as [number, number]) : [];
  }

  private compareCalcsControls?: calculator.MainSkillControls;

  /* CompareTab.lua:4617-4633 */
  CompareModsEqual(sds: calculator.ModSectionData[]): boolean[] {
    const [a, b] = this.compareCalcsEnvs;
    const e = this.compareB?.build;
    if (!a || !b || !this.currentBuild || !e) return sds.map(() => false);
    const [ma, mb] = [this.currentBuild.CalcsInputBool('showMinion'), e.CalcsInputBool('showMinion')];
    return sds.map((sd) => modRowsEqual([...(a.CompareModRows(sd, ma) ?? [])], [...(b.CompareModRows(sd, mb) ?? [])]));
  }

  /* CompareTab.lua:490-718 */
  private compareCalcsInputs(b: pob.PathOfBuilding): CompareCalcsInputs {
    const groups = (b.Skills.SkillSets ?? [])[b.ActiveSkillSetIndex()]?.Skills ?? [];
    return {
      groups: groups.map((g) => g.DisplayLabel || g.Label || '<No active skills>'),
      skillNumber: b.CalcsInputNumber('skill_number', 1),
      buffMode: b.CalcsInputString('misc_buffMode', 'EFFECTIVE'),
      showMinion: b.CalcsInputBool('showMinion')
    };
  }

  /* CompareTab.lua:592-718 */
  SetCompareCalcsControl(field: 'group' | 'mode' | 'showMinion' | MainSkillField, value: number | string | boolean | undefined) {
    const b = this.compareB?.build;
    const c = this.compareCalcsControls;
    if (!b) return;
    const n = typeof value === 'number' ? value : undefined;
    if (field === 'group' && n !== undefined) b.SetCalcsInputNumber('skill_number', n);
    else if (field === 'mode' && typeof value === 'string') b.SetCalcsInputString('misc_buffMode', value);
    else if (field === 'showMinion') b.SetCalcsInputBool('showMinion', !!value);
    else if (!c) return;
    else if (field === 'MainActiveSkill' && n !== undefined) b.SetMainActiveSkillCalcs(c.SkillSet, c.SocketGroup, n);
    else if (c.GemIndex < 0) return;
    else if (field === 'SkillPart' && n !== undefined) b.SetGemSkillPartCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, n);
    else if (field === 'StageCount') b.SetGemSkillStageCountCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, n ?? 0);
    else if (field === 'MineCount') b.SetGemSkillMineCountCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, n ?? -1);
    else if (field === 'Minion') {
      const o = n === undefined ? undefined : c.Minions?.[n - 1];
      if (o?.ItemSetID) b.SetGemSkillMinionItemSetCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, o.ItemSetID);
      else if (o?.MinionID) b.SetGemSkillMinionCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, o.MinionID);
    } else if (field === 'MinionSkill' && n !== undefined) b.SetGemSkillMinionSkillCalcs(c.SkillSet, c.SocketGroup, c.GemIndex, n);
  }

  /* PassiveTreeView.lua:1494-1497 */
  async GetSocketDistance(nodeId: number): Promise<number> {
    if (!this.misc && !this.tickMisc) return 0;
    const m = await this.miscCalculator();
    return m ? m.calc.SocketDistanceToClassStart(nodeId) : 0;
  }

  /* CompareTab.lua:1372-1436 */
  async GetCompareJewels(): Promise<CompareJewel[] | undefined> {
    const b = this.compareB;
    if (!b || !this.currentBuild) return undefined;
    const item = async (build: pob.PathOfBuilding, slot: string, text: string): Promise<CompareItem | undefined> => {
      if (!text) return undefined;
      const parsed = await exposition.ParseItem(build, text, NO_TIP_STATS);
      const it = parsed.ok ? parsed.value : undefined;
      return { slot, name: it?.Name ?? '', rarity: it?.Rarity ?? 'NORMAL', text, line: it?.Explicits?.[0]?.Line ?? it?.BaseName ?? '' };
    };
    const a = this.currentBuild;
    return Promise.all(
      ((await calculator.CompareJewelSlots(a, b.build)) ?? []).map(async (j) => {
        const slot = `Jewel ${j.NodeID}`;
        return {
          slot,
          label: j.Label,
          node: j.NodeID,
          a: await item(a, slot, j.PText),
          b: await item(b.build, slot, j.CText),
          aAllocated: j.PAllocated,
          bAllocated: j.CAllocated
        };
      })
    );
  }

  /* CompareTab.lua:3466-3637 */
  async CompareItemCards(a: string, b: string): Promise<[string[], string[]]> {
    return [[...((await calculator.CompareItemExpanded(a, b)) ?? [])], [...((await calculator.CompareItemExpanded(b, a)) ?? [])]];
  }

  /* CompareCalcsHelpers.lua:355-394 */
  CompareSkillBreakdown(side: 0 | 1, name: string): calculator.CompareSkillBreakdownRow[] {
    return [...(this.compareCalcsEnvs[side]?.CompareSkillBreakdown(name) ?? [])];
  }

  /* CompareTab.lua:2785-2868 */
  async BuildCompareJewelPower(receiver: 'a' | 'b', stats: string[]): Promise<calculator.ComparePowerEntry[] | undefined> {
    const b = this.compareB;
    const a = this.currentBuild;
    if (!b || !a) return undefined;
    const seq = ++this.comparePowerSeq;
    const tick = this.tickSeq;
    const [primary, other] = receiver === 'a' ? [a, b.build] : [b.build, a];
    const builder = calculator.NewCompareJewelPowerBuilder(primary, other, stats);
    if (!builder) return undefined;
    const stale = () => seq !== this.comparePowerSeq || this.compareB !== b || this.currentBuild !== a || this.tickSeq !== tick || this.ticksInFlight > 0;
    try {
      for (let done = 0; done < 1; ) {
        if (stale()) return undefined;
        done = builder.StepMs(40);
        await yieldToQueue();
      }
      return stale() ? undefined : [...(builder.Report() ?? [])];
    } finally {
      builder.release();
    }
  }

  /* CompareTab.lua:1128-1160 */
  async SetCompareConfigOption(key: string, value: boolean | number | string | null): Promise<CompareInfo> {
    const e = this.compareB;
    if (!e) return this.GetCompareInfo();
    const v = reverseConfigOptions[key];
    const remove =
      value === null ||
      value === '' ||
      (v?.type === 'list' && value === v.list[(v.defaultIndex ?? 1) - 1].value) ||
      (v?.type === 'check' && value === (v.defaultState ?? false));
    if (remove) e.build.RemoveConfigOption(key);
    else
      e.build.SetConfigOption(
        typeof value === 'boolean' ? { Name: key, Boolean: value } : typeof value === 'string' ? { Name: key, String: value } : { Name: key, Number: value }
      );
    const info = await this.recalcCompare(e);
    const calc = calculator.NewCalculator(e.build);
    const fresh = calc?.PoB;
    try {
      const inputs = fresh ? (await this.compareSide(fresh, false))?.inputs : undefined;
      if (inputs) e.side = { ...e.side, inputs };
    } finally {
      releaseWrapper(fresh);
      calc?.release();
    }
    return info;
  }

  /* CompareTab.lua:1439-1459 */
  async EquipCompareJewel(text: string, nodeId: number) {
    if (!this.currentBuild) return;
    const id = this.currentBuild.AddItem(text);
    await exposition.SocketTreeJewel(this.currentBuild, nodeId, id);
    this.updateStore();
    void this.Tick('EquipCompareJewel');
  }

  /* Calcs.lua:37-70 */
  async GetModDebugDump(): Promise<string> {
    if (!this.currentBuild) return '';
    const calc = calculator.NewCalculator(this.currentBuild);
    const text = await exposition.ModDebugDump(calc);
    releaseWrapper(calc);
    return text;
  }

  /* ItemsTab.lua:5160-5166 */
  async SetItemModDebug(on: boolean) {
    await exposition.SetItemModDebug(on);
  }

  async ClearDataCache(): Promise<number> {
    const names = await storageFs.readdir('/cache');
    for (const n of names) await storageFs.rm(`/cache/${n}`, { recursive: true, force: true });
    return names.length;
  }

  async ExportBuilds(dir = ''): Promise<{ path: string; data: Uint8Array }[]> {
    const out: { path: string; data: Uint8Array }[] = [];
    for (const n of await storageFs.readdir(`/builds${dir}`)) {
      const p = `${dir}/${n}`;
      if ((await storageFs.stat(`/builds${p}`)).isDirectory()) out.push(...(await this.ExportBuilds(p)));
      else out.push({ path: p.slice(1), data: new Uint8Array(await storageFs.readFile(`/builds${p}`)) });
    }
    return out;
  }

  /* Main.lua:1099-1102 */
  async SetDefaultItemAffixQuality(q: number) {
    await calculator.SetDefaultItemAffixQuality(q);
  }

  /* TreeTab.lua:846-878 */
  GetTattooChoices(nodeId: number, showLegacy: boolean): { dn: string; label: string; descriptions: string[]; legacy: boolean }[] {
    if (!this.currentBuild) return [];
    return (exposition.GetTattooChoices(this.currentBuild, nodeId, showLegacy) ?? []).map((c) => ({
      dn: c.DN,
      label: c.Label,
      descriptions: [...(c.Descriptions ?? [])],
      legacy: c.Legacy
    }));
  }

  /* PassiveTreeView.lua:847-877 */
  GetTreeTattooArt(): { nodeId: number; icon: string; effect: string }[] {
    if (!this.currentBuild) return [];
    return (exposition.TreeTattooArt(this.currentBuild) ?? []).map((a) => ({ nodeId: a.NodeID, icon: a.Icon, effect: a.ActiveEffectImage }));
  }

  /* TreeTab.lua:926-933 */
  AddTattoo(nodeId: number, dn: string): string | undefined {
    if (!this.currentBuild) return 'no build';
    const result = exposition.AddTattoo(this.currentBuild, nodeId, dn);
    if (!result.ok) return String(result.error);
    this.updateStore();
    void this.Tick('AddTattoo');
    return undefined;
  }

  /* ImportTab.lua:701-705 */
  async ImportItemsAndSkills(json: string): Promise<{ ok: boolean; text: string }> {
    if (this.dataReady) await this.dataReady;
    if (!this.currentBuild) return { ok: false, text: 'no build' };
    const result = await exposition.ImportItemsAndSkills(this.currentBuild, json, true, true, false);
    if (!result.ok) return { ok: false, text: result.error.message };
    this.updateStore();
    void this.Tick('ImportItemsAndSkills');
    return { ok: true, text: result.value };
  }

  /* ImportTab.lua:703 */
  async ImportPassiveTreeAndJewels(json: string): Promise<{ ok: boolean; text: string }> {
    if (this.dataReady) await this.dataReady;
    if (!this.currentBuild) return { ok: false, text: 'no build' };
    const result = await exposition.ImportPassiveTreeAndJewels(this.currentBuild, json, true);
    if (!result.ok) return { ok: false, text: result.error.message };
    this.updateStore();
    void this.Tick('ImportPassiveTreeAndJewels');
    return { ok: true, text: result.value };
  }

  /* ImportTab.lua:813-849 */
  async ImportAccountPassiveTree(character: string, passives: string, deleteJewels: boolean): Promise<{ ok: boolean; text: string }> {
    if (this.dataReady) await this.dataReady;
    if (!this.currentBuild) return { ok: false, text: 'no build' };
    const result = await exposition.ImportAccountPassiveTree(this.currentBuild, character, passives, deleteJewels);
    if (!result.ok) return { ok: false, text: result.error.message };
    this.updateStore();
    void this.Tick('ImportAccountPassiveTree');
    return { ok: true, text: result.value };
  }

  /* ImportTab.lua:851-885 */
  async ImportAccountItems(
    character: string,
    items: string,
    clearItems: boolean,
    clearSkills: boolean,
    ignoreWeaponSwap: boolean
  ): Promise<{ ok: boolean; text: string }> {
    if (this.dataReady) await this.dataReady;
    if (!this.currentBuild) return { ok: false, text: 'no build' };
    const result = await exposition.ImportAccountItems(this.currentBuild, character, items, clearItems, clearSkills, ignoreWeaponSwap);
    if (!result.ok) return { ok: false, text: result.error.message };
    this.updateStore();
    void this.Tick('ImportAccountItems');
    return { ok: true, text: result.value };
  }

  /* SkillsTab.lua:948-960 */
  async GemEnableCompare(socketGroup: number, gemIndex: number): Promise<StatCompareSection[] | undefined> {
    if (!this.currentBuild) return undefined;
    const gems = this.fresh((b) => b.Skills.SkillSets[b.ActiveSkillSetIndex()]?.Skills?.[socketGroup]?.Gems) ?? [];
    const gem = gems[gemIndex];
    if (!gem) return undefined;
    return this.gemWhatIf(socketGroup, gemIndex, { ...gem, Enabled: !gem.Enabled });
  }

  GetTreeTattooNodes(): { nodeId: number; name: string; stats: string[]; reminder: string[] }[] {
    if (!this.currentBuild) return [];
    return (exposition.TreeTattooNodes(this.currentBuild) ?? []).map((t) => ({
      nodeId: t.NodeID,
      name: t.Name,
      stats: t.Stats ?? [],
      reminder: t.Reminder ?? []
    }));
  }

  /* Build.lua:753-878 */
  async GetCompareLoadouts(): Promise<{ loadouts: string[]; selected: number }> {
    const b = this.compareB?.build;
    if (!b) return { loadouts: [], selected: -1 };
    const l = await exposition.GetLoadouts(b);
    return { loadouts: [...(l?.Loadouts ?? [])], selected: l?.Selected ?? -1 };
  }

  /* Build.lua:355-414 */
  async SelectCompareLoadout(name: string): Promise<CompareInfo> {
    const e = this.compareB;
    if (!e) return this.GetCompareInfo();
    const item = e.build.Items.ActiveItemSet;
    if (!(await exposition.SelectLoadout(e.build, name))) return this.GetCompareInfo();
    if (e.build.Items.ActiveItemSet !== item) await exposition.SetActiveItemSet(e.build, e.build.Items.ActiveItemSet);
    return this.recalcCompare(e);
  }

  /* PassiveSpec.lua:1614-1635, 2380-2388 */
  async GetTreeConqueredNodes(): Promise<{ nodeId: number; name: string; stats: string[]; replaced: boolean }[]> {
    if (!this.misc && !this.tickMisc) return [];
    const token = this.miscCalculator();
    const m = await token;
    if (!m || token !== this.misc) return [];
    return (exposition.TreeConqueredNodes(m.calc) ?? []).map((r) => ({ nodeId: r.NodeID, name: r.Name, stats: [...(r.Stats ?? [])], replaced: r.Replaced }));
  }

  private sidebarPass?: { tick: number; pass?: exposition.SidebarPass };

  private runSidebarPass(): exposition.SidebarPass | undefined {
    return this.currentBuild ? exposition.NewSidebarPass(this.currentBuild) : undefined;
  }

  private currentSidebarPass(): exposition.SidebarPass | undefined {
    if (!this.currentBuild) return undefined;
    const tick = this.ticksStarted;
    if (this.sidebarPass?.tick !== tick) {
      releaseWrapper(this.sidebarPass?.pass);
      this.sidebarPass = { tick, pass: this.runSidebarPass() };
    }
    return this.sidebarPass.pass;
  }

  /* Build.lua:1872-1908 */
  SidebarBreakdownKey(ref: SidebarStatRef): string | undefined {
    const pass = this.currentSidebarPass();
    if (!pass) return undefined;
    const [key, ok] = pass.StatBreakdownKey(!!ref.minion, ref.stat, ref.childStat ?? '', ref.breakdown ?? '', ref.breakdown !== undefined, !!ref.modNames);
    return ok ? key : undefined;
  }

  SidebarBreakdownSections(minion: boolean, cells: SidebarCell[]): SidebarSection[] {
    const pass = this.currentSidebarPass();
    if (!pass) return [];
    return cells.flatMap((cell): SidebarSection[] => {
      if (cell.breakdown !== undefined) {
        const entry = pass.Entry(minion, cell.breakdown);
        const bd = entry ? breakdownOf(dump(entry, true) as calculator.EntryPlain, Infinity) : undefined;
        return bd ? [{ label: cell.label, bd }] : [];
      }
      if (!cell.modName) return [];
      const rows = pass.ModSection(minion, {
        ModName: [cell.modName].flat(),
        Multi: typeof cell.modName !== 'string',
        ModType: cell.modType ?? '',
        Cfg: cell.cfg ?? '',
        ModSource: cell.modSource ?? '',
        Enemy: !!cell.enemy,
        Actor: cell.actor ?? ''
      } as calculator.ModSectionData);
      return rows?.length ? [{ label: cell.label, mods: [...rows] }] : [];
    });
  }

  /* PassiveTreeView.lua:1755-1762 */
  CompareSpecJewelTooltip(specId: number, nodeId: number): calculator.ItemTooltip | undefined {
    if (!this.currentBuild) return undefined;
    const [tip, ok] = exposition.CompareSpecJewelTooltip(this.currentBuild, specId, nodeId, this.itemTipStats);
    return ok ? tip : undefined;
  }

  PowerPoolWarm(workers: number): void {
    if (workers <= 1) {
      this.powerPool?.terminate();
      return;
    }
    if (!this.wasmModule || !this.currentBuild) return;
    this.powerPool ??= new PowerPool(() => this.wasmModule);
    this.powerPool.warm(workers, new TextDecoder().decode(builds.SerializeBuild(this.currentBuild).unwrap()));
  }

  async RebaseChoices(text: string): Promise<string[]> {
    return [...((await exposition.RebaseChoices(text)) ?? [])];
  }

  async RebaseItemText(text: string, base: string): Promise<{ text: string; dropped: string[] } | undefined> {
    const [r, ok] = await exposition.RebaseItemText(text, base);
    return ok ? { text: r.Text, dropped: [...(r.Dropped ?? [])] } : undefined;
  }
}

export const workerInstance = new PoBWorker();

addEventListener('message', noteCall);
expose(workerInstance);

export type WorkerType = PoBWorker;
