/* eslint-disable */
// @ts-nocheck
export let builds;
export let calculator;
export let config;
export let exposition;
export let pob;
export let raw;
export let storage;

export const initializeCrystalline = () => {
  builds = {
    EmptyBuild: globalThis['go']['go-pob']['builds']['EmptyBuild'],
    ParseBuild: globalThis['go']['go-pob']['builds']['ParseBuild'],
    ParseBuildStr: globalThis['go']['go-pob']['builds']['ParseBuildStr'],
    SerializeBuild: globalThis['go']['go-pob']['builds']['SerializeBuild']
  };
  calculator = {
    NewCalculator: globalThis['go']['go-pob']['calculator']['NewCalculator']
  };
  config = {
    InitLogging: globalThis['go']['go-pob']['config']['InitLogging']
  };
  exposition = {
    CalculateAllocationPaths: globalThis['go']['go-pob']['exposition']['CalculateAllocationPaths'],
    CalculatePrunableNodes: globalThis['go']['go-pob']['exposition']['CalculatePrunableNodes'],
    GetRawTree: globalThis['go']['go-pob']['exposition']['GetRawTree'],
    GetSkillGems: globalThis['go']['go-pob']['exposition']['GetSkillGems'],
    GetStatByIndex: globalThis['go']['go-pob']['exposition']['GetStatByIndex'],
    SetCalcTabElements: globalThis['go']['go-pob']['exposition']['SetCalcTabElements']
  };
  pob = {
    BuildInfo: globalThis['go']['go-pob']['pob']['BuildInfo'],
    CompressEncode: globalThis['go']['go-pob']['pob']['CompressEncode'],
    DecodeDecompress: globalThis['go']['go-pob']['pob']['DecodeDecompress']
  };
  raw = {
    InitializeAll: globalThis['go']['go-pob']['raw']['InitializeAll']
  };
  storage = {
    DeleteBuild: globalThis['go']['go-pob']['storage']['DeleteBuild'],
    GetBuild: globalThis['go']['go-pob']['storage']['GetBuild'],
    InitializeStorage: globalThis['go']['go-pob']['storage']['InitializeStorage'],
    ListBuilds: globalThis['go']['go-pob']['storage']['ListBuilds'],
    NewFolder: globalThis['go']['go-pob']['storage']['NewFolder'],
    SetBuild: globalThis['go']['go-pob']['storage']['SetBuild']
  };
};