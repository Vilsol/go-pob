/* eslint-disable */

const wrap = (name, fn) => {
  return (...args) => {
    const result = fn.call(undefined, ...args);
    if (globalThis.goInternalError) {
      const error = new Error(globalThis.goInternalError);
      globalThis.goInternalError = undefined;
      throw error;
    }
    return result;
  }
};

let initialized = false;

const pending = (name) => new Proxy({}, {
  get(target, property) {
    if (typeof property === 'symbol' || property === 'then') {
      return undefined;
    }
    if (initialized) {
      throw new Error('crystalline: this ' + name + ' was captured before initializeCrystalline() ran, so it is a stale copy. Read it from the module instead of destructuring it earlier, or move the import after initialisation.');
    }
    throw new Error('crystalline: ' + name + '.' + String(property) + ' was read before initializeCrystalline() ran. Start the Go wasm module, then call initializeCrystalline().');
  }
});

export let builds = pending('builds');
export let calculator = pending('calculator');
export let config = pending('config');
export let data = pending('data');
export let exposition = pending('exposition');
export let mod = pending('mod');
export let msgp = pending('msgp');
export let pob = pending('pob');
export let poe = pending('poe');
export let raw = pending('raw');
export let storage = pending('storage');

export const initializeCrystalline = () => {
  if (globalThis['go']?.['go-pob'] === undefined) {
    throw new Error('crystalline: globalThis.go.go-pob is not set. Start the Go wasm module before calling initializeCrystalline().');
  }

  const failedImports = globalThis['go']['go-pob']['__crystalline']?.['importFailures'];
  if (failedImports?.length) {
    throw new Error('crystalline: Go could not reach what it imported: ' + failedImports.join('; '));
  }

  builds = {
    EmptyBuild: wrap('builds.EmptyBuild', globalThis['go']['go-pob']['builds']['EmptyBuild']),
    ParseBuild: wrap('builds.ParseBuild', globalThis['go']['go-pob']['builds']['ParseBuild']),
    ParseBuildOver: wrap('builds.ParseBuildOver', globalThis['go']['go-pob']['builds']['ParseBuildOver']),
    ParseBuildStr: wrap('builds.ParseBuildStr', globalThis['go']['go-pob']['builds']['ParseBuildStr']),
    SavedBuildSummary: wrap('builds.SavedBuildSummary', globalThis['go']['go-pob']['builds']['SavedBuildSummary']),
    SerializeBuild: wrap('builds.SerializeBuild', globalThis['go']['go-pob']['builds']['SerializeBuild']),
    SerializeBuildWithStats: wrap('builds.SerializeBuildWithStats', globalThis['go']['go-pob']['builds']['SerializeBuildWithStats'])
  };
  calculator = {
    CompareItemExpanded: wrap('calculator.CompareItemExpanded', globalThis['go']['go-pob']['calculator']['CompareItemExpanded']),
    CompareJewelSlots: wrap('calculator.CompareJewelSlots', globalThis['go']['go-pob']['calculator']['CompareJewelSlots']),
    ItemRarity: globalThis['go']['go-pob']['calculator']['ItemRarity'],
    NewCalculator: wrap('calculator.NewCalculator', globalThis['go']['go-pob']['calculator']['NewCalculator']),
    NewCompareJewelPowerBuilder: wrap('calculator.NewCompareJewelPowerBuilder', globalThis['go']['go-pob']['calculator']['NewCompareJewelPowerBuilder']),
    NewComparePowerBuilder: wrap('calculator.NewComparePowerBuilder', globalThis['go']['go-pob']['calculator']['NewComparePowerBuilder']),
    OutTable: globalThis['go']['go-pob']['calculator']['OutTable'],
    OutputMode: globalThis['go']['go-pob']['calculator']['OutputMode'],
    SetDefaultItemAffixQuality: wrap('calculator.SetDefaultItemAffixQuality', globalThis['go']['go-pob']['calculator']['SetDefaultItemAffixQuality']),
    SkillFlag: globalThis['go']['go-pob']['calculator']['SkillFlag']
  };
  config = {
    InitLogging: wrap('config.InitLogging', globalThis['go']['go-pob']['config']['InitLogging'])
  };
  data = {
    DamageType: globalThis['go']['go-pob']['data']['DamageType'],
    ItemClassName: globalThis['go']['go-pob']['data']['ItemClassName'],
    SkillType: globalThis['go']['go-pob']['data']['SkillType']
  };
  exposition = {
    AddDBItem: wrap('exposition.AddDBItem', globalThis['go']['go-pob']['exposition']['AddDBItem']),
    AddSharedItem: wrap('exposition.AddSharedItem', globalThis['go']['go-pob']['exposition']['AddSharedItem']),
    AddTattoo: wrap('exposition.AddTattoo', globalThis['go']['go-pob']['exposition']['AddTattoo']),
    AffixEditText: wrap('exposition.AffixEditText', globalThis['go']['go-pob']['exposition']['AffixEditText']),
    AffixRollTooltip: wrap('exposition.AffixRollTooltip', globalThis['go']['go-pob']['exposition']['AffixRollTooltip']),
    AnointNotables: wrap('exposition.AnointNotables', globalThis['go']['go-pob']['exposition']['AnointNotables']),
    CalcConfigStats: wrap('exposition.CalcConfigStats', globalThis['go']['go-pob']['exposition']['CalcConfigStats']),
    CalcCustomModGroupStats: wrap('exposition.CalcCustomModGroupStats', globalThis['go']['go-pob']['exposition']['CalcCustomModGroupStats']),
    CalcGemStats: wrap('exposition.CalcGemStats', globalThis['go']['go-pob']['exposition']['CalcGemStats']),
    CalcMasteryEffect: wrap('exposition.CalcMasteryEffect', globalThis['go']['go-pob']['exposition']['CalcMasteryEffect']),
    CalcTreeSpec: wrap('exposition.CalcTreeSpec', globalThis['go']['go-pob']['exposition']['CalcTreeSpec']),
    CalcTreeVersion: wrap('exposition.CalcTreeVersion', globalThis['go']['go-pob']['exposition']['CalcTreeVersion']),
    CalculateAllocationPaths: wrap('exposition.CalculateAllocationPaths', globalThis['go']['go-pob']['exposition']['CalculateAllocationPaths']),
    CalculateDependentNodes: wrap('exposition.CalculateDependentNodes', globalThis['go']['go-pob']['exposition']['CalculateDependentNodes']),
    CalculatePrunableNodes: wrap('exposition.CalculatePrunableNodes', globalThis['go']['go-pob']['exposition']['CalculatePrunableNodes']),
    ChangeSecondaryAscendancy: wrap('exposition.ChangeSecondaryAscendancy', globalThis['go']['go-pob']['exposition']['ChangeSecondaryAscendancy']),
    ChangeTreeAscendancy: wrap('exposition.ChangeTreeAscendancy', globalThis['go']['go-pob']['exposition']['ChangeTreeAscendancy']),
    ChangeTreeClass: wrap('exposition.ChangeTreeClass', globalThis['go']['go-pob']['exposition']['ChangeTreeClass']),
    ClickTreeAscendancyNode: wrap('exposition.ClickTreeAscendancyNode', globalThis['go']['go-pob']['exposition']['ClickTreeAscendancyNode']),
    ConfigPlaceholders: wrap('exposition.ConfigPlaceholders', globalThis['go']['go-pob']['exposition']['ConfigPlaceholders']),
    ConvertAllTreeSpecs: wrap('exposition.ConvertAllTreeSpecs', globalThis['go']['go-pob']['exposition']['ConvertAllTreeSpecs']),
    ConvertTreeSpec: wrap('exposition.ConvertTreeSpec', globalThis['go']['go-pob']['exposition']['ConvertTreeSpec']),
    CopyItemSet: wrap('exposition.CopyItemSet', globalThis['go']['go-pob']['exposition']['CopyItemSet']),
    CorruptItemText: wrap('exposition.CorruptItemText', globalThis['go']['go-pob']['exposition']['CorruptItemText']),
    CorruptedRangeLine: wrap('exposition.CorruptedRangeLine', globalThis['go']['go-pob']['exposition']['CorruptedRangeLine']),
    CraftBaseGroups: wrap('exposition.CraftBaseGroups', globalThis['go']['go-pob']['exposition']['CraftBaseGroups']),
    CraftItemText: wrap('exposition.CraftItemText', globalThis['go']['go-pob']['exposition']['CraftItemText']),
    CrucibleText: wrap('exposition.CrucibleText', globalThis['go']['go-pob']['exposition']['CrucibleText']),
    CurrentGameVersionLabel: wrap('exposition.CurrentGameVersionLabel', globalThis['go']['go-pob']['exposition']['CurrentGameVersionLabel']),
    CustomModText: wrap('exposition.CustomModText', globalThis['go']['go-pob']['exposition']['CustomModText']),
    DefaultGem: wrap('exposition.DefaultGem', globalThis['go']['go-pob']['exposition']['DefaultGem']),
    DeleteAllItems: wrap('exposition.DeleteAllItems', globalThis['go']['go-pob']['exposition']['DeleteAllItems']),
    DeleteItem: wrap('exposition.DeleteItem', globalThis['go']['go-pob']['exposition']['DeleteItem']),
    DeleteItemPrompt: wrap('exposition.DeleteItemPrompt', globalThis['go']['go-pob']['exposition']['DeleteItemPrompt']),
    DeleteItemSet: wrap('exposition.DeleteItemSet', globalThis['go']['go-pob']['exposition']['DeleteItemSet']),
    DeleteSharedItem: wrap('exposition.DeleteSharedItem', globalThis['go']['go-pob']['exposition']['DeleteSharedItem']),
    DeleteSharedItemSet: wrap('exposition.DeleteSharedItemSet', globalThis['go']['go-pob']['exposition']['DeleteSharedItemSet']),
    DeleteTreeSpec: wrap('exposition.DeleteTreeSpec', globalThis['go']['go-pob']['exposition']['DeleteTreeSpec']),
    DeleteUnusedItems: wrap('exposition.DeleteUnusedItems', globalThis['go']['go-pob']['exposition']['DeleteUnusedItems']),
    DisplayItem: wrap('exposition.DisplayItem', globalThis['go']['go-pob']['exposition']['DisplayItem']),
    DropItemOnSlot: wrap('exposition.DropItemOnSlot', globalThis['go']['go-pob']['exposition']['DropItemOnSlot']),
    EditItem: wrap('exposition.EditItem', globalThis['go']['go-pob']['exposition']['EditItem']),
    EquipItem: wrap('exposition.EquipItem', globalThis['go']['go-pob']['exposition']['EquipItem']),
    EquipItemID: wrap('exposition.EquipItemID', globalThis['go']['go-pob']['exposition']['EquipItemID']),
    EquipItemInSet: wrap('exposition.EquipItemInSet', globalThis['go']['go-pob']['exposition']['EquipItemInSet']),
    EquippedItems: wrap('exposition.EquippedItems', globalThis['go']['go-pob']['exposition']['EquippedItems']),
    ExportTreeLink: wrap('exposition.ExportTreeLink', globalThis['go']['go-pob']['exposition']['ExportTreeLink']),
    GemDefaultLevel: wrap('exposition.GemDefaultLevel', globalThis['go']['go-pob']['exposition']['GemDefaultLevel']),
    GemDefaultLevelMode: wrap('exposition.GemDefaultLevelMode', globalThis['go']['go-pob']['exposition']['GemDefaultLevelMode']),
    GemError: wrap('exposition.GemError', globalThis['go']['go-pob']['exposition']['GemError']),
    GemQualityTooltip: wrap('exposition.GemQualityTooltip', globalThis['go']['go-pob']['exposition']['GemQualityTooltip']),
    GemTooltip: wrap('exposition.GemTooltip', globalThis['go']['go-pob']['exposition']['GemTooltip']),
    GemType: globalThis['go']['go-pob']['exposition']['GemType'],
    GetLoadouts: wrap('exposition.GetLoadouts', globalThis['go']['go-pob']['exposition']['GetLoadouts']),
    GetMasteryOptions: wrap('exposition.GetMasteryOptions', globalThis['go']['go-pob']['exposition']['GetMasteryOptions']),
    GetMasterySelections: wrap('exposition.GetMasterySelections', globalThis['go']['go-pob']['exposition']['GetMasterySelections']),
    GetMiscCalculator: wrap('exposition.GetMiscCalculator', globalThis['go']['go-pob']['exposition']['GetMiscCalculator']),
    GetPartySummaries: wrap('exposition.GetPartySummaries', globalThis['go']['go-pob']['exposition']['GetPartySummaries']),
    GetRawTree: wrap('exposition.GetRawTree', globalThis['go']['go-pob']['exposition']['GetRawTree']),
    GetSkillGems: wrap('exposition.GetSkillGems', globalThis['go']['go-pob']['exposition']['GetSkillGems']),
    GetStatByIndex: wrap('exposition.GetStatByIndex', globalThis['go']['go-pob']['exposition']['GetStatByIndex']),
    GetTattooChoices: wrap('exposition.GetTattooChoices', globalThis['go']['go-pob']['exposition']['GetTattooChoices']),
    GetTattooCountLines: wrap('exposition.GetTattooCountLines', globalThis['go']['go-pob']['exposition']['GetTattooCountLines']),
    GetTimelessData: wrap('exposition.GetTimelessData', globalThis['go']['go-pob']['exposition']['GetTimelessData']),
    GetTreeClusterNodes: wrap('exposition.GetTreeClusterNodes', globalThis['go']['go-pob']['exposition']['GetTreeClusterNodes']),
    GetTreeJewelOverlays: wrap('exposition.GetTreeJewelOverlays', globalThis['go']['go-pob']['exposition']['GetTreeJewelOverlays']),
    GetTreeJewelRings: wrap('exposition.GetTreeJewelRings', globalThis['go']['go-pob']['exposition']['GetTreeJewelRings']),
    GetTreeSocketRadii: wrap('exposition.GetTreeSocketRadii', globalThis['go']['go-pob']['exposition']['GetTreeSocketRadii']),
    GetTreeSpecClusterNodes: wrap('exposition.GetTreeSpecClusterNodes', globalThis['go']['go-pob']['exposition']['GetTreeSpecClusterNodes']),
    GetTreeSpecs: wrap('exposition.GetTreeSpecs', globalThis['go']['go-pob']['exposition']['GetTreeSpecs']),
    GetTreeSpecsPoints: wrap('exposition.GetTreeSpecsPoints', globalThis['go']['go-pob']['exposition']['GetTreeSpecsPoints']),
    GetTreeVersionOptions: wrap('exposition.GetTreeVersionOptions', globalThis['go']['go-pob']['exposition']['GetTreeVersionOptions']),
    GoldRespecPrice: wrap('exposition.GoldRespecPrice', globalThis['go']['go-pob']['exposition']['GoldRespecPrice']),
    ImplicitText: wrap('exposition.ImplicitText', globalThis['go']['go-pob']['exposition']['ImplicitText']),
    ImportItemsAndSkills: wrap('exposition.ImportItemsAndSkills', globalThis['go']['go-pob']['exposition']['ImportItemsAndSkills']),
    ImportSharedItemSet: wrap('exposition.ImportSharedItemSet', globalThis['go']['go-pob']['exposition']['ImportSharedItemSet']),
    ImportTreeLink: wrap('exposition.ImportTreeLink', globalThis['go']['go-pob']['exposition']['ImportTreeLink']),
    ItemAffixControls: wrap('exposition.ItemAffixControls', globalThis['go']['go-pob']['exposition']['ItemAffixControls']),
    ItemCorruptChoices: wrap('exposition.ItemCorruptChoices', globalThis['go']['go-pob']['exposition']['ItemCorruptChoices']),
    ItemCrucibleChoices: wrap('exposition.ItemCrucibleChoices', globalThis['go']['go-pob']['exposition']['ItemCrucibleChoices']),
    ItemCustomModChoices: wrap('exposition.ItemCustomModChoices', globalThis['go']['go-pob']['exposition']['ItemCustomModChoices']),
    ItemDBLists: wrap('exposition.ItemDBLists', globalThis['go']['go-pob']['exposition']['ItemDBLists']),
    ItemDBRaw: wrap('exposition.ItemDBRaw', globalThis['go']['go-pob']['exposition']['ItemDBRaw']),
    ItemDBText: wrap('exposition.ItemDBText', globalThis['go']['go-pob']['exposition']['ItemDBText']),
    ItemDBTooltip: wrap('exposition.ItemDBTooltip', globalThis['go']['go-pob']['exposition']['ItemDBTooltip']),
    ItemEnchantChoices: wrap('exposition.ItemEnchantChoices', globalThis['go']['go-pob']['exposition']['ItemEnchantChoices']),
    ItemImplicitChoices: wrap('exposition.ItemImplicitChoices', globalThis['go']['go-pob']['exposition']['ItemImplicitChoices']),
    ItemListFiltered: wrap('exposition.ItemListFiltered', globalThis['go']['go-pob']['exposition']['ItemListFiltered']),
    ItemListLoadoutFilters: wrap('exposition.ItemListLoadoutFilters', globalThis['go']['go-pob']['exposition']['ItemListLoadoutFilters']),
    ItemPrimarySlot: wrap('exposition.ItemPrimarySlot', globalThis['go']['go-pob']['exposition']['ItemPrimarySlot']),
    ItemSetTooltips: wrap('exposition.ItemSetTooltips', globalThis['go']['go-pob']['exposition']['ItemSetTooltips']),
    ItemSourceTooltip: wrap('exposition.ItemSourceTooltip', globalThis['go']['go-pob']['exposition']['ItemSourceTooltip']),
    ItemsTabSlots: wrap('exposition.ItemsTabSlots', globalThis['go']['go-pob']['exposition']['ItemsTabSlots']),
    LeapHeldNodes: wrap('exposition.LeapHeldNodes', globalThis['go']['go-pob']['exposition']['LeapHeldNodes']),
    LiveHeapMB: wrap('exposition.LiveHeapMB', globalThis['go']['go-pob']['exposition']['LiveHeapMB']),
    MiscCalculatorFor: wrap('exposition.MiscCalculatorFor', globalThis['go']['go-pob']['exposition']['MiscCalculatorFor']),
    MiscGemDisplays: wrap('exposition.MiscGemDisplays', globalThis['go']['go-pob']['exposition']['MiscGemDisplays']),
    MiscGemSortCache: wrap('exposition.MiscGemSortCache', globalThis['go']['go-pob']['exposition']['MiscGemSortCache']),
    MiscGemSupportPairs: wrap('exposition.MiscGemSupportPairs', globalThis['go']['go-pob']['exposition']['MiscGemSupportPairs']),
    MiscSocketGroupTooltip: wrap('exposition.MiscSocketGroupTooltip', globalThis['go']['go-pob']['exposition']['MiscSocketGroupTooltip']),
    ModBrowserList: wrap('exposition.ModBrowserList', globalThis['go']['go-pob']['exposition']['ModBrowserList']),
    ModBrowserSupported: wrap('exposition.ModBrowserSupported', globalThis['go']['go-pob']['exposition']['ModBrowserSupported']),
    ModDebugDump: wrap('exposition.ModDebugDump', globalThis['go']['go-pob']['exposition']['ModDebugDump']),
    ModLineStatus: globalThis['go']['go-pob']['exposition']['ModLineStatus'],
    MoveItemSet: wrap('exposition.MoveItemSet', globalThis['go']['go-pob']['exposition']['MoveItemSet']),
    MoveTreeSpec: wrap('exposition.MoveTreeSpec', globalThis['go']['go-pob']['exposition']['MoveTreeSpec']),
    NewItemSet: wrap('exposition.NewItemSet', globalThis['go']['go-pob']['exposition']['NewItemSet']),
    NewLoadout: wrap('exposition.NewLoadout', globalThis['go']['go-pob']['exposition']['NewLoadout']),
    NewPowerBuilder: wrap('exposition.NewPowerBuilder', globalThis['go']['go-pob']['exposition']['NewPowerBuilder']),
    NewTimelessSearch: wrap('exposition.NewTimelessSearch', globalThis['go']['go-pob']['exposition']['NewTimelessSearch']),
    NewTreeSpec: wrap('exposition.NewTreeSpec', globalThis['go']['go-pob']['exposition']['NewTreeSpec']),
    NodePowerStats: wrap('exposition.NodePowerStats', globalThis['go']['go-pob']['exposition']['NodePowerStats']),
    NodeSkillTooltips: wrap('exposition.NodeSkillTooltips', globalThis['go']['go-pob']['exposition']['NodeSkillTooltips']),
    NormaliseDisplayItem: wrap('exposition.NormaliseDisplayItem', globalThis['go']['go-pob']['exposition']['NormaliseDisplayItem']),
    OptimiseSockets: wrap('exposition.OptimiseSockets', globalThis['go']['go-pob']['exposition']['OptimiseSockets']),
    ParseItem: wrap('exposition.ParseItem', globalThis['go']['go-pob']['exposition']['ParseItem']),
    ParseModLines: wrap('exposition.ParseModLines', globalThis['go']['go-pob']['exposition']['ParseModLines']),
    PowerStats: wrap('exposition.PowerStats', globalThis['go']['go-pob']['exposition']['PowerStats']),
    RemoveAllTreeTattoos: wrap('exposition.RemoveAllTreeTattoos', globalThis['go']['go-pob']['exposition']['RemoveAllTreeTattoos']),
    RenameItemSet: wrap('exposition.RenameItemSet', globalThis['go']['go-pob']['exposition']['RenameItemSet']),
    RenameSharedItemSet: wrap('exposition.RenameSharedItemSet', globalThis['go']['go-pob']['exposition']['RenameSharedItemSet']),
    ResetTreeNodes: wrap('exposition.ResetTreeNodes', globalThis['go']['go-pob']['exposition']['ResetTreeNodes']),
    SecondaryAscendancies: wrap('exposition.SecondaryAscendancies', globalThis['go']['go-pob']['exposition']['SecondaryAscendancies']),
    SelectLoadout: wrap('exposition.SelectLoadout', globalThis['go']['go-pob']['exposition']['SelectLoadout']),
    SelectMasteryEffect: wrap('exposition.SelectMasteryEffect', globalThis['go']['go-pob']['exposition']['SelectMasteryEffect']),
    SetActiveItemSet: wrap('exposition.SetActiveItemSet', globalThis['go']['go-pob']['exposition']['SetActiveItemSet']),
    SetActiveTreeSpec: wrap('exposition.SetActiveTreeSpec', globalThis['go']['go-pob']['exposition']['SetActiveTreeSpec']),
    SetCalcTabElements: wrap('exposition.SetCalcTabElements', globalThis['go']['go-pob']['exposition']['SetCalcTabElements']),
    SetItemModDebug: wrap('exposition.SetItemModDebug', globalThis['go']['go-pob']['exposition']['SetItemModDebug']),
    SetShowFlavourText: wrap('exposition.SetShowFlavourText', globalThis['go']['go-pob']['exposition']['SetShowFlavourText']),
    SetTimelessData: wrap('exposition.SetTimelessData', globalThis['go']['go-pob']['exposition']['SetTimelessData']),
    SetWeaponSet: wrap('exposition.SetWeaponSet', globalThis['go']['go-pob']['exposition']['SetWeaponSet']),
    ShareItem: wrap('exposition.ShareItem', globalThis['go']['go-pob']['exposition']['ShareItem']),
    ShareItemSet: wrap('exposition.ShareItemSet', globalThis['go']['go-pob']['exposition']['ShareItemSet']),
    SharedItemSets: wrap('exposition.SharedItemSets', globalThis['go']['go-pob']['exposition']['SharedItemSets']),
    SharedItems: wrap('exposition.SharedItems', globalThis['go']['go-pob']['exposition']['SharedItems']),
    SlotItemLists: wrap('exposition.SlotItemLists', globalThis['go']['go-pob']['exposition']['SlotItemLists']),
    SocketGroupSourceNames: wrap('exposition.SocketGroupSourceNames', globalThis['go']['go-pob']['exposition']['SocketGroupSourceNames']),
    SocketTreeJewel: wrap('exposition.SocketTreeJewel', globalThis['go']['go-pob']['exposition']['SocketTreeJewel']),
    SocketedInItemNames: wrap('exposition.SocketedInItemNames', globalThis['go']['go-pob']['exposition']['SocketedInItemNames']),
    SortItems: wrap('exposition.SortItems', globalThis['go']['go-pob']['exposition']['SortItems']),
    SpectreLibrary: wrap('exposition.SpectreLibrary', globalThis['go']['go-pob']['exposition']['SpectreLibrary']),
    SpectreRows: wrap('exposition.SpectreRows', globalThis['go']['go-pob']['exposition']['SpectreRows']),
    TattooEditKind: wrap('exposition.TattooEditKind', globalThis['go']['go-pob']['exposition']['TattooEditKind']),
    TimelessConquerors: wrap('exposition.TimelessConquerors', globalThis['go']['go-pob']['exposition']['TimelessConquerors']),
    TimelessDevotionVariants: wrap('exposition.TimelessDevotionVariants', globalThis['go']['go-pob']['exposition']['TimelessDevotionVariants']),
    TimelessFallbackModes: wrap('exposition.TimelessFallbackModes', globalThis['go']['go-pob']['exposition']['TimelessFallbackModes']),
    TimelessJewelTypes: wrap('exposition.TimelessJewelTypes', globalThis['go']['go-pob']['exposition']['TimelessJewelTypes']),
    TimelessMods: wrap('exposition.TimelessMods', globalThis['go']['go-pob']['exposition']['TimelessMods']),
    TimelessProtectableNodes: wrap('exposition.TimelessProtectableNodes', globalThis['go']['go-pob']['exposition']['TimelessProtectableNodes']),
    TimelessRealms: wrap('exposition.TimelessRealms', globalThis['go']['go-pob']['exposition']['TimelessRealms']),
    TimelessSockets: wrap('exposition.TimelessSockets', globalThis['go']['go-pob']['exposition']['TimelessSockets']),
    TimelessTradeTypes: wrap('exposition.TimelessTradeTypes', globalThis['go']['go-pob']['exposition']['TimelessTradeTypes']),
    ToggleEquipItem: wrap('exposition.ToggleEquipItem', globalThis['go']['go-pob']['exposition']['ToggleEquipItem']),
    TreeSocketChoices: wrap('exposition.TreeSocketChoices', globalThis['go']['go-pob']['exposition']['TreeSocketChoices']),
    TreeSocketedJewels: wrap('exposition.TreeSocketedJewels', globalThis['go']['go-pob']['exposition']['TreeSocketedJewels']),
    TreeSpecCompare: wrap('exposition.TreeSpecCompare', globalThis['go']['go-pob']['exposition']['TreeSpecCompare']),
    TreeSpecRespecCost: wrap('exposition.TreeSpecRespecCost', globalThis['go']['go-pob']['exposition']['TreeSpecRespecCost']),
    TreeTattooArt: wrap('exposition.TreeTattooArt', globalThis['go']['go-pob']['exposition']['TreeTattooArt'])
  };
  mod = {
    KeywordFlag: globalThis['go']['go-pob']['mod']['KeywordFlag'],
    MFlag: globalThis['go']['go-pob']['mod']['MFlag'],
    ModValueMultiType: globalThis['go']['go-pob']['mod']['ModValueMultiType'],
    Source: globalThis['go']['go-pob']['mod']['Source'],
    Type: globalThis['go']['go-pob']['mod']['Type']
  };
  msgp = {
    Type: globalThis['go']['go-pob']['msgp']['Type']
  };
  pob = {
    AscendancyID: globalThis['go']['go-pob']['pob']['AscendancyID'],
    BuildInfo: globalThis['go']['go-pob']['pob']['BuildInfo'],
    BuildViewMode: globalThis['go']['go-pob']['pob']['BuildViewMode'],
    ClassID: globalThis['go']['go-pob']['pob']['ClassID'],
    CompressEncode: wrap('pob.CompressEncode', globalThis['go']['go-pob']['pob']['CompressEncode']),
    DecodeDecompress: wrap('pob.DecodeDecompress', globalThis['go']['go-pob']['pob']['DecodeDecompress']),
    GameVersion: globalThis['go']['go-pob']['pob']['GameVersion']
  };
  poe = {
    SkillFlag: globalThis['go']['go-pob']['poe']['SkillFlag'],
    SkillType: globalThis['go']['go-pob']['poe']['SkillType']
  };
  raw = {
    GameDataVersion: wrap('raw.GameDataVersion', globalThis['go']['go-pob']['raw']['GameDataVersion']),
    InitializeAll: wrap('raw.InitializeAll', globalThis['go']['go-pob']['raw']['InitializeAll']),
    LoadDeferred: wrap('raw.LoadDeferred', globalThis['go']['go-pob']['raw']['LoadDeferred']),
    TagName: globalThis['go']['go-pob']['raw']['TagName']
  };
  storage = {
    DeleteBuild: wrap('storage.DeleteBuild', globalThis['go']['go-pob']['storage']['DeleteBuild']),
    DirEntryType: globalThis['go']['go-pob']['storage']['DirEntryType'],
    GetBuild: wrap('storage.GetBuild', globalThis['go']['go-pob']['storage']['GetBuild']),
    InitializeStorage: wrap('storage.InitializeStorage', globalThis['go']['go-pob']['storage']['InitializeStorage']),
    ListBuilds: wrap('storage.ListBuilds', globalThis['go']['go-pob']['storage']['ListBuilds']),
    NewFolder: wrap('storage.NewFolder', globalThis['go']['go-pob']['storage']['NewFolder']),
    SetBuild: wrap('storage.SetBuild', globalThis['go']['go-pob']['storage']['SetBuild'])
  };

  initialized = true;
};

export const boot = async (wasm) => {
  if (globalThis['Go'] === undefined) {
    throw new Error('crystalline: the Go runtime shim is missing. Load wasm_exec.js from your Go toolchain before calling boot().');
  }

  const runtime = new globalThis['Go']();

  let source = wasm;

  if (!(wasm instanceof ArrayBuffer) && !ArrayBuffer.isView(wasm)) {
    const response = await fetch(wasm);

    if (!response.ok) {
      throw new Error('crystalline: could not fetch ' + wasm + ': ' + response.status);
    }

    source = await response.arrayBuffer();
  }

  const { instance } = await WebAssembly.instantiate(source, runtime.importObject);

  // Not awaited: the Go program parks, so this never settles.
  runtime.run(instance);

  initializeCrystalline();

  return { builds, calculator, config, data, exposition, mod, msgp, pob, poe, raw, storage };
};